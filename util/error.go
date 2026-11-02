package util

import (
	"fmt"
	"io"
	"log/slog"
)

func PanicOnError(e error, message string) {
	if e != nil {
		slog.Error(message)
		panic(e)
	}
}

func PanicOnIoError(e error, message string) {
	if e != nil && e != io.EOF {
		slog.Error(message)
		fmt.Printf("Check/Panic: %s\n", e)
		panic(e)
	}
}

func LogError(e error) {
	if e != nil {
		slog.Error("Error", "cause", e)
	}
}
