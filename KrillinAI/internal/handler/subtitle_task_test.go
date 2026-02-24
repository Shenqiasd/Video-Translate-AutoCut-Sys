package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"krillin-ai/internal/appdirs"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configurePathResolverForTest(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	originalResolver := appDirsResolver
	appDirsResolver = func() (appdirs.Paths, error) {
		return appdirs.Paths{
			OutputDir: filepath.Join(tempDir, "output"),
			CacheDir:  filepath.Join(tempDir, "cache"),
		}, nil
	}
	t.Cleanup(func() {
		appDirsResolver = originalResolver
	})
	return tempDir
}

func buildFileRouter() *gin.Engine {
	router := gin.New()
	h := Handler{}
	router.GET("/api/file/*filepath", h.DownloadFile)
	router.HEAD("/api/file/*filepath", h.DownloadFile)
	return router
}

func TestDownloadFile_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configurePathResolverForTest(t)

	router := buildFileRouter()

	// Test 1: File does not exist - should return 404
	req, _ := http.NewRequest("HEAD", "/api/file/tasks/nonexistent/output/test.mp4", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 for non-existent file")
}

func TestDownloadFile_Exists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tempDir := configurePathResolverForTest(t)

	tasksDir := filepath.Join(tempDir, "output", "tasks", "test_task_exists", "output")
	err := os.MkdirAll(tasksDir, 0o755)
	require.NoError(t, err)

	testFile := filepath.Join(tasksDir, "test_file.txt")
	err = os.WriteFile(testFile, []byte("hello world"), 0o644)
	require.NoError(t, err)

	router := buildFileRouter()

	req, _ := http.NewRequest("HEAD", "/api/file/tasks/test_task_exists/output/test_file.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 for existing file")
}

func TestDownloadFile_EmptyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configurePathResolverForTest(t)

	router := buildFileRouter()

	req, _ := http.NewRequest("GET", "/api/file/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "Empty path should not resolve to a file")
}

func TestDownloadFile_GET_ReturnsFileContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tempDir := configurePathResolverForTest(t)

	tasksDir := filepath.Join(tempDir, "output", "tasks", "test_download_task", "output")
	err := os.MkdirAll(tasksDir, 0o755)
	require.NoError(t, err)

	testContent := "This is the file content for testing"
	testFile := filepath.Join(tasksDir, "download_test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0o644)
	require.NoError(t, err)

	router := buildFileRouter()

	req, _ := http.NewRequest("GET", "/api/file/tasks/test_download_task/output/download_test.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "GET should return 200 for existing file")
	assert.Equal(t, testContent, w.Body.String(), "GET should return file content")
}

func TestDownloadFile_PathTraversalBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configurePathResolverForTest(t)

	router := buildFileRouter()
	req, _ := http.NewRequest("GET", "/api/file/tasks/../../etc/passwd", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code, "Traversal path should be blocked")
}
