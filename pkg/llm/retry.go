package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (default: 3)
	MaxRetries int
	// InitialDelay is the initial delay before the first retry (default: 1 second)
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries (default: 60 seconds)
	MaxDelay time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff (default: 2.0)
	BackoffMultiplier float64
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          60 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// RetryClient wraps an LLM client and adds retry logic with exponential backoff
type RetryClient struct {
	client Client
	config *RetryConfig
}

// NewRetryClient creates a new retry client wrapper
func NewRetryClient(client Client, config *RetryConfig) *RetryClient {
	if config == nil {
		config = DefaultRetryConfig()
	}
	// Ensure sensible defaults
	if config.MaxRetries < 0 {
		config.MaxRetries = 3
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 1 * time.Second
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 60 * time.Second
	}
	if config.BackoffMultiplier <= 0 {
		config.BackoffMultiplier = 2.0
	}

	return &RetryClient{
		client: client,
		config: config,
	}
}

// Chat implements the Client interface with retry logic
func (r *RetryClient) Chat(ctx context.Context, messages []Message) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// If this is a retry, wait with exponential backoff
		if attempt > 0 {
			delay := r.calculateDelay(attempt)
			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
			}
		}

		// Make the LLM call
		resp, err := r.client.Chat(ctx, messages)
		if err == nil {
			return resp, nil
		}

		// Store the error
		lastErr = err

		// Check if the error is retryable
		if !isRetryableError(err) {
			// Non-retryable error, fail immediately
			return nil, err
		}

		// Log retry attempt (in production, this should use a logger)
		// For now, we just continue to retry
	}

	// All retries exhausted
	return nil, fmt.Errorf("failed after %d retries: %w", r.config.MaxRetries, lastErr)
}

// ChatWithStructuredOutput implements the Client interface with retry logic
func (r *RetryClient) ChatWithStructuredOutput(ctx context.Context, messages []Message, schema any) (json.RawMessage, error) {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// If this is a retry, wait with exponential backoff
		if attempt > 0 {
			delay := r.calculateDelay(attempt)
			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
			}
		}

		// Make the LLM call
		result, err := r.client.ChatWithStructuredOutput(ctx, messages, schema)
		if err == nil {
			return result, nil
		}

		// Store the error
		lastErr = err

		// Check if the error is retryable
		if !isRetryableError(err) {
			// Non-retryable error, fail immediately
			return nil, err
		}

		// Log retry attempt (in production, this should use a logger)
		// For now, we just continue to retry
	}

	// All retries exhausted
	return nil, fmt.Errorf("failed after %d retries: %w", r.config.MaxRetries, lastErr)
}

// Close implements the Client interface
func (r *RetryClient) Close() error {
	return r.client.Close()
}

// calculateDelay calculates the delay for a given retry attempt using exponential backoff
func (r *RetryClient) calculateDelay(attempt int) time.Duration {
	// Calculate exponential backoff: InitialDelay * (BackoffMultiplier ^ (attempt - 1))
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.BackoffMultiplier, float64(attempt-1))

	// Cap at MaxDelay
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	return time.Duration(delay)
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Rate limit errors should be retried
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	// Check for standard rate limit error
	if errors.Is(err, ErrRateLimit) {
		return true
	}

	// Check error message for common retryable patterns
	errMsg := strings.ToLower(err.Error())

	// HTTP 5xx errors (server errors)
	retryablePatterns := []string{
		"500", "internal server error",
		"502", "bad gateway",
		"503", "service unavailable",
		"504", "gateway timeout",
		"timeout",
		"connection reset",
		"connection refused",
		"temporary failure",
		"rate limit",
		"too many requests",
		"429",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// Check for HTTP status codes if available
	// This will work with errors that wrap http.Response
	type httpErrorWithStatusCode interface {
		HTTPStatusCode() int
	}

	if httpErr, ok := err.(httpErrorWithStatusCode); ok {
		statusCode := httpErr.HTTPStatusCode()
		// Retry on 5xx errors and 429 (rate limit)
		if statusCode >= 500 || statusCode == http.StatusTooManyRequests {
			return true
		}
	}

	return false
}
