package deps

import (
	"errors"
	"fmt"
	"krillin-ai/internal/storage"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type DependencyTier string

const (
	DependencyTierMust     DependencyTier = "must"
	DependencyTierShould   DependencyTier = "should"
	DependencyTierOptional DependencyTier = "optional"
)

type DependencyStatus string

const (
	DependencyStatusOK      DependencyStatus = "ok"
	DependencyStatusMissing DependencyStatus = "missing"
	DependencyStatusError   DependencyStatus = "error"
)

type DependencySource string

const (
	DependencySourceStorage  DependencySource = "storage"
	DependencySourceLookPath DependencySource = "lookpath"
)

type DependencySpec struct {
	ID          string
	Name        string
	Command     string
	Tier        DependencyTier
	StoragePath string
	Hint        string
}

type DependencyState struct {
	DependencySpec
	ResolvedPath string
	Status       DependencyStatus
	Source       DependencySource
	Error        string
}

type PathResolver struct {
	LookPath func(file string) (string, error)
	AbsPath  func(path string) (string, error)
	Stat     func(name string) (os.FileInfo, error)
}

func NewPathResolver() PathResolver {
	return PathResolver{
		LookPath: exec.LookPath,
		AbsPath:  filepath.Abs,
		Stat:     os.Stat,
	}
}

func (r PathResolver) Resolve(spec DependencySpec) DependencyState {
	state := DependencyState{DependencySpec: spec}
	configured := strings.TrimSpace(spec.StoragePath)

	if configured != "" {
		state.Source = DependencySourceStorage
		resolvedPath, err := r.resolveConfiguredPath(configured)
		if err == nil {
			state.Status = DependencyStatusOK
			state.ResolvedPath = resolvedPath
			return state
		}

		if absPath, absErr := r.AbsPath(configured); absErr == nil {
			state.ResolvedPath = absPath
		} else {
			state.ResolvedPath = configured
		}
		state.Error = err.Error()
		if isMissingPathError(err) {
			state.Status = DependencyStatusMissing
		} else {
			state.Status = DependencyStatusError
		}
		return state
	}

	state.Source = DependencySourceLookPath
	resolvedPath, err := r.LookPath(spec.Command)
	if err == nil {
		state.Status = DependencyStatusOK
		state.ResolvedPath = resolvedPath
		return state
	}

	state.Error = err.Error()
	if isMissingPathError(err) {
		state.Status = DependencyStatusMissing
		return state
	}
	state.Status = DependencyStatusError
	return state
}

func (r PathResolver) resolveConfiguredPath(configuredPath string) (string, error) {
	if resolvedPath, err := r.LookPath(configuredPath); err == nil {
		return resolvedPath, nil
	}

	absPath, err := r.AbsPath(configuredPath)
	if err != nil {
		return "", err
	}
	if _, err = r.Stat(absPath); err != nil {
		return "", err
	}
	return absPath, nil
}

func ResolveDependencyStates(specs []DependencySpec, resolver PathResolver) []DependencyState {
	resolved := make([]DependencyState, 0, len(specs))
	for _, spec := range specs {
		resolved = append(resolved, resolver.Resolve(spec))
	}
	return resolved
}

func ResolveDependencyInventory(transcribeProvider, ttsProvider string) []DependencyState {
	specs := BuildDependencyInventory(transcribeProvider, ttsProvider)
	return ResolveDependencyStates(specs, NewPathResolver())
}

func BuildDependencyInventory(transcribeProvider, ttsProvider string) []DependencySpec {
	normalizedTranscribeProvider := strings.ToLower(strings.TrimSpace(transcribeProvider))
	normalizedTtsProvider := strings.ToLower(strings.TrimSpace(ttsProvider))

	edgeTier := DependencyTierOptional
	edgeHint := "Only needed when TTS provider is edge-tts."
	if normalizedTtsProvider == "edge-tts" {
		edgeTier = DependencyTierShould
		edgeHint = "Current TTS provider is edge-tts; install this binary for local speech synthesis."
	}

	return []DependencySpec{
		{
			ID:          "ffmpeg",
			Name:        "ffmpeg",
			Command:     "ffmpeg",
			Tier:        DependencyTierMust,
			StoragePath: storage.FfmpegPath,
			Hint:        "Required for audio extraction and video composition.",
		},
		{
			ID:          "ffprobe",
			Name:        "ffprobe",
			Command:     "ffprobe",
			Tier:        DependencyTierMust,
			StoragePath: storage.FfprobePath,
			Hint:        "Required for media metadata detection.",
		},
		{
			ID:          "yt-dlp",
			Name:        "yt-dlp",
			Command:     "yt-dlp",
			Tier:        DependencyTierMust,
			StoragePath: storage.YtdlpPath,
			Hint:        "Required for URL downloads (Paste a link mode).",
		},
		{
			ID:          "edge-tts",
			Name:        "edge-tts",
			Command:     "edge-tts",
			Tier:        edgeTier,
			StoragePath: storage.EdgeTtsPath,
			Hint:        edgeHint,
		},
		{
			ID:          "fasterwhisper",
			Name:        "fasterwhisper",
			Command:     "faster-whisper-xxl",
			Tier:        DependencyTierOptional,
			StoragePath: storage.FasterwhisperPath,
			Hint: providerHint(
				normalizedTranscribeProvider,
				"fasterwhisper",
				"Current transcribe provider is fasterwhisper; this binary is required.",
				"Needed only if you switch Transcribe provider to fasterwhisper.",
			),
		},
		{
			ID:          "whispercpp",
			Name:        "whispercpp",
			Command:     "whisper-cli",
			Tier:        DependencyTierOptional,
			StoragePath: storage.WhispercppPath,
			Hint: providerHint(
				normalizedTranscribeProvider,
				"whispercpp",
				"Current transcribe provider is whispercpp; this binary is required.",
				"Needed only if you switch Transcribe provider to whispercpp.",
			),
		},
		{
			ID:          "whisperx",
			Name:        "whisperx",
			Command:     "whisperx",
			Tier:        DependencyTierOptional,
			StoragePath: storage.WhisperXPath,
			Hint: providerHint(
				normalizedTranscribeProvider,
				"whisperx",
				"Current transcribe provider is whisperx; this binary is required.",
				"Needed only if you switch Transcribe provider to whisperx.",
			),
		},
		{
			ID:          "whisperkit",
			Name:        "whisperkit",
			Command:     "whisperkit-cli",
			Tier:        DependencyTierOptional,
			StoragePath: storage.WhisperKitPath,
			Hint: providerHint(
				normalizedTranscribeProvider,
				"whisperkit",
				"Current transcribe provider is whisperkit; this binary is required.",
				"Needed only if you switch Transcribe provider to whisperkit.",
			),
		},
	}
}

func FormatDependencyReport(states []DependencyState) string {
	if len(states) == 0 {
		return "No dependencies to diagnose."
	}

	var builder strings.Builder
	builder.WriteString("Dependency status")

	for _, state := range states {
		resolvedPath := strings.TrimSpace(state.ResolvedPath)
		if resolvedPath == "" {
			resolvedPath = "unknown"
		}

		source := strings.TrimSpace(string(state.Source))
		if source == "" {
			source = "n/a"
		}

		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("- %s [%s]: %s | path=%s | source=%s", state.Name, strings.ToUpper(string(state.Tier)), state.Status, resolvedPath, source))
		if state.Error != "" {
			builder.WriteString("\n")
			builder.WriteString("  error: ")
			builder.WriteString(state.Error)
		}
		if state.Hint != "" {
			builder.WriteString("\n")
			builder.WriteString("  hint: ")
			builder.WriteString(state.Hint)
		}
	}

	return builder.String()
}

func providerHint(provider, target, activeHint, inactiveHint string) string {
	if provider == target {
		return activeHint
	}
	return inactiveHint
}

func isMissingPathError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, exec.ErrNotFound) {
		return true
	}

	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		if errors.Is(pathErr.Err, os.ErrNotExist) {
			return true
		}
	}

	var execErr *exec.Error
	if errors.As(err, &execErr) {
		if errors.Is(execErr.Err, exec.ErrNotFound) {
			return true
		}
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not found") || strings.Contains(message, "cannot find")
}
