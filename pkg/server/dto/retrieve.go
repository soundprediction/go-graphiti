package dto

import "time"

// SearchQuery represents a search query request
type SearchQuery struct {
	Query    string   `json:"query" binding:"required"`
	GroupIDs []string `json:"group_ids,omitempty"`
	MaxFacts int      `json:"max_facts,omitempty"`
}

// SearchResults represents search results
type SearchResults struct {
	Facts []FactResult `json:"facts"`
	Total int          `json:"total"`
}

// GetMemoryRequest represents a request to get memory
type GetMemoryRequest struct {
	Messages []Message `json:"messages" binding:"required"`
	GroupIDs []string  `json:"group_ids,omitempty"`
	MaxFacts int       `json:"max_facts,omitempty"`
}

// GetMemoryResponse represents a memory response
type GetMemoryResponse struct {
	Facts []FactResult `json:"facts"`
	Total int          `json:"total"`
}

// GetEpisodesRequest represents a request to get episodes
type GetEpisodesRequest struct {
	GroupID string `json:"group_id" binding:"required"`
	LastN   int    `json:"last_n,omitempty"`
}

// Episode represents an episode in the knowledge graph
type Episode struct {
	UUID      string    `json:"uuid"`
	GroupID   string    `json:"group_id"`
	Content   string    `json:"content"`
	Source    string    `json:"source,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// GetEpisodesResponse represents episodes response
type GetEpisodesResponse struct {
	Episodes []Episode `json:"episodes"`
	Total    int       `json:"total"`
}