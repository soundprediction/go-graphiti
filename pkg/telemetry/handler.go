package telemetry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/google/uuid"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// DuckDBHandler is a slog.Handler that writes error logs to DuckDB
type DuckDBHandler struct {
	next slog.Handler
	db   *sql.DB
}

// NewDuckDBHandler creates a new DuckDBHandler
func NewDuckDBHandler(next slog.Handler, db *sql.DB) (*DuckDBHandler, error) {
	h := &DuckDBHandler{
		next: next,
		db:   db,
	}

	if err := h.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return h, nil
}

// initSchema creates the execution_errors table
func (h *DuckDBHandler) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS execution_errors (
		id VARCHAR,
		timestamp TIMESTAMP,
		level VARCHAR,
		message VARCHAR,
		user_id VARCHAR,
		session_id VARCHAR,
		request_source VARCHAR,
		source_file VARCHAR,
		line_number INTEGER,
		attributes JSON
	);
	`
	_, err := h.db.Exec(query)
	return err
}

// Enabled implements slog.Handler
func (h *DuckDBHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle implements slog.Handler
func (h *DuckDBHandler) Handle(ctx context.Context, r slog.Record) error {
	// Always pass to next handler first
	if err := h.next.Handle(ctx, r); err != nil {
		return err
	}

	// Only log errors (and above) to DB
	if r.Level < slog.LevelError {
		return nil
	}

	// Extract context info
	var userID, sessionID, requestSource string
	if v, ok := ctx.Value(types.ContextKeyUserID).(string); ok {
		userID = v
	}
	if v, ok := ctx.Value(types.ContextKeySessionID).(string); ok {
		sessionID = v
	}
	if v, ok := ctx.Value(types.ContextKeyRequestSource).(string); ok {
		requestSource = v
	}

	// Extract attributes
	attrs := make(map[string]interface{})
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	// Add error from record if present (standard key "error" or just msg)
	// slog often puts the error in the message or attributes.

	attrsJson, _ := json.Marshal(attrs)

	// Get source info
	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()
	sourceFile := f.File
	line := f.Line

	// Insert into DB
	id := uuid.New().String()
	timestamp := r.Time.UTC()
	level := r.Level.String()
	msg := r.Message

	query := `
	INSERT INTO execution_errors (
		id, timestamp, level, message, 
		user_id, session_id, request_source,
		source_file, line_number, attributes
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	// Fire and forget - don't block heavily on DB (though Scan is blocking, we hope it's fast)
	// For production, this might want to be a buffered channel worker.
	// For now, simple direct write.
	go func() {
		_, err := h.db.Exec(query,
			id, timestamp, level, msg,
			userID, sessionID, requestSource,
			sourceFile, line, string(attrsJson),
		)
		if err != nil {
			// Fallback: print to stderr if DB logging fails
			fmt.Printf("Failed to log error to DuckDB: %v\n", err)
		}
	}()

	return nil
}

// WithAttrs implements slog.Handler
func (h *DuckDBHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &DuckDBHandler{
		next: h.next.WithAttrs(attrs),
		db:   h.db,
	}
}

// WithGroup implements slog.Handler
func (h *DuckDBHandler) WithGroup(name string) slog.Handler {
	return &DuckDBHandler{
		next: h.next.WithGroup(name),
		db:   h.db,
	}
}
