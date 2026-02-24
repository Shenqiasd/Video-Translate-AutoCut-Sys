package config

import (
	"krillin-ai/internal/appdirs"
	"krillin-ai/log"
	"os"
	"path/filepath"
	"testing"
)

func setupPortableTestEnv(t *testing.T, tmp string) {
	t.Helper()

	oldEnv := os.Getenv(appdirs.PortableEnv)
	t.Cleanup(func() {
		if oldEnv == "" {
			_ = os.Unsetenv(appdirs.PortableEnv)
		} else {
			_ = os.Setenv(appdirs.PortableEnv, oldEnv)
		}
	})
	_ = os.Setenv(appdirs.PortableEnv, "true")

	oldExe := os.Getenv("KRILLINAI_TEST_EXECUTABLE")
	t.Cleanup(func() {
		if oldExe == "" {
			_ = os.Unsetenv("KRILLINAI_TEST_EXECUTABLE")
		} else {
			_ = os.Setenv("KRILLINAI_TEST_EXECUTABLE", oldExe)
		}
	})
	_ = os.Setenv("KRILLINAI_TEST_EXECUTABLE", filepath.Join(tmp, "KrillinAI.exe"))
}

func TestSaveConfigCreatesParentDir(t *testing.T) {
	// Initialize logger to avoid nil pointer in tests that use logging
	log.InitLogger()

	tmp := t.TempDir()
	setupPortableTestEnv(t, tmp)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir(tmp): %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

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

func TestLoadOrCreateConfigGeneratesDefaultWhenMissing(t *testing.T) {
	// Initialize logger to avoid nil pointer in tests that use logging
	log.InitLogger()

	tmp := t.TempDir()
	setupPortableTestEnv(t, tmp)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir(tmp): %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	// Ensure no config file exists
	p, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath: %v", err)
	}
	_ = os.RemoveAll(p)

	// Reset Conf to zero values to simulate fresh start
	Conf = Config{}

	// LoadOrCreateConfig should generate default config when file is missing
	created, err := LoadOrCreateConfig()
	if err != nil {
		t.Fatalf("LoadOrCreateConfig: %v", err)
	}
	if !created {
		t.Fatal("expected created=true when config file is missing")
	}

	// Verify config file was created
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected config file to be created at %s: %v", p, err)
	}

	// Verify default values were written (check a few key fields)
	if Conf.App.SegmentDuration != 5 {
		t.Errorf("expected default SegmentDuration=5, got %d", Conf.App.SegmentDuration)
	}
	if Conf.Server.Host != "127.0.0.1" {
		t.Errorf("expected default Server.Host=127.0.0.1, got %s", Conf.Server.Host)
	}
	if Conf.Server.Port != 8888 {
		t.Errorf("expected default Server.Port=8888, got %d", Conf.Server.Port)
	}
}

func TestLoadOrCreateConfigLoadsExisting(t *testing.T) {
	// Initialize logger to avoid nil pointer in tests that use logging
	log.InitLogger()

	tmp := t.TempDir()
	setupPortableTestEnv(t, tmp)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir(tmp): %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	// Set a custom config value and save
	Conf = Config{
		Server: Server{
			Host: "0.0.0.0",
			Port: 9999,
		},
	}
	if err := SaveConfig(); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Reset Conf to zero values
	Conf = Config{}

	// LoadOrCreateConfig should load existing config
	created, err := LoadOrCreateConfig()
	if err != nil {
		t.Fatalf("LoadOrCreateConfig: %v", err)
	}
	if created {
		t.Fatal("expected created=false when config file exists")
	}

	// Verify loaded values
	if Conf.Server.Host != "0.0.0.0" {
		t.Errorf("expected loaded Server.Host=0.0.0.0, got %s", Conf.Server.Host)
	}
	if Conf.Server.Port != 9999 {
		t.Errorf("expected loaded Server.Port=9999, got %d", Conf.Server.Port)
	}
}
