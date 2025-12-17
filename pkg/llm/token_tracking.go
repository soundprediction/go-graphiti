package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/soundprediction/go-graphiti/pkg/types"
)

// TokenStats tracks token usage
type TokenStats struct {
	TotalTokens      int `json:"total_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// TokenTracker handles persistence of token usage stats
type TokenTracker struct {
	path  string
	mu    sync.Mutex
	Stats TokenStats `json:"stats"`
}

// NewTokenTracker creates a new token tracker
func NewTokenTracker(path string) (*TokenTracker, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	tracker := &TokenTracker{
		path: absPath,
	}

	// Load existing stats if file occurs
	if err := tracker.Create(); err != nil {
		// If loading fails, just start fresh (or return error depending on strictness)
		// For now, we prefer not to fail hard on tracking issues
		fmt.Printf("Warning: Failed to load previous token stats: %v\n", err)
	}

	return tracker, nil
}

// Create loads or creates the tracking file
func (t *TokenTracker) Create() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create directory if not exists
	dir := filepath.Dir(t.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Read file if exists
	data, err := os.ReadFile(t.path)
	if err == nil {
		if err := json.Unmarshal(data, &t.Stats); err != nil {
			return fmt.Errorf("failed to parse stats: %w", err)
		}
	}

	return nil
}

// AddUsage adds usage to the tracker and saves explicitly
func (t *TokenTracker) AddUsage(usage *types.TokenUsage) error {
	if usage == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.Stats.TotalTokens += usage.TotalTokens
	t.Stats.PromptTokens += usage.PromptTokens
	t.Stats.CompletionTokens += usage.CompletionTokens

	return t.saveLocked()
}

// saveLocked saves stats to disk (must hold lock)
func (t *TokenTracker) saveLocked() error {
	data, err := json.MarshalIndent(t.Stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	return os.WriteFile(t.path, data, 0644)
}

// TokenTrackingClient wraps a Client to track usage
type TokenTrackingClient struct {
	client  Client
	tracker *TokenTracker
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
		if err := c.tracker.AddUsage(resp.TokensUsed); err != nil {
			fmt.Printf("Warning: Failed to save token usage: %v\n", err)
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
		if err := c.tracker.AddUsage(resp.TokensUsed); err != nil {
			fmt.Printf("Warning: Failed to save token usage: %v\n", err)
		}
	}

	return resp, nil
}

// Close implements Client
func (c *TokenTrackingClient) Close() error {
	return c.client.Close()
}
