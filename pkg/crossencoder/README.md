# CrossEncoder Package

The `crossencoder` package provides cross-encoder functionality for ranking passages based on their relevance to a query. This is a Go port of the cross-encoder functionality from the Python Graphiti project.

## Overview

Cross-encoders are neural models used in information retrieval to compute relevance scores between a query and candidate passages. Unlike bi-encoders that encode queries and documents separately, cross-encoders process query-document pairs together, often resulting in better ranking accuracy.

## Implementations

### 1. OpenAI Reranker (`OpenAIRerankerClient`)

Uses OpenAI's API with boolean classification prompts to determine passage relevance.

```go
llmClient := llm.NewOpenAIClient("api-key", llm.Config{Model: "gpt-4o-mini"})
reranker := crossencoder.NewOpenAIRerankerClient(llmClient, crossencoder.Config{
    MaxConcurrency: 5,
})
defer reranker.Close()

results, err := reranker.Rank(ctx, "machine learning", passages)
```

### 2. Local Reranker (`LocalRerankerClient`)

Uses local text similarity algorithms (cosine similarity of term frequency vectors).

```go
reranker := crossencoder.NewLocalRerankerClient(crossencoder.Config{})
defer reranker.Close()

results, err := reranker.Rank(ctx, query, passages)
```

### 3. Mock Reranker (`MockRerankerClient`)

Provides deterministic mock implementation for testing.

```go
reranker := crossencoder.NewMockRerankerClient(crossencoder.Config{})
defer reranker.Close()

results, err := reranker.Rank(ctx, query, passages)
```

## Factory Function

Use the factory function for provider-based client creation:

```go
client, err := crossencoder.NewClient(crossencoder.ClientConfig{
    Provider: crossencoder.ProviderLocal,
    Config:   crossencoder.DefaultConfig(crossencoder.ProviderLocal),
})
defer client.Close()
```

## Configuration

```go
config := crossencoder.Config{
    Model:          "gpt-4o-mini",     // Model name (OpenAI only)
    BatchSize:      10,                // Batch processing size  
    MaxConcurrency: 5,                 // Max concurrent requests (OpenAI only)
}
```

## Providers

- `ProviderOpenAI`: Uses OpenAI API for high-accuracy reranking
- `ProviderLocal`: Uses local text similarity algorithms
- `ProviderMock`: Deterministic mock for testing

## Usage Patterns

### Basic Ranking

```go
ctx := context.Background()
query := "machine learning algorithms"
passages := []string{
    "Machine learning algorithms are used for data analysis",
    "Cooking recipes and meal preparation", 
    "Deep learning neural networks and AI",
}

results, err := reranker.Rank(ctx, query, passages)
if err != nil {
    log.Fatal(err)
}

// Results are sorted by relevance score (descending)
for i, result := range results {
    fmt.Printf("%d. %s (score: %.3f)\n", i+1, result.Passage, result.Score)
}
```

### Integration with Search

Cross-encoders are typically used as rerankers in multi-stage retrieval:

1. **Initial Retrieval**: Fast methods (BM25, vector similarity)
2. **Reranking**: Cross-encoder for improved relevance

```go
// Get initial candidates from search
candidates := performInitialSearch(query)

// Extract passages for reranking
passages := make([]string, len(candidates))
for i, candidate := range candidates {
    passages[i] = candidate.Content
}

// Rerank using cross-encoder
reranked, err := reranker.Rank(ctx, query, passages)
if err != nil {
    return err
}

// Use reranked results
for _, result := range reranked[:topK] {
    // Process top results
}
```

## Performance Considerations

| Provider | Accuracy | Speed | External Deps | Use Case |
|----------|----------|-------|---------------|----------|
| OpenAI   | High     | Slow  | API calls     | Production with budget |
| Local    | Medium   | Fast  | None          | Local/offline systems |
| Mock     | Low      | Fastest| None         | Testing/development |

## Testing

Run tests with:

```bash
go test ./pkg/crossencoder/...
```

Run benchmarks:

```bash
go test ./pkg/crossencoder/... -bench=.
```

## Examples

See `example_test.go` for comprehensive usage examples:

```bash
go test ./pkg/crossencoder/... -run Example
```

## Integration

The crossencoder package integrates with the search functionality in go-graphiti to provide reranking capabilities for:

- Node search results
- Edge search results  
- Episode search results
- Community search results

When configured in search settings, cross-encoders automatically rerank initial search results to improve relevance ordering.