package logger_test

import (
	"log/slog"

	"github.com/soundprediction/go-graphiti/pkg/logger"
)

func ExampleNewDefaultLogger() {
	// Create a logger with default settings
	log := logger.NewDefaultLogger(slog.LevelDebug)

	// Log different levels
	log.Debug("This is a debug message")
	log.Info("This is an info message")
	log.Info("Persisting nodes to database")    // Will be green in terminal
	log.Warn("This is a warning message")        // Will be yellow in terminal
	log.Error("This is an error message")        // Will be red in terminal
}

func ExampleNewLogger() {
	// Create a logger with custom configuration
	log := logger.NewDefaultLogger(slog.LevelInfo)

	// Log with attributes
	log.Info("Processing request", "user_id", "12345", "action", "create")
	log.Info("Persisting deduplicated nodes", "count", 42, "batch_size", 100) // Green
	log.Warn("Rate limit approaching", "current", 95, "limit", 100)           // Yellow
	log.Error("Database connection failed", "error", "timeout", "retry_count", 3) // Red
}
