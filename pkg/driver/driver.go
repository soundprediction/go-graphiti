package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
)

// GraphProvider represents the type of graph database provider
type GraphProvider string

const (
	GraphProviderNeo4j    GraphProvider = "neo4j"
	GraphProviderMemgraph GraphProvider = "memgraph"
	GraphProviderFalkorDB GraphProvider = "falkordb"
	GraphProviderKuzu     GraphProvider = "kuzu"
	GraphProviderNeptune  GraphProvider = "neptune"
)

// GraphDriverSession defines the interface for database sessions (matching Python GraphDriverSession)
type GraphDriverSession interface {
	// Session management
	Enter(ctx context.Context) (GraphDriverSession, error)
	Exit(ctx context.Context, excType, excVal, excTb interface{}) error
	Close() error

	// Query execution
	Run(ctx context.Context, query interface{}, kwargs map[string]interface{}) error
	ExecuteWrite(ctx context.Context, fn func(context.Context, GraphDriverSession, ...interface{}) (interface{}, error), args ...interface{}) (interface{}, error)

	// Provider info
	Provider() GraphProvider
}

// GraphDriver defines the interface for graph database operations (matching Python GraphDriver)
type GraphDriver interface {
	// Core methods matching Python interface
	ExecuteQuery(cypherQuery string, kwargs map[string]interface{}) (interface{}, interface{}, interface{}, error)
	Session(database *string) GraphDriverSession
	Close() error
	DeleteAllIndexes(database string)
	Provider() GraphProvider
	GetAossClient() interface{}

	// Database-specific extensions (these can remain for compatibility)
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
	GetNodeNeighbors(ctx context.Context, nodeUUID, groupID string) ([]types.Neighbor, error)
	// GetBetweenNodes retrieves edges between two specific nodes using the proper query pattern
	GetBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string) ([]*types.Edge, error)

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
	GetExistingCommunity(ctx context.Context, entityUUID string) (*types.Node, error)
	FindModalCommunity(ctx context.Context, entityUUID string) (*types.Node, error)

	// Database maintenance
	CreateIndices(ctx context.Context) error
	GetStats(ctx context.Context, groupID string) (*GraphStats, error)
}

// GraphStats holds statistics about the graph.
type GraphStats struct {
	NodeCount      int64            `json:"node_count"`
	EdgeCount      int64            `json:"edge_count"`
	NodesByType    map[string]int64 `json:"nodes_by_type"`
	EdgesByType    map[string]int64 `json:"edges_by_type"`
	CommunityCount int64            `json:"community_count"`
	LastUpdated    time.Time        `json:"last_updated"`
}

// QueryOptions holds options for database queries.
type QueryOptions struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
	Filters   map[string]interface{}
}

// SearchOptions holds options for text-based search operations.
type SearchOptions struct {
	Limit       int              `json:"limit"`
	UseFullText bool             `json:"use_fulltext"`
	NodeTypes   []types.NodeType `json:"node_types,omitempty"`
	EdgeTypes   []types.EdgeType `json:"edge_types,omitempty"`
	TimeRange   *types.TimeRange `json:"time_range,omitempty"`
}

// VectorSearchOptions holds options for vector similarity search operations.
type VectorSearchOptions struct {
	Limit     int              `json:"limit"`
	MinScore  float64          `json:"min_score"`
	NodeTypes []types.NodeType `json:"node_types,omitempty"`
	EdgeTypes []types.EdgeType `json:"edge_types,omitempty"`
	TimeRange *types.TimeRange `json:"time_range,omitempty"`
}

// convertRecordToEdge converts a database record to an Edge object
func convertRecordToEdge(record map[string]interface{}) (*types.Edge, error) {
	edge := &types.Edge{}

	// Extract basic fields
	if uuid, ok := record["uuid"].(string); ok {
		edge.ID = uuid
	} else {
		return nil, fmt.Errorf("missing or invalid uuid field")
	}

	if name, ok := record["name"].(string); ok {
		edge.Name = name
	}

	if fact, ok := record["fact"].(string); ok {
		edge.Summary = fact
	}

	if groupID, ok := record["group_id"].(string); ok {
		edge.GroupID = groupID
	}

	// Extract source and target IDs
	if sourceID, ok := record["source_id"].(string); ok {
		edge.SourceID = sourceID
	}
	if targetID, ok := record["target_id"].(string); ok {
		edge.TargetID = targetID
	}

	// Extract timestamps
	if createdAt, ok := record["created_at"].(time.Time); ok {
		edge.CreatedAt = createdAt
	}
	if updatedAt, ok := record["updated_at"].(time.Time); ok {
		edge.UpdatedAt = updatedAt
	}
	if validFrom, ok := record["valid_from"].(time.Time); ok {
		edge.ValidFrom = validFrom
	}
	if validTo, ok := record["valid_to"].(time.Time); ok {
		edge.ValidTo = &validTo
	}

	// Set edge type - assume EntityEdge for relationships from RelatesToNode_
	edge.Type = types.EntityEdgeType

	// Extract source IDs if present
	if sourceIDs, ok := record["source_ids"].([]interface{}); ok {
		strSourceIDs := make([]string, len(sourceIDs))
		for i, id := range sourceIDs {
			if strID, ok := id.(string); ok {
				strSourceIDs[i] = strID
			}
		}
		edge.SourceIDs = strSourceIDs
	}

	return edge, nil
}
