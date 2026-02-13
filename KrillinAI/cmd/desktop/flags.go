package main

import (
	"flag"
	"fmt"
	"krillin-ai/internal/appdirs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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

	dirs, err := appdirs.Resolve()
	if err != nil {
		fmt.Printf("paths: <error: %v>\n", err)
	} else {
		fmt.Printf("portable_mode: %t\n", dirs.Portable)
		printPath("config_dir", dirs.ConfigDir)
		printPath("config_file", dirs.ConfigFile)
		printPath("log_dir", dirs.LogDir)
		printPath("output_dir", dirs.OutputDir)
		printPath("cache_dir", dirs.CacheDir)
	}

	if ffmpegPath, err := exec.LookPath("ffmpeg"); err == nil {
		fmt.Printf("dependency.ffmpeg: found (%s)\n", ffmpegPath)
	} else {
		fmt.Printf("dependency.ffmpeg: missing (%v)\n", err)
	}
}

func printPath(name, value string) {
	absPath, err := filepath.Abs(value)
	if err != nil {
		fmt.Printf("path.%s: %s (abs_error=%v)\n", name, value, err)
		return
	}

	if _, err = os.Stat(absPath); err == nil {
		fmt.Printf("path.%s: %s (exists)\n", name, absPath)
		return
	}
	if os.IsNotExist(err) {
		fmt.Printf("path.%s: %s (missing)\n", name, absPath)
		return
	}

	fmt.Printf("path.%s: %s (error=%v)\n", name, absPath, err)
}
