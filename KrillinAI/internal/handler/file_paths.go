package handler

import (
	"os"
	"path/filepath"
	"strings"

	"krillin-ai/internal/appdirs"
)

var appDirsResolver = appdirs.Resolve

type downloadRoot struct {
	alias string
	dirs  []string
}

func taskDirCandidates(taskID string) []string {
	candidates := make([]string, 0, 2)
	if dirs, err := appDirsResolver(); err == nil {
		candidates = append(candidates, appdirs.TaskDirFor(dirs, taskID))
	}
	candidates = append(candidates, filepath.Join("tasks", taskID))
	return uniquePaths(candidates...)
}

func uploadRootCandidates() []string {
	candidates := make([]string, 0, 2)
	if dirs, err := appDirsResolver(); err == nil {
		candidates = append(candidates, appdirs.UploadRootFor(dirs))
	}
	candidates = append(candidates, "uploads")
	return uniquePaths(candidates...)
}

func preferredUploadRoot() string {
	candidates := uploadRootCandidates()
	if len(candidates) == 0 {
		return "uploads"
	}
	return candidates[0]
}

func resolveDownloadPath(requested string) (string, bool) {
	requested = strings.TrimSpace(requested)
	requested = strings.TrimPrefix(requested, string(filepath.Separator))
	requested = strings.TrimPrefix(requested, "/")
	if hasParentTraversal(requested) {
		return "", false
	}
	requested = filepath.Clean(requested)
	if requested == "." {
		requested = ""
	}
	requested = filepath.ToSlash(requested)

	roots := []downloadRoot{
		{alias: appdirs.TaskRootName, dirs: taskRootCandidates()},
		{alias: appdirs.UploadRootName, dirs: uploadRootCandidates()},
		{alias: "static", dirs: []string{"static"}},
	}

	matchedAlias := ""
	relativePath := requested
	for _, root := range roots {
		prefix := root.alias + "/"
		if requested == root.alias {
			matchedAlias = root.alias
			relativePath = ""
			break
		}
		if strings.HasPrefix(requested, prefix) {
			matchedAlias = root.alias
			relativePath = strings.TrimPrefix(requested, prefix)
			break
		}
	}

	var fallback string
	for _, root := range roots {
		if matchedAlias != "" && root.alias != matchedAlias {
			continue
		}

		pathToJoin := requested
		if matchedAlias == root.alias {
			pathToJoin = relativePath
		}
		pathToJoin = filepath.FromSlash(pathToJoin)

		for _, rootDir := range root.dirs {
			candidate := filepath.Clean(filepath.Join(rootDir, pathToJoin))
			if !isPathWithinRoot(rootDir, candidate) {
				continue
			}
			if fallback == "" {
				fallback = candidate
			}
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, true
			}
		}
	}

	if fallback == "" {
		return "", false
	}
	return fallback, true
}

func taskRootCandidates() []string {
	candidates := make([]string, 0, 2)
	if dirs, err := appDirsResolver(); err == nil {
		candidates = append(candidates, appdirs.TaskRootFor(dirs))
	}
	candidates = append(candidates, "tasks")
	return uniquePaths(candidates...)
}

func uniquePaths(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	paths := make([]string, 0, len(values))
	for _, value := range values {
		cleaned := strings.TrimSpace(value)
		if cleaned == "" {
			continue
		}
		cleaned = filepath.Clean(cleaned)
		if _, exists := seen[cleaned]; exists {
			continue
		}
		seen[cleaned] = struct{}{}
		paths = append(paths, cleaned)
	}
	return paths
}

func isPathWithinRoot(root, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)

	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func hasParentTraversal(path string) bool {
	normalized := strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}
	return false
}
