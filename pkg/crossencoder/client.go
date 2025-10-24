package crossencoder

import "context"

// RankedPassage represents a passage with its relevance score
type RankedPassage struct {
	Passage string  `json:"passage"`
	Score   float64 `json:"score"`
}

// Client defines the interface for cross-encoder models used for ranking passages
// based on their relevance to a query. It allows for different implementations
// of cross-encoder models to be used interchangeably.
type Client interface {
	// Rank ranks the given passages based on their relevance to the query.
	// Returns a list of RankedPassage sorted in descending order of relevance.
	Rank(ctx context.Context, query string, passages []string) ([]RankedPassage, error)

	// Close cleans up any resources used by the client
	Close() error
}

// Config holds common configuration for cross-encoder clients
type Config struct {
	// Model specifies the model to use for ranking
	Model string `json:"model,omitempty"`

	// BatchSize specifies how many passages to process at once
	BatchSize int `json:"batch_size,omitempty"`

	// MaxConcurrency limits the number of concurrent requests
	MaxConcurrency int `json:"max_concurrency,omitempty"`
}
