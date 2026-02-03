package service

import (
	"context"
	"fmt"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
)

// SeparationResult holds the paths to separated audio files
type SeparationResult struct {
	VocalsPath       string // Path to the extracted vocals (original speech)
	InstrumentalPath string // Path to the background music/instrumental
}

// SeparateAudioSources uses audio-separator to split audio into vocals and instrumental
// This enables us to retain background music while replacing spoken dialogue
func (s Service) SeparateAudioSources(ctx context.Context, stepParam *types.SubtitleTaskStepParam) (*SeparationResult, error) {
	audioPath := stepParam.AudioFilePath
	outputDir := stepParam.TaskBasePath

	// Check if audio file exists
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("audio file not found: %s", audioPath)
	}

	log.GetLogger().Info("Starting audio separation",
		zap.String("taskId", stepParam.TaskId),
		zap.String("audioPath", audioPath),
		zap.String("outputDir", outputDir))

	// Use audio-separator CLI
	// Model: UVR-MDX-NET-Inst_HQ_3 is known for high-quality instrumental extraction
	// --single_stem=instrumental will output only the instrumental track
	// We need both, so we run without --single_stem to get both outputs
	cmd := exec.CommandContext(ctx, "audio-separator",
		"--model_filename", "UVR-MDX-NET-Inst_HQ_3.onnx",
		"--output_dir", outputDir,
		"--output_format", "wav",
		audioPath,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.GetLogger().Error("audio-separator failed",
			zap.String("taskId", stepParam.TaskId),
			zap.Error(err))
		return nil, fmt.Errorf("audio separation failed: %w", err)
	}

	// audio-separator outputs files with naming convention:
	// <original_filename>_(Vocals)_<model>.wav
	// <original_filename>_(Instrumental)_<model>.wav
	baseName := filepath.Base(audioPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := baseName[:len(baseName)-len(ext)]

	// Find the output files (naming may vary slightly based on model)
	vocalsPath := filepath.Join(outputDir, fmt.Sprintf("%s_(Vocals)_UVR-MDX-NET-Inst_HQ_3.wav", nameWithoutExt))
	instrumentalPath := filepath.Join(outputDir, fmt.Sprintf("%s_(Instrumental)_UVR-MDX-NET-Inst_HQ_3.wav", nameWithoutExt))

	// Check if outputs exist
	if _, err := os.Stat(instrumentalPath); os.IsNotExist(err) {
		// Try alternative naming patterns
		matches, _ := filepath.Glob(filepath.Join(outputDir, "*Instrumental*.wav"))
		if len(matches) > 0 {
			instrumentalPath = matches[0]
		} else {
			return nil, fmt.Errorf("instrumental output not found after separation")
		}
	}

	if _, err := os.Stat(vocalsPath); os.IsNotExist(err) {
		// Vocals file is optional (we may not need it), but log a warning
		log.GetLogger().Warn("Vocals output not found (may be expected for some models)",
			zap.String("expectedPath", vocalsPath))
		vocalsPath = "" // Set to empty if not found
	}

	log.GetLogger().Info("Audio separation completed",
		zap.String("taskId", stepParam.TaskId),
		zap.String("instrumentalPath", instrumentalPath),
		zap.String("vocalsPath", vocalsPath))

	return &SeparationResult{
		VocalsPath:       vocalsPath,
		InstrumentalPath: instrumentalPath,
	}, nil
}
