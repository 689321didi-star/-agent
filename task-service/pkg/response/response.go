package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Result{
		Code:    0,
		Message: "ok",
		Data:    data,
		TraceID: c.GetString("trace_id"),
	})
}

func Fail(c *gin.Context, httpCode int, msg string) {
	c.JSON(httpCode, Result{
		Code:    httpCode,
		Message: msg,
		TraceID: c.GetString("trace_id"),
	})
}
