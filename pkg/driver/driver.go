package driver

import (
	"context"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
)

// GraphDriver defines the interface for graph database operations.
// It provides methods for storing and retrieving nodes and edges
// from a graph database backend.
type GraphDriver interface {
	// Node operations
	GetNode(ctx context.Context, nodeID, groupID string) (*types.Node, error)
	UpsertNode(ctx context.Context, node *types.Node) error
	DeleteNode(ctx context.Context, nodeID, groupID string) error
	GetNodes(ctx context.Context, nodeIDs []string, groupID string) ([]*types.Node, error)

	// Edge operations  
	GetEdge(ctx context.Context, edgeID, groupID string) (*types.Edge, error)
	UpsertEdge(ctx context.Context, edge *types.Edge) error
	DeleteEdge(ctx context.Context, edgeID, groupID string) error
	GetEdges(ctx context.Context, edgeIDs []string, groupID string) ([]*types.Edge, error)

	// Graph traversal operations
	GetNeighbors(ctx context.Context, nodeID, groupID string, maxDistance int) ([]*types.Node, error)
	GetRelatedNodes(ctx context.Context, nodeID, groupID string, edgeTypes []types.EdgeType) ([]*types.Node, error)

	// Search operations
	SearchNodesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Node, error)
	SearchEdgesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Edge, error)
	SearchNodes(ctx context.Context, query, groupID string, options *SearchOptions) ([]*types.Node, error)
	SearchEdges(ctx context.Context, query, groupID string, options *SearchOptions) ([]*types.Edge, error)
	SearchNodesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Node, error)
	SearchEdgesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Edge, error)

	// Bulk operations
	UpsertNodes(ctx context.Context, nodes []*types.Node) error
	UpsertEdges(ctx context.Context, edges []*types.Edge) error

	// Temporal operations
	GetNodesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Node, error)
	GetEdgesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Edge, error)

	// Community operations
	GetCommunities(ctx context.Context, groupID string, level int) ([]*types.Node, error)
	BuildCommunities(ctx context.Context, groupID string) error

	// Database maintenance
	CreateIndices(ctx context.Context) error
	GetStats(ctx context.Context, groupID string) (*GraphStats, error)

	// Connection management
	Close(ctx context.Context) error
}

// GraphStats holds statistics about the graph.
type GraphStats struct {
	NodeCount            int64            `json:"node_count"`
	EdgeCount            int64            `json:"edge_count"`
	NodesByType          map[string]int64 `json:"nodes_by_type"`
	EdgesByType          map[string]int64 `json:"edges_by_type"`
	CommunityCount       int64            `json:"community_count"`
	LastUpdated          time.Time        `json:"last_updated"`
}

// QueryOptions holds options for database queries.
type QueryOptions struct {
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
	Filters    map[string]interface{}
}

// SearchOptions holds options for text-based search operations.
type SearchOptions struct {
	Limit       int                  `json:"limit"`
	UseFullText bool                 `json:"use_fulltext"`
	NodeTypes   []types.NodeType     `json:"node_types,omitempty"`
	EdgeTypes   []types.EdgeType     `json:"edge_types,omitempty"`
	TimeRange   *types.TimeRange     `json:"time_range,omitempty"`
}

// VectorSearchOptions holds options for vector similarity search operations.
type VectorSearchOptions struct {
	Limit     int                  `json:"limit"`
	MinScore  float64              `json:"min_score"`
	NodeTypes []types.NodeType     `json:"node_types,omitempty"`
	EdgeTypes []types.EdgeType     `json:"edge_types,omitempty"`
	TimeRange *types.TimeRange     `json:"time_range,omitempty"`
}