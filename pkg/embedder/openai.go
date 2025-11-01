package embedder

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Constants for retry configuration
const (
	DefaultMaxRetries = 3
)

// OpenAIEmbedder implements the Client interface for OpenAI's embedding models.
type OpenAIEmbedder struct {
	client     *openai.Client
	config     Config
	maxRetries int
}

// NewOpenAIEmbedder creates a new OpenAI embedder client.
// Supports OpenAI-compatible services through custom BaseURL configuration.
func NewOpenAIEmbedder(apiKey string, config Config) *OpenAIEmbedder {
	var client *openai.Client

	if config.BaseURL != "" {
		// Create client with custom base URL for OpenAI-compatible services
		clientConfig := openai.DefaultConfig(apiKey)
		clientConfig.BaseURL = config.BaseURL
		client = openai.NewClientWithConfig(clientConfig)
	} else {
		// Use default OpenAI client
		client = openai.NewClient(apiKey)
	}

	if config.Model == "" {
		config.Model = "text-embedding-ada-002"
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.Dimensions == 0 {
		// Set default dimensions based on model
		switch config.Model {
		case "text-embedding-ada-002":
			config.Dimensions = 1536
		case "text-embedding-3-small":
			config.Dimensions = 1536
		case "text-embedding-3-large":
			config.Dimensions = 3072
		default:
			// config.Dimensions = 1536
			config.Dimensions = 0
		}
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = DefaultMaxRetries
	}

	return &OpenAIEmbedder{
		client:     client,
		config:     config,
		maxRetries: config.MaxRetries,
	}
}

// Embed generates embeddings for multiple texts.
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	var allEmbeddings [][]float32

	// Process in batches
	for i := 0; i < len(texts); i += e.config.BatchSize {
		end := i + e.config.BatchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		batchEmbeddings, err := e.embedBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to embed batch %d-%d: %w", i, end, err)
		}

		allEmbeddings = append(allEmbeddings, batchEmbeddings...)
	}

	return allEmbeddings, nil
}

// EmbedSingle generates an embedding for a single text.
func (e *OpenAIEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embeddings[0], nil
}

// Dimensions returns the number of dimensions in the embeddings.
func (e *OpenAIEmbedder) Dimensions() int {
	return e.config.Dimensions
}

// Close cleans up resources (no-op for OpenAI embedder).
func (e *OpenAIEmbedder) Close() error {
	return nil
}

func (e *OpenAIEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	var lastError error

	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			backoff := time.Duration(attempt*attempt) * time.Second
			log.Printf("Retrying embedding request after %v (attempt %d/%d)", backoff, attempt+1, e.maxRetries+1)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req := openai.EmbeddingRequest{
			Input: texts,
			Model: openai.EmbeddingModel(e.config.Model),
		}

		// Add custom dimensions if specified (useful for OpenAI-compatible services)
		if e.config.Dimensions > 0 {
			req.Dimensions = e.config.Dimensions
		}

		resp, err := e.client.CreateEmbeddings(ctx, req)
		if err != nil {
			lastError = err

			// Check if this is a rate limit error
			if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "rate_limit") {
				if attempt < e.maxRetries {
					continue
				}
				return nil, fmt.Errorf("rate limit exceeded: %w", err)
			}

			// Check for retriable errors
			if isRetriableEmbeddingError(err) && attempt < e.maxRetries {
				continue
			}

			// Non-retriable error, return immediately
			return nil, fmt.Errorf("openai embedding request failed: %w", err)
		}

		// Success, convert and return embeddings
		embeddings := make([][]float32, len(resp.Data))
		for i, embedding := range resp.Data {
			// Convert []float64 to []float32
			float32Embedding := make([]float32, len(embedding.Embedding))
			for j, val := range embedding.Embedding {
				float32Embedding[j] = float32(val)
			}
			embeddings[i] = float32Embedding
		}

		return embeddings, nil
	}

	// All retries exhausted
	return nil, fmt.Errorf("all retries exhausted, last error: %w", lastError)
}

// isRetriableEmbeddingError determines if an embedding error should trigger a retry
func isRetriableEmbeddingError(err error) bool {
	errStr := strings.ToLower(err.Error())
	retriableErrors := []string{
		"timeout",
		"connection",
		"internal server error",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"temporary failure",
		"network error",
	}

	for _, retriable := range retriableErrors {
		if strings.Contains(errStr, retriable) {
			return true
		}
	}

	return false
}
