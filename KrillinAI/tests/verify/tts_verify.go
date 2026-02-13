//go:build verify
// +build verify

package main

import (
	"fmt"
	"krillin-ai/pkg/doubao"
	"os"
)

func main() {
	// Credentials from config.toml
	appId := "1871270234"
	token := "sJblaYaMh4UAu2WEhvk3lUn0Z-BiniNQ"

	clusters := []string{"volcano_tts", "volcengine_tts"}

	for _, cluster := range clusters {
		fmt.Printf("\n--- Testing Cluster: %s ---\n", cluster)

		fmt.Println("Initializing Doubao Client...")
		client := doubao.NewDoubaoClient(appId, token, cluster)

		outputFile := fmt.Sprintf("doubao_test_audio_%s.mp3", cluster)
		text := "你好，这是一个豆包语音合成的测试音频。"
		voice := "zh_female_qingxin"

		fmt.Printf("Generating TTS for text: %s\n", text)
		err := client.Text2Speech(text, voice, outputFile)
		if err != nil {
			fmt.Printf("TTS failed for cluster %s: %v\n", cluster, err)
		} else {
			fmt.Printf("TTS success for cluster %s! Audio saved to %s\n", cluster, outputFile)
			os.Exit(0) // Success
		}
	}
	fmt.Println("\nAll clusters failed.")
	os.Exit(1)
}
