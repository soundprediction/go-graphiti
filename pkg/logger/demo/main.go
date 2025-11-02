package main

import (
	"log/slog"

	"github.com/soundprediction/go-graphiti/pkg/logger"
)

func main() {
	// Create a colored logger
	log := logger.NewDefaultLogger(slog.LevelDebug)

	log.Info("============================================")
	log.Info("    Graphiti Colored Logger Demo")
	log.Info("============================================")
	log.Info("")

	log.Debug("Debug message - standard color")
	log.Info("Info message - standard color")
	log.Info("Persisting nodes to database - green!")
	log.Info("Nodes persisted successfully - also green!")
	log.Warn("Warning message - yellow!")
	log.Error("Error message - red!")

	log.Info("")
	log.Info("Database operations are highlighted in green:")
	log.Info("Persisting deduplicated nodes early", "count", 42, "batch_size", 100)
	log.Info("Deduplicated nodes persisted", "duration", "2.5s")
	log.Info("Persisting resolved edges early", "count", 156)
	log.Info("Resolved edges persisted", "duration", "1.8s")

	log.Info("")
	log.Warn("Warnings appear in yellow for attention")
	log.Error("Errors appear in red for immediate visibility")

	log.Info("")
	log.Info("Demo complete!")
}
