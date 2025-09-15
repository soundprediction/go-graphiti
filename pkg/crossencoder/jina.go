package crossencoder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// JinaRerankerClient implements cross-encoder functionality using Jina's reranking API
// This is compatible with VLLM's Jina API implementation for cross-encoder models
type JinaRerankerClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	config     Config
}

// JinaRerankRequest represents the request structure for Jina's rerank API
type JinaRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopK      *int     `json:"top_k,omitempty"`
}

// JinaRerankResponse represents the response structure from Jina's rerank API
type JinaRerankResponse struct {
	Results []JinaRankedResult `json:"results"`
	Model   string             `json:"model"`
	Usage   *JinaUsage         `json:"usage,omitempty"`
}

// JinaRankedResult represents a single ranking result
type JinaRankedResult struct {
	Index          int     `json:"index"`
	Document       string  `json:"document"`
	RelevanceScore float64 `json:"relevance_score"`
}

// JinaUsage represents token usage information
type JinaUsage struct {
	TotalTokens  int `json:"total_tokens"`
	PromptTokens int `json:"prompt_tokens"`
}

// JinaConfig holds Jina-specific configuration
type JinaConfig struct {
	Config
	BaseURL string `json:"base_url,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
	TopK    *int   `json:"top_k,omitempty"`
}

// NewJinaRerankerClient creates a new Jina-based reranker client
func NewJinaRerankerClient(config JinaConfig) *JinaRerankerClient {
	if config.Model == "" {
		config.Model = "jina-reranker-v1-base-en"
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.jina.ai/v1"
	}

	return &JinaRerankerClient{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config.Config,
	}
}

// Rank ranks the given passages based on their relevance to the query using Jina's API
func (c *JinaRerankerClient) Rank(ctx context.Context, query string, passages []string) ([]RankedPassage, error) {
	if len(passages) == 0 {
		return []RankedPassage{}, nil
	}

	// Prepare the request
	request := JinaRerankRequest{
		Model:     c.config.Model,
		Query:     query,
		Documents: passages,
	}

	// Marshal the request
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/rerank", bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBytes))
	}

	// Parse the response
	var jinaResponse JinaRerankResponse
	if err := json.Unmarshal(responseBytes, &jinaResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Convert to RankedPassage format
	results := make([]RankedPassage, len(jinaResponse.Results))
	for i, result := range jinaResponse.Results {
		results[i] = RankedPassage{
			Passage: result.Document,
			Score:   result.RelevanceScore,
		}
	}

	// Sort by score (descending) - Jina should return them sorted, but ensure consistency
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// Close cleans up any resources used by the client
func (c *JinaRerankerClient) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}