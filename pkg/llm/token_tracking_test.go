package llm

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/telemetry"
	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuckDBTokenTracker(t *testing.T) {
	// Create temp dir for db
	tempDir, err := os.MkdirTemp("", "graphiti-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "token_usage.duckdb")

	db, err := sql.Open("duckdb", dbPath)
	require.NoError(t, err)
	defer db.Close()

	tracker, err := NewTokenTracker(db)
	require.NoError(t, err)

	ctx := context.Background()
	ctx = context.WithValue(ctx, types.ContextKeyUserID, "test-user")
	ctx = context.WithValue(ctx, types.ContextKeySessionID, "test-session")
	ctx = context.WithValue(ctx, types.ContextKeyRequestSource, "test-source")
	ctx = context.WithValue(ctx, types.ContextKeyIngestionSource, "test-episode")
	ctx = context.WithValue(ctx, types.ContextKeySystemCall, true)

	usage := &types.TokenUsage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}
	model := "gpt-4-test"

	err = tracker.AddUsage(ctx, usage, model)
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM token_usage").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify Token Data
	var userID, sessionID, modelDB string
	var total, prompt, completion int
	var isSystem bool

	err = db.QueryRow("SELECT user_id, session_id, model, total_tokens, prompt_tokens, completion_tokens, is_system_call FROM token_usage").Scan(&userID, &sessionID, &modelDB, &total, &prompt, &completion, &isSystem)
	require.NoError(t, err)

	assert.Equal(t, "test-user", userID)
	assert.Equal(t, "test-session", sessionID)
	assert.Equal(t, "gpt-4-test", modelDB)

	// Test Error Tracking
	handler, err := telemetry.NewDuckDBHandler(slog.Default().Handler(), tracker.db)
	require.NoError(t, err)

	logger := slog.New(handler)

	ctxError := context.WithValue(context.Background(), types.ContextKeyUserID, "error-user")
	logger.ErrorContext(ctxError, "test error message", "key", "val")

	// Wait for async write
	time.Sleep(100 * time.Millisecond)

	var errorCount int
	err = db.QueryRow("SELECT COUNT(*) FROM execution_errors").Scan(&errorCount)
	require.NoError(t, err)
	assert.Equal(t, 1, errorCount)

	var errUser, errMsg, errLevel string
	err = db.QueryRow("SELECT user_id, message, level FROM execution_errors").Scan(&errUser, &errMsg, &errLevel)
	require.NoError(t, err)
	assert.Equal(t, "error-user", errUser)
	assert.Equal(t, "test error message", errMsg)
	assert.Equal(t, "ERROR", errLevel)
}
