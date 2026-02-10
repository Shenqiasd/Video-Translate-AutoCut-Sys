// Package mocks provides mock implementations of core interfaces for testing.
package mocks

import (
	"context"
	"krillin-ai/internal/types"

	"github.com/stretchr/testify/mock"
)

// MockTranscriber is a mock implementation of types.Transcriber
type MockTranscriber struct {
	mock.Mock
}

func (m *MockTranscriber) Transcribe(ctx context.Context, audioFile string, language types.StandardLanguageCode) ([]types.TranscribedAudioInfo, error) {
	args := m.Called(ctx, audioFile, language)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.TranscribedAudioInfo), args.Error(1)
}

// MockChatCompleter is a mock implementation of types.ChatCompleter
type MockChatCompleter struct {
	mock.Mock
}

func (m *MockChatCompleter) ChatCompletion(systemPrompt, userPrompt string) (string, error) {
	args := m.Called(systemPrompt, userPrompt)
	return args.String(0), args.Error(1)
}

func (m *MockChatCompleter) ChatCompletionWithHistory(systemPrompt string, history []types.Message, userPrompt string) (string, error) {
	args := m.Called(systemPrompt, history, userPrompt)
	return args.String(0), args.Error(1)
}

// MockTtser is a mock implementation of types.Ttser
type MockTtser struct {
	mock.Mock
}

func (m *MockTtser) GetAudio(text string, voice string, outputPath string) error {
	args := m.Called(text, voice, outputPath)
	return args.Error(0)
}
