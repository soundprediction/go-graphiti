# LLM Package - Retry Functionality

## Overview

The LLM package includes a retry wrapper that automatically retries failed LLM API calls when encountering transient errors like server errors (500, 503), timeouts, or rate limits.

## Features

- **Automatic Retry**: Automatically retries failed API calls
- **Exponential Backoff**: Uses exponential backoff to avoid overwhelming servers
- **Smart Error Detection**: Distinguishes between retryable (5xx, timeouts) and non-retryable errors (4xx)
- **Configurable**: Customize max retries, delays, and backoff multiplier
- **Context Aware**: Respects context cancellation during retries

## Usage

### Basic Usage with Default Configuration

```go
import (
    "github.com/soundprediction/go-predicato/pkg/llm"
)

// Create your base LLM client
baseClient, err := llm.NewOpenAIGenericClient(apiKey, llm.Config{
    Model: "gpt-4",
})
if err != nil {
    return err
}

// Wrap with retry logic using defaults (3 retries, 1s initial delay, 60s max delay)
retryClient := llm.NewRetryClient(baseClient, llm.DefaultRetryConfig())

// Use the retry client as normal
response, err := retryClient.Chat(ctx, messages)
```

### Custom Retry Configuration

```go
// Create custom retry configuration
retryConfig := &llm.RetryConfig{
    MaxRetries:        5,                    // Retry up to 5 times
    InitialDelay:      2 * time.Second,      // Start with 2 second delay
    MaxDelay:          120 * time.Second,    // Cap delays at 2 minutes
    BackoffMultiplier: 3.0,                  // Triple delay each retry
}

retryClient := llm.NewRetryClient(baseClient, retryConfig)
```

### Retry Behavior

The retry client will automatically retry on the following errors:

- **HTTP 5xx errors**: 500, 502, 503, 504 (server errors)
- **HTTP 429**: Rate limit exceeded / Too many requests
- **Timeouts**: Connection timeouts, gateway timeouts
- **Connection errors**: Connection reset, connection refused
- **Rate limit errors**: Custom `RateLimitError` type

It will **NOT** retry on:

- **HTTP 4xx errors**: 400 (bad request), 401 (unauthorized), 403 (forbidden), 404 (not found)
- **Refusal errors**: When the LLM refuses to respond
- **Empty response errors**: When configured as non-retryable

### Exponential Backoff

The delay between retries follows an exponential backoff pattern:

- **Retry 1**: `InitialDelay` (default: 1s)
- **Retry 2**: `InitialDelay * BackoffMultiplier` (default: 2s)
- **Retry 3**: `InitialDelay * BackoffMultiplier^2` (default: 4s)
- And so on, capped at `MaxDelay` (default: 60s)

Example with defaults:
- Retry 1: Wait 1 second
- Retry 2: Wait 2 seconds
- Retry 3: Wait 4 seconds
- Retry 4: Wait 8 seconds (if MaxRetries > 3)

### Context Cancellation

The retry client respects context cancellation. If the context is cancelled during a backoff period, the retry will stop immediately:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Will stop retrying if 10 seconds elapses
response, err := retryClient.Chat(ctx, messages)
```

## Configuration Reference

### RetryConfig

```go
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
```

### Default Configuration

```go
func DefaultRetryConfig() *RetryConfig {
    return &RetryConfig{
        MaxRetries:        3,
        InitialDelay:      1 * time.Second,
        MaxDelay:          60 * time.Second,
        BackoffMultiplier: 2.0,
    }
}
```

## Examples

### Example 1: Production Use with Conservative Retries

```go
// Conservative production settings
retryConfig := &llm.RetryConfig{
    MaxRetries:        2,                  // Only retry twice
    InitialDelay:      500 * time.Millisecond,
    MaxDelay:          30 * time.Second,
    BackoffMultiplier: 2.0,
}

client := llm.NewRetryClient(baseClient, retryConfig)
```

### Example 2: Aggressive Retries for Batch Processing

```go
// Aggressive retries for background batch jobs
retryConfig := &llm.RetryConfig{
    MaxRetries:        10,                 // Retry up to 10 times
    InitialDelay:      1 * time.Second,
    MaxDelay:          5 * time.Minute,    // Allow long delays
    BackoffMultiplier: 2.0,
}

client := llm.NewRetryClient(baseClient, retryConfig)
```

### Example 3: Quick Retries for Real-time Systems

```go
// Fast retries for real-time systems
retryConfig := &llm.RetryConfig{
    MaxRetries:        3,
    InitialDelay:      100 * time.Millisecond,  // Start fast
    MaxDelay:          5 * time.Second,         // Don't wait too long
    BackoffMultiplier: 1.5,                     // Slower growth
}

client := llm.NewRetryClient(baseClient, retryConfig)
```

## Error Handling

When all retries are exhausted, the retry client returns an error that wraps the original error:

```go
response, err := retryClient.Chat(ctx, messages)
if err != nil {
    // Check if it's a wrapped retry failure
    if strings.Contains(err.Error(), "failed after") {
        // All retries exhausted
        log.Printf("Failed after retries: %v", err)
    }
    return err
}
```

## Testing

The retry functionality includes comprehensive tests. Run them with:

```bash
go test ./pkg/llm -run TestRetry -v
```

## Best Practices

1. **Use Default Config for Most Cases**: The default configuration is suitable for most use cases
2. **Adjust for Your SLA**: Configure based on your service's SLA requirements
3. **Consider Timeout**: Set appropriate context timeouts to prevent indefinite waiting
4. **Monitor Retry Rates**: Track how often retries occur to identify API stability issues
5. **Don't Over-Retry**: Too many retries can cause cascading failures
