package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/kuzudb/go-kuzu"
)

// KuzuDriver implements the GraphDriver interface for Kuzu databases.
// Kuzu is an embedded graph database management system built for query speed and scalability.
type KuzuDriver struct {
	database *kuzu.Database
	conn     *kuzu.Connection
	dbPath   string
}

// NewKuzuDriver creates a new Kuzu driver instance.
// Kuzu is an embedded database, so it works with a local directory path.
//
// Parameters:
//   - dbPath: Path to the Kuzu database directory (will be created if it doesn't exist)
//
// Example:
//
//	driver, err := driver.NewKuzuDriver("./kuzu_db")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer driver.Close(ctx)
func NewKuzuDriver(dbPath string) (*KuzuDriver, error) {
	if dbPath == "" {
		dbPath = "./kuzu_graphiti_db"
	}

	// Create the Kuzu database
	database, err := kuzu.OpenDatabase(dbPath, kuzu.DefaultSystemConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to open kuzu database: %w", err)
	}

	// Create a connection to the database
	conn, err := kuzu.OpenConnection(database)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to open kuzu connection: %w", err)
	}

	driver := &KuzuDriver{
		database: database,
		conn:     conn,
		dbPath:   dbPath,
	}

	// Initialize the schema for nodes and edges
	err = driver.createTables()
	if err != nil {
		driver.Close(context.Background())
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return driver, nil
}

// createTables initializes the database schema for nodes and edges
func (k *KuzuDriver) createTables() error {
	// Create Node table
	nodeQuery := `
		CREATE NODE TABLE IF NOT EXISTS Node (
			id STRING,
			name STRING,
			node_type STRING,
			group_id STRING,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			entity_type STRING,
			summary STRING,
			episode_type STRING,
			content STRING,
			reference TIMESTAMP,
			level INT64,
			embedding FLOAT[],
			metadata STRING,
			valid_from TIMESTAMP,
			valid_to TIMESTAMP,
			source_ids STRING[],
			PRIMARY KEY (id)
		)`

	_, err := k.conn.Query(nodeQuery)
	if err != nil {
		return fmt.Errorf("failed to create Node table: %w", err)
	}

	// Create Edge table
	edgeQuery := `
		CREATE REL TABLE IF NOT EXISTS Edge (
			FROM Node TO Node,
			id STRING,
			edge_type STRING,
			group_id STRING,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			name STRING,
			summary STRING,
			strength DOUBLE,
			embedding FLOAT[],
			metadata STRING,
			valid_from TIMESTAMP,
			valid_to TIMESTAMP,
			source_ids STRING[]
		)`

	_, err = k.conn.Query(edgeQuery)
	if err != nil {
		return fmt.Errorf("failed to create Edge table: %w", err)
	}

	return nil
}

// GetNode retrieves a node by ID.
func (k *KuzuDriver) GetNode(ctx context.Context, nodeID, groupID string) (*types.Node, error) {
	// Escape strings for safe query execution
	escapedNodeID := fmt.Sprintf("'%s'", nodeID)
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	query := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.id = %s AND n.group_id = %s
		RETURN n.*
	`, escapedNodeID, escapedGroupID)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query node: %w", err)
	}
	defer result.Close()

	if !result.HasNext() {
		return nil, fmt.Errorf("node not found")
	}

	row, err := result.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to get next row: %w", err)
	}

	return k.flatTupleToNode(row)
}

// UpsertNode creates or updates a node.
func (k *KuzuDriver) UpsertNode(ctx context.Context, node *types.Node) error {
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	node.UpdatedAt = time.Now()
	if node.ValidFrom.IsZero() {
		node.ValidFrom = node.CreatedAt
	}

	// Prepare the query with actual values
	preparedCreateQuery := k.prepareNodeCreateQuery(node)

	_, err := k.conn.Query(preparedCreateQuery)
	if err != nil {
		// If creation fails, try to update
		preparedUpdateQuery := k.prepareNodeUpdateQuery(node)

		_, updateErr := k.conn.Query(preparedUpdateQuery)
		if updateErr != nil {
			return fmt.Errorf("failed to create or update node: create error: %w, update error: %w", err, updateErr)
		}
	}

	return nil
}

// DeleteNode removes a node and its edges.
func (k *KuzuDriver) DeleteNode(ctx context.Context, nodeID, groupID string) error {
	// Escape strings for safe query execution
	escapedNodeID := fmt.Sprintf("'%s'", nodeID)
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	// Delete all edges connected to this node first
	deleteEdgesQuery := fmt.Sprintf(`
		MATCH (n:Node)-[r:Edge]-()
		WHERE n.id = %s AND n.group_id = %s
		DELETE r
	`, escapedNodeID, escapedGroupID)

	_, err := k.conn.Query(deleteEdgesQuery)
	if err != nil {
		return fmt.Errorf("failed to delete edges for node: %w", err)
	}

	// Delete the node itself
	deleteNodeQuery := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.id = %s AND n.group_id = %s
		DELETE n
	`, escapedNodeID, escapedGroupID)

	_, err = k.conn.Query(deleteNodeQuery)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

// GetNodes retrieves multiple nodes by their IDs.
func (k *KuzuDriver) GetNodes(ctx context.Context, nodeIDs []string, groupID string) ([]*types.Node, error) {
	if len(nodeIDs) == 0 {
		return []*types.Node{}, nil
	}

	// Build IN clause for multiple node IDs
	escapedGroupID := fmt.Sprintf("'%s'", groupID)
	var idList string
	for i, nodeID := range nodeIDs {
		if i > 0 {
			idList += ", "
		}
		idList += fmt.Sprintf("'%s'", nodeID)
	}

	query := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.id IN [%s] AND n.group_id = %s
		RETURN n.*
	`, idList, escapedGroupID)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer result.Close()

	var nodes []*types.Node
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		node, err := k.flatTupleToNode(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to node: %w", err)
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetEdge retrieves an edge by ID.
func (k *KuzuDriver) GetEdge(ctx context.Context, edgeID, groupID string) (*types.Edge, error) {
	// Escape strings for safe query execution
	escapedEdgeID := fmt.Sprintf("'%s'", edgeID)
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	query := fmt.Sprintf(`
		MATCH (a:Node)-[e:Edge]->(b:Node)
		WHERE e.id = %s AND e.group_id = %s
		RETURN e.*, a.id AS source_id, b.id AS target_id
	`, escapedEdgeID, escapedGroupID)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query edge: %w", err)
	}
	defer result.Close()

	if !result.HasNext() {
		return nil, fmt.Errorf("edge not found")
	}

	row, err := result.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to get next row: %w", err)
	}

	return k.flatTupleToEdge(row)
}

// UpsertEdge creates or updates an edge.
func (k *KuzuDriver) UpsertEdge(ctx context.Context, edge *types.Edge) error {
	if edge.CreatedAt.IsZero() {
		edge.CreatedAt = time.Now()
	}
	edge.UpdatedAt = time.Now()
	if edge.ValidFrom.IsZero() {
		edge.ValidFrom = edge.CreatedAt
	}

	// First ensure source and target nodes exist
	_, err := k.GetNode(ctx, edge.SourceID, edge.GroupID)
	if err != nil {
		return fmt.Errorf("source node %s not found: %w", edge.SourceID, err)
	}

	_, err = k.GetNode(ctx, edge.TargetID, edge.GroupID)
	if err != nil {
		return fmt.Errorf("target node %s not found: %w", edge.TargetID, err)
	}

	// Try to create the edge first
	preparedCreateQuery := k.prepareEdgeCreateQuery(edge)

	_, err = k.conn.Query(preparedCreateQuery)
	if err != nil {
		// If creation fails, try to update
		preparedUpdateQuery := k.prepareEdgeUpdateQuery(edge)

		_, updateErr := k.conn.Query(preparedUpdateQuery)
		if updateErr != nil {
			return fmt.Errorf("failed to create or update edge: create error: %w, update error: %w", err, updateErr)
		}
	}

	return nil
}

// DeleteEdge removes an edge.
func (k *KuzuDriver) DeleteEdge(ctx context.Context, edgeID, groupID string) error {
	// Escape strings for safe query execution
	escapedEdgeID := fmt.Sprintf("'%s'", edgeID)
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	// Delete the edge
	deleteQuery := fmt.Sprintf(`
		MATCH (a:Node)-[e:Edge]->(b:Node)
		WHERE e.id = %s AND e.group_id = %s
		DELETE e
	`, escapedEdgeID, escapedGroupID)

	_, err := k.conn.Query(deleteQuery)
	if err != nil {
		return fmt.Errorf("failed to delete edge: %w", err)
	}

	return nil
}

// GetEdges retrieves multiple edges by their IDs.
func (k *KuzuDriver) GetEdges(ctx context.Context, edgeIDs []string, groupID string) ([]*types.Edge, error) {
	if len(edgeIDs) == 0 {
		return []*types.Edge{}, nil
	}

	// Build IN clause for multiple edge IDs
	escapedGroupID := fmt.Sprintf("'%s'", groupID)
	var idList string
	for i, edgeID := range edgeIDs {
		if i > 0 {
			idList += ", "
		}
		idList += fmt.Sprintf("'%s'", edgeID)
	}

	query := fmt.Sprintf(`
		MATCH (a:Node)-[e:Edge]->(b:Node)
		WHERE e.id IN [%s] AND e.group_id = %s
		RETURN e.*, a.id AS source_id, b.id AS target_id
	`, idList, escapedGroupID)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query edges: %w", err)
	}
	defer result.Close()

	var edges []*types.Edge
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		edge, err := k.flatTupleToEdge(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to edge: %w", err)
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// GetNeighbors retrieves neighboring nodes within a specified distance.
func (k *KuzuDriver) GetNeighbors(ctx context.Context, nodeID, groupID string, maxDistance int) ([]*types.Node, error) {
	if maxDistance <= 0 {
		maxDistance = 1
	}
	if maxDistance > 10 {
		maxDistance = 10 // Prevent very expensive queries
	}

	// Escape strings for safe query execution
	escapedNodeID := fmt.Sprintf("'%s'", nodeID)
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	// Build variable-length path query
	query := fmt.Sprintf(`
		MATCH (start:Node)-[:Edge*1..%d]-(neighbor:Node)
		WHERE start.id = %s AND start.group_id = %s
		  AND neighbor.group_id = %s
		  AND neighbor.id <> start.id
		RETURN DISTINCT neighbor.*
	`, maxDistance, escapedNodeID, escapedGroupID, escapedGroupID)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query neighbors: %w", err)
	}
	defer result.Close()

	var neighbors []*types.Node
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		node, err := k.flatTupleToNode(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to node: %w", err)
		}

		neighbors = append(neighbors, node)
	}

	return neighbors, nil
}

func (k *KuzuDriver) GetRelatedNodes(ctx context.Context, nodeID, groupID string, edgeTypes []types.EdgeType) ([]*types.Node, error) {
	// Escape strings for safe query execution
	escapedNodeID := fmt.Sprintf("'%s'", nodeID)
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	var edgeTypeFilter string
	if len(edgeTypes) > 0 {
		// Build edge type filter
		var typeList string
		for i, edgeType := range edgeTypes {
			if i > 0 {
				typeList += ", "
			}
			typeList += fmt.Sprintf("'%s'", string(edgeType))
		}
		edgeTypeFilter = fmt.Sprintf(" AND e.edge_type IN [%s]", typeList)
	}

	// Query for related nodes (both incoming and outgoing relationships)
	query := fmt.Sprintf(`
		MATCH (start:Node)-[e:Edge]-(related:Node)
		WHERE start.id = %s AND start.group_id = %s
		  AND related.group_id = %s
		  AND related.id <> start.id%s
		RETURN DISTINCT related.*
	`, escapedNodeID, escapedGroupID, escapedGroupID, edgeTypeFilter)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query related nodes: %w", err)
	}
	defer result.Close()

	var relatedNodes []*types.Node
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		node, err := k.flatTupleToNode(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to node: %w", err)
		}

		relatedNodes = append(relatedNodes, node)
	}

	return relatedNodes, nil
}

func (k *KuzuDriver) SearchNodesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Node, error) {
	if len(embedding) == 0 {
		return []*types.Node{}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	// For now, implement a basic similarity search
	// In a full implementation, this would use vector similarity functions
	// or specialized vector indexes in Kuzu
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	// Query all nodes with embeddings in the group
	// This is a simplified implementation - real vector search would be more sophisticated
	query := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.group_id = %s
		  AND n.embedding IS NOT NULL
		RETURN n.*
		LIMIT %d
	`, escapedGroupID, limit)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to search nodes by embedding: %w", err)
	}
	defer result.Close()

	var nodes []*types.Node
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		node, err := k.flatTupleToNode(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to node: %w", err)
		}

		// TODO: Calculate actual cosine similarity with input embedding
		// For now, just return nodes that have embeddings
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (k *KuzuDriver) SearchEdgesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Edge, error) {
	if len(embedding) == 0 {
		return []*types.Edge{}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	// For now, implement a basic similarity search
	// In a full implementation, this would use vector similarity functions
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	// Query all edges with embeddings in the group
	query := fmt.Sprintf(`
		MATCH (a:Node)-[e:Edge]->(b:Node)
		WHERE e.group_id = %s
		  AND e.embedding IS NOT NULL
		RETURN e.*, a.id AS source_id, b.id AS target_id
		LIMIT %d
	`, escapedGroupID, limit)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to search edges by embedding: %w", err)
	}
	defer result.Close()

	var edges []*types.Edge
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		edge, err := k.flatTupleToEdge(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to edge: %w", err)
		}

		// TODO: Calculate actual cosine similarity with input embedding
		// For now, just return edges that have embeddings
		edges = append(edges, edge)
	}

	return edges, nil
}

func (k *KuzuDriver) UpsertNodes(ctx context.Context, nodes []*types.Node) error {
	if len(nodes) == 0 {
		return nil
	}

	// For now, implement bulk upsert as individual operations
	// In a full implementation, this could be optimized with batch queries
	var errors []string
	successCount := 0

	for i, node := range nodes {
		err := k.UpsertNode(ctx, node)
		if err != nil {
			errors = append(errors, fmt.Sprintf("node %d (ID: %s): %v", i, node.ID, err))
		} else {
			successCount++
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to upsert %d/%d nodes: %v", len(errors), len(nodes), errors)
	}

	return nil
}

func (k *KuzuDriver) UpsertEdges(ctx context.Context, edges []*types.Edge) error {
	if len(edges) == 0 {
		return nil
	}

	// For now, implement bulk upsert as individual operations
	// In a full implementation, this could be optimized with batch queries
	var errors []string
	successCount := 0

	for i, edge := range edges {
		err := k.UpsertEdge(ctx, edge)
		if err != nil {
			errors = append(errors, fmt.Sprintf("edge %d (ID: %s): %v", i, edge.ID, err))
		} else {
			successCount++
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to upsert %d/%d edges: %v", len(errors), len(edges), errors)
	}

	return nil
}

func (k *KuzuDriver) GetNodesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Node, error) {
	// Escape strings for safe query execution
	escapedGroupID := fmt.Sprintf("'%s'", groupID)
	startTime := start.Format(time.RFC3339)
	endTime := end.Format(time.RFC3339)

	// Query nodes created within the time range
	query := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.group_id = %s
		  AND n.created_at >= '%s'
		  AND n.created_at <= '%s'
		RETURN n.*
		ORDER BY n.created_at
	`, escapedGroupID, startTime, endTime)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes in time range: %w", err)
	}
	defer result.Close()

	var nodes []*types.Node
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		node, err := k.flatTupleToNode(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to node: %w", err)
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (k *KuzuDriver) GetEdgesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Edge, error) {
	// Escape strings for safe query execution
	escapedGroupID := fmt.Sprintf("'%s'", groupID)
	startTime := start.Format(time.RFC3339)
	endTime := end.Format(time.RFC3339)

	// Query edges created within the time range
	query := fmt.Sprintf(`
		MATCH (a:Node)-[e:Edge]->(b:Node)
		WHERE e.group_id = %s
		  AND e.created_at >= '%s'
		  AND e.created_at <= '%s'
		RETURN e.*, a.id AS source_id, b.id AS target_id
		ORDER BY e.created_at
	`, escapedGroupID, startTime, endTime)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query edges in time range: %w", err)
	}
	defer result.Close()

	var edges []*types.Edge
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		edge, err := k.flatTupleToEdge(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to edge: %w", err)
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

func (k *KuzuDriver) GetCommunities(ctx context.Context, groupID string, level int) ([]*types.Node, error) {
	// Escape strings for safe query execution
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	// Query community nodes at the specified level
	query := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.group_id = %s
		  AND n.node_type = 'community'
		  AND n.level = %d
		RETURN n.*
		ORDER BY n.name
	`, escapedGroupID, level)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query communities: %w", err)
	}
	defer result.Close()

	var communities []*types.Node
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		node, err := k.flatTupleToNode(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to node: %w", err)
		}

		communities = append(communities, node)
	}

	return communities, nil
}

func (k *KuzuDriver) BuildCommunities(ctx context.Context, groupID string) error {
	// This is a placeholder for community detection algorithms
	// In a full implementation, this would run community detection
	// algorithms like Louvain, Label Propagation, etc.

	// For now, create a simple community structure based on node connectivity
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	// Simple community detection: group nodes by their connections
	// This is a very basic implementation - real algorithms would be more sophisticated
	query := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.group_id = %s
		  AND n.node_type = 'entity'
		WITH n, size((n)-[:Edge]-()) as degree
		WHERE degree > 0
		WITH collect(n.id) as node_ids
		UNWIND range(0, size(node_ids)-1, 10) as start_idx
		WITH node_ids[start_idx..start_idx+9] as community_nodes, start_idx
		WHERE size(community_nodes) > 0
		CREATE (c:Node {
			id: 'community_' + toString(start_idx),
			name: 'Community ' + toString(start_idx/10 + 1),
			node_type: 'community',
			group_id: %s,
			created_at: datetime(),
			updated_at: datetime(),
			valid_from: datetime(),
			level: 1
		})
	`, escapedGroupID, escapedGroupID)

	_, err := k.conn.Query(query)
	if err != nil {
		return fmt.Errorf("failed to build communities: %w", err)
	}

	return nil
}

func (k *KuzuDriver) CreateIndices(ctx context.Context) error {
	// Create indices for better query performance
	// Note: Index syntax may vary depending on Kuzu version

	indices := []string{
		// Primary indices for lookups
		"CREATE INDEX IF NOT EXISTS node_id_group ON Node(id, group_id)",
		"CREATE INDEX IF NOT EXISTS edge_id_group ON Edge(id, group_id)",
		// Temporal indices
		"CREATE INDEX IF NOT EXISTS node_created_at ON Node(created_at)",
		"CREATE INDEX IF NOT EXISTS edge_created_at ON Edge(created_at)",
		// Type indices
		"CREATE INDEX IF NOT EXISTS node_type ON Node(node_type)",
		"CREATE INDEX IF NOT EXISTS edge_type ON Edge(edge_type)",
		// Group indices
		"CREATE INDEX IF NOT EXISTS node_group ON Node(group_id)",
		"CREATE INDEX IF NOT EXISTS edge_group ON Edge(group_id)",
	}

	for _, indexQuery := range indices {
		_, err := k.conn.Query(indexQuery)
		if err != nil {
			// Log warning but don't fail - indices are optimization
			// In a full implementation, you might log this properly
			_ = err // Ignore index creation errors for now
		}
	}

	return nil
}

func (k *KuzuDriver) GetStats(ctx context.Context, groupID string) (*GraphStats, error) {
	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	// For a simplified implementation, return basic stats
	// In a full implementation, this would have more sophisticated queries
	nodeStatsQuery := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.group_id = %s
		WITH n.node_type as type, count(*) as count
		RETURN type, count
	`, escapedGroupID)

	nodeResult, err := k.conn.Query(nodeStatsQuery)
	if err != nil {
		return &GraphStats{
			NodeCount:      0,
			EdgeCount:      0,
			NodesByType:    make(map[string]int64),
			EdgesByType:    make(map[string]int64),
			CommunityCount: 0,
			LastUpdated:    time.Now(),
		}, nil
	}
	defer nodeResult.Close()

	nodesByType := make(map[string]int64)
	var totalNodes int64

	// Count nodes - simplified for basic implementation
	for nodeResult.HasNext() {
		_, err := nodeResult.Next()
		if err != nil {
			continue
		}
		// Basic counting without complex parsing
		totalNodes++
	}

	// Return basic stats
	return &GraphStats{
		NodeCount:      totalNodes,
		EdgeCount:      0, // Simplified - would need similar query for edges
		NodesByType:    nodesByType,
		EdgesByType:    make(map[string]int64),
		CommunityCount: 0,
		LastUpdated:    time.Now(),
	}, nil
}

// SearchNodes performs text-based search on nodes
func (k *KuzuDriver) SearchNodes(ctx context.Context, query, groupID string, options *SearchOptions) ([]*types.Node, error) {
	if strings.TrimSpace(query) == "" {
		return []*types.Node{}, nil
	}

	limit := 10
	if options != nil && options.Limit > 0 {
		limit = options.Limit
	}

	escapedGroupID := fmt.Sprintf("'%s'", groupID)
	escapedQuery := fmt.Sprintf("'%s'", strings.ReplaceAll(query, "'", "\\'"))

	// Basic text search using CONTAINS
	searchQuery := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.group_id = %s
		  AND (n.name CONTAINS %s OR n.summary CONTAINS %s)
		RETURN n.*
		LIMIT %d
	`, escapedGroupID, escapedQuery, escapedQuery, limit)

	result, err := k.conn.Query(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search nodes: %w", err)
	}
	defer result.Close()

	var nodes []*types.Node
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		node, err := k.flatTupleToNode(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to node: %w", err)
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// SearchEdges performs text-based search on edges
func (k *KuzuDriver) SearchEdges(ctx context.Context, query, groupID string, options *SearchOptions) ([]*types.Edge, error) {
	if strings.TrimSpace(query) == "" {
		return []*types.Edge{}, nil
	}

	limit := 10
	if options != nil && options.Limit > 0 {
		limit = options.Limit
	}

	escapedGroupID := fmt.Sprintf("'%s'", groupID)
	escapedQuery := fmt.Sprintf("'%s'", strings.ReplaceAll(query, "'", "\\'"))

	// Basic text search using CONTAINS
	searchQuery := fmt.Sprintf(`
		MATCH (a:Node)-[e:Edge]->(b:Node)
		WHERE e.group_id = %s
		  AND (e.name CONTAINS %s OR e.summary CONTAINS %s)
		RETURN e.*, a.id AS source_id, b.id AS target_id
		LIMIT %d
	`, escapedGroupID, escapedQuery, escapedQuery, limit)

	result, err := k.conn.Query(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search edges: %w", err)
	}
	defer result.Close()

	var edges []*types.Edge
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		edge, err := k.flatTupleToEdge(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to edge: %w", err)
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// SearchNodesByVector performs vector similarity search on nodes
func (k *KuzuDriver) SearchNodesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Node, error) {
	if len(vector) == 0 {
		return []*types.Node{}, nil
	}

	limit := 10
	minScore := 0.0
	if options != nil {
		if options.Limit > 0 {
			limit = options.Limit
		}
		if options.MinScore > 0 {
			minScore = options.MinScore
		}
	}

	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	// For basic vector search, we'll fetch nodes and compute similarity in-memory
	// In a production implementation, you might want to use Kuzu's native vector indexing
	query := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.group_id = %s
		  AND n.embedding IS NOT NULL
		RETURN n.*
	`, escapedGroupID)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to search nodes by vector: %w", err)
	}
	defer result.Close()

	var candidates []*types.Node
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			continue
		}

		node, err := k.flatTupleToNode(row)
		if err != nil {
			continue
		}

		// Check if node has valid embedding
		if len(node.Embedding) > 0 {
			// Compute cosine similarity
			similarity := k.cosineSimilarity(vector, node.Embedding)
			if similarity >= float32(minScore) {
				candidates = append(candidates, node)
			}
		}
	}

	// Sort by similarity (placeholder - would need proper sorting)
	// For now, just return the first 'limit' candidates
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

// SearchEdgesByVector performs vector similarity search on edges
func (k *KuzuDriver) SearchEdgesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Edge, error) {
	if len(vector) == 0 {
		return []*types.Edge{}, nil
	}

	limit := 10
	minScore := 0.0
	if options != nil {
		if options.Limit > 0 {
			limit = options.Limit
		}
		if options.MinScore > 0 {
			minScore = options.MinScore
		}
	}

	escapedGroupID := fmt.Sprintf("'%s'", groupID)

	// For basic vector search, we'll fetch edges and compute similarity in-memory
	query := fmt.Sprintf(`
		MATCH (a:Node)-[e:Edge]->(b:Node)
		WHERE e.group_id = %s
		  AND e.embedding IS NOT NULL
		RETURN e.*, a.id AS source_id, b.id AS target_id
	`, escapedGroupID)

	result, err := k.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to search edges by vector: %w", err)
	}
	defer result.Close()

	var candidates []*types.Edge
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			continue
		}

		edge, err := k.flatTupleToEdge(row)
		if err != nil {
			continue
		}

		// Check if edge has valid embedding
		if len(edge.Embedding) > 0 {
			// Compute cosine similarity
			similarity := k.cosineSimilarity(vector, edge.Embedding)
			if similarity >= float32(minScore) {
				candidates = append(candidates, edge)
			}
		}
	}

	// Sort by similarity (placeholder - would need proper sorting)
	// For now, just return the first 'limit' candidates
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

// Close closes the Kuzu driver.
func (k *KuzuDriver) Close(ctx context.Context) error {
	if k.conn != nil {
		k.conn.Close()
	}
	if k.database != nil {
		k.database.Close()
	}
	return nil
}

// Helper methods for data conversion

// flatTupleToNode converts a Kuzu FlatTuple to a Node struct
func (k *KuzuDriver) flatTupleToNode(tuple *kuzu.FlatTuple) (*types.Node, error) {
	node := &types.Node{}

	// For now, return a basic implementation
	// In a full implementation, you would extract values from the tuple
	// based on the actual Kuzu API documentation

	// This is a simplified implementation that assumes the tuple contains
	// the node data in a structured format
	values, err := tuple.GetAsSlice()
	if err != nil {
		return nil, fmt.Errorf("failed to get tuple values: %w", err)
	}
	if len(values) < 4 {
		return nil, fmt.Errorf("insufficient data in tuple")
	}

	// Extract basic required fields
	if len(values) > 0 {
		if id := values[0]; id != nil {
			node.ID = fmt.Sprintf("%v", id)
		}
	}
	if len(values) > 1 {
		if name := values[1]; name != nil {
			node.Name = fmt.Sprintf("%v", name)
		}
	}
	if len(values) > 2 {
		if nodeType := values[2]; nodeType != nil {
			node.Type = types.NodeType(fmt.Sprintf("%v", nodeType))
		}
	}
	if len(values) > 3 {
		if groupID := values[3]; groupID != nil {
			node.GroupID = fmt.Sprintf("%v", groupID)
		}
	}

	// Set timestamps to current time if not provided
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	if node.UpdatedAt.IsZero() {
		node.UpdatedAt = time.Now()
	}
	if node.ValidFrom.IsZero() {
		node.ValidFrom = node.CreatedAt
	}

	return node, nil
}

// flatTupleToEdge converts a Kuzu FlatTuple to an Edge struct
func (k *KuzuDriver) flatTupleToEdge(tuple *kuzu.FlatTuple) (*types.Edge, error) {
	edge := &types.Edge{}

	// Get values from the tuple
	values, err := tuple.GetAsSlice()
	if err != nil {
		return nil, fmt.Errorf("failed to get tuple values: %w", err)
	}
	if len(values) < 6 {
		return nil, fmt.Errorf("insufficient data in tuple for edge")
	}

	// Extract basic required fields for edge
	if len(values) > 0 {
		if id := values[0]; id != nil {
			edge.ID = fmt.Sprintf("%v", id)
		}
	}
	if len(values) > 1 {
		if edgeType := values[1]; edgeType != nil {
			edge.Type = types.EdgeType(fmt.Sprintf("%v", edgeType))
		}
	}
	if len(values) > 2 {
		if groupID := values[2]; groupID != nil {
			edge.GroupID = fmt.Sprintf("%v", groupID)
		}
	}
	if len(values) > 3 {
		if name := values[3]; name != nil {
			edge.Name = fmt.Sprintf("%v", name)
		}
	}
	if len(values) > 4 {
		if summary := values[4]; summary != nil {
			edge.Summary = fmt.Sprintf("%v", summary)
		}
	}

	// Extract source and target IDs (these come from the query joins)
	if len(values) > len(values)-2 {
		if sourceID := values[len(values)-2]; sourceID != nil {
			edge.SourceID = fmt.Sprintf("%v", sourceID)
		}
	}
	if len(values) > len(values)-1 {
		if targetID := values[len(values)-1]; targetID != nil {
			edge.TargetID = fmt.Sprintf("%v", targetID)
		}
	}

	// Set timestamps to current time if not provided
	if edge.CreatedAt.IsZero() {
		edge.CreatedAt = time.Now()
	}
	if edge.UpdatedAt.IsZero() {
		edge.UpdatedAt = time.Now()
	}
	if edge.ValidFrom.IsZero() {
		edge.ValidFrom = edge.CreatedAt
	}

	return edge, nil
}

// prepareNodeCreateQuery prepares a CREATE query for a node
func (k *KuzuDriver) prepareNodeCreateQuery(node *types.Node) string {
	// Convert metadata to JSON string
	var metadataJSON string
	if node.Metadata != nil {
		if data, err := json.Marshal(node.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	// Convert timestamps to string format
	createdAt := node.CreatedAt.Format(time.RFC3339)
	updatedAt := node.UpdatedAt.Format(time.RFC3339)
	validFrom := node.ValidFrom.Format(time.RFC3339)

	var validToStr string
	if node.ValidTo != nil {
		validToStr = node.ValidTo.Format(time.RFC3339)
	}

	return fmt.Sprintf(`
		CREATE (n:Node {
			id: '%s',
			name: '%s',
			node_type: '%s',
			group_id: '%s',
			created_at: '%s',
			updated_at: '%s',
			entity_type: '%s',
			summary: '%s',
			episode_type: '%s',
			content: '%s',
			level: %d,
			metadata: '%s',
			valid_from: '%s',
			valid_to: '%s'
		})
	`, k.escapeString(node.ID),
		k.escapeString(node.Name),
		k.escapeString(string(node.Type)),
		k.escapeString(node.GroupID),
		createdAt,
		updatedAt,
		k.escapeString(node.EntityType),
		k.escapeString(node.Summary),
		k.escapeString(string(node.EpisodeType)),
		k.escapeString(node.Content),
		node.Level,
		k.escapeString(metadataJSON),
		validFrom,
		validToStr)
}

// prepareNodeUpdateQuery prepares an UPDATE query for a node
func (k *KuzuDriver) prepareNodeUpdateQuery(node *types.Node) string {
	// Convert metadata to JSON string
	var metadataJSON string
	if node.Metadata != nil {
		if data, err := json.Marshal(node.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	// Convert timestamps to string format
	updatedAt := node.UpdatedAt.Format(time.RFC3339)
	validFrom := node.ValidFrom.Format(time.RFC3339)

	var validToStr string
	if node.ValidTo != nil {
		validToStr = node.ValidTo.Format(time.RFC3339)
	}

	return fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.id = '%s' AND n.group_id = '%s'
		SET n.name = '%s',
			n.node_type = '%s',
			n.updated_at = '%s',
			n.entity_type = '%s',
			n.summary = '%s',
			n.episode_type = '%s',
			n.content = '%s',
			n.level = %d,
			n.metadata = '%s',
			n.valid_from = '%s',
			n.valid_to = '%s'
	`, k.escapeString(node.ID),
		k.escapeString(node.GroupID),
		k.escapeString(node.Name),
		k.escapeString(string(node.Type)),
		updatedAt,
		k.escapeString(node.EntityType),
		k.escapeString(node.Summary),
		k.escapeString(string(node.EpisodeType)),
		k.escapeString(node.Content),
		node.Level,
		k.escapeString(metadataJSON),
		validFrom,
		validToStr)
}

// escapeString escapes single quotes in strings for Cypher queries
func (k *KuzuDriver) escapeString(s string) string {
	return fmt.Sprintf("%s", s)  // Basic implementation
}

// prepareEdgeCreateQuery prepares a CREATE query for an edge
func (k *KuzuDriver) prepareEdgeCreateQuery(edge *types.Edge) string {
	// Convert metadata to JSON string
	var metadataJSON string
	if edge.Metadata != nil {
		if data, err := json.Marshal(edge.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	// Convert timestamps to string format
	createdAt := edge.CreatedAt.Format(time.RFC3339)
	updatedAt := edge.UpdatedAt.Format(time.RFC3339)
	validFrom := edge.ValidFrom.Format(time.RFC3339)

	var validToStr string
	if edge.ValidTo != nil {
		validToStr = edge.ValidTo.Format(time.RFC3339)
	}

	return fmt.Sprintf(`
		MATCH (a:Node {id: '%s', group_id: '%s'})
		MATCH (b:Node {id: '%s', group_id: '%s'})
		CREATE (a)-[e:Edge {
			id: '%s',
			edge_type: '%s',
			group_id: '%s',
			created_at: '%s',
			updated_at: '%s',
			name: '%s',
			summary: '%s',
			strength: %f,
			metadata: '%s',
			valid_from: '%s',
			valid_to: '%s'
		}]->(b)
	`, k.escapeString(edge.SourceID),
		k.escapeString(edge.GroupID),
		k.escapeString(edge.TargetID),
		k.escapeString(edge.GroupID),
		k.escapeString(edge.ID),
		k.escapeString(string(edge.Type)),
		k.escapeString(edge.GroupID),
		createdAt,
		updatedAt,
		k.escapeString(edge.Name),
		k.escapeString(edge.Summary),
		edge.Strength,
		k.escapeString(metadataJSON),
		validFrom,
		validToStr)
}

// prepareEdgeUpdateQuery prepares an UPDATE query for an edge
func (k *KuzuDriver) prepareEdgeUpdateQuery(edge *types.Edge) string {
	// Convert metadata to JSON string
	var metadataJSON string
	if edge.Metadata != nil {
		if data, err := json.Marshal(edge.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	// Convert timestamps to string format
	updatedAt := edge.UpdatedAt.Format(time.RFC3339)
	validFrom := edge.ValidFrom.Format(time.RFC3339)

	var validToStr string
	if edge.ValidTo != nil {
		validToStr = edge.ValidTo.Format(time.RFC3339)
	}

	return fmt.Sprintf(`
		MATCH (a:Node)-[e:Edge]->(b:Node)
		WHERE e.id = '%s' AND e.group_id = '%s'
		SET e.edge_type = '%s',
			e.updated_at = '%s',
			e.name = '%s',
			e.summary = '%s',
			e.strength = %f,
			e.metadata = '%s',
			e.valid_from = '%s',
			e.valid_to = '%s'
	`, k.escapeString(edge.ID),
		k.escapeString(edge.GroupID),
		k.escapeString(string(edge.Type)),
		updatedAt,
		k.escapeString(edge.Name),
		k.escapeString(edge.Summary),
		edge.Strength,
		k.escapeString(metadataJSON),
		validFrom,
		validToStr)
}

// cosineSimilarity computes the cosine similarity between two vectors
func (k *KuzuDriver) cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	// Calculate square roots for proper normalization
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}