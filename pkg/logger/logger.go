package logger

import (
	"io"
	"log/slog"
	"os"
)

// NewDefaultLogger creates a new logger with color support for errors and warnings
// This is a convenience function for common use cases
func NewDefaultLogger(level slog.Level) *slog.Logger {
	return slog.New(NewColorHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

// NewLogger creates a new logger with color support using a custom writer
func NewLogger(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(NewColorHandler(w, &slog.HandlerOptions{
		Level: level,
	}))
}
