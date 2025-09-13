package dto

import "time"

// AddMessagesRequest represents a request to add messages to the knowledge graph
type AddMessagesRequest struct {
	GroupID   string    `json:"group_id" binding:"required"`
	Messages  []Message `json:"messages" binding:"required"`
	Reference *time.Time `json:"reference,omitempty"`
}

// AddEntityNodeRequest represents a request to add an entity node
type AddEntityNodeRequest struct {
	GroupID    string                 `json:"group_id" binding:"required"`
	Name       string                 `json:"name" binding:"required"`
	EntityType string                 `json:"entity_type,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// ClearDataRequest represents a request to clear graph data
type ClearDataRequest struct {
	GroupIDs []string `json:"group_ids,omitempty"`
}

// IngestResponse represents a response from ingest operations
type IngestResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	ProcessID string `json:"process_id,omitempty"`
}