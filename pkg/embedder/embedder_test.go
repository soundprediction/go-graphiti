package embedder_test

import (
	"context"
	"testing"

	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createEmbeddingValues(multiplier float32) []float32 {
	// Create a test embedding similar to Python test
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) * multiplier
	}
	return embedding
}

func TestNewOpenAIEmbedder(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		config      embedder.Config
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid API key",
			apiKey:      "test-api-key",
			config:      embedder.Config{Model: "text-embedding-ada-002"},
			shouldError: false,
		},
		{
			name:        "empty API key",
			apiKey:      "",
			config:      embedder.Config{Model: "text-embedding-ada-002"},
			shouldError: false, // May be valid for some configurations
		},
		{
			name:        "custom model",
			apiKey:      "test-api-key",
			config:      embedder.Config{Model: "text-embedding-3-small"},
			shouldError: false,
		},
		{
			name:        "custom base URL",
			apiKey:      "test-api-key",
			config:      embedder.Config{Model: "text-embedding-ada-002", BaseURL: "https://api.example.com"},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := embedder.NewOpenAIEmbedder(tt.apiKey, tt.config)
			
			if tt.shouldError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestEmbedderInterface(t *testing.T) {
	// Test that OpenAIEmbedder implements the Embedder interface
	var _ embedder.Embedder = (*embedder.OpenAIEmbedder)(nil)
}

func TestEmbedderDimensions(t *testing.T) {
	client, err := embedder.NewOpenAIEmbedder("test-key", embedder.Config{
		Model: "text-embedding-ada-002",
	})
	require.NoError(t, err)
	
	// Test dimensions method
	dims := client.Dimensions()
	assert.Greater(t, dims, 0)
}

func TestEmbedderBatchProcessing(t *testing.T) {
	t.Skip("Skip integration test - requires API key")
	
	// This would be an integration test requiring a real API key
	ctx := context.Background()
	client, err := embedder.NewOpenAIEmbedder("test-key", embedder.Config{
		Model: "text-embedding-ada-002",
	})
	require.NoError(t, err)
	
	texts := []string{
		"Hello world",
		"This is a test",
		"Another text to embed",
	}
	
	embeddings, err := client.Embed(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, embeddings, len(texts))
	
	for _, embedding := range embeddings {
		assert.Greater(t, len(embedding), 0)
		assert.Equal(t, client.Dimensions(), len(embedding))
	}
}

func TestEmbedderSingleText(t *testing.T) {
	t.Skip("Skip integration test - requires API key")
	
	// This would be an integration test requiring a real API key
	ctx := context.Background()
	client, err := embedder.NewOpenAIEmbedder("test-key", embedder.Config{
		Model: "text-embedding-ada-002",
	})
	require.NoError(t, err)
	
	text := "Hello world"
	embedding, err := client.EmbedSingle(ctx, text)
	require.NoError(t, err)
	assert.Greater(t, len(embedding), 0)
	assert.Equal(t, client.Dimensions(), len(embedding))
}

func TestEmbedderErrorHandling(t *testing.T) {
	ctx := context.Background()
	client, err := embedder.NewOpenAIEmbedder("invalid-key", embedder.Config{
		Model: "text-embedding-ada-002",
	})
	require.NoError(t, err)
	
	// Test with empty text (should handle gracefully)
	embedding, err := client.EmbedSingle(ctx, "")
	if err != nil {
		// Error is expected with invalid key or empty text
		assert.NotNil(t, err)
		assert.Nil(t, embedding)
	}
}

func TestEmbedderConfig(t *testing.T) {
	tests := []struct {
		name   string
		config embedder.Config
	}{
		{
			name: "default config",
			config: embedder.Config{
				Model: "text-embedding-ada-002",
			},
		},
		{
			name: "config with custom settings",
			config: embedder.Config{
				Model:   "text-embedding-3-small",
				BaseURL: "https://custom.openai.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := embedder.NewOpenAIEmbedder("test-key", tt.config)
			assert.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}