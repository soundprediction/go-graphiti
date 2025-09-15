/*
Package crossencoder provides cross-encoder functionality for ranking passages
based on their relevance to a query.

Cross-encoders are used in information retrieval to rerank search results by
computing relevance scores between a query and candidate passages. This package
provides multiple implementations including OpenAI-based, local similarity-based,
and mock implementations for testing.

Usage:

	// Using OpenAI reranker
	llmClient := llm.NewOpenAIClient("api-key", llm.Config{Model: "gpt-4o-mini"})
	reranker := crossencoder.NewOpenAIRerankerClient(llmClient, crossencoder.Config{
		MaxConcurrency: 5,
	})
	
	// Rank passages
	results, err := reranker.Rank(ctx, "search query", []string{
		"passage 1 text",
		"passage 2 text",
	})
	
	// Using local similarity reranker
	localReranker := crossencoder.NewLocalRerankerClient(crossencoder.Config{})
	results, err := localReranker.Rank(ctx, query, passages)

The package supports different reranking strategies:
- OpenAI API-based reranking using boolean classification prompts
- Local text similarity using cosine similarity of term frequency vectors
- Mock implementation for testing with deterministic results
*/
package crossencoder

import (
	"fmt"

	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
)

// Provider represents the type of cross-encoder provider
type Provider string

const (
	// ProviderOpenAI uses OpenAI API for reranking
	ProviderOpenAI Provider = "openai"

	// ProviderLocal uses local text similarity algorithms
	ProviderLocal Provider = "local"

	// ProviderMock uses mock implementation for testing
	ProviderMock Provider = "mock"

	// ProviderJina uses Jina's reranking API (compatible with VLLM)
	ProviderJina Provider = "jina"

	// ProviderEmbedding uses embedding-based similarity for reranking
	ProviderEmbedding Provider = "embedding"
)

// ClientConfig holds configuration for creating cross-encoder clients
type ClientConfig struct {
	Provider        Provider         `json:"provider"`
	Config          Config           `json:"config"`
	LLMClient       llm.Client       `json:"-"` // Not serialized, passed at runtime
	EmbedderClient  embedder.Client  `json:"-"` // Required for embedding provider
	JinaConfig      *JinaConfig      `json:"jina_config,omitempty"`     // Jina-specific config
	EmbeddingConfig *EmbeddingConfig `json:"embedding_config,omitempty"` // Embedding-specific config
}

// NewClient creates a new cross-encoder client based on the provider type
func NewClient(clientConfig ClientConfig) (Client, error) {
	switch clientConfig.Provider {
	case ProviderOpenAI:
		if clientConfig.LLMClient == nil {
			return nil, fmt.Errorf("LLM client is required for OpenAI provider")
		}
		return NewOpenAIRerankerClient(clientConfig.LLMClient, clientConfig.Config), nil

	case ProviderLocal:
		return NewLocalRerankerClient(clientConfig.Config), nil

	case ProviderMock:
		return NewMockRerankerClient(clientConfig.Config), nil

	case ProviderJina:
		jinaConfig := JinaConfig{Config: clientConfig.Config}
		if clientConfig.JinaConfig != nil {
			jinaConfig = *clientConfig.JinaConfig
		}
		return NewJinaRerankerClient(jinaConfig), nil

	case ProviderEmbedding:
		if clientConfig.EmbedderClient == nil {
			return nil, fmt.Errorf("embedder client is required for embedding provider")
		}
		embeddingConfig := EmbeddingConfig{Config: clientConfig.Config}
		if clientConfig.EmbeddingConfig != nil {
			embeddingConfig = *clientConfig.EmbeddingConfig
		}
		return NewEmbeddingRerankerClient(clientConfig.EmbedderClient, embeddingConfig), nil

	default:
		return nil, fmt.Errorf("unsupported cross-encoder provider: %s", clientConfig.Provider)
	}
}

// DefaultConfig returns a default configuration for the given provider
func DefaultConfig(provider Provider) Config {
	switch provider {
	case ProviderOpenAI:
		return Config{
			Model:          "gpt-4o-mini",
			BatchSize:      10,
			MaxConcurrency: 5,
		}
	case ProviderLocal:
		return Config{
			BatchSize: 100, // Local processing can handle larger batches
		}
	case ProviderMock:
		return Config{
			BatchSize: 100,
		}
	case ProviderJina:
		return Config{
			Model:          "jina-reranker-v1-base-en",
			BatchSize:      100, // Jina API can handle large batches
			MaxConcurrency: 3,   // Conservative for external API
		}
	case ProviderEmbedding:
		return Config{
			BatchSize:      50, // Moderate batch size for embedding computation
			MaxConcurrency: 10, // Can be higher since embeddings are typically faster
		}
	default:
		return Config{}
	}
}