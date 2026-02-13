package appdirs

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	PortableEnv = "KRILLINAI_PORTABLE"

	appName        = "KrillinAI"
	configFileName = "config.toml"
)

type Paths struct {
	Portable   bool
	ConfigDir  string
	ConfigFile string
	LogDir     string
	OutputDir  string
	CacheDir   string
}

type resolveDeps struct {
	goos          string
	getenv        func(string) string
	executable    func() (string, error)
	userConfigDir func() (string, error)
	userCacheDir  func() (string, error)
}

func Resolve() (Paths, error) {
	return resolve(resolveDeps{
		goos:          runtime.GOOS,
		getenv:        os.Getenv,
		executable:    os.Executable,
		userConfigDir: os.UserConfigDir,
		userCacheDir:  os.UserCacheDir,
	})
}

func resolve(rawDeps resolveDeps) (Paths, error) {
	deps := withDefaults(rawDeps)
	if isPortableEnabled(deps.getenv(PortableEnv)) {
		return resolvePortable(deps)
	}
	if deps.goos == "windows" {
		return resolveWindows(deps)
	}
	return defaultNonWindowsPaths(), nil
}

func withDefaults(deps resolveDeps) resolveDeps {
	if deps.goos == "" {
		deps.goos = runtime.GOOS
	}
	if deps.getenv == nil {
		deps.getenv = os.Getenv
	}
	if deps.executable == nil {
		deps.executable = os.Executable
	}
	if deps.userConfigDir == nil {
		deps.userConfigDir = os.UserConfigDir
	}
	if deps.userCacheDir == nil {
		deps.userCacheDir = os.UserCacheDir
	}
	return deps
}

func resolvePortable(deps resolveDeps) (Paths, error) {
	executablePath, err := deps.executable()
	if err != nil {
		return Paths{}, err
	}

	dataDir := filepath.Join(filepath.Dir(executablePath), "data")
	configDir := filepath.Join(dataDir, "config")
	return Paths{
		Portable:   true,
		ConfigDir:  configDir,
		ConfigFile: filepath.Join(configDir, configFileName),
		LogDir:     filepath.Join(dataDir, "logs"),
		OutputDir:  filepath.Join(dataDir, "output"),
		CacheDir:   filepath.Join(dataDir, "cache"),
	}, nil
}

func resolveWindows(deps resolveDeps) (Paths, error) {
	configRoot, err := deps.userConfigDir()
	if err != nil {
		return Paths{}, err
	}
	if strings.TrimSpace(configRoot) == "" {
		return Paths{}, errors.New("user config dir is empty")
	}

	cacheRoot, err := deps.userCacheDir()
	if err != nil {
		return Paths{}, err
	}
	if strings.TrimSpace(cacheRoot) == "" {
		return Paths{}, errors.New("user cache dir is empty")
	}

	configDir := filepath.Join(configRoot, appName)
	cacheBaseDir := filepath.Join(cacheRoot, appName)
	return Paths{
		ConfigDir:  configDir,
		ConfigFile: filepath.Join(configDir, configFileName),
		LogDir:     filepath.Join(cacheBaseDir, "logs"),
		OutputDir:  filepath.Join(cacheBaseDir, "output"),
		CacheDir:   filepath.Join(cacheBaseDir, "cache"),
	}, nil
}

func defaultNonWindowsPaths() Paths {
	configDir := "config"
	return Paths{
		ConfigDir:  configDir,
		ConfigFile: filepath.Join(configDir, configFileName),
		LogDir:     ".",
		OutputDir:  "tasks",
		CacheDir:   "cache",
	}
}

func isPortableEnabled(value string) bool {
	normalized := strings.TrimSpace(strings.ToLower(value))
	return normalized == "1" || normalized == "true"
}
