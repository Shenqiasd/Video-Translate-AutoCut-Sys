// Package mocks provides mock implementations of core interfaces for testing.
package mocks

import (
	"krillin-ai/internal/types"

	"github.com/stretchr/testify/mock"
)

// MockTranscriber is a mock implementation of types.Transcriber
type MockTranscriber struct {
	mock.Mock
}

func (m *MockTranscriber) Transcription(audioFile, language, wordDir string) (*types.TranscriptionData, error) {
	args := m.Called(audioFile, language, wordDir)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TranscriptionData), args.Error(1)
}

// MockChatCompleter is a mock implementation of types.ChatCompleter
type MockChatCompleter struct {
	mock.Mock
}

func (m *MockChatCompleter) ChatCompletion(query string) (string, error) {
	args := m.Called(query)
	return args.String(0), args.Error(1)
}

// MockTtser is a mock implementation of types.Ttser
type MockTtser struct {
	mock.Mock
}

func (m *MockTtser) Text2Speech(text string, voice string, outputFile string) error {
	args := m.Called(text, voice, outputFile)
	return args.Error(0)
}
