package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Error(t *testing.T) {
	// Test without cause
	err := New(CodeVideoDownload, "Test error")
	assert.Equal(t, "[1100] Test error", err.Error())

	// Test with cause
	cause := errors.New("underlying error")
	errWithCause := Wrap(CodeVideoDownload, "Test error", cause)
	assert.Contains(t, errWithCause.Error(), "underlying error")
	assert.Contains(t, errWithCause.Error(), "1100")
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(CodeTranscribeFailed, "Transcription failed", cause)

	// Test Unwrap
	assert.Equal(t, cause, err.Unwrap())

	// Test errors.Is
	assert.True(t, errors.Is(err, cause))
}

func TestIs(t *testing.T) {
	err := New(CodeTTSFailed, "TTS failed")

	assert.True(t, Is(err, CodeTTSFailed))
	assert.False(t, Is(err, CodeVideoDownload))

	// Test with regular error
	regularErr := errors.New("regular error")
	assert.False(t, Is(regularErr, CodeTTSFailed))
}

func TestGetCode(t *testing.T) {
	// AppError
	appErr := New(CodeLLMQuotaExceeded, "Quota exceeded")
	assert.Equal(t, CodeLLMQuotaExceeded, GetCode(appErr))

	// Regular error returns CodeUnknown
	regularErr := errors.New("regular error")
	assert.Equal(t, CodeUnknown, GetCode(regularErr))
}

func TestGetMessage(t *testing.T) {
	// AppError
	appErr := New(CodeFileNotFound, "文件不存在 File not found")
	assert.Equal(t, "文件不存在 File not found", GetMessage(appErr))

	// Regular error returns error message
	regularErr := errors.New("regular error message")
	assert.Equal(t, "regular error message", GetMessage(regularErr))
}

func TestWrapWithDetail(t *testing.T) {
	cause := errors.New("connection refused")
	err := WrapWithDetail(CodeVideoDownload, "Download failed", "URL: https://example.com", cause)

	assert.Equal(t, CodeVideoDownload, err.Code)
	assert.Equal(t, "Download failed", err.Message)
	assert.Equal(t, "URL: https://example.com", err.Detail)
	assert.Equal(t, cause, err.Cause)
}

func TestPredefinedErrors(t *testing.T) {
	// Verify predefined errors have correct codes
	assert.Equal(t, CodeInvalidParams, ErrInvalidParams.Code)
	assert.Equal(t, CodeVideoDownload, ErrVideoDownload.Code)
	assert.Equal(t, CodeTranscribeFailed, ErrTranscribeFailed.Code)
	assert.Equal(t, CodeTTSFailed, ErrTTSFailed.Code)
	assert.Equal(t, CodeDBError, ErrDBError.Code)
}
