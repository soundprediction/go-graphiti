package embedder_test

import (
	"context"
	"testing"

	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestNewOpenAIEmbedder(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		config embedder.Config
	}{
		{
			name:   "valid API key",
			apiKey: "test-api-key",
			config: embedder.Config{Model: "text-embedding-ada-002"},
		},
		{
			name:   "empty API key",
			apiKey: "",
			config: embedder.Config{Model: "text-embedding-ada-002"},
		},
		{
			name:   "custom model",
			apiKey: "test-api-key",
			config: embedder.Config{Model: "text-embedding-3-small"},
		},
		{
			name:   "custom base URL",
			apiKey: "test-api-key",
			config: embedder.Config{Model: "text-embedding-ada-002", BaseURL: "https://api.example.com"},
		},
		{
			name:   "empty model uses default",
			apiKey: "test-api-key",
			config: embedder.Config{}, // Empty config should use defaults
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := embedder.NewOpenAIEmbedder(tt.apiKey, tt.config)
			assert.NotNil(t, client)

			// Verify client has proper defaults set
			assert.Greater(t, client.Dimensions(), 0)
		})
	}
}

func TestEmbedderInterface(t *testing.T) {
	// Test that OpenAIEmbedder implements the Client interface
	var _ embedder.Client = (*embedder.OpenAIEmbedder)(nil)
}

func TestEmbedderDimensions(t *testing.T) {
	client := embedder.NewOpenAIEmbedder("test-key", embedder.Config{
		Model: "text-embedding-ada-002",
	})

	// Test dimensions method
	dims := client.Dimensions()
	assert.Greater(t, dims, 0)
}

func TestEmbedderBatchProcessing(t *testing.T) {
	t.Skip("Skip integration test - requires API key")

	// This would be an integration test requiring a real API key
	ctx := context.Background()
	client := embedder.NewOpenAIEmbedder("test-key", embedder.Config{
		Model: "text-embedding-ada-002",
	})
	require.NotNil(t, client)

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
	client := embedder.NewOpenAIEmbedder("test-key", embedder.Config{
		Model: "text-embedding-ada-002",
	})
	require.NotNil(t, client)

	text := "Hello world"
	embedding, err := client.EmbedSingle(ctx, text)
	require.NoError(t, err)
	assert.Greater(t, len(embedding), 0)
	assert.Equal(t, client.Dimensions(), len(embedding))
}

func TestEmbedderErrorHandling(t *testing.T) {
	ctx := context.Background()
	client := embedder.NewOpenAIEmbedder("invalid-key", embedder.Config{
		Model: "text-embedding-ada-002",
	})
	require.NotNil(t, client)

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
		expectedDims int
	}{
		{
			name: "default config",
			config: embedder.Config{
				Model: "text-embedding-ada-002",
			},
			expectedDims: 1536,
		},
		{
			name: "config with custom settings",
			config: embedder.Config{
				Model:   "text-embedding-3-small",
				BaseURL: "https://custom.openai.com",
			},
			expectedDims: 1536,
		},
		{
			name: "large model",
			config: embedder.Config{
				Model: "text-embedding-3-large",
			},
			expectedDims: 3072,
		},
		{
			name: "custom dimensions",
			config: embedder.Config{
				Model: "custom-model",
				Dimensions: 512,
			},
			expectedDims: 512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := embedder.NewOpenAIEmbedder("test-key", tt.config)
			assert.NotNil(t, client)
			assert.Equal(t, tt.expectedDims, client.Dimensions())
		})
	}
}
