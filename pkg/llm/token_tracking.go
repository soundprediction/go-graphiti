package llm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/google/uuid"
	"github.com/soundprediction/go-graphiti/pkg/cost"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// TokenUsageRecord represents a single log entry for token usage
type TokenUsageRecord struct {
	ID               string
	Timestamp        time.Time
	Model            string
	TotalTokens      int
	PromptTokens     int
	CompletionTokens int
	EstimatedCost    float64
	UserID           string
	SessionID        string
	RequestSource    string
	IngestionSource  string
	IsSystemCall     bool
}

// TokenTracker handles persistence of token usage stats
type TokenTracker struct {
	db             *sql.DB
	costCalculator *cost.CostCalculator
	Model          string // Default model fallback
}

// NewTokenTracker creates a new token tracker using an existing DuckDB connection
func NewTokenTracker(db *sql.DB) (*TokenTracker, error) {
	tracker := &TokenTracker{
		db:             db,
		costCalculator: cost.NewCostCalculator(),
	}

	if err := tracker.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return tracker, nil
}

// initSchema creates the necessary table if it doesn't exist
func (t *TokenTracker) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS token_usage (
		id VARCHAR,
		timestamp TIMESTAMP,
		model VARCHAR,
		total_tokens INTEGER,
		prompt_tokens INTEGER,
		completion_tokens INTEGER,
		cost_usd DOUBLE,
		user_id VARCHAR,
		session_id VARCHAR,
		request_source VARCHAR,
		ingestion_source VARCHAR,
		is_system_call BOOLEAN
	);
	`
	_, err := t.db.Exec(query)
	// Check if cost_usd column exists (migration)
	// DuckDB allows ADD COLUMN IF NOT EXISTS in newer versions, or we can catch error
	// Simple approach: try to add column, ignore error if exists
	t.db.Exec("ALTER TABLE token_usage ADD COLUMN IF NOT EXISTS cost_usd DOUBLE")

	return err
}

// AddUsage adds usage to the tracker
func (t *TokenTracker) AddUsage(ctx context.Context, usage *types.TokenUsage, model string) error {
	if usage == nil {
		return nil
	}

	costUSD := t.costCalculator.CalculateCost(model, usage.PromptTokens, usage.CompletionTokens)

	record := TokenUsageRecord{
		ID:               uuid.New().String(),
		Timestamp:        time.Now().UTC(),
		Model:            model,
		TotalTokens:      usage.TotalTokens,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		EstimatedCost:    costUSD,
	}

	// Extract context
	if v, ok := ctx.Value(types.ContextKeyUserID).(string); ok {
		record.UserID = v
	}
	if v, ok := ctx.Value(types.ContextKeySessionID).(string); ok {
		record.SessionID = v
	}
	if v, ok := ctx.Value(types.ContextKeyRequestSource).(string); ok {
		record.RequestSource = v
	}
	if v, ok := ctx.Value(types.ContextKeyIngestionSource).(string); ok {
		record.IngestionSource = v
	}
	if v, ok := ctx.Value(types.ContextKeySystemCall).(bool); ok {
		record.IsSystemCall = v
	}

	query := `
	INSERT INTO token_usage (
		id, timestamp, model, total_tokens, prompt_tokens, completion_tokens, cost_usd,
		user_id, session_id, request_source, ingestion_source, is_system_call
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	_, err := t.db.Exec(query,
		record.ID,
		record.Timestamp,
		record.Model,
		record.TotalTokens,
		record.PromptTokens,
		record.CompletionTokens,
		record.EstimatedCost,
		record.UserID,
		record.SessionID,
		record.RequestSource,
		record.IngestionSource,
		record.IsSystemCall,
	)

	return err
}

// TokenTrackingClient wraps a Client to track usage
type TokenTrackingClient struct {
	client  Client
	tracker *TokenTracker
	// We might store config reference to get default model if needed
}

// NewTokenTrackingClient creates a wrapper client
func NewTokenTrackingClient(client Client, tracker *TokenTracker) *TokenTrackingClient {
	return &TokenTrackingClient{
		client:  client,
		tracker: tracker,
	}
}

// Chat implements Client
func (c *TokenTrackingClient) Chat(ctx context.Context, messages []types.Message) (*types.Response, error) {
	resp, err := c.client.Chat(ctx, messages)
	if err != nil {
		return nil, err
	}

	if resp.TokensUsed != nil {
		// Use model from response if available
		model := resp.Model
		if model == "" {
			model = "unknown"
		}

		if err := c.tracker.AddUsage(ctx, resp.TokensUsed, model); err != nil {
			fmt.Printf("Warning: Failed to log token usage: %v\n", err)
		}
	}

	return resp, nil
}

// ChatWithStructuredOutput implements Client
func (c *TokenTrackingClient) ChatWithStructuredOutput(ctx context.Context, messages []types.Message, schema any) (*types.Response, error) {
	resp, err := c.client.ChatWithStructuredOutput(ctx, messages, schema)
	if err != nil {
		return nil, err
	}

	if resp.TokensUsed != nil {
		// Use model from response if available
		model := resp.Model
		if model == "" {
			model = "unknown"
		}

		if err := c.tracker.AddUsage(ctx, resp.TokensUsed, model); err != nil {
			fmt.Printf("Warning: Failed to log token usage: %v\n", err)
		}
	}

	return resp, nil
}

// Close implements Client
func (c *TokenTrackingClient) Close() error {
	return c.client.Close()
}
