package config

import (
	"krillin-ai/internal/appdirs"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveConfigCreatesParentDir(t *testing.T) {
	oldEnv := os.Getenv(appdirs.PortableEnv)
	t.Cleanup(func() {
		if oldEnv == "" {
			_ = os.Unsetenv(appdirs.PortableEnv)
		} else {
			_ = os.Setenv(appdirs.PortableEnv, oldEnv)
		}
	})

	// Force portable so ResolveConfigPath uses the executable directory layout.
	_ = os.Setenv(appdirs.PortableEnv, "true")

	tmp := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir(tmp): %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	// Provide a fake executable path so appdirs.ResolvePortable uses tmp.
	oldExe := os.Getenv("KRILLINAI_TEST_EXECUTABLE")
	t.Cleanup(func() {
		if oldExe == "" {
			_ = os.Unsetenv("KRILLINAI_TEST_EXECUTABLE")
		} else {
			_ = os.Setenv("KRILLINAI_TEST_EXECUTABLE", oldExe)
		}
	})
	_ = os.Setenv("KRILLINAI_TEST_EXECUTABLE", filepath.Join(tmp, "KrillinAI.exe"))

	// Reset to a minimal config value; SaveConfig writes whatever Conf currently contains.
	Conf = Config{}

	if err := SaveConfig(); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	p, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected config file at %s: %v", p, err)
	}
}
