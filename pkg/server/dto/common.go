package dto

import "time"

// Message represents a chat message
type Message struct {
	Role      string     `json:"role" binding:"required"`
	Content   string     `json:"content" binding:"required"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

// Result represents a generic API result
type Result struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// FactResult represents a fact result from the knowledge graph
type FactResult struct {
	UUID         string     `json:"uuid"`
	Fact         string     `json:"fact"`
	SourceName   string     `json:"source_name"`
	TargetName   string     `json:"target_name"`
	RelationType string     `json:"relation_type"`
	ValidAt      *time.Time `json:"valid_at,omitempty"`
	InvalidAt    *time.Time `json:"invalid_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	Score        *float64   `json:"score,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}
