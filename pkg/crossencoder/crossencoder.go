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
)

// ClientConfig holds configuration for creating cross-encoder clients
type ClientConfig struct {
	Provider       Provider    `json:"provider"`
	Config         Config      `json:"config"`
	LLMClient      llm.Client  `json:"-"` // Not serialized, passed at runtime
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
	default:
		return Config{}
	}
}