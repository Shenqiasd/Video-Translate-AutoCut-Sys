package deps

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"krillin-ai/config"
	"krillin-ai/internal/appdirs"
	"krillin-ai/internal/storage"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	DependencyIDFFmpeg  = "ffmpeg"
	DependencyIDFFprobe = "ffprobe"
	DependencyIDYtDlp   = "yt-dlp"

	windowsPackageIDFFmpeg = "ffmpeg-suite"
	windowsPackageIDYtDlp  = "yt-dlp"

	windowsInstallStagePreparing   = "preparing"
	windowsInstallStageDownloading = "downloading"
	windowsInstallStageVerifying   = "verifying"
	windowsInstallStageExtracting  = "extracting"
	windowsInstallStageInstalling  = "installing"
	windowsInstallStageDone        = "done"
)

const (
	ffmpegWindowsVersion = "n7.1.3-40-gcddd06f3b9"
	ffmpegWindowsURL     = "https://github.com/BtbN/FFmpeg-Builds/releases/download/autobuild-2026-02-18-13-03/ffmpeg-n7.1.3-40-gcddd06f3b9-win64-gpl-7.1.zip"
	ffmpegWindowsSHA256  = "8624d6006289c5ca2c1f2f65c19f5812a44261ce9d0fa4c1dc9a45063b3c0735"

	ytDlpWindowsVersion = "2026.01.31"
	ytDlpWindowsURL     = "https://github.com/yt-dlp/yt-dlp/releases/download/2026.01.31/yt-dlp.exe"
	ytDlpWindowsSHA256  = "766b70db21f53d05ae12a8aaefc88421de712360ec28a419046b4157a8a5599c"
)

type InstallProgress struct {
	DependencyID string
	Stage        string
	Message      string
	Downloaded   int64
	Total        int64
	Percent      float64
}

type InstallProgressCallback func(progress InstallProgress)

type windowsPackageFormat string

const (
	windowsPackageFormatZip    windowsPackageFormat = "zip"
	windowsPackageFormatBinary windowsPackageFormat = "binary"
)

type windowsPackageTool struct {
	ID         string
	Executable string
}

type windowsPackageSpec struct {
	ID      string
	Version string
	URL     string
	SHA256  string
	Format  windowsPackageFormat
	Tools   []windowsPackageTool
}

type windowsInstallerOptions struct {
	CacheDir      string
	HTTPClient    *http.Client
	Packages      map[string]windowsPackageSpec
	ToolToPackage map[string]string
	Progress      InstallProgressCallback
}

func defaultWindowsPackages() map[string]windowsPackageSpec {
	return map[string]windowsPackageSpec{
		windowsPackageIDFFmpeg: {
			ID:      windowsPackageIDFFmpeg,
			Version: ffmpegWindowsVersion,
			URL:     ffmpegWindowsURL,
			SHA256:  ffmpegWindowsSHA256,
			Format:  windowsPackageFormatZip,
			Tools: []windowsPackageTool{
				{ID: DependencyIDFFmpeg, Executable: "ffmpeg.exe"},
				{ID: DependencyIDFFprobe, Executable: "ffprobe.exe"},
			},
		},
		windowsPackageIDYtDlp: {
			ID:      windowsPackageIDYtDlp,
			Version: ytDlpWindowsVersion,
			URL:     ytDlpWindowsURL,
			SHA256:  ytDlpWindowsSHA256,
			Format:  windowsPackageFormatBinary,
			Tools: []windowsPackageTool{
				{ID: DependencyIDYtDlp, Executable: "yt-dlp.exe"},
			},
		},
	}
}

func defaultWindowsToolPackageMap() map[string]string {
	return map[string]string{
		DependencyIDFFmpeg:  windowsPackageIDFFmpeg,
		DependencyIDFFprobe: windowsPackageIDFFmpeg,
		DependencyIDYtDlp:   windowsPackageIDYtDlp,
	}
}

func defaultWindowsInstallerOptions() (windowsInstallerOptions, error) {
	cacheDir, err := resolveDependencyCacheDir()
	if err != nil {
		return windowsInstallerOptions{}, err
	}

	httpClient, err := newDownloadHTTPClient(config.Conf.App.Proxy)
	if err != nil {
		return windowsInstallerOptions{}, err
	}

	return windowsInstallerOptions{
		CacheDir:      cacheDir,
		HTTPClient:    httpClient,
		Packages:      defaultWindowsPackages(),
		ToolToPackage: defaultWindowsToolPackageMap(),
	}, nil
}

func resolveDependencyCacheDir() (string, error) {
	paths, err := appdirs.Resolve()
	if err != nil {
		return "", err
	}
	cacheDir := strings.TrimSpace(paths.CacheDir)
	if cacheDir == "" {
		cacheDir = "cache"
	}
	return cacheDir, nil
}

func newDownloadHTTPClient(proxyAddr string) (*http.Client, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	trimmedProxy := strings.TrimSpace(proxyAddr)
	if trimmedProxy != "" {
		proxyURL, err := neturl.Parse(trimmedProxy)
		if err != nil {
			return nil, fmt.Errorf("parse proxy %q: %w", trimmedProxy, err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &http.Client{
		Timeout:   30 * time.Minute,
		Transport: transport,
	}, nil
}

func normalizeDependencyID(dependencyID string) string {
	return strings.ToLower(strings.TrimSpace(dependencyID))
}

func CanAutoInstallDependency(dependencyID string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	normalized := normalizeDependencyID(dependencyID)
	_, ok := defaultWindowsToolPackageMap()[normalized]
	return ok
}

func InstallDependency(dependencyID string, progressCallback InstallProgressCallback) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("automatic dependency install currently supports Windows only")
	}

	normalizedID := normalizeDependencyID(dependencyID)
	if !CanAutoInstallDependency(normalizedID) {
		return fmt.Errorf("dependency %q does not support automatic install", dependencyID)
	}

	options, err := defaultWindowsInstallerOptions()
	if err != nil {
		return err
	}
	options.Progress = progressCallback

	if err = installWindowsDependencyWithOptions(context.Background(), normalizedID, options); err != nil {
		return err
	}
	EnsureManagedDependencyPaths()
	return nil
}

func installWindowsDependencyWithOptions(ctx context.Context, dependencyID string, options windowsInstallerOptions) error {
	normalizedID := normalizeDependencyID(dependencyID)
	if normalizedID == "" {
		return fmt.Errorf("dependency id is empty")
	}

	resolvedOptions, err := options.withDefaults()
	if err != nil {
		return err
	}

	packageID, ok := resolvedOptions.ToolToPackage[normalizedID]
	if !ok {
		return fmt.Errorf("unsupported dependency id %q", normalizedID)
	}

	packageSpec, ok := resolvedOptions.Packages[packageID]
	if !ok {
		return fmt.Errorf("missing package spec for %q", packageID)
	}

	targetPaths, err := packageToolTargetPaths(resolvedOptions.CacheDir, packageSpec)
	if err != nil {
		return err
	}

	if packageToolsInstalled(targetPaths) {
		for toolID, path := range targetPaths {
			setStoragePathForDependency(toolID, path)
		}
		emitInstallProgress(resolvedOptions.Progress, InstallProgress{
			DependencyID: normalizedID,
			Stage:        windowsInstallStageDone,
			Message:      "Dependency already installed",
			Percent:      1,
		})
		return nil
	}

	if err = os.MkdirAll(filepath.Join(resolvedOptions.CacheDir, "bin"), 0o755); err != nil {
		return fmt.Errorf("create install root: %w", err)
	}

	emitInstallProgress(resolvedOptions.Progress, InstallProgress{
		DependencyID: normalizedID,
		Stage:        windowsInstallStagePreparing,
		Message:      fmt.Sprintf("Preparing %s installer", normalizedID),
		Percent:      0,
	})

	downloadPath, err := downloadPackageAndVerifyChecksum(ctx, normalizedID, packageSpec, resolvedOptions)
	if err != nil {
		return err
	}
	defer os.Remove(downloadPath)

	switch packageSpec.Format {
	case windowsPackageFormatZip:
		if err = extractWindowsPackage(downloadPath, targetPaths, normalizedID, resolvedOptions.Progress); err != nil {
			return err
		}
	case windowsPackageFormatBinary:
		if err = installWindowsBinary(downloadPath, targetPaths, normalizedID, resolvedOptions.Progress); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported package format %q", packageSpec.Format)
	}

	for toolID, path := range targetPaths {
		setStoragePathForDependency(toolID, path)
	}

	emitInstallProgress(resolvedOptions.Progress, InstallProgress{
		DependencyID: normalizedID,
		Stage:        windowsInstallStageDone,
		Message:      fmt.Sprintf("%s installed successfully", normalizedID),
		Percent:      1,
	})
	return nil
}

func (options windowsInstallerOptions) withDefaults() (windowsInstallerOptions, error) {
	if strings.TrimSpace(options.CacheDir) == "" {
		options.CacheDir = "cache"
	}
	if options.HTTPClient == nil {
		httpClient, err := newDownloadHTTPClient("")
		if err != nil {
			return windowsInstallerOptions{}, err
		}
		options.HTTPClient = httpClient
	}
	if options.Packages == nil {
		options.Packages = defaultWindowsPackages()
	}
	if options.ToolToPackage == nil {
		options.ToolToPackage = defaultWindowsToolPackageMap()
	}
	return options, nil
}

func packageToolTargetPaths(cacheDir string, packageSpec windowsPackageSpec) (map[string]string, error) {
	targetPaths := make(map[string]string, len(packageSpec.Tools))
	for _, tool := range packageSpec.Tools {
		if tool.ID == "" {
			return nil, fmt.Errorf("package %q contains empty tool id", packageSpec.ID)
		}
		if tool.Executable == "" {
			return nil, fmt.Errorf("package %q has empty executable for %q", packageSpec.ID, tool.ID)
		}
		targetPaths[tool.ID] = filepath.Join(cacheDir, "bin", tool.ID, tool.Executable)
	}
	return targetPaths, nil
}

func packageToolsInstalled(targetPaths map[string]string) bool {
	if len(targetPaths) == 0 {
		return false
	}
	for _, path := range targetPaths {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}

func downloadPackageAndVerifyChecksum(
	ctx context.Context,
	dependencyID string,
	packageSpec windowsPackageSpec,
	options windowsInstallerOptions,
) (string, error) {
	downloadDir := filepath.Join(options.CacheDir, "bin", "downloads")
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return "", fmt.Errorf("create download directory: %w", err)
	}

	tempFile, err := os.CreateTemp(downloadDir, packageSpec.ID+"-*.pkg")
	if err != nil {
		return "", fmt.Errorf("create temp download file: %w", err)
	}
	downloadPath := tempFile.Name()
	if err = tempFile.Close(); err != nil {
		os.Remove(downloadPath)
		return "", fmt.Errorf("close temp download file: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, packageSpec.URL, nil)
	if err != nil {
		os.Remove(downloadPath)
		return "", err
	}

	resp, err := options.HTTPClient.Do(req)
	if err != nil {
		os.Remove(downloadPath)
		return "", fmt.Errorf("download %s: %w", packageSpec.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Remove(downloadPath)
		return "", fmt.Errorf("download %s: unexpected status %s", packageSpec.URL, resp.Status)
	}

	out, err := os.Create(downloadPath)
	if err != nil {
		os.Remove(downloadPath)
		return "", fmt.Errorf("create download file: %w", err)
	}
	defer out.Close()

	hasher := sha256.New()
	total := resp.ContentLength
	var downloaded int64
	var lastProgress time.Time
	buf := make([]byte, 64*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if _, err = out.Write(chunk); err != nil {
				os.Remove(downloadPath)
				return "", fmt.Errorf("write download file: %w", err)
			}
			if _, err = hasher.Write(chunk); err != nil {
				os.Remove(downloadPath)
				return "", fmt.Errorf("hash download file: %w", err)
			}

			downloaded += int64(n)
			if time.Since(lastProgress) >= 120*time.Millisecond || (total > 0 && downloaded >= total) {
				percent := 0.75
				if total > 0 {
					percent = 0.75 * float64(downloaded) / float64(total)
				}
				emitInstallProgress(options.Progress, InstallProgress{
					DependencyID: dependencyID,
					Stage:        windowsInstallStageDownloading,
					Message:      fmt.Sprintf("Downloading %s", packageSpec.ID),
					Downloaded:   downloaded,
					Total:        total,
					Percent:      percent,
				})
				lastProgress = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			os.Remove(downloadPath)
			return "", fmt.Errorf("read download response: %w", readErr)
		}
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	expectedChecksum := strings.ToLower(strings.TrimSpace(packageSpec.SHA256))
	emitInstallProgress(options.Progress, InstallProgress{
		DependencyID: dependencyID,
		Stage:        windowsInstallStageVerifying,
		Message:      fmt.Sprintf("Verifying checksum for %s", packageSpec.ID),
		Downloaded:   downloaded,
		Total:        total,
		Percent:      0.85,
	})

	if expectedChecksum != "" && actualChecksum != expectedChecksum {
		os.Remove(downloadPath)
		return "", fmt.Errorf(
			"checksum mismatch for %s: expected %s, got %s",
			packageSpec.ID,
			expectedChecksum,
			actualChecksum,
		)
	}

	return downloadPath, nil
}

func extractWindowsPackage(
	archivePath string,
	targetPaths map[string]string,
	dependencyID string,
	progressCallback InstallProgressCallback,
) error {
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip archive: %w", err)
	}
	defer zipReader.Close()

	matchedTools := make(map[string]bool, len(targetPaths))
	targetByExecutable := make(map[string]string, len(targetPaths))
	for toolID, targetPath := range targetPaths {
		targetByExecutable[strings.ToLower(filepath.Base(targetPath))] = toolID
	}

	totalEntries := len(zipReader.File)
	if totalEntries == 0 {
		return fmt.Errorf("zip archive is empty")
	}

	for i, file := range zipReader.File {
		progressPercent := 0.85 + 0.1*float64(i+1)/float64(totalEntries)
		emitInstallProgress(progressCallback, InstallProgress{
			DependencyID: dependencyID,
			Stage:        windowsInstallStageExtracting,
			Message:      "Extracting dependency package",
			Percent:      progressPercent,
		})

		if file.FileInfo().IsDir() {
			continue
		}

		fileName := strings.ToLower(filepath.Base(file.Name))
		toolID, ok := targetByExecutable[fileName]
		if !ok {
			continue
		}

		if err = extractZipEntryToPath(file, targetPaths[toolID]); err != nil {
			return err
		}
		matchedTools[toolID] = true
	}

	var missingTools []string
	for toolID := range targetPaths {
		if !matchedTools[toolID] {
			missingTools = append(missingTools, toolID)
		}
	}
	if len(missingTools) > 0 {
		return fmt.Errorf("archive missing executables for: %s", strings.Join(missingTools, ", "))
	}

	return nil
}

func extractZipEntryToPath(file *zip.File, targetPath string) error {
	sourceReader, err := file.Open()
	if err != nil {
		return fmt.Errorf("open zip entry %q: %w", file.Name, err)
	}
	defer sourceReader.Close()

	if err = os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create target file %q: %w", targetPath, err)
	}
	defer targetFile.Close()

	if _, err = io.Copy(targetFile, sourceReader); err != nil {
		return fmt.Errorf("copy zip entry to %q: %w", targetPath, err)
	}

	if err = os.Chmod(targetPath, 0o755); err != nil {
		return fmt.Errorf("chmod %q: %w", targetPath, err)
	}
	return nil
}

func installWindowsBinary(
	sourcePath string,
	targetPaths map[string]string,
	dependencyID string,
	progressCallback InstallProgressCallback,
) error {
	if len(targetPaths) == 0 {
		return fmt.Errorf("binary package has no targets")
	}

	emitInstallProgress(progressCallback, InstallProgress{
		DependencyID: dependencyID,
		Stage:        windowsInstallStageInstalling,
		Message:      "Installing dependency binary",
		Percent:      0.9,
	})

	for _, targetPath := range targetPaths {
		if err := copyFile(sourcePath, targetPath); err != nil {
			return err
		}
		if err := os.Chmod(targetPath, 0o755); err != nil {
			return fmt.Errorf("chmod %q: %w", targetPath, err)
		}
	}

	return nil
}

func copyFile(sourcePath, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file %q: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	if err = os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create target file %q: %w", targetPath, err)
	}
	defer targetFile.Close()

	if _, err = io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("copy file to %q: %w", targetPath, err)
	}
	return nil
}

func emitInstallProgress(callback InstallProgressCallback, progress InstallProgress) {
	if callback == nil {
		return
	}
	if progress.Percent < 0 {
		progress.Percent = 0
	}
	if progress.Percent > 1 {
		progress.Percent = 1
	}
	callback(progress)
}

func managedDependencyExecutableName(dependencyID string) (string, bool) {
	normalized := normalizeDependencyID(dependencyID)
	switch normalized {
	case DependencyIDFFmpeg:
		return "ffmpeg.exe", true
	case DependencyIDFFprobe:
		return "ffprobe.exe", true
	case DependencyIDYtDlp:
		return "yt-dlp.exe", true
	default:
		return "", false
	}
}

func resolveManagedDependencyPathForCache(cacheDir, dependencyID string) (string, error) {
	executableName, ok := managedDependencyExecutableName(dependencyID)
	if !ok {
		return "", fmt.Errorf("unsupported dependency id %q", dependencyID)
	}
	return filepath.Join(cacheDir, "bin", normalizeDependencyID(dependencyID), executableName), nil
}

func ResolveManagedDependencyPath(dependencyID string) (string, error) {
	cacheDir, err := resolveDependencyCacheDir()
	if err != nil {
		return "", err
	}
	return resolveManagedDependencyPathForCache(cacheDir, dependencyID)
}

func EnsureManagedDependencyPaths() {
	for _, dependencyID := range []string{DependencyIDFFmpeg, DependencyIDFFprobe, DependencyIDYtDlp} {
		existingPath := getStoragePathForDependency(dependencyID)
		if resolvedPath, ok := resolveExistingBinaryPath(existingPath); ok {
			setStoragePathForDependency(dependencyID, resolvedPath)
			continue
		}

		managedPath, err := ResolveManagedDependencyPath(dependencyID)
		if err != nil {
			continue
		}
		if _, err = os.Stat(managedPath); err != nil {
			continue
		}
		setStoragePathForDependency(dependencyID, managedPath)
	}
}

func getStoragePathForDependency(dependencyID string) string {
	switch normalizeDependencyID(dependencyID) {
	case DependencyIDFFmpeg:
		return storage.FfmpegPath
	case DependencyIDFFprobe:
		return storage.FfprobePath
	case DependencyIDYtDlp:
		return storage.YtdlpPath
	default:
		return ""
	}
}

func setStoragePathForDependency(dependencyID, path string) {
	switch normalizeDependencyID(dependencyID) {
	case DependencyIDFFmpeg:
		storage.FfmpegPath = path
	case DependencyIDFFprobe:
		storage.FfprobePath = path
	case DependencyIDYtDlp:
		storage.YtdlpPath = path
	}
}

func resolveExistingBinaryPath(configuredPath string) (string, bool) {
	cleanedPath := strings.TrimSpace(configuredPath)
	if cleanedPath == "" {
		return "", false
	}

	if resolvedPath, err := exec.LookPath(cleanedPath); err == nil {
		return resolvedPath, true
	}

	absPath, err := filepath.Abs(cleanedPath)
	if err != nil {
		return "", false
	}
	if _, err = os.Stat(absPath); err != nil {
		return "", false
	}
	return absPath, true
}
