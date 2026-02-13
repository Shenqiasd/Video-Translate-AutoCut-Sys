//go:build verify
// +build verify

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

// Define structs locally to ensure exact control over JSON marshaling
type DoubaoTTSRequest struct {
	App     DoubaoApp     `json:"app"`
	User    DoubaoUser    `json:"user"`
	Audio   DoubaoAudio   `json:"audio"`
	Request DoubaoReqBody `json:"request"`
}

type DoubaoApp struct {
	AppId   string `json:"appid"`
	Token   string `json:"token"`
	Cluster string `json:"cluster"`
}

type DoubaoUser struct {
	Uid string `json:"uid"`
}

type DoubaoAudio struct {
	VoiceType  string `json:"voice_type"`
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"` // integer, not string
}

type DoubaoReqBody struct {
	ReqId     string `json:"reqid"`
	Text      string `json:"text"`
	Operation string `json:"operation"`
}

func main() {
	appId := "1871270234"
	token := "sJblaYaMh4UAu2WEhvk3lUn0Z-BiniNQ"
	cluster := "volcano_tts"
	url := "https://openspeech.bytedance.com/api/v1/tts"

	// Header formats to test
	formats := []string{
		"Bearer;" + token,  // No space
		"Bearer; " + token, // With space
		"Bearer " + token,  // Standard space
	}

	for _, authorization := range formats {
		fmt.Printf("\n----------------------------------------------------------------\n")
		fmt.Printf("Testing Authorization Header: '%s...'\n", authorization[:15]) // Truncate for security in logs

		reqId := uuid.New().String()
		reqBody := DoubaoTTSRequest{
			App: DoubaoApp{
				AppId:   appId,
				Token:   "access_token", // Fake token as per docs
				Cluster: cluster,
			},
			User: DoubaoUser{
				Uid: "test_user_id",
			},
			Audio: DoubaoAudio{
				VoiceType:  "zh_female_qingxin",
				Encoding:   "mp3",
				SampleRate: 24000,
			},
			Request: DoubaoReqBody{
				ReqId:     reqId,
				Text:      "测试豆包语音合成。",
				Operation: "query",
			},
		}

		jsonData, _ := json.Marshal(reqBody)
		fmt.Printf("Request Body: %s\n", string(jsonData))

		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", authorization)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("HTTP Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Response Status: %s\n", resp.Status)
		fmt.Printf("Response Body: %s\n", string(bodyBytes))
	}
}
