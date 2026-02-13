package appdirs

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestResolveLayouts(t *testing.T) {
	portableExePath := filepath.Join("/", "apps", "KrillinAI", "KrillinAI.exe")
	portableDataDir := filepath.Join(filepath.Dir(portableExePath), "data")

	windowsConfigRoot := filepath.Join("C:", "Users", "alice", "AppData", "Roaming")
	windowsCacheRoot := filepath.Join("C:", "Users", "alice", "AppData", "Local")

	testCases := []struct {
		name           string
		goos           string
		portableEnv    string
		executablePath string
		userConfigDir  string
		userCacheDir   string
		want           Paths
		wantExeCall    bool
		wantConfigCall bool
		wantCacheCall  bool
	}{
		{
			name:           "portable layout when env is true",
			goos:           "windows",
			portableEnv:    "true",
			executablePath: portableExePath,
			want: Paths{
				Portable:   true,
				ConfigDir:  filepath.Join(portableDataDir, "config"),
				ConfigFile: filepath.Join(portableDataDir, "config", "config.toml"),
				LogDir:     filepath.Join(portableDataDir, "logs"),
				OutputDir:  filepath.Join(portableDataDir, "output"),
				CacheDir:   filepath.Join(portableDataDir, "cache"),
			},
			wantExeCall: true,
		},
		{
			name:          "windows layout when portable mode is disabled",
			goos:          "windows",
			portableEnv:   "",
			userConfigDir: windowsConfigRoot,
			userCacheDir:  windowsCacheRoot,
			want: Paths{
				ConfigDir:  filepath.Join(windowsConfigRoot, "KrillinAI"),
				ConfigFile: filepath.Join(windowsConfigRoot, "KrillinAI", "config.toml"),
				LogDir:     filepath.Join(windowsCacheRoot, "KrillinAI", "logs"),
				OutputDir:  filepath.Join(windowsCacheRoot, "KrillinAI", "output"),
				CacheDir:   filepath.Join(windowsCacheRoot, "KrillinAI", "cache"),
			},
			wantConfigCall: true,
			wantCacheCall:  true,
		},
		{
			name:        "non windows keeps legacy relative defaults",
			goos:        "linux",
			portableEnv: "",
			want: Paths{
				ConfigDir:  "config",
				ConfigFile: filepath.Join("config", "config.toml"),
				LogDir:     ".",
				OutputDir:  "tasks",
				CacheDir:   "cache",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			exeCalled := false
			configCalled := false
			cacheCalled := false

			got, err := resolve(resolveDeps{
				goos: tc.goos,
				getenv: func(key string) string {
					if key == PortableEnv {
						return tc.portableEnv
					}
					return ""
				},
				executable: func() (string, error) {
					exeCalled = true
					return tc.executablePath, nil
				},
				userConfigDir: func() (string, error) {
					configCalled = true
					return tc.userConfigDir, nil
				},
				userCacheDir: func() (string, error) {
					cacheCalled = true
					return tc.userCacheDir, nil
				},
			})
			if err != nil {
				t.Fatalf("resolve() returned unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("resolve() = %+v, want %+v", got, tc.want)
			}

			if exeCalled != tc.wantExeCall {
				t.Fatalf("executable() called = %t, want %t", exeCalled, tc.wantExeCall)
			}
			if configCalled != tc.wantConfigCall {
				t.Fatalf("userConfigDir() called = %t, want %t", configCalled, tc.wantConfigCall)
			}
			if cacheCalled != tc.wantCacheCall {
				t.Fatalf("userCacheDir() called = %t, want %t", cacheCalled, tc.wantCacheCall)
			}
		})
	}
}

func TestResolveErrors(t *testing.T) {
	testCases := []struct {
		name       string
		deps       resolveDeps
		wantErrSub string
	}{
		{
			name: "portable mode returns executable lookup error",
			deps: resolveDeps{
				goos: "windows",
				getenv: func(key string) string {
					if key == PortableEnv {
						return "1"
					}
					return ""
				},
				executable: func() (string, error) {
					return "", errors.New("no executable")
				},
			},
			wantErrSub: "no executable",
		},
		{
			name: "windows mode returns user config error",
			deps: resolveDeps{
				goos:   "windows",
				getenv: func(string) string { return "" },
				userConfigDir: func() (string, error) {
					return "", errors.New("no config dir")
				},
			},
			wantErrSub: "no config dir",
		},
		{
			name: "windows mode returns empty cache path error",
			deps: resolveDeps{
				goos:   "windows",
				getenv: func(string) string { return "" },
				userConfigDir: func() (string, error) {
					return filepath.Join("C:", "Users", "alice", "AppData", "Roaming"), nil
				},
				userCacheDir: func() (string, error) {
					return "   ", nil
				},
			},
			wantErrSub: "user cache dir is empty",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolve(tc.deps)
			if err == nil {
				t.Fatal("resolve() returned nil error")
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("resolve() error = %q, want containing %q", err.Error(), tc.wantErrSub)
			}
		})
	}
}

func TestIsPortableEnabled(t *testing.T) {
	testCases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty value", value: "", want: false},
		{name: "zero", value: "0", want: false},
		{name: "one", value: "1", want: true},
		{name: "true lowercase", value: "true", want: true},
		{name: "true uppercase", value: "TRUE", want: true},
		{name: "trimmed true", value: "  true  ", want: true},
		{name: "false", value: "false", want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := isPortableEnabled(tc.value); got != tc.want {
				t.Fatalf("isPortableEnabled(%q) = %t, want %t", tc.value, got, tc.want)
			}
		})
	}
}
