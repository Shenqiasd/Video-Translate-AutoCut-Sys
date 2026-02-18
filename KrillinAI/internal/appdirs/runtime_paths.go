package appdirs

import (
	"path/filepath"
	"strings"
)

const (
	TaskRootName   = "tasks"
	UploadRootName = "uploads"
	dbFileName     = "krillin.db"
)

func TaskRootFor(paths Paths) string {
	return filepath.Join(normalizeOutputDir(paths.OutputDir), TaskRootName)
}

func TaskDirFor(paths Paths, taskID string) string {
	return filepath.Join(TaskRootFor(paths), taskID)
}

func UploadRootFor(paths Paths) string {
	return filepath.Join(normalizeOutputDir(paths.OutputDir), UploadRootName)
}

func DBPathFor(paths Paths) string {
	return filepath.Join(normalizeCacheDir(paths.CacheDir), dbFileName)
}

func ResolveTaskRoot() (string, error) {
	paths, err := Resolve()
	if err != nil {
		return "", err
	}
	return TaskRootFor(paths), nil
}

func ResolveTaskDir(taskID string) (string, error) {
	paths, err := Resolve()
	if err != nil {
		return "", err
	}
	return TaskDirFor(paths, taskID), nil
}

func ResolveUploadRoot() (string, error) {
	paths, err := Resolve()
	if err != nil {
		return "", err
	}
	return UploadRootFor(paths), nil
}

func ResolveDBPath() (string, error) {
	paths, err := Resolve()
	if err != nil {
		return "", err
	}
	return DBPathFor(paths), nil
}

func normalizeOutputDir(outputDir string) string {
	cleaned := strings.TrimSpace(outputDir)
	if cleaned == "" {
		return "."
	}
	return filepath.Clean(cleaned)
}

func normalizeCacheDir(cacheDir string) string {
	cleaned := strings.TrimSpace(cacheDir)
	if cleaned == "" {
		return "cache"
	}
	return filepath.Clean(cleaned)
}
