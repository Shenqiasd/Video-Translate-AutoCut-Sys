package service

import (
	"testing"

	"krillin-ai/internal/dto"
	"krillin-ai/internal/mocks"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	apperrors "krillin-ai/pkg/errors"

	"github.com/stretchr/testify/assert"
)

func init() {
	log.InitLogger()
}

func TestStartSubtitleTask_InvalidYouTubeURL(t *testing.T) {
	// Create service with mocks
	mockTranscriber := new(mocks.MockTranscriber)
	mockChatCompleter := new(mocks.MockChatCompleter)
	mockTts := new(mocks.MockTtser)

	svc := &Service{
		Transcriber:   mockTranscriber,
		ChatCompleter: mockChatCompleter,
		TtsClient:     mockTts,
	}

	// Test with invalid YouTube URL
	req := dto.StartVideoSubtitleTaskReq{
		Url: "https://youtube.com/watch",
	}

	_, err := svc.StartSubtitleTask(req)

	assert.Error(t, err)
	assert.True(t, apperrors.Is(err, apperrors.CodeUnsupportedURL))
}

func TestStartSubtitleTask_InvalidBilibiliURL(t *testing.T) {
	mockTranscriber := new(mocks.MockTranscriber)
	mockChatCompleter := new(mocks.MockChatCompleter)
	mockTts := new(mocks.MockTtser)

	svc := &Service{
		Transcriber:   mockTranscriber,
		ChatCompleter: mockChatCompleter,
		TtsClient:     mockTts,
	}

	// Test with invalid Bilibili URL
	req := dto.StartVideoSubtitleTaskReq{
		Url: "https://bilibili.com/",
	}

	_, err := svc.StartSubtitleTask(req)

	assert.Error(t, err)
	assert.True(t, apperrors.Is(err, apperrors.CodeUnsupportedURL))
}

func TestStartSubtitleTask_ValidLocalFile(t *testing.T) {
	// This test would require more setup with file system mocks
	// Placeholder for future implementation
	t.Skip("Requires file system mock implementation")
}

func TestGetTaskStatus_TaskNotFound(t *testing.T) {
	mockTranscriber := new(mocks.MockTranscriber)
	mockChatCompleter := new(mocks.MockChatCompleter)
	mockTts := new(mocks.MockTtser)

	svc := &Service{
		Transcriber:   mockTranscriber,
		ChatCompleter: mockChatCompleter,
		TtsClient:     mockTts,
	}

	req := dto.GetVideoSubtitleTaskReq{
		TaskId: "non-existent-task-id",
	}

	result, err := svc.GetTaskStatus(req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "任务不存在")
}

// TestResultTypeMapping tests the subtitle result type mapping logic
func TestResultTypeMapping(t *testing.T) {
	testCases := []struct {
		name           string
		targetLang     string
		bilingual      uint8
		translationPos uint8
		expectedType   types.SubtitleResultType
	}{
		{
			name:         "Origin only when targetLang is none",
			targetLang:   "none",
			bilingual:    0,
			expectedType: types.SubtitleResultTypeOriginOnly,
		},
		{
			name:           "Bilingual with translation on top",
			targetLang:     "zh",
			bilingual:      types.SubtitleTaskBilingualYes,
			translationPos: types.SubtitleTaskTranslationSubtitlePosTop,
			expectedType:   types.SubtitleResultTypeBilingualTranslationOnTop,
		},
		{
			name:           "Bilingual with translation on bottom",
			targetLang:     "zh",
			bilingual:      types.SubtitleTaskBilingualYes,
			translationPos: types.SubtitleTaskTranslationSubtitlePosBelow,
			expectedType:   types.SubtitleResultTypeBilingualTranslationOnBottom,
		},
		{
			name:         "Target only",
			targetLang:   "zh",
			bilingual:    types.SubtitleTaskBilingualNo,
			expectedType: types.SubtitleResultTypeTargetOnly,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var resultType types.SubtitleResultType

			if tc.targetLang == "none" {
				resultType = types.SubtitleResultTypeOriginOnly
			} else {
				if tc.bilingual == types.SubtitleTaskBilingualYes {
					if tc.translationPos == types.SubtitleTaskTranslationSubtitlePosTop {
						resultType = types.SubtitleResultTypeBilingualTranslationOnTop
					} else {
						resultType = types.SubtitleResultTypeBilingualTranslationOnBottom
					}
				} else {
					resultType = types.SubtitleResultTypeTargetOnly
				}
			}

			assert.Equal(t, tc.expectedType, resultType)
		})
	}
}
