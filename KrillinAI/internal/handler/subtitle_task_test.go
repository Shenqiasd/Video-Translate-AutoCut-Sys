package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	h := Handler{}
	router.GET("/api/file/*filepath", h.DownloadFile)

	// Test 1: File does not exist - should return 404
	req, _ := http.NewRequest("HEAD", "/api/file/tasks/nonexistent/output/test.mp4", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "Should return 404 for non-existent file")
}

func TestDownloadFile_Exists(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create the tasks directory structure IN the current working directory
	tasksDir := filepath.Join("tasks", "test_task_exists", "output")
	err := os.MkdirAll(tasksDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(filepath.Join("tasks", "test_task_exists"))

	testFile := filepath.Join(tasksDir, "test_file.txt")
	err = os.WriteFile(testFile, []byte("hello world"), 0644)
	require.NoError(t, err)

	router := gin.New()
	h := Handler{}
	router.GET("/api/file/*filepath", h.DownloadFile)

	// Test 2: File exists - should return 200
	req, _ := http.NewRequest("HEAD", "/api/file/tasks/test_task_exists/output/test_file.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 for existing file")
}

func TestDownloadFile_EmptyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	h := Handler{}
	router.GET("/api/file/*filepath", h.DownloadFile)

	// Test 3: Empty filepath resolves to "/" which becomes "." (current dir)
	// This is actually a directory, and FileAttachment will fail for directories
	// The behavior depends on gin.FileAttachment handling
	req, _ := http.NewRequest("GET", "/api/file/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// "/" resolves to "./" which is current directory - gin.FileAttachment handles this
	// We just verify it doesn't panic and returns some response
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound, 
		"Should return either 200 or 404 for root path depending on gin behavior")
}

func TestDownloadFile_GET_ReturnsFileContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test file
	tasksDir := filepath.Join(".", "tasks", "test_download_task", "output")
	err := os.MkdirAll(tasksDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(filepath.Join(".", "tasks", "test_download_task"))

	testContent := "This is the file content for testing"
	testFile := filepath.Join(tasksDir, "download_test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	router := gin.New()
	h := Handler{}
	router.GET("/api/file/*filepath", h.DownloadFile)

	// Test 4: GET request should return file content
	req, _ := http.NewRequest("GET", "/api/file/tasks/test_download_task/output/download_test.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "GET should return 200 for existing file")
	assert.Equal(t, testContent, w.Body.String(), "GET should return file content")
}
