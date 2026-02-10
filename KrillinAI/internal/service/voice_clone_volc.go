package service

import (
	"context"
	"fmt"
	"krillin-ai/config"
	"krillin-ai/log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// prepareVolcVoiceClone trains/updates a Volcengine speaker_id (S_*) using a local reference audio.
// It returns the trained speaker_id.
//
// Design choices:
//   - We require the user-provided voice code to be a speaker_id (S_*) when voice cloning is requested.
//     This avoids inventing speaker IDs and matches the doc note: speaker_id should come from console.
//   - model_type default comes from config (user selected 4 = ICL2.0).
func (s Service) prepareVolcVoiceClone(ctx context.Context, localAudioPath string, speakerID string) (string, error) {
	if s.VoiceCloneClient == nil {
		return "", fmt.Errorf("voice clone client not configured")
	}
	if speakerID == "" {
		return "", fmt.Errorf("voice clone requires speaker_id")
	}
	if !strings.HasPrefix(speakerID, "S_") {
		return "", fmt.Errorf("voice clone speaker_id must start with S_ (got %q)", speakerID)
	}

	b, err := os.ReadFile(localAudioPath)
	if err != nil {
		return "", fmt.Errorf("read voice clone audio failed: %w", err)
	}

	audioFormat := strings.TrimPrefix(strings.ToLower(filepath.Ext(localAudioPath)), ".")
	if audioFormat == "" {
		audioFormat = "wav"
	}

	language := config.Conf.Tts.VoiceCloneVolc.Language
	modelType := config.Conf.Tts.VoiceCloneVolc.ModelType
	// Extra params are optional; keep empty by default to minimize surprises.
	extraParams := ""

	_, err = s.VoiceCloneClient.UploadAndTrain(speakerID, b, audioFormat, language, modelType, extraParams)
	if err != nil {
		return "", err
	}

	// Poll status until Success/Active or timeout.
	deadline := time.Now().Add(3 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		st, err := s.VoiceCloneClient.Status(speakerID)
		if err != nil {
			return "", err
		}

		// status: enum { NotFound = 0 Training = 1 Success = 2 Failed = 3 Active = 4 }
		switch st.Status {
		case 2, 4:
			log.GetLogger().Info("Volc voice clone ready", zap.String("speaker_id", speakerID), zap.Int("status", st.Status), zap.String("version", st.Version))
			return speakerID, nil
		case 3:
			return "", fmt.Errorf("voice clone training failed for speaker_id=%s", speakerID)
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("voice clone training timeout for speaker_id=%s", speakerID)
		}

		time.Sleep(2 * time.Second)
	}
}
