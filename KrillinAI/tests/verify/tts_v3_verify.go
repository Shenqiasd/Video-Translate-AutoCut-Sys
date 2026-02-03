package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// V3 Request Structure (from official docs)
type V3TTSRequest struct {
	User      V3User      `json:"user"`
	ReqParams V3ReqParams `json:"req_params"`
}

type V3User struct {
	Uid string `json:"uid"`
}

type V3ReqParams struct {
	Text        string       `json:"text"`
	Speaker     string       `json:"speaker"`
	AudioParams V3AudioParams `json:"audio_params"`
}

type V3AudioParams struct {
	Format     string `json:"format"`
	SampleRate int    `json:"sample_rate"`
}

func main() {
	// NEW Credentials from user
	appId := "6940236983"
	accessToken := "yN_0wdVIr_nsn1hdzBZZJ8KSH_8jgjt5"
	
	// Unidirectional HTTP endpoint (simpler than WebSocket for testing)
	url := "https://openspeech.bytedance.com/api/v3/tts/unidirectional"

	// Resource IDs to test (Character version 1.0)
	resourceIds := []string{
		"seed-tts-1.0",             // Character version 1.0 (alias)
		"volc.service_type.10029",  // Character version 1.0 (numeric)
	}

	// Voice from user's list
	voice := "zh_female_wanqudashu_moon_bigtts"

	for _, resId := range resourceIds {
		fmt.Printf("\n=== Testing Resource ID: %s ===\n", resId)
		
		reqBody := V3TTSRequest{
			User: V3User{
				Uid: "test_user_" + uuid.New().String()[:8],
			},
			ReqParams: V3ReqParams{
				Text:    "你好，这是测试语音。",
				Speaker: voice,
				AudioParams: V3AudioParams{
					Format:     "mp3",
					SampleRate: 24000,
				},
			},
		}

		jsonData, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		
		// CORRECT Headers per official docs:
		// X-Api-App-Key = APP ID (NOT X-Api-App-Id!)
		// X-Api-Access-Key = Access Token
		// X-Api-Resource-Id = Resource ID
		req.Header.Set("X-Api-App-Key", appId)
		req.Header.Set("X-Api-Access-Key", accessToken)
		req.Header.Set("X-Api-Resource-Id", resId)
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("HTTP Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Status: %s\n", resp.Status)
		
		if len(bodyBytes) > 500 {
			// If large response, likely audio data - SUCCESS!
			fmt.Printf("Response Size: %d bytes (likely audio data - SUCCESS!)\n", len(bodyBytes))
			fmt.Printf("First 200 bytes: %s\n", string(bodyBytes[:200]))
		} else {
			fmt.Printf("Response: %s\n", string(bodyBytes))
		}
	}
}
