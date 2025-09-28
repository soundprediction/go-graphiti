package types

import (
	"time"
)

// Node represents a node in the knowledge graph.
type Node struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      NodeType  `json:"type"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Entity-specific fields
	EntityType string `json:"entity_type,omitempty"`
	Summary    string `json:"summary,omitempty"`

	// Episode-specific fields
	EpisodeType EpisodeType `json:"episode_type,omitempty"`
	Content     string      `json:"content,omitempty"`
	Reference   time.Time   `json:"reference,omitempty"`
	EntityEdges []string    `json:"entity_edges,omitempty"`

	// Community-specific fields
	Level int `json:"level,omitempty"`

	// Common fields
	Embedding     []float32              `json:"embedding,omitempty"`
	NameEmbedding []float32              `json:"name_embedding,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`

	// Temporal fields
	ValidFrom time.Time  `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`

	// Source tracking
	SourceIDs []string `json:"source_ids,omitempty"`
}

// Edge is an alias for EntityEdge to maintain backward compatibility
// Use EntityEdge, EpisodicEdge, or CommunityEdge directly for type-safe operations
type Edge = EntityEdge

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

// EdgeType and related constants are now defined in edge.go

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
	ID               string
	Name             string
	Content          string
	Reference        time.Time
	CreatedAt        time.Time
	GroupID          string
	Metadata         map[string]interface{}
	ContentEmbedding []float32
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

// AddEpisodeResults represents the result of adding a single episode to the knowledge graph.
type AddEpisodeResults struct {
	// Episode is the episodic node that was created.
	Episode *Node `json:"episode"`
	// EpisodicEdges are the edges connecting the episode to entities.
	EpisodicEdges []*Edge `json:"episodic_edges"`
	// Nodes are the entity nodes that were extracted or updated.
	Nodes []*Node `json:"nodes"`
	// Edges are the entity relationships that were extracted or updated.
	Edges []*Edge `json:"edges"`
	// Communities are the community nodes that were created or updated.
	Communities []*Node `json:"communities"`
	// CommunityEdges are the edges connecting communities to entities.
	CommunityEdges []*Edge `json:"community_edges"`
}

// AddBulkEpisodeResults represents the result of adding multiple episodes to the knowledge graph.
type AddBulkEpisodeResults struct {
	// Episodes are the episodic nodes that were created.
	Episodes []*Node `json:"episodes"`
	// EpisodicEdges are the edges connecting episodes to entities.
	EpisodicEdges []*Edge `json:"episodic_edges"`
	// Nodes are the entity nodes that were extracted or updated.
	Nodes []*Node `json:"nodes"`
	// Edges are the entity relationships that were extracted or updated.
	Edges []*Edge `json:"edges"`
	// Communities are the community nodes that were created or updated.
	Communities []*Node `json:"communities"`
	// CommunityEdges are the edges connecting communities to entities.
	CommunityEdges []*Edge `json:"community_edges"`
}

// AddTripletResults represents the result of adding a triplet (subject-predicate-object) to the knowledge graph.
type AddTripletResults struct {
	// Nodes are the entity nodes that were created or updated (subject and object).
	Nodes []*Node `json:"nodes"`
	// Edges are the relationship edges that were created (predicate).
	Edges []*Edge `json:"edges"`
}

// EpisodeProcessingResult represents the result of processing a single episode.
// This is used internally during episode processing.
type EpisodeProcessingResult struct {
	// Episode is the processed episode node.
	Episode *Node `json:"episode"`
	// ExtractedEntities are the entities found in the episode.
	ExtractedEntities []*Node `json:"extracted_entities"`
	// ExtractedRelationships are the relationships found in the episode.
	ExtractedRelationships []*Edge `json:"extracted_relationships"`
	// EpisodicEdges connect the episode to the extracted entities.
	EpisodicEdges []*Edge `json:"episodic_edges"`
	// ProcessingTime is the time taken to process this episode.
	ProcessingTime time.Duration `json:"processing_time"`
	// Errors encountered during processing.
	Errors []string `json:"errors,omitempty"`
}

// BulkEpisodeResults represents the result of bulk episode processing.
type BulkEpisodeResults struct {
	// ProcessedEpisodes contains results for each processed episode.
	ProcessedEpisodes []*EpisodeProcessingResult `json:"processed_episodes"`
	// TotalProcessingTime is the total time for all episodes.
	TotalProcessingTime time.Duration `json:"total_processing_time"`
	// SuccessCount is the number of successfully processed episodes.
	SuccessCount int `json:"success_count"`
	// ErrorCount is the number of episodes that failed processing.
	ErrorCount int `json:"error_count"`
}

// TripletResults represents the result of triplet operations.
type TripletResults struct {
	// SubjectNode is the subject entity node.
	SubjectNode *Node `json:"subject_node"`
	// ObjectNode is the object entity node.
	ObjectNode *Node `json:"object_node"`
	// PredicateEdge is the relationship edge between subject and object.
	PredicateEdge *Edge `json:"predicate_edge"`
	// Created indicates if new nodes/edges were created (vs updated).
	Created bool `json:"created"`
}
