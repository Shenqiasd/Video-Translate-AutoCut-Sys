//go:build !desktop && !windows
// +build !desktop,!windows

package main

import "os"

func main() {
	if handled, exitCode := handleCLIFlags(); handled {
		os.Exit(exitCode)
	}
}
