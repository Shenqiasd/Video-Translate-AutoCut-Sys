package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestLoadOrCreateConfigMissingCreatesDefault(t *testing.T) {
	tmp := t.TempDir()

	configPath := filepath.Join(tmp, "config", "config.toml")
	old := resolveConfigPath
	resolveConfigPath = func() (string, error) { return configPath, nil }
	t.Cleanup(func() { resolveConfigPath = old })

	// Ensure missing
	if _, err := os.Stat(configPath); err == nil {
		t.Fatalf("expected config file to be missing")
	}

	created, err := LoadOrCreateConfig()
	if err != nil {
		t.Fatalf("LoadOrCreateConfig() error: %v", err)
	}
	if !created {
		t.Fatalf("LoadOrCreateConfig() created=false, want true")
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to be created: %v", err)
	}

	var got Config
	if _, err := toml.DecodeFile(configPath, &got); err != nil {
		t.Fatalf("decode created config: %v", err)
	}
	if got.Server.Host != "127.0.0.1" {
		t.Fatalf("default server host = %q, want %q", got.Server.Host, "127.0.0.1")
	}
	if got.Server.Port != 8888 {
		t.Fatalf("default server port = %d, want %d", got.Server.Port, 8888)
	}
}

func TestSaveConfigCreatesParentDirs(t *testing.T) {
	tmp := t.TempDir()

	configPath := filepath.Join(tmp, "deep", "nest", "config.toml")
	old := resolveConfigPath
	resolveConfigPath = func() (string, error) { return configPath, nil }
	t.Cleanup(func() { resolveConfigPath = old })

	Conf = defaultConfig()
	Conf.Server.Port = 9999

	if err := SaveConfig(); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(configPath)); err != nil {
		t.Fatalf("expected parent directories to exist: %v", err)
	}

	var got Config
	if _, err := toml.DecodeFile(configPath, &got); err != nil {
		t.Fatalf("decode saved config: %v", err)
	}
	if got.Server.Port != 9999 {
		t.Fatalf("saved server port = %d, want %d", got.Server.Port, 9999)
	}
}
