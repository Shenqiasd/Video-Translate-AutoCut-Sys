//go:build verify
// +build verify

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type SrtSentenceWithStrTime struct {
	Start string
	End   string
	Text  string
}

func main() {
	filePath := "/app/tasks/GknQjNXSG9A______Zjh9/bilingual_srt.srt"
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fmt.Printf("File size: %d bytes\n", len(data))

	// Exact regex from srt2speech.go
	re := regexp.MustCompile(`(\d{2}:\d{2}:\d{2},\d{3}) --> (\d{2}:\d{2}:\d{2},\d{3})\s+(.+?)\n`)
	matches := re.FindAllStringSubmatch(string(data), -1)

	fmt.Printf("Found %d matches\n", len(matches))

	for i, match := range matches {
		if i < 3 {
			fmt.Printf("Match %d Text: [%s]\n", i, strings.Replace(match[3], "\n", " ", -1))
		}
	}
}
