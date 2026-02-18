package main

import (
	"flag"
	"fmt"
	"krillin-ai/internal/appdirs"
	"krillin-ai/internal/storage"
	"krillin-ai/log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	appDirsResolver    = appdirs.Resolve
	logFileResolver    = log.ResolveLogFilePath
	lookPathResolver   = exec.LookPath
	mkdirAllResolver   = os.MkdirAll
	createTempResolver = os.CreateTemp
	removeResolver     = os.Remove
)

func handleCLIFlags() (bool, int) {
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	showVersion := flags.Bool("version", false, "print version information")
	showDiagnose := flags.Bool("diagnose", false, "print runtime diagnostics")

	if err := flags.Parse(os.Args[1:]); err != nil {
		return true, 2
	}

	if !*showVersion && !*showDiagnose {
		return false, 0
	}

	if *showVersion {
		printVersion()
	}

	if *showDiagnose {
		if *showVersion {
			fmt.Println()
		}
		printDiagnose()
	}

	return true, 0
}

func printVersion() {
	fmt.Printf("version: %s\ncommit: %s\ndate: %s\n", version, commit, date)
}

func printDiagnose() {
	fmt.Printf("runtime: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("version: %s\n", version)
	fmt.Printf("commit: %s\n", commit)
	fmt.Printf("date: %s\n", date)

	if wd, err := os.Getwd(); err == nil {
		fmt.Printf("working_dir: %s\n", wd)
	} else {
		fmt.Printf("working_dir: <error: %v>\n", err)
	}

	if exePath, err := os.Executable(); err == nil {
		fmt.Printf("executable: %s\n", exePath)
	} else {
		fmt.Printf("executable: <error: %v>\n", err)
	}

	dirs, err := appDirsResolver()
	if err != nil {
		fmt.Printf("paths: <error: %v>\n", err)
	} else {
		logDir := effectiveDir(dirs.LogDir, ".")
		taskRoot := appdirs.TaskRootFor(dirs)
		dbFile := appdirs.DBPathFor(dirs)
		logFilePath, logFileErr := logFileResolver()
		if logFileErr != nil {
			logFilePath = filepath.Join(logDir, "app.log")
		}

		fmt.Printf("portable_mode: %t\n", dirs.Portable)
		configDir := printPath("config_dir", dirs.ConfigDir)
		printPath("config_file", dirs.ConfigFile)
		logDir = printPath("log_dir", logDir)
		printPath("log_file", logFilePath)
		outputDir := printPath("output_dir", dirs.OutputDir)
		taskRoot = printPath("task_root", taskRoot)
		cacheDir := printPath("cache_dir", dirs.CacheDir)
		printPath("db_file", dbFile)

		printWritability("config_dir", configDir)
		printWritability("log_dir", logDir)
		printWritability("output_dir", outputDir)
		printWritability("task_root", taskRoot)
		printWritability("cache_dir", cacheDir)
	}

	printDependency("ffmpeg", "ffmpeg", storage.FfmpegPath)
	printDependency("ffprobe", "ffprobe", storage.FfprobePath)
	printDependency("yt-dlp", "yt-dlp", storage.YtdlpPath)
	printDependency("edge-tts", "edge-tts", storage.EdgeTtsPath)
}

func printPath(name, value string) string {
	absPath, err := filepath.Abs(value)
	if err != nil {
		fmt.Printf("path.%s: %s (abs_error=%v)\n", name, value, err)
		return value
	}

	if _, err = os.Stat(absPath); err == nil {
		fmt.Printf("path.%s: %s (exists)\n", name, absPath)
		return absPath
	}
	if os.IsNotExist(err) {
		fmt.Printf("path.%s: %s (missing)\n", name, absPath)
		return absPath
	}

	fmt.Printf("path.%s: %s (error=%v)\n", name, absPath, err)
	return absPath
}

func effectiveDir(value, fallback string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return fallback
	}
	return cleaned
}

func printWritability(name, dir string) {
	if err := checkDirectoryWritable(dir); err != nil {
		fmt.Printf("writable.%s: error (%v)\n", name, err)
		return
	}
	fmt.Printf("writable.%s: ok\n", name)
}

func checkDirectoryWritable(dir string) error {
	if err := mkdirAllResolver(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	tempFile, err := createTempResolver(dir, ".krillin-write-check-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tempFilePath := tempFile.Name()
	closeErr := tempFile.Close()
	removeErr := removeResolver(tempFilePath)

	if closeErr != nil && removeErr != nil {
		return fmt.Errorf("close temp file: %v; cleanup temp file: %w", closeErr, removeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close temp file: %w", closeErr)
	}
	if removeErr != nil {
		return fmt.Errorf("cleanup temp file: %w", removeErr)
	}
	return nil
}

func printDependency(name, command, configuredPath string) {
	dependencyPath, source, err := resolveDependencyPath(command, configuredPath)
	if err != nil {
		fmt.Printf("dependency.%s: unknown (%v)\n", name, err)
		return
	}
	fmt.Printf("dependency.%s: %s (%s)\n", name, dependencyPath, source)
}

func resolveDependencyPath(command, configuredPath string) (string, string, error) {
	configured := strings.TrimSpace(configuredPath)
	if configured != "" {
		if resolvedPath, err := resolveConfiguredPath(configured); err == nil {
			return resolvedPath, "storage", nil
		}
		return configured, "storage", nil
	}

	resolvedPath, err := lookPathResolver(command)
	if err != nil {
		return "", "", err
	}
	return resolvedPath, "lookpath", nil
}

func resolveConfiguredPath(configuredPath string) (string, error) {
	if resolvedPath, err := lookPathResolver(configuredPath); err == nil {
		return resolvedPath, nil
	}

	absPath, err := filepath.Abs(configuredPath)
	if err != nil {
		return "", err
	}
	if _, err = os.Stat(absPath); err != nil {
		return "", err
	}
	return absPath, nil
}
