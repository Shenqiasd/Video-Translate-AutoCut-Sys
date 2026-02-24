package storage

import (
	"path/filepath"
	"testing"

	"krillin-ai/internal/appdirs"
)

func TestResolveDBPathUsesCacheDir(t *testing.T) {
	originalResolver := appDirsResolver
	t.Cleanup(func() {
		appDirsResolver = originalResolver
	})

	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache-root")
	appDirsResolver = func() (appdirs.Paths, error) {
		return appdirs.Paths{
			OutputDir: filepath.Join(tempDir, "output-root"),
			CacheDir:  cacheDir,
		}, nil
	}

	got, err := resolveDBPath()
	if err != nil {
		t.Fatalf("resolveDBPath() returned error: %v", err)
	}

	want := filepath.Join(cacheDir, "krillin.db")
	if got != want {
		t.Fatalf("resolveDBPath() = %q, want %q", got, want)
	}
}
