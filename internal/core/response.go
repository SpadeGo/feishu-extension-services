package core

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const CodeSuccess = 0

// Response is the standard API response envelope.
// All JSON endpoints return { code, message, data }.
// code == 0 means success; non-zero indicates an error.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Success writes a 200 response with code=0.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// Fail writes a 200 response with an error code.
// Use for JSON API endpoints — the client checks code !== 0.
func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// FailWithStatus writes an error response with a custom HTTP status.
// Use for endpoints where success isn't JSON (e.g. binary download proxy),
// so the client can distinguish success vs error by response.ok.
func FailWithStatus(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}
