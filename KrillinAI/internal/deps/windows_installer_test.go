package deps

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"krillin-ai/internal/storage"
)

func TestInstallWindowsDependencyWithZipPackage(t *testing.T) {
	originalFfmpegPath := storage.FfmpegPath
	originalFfprobePath := storage.FfprobePath
	t.Cleanup(func() {
		storage.FfmpegPath = originalFfmpegPath
		storage.FfprobePath = originalFfprobePath
	})

	archiveBytes := mustBuildZipArchive(t, map[string][]byte{
		"ffmpeg-build/bin/ffmpeg.exe":  []byte("fake-ffmpeg-binary"),
		"ffmpeg-build/bin/ffprobe.exe": []byte("fake-ffprobe-binary"),
		"ffmpeg-build/doc/readme.txt":  []byte("not-needed"),
	})
	checksum := sha256Hex(archiveBytes)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/ffmpeg.zip" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Length", strconv.Itoa(len(archiveBytes)))
		_, _ = writer.Write(archiveBytes)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	var progressStages []string
	err := installWindowsDependencyWithOptions(context.Background(), DependencyIDFFmpeg, windowsInstallerOptions{
		CacheDir:   cacheDir,
		HTTPClient: server.Client(),
		Packages: map[string]windowsPackageSpec{
			windowsPackageIDFFmpeg: {
				ID:      windowsPackageIDFFmpeg,
				Version: "test",
				URL:     server.URL + "/ffmpeg.zip",
				SHA256:  checksum,
				Format:  windowsPackageFormatZip,
				Tools: []windowsPackageTool{
					{ID: DependencyIDFFmpeg, Executable: "ffmpeg.exe"},
					{ID: DependencyIDFFprobe, Executable: "ffprobe.exe"},
				},
			},
		},
		ToolToPackage: map[string]string{
			DependencyIDFFmpeg:  windowsPackageIDFFmpeg,
			DependencyIDFFprobe: windowsPackageIDFFmpeg,
		},
		Progress: func(progress InstallProgress) {
			progressStages = append(progressStages, progress.Stage)
		},
	})
	if err != nil {
		t.Fatalf("installWindowsDependencyWithOptions() error = %v", err)
	}

	ffmpegPath := filepath.Join(cacheDir, "bin", "ffmpeg", "ffmpeg.exe")
	ffprobePath := filepath.Join(cacheDir, "bin", "ffprobe", "ffprobe.exe")

	ffmpegData, err := os.ReadFile(ffmpegPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", ffmpegPath, err)
	}
	ffprobeData, err := os.ReadFile(ffprobePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", ffprobePath, err)
	}

	if string(ffmpegData) != "fake-ffmpeg-binary" {
		t.Fatalf("ffmpeg content = %q, want %q", string(ffmpegData), "fake-ffmpeg-binary")
	}
	if string(ffprobeData) != "fake-ffprobe-binary" {
		t.Fatalf("ffprobe content = %q, want %q", string(ffprobeData), "fake-ffprobe-binary")
	}
	if storage.FfmpegPath != ffmpegPath {
		t.Fatalf("storage.FfmpegPath = %q, want %q", storage.FfmpegPath, ffmpegPath)
	}
	if storage.FfprobePath != ffprobePath {
		t.Fatalf("storage.FfprobePath = %q, want %q", storage.FfprobePath, ffprobePath)
	}

	if !containsProgressStage(progressStages, windowsInstallStageDownloading) {
		t.Fatalf("progress stages %v do not contain %q", progressStages, windowsInstallStageDownloading)
	}
	if !containsProgressStage(progressStages, windowsInstallStageDone) {
		t.Fatalf("progress stages %v do not contain %q", progressStages, windowsInstallStageDone)
	}
}

func TestInstallWindowsDependencyWithBinaryPackage(t *testing.T) {
	originalYtdlpPath := storage.YtdlpPath
	t.Cleanup(func() {
		storage.YtdlpPath = originalYtdlpPath
	})

	binaryBytes := []byte("fake-yt-dlp-binary")
	checksum := sha256Hex(binaryBytes)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/yt-dlp.exe" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Length", strconv.Itoa(len(binaryBytes)))
		_, _ = writer.Write(binaryBytes)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	err := installWindowsDependencyWithOptions(context.Background(), DependencyIDYtDlp, windowsInstallerOptions{
		CacheDir:   cacheDir,
		HTTPClient: server.Client(),
		Packages: map[string]windowsPackageSpec{
			windowsPackageIDYtDlp: {
				ID:      windowsPackageIDYtDlp,
				Version: "test",
				URL:     server.URL + "/yt-dlp.exe",
				SHA256:  checksum,
				Format:  windowsPackageFormatBinary,
				Tools: []windowsPackageTool{
					{ID: DependencyIDYtDlp, Executable: "yt-dlp.exe"},
				},
			},
		},
		ToolToPackage: map[string]string{
			DependencyIDYtDlp: windowsPackageIDYtDlp,
		},
	})
	if err != nil {
		t.Fatalf("installWindowsDependencyWithOptions() error = %v", err)
	}

	targetPath := filepath.Join(cacheDir, "bin", "yt-dlp", "yt-dlp.exe")
	targetData, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", targetPath, err)
	}
	if string(targetData) != "fake-yt-dlp-binary" {
		t.Fatalf("yt-dlp content = %q, want %q", string(targetData), "fake-yt-dlp-binary")
	}
	if storage.YtdlpPath != targetPath {
		t.Fatalf("storage.YtdlpPath = %q, want %q", storage.YtdlpPath, targetPath)
	}
}

func TestInstallWindowsDependencyChecksumMismatch(t *testing.T) {
	binaryBytes := []byte("fake-binary")

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/yt-dlp.exe" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Length", strconv.Itoa(len(binaryBytes)))
		_, _ = writer.Write(binaryBytes)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	err := installWindowsDependencyWithOptions(context.Background(), DependencyIDYtDlp, windowsInstallerOptions{
		CacheDir:   cacheDir,
		HTTPClient: server.Client(),
		Packages: map[string]windowsPackageSpec{
			windowsPackageIDYtDlp: {
				ID:      windowsPackageIDYtDlp,
				Version: "test",
				URL:     server.URL + "/yt-dlp.exe",
				SHA256:  strings.Repeat("0", 64),
				Format:  windowsPackageFormatBinary,
				Tools: []windowsPackageTool{
					{ID: DependencyIDYtDlp, Executable: "yt-dlp.exe"},
				},
			},
		},
		ToolToPackage: map[string]string{
			DependencyIDYtDlp: windowsPackageIDYtDlp,
		},
	})
	if err == nil {
		t.Fatalf("installWindowsDependencyWithOptions() expected checksum error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "checksum mismatch")
	}

	targetPath := filepath.Join(cacheDir, "bin", "yt-dlp", "yt-dlp.exe")
	if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
		t.Fatalf("os.Stat(%q) error = %v, want not exists", targetPath, statErr)
	}
}

func mustBuildZipArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	for name, content := range files {
		entry, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("zipWriter.Create(%q) error = %v", name, err)
		}
		if _, err = entry.Write(content); err != nil {
			t.Fatalf("entry.Write(%q) error = %v", name, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("zipWriter.Close() error = %v", err)
	}

	return buffer.Bytes()
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func containsProgressStage(stages []string, target string) bool {
	for _, stage := range stages {
		if stage == target {
			return true
		}
	}
	return false
}
