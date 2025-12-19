package crossencoder

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/soundprediction/go-predicato/pkg/embedder"
)

// EmbeddingRerankerClient implements cross-encoder functionality using embeddings
// This reranker computes cosine similarity between query and passage embeddings
// to generate relevance scores. While not a true cross-encoder (which processes
// query-document pairs together), it provides good reranking performance using
// bi-encoder embeddings.
type EmbeddingRerankerClient struct {
	embedder embedder.Client
	config   Config
}

// EmbeddingConfig holds embedding-specific configuration
type EmbeddingConfig struct {
	Config
	// SimilarityThreshold is the minimum cosine similarity to consider relevant
	SimilarityThreshold float64 `json:"similarity_threshold,omitempty"`
	// NormalizeScores whether to normalize scores to 0-1 range
	NormalizeScores bool `json:"normalize_scores,omitempty"`
}

// NewEmbeddingRerankerClient creates a new embedding-based reranker client
func NewEmbeddingRerankerClient(embedderClient embedder.Client, config EmbeddingConfig) *EmbeddingRerankerClient {
	if config.SimilarityThreshold == 0 {
		config.SimilarityThreshold = -1.0 // Accept all similarities
	}

	return &EmbeddingRerankerClient{
		embedder: embedderClient,
		config:   config.Config,
	}
}

// Rank ranks the given passages based on their relevance to the query using embeddings
func (c *EmbeddingRerankerClient) Rank(ctx context.Context, query string, passages []string) ([]RankedPassage, error) {
	if len(passages) == 0 {
		return []RankedPassage{}, nil
	}

	if c.embedder == nil {
		return nil, fmt.Errorf("embedder client is nil")
	}

	// Get query embedding
	queryEmbedding, err := c.embedder.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("query embedding is empty")
	}

	// Get embeddings for all passages
	passageEmbeddings := make([][]float32, len(passages))
	for i, passage := range passages {
		embedding, err := c.embedder.EmbedSingle(ctx, passage)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding for passage %d: %w", i, err)
		}
		passageEmbeddings[i] = embedding
	}

	// Calculate similarities and create results
	results := make([]RankedPassage, 0, len(passages))
	similarities := make([]float64, len(passages))

	for i, passage := range passages {
		similarity := cosineSimilarity(queryEmbedding, passageEmbeddings[i])
		similarities[i] = similarity

		// Apply threshold if configured
		if similarity >= 0 { // Always include for now, could add threshold logic here
			results = append(results, RankedPassage{
				Passage: passage,
				Score:   similarity,
			})
		}
	}

	// Normalize scores if requested
	if len(results) > 0 {
		// Find min and max for normalization
		minScore := results[0].Score
		maxScore := results[0].Score
		for _, result := range results[1:] {
			if result.Score < minScore {
				minScore = result.Score
			}
			if result.Score > maxScore {
				maxScore = result.Score
			}
		}

		// Normalize to 0-1 range if there's variance
		if maxScore > minScore {
			scoreRange := maxScore - minScore
			for i := range results {
				results[i].Score = (results[i].Score - minScore) / scoreRange
			}
		} else {
			// If all scores are the same, set them all to 1.0
			for i := range results {
				results[i].Score = 1.0
			}
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// Close cleans up any resources used by the client
func (c *EmbeddingRerankerClient) Close() error {
	if c.embedder != nil {
		return c.embedder.Close()
	}
	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
