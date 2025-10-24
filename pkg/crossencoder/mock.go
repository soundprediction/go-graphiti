package crossencoder

import (
	"context"
	"crypto/md5"
	"sort"
	"strings"
)

// MockRerankerClient provides a mock implementation for testing purposes
// It uses simple text similarity heuristics to rank passages
type MockRerankerClient struct {
	config Config
}

// NewMockRerankerClient creates a new mock reranker client
func NewMockRerankerClient(config Config) *MockRerankerClient {
	return &MockRerankerClient{
		config: config,
	}
}

// Rank ranks passages using simple text similarity heuristics
func (c *MockRerankerClient) Rank(ctx context.Context, query string, passages []string) ([]RankedPassage, error) {
	if len(passages) == 0 {
		return []RankedPassage{}, nil
	}

	var results []RankedPassage
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	for _, passage := range passages {
		score := c.calculateSimilarity(queryLower, queryWords, passage)
		results = append(results, RankedPassage{
			Passage: passage,
			Score:   score,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// calculateSimilarity calculates a simple similarity score between query and passage
func (c *MockRerankerClient) calculateSimilarity(queryLower string, queryWords []string, passage string) float64 {
	passageLower := strings.ToLower(passage)
	passageWords := strings.Fields(passageLower)

	if len(queryWords) == 0 || len(passageWords) == 0 {
		return 0.0
	}

	// Exact substring match gets high score
	if strings.Contains(passageLower, queryLower) {
		return 0.9
	}

	// Word overlap scoring
	queryWordSet := make(map[string]bool)
	for _, word := range queryWords {
		queryWordSet[word] = true
	}

	matchCount := 0
	for _, word := range passageWords {
		if queryWordSet[word] {
			matchCount++
		}
	}

	// Jaccard similarity with some adjustments
	overlap := float64(matchCount)
	union := float64(len(queryWords) + len(passageWords) - matchCount)

	if union == 0 {
		return 0.0
	}

	similarity := overlap / union

	// Add some randomness based on content hash for consistent but varied results
	hash := md5.Sum([]byte(passage))
	randomFactor := float64(hash[0]) / 255.0 * 0.1 // 0-0.1 range

	return similarity + randomFactor
}

// Close cleans up any resources used by the client
func (c *MockRerankerClient) Close() error {
	return nil
}
