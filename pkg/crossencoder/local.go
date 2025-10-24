package crossencoder

import (
	"context"
	"math"
	"sort"
	"strings"
)

// LocalRerankerClient provides a local implementation using simple text similarity
// This can be extended to work with local embedding models or transformers
type LocalRerankerClient struct {
	config Config
}

// NewLocalRerankerClient creates a new local reranker client
func NewLocalRerankerClient(config Config) *LocalRerankerClient {
	return &LocalRerankerClient{
		config: config,
	}
}

// Rank ranks passages using local text similarity algorithms
func (c *LocalRerankerClient) Rank(ctx context.Context, query string, passages []string) ([]RankedPassage, error) {
	if len(passages) == 0 {
		return []RankedPassage{}, nil
	}

	var results []RankedPassage

	for _, passage := range passages {
		score := c.calculateCosineTextSimilarity(query, passage)
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

// calculateCosineTextSimilarity calculates cosine similarity between query and passage
// using simple term frequency vectors
func (c *LocalRerankerClient) calculateCosineTextSimilarity(query, passage string) float64 {
	queryTokens := c.tokenize(query)
	passageTokens := c.tokenize(passage)

	if len(queryTokens) == 0 || len(passageTokens) == 0 {
		return 0.0
	}

	// Build term frequency maps
	queryTF := c.buildTermFrequency(queryTokens)
	passageTF := c.buildTermFrequency(passageTokens)

	// Get all unique terms
	allTerms := make(map[string]bool)
	for term := range queryTF {
		allTerms[term] = true
	}
	for term := range passageTF {
		allTerms[term] = true
	}

	// Calculate dot product and norms
	var dotProduct, queryNorm, passageNorm float64

	for term := range allTerms {
		queryVal := float64(queryTF[term])
		passageVal := float64(passageTF[term])

		dotProduct += queryVal * passageVal
		queryNorm += queryVal * queryVal
		passageNorm += passageVal * passageVal
	}

	queryNorm = math.Sqrt(queryNorm)
	passageNorm = math.Sqrt(passageNorm)

	if queryNorm == 0 || passageNorm == 0 {
		return 0.0
	}

	return dotProduct / (queryNorm * passageNorm)
}

// tokenize splits text into tokens, removing common stop words and normalizing
func (c *LocalRerankerClient) tokenize(text string) []string {
	// Simple tokenization - could be enhanced with proper NLP libraries
	text = strings.ToLower(text)

	// Remove punctuation and split on whitespace
	var tokens []string
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})

	// Filter out stop words and very short words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "should": true, "could": true,
		"can": true, "may": true, "might": true, "must": true, "shall": true, "this": true,
		"that": true, "these": true, "those": true, "i": true, "you": true, "he": true,
		"she": true, "it": true, "we": true, "they": true, "me": true, "him": true,
		"her": true, "us": true, "them": true, "my": true, "your": true, "his": true,
		"its": true, "our": true, "their": true,
	}

	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			tokens = append(tokens, word)
		}
	}

	return tokens
}

// buildTermFrequency builds a term frequency map from tokens
func (c *LocalRerankerClient) buildTermFrequency(tokens []string) map[string]int {
	tf := make(map[string]int)
	for _, token := range tokens {
		tf[token]++
	}
	return tf
}

// Close cleans up any resources used by the client
func (c *LocalRerankerClient) Close() error {
	return nil
}
