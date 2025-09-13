/*
Package crossencoder provides cross-encoder functionality for ranking passages
based on their relevance to a query.

# Overview

Cross-encoders are neural models used in information retrieval and natural language
processing to compute relevance scores between a query and candidate passages. Unlike
bi-encoders that encode queries and documents separately, cross-encoders process 
query-document pairs together, often resulting in better ranking accuracy at the cost
of increased computational overhead.

# Implementations

This package provides several implementations:

## OpenAI Reranker (OpenAIRerankerClient)

Uses OpenAI's API to run boolean classification prompts for each passage. The model
determines whether each passage is relevant to the query, and log-probabilities are
used to compute relevance scores.

	llmClient := llm.NewOpenAIClient("api-key", llm.Config{Model: "gpt-4o-mini"})
	reranker := crossencoder.NewOpenAIRerankerClient(llmClient, crossencoder.Config{
		MaxConcurrency: 5,
	})
	
	results, err := reranker.Rank(ctx, "search query", passages)

## Local Reranker (LocalRerankerClient)

Uses local text similarity algorithms, specifically cosine similarity of term frequency
vectors. This implementation doesn't require external API calls and provides reasonable
results for basic text matching scenarios.

	reranker := crossencoder.NewLocalRerankerClient(crossencoder.Config{})
	results, err := reranker.Rank(ctx, query, passages)

## Mock Reranker (MockRerankerClient)

Provides a deterministic mock implementation for testing purposes. Uses simple text
similarity heuristics with consistent but varied results based on content hashing.

	reranker := crossencoder.NewMockRerankerClient(crossencoder.Config{})
	results, err := reranker.Rank(ctx, query, passages)

# Factory Function

The NewClient function provides a convenient way to create clients based on provider type:

	client, err := crossencoder.NewClient(crossencoder.ClientConfig{
		Provider: crossencoder.ProviderOpenAI,
		Config:   crossencoder.DefaultConfig(crossencoder.ProviderOpenAI),
		LLMClient: llmClient, // Required for OpenAI provider
	})

# Configuration

Each implementation accepts a Config struct with provider-specific options:

	config := crossencoder.Config{
		Model:          "gpt-4o-mini",     // Model name (OpenAI only)
		BatchSize:      10,                // Batch processing size
		MaxConcurrency: 5,                 // Max concurrent requests (OpenAI only)
	}

# Usage in Search

Cross-encoders are typically used as rerankers in multi-stage retrieval systems:

1. Initial retrieval using fast methods (e.g., BM25, vector similarity)
2. Reranking top candidates using cross-encoder for improved relevance

The package integrates with the search functionality in go-graphiti to provide
reranking capabilities for nodes, edges, episodes, and communities.

# Performance Considerations

- OpenAI reranker: Higher accuracy but requires API calls and has rate limits
- Local reranker: Fast and no external dependencies but lower accuracy
- Mock reranker: Fastest, suitable for testing and development

Choose the implementation based on your accuracy requirements, latency constraints,
and available resources.
*/
package crossencoder