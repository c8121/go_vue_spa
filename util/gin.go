package util

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Message string
}

func AboutWithStatus(c *gin.Context, code int, message string, err error) {
	if err != nil {
		slog.Warn("Abort request", "code", code, "message", message, "error", err)
	} else {
		slog.Warn("Abort request", "code", code, "message", message)
	}

	c.AbortWithStatusJSON(code, &ErrorResponse{
		Message: message,
	})
}
