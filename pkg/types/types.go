package types

import (
	"time"
)

// Node represents a node in the knowledge graph.
type Node struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      NodeType               `json:"type"`
	GroupID   string                 `json:"group_id"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	
	// Entity-specific fields
	EntityType   string                 `json:"entity_type,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	
	// Episode-specific fields
	EpisodeType  EpisodeType            `json:"episode_type,omitempty"`
	Content      string                 `json:"content,omitempty"`
	Reference    time.Time              `json:"reference,omitempty"`
	
	// Community-specific fields
	Level        int                    `json:"level,omitempty"`
	
	// Common fields
	Embedding    []float32              `json:"embedding,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	
	// Temporal fields
	ValidFrom    time.Time              `json:"valid_from"`
	ValidTo      *time.Time             `json:"valid_to,omitempty"`
	
	// Source tracking
	SourceIDs    []string               `json:"source_ids,omitempty"`
}

// Edge represents a relationship between nodes in the knowledge graph.
type Edge struct {
	ID           string                 `json:"id"`
	Type         EdgeType               `json:"type"`
	SourceID     string                 `json:"source_id"`
	TargetID     string                 `json:"target_id"`
	GroupID      string                 `json:"group_id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	
	// Relationship details
	Name         string                 `json:"name,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	Strength     float64                `json:"strength,omitempty"`
	
	// Embedding for semantic search
	Embedding    []float32              `json:"embedding,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	
	// Temporal fields
	ValidFrom    time.Time              `json:"valid_from"`
	ValidTo      *time.Time             `json:"valid_to,omitempty"`
	
	// Source tracking
	SourceIDs    []string               `json:"source_ids,omitempty"`
}

// NodeType represents the type of a node.
type NodeType string

const (
	// EntityNodeType represents entities extracted from content.
	EntityNodeType NodeType = "entity"
	// EpisodicNodeType represents episodic memories or events.
	EpisodicNodeType NodeType = "episodic"
	// CommunityNodeType represents communities of related entities.
	CommunityNodeType NodeType = "community"
)

// EdgeType represents the type of an edge.
type EdgeType string

const (
	// EntityEdgeType represents relationships between entities.
	EntityEdgeType EdgeType = "entity"
	// EpisodicEdgeType represents episodic relationships.
	EpisodicEdgeType EdgeType = "episodic" 
	// CommunityEdgeType represents community relationships.
	CommunityEdgeType EdgeType = "community"
)

// EpisodeType represents the type of an episode.
type EpisodeType string

const (
	// ConversationEpisodeType for conversational data.
	ConversationEpisodeType EpisodeType = "conversation"
	// DocumentEpisodeType for document content.
	DocumentEpisodeType EpisodeType = "document"
	// EventEpisodeType for events or actions.
	EventEpisodeType EpisodeType = "event"
)

// Episode represents a temporal data unit to be processed.
type Episode struct {
	ID        string
	Name      string
	Content   string
	Reference time.Time
	CreatedAt time.Time
	GroupID   string
	Metadata  map[string]interface{}
}

// SearchConfig holds configuration for search operations.
type SearchConfig struct {
	// Limit is the maximum number of results to return.
	Limit int
	// CenterNodeDistance is the maximum distance from center nodes.
	CenterNodeDistance int
	// MinScore is the minimum relevance score for results.
	MinScore float64
	// IncludeEdges determines if edges should be included in results.
	IncludeEdges bool
	// Rerank determines if results should be reranked.
	Rerank bool
	// Filters for constraining search results.
	Filters *SearchFilters
	// NodeConfig holds configuration for node search.
	NodeConfig *NodeSearchConfig
	// EdgeConfig holds configuration for edge search.
	EdgeConfig *EdgeSearchConfig
}

// NodeSearchConfig holds configuration for node search operations.
type NodeSearchConfig struct {
	// SearchMethods defines which search methods to use.
	SearchMethods []string
	// Reranker defines which reranking method to use.
	Reranker string
	// MinScore is the minimum score for results.
	MinScore float64
}

// EdgeSearchConfig holds configuration for edge search operations.
type EdgeSearchConfig struct {
	// SearchMethods defines which search methods to use.
	SearchMethods []string
	// Reranker defines which reranking method to use.
	Reranker string
	// MinScore is the minimum score for results.
	MinScore float64
}

// SearchFilters holds filters for search operations.
type SearchFilters struct {
	// GroupIDs to include in search.
	GroupIDs []string
	// NodeTypes to include.
	NodeTypes []NodeType
	// EdgeTypes to include.
	EdgeTypes []EdgeType
	// EntityTypes to include.
	EntityTypes []string
	// TimeRange for temporal filtering.
	TimeRange *TimeRange
}

// TimeRange represents a time range for filtering.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// SearchResults holds the results of a search operation.
type SearchResults struct {
	// Nodes found in the search.
	Nodes []*Node
	// Edges found in the search.
	Edges []*Edge
	// Query used for the search.
	Query string
	// Total number of results found (before limit).
	Total int
}

// ExtractedEntity represents an entity extracted from content.
type ExtractedEntity struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Summary  string            `json:"summary"`
	Metadata map[string]string `json:"metadata"`
}

// ExtractedRelationship represents a relationship extracted from content.
type ExtractedRelationship struct {
	SourceEntity string            `json:"source_entity"`
	TargetEntity string            `json:"target_entity"`
	Name         string            `json:"name"`
	Summary      string            `json:"summary"`
	Strength     float64           `json:"strength"`
	Metadata     map[string]string `json:"metadata"`
}