package service

import (
	"path/filepath"
	"strings"
	"testing"

	"krillin-ai/internal/appdirs"
)

func TestResolveTaskDirUsesOutputDir(t *testing.T) {
	tempDir := t.TempDir()
	originalResolver := appDirsResolver
	t.Cleanup(func() {
		appDirsResolver = originalResolver
	})

	outputDir := filepath.Join(tempDir, "output-root")
	appDirsResolver = func() (appdirs.Paths, error) {
		return appdirs.Paths{
			OutputDir: outputDir,
			CacheDir:  filepath.Join(tempDir, "cache-root"),
		}, nil
	}

	got, err := resolveTaskDir("task-001")
	if err != nil {
		t.Fatalf("resolveTaskDir() returned error: %v", err)
	}

	want := filepath.Join(outputDir, "tasks", "task-001")
	if got != want {
		t.Fatalf("resolveTaskDir() = %q, want %q", got, want)
	}
}

func TestResolveTaskDownloadPath(t *testing.T) {
	tempDir := t.TempDir()
	originalResolver := appDirsResolver
	t.Cleanup(func() {
		appDirsResolver = originalResolver
	})

	outputDir := filepath.Join(tempDir, "output-root")
	appDirsResolver = func() (appdirs.Paths, error) {
		return appdirs.Paths{OutputDir: outputDir, CacheDir: filepath.Join(tempDir, "cache-root")}, nil
	}

	localArtifact := filepath.Join(outputDir, "tasks", "task-001", "output", "subtitle.srt")
	got, err := resolveTaskDownloadPath(localArtifact)
	if err != nil {
		t.Fatalf("resolveTaskDownloadPath() returned error: %v", err)
	}

	want := "tasks/task-001/output/subtitle.srt"
	if got != want {
		t.Fatalf("resolveTaskDownloadPath() = %q, want %q", got, want)
	}
}

func TestResolveTaskDownloadPathRejectsOutsideTaskRoot(t *testing.T) {
	tempDir := t.TempDir()
	originalResolver := appDirsResolver
	t.Cleanup(func() {
		appDirsResolver = originalResolver
	})

	outputDir := filepath.Join(tempDir, "output-root")
	appDirsResolver = func() (appdirs.Paths, error) {
		return appdirs.Paths{OutputDir: outputDir, CacheDir: filepath.Join(tempDir, "cache-root")}, nil
	}

	_, err := resolveTaskDownloadPath(filepath.Join(tempDir, "not-task-root", "subtitle.srt"))
	if err == nil {
		t.Fatal("resolveTaskDownloadPath() returned nil error for path outside task root")
	}
	if !strings.Contains(err.Error(), "outside task root") {
		t.Fatalf("resolveTaskDownloadPath() error = %q, want containing %q", err.Error(), "outside task root")
	}
}
