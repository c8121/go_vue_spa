package util

import (
	"log/slog"
	"os"
	"strings"
)

const (
	LevelNoLog = -99
)

func IsExistingDirectory(dir string, level slog.Level) bool {

	if strings.TrimSpace(dir) == "" {
		return false
	}

	stat, err := os.Stat(dir)
	if err != nil {
		if level != LevelNoLog {
			slog.Log(nil, level, "Directory does not exist", "path", dir)
		}
		return false
	}
	if !stat.IsDir() {
		if level != LevelNoLog {
			slog.Log(nil, level, "Not a directory", "path", dir)
		}
		return false
	}
	return true
}

func IsExistingFile(path string, level slog.Level) bool {

	if strings.TrimSpace(path) == "" {
		return false
	}

	stat, err := os.Stat(path)
	if err != nil {
		if level != LevelNoLog {
			slog.Log(nil, level, "File does not exist", "path", path)
		}
		return false
	}
	if !stat.Mode().IsRegular() {
		if level != LevelNoLog {
			slog.Log(nil, level, "Not a file", "path", path)
		}
		return false
	}
	return true
}
