package deps

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func notFoundErr(command string) error {
	return &exec.Error{Name: command, Err: exec.ErrNotFound}
}

func TestPathResolverResolvePrefersStoragePath(t *testing.T) {
	binPath := filepath.Join(t.TempDir(), "ffmpeg-custom")
	if err := os.WriteFile(binPath, []byte("ffmpeg"), 0o755); err != nil {
		t.Fatalf("os.WriteFile() failed: %v", err)
	}

	resolver := NewPathResolver()
	resolver.LookPath = func(file string) (string, error) {
		return "", notFoundErr(file)
	}

	state := resolver.Resolve(DependencySpec{
		Name:        "ffmpeg",
		Command:     "ffmpeg",
		StoragePath: binPath,
	})

	if state.Status != DependencyStatusOK {
		t.Fatalf("state.Status = %q, want %q", state.Status, DependencyStatusOK)
	}
	if state.Source != DependencySourceStorage {
		t.Fatalf("state.Source = %q, want %q", state.Source, DependencySourceStorage)
	}
	if state.ResolvedPath != binPath {
		t.Fatalf("state.ResolvedPath = %q, want %q", state.ResolvedPath, binPath)
	}
}

func TestPathResolverResolveFallsBackToLookPath(t *testing.T) {
	resolver := NewPathResolver()
	resolver.LookPath = func(file string) (string, error) {
		if file != "ffmpeg" {
			t.Fatalf("LookPath() received %q, want %q", file, "ffmpeg")
		}
		return "/mock/bin/ffmpeg", nil
	}

	state := resolver.Resolve(DependencySpec{Name: "ffmpeg", Command: "ffmpeg"})

	if state.Status != DependencyStatusOK {
		t.Fatalf("state.Status = %q, want %q", state.Status, DependencyStatusOK)
	}
	if state.Source != DependencySourceLookPath {
		t.Fatalf("state.Source = %q, want %q", state.Source, DependencySourceLookPath)
	}
	if state.ResolvedPath != "/mock/bin/ffmpeg" {
		t.Fatalf("state.ResolvedPath = %q, want %q", state.ResolvedPath, "/mock/bin/ffmpeg")
	}
}

func TestPathResolverResolveReportsMissingWhenNotFound(t *testing.T) {
	resolver := NewPathResolver()
	resolver.LookPath = func(file string) (string, error) {
		return "", notFoundErr(file)
	}

	state := resolver.Resolve(DependencySpec{Name: "ffmpeg", Command: "ffmpeg"})

	if state.Status != DependencyStatusMissing {
		t.Fatalf("state.Status = %q, want %q", state.Status, DependencyStatusMissing)
	}
	if state.Source != DependencySourceLookPath {
		t.Fatalf("state.Source = %q, want %q", state.Source, DependencySourceLookPath)
	}
	if state.ResolvedPath != "" {
		t.Fatalf("state.ResolvedPath = %q, want empty", state.ResolvedPath)
	}
	if state.Error == "" {
		t.Fatalf("state.Error should not be empty")
	}
}

func TestPathResolverResolveConfiguredMissingReturnsMissing(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing-ffmpeg")

	resolver := NewPathResolver()
	resolver.LookPath = func(file string) (string, error) {
		return "", notFoundErr(file)
	}

	state := resolver.Resolve(DependencySpec{
		Name:        "ffmpeg",
		Command:     "ffmpeg",
		StoragePath: missingPath,
	})

	if state.Status != DependencyStatusMissing {
		t.Fatalf("state.Status = %q, want %q", state.Status, DependencyStatusMissing)
	}
	if state.Source != DependencySourceStorage {
		t.Fatalf("state.Source = %q, want %q", state.Source, DependencySourceStorage)
	}
	if state.ResolvedPath != missingPath {
		t.Fatalf("state.ResolvedPath = %q, want %q", state.ResolvedPath, missingPath)
	}
	if state.Error == "" {
		t.Fatalf("state.Error should not be empty")
	}
}

func TestPathResolverResolveConfiguredStatFailureReturnsError(t *testing.T) {
	resolver := NewPathResolver()
	resolver.LookPath = func(file string) (string, error) {
		return "", notFoundErr(file)
	}
	resolver.AbsPath = func(path string) (string, error) {
		return "/mock/configured/path", nil
	}
	resolver.Stat = func(name string) (os.FileInfo, error) {
		if name != "/mock/configured/path" {
			t.Fatalf("Stat() received %q, want %q", name, "/mock/configured/path")
		}
		return nil, errors.New("permission denied")
	}

	state := resolver.Resolve(DependencySpec{
		Name:        "ffmpeg",
		Command:     "ffmpeg",
		StoragePath: "ignored",
	})

	if state.Status != DependencyStatusError {
		t.Fatalf("state.Status = %q, want %q", state.Status, DependencyStatusError)
	}
	if state.Source != DependencySourceStorage {
		t.Fatalf("state.Source = %q, want %q", state.Source, DependencySourceStorage)
	}
	if state.ResolvedPath != "/mock/configured/path" {
		t.Fatalf("state.ResolvedPath = %q, want %q", state.ResolvedPath, "/mock/configured/path")
	}
	if !strings.Contains(state.Error, "permission denied") {
		t.Fatalf("state.Error = %q, want to contain %q", state.Error, "permission denied")
	}
}

func TestBuildDependencyInventorySetsEdgeTierByProvider(t *testing.T) {
	withEdge := BuildDependencyInventory("openai", "edge-tts")
	withoutEdge := BuildDependencyInventory("openai", "openai")

	withEdgeSpec, ok := findDependencySpec(withEdge, "edge-tts")
	if !ok {
		t.Fatalf("edge-tts spec not found")
	}
	withoutEdgeSpec, ok := findDependencySpec(withoutEdge, "edge-tts")
	if !ok {
		t.Fatalf("edge-tts spec not found")
	}

	if withEdgeSpec.Tier != DependencyTierShould {
		t.Fatalf("withEdgeSpec.Tier = %q, want %q", withEdgeSpec.Tier, DependencyTierShould)
	}
	if withoutEdgeSpec.Tier != DependencyTierOptional {
		t.Fatalf("withoutEdgeSpec.Tier = %q, want %q", withoutEdgeSpec.Tier, DependencyTierOptional)
	}
}

func findDependencySpec(specs []DependencySpec, id string) (DependencySpec, bool) {
	for _, spec := range specs {
		if spec.ID == id {
			return spec, true
		}
	}
	return DependencySpec{}, false
}
