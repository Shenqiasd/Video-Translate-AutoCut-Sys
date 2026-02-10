package response

import (
	"github.com/gin-gonic/gin"
	apperrors "krillin-ai/pkg/errors"
)

// Response is the standard API response structure
type Response struct {
	Error  int32  `json:"error"`  // Error code (0 = success)
	Msg    string `json:"msg"`    // Human-readable message
	Detail string `json:"detail,omitempty"` // Additional error details
	Data   any    `json:"data"`   // Response payload
}

// R sends a JSON response
func R(c *gin.Context, data any) {
	c.JSON(200, data)
}

// Success returns a success response with data
func Success(c *gin.Context, data any) {
	c.JSON(200, Response{
		Error: 0,
		Msg:   "成功 Success",
		Data:  data,
	})
}

// Error returns an error response with code and message
func Error(c *gin.Context, code int, msg string) {
	c.JSON(200, Response{
		Error: int32(code),
		Msg:   msg,
		Data:  nil,
	})
}

// FromError converts an error to a Response
// If the error is an AppError, it extracts code and message
// Otherwise, it uses CodeUnknown
func FromError(err error) Response {
	if err == nil {
		return Response{
			Error: 0,
			Msg:   "成功 Success",
		}
	}

	code := apperrors.GetCode(err)
	msg := apperrors.GetMessage(err)

	var detail string
	if appErr, ok := err.(*apperrors.AppError); ok {
		detail = appErr.Detail
	}

	return Response{
		Error:  int32(code),
		Msg:    msg,
		Detail: detail,
		Data:   nil,
	}
}

// ErrorResponse sends an error response from an error
func ErrorResponse(c *gin.Context, err error) {
	c.JSON(200, FromError(err))
}

