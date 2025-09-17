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
// This implementation follows the schema pattern from the Python Graphiti Kuzu driver.
type KuzuDriver struct {
	database *kuzu.Database
	conn     *kuzu.Connection
	dbPath   string
}

// Schema queries for Kuzu database initialization
// Following the Python implementation's schema design
const SCHEMA_QUERIES = `
    CREATE NODE TABLE IF NOT EXISTS Episodic (
        uuid STRING PRIMARY KEY,
        name STRING,
        group_id STRING,
        created_at TIMESTAMP,
        source STRING,
        source_description STRING,
        content STRING,
        valid_at TIMESTAMP,
        entity_edges STRING[]
    );
    CREATE NODE TABLE IF NOT EXISTS Entity (
        uuid STRING PRIMARY KEY,
        name STRING,
        group_id STRING,
        labels STRING[],
        created_at TIMESTAMP,
        name_embedding FLOAT[],
        summary STRING,
        attributes STRING
    );
    CREATE NODE TABLE IF NOT EXISTS Community (
        uuid STRING PRIMARY KEY,
        name STRING,
        group_id STRING,
        created_at TIMESTAMP,
        name_embedding FLOAT[],
        summary STRING
    );
    CREATE NODE TABLE IF NOT EXISTS RelatesToNode_ (
        uuid STRING PRIMARY KEY,
        group_id STRING,
        created_at TIMESTAMP,
        name STRING,
        fact STRING,
        fact_embedding FLOAT[],
        episodes STRING[],
        expired_at TIMESTAMP,
        valid_at TIMESTAMP,
        invalid_at TIMESTAMP,
        attributes STRING
    );
    CREATE REL TABLE IF NOT EXISTS RELATES_TO(
        FROM Entity TO RelatesToNode_,
        FROM RelatesToNode_ TO Entity
    );
    CREATE REL TABLE IF NOT EXISTS MENTIONS(
        FROM Episodic TO Entity,
        uuid STRING PRIMARY KEY,
        group_id STRING,
        created_at TIMESTAMP
    );
    CREATE REL TABLE IF NOT EXISTS HAS_MEMBER(
        FROM Community TO Entity,
        FROM Community TO Community,
        uuid STRING,
        group_id STRING,
        created_at TIMESTAMP
    );
`

// NewKuzuDriver creates a new Kuzu driver instance.
// Kuzu is an embedded database, so it works with a local directory path.
// Uses :memory: for in-memory database by default, matching Python implementation.
//
// Parameters:
//   - dbPath: Path to the Kuzu database directory (use ":memory:" for in-memory)
//
// Example:
//
//	driver, err := driver.NewKuzuDriver(":memory:")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer driver.Close(ctx)
func NewKuzuDriver(dbPath string) (*KuzuDriver, error) {
	if dbPath == "" {
		dbPath = ":memory:"
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

	// Initialize the schema following Python implementation
	err = driver.setupSchema()
	if err != nil {
		driver.Close(context.Background())
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return driver, nil
}

// setupSchema initializes the database schema following the Python implementation
func (k *KuzuDriver) setupSchema() error {
	_, err := k.conn.Query(SCHEMA_QUERIES)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

// executeQuery executes a query with parameters, following Python implementation pattern
func (k *KuzuDriver) executeQuery(query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	// Filter out unsupported parameters (matching Python implementation)
	filteredParams := make(map[string]interface{})
	for key, value := range params {
		if value != nil && key != "database_" && key != "routing_" {
			filteredParams[key] = value
		}
	}

	result, err := k.conn.Query(query)
	if err != nil {
		// Log error with truncated params for debugging (matching Python behavior)
		truncatedParams := make(map[string]interface{})
		for key, value := range filteredParams {
			if arr, ok := value.([]interface{}); ok && len(arr) > 5 {
				truncatedParams[key] = arr[:5]
			} else {
				truncatedParams[key] = value
			}
		}
		return nil, fmt.Errorf("error executing Kuzu query: %w\nQuery: %s\nParams: %v", err, query, truncatedParams)
	}
	defer result.Close()

	if !result.HasNext() {
		return []map[string]interface{}{}, nil
	}

	var results []map[string]interface{}
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to get next row: %w", err)
		}

		// Convert to map - this is simplified, real implementation would need proper conversion
		rowMap := make(map[string]interface{})
		if values, err := row.GetAsSlice(); err == nil {
			for i, value := range values {
				rowMap[fmt.Sprintf("col_%d", i)] = value
			}
		}
		results = append(results, rowMap)
	}

	return results, nil
}

// GetNode retrieves a node by ID from the appropriate table based on node type.
func (k *KuzuDriver) GetNode(ctx context.Context, nodeID, groupID string) (*types.Node, error) {
	// Try to find node in each table type
	tables := []string{"Entity", "Episodic", "Community", "RelatesToNode_"}

	for _, table := range tables {
		query := fmt.Sprintf(`
			MATCH (n:%s)
			WHERE n.uuid = '%s' AND n.group_id = '%s'
			RETURN n.*
		`, table, strings.ReplaceAll(nodeID, "'", "\\'"), strings.ReplaceAll(groupID, "'", "\\'"))

		result, err := k.conn.Query(query)
		if err != nil {
			continue
		}
		defer result.Close()

		if result.HasNext() {
			row, err := result.Next()
			if err != nil {
				continue
			}
			return k.flatTupleToNode(row, table)
		}
	}

	return nil, fmt.Errorf("node not found")
}

// UpsertNode creates or updates a node in the appropriate table based on node type.
func (k *KuzuDriver) UpsertNode(ctx context.Context, node *types.Node) error {
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	node.UpdatedAt = time.Now()
	if node.ValidFrom.IsZero() {
		node.ValidFrom = node.CreatedAt
	}

	// Determine which table to use based on node type
	tableName := k.getTableNameForNodeType(node.Type)

	// Try to create first
	createQuery := k.prepareNodeCreateQuery(node, tableName)
	_, err := k.conn.Query(createQuery)
	if err != nil {
		// If creation fails, try to update
		updateQuery := k.prepareNodeUpdateQuery(node, tableName)
		_, updateErr := k.conn.Query(updateQuery)
		if updateErr != nil {
			return fmt.Errorf("failed to create or update node: create error: %w, update error: %w", err, updateErr)
		}
	}

	return nil
}

// DeleteNode removes a node and its relationships from all tables.
func (k *KuzuDriver) DeleteNode(ctx context.Context, nodeID, groupID string) error {
	escapedNodeID := strings.ReplaceAll(nodeID, "'", "\\'")
	escapedGroupID := strings.ReplaceAll(groupID, "'", "\\'")

	// Delete from all possible tables
	tables := []string{"Entity", "Episodic", "Community", "RelatesToNode_"}

	for _, table := range tables {
		// Delete relationships first
		deleteRelsQuery := fmt.Sprintf(`
			MATCH (n:%s)-[r]-()
			WHERE n.uuid = '%s' AND n.group_id = '%s'
			DELETE r
		`, table, escapedNodeID, escapedGroupID)

		k.conn.Query(deleteRelsQuery) // Ignore errors for missing relationships

		// Delete the node
		deleteNodeQuery := fmt.Sprintf(`
			MATCH (n:%s)
			WHERE n.uuid = '%s' AND n.group_id = '%s'
			DELETE n
		`, table, escapedNodeID, escapedGroupID)

		k.conn.Query(deleteNodeQuery) // Ignore errors for nodes not in this table
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

		node, err := k.flatTupleToNode(row, "Unknown")
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to node: %w", err)
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetEdge retrieves an edge by ID using the RelatesToNode_ pattern.
func (k *KuzuDriver) GetEdge(ctx context.Context, edgeID, groupID string) (*types.Edge, error) {
	escapedEdgeID := k.escapeString(edgeID)
	escapedGroupID := k.escapeString(groupID)

	// Query using the RelatesToNode_ pattern from Python implementation
	query := fmt.Sprintf(`
		MATCH (a:Entity)-[:RELATES_TO]->(rel:RelatesToNode_)-[:RELATES_TO]->(b:Entity)
		WHERE rel.uuid = '%s' AND rel.group_id = '%s'
		RETURN rel.*, a.uuid AS source_id, b.uuid AS target_id
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

// UpsertEdge creates or updates an edge using the RelatesToNode_ pattern.
func (k *KuzuDriver) UpsertEdge(ctx context.Context, edge *types.Edge) error {
	if edge.CreatedAt.IsZero() {
		edge.CreatedAt = time.Now()
	}
	edge.UpdatedAt = time.Now()
	if edge.ValidFrom.IsZero() {
		edge.ValidFrom = edge.CreatedAt
	}

	// First ensure source and target nodes exist as Entity nodes
	_, err := k.GetNode(ctx, edge.SourceID, edge.GroupID)
	if err != nil {
		return fmt.Errorf("source node %s not found: %w", edge.SourceID, err)
	}

	_, err = k.GetNode(ctx, edge.TargetID, edge.GroupID)
	if err != nil {
		return fmt.Errorf("target node %s not found: %w", edge.TargetID, err)
	}

	// Try to create the edge using RelatesToNode_ pattern
	createQuery := k.prepareEdgeCreateQuery(edge)
	_, err = k.conn.Query(createQuery)
	if err != nil {
		// If creation fails, try to update
		updateQuery := k.prepareEdgeUpdateQuery(edge)
		_, updateErr := k.conn.Query(updateQuery)
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

		node, err := k.flatTupleToNode(row, "Unknown")
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

		node, err := k.flatTupleToNode(row, "Unknown")
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

		node, err := k.flatTupleToNode(row, "Unknown")
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
	startTime := fmt.Sprintf("TIMESTAMP('%s')", start.Format(time.RFC3339))
	endTime := fmt.Sprintf("TIMESTAMP('%s')", end.Format(time.RFC3339))

	// Query nodes created within the time range
	query := fmt.Sprintf(`
		MATCH (n:Node)
		WHERE n.group_id = %s
		  AND n.created_at >= %s
		  AND n.created_at <= %s
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

		node, err := k.flatTupleToNode(row, "Unknown")
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
	startTime := fmt.Sprintf("TIMESTAMP('%s')", start.Format(time.RFC3339))
	endTime := fmt.Sprintf("TIMESTAMP('%s')", end.Format(time.RFC3339))

	// Query edges created within the time range
	query := fmt.Sprintf(`
		MATCH (a:Node)-[e:Edge]->(b:Node)
		WHERE e.group_id = %s
		  AND e.created_at >= %s
		  AND e.created_at <= %s
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

		node, err := k.flatTupleToNode(row, "Unknown")
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

		node, err := k.flatTupleToNode(row, "Unknown")
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

		node, err := k.flatTupleToNode(row, "Unknown")
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

// getTableNameForNodeType returns the appropriate table name for a node type
func (k *KuzuDriver) getTableNameForNodeType(nodeType types.NodeType) string {
	switch nodeType {
	case types.EpisodicNodeType:
		return "Episodic"
	case types.EntityNodeType:
		return "Entity"
	case types.CommunityNodeType:
		return "Community"
	default:
		return "Entity" // Default to Entity table
	}
}

// flatTupleToNode converts a Kuzu FlatTuple to a Node struct
func (k *KuzuDriver) flatTupleToNode(tuple *kuzu.FlatTuple, tableName string) (*types.Node, error) {
	node := &types.Node{}

	// Extract values from the tuple based on table schema
	values, err := tuple.GetAsSlice()
	if err != nil {
		return nil, fmt.Errorf("failed to get tuple values: %w", err)
	}

	// Handle different table schemas
	switch tableName {
	case "Episodic":
		node.Type = types.EpisodicNodeType
		if len(values) > 0 && values[0] != nil {
			node.ID = fmt.Sprintf("%v", values[0]) // uuid
		}
		if len(values) > 1 && values[1] != nil {
			node.Name = fmt.Sprintf("%v", values[1]) // name
		}
		if len(values) > 2 && values[2] != nil {
			node.GroupID = fmt.Sprintf("%v", values[2]) // group_id
		}
		if len(values) > 6 && values[6] != nil {
			node.Content = fmt.Sprintf("%v", values[6]) // content
		}
	case "Entity":
		node.Type = types.EntityNodeType
		if len(values) > 0 && values[0] != nil {
			node.ID = fmt.Sprintf("%v", values[0]) // uuid
		}
		if len(values) > 1 && values[1] != nil {
			node.Name = fmt.Sprintf("%v", values[1]) // name
		}
		if len(values) > 2 && values[2] != nil {
			node.GroupID = fmt.Sprintf("%v", values[2]) // group_id
		}
		if len(values) > 6 && values[6] != nil {
			node.Summary = fmt.Sprintf("%v", values[6]) // summary
		}
	case "Community":
		node.Type = types.CommunityNodeType
		if len(values) > 0 && values[0] != nil {
			node.ID = fmt.Sprintf("%v", values[0]) // uuid
		}
		if len(values) > 1 && values[1] != nil {
			node.Name = fmt.Sprintf("%v", values[1]) // name
		}
		if len(values) > 2 && values[2] != nil {
			node.GroupID = fmt.Sprintf("%v", values[2]) // group_id
		}
		if len(values) > 5 && values[5] != nil {
			node.Summary = fmt.Sprintf("%v", values[5]) // summary
		}
	default:
		// Default to Entity type
		node.Type = types.EntityNodeType
		if len(values) > 0 && values[0] != nil {
			node.ID = fmt.Sprintf("%v", values[0])
		}
		if len(values) > 1 && values[1] != nil {
			node.Name = fmt.Sprintf("%v", values[1])
		}
		if len(values) > 2 && values[2] != nil {
			node.GroupID = fmt.Sprintf("%v", values[2])
		}
	}

	// Parse timestamps from data if available
	if len(values) > 4 && values[4] != nil {
		if createdAtStr := fmt.Sprintf("%v", values[4]); createdAtStr != "" {
			if parsedTime, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
				node.CreatedAt = parsedTime
			}
		}
	}

	// Set default timestamps if not provided
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

// prepareNodeCreateQuery prepares a CREATE query for a node in the specified table
func (k *KuzuDriver) prepareNodeCreateQuery(node *types.Node, tableName string) string {
	// Convert metadata to JSON string
	var metadataJSON string
	if node.Metadata != nil {
		if data, err := json.Marshal(node.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	// Convert timestamps to Kuzu TIMESTAMP format (without quotes)
	createdAt := fmt.Sprintf("TIMESTAMP('%s')", node.CreatedAt.Format(time.RFC3339))
	validFrom := fmt.Sprintf("TIMESTAMP('%s')", node.ValidFrom.Format(time.RFC3339))

	// Generate query based on table type
	switch tableName {
	case "Episodic":
		return fmt.Sprintf(`
			CREATE (n:Episodic {
				uuid: '%s',
				name: '%s',
				group_id: '%s',
				created_at: %s,
				source: '%s',
				source_description: '%s',
				content: '%s',
				valid_at: %s,
				entity_edges: []
			})
		`, k.escapeString(node.ID),
			k.escapeString(node.Name),
			k.escapeString(node.GroupID),
			createdAt,
			k.escapeString(node.Reference.Format(time.RFC3339)),
			k.escapeString(""), // source_description from metadata if available
			k.escapeString(node.Content),
			validFrom)
	case "Entity":
		return fmt.Sprintf(`
			CREATE (n:Entity {
				uuid: '%s',
				name: '%s',
				group_id: '%s',
				labels: [],
				created_at: %s,
				name_embedding: [],
				summary: '%s',
				attributes: '%s'
			})
		`, k.escapeString(node.ID),
			k.escapeString(node.Name),
			k.escapeString(node.GroupID),
			createdAt,
			k.escapeString(node.Summary),
			k.escapeString(metadataJSON))
	case "Community":
		return fmt.Sprintf(`
			CREATE (n:Community {
				uuid: '%s',
				name: '%s',
				group_id: '%s',
				created_at: %s,
				name_embedding: [],
				summary: '%s'
			})
		`, k.escapeString(node.ID),
			k.escapeString(node.Name),
			k.escapeString(node.GroupID),
			createdAt,
			k.escapeString(node.Summary))
	default:
		// Default to Entity
		return fmt.Sprintf(`
			CREATE (n:Entity {
				uuid: '%s',
				name: '%s',
				group_id: '%s',
				labels: [],
				created_at: %s,
				name_embedding: [],
				summary: '%s',
				attributes: '%s'
			})
		`, k.escapeString(node.ID),
			k.escapeString(node.Name),
			k.escapeString(node.GroupID),
			createdAt,
			k.escapeString(node.Summary),
			k.escapeString(metadataJSON))
	}
}

// prepareNodeUpdateQuery prepares an UPDATE query for a node in the specified table
func (k *KuzuDriver) prepareNodeUpdateQuery(node *types.Node, tableName string) string {
	// Convert metadata to JSON string
	var metadataJSON string
	if node.Metadata != nil {
		if data, err := json.Marshal(node.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	// Convert timestamps to Kuzu TIMESTAMP format (without quotes)
	validFrom := fmt.Sprintf("TIMESTAMP('%s')", node.ValidFrom.Format(time.RFC3339))

	// Generate update query based on table type
	switch tableName {
	case "Episodic":
		return fmt.Sprintf(`
			MATCH (n:Episodic)
			WHERE n.uuid = '%s' AND n.group_id = '%s'
			SET n.name = '%s',
				n.content = '%s',
				n.valid_at = %s
		`, k.escapeString(node.ID),
			k.escapeString(node.GroupID),
			k.escapeString(node.Name),
			k.escapeString(node.Content),
			validFrom)
	case "Entity":
		return fmt.Sprintf(`
			MATCH (n:Entity)
			WHERE n.uuid = '%s' AND n.group_id = '%s'
			SET n.name = '%s',
				n.summary = '%s',
				n.attributes = '%s'
		`, k.escapeString(node.ID),
			k.escapeString(node.GroupID),
			k.escapeString(node.Name),
			k.escapeString(node.Summary),
			k.escapeString(metadataJSON))
	case "Community":
		return fmt.Sprintf(`
			MATCH (n:Community)
			WHERE n.uuid = '%s' AND n.group_id = '%s'
			SET n.name = '%s',
				n.summary = '%s'
		`, k.escapeString(node.ID),
			k.escapeString(node.GroupID),
			k.escapeString(node.Name),
			k.escapeString(node.Summary))
	default:
		// Default to Entity
		return fmt.Sprintf(`
			MATCH (n:Entity)
			WHERE n.uuid = '%s' AND n.group_id = '%s'
			SET n.name = '%s',
				n.summary = '%s',
				n.attributes = '%s'
		`, k.escapeString(node.ID),
			k.escapeString(node.GroupID),
			k.escapeString(node.Name),
			k.escapeString(node.Summary),
			k.escapeString(metadataJSON))
	}
}

// escapeString escapes dangerous characters in strings for Kuzu queries
func (k *KuzuDriver) escapeString(s string) string {
	// Escape single quotes
	s = strings.ReplaceAll(s, "'", "\\'")
	// Escape double quotes
	s = strings.ReplaceAll(s, "\"", "\\\"")
	// Escape backslashes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	// Escape newlines
	s = strings.ReplaceAll(s, "\n", "\\n")
	// Escape carriage returns
	s = strings.ReplaceAll(s, "\r", "\\r")
	// Escape tabs
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// prepareEdgeCreateQuery prepares a CREATE query for an edge using RelatesToNode_ pattern
func (k *KuzuDriver) prepareEdgeCreateQuery(edge *types.Edge) string {
	// Convert metadata to JSON string
	var metadataJSON string
	if edge.Metadata != nil {
		if data, err := json.Marshal(edge.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	// Convert timestamps to Kuzu TIMESTAMP format (without quotes)
	createdAt := fmt.Sprintf("TIMESTAMP('%s')", edge.CreatedAt.Format(time.RFC3339))
	validFrom := fmt.Sprintf("TIMESTAMP('%s')", edge.ValidFrom.Format(time.RFC3339))

	var validToStr string
	if edge.ValidTo != nil {
		validToStr = fmt.Sprintf("TIMESTAMP('%s')", edge.ValidTo.Format(time.RFC3339))
	} else {
		validToStr = "NULL"
	}

	var invalidAtStr string
	if edge.ValidTo != nil {
		invalidAtStr = fmt.Sprintf("TIMESTAMP('%s')", edge.ValidTo.Format(time.RFC3339))
	} else {
		invalidAtStr = "NULL"
	}

	// Use the RelatesToNode_ pattern from Python implementation
	return fmt.Sprintf(`
		MATCH (a:Entity {uuid: '%s', group_id: '%s'})
		MATCH (b:Entity {uuid: '%s', group_id: '%s'})
		CREATE (rel:RelatesToNode_ {
			uuid: '%s',
			group_id: '%s',
			created_at: %s,
			name: '%s',
			fact: '%s',
			fact_embedding: [],
			episodes: [],
			expired_at: %s,
			valid_at: %s,
			invalid_at: %s,
			attributes: '%s'
		})
		CREATE (a)-[:RELATES_TO]->(rel)
		CREATE (rel)-[:RELATES_TO]->(b)
	`, k.escapeString(edge.SourceID),
		k.escapeString(edge.GroupID),
		k.escapeString(edge.TargetID),
		k.escapeString(edge.GroupID),
		k.escapeString(edge.ID),
		k.escapeString(edge.GroupID),
		createdAt,
		k.escapeString(edge.Name),
		k.escapeString(edge.Summary), // Use summary as fact
		validToStr, // expired_at
		validFrom, // valid_at
		invalidAtStr, // invalid_at
		k.escapeString(metadataJSON))
}

// prepareEdgeUpdateQuery prepares an UPDATE query for an edge using RelatesToNode_ pattern
func (k *KuzuDriver) prepareEdgeUpdateQuery(edge *types.Edge) string {
	// Convert metadata to JSON string
	var metadataJSON string
	if edge.Metadata != nil {
		if data, err := json.Marshal(edge.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	// Convert timestamps to Kuzu TIMESTAMP format (without quotes)
	validFrom := fmt.Sprintf("TIMESTAMP('%s')", edge.ValidFrom.Format(time.RFC3339))

	var validToStr string
	if edge.ValidTo != nil {
		validToStr = fmt.Sprintf("TIMESTAMP('%s')", edge.ValidTo.Format(time.RFC3339))
	} else {
		validToStr = "NULL"
	}

	var invalidAtStr string
	if edge.ValidTo != nil {
		invalidAtStr = fmt.Sprintf("TIMESTAMP('%s')", edge.ValidTo.Format(time.RFC3339))
	} else {
		invalidAtStr = "NULL"
	}

	// Update using RelatesToNode_ pattern
	return fmt.Sprintf(`
		MATCH (rel:RelatesToNode_)
		WHERE rel.uuid = '%s' AND rel.group_id = '%s'
		SET rel.name = '%s',
			rel.fact = '%s',
			rel.expired_at = %s,
			rel.valid_at = %s,
			rel.invalid_at = %s,
			rel.attributes = '%s'
	`, k.escapeString(edge.ID),
		k.escapeString(edge.GroupID),
		k.escapeString(edge.Name),
		k.escapeString(edge.Summary), // Use summary as fact
		validToStr, // expired_at
		validFrom, // valid_at
		invalidAtStr, // invalid_at
		k.escapeString(metadataJSON))
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