package doubao

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"krillin-ai/log"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// VoiceCloneClient implements Volcengine MegaTTS voice clone (ICL/DiT) APIs.
// Doc excerpt provided by user (upload + status):
// - POST https://openspeech.bytedance.com/api/v1/mega_tts/audio/upload
// - POST https://openspeech.bytedance.com/api/v1/mega_tts/status
// Auth: Header Authorization: "Bearer;${token}" (note semicolon)
// Resource selector: Header Resource-Id: seed-icl-1.0 | seed-icl-2.0
// After training, use TTS with cluster volcano_icl(/_concurr) and voice_type = speaker_id (S_*).
//
// NOTE: We require a speaker_id (S_*) from console or caller; upload trains/updates that speaker_id.
// This matches current product behavior described in docs.

type VoiceCloneClient struct {
	AppId       string
	AccessToken string
	ResourceId  string
	BaseURL     string
}

func NewVoiceCloneClient(appId, accessToken, resourceId string) *VoiceCloneClient {
	if resourceId == "" {
		resourceId = "seed-icl-2.0"
	}
	return &VoiceCloneClient{
		AppId:       appId,
		AccessToken: accessToken,
		ResourceId:  resourceId,
		BaseURL:     "https://openspeech.bytedance.com",
	}
}

type voiceCloneUploadReq struct {
	AppID       string            `json:"appid"`
	SpeakerID   string            `json:"speaker_id"`
	Audios      []voiceCloneAudio `json:"audios"`
	Source      int               `json:"source"`
	Language    int               `json:"language,omitempty"`
	ModelType   int               `json:"model_type,omitempty"`
	ExtraParams string            `json:"extra_params,omitempty"` // json string
}

type voiceCloneAudio struct {
	AudioBytes  string `json:"audio_bytes"`
	AudioFormat string `json:"audio_format,omitempty"`
	Text        string `json:"text,omitempty"`
}

type voiceCloneBaseResp struct {
	StatusCode    int    `json:"StatusCode"`
	StatusMessage string `json:"StatusMessage"`
}

type VoiceCloneUploadResp struct {
	BaseResp  voiceCloneBaseResp `json:"BaseResp"`
	SpeakerID string             `json:"speaker_id"`
}

type VoiceCloneStatusResp struct {
	BaseResp   voiceCloneBaseResp `json:"BaseResp"`
	SpeakerID  string             `json:"speaker_id"`
	Status     int                `json:"status"`
	CreateTime int64              `json:"create_time"`
	Version    string             `json:"version"`
	DemoAudio  string             `json:"demo_audio"`
}

// UploadAndTrain uploads one reference audio and starts training for the given speaker_id.
// audioBytes is raw bytes; we base64-encode for API.
func (c *VoiceCloneClient) UploadAndTrain(speakerID string, audioBytes []byte, audioFormat string, language int, modelType int, extraParamsJSON string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("voice clone client is nil")
	}
	if c.AppId == "" || c.AccessToken == "" {
		return "", fmt.Errorf("voice clone requires app_id and access_token")
	}
	if speakerID == "" {
		return "", fmt.Errorf("voice clone requires speaker_id (S_*)")
	}

	reqBody := voiceCloneUploadReq{
		AppID:     c.AppId,
		SpeakerID: speakerID,
		Audios: []voiceCloneAudio{
			{
				AudioBytes:  base64.StdEncoding.EncodeToString(audioBytes),
				AudioFormat: audioFormat,
			},
		},
		Source:    2,
		Language:  language,
		ModelType: modelType,
	}
	if extraParamsJSON != "" {
		reqBody.ExtraParams = extraParamsJSON
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal voice clone upload req failed: %w", err)
	}

	url := c.BaseURL + "/api/v1/mega_tts/audio/upload"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("create voice clone upload req failed: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("voice clone upload http failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("voice clone upload http status=%d body=%s", resp.StatusCode, string(body))
	}

	var out VoiceCloneUploadResp
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("voice clone upload decode failed: %w body=%s", err, string(body))
	}
	if out.BaseResp.StatusCode != 0 {
		return "", fmt.Errorf("voice clone upload failed code=%d msg=%s", out.BaseResp.StatusCode, out.BaseResp.StatusMessage)
	}

	log.GetLogger().Info("Volc voice clone upload ok", zap.String("speaker_id", out.SpeakerID), zap.String("resource_id", c.ResourceId), zap.Int("model_type", modelType))
	return out.SpeakerID, nil
}

func (c *VoiceCloneClient) Status(speakerID string) (*VoiceCloneStatusResp, error) {
	if c == nil {
		return nil, fmt.Errorf("voice clone client is nil")
	}
	if c.AppId == "" || c.AccessToken == "" {
		return nil, fmt.Errorf("voice clone requires app_id and access_token")
	}
	if speakerID == "" {
		return nil, fmt.Errorf("voice clone status requires speaker_id")
	}

	payload, err := json.Marshal(map[string]string{
		"appid":      c.AppId,
		"speaker_id": speakerID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal voice clone status req failed: %w", err)
	}

	url := c.BaseURL + "/api/v1/mega_tts/status"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("create voice clone status req failed: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voice clone status http failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("voice clone status http status=%d body=%s", resp.StatusCode, string(body))
	}

	var out VoiceCloneStatusResp
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("voice clone status decode failed: %w body=%s", err, string(body))
	}
	if out.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("voice clone status failed code=%d msg=%s", out.BaseResp.StatusCode, out.BaseResp.StatusMessage)
	}
	return &out, nil
}

func (c *VoiceCloneClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer;"+c.AccessToken)
	req.Header.Set("Resource-Id", c.ResourceId)
}
