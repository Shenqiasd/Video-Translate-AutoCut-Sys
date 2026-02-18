package main

import (
	"bytes"
	"errors"
	"io"
	"krillin-ai/internal/appdirs"
	"krillin-ai/internal/storage"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() failed: %v", err)
	}

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, reader); err != nil {
		t.Fatalf("io.Copy() failed: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close() failed: %v", err)
	}

	return buffer.String()
}

func restoreDiagnoseGlobals(t *testing.T) {
	t.Helper()

	originalAppDirsResolver := appDirsResolver
	originalLogFileResolver := logFileResolver
	originalLookPathResolver := lookPathResolver
	originalMkdirAllResolver := mkdirAllResolver
	originalCreateTempResolver := createTempResolver
	originalRemoveResolver := removeResolver

	originalFfmpegPath := storage.FfmpegPath
	originalFfprobePath := storage.FfprobePath
	originalYtdlpPath := storage.YtdlpPath
	originalEdgeTtsPath := storage.EdgeTtsPath

	t.Cleanup(func() {
		appDirsResolver = originalAppDirsResolver
		logFileResolver = originalLogFileResolver
		lookPathResolver = originalLookPathResolver
		mkdirAllResolver = originalMkdirAllResolver
		createTempResolver = originalCreateTempResolver
		removeResolver = originalRemoveResolver

		storage.FfmpegPath = originalFfmpegPath
		storage.FfprobePath = originalFfprobePath
		storage.YtdlpPath = originalYtdlpPath
		storage.EdgeTtsPath = originalEdgeTtsPath
	})
}

func TestPrintDiagnoseShowsEffectivePathsWritabilityAndDependencies(t *testing.T) {
	restoreDiagnoseGlobals(t)

	tempRoot := t.TempDir()
	paths := appdirs.Paths{
		Portable:   true,
		ConfigDir:  filepath.Join(tempRoot, "config"),
		ConfigFile: filepath.Join(tempRoot, "config", "config.toml"),
		LogDir:     filepath.Join(tempRoot, "logs"),
		OutputDir:  filepath.Join(tempRoot, "output"),
		CacheDir:   filepath.Join(tempRoot, "cache"),
	}

	appDirsResolver = func() (appdirs.Paths, error) {
		return paths, nil
	}
	logFileResolver = func() (string, error) {
		return filepath.Join(paths.LogDir, "app.log"), nil
	}
	lookPathResolver = func(command string) (string, error) {
		mocks := map[string]string{
			"ffmpeg":   "/mock/bin/ffmpeg",
			"ffprobe":  "/mock/bin/ffprobe",
			"yt-dlp":   "/mock/bin/yt-dlp",
			"edge-tts": "/mock/bin/edge-tts",
		}
		path, ok := mocks[command]
		if !ok {
			return "", errors.New("missing")
		}
		return path, nil
	}

	storage.FfmpegPath = ""
	storage.FfprobePath = ""
	storage.YtdlpPath = ""
	storage.EdgeTtsPath = ""

	output := captureStdout(t, printDiagnose)

	expected := []string{
		"portable_mode: true",
		"path.config_dir:",
		"path.config_file:",
		"path.log_dir:",
		"path.log_file:",
		"path.output_dir:",
		"path.task_root:",
		"path.cache_dir:",
		"path.db_file:",
		"writable.config_dir: ok",
		"writable.log_dir: ok",
		"writable.output_dir: ok",
		"writable.task_root: ok",
		"writable.cache_dir: ok",
		"dependency.ffmpeg: /mock/bin/ffmpeg (lookpath)",
		"dependency.ffprobe: /mock/bin/ffprobe (lookpath)",
		"dependency.yt-dlp: /mock/bin/yt-dlp (lookpath)",
		"dependency.edge-tts: /mock/bin/edge-tts (lookpath)",
	}

	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Fatalf("printDiagnose() output missing %q\nfull output:\n%s", want, output)
		}
	}
}

func TestPrintDiagnoseDependencySummaryStorageAndUnknown(t *testing.T) {
	restoreDiagnoseGlobals(t)

	tempRoot := t.TempDir()
	paths := appdirs.Paths{
		ConfigDir:  filepath.Join(tempRoot, "config"),
		ConfigFile: filepath.Join(tempRoot, "config", "config.toml"),
		LogDir:     filepath.Join(tempRoot, "logs"),
		OutputDir:  filepath.Join(tempRoot, "output"),
		CacheDir:   filepath.Join(tempRoot, "cache"),
	}

	configuredFFmpegPath := filepath.Join(tempRoot, "bin", "ffmpeg-custom")
	if err := os.MkdirAll(filepath.Dir(configuredFFmpegPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() failed: %v", err)
	}
	if err := os.WriteFile(configuredFFmpegPath, []byte("ffmpeg"), 0o755); err != nil {
		t.Fatalf("os.WriteFile() failed: %v", err)
	}

	appDirsResolver = func() (appdirs.Paths, error) {
		return paths, nil
	}
	logFileResolver = func() (string, error) {
		return filepath.Join(paths.LogDir, "app.log"), nil
	}
	lookPathResolver = func(command string) (string, error) {
		return "", errors.New("not found")
	}

	storage.FfmpegPath = configuredFFmpegPath
	storage.FfprobePath = ""
	storage.YtdlpPath = ""
	storage.EdgeTtsPath = ""

	output := captureStdout(t, printDiagnose)

	if !strings.Contains(output, "dependency.ffmpeg: "+configuredFFmpegPath+" (storage)") {
		t.Fatalf("printDiagnose() output missing storage ffmpeg path\nfull output:\n%s", output)
	}
	if !strings.Contains(output, "dependency.ffprobe: unknown (not found)") {
		t.Fatalf("printDiagnose() output missing ffprobe unknown status\nfull output:\n%s", output)
	}
	if !strings.Contains(output, "dependency.yt-dlp: unknown (not found)") {
		t.Fatalf("printDiagnose() output missing yt-dlp unknown status\nfull output:\n%s", output)
	}
	if !strings.Contains(output, "dependency.edge-tts: unknown (not found)") {
		t.Fatalf("printDiagnose() output missing edge-tts unknown status\nfull output:\n%s", output)
	}
}

func TestPrintWritabilityReportsError(t *testing.T) {
	restoreDiagnoseGlobals(t)

	mkdirAllResolver = func(path string, perm os.FileMode) error {
		return nil
	}
	createTempResolver = func(dir, pattern string) (*os.File, error) {
		return nil, errors.New("boom")
	}

	output := captureStdout(t, func() {
		printWritability("cache_dir", t.TempDir())
	})
	if !strings.Contains(output, "writable.cache_dir: error (create temp file: boom)") {
		t.Fatalf("printWritability() output missing create temp error\nfull output:\n%s", output)
	}
}
