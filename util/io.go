package util

import (
	"io"
	"log/slog"
)

// CloseOrLog closes the given Closer and logs error, if one occurs.
func CloseOrLog(c io.Closer) {
	err := c.Close()
	if err != nil {
		slog.Error("Close failed", "error", err)
	}
}
