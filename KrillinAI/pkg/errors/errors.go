// Package errors provides structured error handling for the application.
// It defines AppError type with error codes for consistent API responses.
package errors

import (
	"errors"
	"fmt"
)

// Error codes organized by category
const (
	// General errors (1000-1099)
	CodeSuccess       = 0
	CodeUnknown       = 1000
	CodeInvalidParams = 1001
	CodeNotFound      = 1002
	CodeUnauthorized  = 1003

	// Video/Audio processing errors (1100-1199)
	CodeVideoDownload    = 1100
	CodeAudioExtract     = 1101
	CodeVideoNotFound    = 1102
	CodeUnsupportedURL   = 1103
	CodeCookiesExpired   = 1104
	CodeRateLimited      = 1105

	// Transcription errors (1200-1299)
	CodeTranscribeFailed  = 1200
	CodeTranscribeTimeout = 1201
	CodeModelNotFound     = 1202

	// Translation errors (1300-1399)
	CodeTranslateFailed  = 1300
	CodeTranslateTimeout = 1301
	CodeLLMQuotaExceeded = 1302

	// TTS errors (1400-1499)
	CodeTTSFailed       = 1400
	CodeTTSQuotaExceeded = 1401
	CodeVoiceNotFound   = 1402
	CodeAudioMixFailed  = 1403

	// Storage errors (1500-1599)
	CodeDBError        = 1500
	CodeFileNotFound   = 1501
	CodeFileWriteError = 1502

	// Smart Clipper errors (1600-1699)
	CodeClipAnalysisFailed = 1600
	CodeClipSplitFailed    = 1601
	CodeSubtitleNotFound   = 1602
)

// AppError represents a structured application error
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
	Cause   error  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New creates a new AppError
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with an AppError
func Wrap(code int, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WrapWithDetail wraps an error with additional detail
func WrapWithDetail(code int, message string, detail string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Detail:  detail,
		Cause:   cause,
	}
}

// Is checks if the target error is an AppError with the specified code
func Is(err error, code int) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// GetCode extracts error code from error, returns CodeUnknown if not AppError
func GetCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeUnknown
}

// GetMessage extracts message from error
func GetMessage(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}
	return err.Error()
}

// Predefined common errors
var (
	ErrInvalidParams = New(CodeInvalidParams, "参数错误 Invalid parameters")
	ErrNotFound      = New(CodeNotFound, "资源不存在 Resource not found")
	ErrUnauthorized  = New(CodeUnauthorized, "未授权 Unauthorized")

	// Video/Audio
	ErrVideoDownload  = New(CodeVideoDownload, "视频下载失败 Video download failed")
	ErrAudioExtract   = New(CodeAudioExtract, "音频提取失败 Audio extraction failed")
	ErrCookiesExpired = New(CodeCookiesExpired, "Cookies已过期 Cookies expired")
	ErrRateLimited    = New(CodeRateLimited, "请求频率限制 Rate limited")

	// Transcription
	ErrTranscribeFailed  = New(CodeTranscribeFailed, "语音识别失败 Transcription failed")
	ErrTranscribeTimeout = New(CodeTranscribeTimeout, "语音识别超时 Transcription timeout")

	// Translation
	ErrTranslateFailed   = New(CodeTranslateFailed, "翻译失败 Translation failed")
	ErrLLMQuotaExceeded  = New(CodeLLMQuotaExceeded, "LLM配额耗尽 LLM quota exceeded")

	// TTS
	ErrTTSFailed        = New(CodeTTSFailed, "语音合成失败 TTS failed")
	ErrTTSQuotaExceeded = New(CodeTTSQuotaExceeded, "TTS配额耗尽 TTS quota exceeded")
	ErrVoiceNotFound    = New(CodeVoiceNotFound, "音色不存在 Voice not found")

	// Storage
	ErrDBError       = New(CodeDBError, "数据库错误 Database error")
	ErrFileNotFound  = New(CodeFileNotFound, "文件不存在 File not found")

	// Smart Clipper
	ErrClipAnalysisFailed = New(CodeClipAnalysisFailed, "智能切片分析失败 Clip analysis failed")
	ErrSubtitleNotFound   = New(CodeSubtitleNotFound, "未找到字幕 Subtitle not found")
)
