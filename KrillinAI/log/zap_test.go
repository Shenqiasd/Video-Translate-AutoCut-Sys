package log

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"krillin-ai/internal/appdirs"
)

func setAppDirsResolverForTest(t *testing.T, resolver func() (appdirs.Paths, error)) {
	t.Helper()

	originalResolver := appDirsResolver
	appDirsResolver = resolver
	t.Cleanup(func() {
		appDirsResolver = originalResolver
	})
}

func TestResolveLogDir(t *testing.T) {
	t.Run("uses resolved log dir", func(t *testing.T) {
		expectedDir := filepath.Join("tmp", "logs")
		setAppDirsResolverForTest(t, func() (appdirs.Paths, error) {
			return appdirs.Paths{LogDir: expectedDir}, nil
		})

		logDir, err := ResolveLogDir()
		if err != nil {
			t.Fatalf("ResolveLogDir() returned unexpected error: %v", err)
		}
		if logDir != expectedDir {
			t.Fatalf("ResolveLogDir() = %q, want %q", logDir, expectedDir)
		}
	})

	t.Run("falls back to current dir when empty", func(t *testing.T) {
		setAppDirsResolverForTest(t, func() (appdirs.Paths, error) {
			return appdirs.Paths{LogDir: " \t "}, nil
		})

		logDir, err := ResolveLogDir()
		if err != nil {
			t.Fatalf("ResolveLogDir() returned unexpected error: %v", err)
		}
		if logDir != "." {
			t.Fatalf("ResolveLogDir() = %q, want %q", logDir, ".")
		}
	})

	t.Run("returns resolver error", func(t *testing.T) {
		setAppDirsResolverForTest(t, func() (appdirs.Paths, error) {
			return appdirs.Paths{}, errors.New("resolve failed")
		})

		_, err := ResolveLogDir()
		if err == nil {
			t.Fatal("ResolveLogDir() returned nil error")
		}
		if !strings.Contains(err.Error(), "resolve failed") {
			t.Fatalf("ResolveLogDir() error = %q, want containing %q", err.Error(), "resolve failed")
		}
	})
}

func TestInitLoggerCreatesLogDirectory(t *testing.T) {
	baseDir := t.TempDir()
	targetLogDir := filepath.Join(baseDir, "data", "logs")
	setAppDirsResolverForTest(t, func() (appdirs.Paths, error) {
		return appdirs.Paths{LogDir: targetLogDir}, nil
	})

	InitLogger()
	if GetLogger() == nil {
		t.Fatal("GetLogger() returned nil after InitLogger()")
	}
	defer GetLogger().Sync()

	GetLogger().Info("logger test line")
	_ = GetLogger().Sync()

	logFilePath := filepath.Join(targetLogDir, logFileName)
	if _, err := os.Stat(logFilePath); err != nil {
		t.Fatalf("expected log file %q to exist: %v", logFilePath, err)
	}
}
