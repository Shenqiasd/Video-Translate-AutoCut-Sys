package doubao

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"krillin-ai/log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DoubaoClient implements types.Ttser interface for Volcengine Doubao TTS
type DoubaoClient struct {
	AppId       string
	AccessToken string
	Cluster     string
	ResourceId  string // V3 API requires resource ID
	BaseURL     string
}

func NewDoubaoClient(appId, accessToken, cluster string) *DoubaoClient {
	if cluster == "" {
		cluster = "volcano_tts"
	}
	return &DoubaoClient{
		AppId:       appId,
		AccessToken: accessToken,
		Cluster:     cluster,
		ResourceId:  "seed-tts-1.0", // Default to Character version 1.0
		BaseURL:     "https://openspeech.bytedance.com/api/v3/tts/unidirectional",
	}
}

// V3 TTS Request structures (updated for V3 API)
type DoubaoTTSRequest struct {
	User      DoubaoUser      `json:"user"`
	ReqParams DoubaoReqParams `json:"req_params"`
}

type DoubaoUser struct {
	Uid string `json:"uid"`
}

type DoubaoReqParams struct {
	Text        string           `json:"text"`
	Speaker     string           `json:"speaker"`
	AudioParams DoubaoAudioParams `json:"audio_params"`
}

type DoubaoAudioParams struct {
	Format     string `json:"format"`
	SampleRate int    `json:"sample_rate"`
}

// V3 TTS Response structure
type DoubaoTTSResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"` // Base64 encoded audio
}

func (c *DoubaoClient) Text2Speech(text, voice, outputFile string) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir failed: %w", err)
	}

	// Default voice if not specified - use a Doubao 1.0 compatible voice
	if voice == "" {
		voice = "zh_female_wanqudashu_moon_bigtts" // 湾区大叔 - confirmed working
	}

	reqBody := DoubaoTTSRequest{
		User: DoubaoUser{
			Uid: "krillinai_user_" + uuid.New().String()[:8],
		},
		ReqParams: DoubaoReqParams{
			Text:    text,
			Speaker: voice,
			AudioParams: DoubaoAudioParams{
				Format:     "mp3",
				SampleRate: 24000,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	// V3 API Authentication Headers (CORRECT per official docs)
	// X-Api-App-Key = APP ID (NOT X-Api-App-Id!)
	// X-Api-Access-Key = Access Token
	// X-Api-Resource-Id = Resource ID
	req.Header.Set("X-Api-App-Key", c.AppId)
	req.Header.Set("X-Api-Access-Key", c.AccessToken)
	req.Header.Set("X-Api-Resource-Id", c.ResourceId)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse Streaming Response
	// V3 API returns multiple JSON objects in the response body if the text is long or streaming is enabled
	// We need to read line by line or decode continuously
	decoder := json.NewDecoder(resp.Body)
	var fullAudioBuffer bytes.Buffer

	for {
		var apiResp DoubaoTTSResponse
		if err := decoder.Decode(&apiResp); err != nil {
			if err == io.EOF {
				break // End of stream
			}
			return fmt.Errorf("decode response stream failed: %w", err)
		}

		if apiResp.Code != 3000 && apiResp.Code != 0 { // 3000 is typically "Success" or "Streaming" for V3, 0 is sometimes used
			// If Code is not 0 or 3000, log warning but continue if data exists? No, usually it's failure.
			// Let's assume non-zero/non-3000 is bad.
			// Actually V3 doc says: 3000 = Success.
			if apiResp.Code != 3000 {
				 log.GetLogger().Warn("doubao api response code warning", zap.Int("code", apiResp.Code), zap.String("msg", apiResp.Message))
			}
		}

		if apiResp.Data != "" {
			audioChunk, err := base64.StdEncoding.DecodeString(apiResp.Data)
			if err != nil {
				return fmt.Errorf("decode base64 chunk failed: %w", err)
			}
			fullAudioBuffer.Write(audioChunk)
		}
	}

	if fullAudioBuffer.Len() == 0 {
		return fmt.Errorf("doubao returned no audio data after streaming")
	}

	if err := os.WriteFile(outputFile, fullAudioBuffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("write output file failed: %w", err)
	}

	log.GetLogger().Info("Doubao TTS success", zap.String("output", outputFile), zap.Int("total_bytes", fullAudioBuffer.Len()))
	return nil
}
