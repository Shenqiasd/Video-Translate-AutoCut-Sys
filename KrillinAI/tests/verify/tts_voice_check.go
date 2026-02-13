//go:build verify
// +build verify

package main

import (
	"context"
	"fmt"
	"krillin-ai/config"
	"krillin-ai/pkg/doubao"
	"log"
	"os"
)

func main() {
	// Load config manually or hardcode for test
	appId := "6940236983"
	accessToken := "yN_0wdVIr_nsn1hdzBZZJ8KSH_8jgjt5"
	cluster := "volcano_tts"

	client := doubao.NewDoubaoClient(appId, accessToken, cluster)

	voices := []string{
		"zh_female_qingxin_mars_bigtts",
		"zh_male_chunhou_mars_bigtts",
		"zh_female_sichuan_mars_bigtts",
		"zh_male_xiaoming_mars_bigtts",
		"zh_female_tianmei_mars_bigtts",
		"zh_female_cancan_mars_bigtts", // known good
	}

	for _, v := range voices {
		fmt.Printf("Testing voice: %s ... ", v)
		err := client.Text2Speech("你好，我是测试语音。", v, fmt.Sprintf("test_%s.mp3", v))
		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
		} else {
			fmt.Printf("SUCCESS\n")
			os.Remove(fmt.Sprintf("test_%s.mp3", v))
		}
	}
}
