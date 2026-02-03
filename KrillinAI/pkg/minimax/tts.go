package minimax

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"krillin-ai/log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// MiniMaxClient implements types.Ttser interface
type MiniMaxClient struct {
	ApiKey  string
	GroupId string
	Model   string
	BaseURL string
}

func NewMiniMaxClient(apiKey, groupId, model string) *MiniMaxClient {
	if model == "" {
		model = "speech-01-turbo"
	}
	return &MiniMaxClient{
		ApiKey:  apiKey,
		GroupId: groupId,
		Model:   model,
		BaseURL: "https://api.minimax.chat/v1/t2a_v2",
	}
}

type T2ARequest struct {
	Model        string        `json:"model"`
	Text         string        `json:"text"`
	VoiceSetting VoiceSetting  `json:"voice_setting"`
	AudioSetting AudioSetting  `json:"audio_setting"`
	Stream       bool          `json:"stream"`
}

type VoiceSetting struct {
	VoiceId string `json:"voice_id"`
}

type AudioSetting struct {
	SampleRate int    `json:"sample_rate"`
	Format     string `json:"format"`
	Channel    int    `json:"channel"`
}

type T2AResponse struct {
	BaseResp
	Data struct {
		Audio      string `json:"audio"`
		Status     int    `json:"status"` // 2 means finished
		TraceId    string `json:"trace_id"`
	} `json:"data"`
}

type BaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

func (c *MiniMaxClient) Text2Speech(text, voice, outputFile string) error {
	// 确保输出目录存在
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir failed: %w", err)
	}

	// 构造请求URL (必须包含GroupId)
	url := fmt.Sprintf("%s?GroupId=%s", c.BaseURL, c.GroupId)

	// 默认使用 male-qn-qingse 如果未指定 voice
	if voice == "" {
		voice = "male-qn-qingse"
	}

	reqBody := T2ARequest{
		Model: c.Model,
		Text:  text,
		VoiceSetting: VoiceSetting{
			VoiceId: voice,
		},
		AudioSetting: AudioSetting{
			SampleRate: 32000,
			Format:     "mp3",
			Channel:    1,
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api request failed with status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp T2AResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}

	if apiResp.StatusCode != 0 && apiResp.StatusCode != 200 { 
		return fmt.Errorf("minimax api error: %d - %s", apiResp.StatusCode, apiResp.StatusMsg)
	}

	if apiResp.Data.Audio == "" {
		return fmt.Errorf("minimax returned empty audio")
	}

	// Convert hex string to bytes
	audioBytes, err := hex.DecodeString(apiResp.Data.Audio)
	if err != nil {
		return fmt.Errorf("decode hex audio failed: %w", err)
	}

	if err := os.WriteFile(outputFile, audioBytes, 0644); err != nil {
		return fmt.Errorf("write output file failed: %w", err)
	}

	log.GetLogger().Info("MiniMax TTS success", zap.String("output", outputFile), zap.Int("bytes", len(audioBytes)))
	return nil
}
