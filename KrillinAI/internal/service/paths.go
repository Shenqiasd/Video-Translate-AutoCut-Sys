package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"krillin-ai/internal/appdirs"
)

var appDirsResolver = appdirs.Resolve

func resolveTaskRoot() (string, error) {
	dirs, err := appDirsResolver()
	if err != nil {
		return "", err
	}
	return appdirs.TaskRootFor(dirs), nil
}

func resolveTaskDir(taskID string) (string, error) {
	if strings.TrimSpace(taskID) == "" {
		return "", fmt.Errorf("task id is empty")
	}

	taskRoot, err := resolveTaskRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(taskRoot, taskID), nil
}

func resolveTaskDownloadPath(localPath string) (string, error) {
	dirs, err := appDirsResolver()
	if err != nil {
		return "", err
	}

	taskRoot := appdirs.TaskRootFor(dirs)
	cleanedLocalPath := filepath.Clean(localPath)
	relPath, err := filepath.Rel(taskRoot, cleanedLocalPath)
	if err != nil {
		return "", err
	}
	if relPath == "." || relPath == "" {
		return "", fmt.Errorf("task artifact path %q is not a file path", localPath)
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("task artifact path %q is outside task root %q", localPath, taskRoot)
	}
	return filepath.ToSlash(filepath.Join(appdirs.TaskRootName, relPath)), nil
}

func resolveCacheDirPath(pathItems ...string) (string, error) {
	dirs, err := appDirsResolver()
	if err != nil {
		return "", err
	}
	cacheRoot := strings.TrimSpace(dirs.CacheDir)
	if cacheRoot == "" {
		cacheRoot = "cache"
	}
	return filepath.Join(append([]string{cacheRoot}, pathItems...)...), nil
}
