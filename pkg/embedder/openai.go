package embedder

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAIEmbedder implements the Client interface for OpenAI's embedding models.
type OpenAIEmbedder struct {
	client *openai.Client
	config Config
}

// NewOpenAIEmbedder creates a new OpenAI embedder client.
func NewOpenAIEmbedder(apiKey string, config Config) *OpenAIEmbedder {
	client := openai.NewClient(apiKey)
	
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
			config.Dimensions = 1536
		}
	}
	
	return &OpenAIEmbedder{
		client: client,
		config: config,
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
	req := openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(e.config.Model),
	}
	
	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai embedding request failed: %w", err)
	}
	
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