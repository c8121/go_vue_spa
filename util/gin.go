package util

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Status  string
	Message string
}

func AboutWithStatus(c *gin.Context, code int, message string, err error) {
	if err != nil {
		slog.Warn("Abort request", "code", code, "message", message, "error", err)
	} else {
		slog.Warn("Abort request", "code", code, "message", message)
	}

	c.AbortWithStatusJSON(code, &Response{
		Status:  "error",
		Message: message,
	})
}
