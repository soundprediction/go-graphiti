package logger

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
)

// ColorHandler wraps an slog.Handler and adds color to messages based on level and content:
// - Error messages: Red
// - Warning messages: Yellow
// - Info messages containing "persist": Green (for database operations)
// - Other messages: Standard output
type ColorHandler struct {
	handler slog.Handler
}

// NewColorHandler creates a new colored handler that wraps the given handler
func NewColorHandler(w io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	return &ColorHandler{
		handler: slog.NewTextHandler(w, opts),
	}
}

// Enabled implements slog.Handler
func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle implements slog.Handler and adds color based on log level
func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	// Clone the record so we can modify it
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	// Add color prefix based on level and message content
	switch r.Level {
	case slog.LevelError:
		newRecord.Message = colorRed + r.Message + colorReset
	case slog.LevelWarn:
		newRecord.Message = colorYellow + r.Message + colorReset
	case slog.LevelInfo:
		// Color persist messages green
		msgLower := strings.ToLower(r.Message)
		if strings.Contains(msgLower, "persist") {
			newRecord.Message = colorGreen + r.Message + colorReset
		} else {
			newRecord.Message = r.Message
		}
	default:
		newRecord.Message = r.Message
	}

	// Copy all attributes from original record
	r.Attrs(func(a slog.Attr) bool {
		newRecord.AddAttrs(a)
		return true
	})

	return h.handler.Handle(ctx, newRecord)
}

// WithAttrs implements slog.Handler
func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ColorHandler{
		handler: h.handler.WithAttrs(attrs),
	}
}

// WithGroup implements slog.Handler
func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return &ColorHandler{
		handler: h.handler.WithGroup(name),
	}
}
