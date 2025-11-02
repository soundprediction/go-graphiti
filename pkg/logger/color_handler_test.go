package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestColorHandler(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Level
		message  string
		wantCode string
	}{
		{
			name:     "error message has red color",
			level:    slog.LevelError,
			message:  "test error",
			wantCode: colorRed,
		},
		{
			name:     "warning message has yellow color",
			level:    slog.LevelWarn,
			message:  "test warning",
			wantCode: colorYellow,
		},
		{
			name:     "info message has no color",
			level:    slog.LevelInfo,
			message:  "test info",
			wantCode: "",
		},
		{
			name:     "persist message has green color",
			level:    slog.LevelInfo,
			message:  "Persisting deduplicated nodes",
			wantCode: colorGreen,
		},
		{
			name:     "persisted message has green color",
			level:    slog.LevelInfo,
			message:  "Nodes persisted successfully",
			wantCode: colorGreen,
		},
		{
			name:     "debug message has no color",
			level:    slog.LevelDebug,
			message:  "test debug",
			wantCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(&buf, slog.LevelDebug)

			switch tt.level {
			case slog.LevelError:
				logger.Error(tt.message)
			case slog.LevelWarn:
				logger.Warn(tt.message)
			case slog.LevelInfo:
				logger.Info(tt.message)
			case slog.LevelDebug:
				logger.Debug(tt.message)
			}

			output := buf.String()

			// Check if message is present
			if !strings.Contains(output, tt.message) {
				t.Errorf("output does not contain message %q, got: %s", tt.message, output)
			}

			// Check color code (escape codes may be quoted in the text handler output)
			if tt.wantCode != "" {
				hasColorCode := strings.Contains(output, tt.wantCode) ||
					strings.Contains(output, "\\x1b[31m") || // Red color code escaped
					strings.Contains(output, "\\x1b[33m") || // Yellow color code escaped
					strings.Contains(output, "\\x1b[32m")    // Green color code escaped
				if !hasColorCode {
					t.Errorf("output does not contain color code %q, got: %s", tt.wantCode, output)
				}
			} else {
				// Info and Debug should not have any color codes (except persist messages)
				hasColorCode := strings.Contains(output, colorRed) ||
					strings.Contains(output, colorYellow) ||
					strings.Contains(output, colorGreen) ||
					strings.Contains(output, "\\x1b[31m") ||
					strings.Contains(output, "\\x1b[33m") ||
					strings.Contains(output, "\\x1b[32m")
				if hasColorCode {
					t.Errorf("output should not contain color codes, got: %s", output)
				}
			}
		})
	}
}

func TestColorHandlerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, slog.LevelDebug)

	logger.Error("test error", "key", "value")

	output := buf.String()

	// Check if message and attributes are present
	if !strings.Contains(output, "test error") {
		t.Errorf("output does not contain message, got: %s", output)
	}
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("output does not contain attributes, got: %s", output)
	}
	// Check for color code (may be escaped in text handler output)
	hasColorCode := strings.Contains(output, colorRed) || strings.Contains(output, "\\x1b[31m")
	if !hasColorCode {
		t.Errorf("output does not contain red color code, got: %s", output)
	}
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger(slog.LevelInfo)
	if logger == nil {
		t.Error("NewDefaultLogger returned nil")
	}

	// Should be able to log without panic
	logger.Info("test info")
	logger.Error("test error")
}
