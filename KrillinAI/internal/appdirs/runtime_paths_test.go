package appdirs

import (
	"path/filepath"
	"testing"
)

func TestRuntimePathDerivations(t *testing.T) {
	paths := Paths{
		OutputDir: filepath.Join("var", "krillin", "output"),
		CacheDir:  filepath.Join("var", "krillin", "cache"),
	}

	if got, want := TaskRootFor(paths), filepath.Join("var", "krillin", "output", "tasks"); got != want {
		t.Fatalf("TaskRootFor() = %q, want %q", got, want)
	}

	if got, want := TaskDirFor(paths, "task_123"), filepath.Join("var", "krillin", "output", "tasks", "task_123"); got != want {
		t.Fatalf("TaskDirFor() = %q, want %q", got, want)
	}

	if got, want := UploadRootFor(paths), filepath.Join("var", "krillin", "output", "uploads"); got != want {
		t.Fatalf("UploadRootFor() = %q, want %q", got, want)
	}

	if got, want := DBPathFor(paths), filepath.Join("var", "krillin", "cache", "krillin.db"); got != want {
		t.Fatalf("DBPathFor() = %q, want %q", got, want)
	}
}

func TestRuntimePathDerivationsWithFallbacks(t *testing.T) {
	paths := Paths{}

	if got, want := TaskRootFor(paths), "tasks"; got != want {
		t.Fatalf("TaskRootFor() with empty output dir = %q, want %q", got, want)
	}

	if got, want := UploadRootFor(paths), "uploads"; got != want {
		t.Fatalf("UploadRootFor() with empty output dir = %q, want %q", got, want)
	}

	if got, want := DBPathFor(paths), filepath.Join("cache", "krillin.db"); got != want {
		t.Fatalf("DBPathFor() with empty cache dir = %q, want %q", got, want)
	}
}
