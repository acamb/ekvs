package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Logger is the interface used throughout the application for structured logging.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type slogLogger struct {
	l *slog.Logger
}

func (s *slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogLogger) Error(msg string, args ...any) { s.l.Error(msg, args...) }
func (s *slogLogger) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }

// New creates a Logger backed by log/slog writing to stdout.
// Accepted levels: "debug", "info", "warn", "error" (case-insensitive).
// Defaults to "info" for any unrecognised value.
func New(level string) Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return &slogLogger{l: slog.New(h)}
}
