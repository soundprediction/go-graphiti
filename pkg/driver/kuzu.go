package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/kuzudb/go-kuzu"
)

// SCHEMA_QUERIES defines the Kuzu database schema exactly as in Python implementation
// Kuzu requires an explicit schema.
// As Kuzu currently does not support creating full text indexes on edge properties,
// we work around this by representing (n:Entity)-[:RELATES_TO]->(m:Entity) as
// (n)-[:RELATES_TO]->(e:RelatesToNode_)-[:RELATES_TO]->(m).
const KuzuSchemaQueries = `
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

// KuzuDriver implements the GraphDriver interface for Kuzu databases exactly like Python implementation
type KuzuDriver struct {
	provider GraphProvider
	db       *kuzu.Database
	client   *kuzu.Connection // Note: Python uses AsyncConnection, but Go kuzu doesn't have async
	dbPath   string
}

// NewKuzuDriver creates a new Kuzu driver instance with exact same signature as Python
// Parameters:
//   - db: Database path (defaults to ":memory:" like Python)
//   - maxConcurrentQueries: Maximum concurrent queries (defaults to 1 like Python)
func NewKuzuDriver(db string, maxConcurrentQueries int) (*KuzuDriver, error) {
	if db == "" {
		db = ":memory:"
	}
	if maxConcurrentQueries <= 0 {
		maxConcurrentQueries = 1
	}

	// Create the Kuzu database
	database, err := kuzu.OpenDatabase(db, kuzu.DefaultSystemConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to open kuzu database: %w", err)
	}

	driver := &KuzuDriver{
		provider: GraphProviderKuzu,
		db:       database,
		dbPath:   db,
	}

	// Setup schema exactly like Python
	driver.setupSchema()

	// Create connection - Go kuzu doesn't have AsyncConnection but we simulate the interface
	client, err := kuzu.OpenConnection(database)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to open kuzu connection: %w", err)
	}
	driver.client = client

	return driver, nil
}

// ExecuteQuery executes a query with parameters, exactly matching Python signature
// Returns (results, summary, keys) tuple like Python, though summary and keys are unused in Kuzu
func (k *KuzuDriver) ExecuteQuery(cypherQuery string, kwargs map[string]interface{}) (interface{}, interface{}, interface{}, error) {
	// Filter parameters exactly like Python implementation
	params := make(map[string]any) // Use 'any' instead of 'interface{}' for go-kuzu compatibility
	for key, value := range kwargs {
		if value != nil {
			params[key] = value
		}
	}

	// Kuzu does not support these parameters (matching Python comment)
	delete(params, "database_")
	delete(params, "routing_")

	var results *kuzu.QueryResult
	var err error

	// Check if we have parameters to use prepared statement
	if len(params) > 0 {
		// Use prepared statement for parameterized queries
		preparedStatement, err := k.client.Prepare(cypherQuery)
		if err != nil {
			// Log error with truncated params for debugging (matching Python behavior)
			truncatedParams := make(map[string]interface{})
			for key, value := range params {
				if arr, ok := value.([]interface{}); ok && len(arr) > 5 {
					truncatedParams[key] = arr[:5]
				} else {
					truncatedParams[key] = value
				}
			}
			log.Printf("Error preparing Kuzu query: %v\nQuery: %s\nParams: %v", err, cypherQuery, truncatedParams)
			return nil, nil, nil, err
		}

		results, err = k.client.Execute(preparedStatement, params)
		if err != nil {
			// Log error with truncated params for debugging (matching Python behavior)
			truncatedParams := make(map[string]interface{})
			for key, value := range params {
				if arr, ok := value.([]interface{}); ok && len(arr) > 5 {
					truncatedParams[key] = arr[:5]
				} else {
					truncatedParams[key] = value
				}
			}
			log.Printf("Error executing Kuzu query: %v\nQuery: %s\nParams: %v", err, cypherQuery, truncatedParams)
			return nil, nil, nil, err
		}
	} else {
		// Use simple Query for queries without parameters
		results, err = k.client.Query(cypherQuery)
		if err != nil {
			log.Printf("Error executing Kuzu query: %v\nQuery: %s", err, cypherQuery)
			return nil, nil, nil, err
		}
	}

	defer results.Close()

	if !results.HasNext() {
		return []map[string]interface{}{}, nil, nil, nil
	}

	// Convert results to list of dictionaries like Python
	var dictResults []map[string]interface{}
	for results.HasNext() {
		row, err := results.Next()
		if err != nil {
			continue
		}

		// Convert FlatTuple to map[string]interface{} to match Python rows_as_dict()
		rowDict, err := k.flatTupleToDict(row)
		if err != nil {
			continue
		}
		dictResults = append(dictResults, rowDict)
	}

	return dictResults, nil, nil, nil
}

// Session creates a new session exactly like Python implementation
func (k *KuzuDriver) Session(database *string) GraphDriverSession {
	return NewKuzuDriverSession(k)
}

// Close closes the driver exactly like Python implementation
func (k *KuzuDriver) Close() error {
	// Do not explicitly close the connection, instead rely on GC (matching Python comment)
	return nil
}

// DeleteAllIndexes does nothing for Kuzu (matching Python implementation)
func (k *KuzuDriver) DeleteAllIndexes(database string) {
	// pass (matching Python implementation)
}

// setupSchema initializes the database schema exactly like Python implementation
func (k *KuzuDriver) setupSchema() {
	conn, err := kuzu.OpenConnection(k.db)
	if err != nil {
		log.Printf("Failed to create connection for schema setup: %v", err)
		return
	}
	defer conn.Close()

	_, err = conn.Query(KuzuSchemaQueries)
	if err != nil {
		log.Printf("Failed to create schema: %v", err)
	}
}

// Provider returns the graph provider type
func (k *KuzuDriver) Provider() GraphProvider {
	return k.provider
}

// GetAossClient returns nil for Kuzu (matching Python implementation)
func (k *KuzuDriver) GetAossClient() interface{} {
	return nil // aoss_client: None = None
}

// flatTupleToDict converts a Kuzu FlatTuple to a map to simulate Python's rows_as_dict()
func (k *KuzuDriver) flatTupleToDict(tuple *kuzu.FlatTuple) (map[string]interface{}, error) {
	values, err := tuple.GetAsSlice()
	if err != nil {
		return nil, err
	}

	// For now, create generic column names since Kuzu Go doesn't expose column names easily
	// In a full implementation, this would need proper column name extraction
	result := make(map[string]interface{})
	for i, value := range values {
		result[fmt.Sprintf("col_%d", i)] = value
	}

	return result, nil
}

// === Backward compatibility methods for existing interface ===

// GetNode retrieves a node by ID from the appropriate table based on node type.
func (k *KuzuDriver) GetNode(ctx context.Context, nodeID, groupID string) (*types.Node, error) {
	// Try to find node in each table type
	tables := []string{"Entity", "Episodic", "Community", "RelatesToNode_"}

	for _, table := range tables {
		query := fmt.Sprintf(`
			MATCH (n:%s)
			WHERE n.uuid = $uuid AND n.group_id = $group_id
			RETURN n.*
		`, table)

		params := map[string]interface{}{
			"uuid":     nodeID,
			"group_id": groupID,
		}

		result, _, _, err := k.ExecuteQuery(query, params)
		if err != nil {
			continue
		}

		if resultList, ok := result.([]map[string]interface{}); ok && len(resultList) > 0 {
			return k.mapToNode(resultList[0], table)
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
	err := k.executeNodeCreateQuery(node, tableName)
	if err != nil {
		// If creation fails, try to update
		updateErr := k.executeNodeUpdateQuery(node, tableName)
		if updateErr != nil {
			return fmt.Errorf("failed to create or update node: create error: %w, update error: %w", err, updateErr)
		}
	}

	return nil
}

// DeleteNode removes a node and its relationships from all tables.
func (k *KuzuDriver) DeleteNode(ctx context.Context, nodeID, groupID string) error {
	// Delete from all possible tables
	tables := []string{"Entity", "Episodic", "Community", "RelatesToNode_"}

	for _, table := range tables {
		// Delete relationships first
		deleteRelsQuery := fmt.Sprintf(`
			MATCH (n:%s)-[r]-()
			WHERE n.uuid = '%s' AND n.group_id = '%s'
			DELETE r
		`, table, strings.ReplaceAll(nodeID, "'", "\\'"), strings.ReplaceAll(groupID, "'", "\\'"))

		k.ExecuteQuery(deleteRelsQuery, nil) // Ignore errors for missing relationships

		// Delete the node
		deleteNodeQuery := fmt.Sprintf(`
			MATCH (n:%s)
			WHERE n.uuid = '%s' AND n.group_id = '%s'
			DELETE n
		`, table, strings.ReplaceAll(nodeID, "'", "\\'"), strings.ReplaceAll(groupID, "'", "\\'"))

		k.ExecuteQuery(deleteNodeQuery, nil) // Ignore errors for nodes not in this table
	}

	return nil
}

// GetNodes retrieves multiple nodes by their IDs.
func (k *KuzuDriver) GetNodes(ctx context.Context, nodeIDs []string, groupID string) ([]*types.Node, error) {
	if len(nodeIDs) == 0 {
		return []*types.Node{}, nil
	}

	var nodes []*types.Node
	for _, nodeID := range nodeIDs {
		node, err := k.GetNode(ctx, nodeID, groupID)
		if err == nil {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// GetEdge retrieves an edge by ID using the RelatesToNode_ pattern.
func (k *KuzuDriver) GetEdge(ctx context.Context, edgeID, groupID string) (*types.Edge, error) {
	// Query using the RelatesToNode_ pattern from Python implementation
	query := fmt.Sprintf(`
		MATCH (a:Entity)-[:RELATES_TO]->(rel:RelatesToNode_)-[:RELATES_TO]->(b:Entity)
		WHERE rel.uuid = '%s' AND rel.group_id = '%s'
		RETURN rel.*, a.uuid AS source_id, b.uuid AS target_id
	`, strings.ReplaceAll(edgeID, "'", "\\'"), strings.ReplaceAll(groupID, "'", "\\'"))

	result, _, _, err := k.ExecuteQuery(query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query edge: %w", err)
	}

	if resultList, ok := result.([]map[string]interface{}); ok && len(resultList) > 0 {
		return k.mapToEdge(resultList[0])
	}

	return nil, fmt.Errorf("edge not found")
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

	// Try to create the edge using RelatesToNode_ pattern
	err := k.executeEdgeCreateQuery(edge)
	if err != nil {
		// If creation fails, try to update
		updateErr := k.executeEdgeUpdateQuery(edge)
		if updateErr != nil {
			return fmt.Errorf("failed to create or update edge: create error: %w, update error: %w", err, updateErr)
		}
	}

	return nil
}

// DeleteEdge removes an edge.
func (k *KuzuDriver) DeleteEdge(ctx context.Context, edgeID, groupID string) error {
	// Delete using RelatesToNode_ pattern
	deleteQuery := fmt.Sprintf(`
		MATCH (a:Entity)-[:RELATES_TO]->(rel:RelatesToNode_)-[:RELATES_TO]->(b:Entity)
		WHERE rel.uuid = '%s' AND rel.group_id = '%s'
		DELETE rel
	`, strings.ReplaceAll(edgeID, "'", "\\'"), strings.ReplaceAll(groupID, "'", "\\'"))

	_, _, _, err := k.ExecuteQuery(deleteQuery, nil)
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

	var edges []*types.Edge
	for _, edgeID := range edgeIDs {
		edge, err := k.GetEdge(ctx, edgeID, groupID)
		if err == nil {
			edges = append(edges, edge)
		}
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

	// Build variable-length path query
	query := fmt.Sprintf(`
		MATCH (start:Entity)-[:RELATES_TO*1..%d]-(neighbor:Entity)
		WHERE start.uuid = '%s' AND start.group_id = '%s'
		  AND neighbor.group_id = '%s'
		  AND neighbor.uuid <> start.uuid
		RETURN DISTINCT neighbor.*
	`, maxDistance, strings.ReplaceAll(nodeID, "'", "\\'"),
		strings.ReplaceAll(groupID, "'", "\\'"), strings.ReplaceAll(groupID, "'", "\\'"))

	result, _, _, err := k.ExecuteQuery(query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query neighbors: %w", err)
	}

	var neighbors []*types.Node
	if resultList, ok := result.([]map[string]interface{}); ok {
		for _, row := range resultList {
			node, err := k.mapToNode(row, "Entity")
			if err == nil {
				neighbors = append(neighbors, node)
			}
		}
	}

	return neighbors, nil
}

// GetRelatedNodes retrieves nodes related through specific edge types
func (k *KuzuDriver) GetRelatedNodes(ctx context.Context, nodeID, groupID string, edgeTypes []types.EdgeType) ([]*types.Node, error) {
	// Simple implementation for now
	return k.GetNeighbors(ctx, nodeID, groupID, 1)
}

// SearchNodesByEmbedding performs basic embedding search
func (k *KuzuDriver) SearchNodesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Node, error) {
	// Simplified implementation - would need proper vector search
	return []*types.Node{}, nil
}

// SearchEdgesByEmbedding performs basic embedding search
func (k *KuzuDriver) SearchEdgesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Edge, error) {
	// Simplified implementation - would need proper vector search
	return []*types.Edge{}, nil
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

	// Basic text search using CONTAINS
	searchQuery := fmt.Sprintf(`
		MATCH (n:Entity)
		WHERE n.group_id = '%s'
		  AND (n.name CONTAINS '%s' OR n.summary CONTAINS '%s')
		RETURN n.*
		LIMIT %d
	`, strings.ReplaceAll(groupID, "'", "\\'"),
		strings.ReplaceAll(query, "'", "\\'"),
		strings.ReplaceAll(query, "'", "\\'"), limit)

	result, _, _, err := k.ExecuteQuery(searchQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search nodes: %w", err)
	}

	var nodes []*types.Node
	if resultList, ok := result.([]map[string]interface{}); ok {
		for _, row := range resultList {
			node, err := k.mapToNode(row, "Entity")
			if err == nil {
				nodes = append(nodes, node)
			}
		}
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

	// Basic text search using CONTAINS
	searchQuery := fmt.Sprintf(`
		MATCH (a:Entity)-[:RELATES_TO]->(rel:RelatesToNode_)-[:RELATES_TO]->(b:Entity)
		WHERE rel.group_id = '%s'
		  AND (rel.name CONTAINS '%s' OR rel.fact CONTAINS '%s')
		RETURN rel.*, a.uuid AS source_id, b.uuid AS target_id
		LIMIT %d
	`, strings.ReplaceAll(groupID, "'", "\\'"),
		strings.ReplaceAll(query, "'", "\\'"),
		strings.ReplaceAll(query, "'", "\\'"), limit)

	result, _, _, err := k.ExecuteQuery(searchQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search edges: %w", err)
	}

	var edges []*types.Edge
	if resultList, ok := result.([]map[string]interface{}); ok {
		for _, row := range resultList {
			edge, err := k.mapToEdge(row)
			if err == nil {
				edges = append(edges, edge)
			}
		}
	}

	return edges, nil
}

// SearchNodesByVector performs vector search (placeholder)
func (k *KuzuDriver) SearchNodesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Node, error) {
	return []*types.Node{}, nil
}

// SearchEdgesByVector performs vector search (placeholder)
func (k *KuzuDriver) SearchEdgesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Edge, error) {
	return []*types.Edge{}, nil
}

// UpsertNodes bulk upserts nodes
func (k *KuzuDriver) UpsertNodes(ctx context.Context, nodes []*types.Node) error {
	for _, node := range nodes {
		if err := k.UpsertNode(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

// UpsertEdges bulk upserts edges
func (k *KuzuDriver) UpsertEdges(ctx context.Context, edges []*types.Edge) error {
	for _, edge := range edges {
		if err := k.UpsertEdge(ctx, edge); err != nil {
			return err
		}
	}
	return nil
}

// GetNodesInTimeRange retrieves nodes in a time range
func (k *KuzuDriver) GetNodesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Node, error) {
	return []*types.Node{}, nil // Placeholder
}

// GetEdgesInTimeRange retrieves edges in a time range
func (k *KuzuDriver) GetEdgesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Edge, error) {
	return []*types.Edge{}, nil // Placeholder
}

// GetCommunities retrieves community nodes
func (k *KuzuDriver) GetCommunities(ctx context.Context, groupID string, level int) ([]*types.Node, error) {
	return []*types.Node{}, nil // Placeholder
}

// BuildCommunities builds community structure
func (k *KuzuDriver) BuildCommunities(ctx context.Context, groupID string) error {
	return nil // Placeholder
}

// CreateIndices creates database indices
func (k *KuzuDriver) CreateIndices(ctx context.Context) error {
	return nil // Placeholder
}

// GetStats returns graph statistics
func (k *KuzuDriver) GetStats(ctx context.Context, groupID string) (*GraphStats, error) {
	return &GraphStats{
		NodeCount:      0,
		EdgeCount:      0,
		NodesByType:    make(map[string]int64),
		EdgesByType:    make(map[string]int64),
		CommunityCount: 0,
		LastUpdated:    time.Now(),
	}, nil
}

// === Helper methods ===

func (k *KuzuDriver) getTableNameForNodeType(nodeType types.NodeType) string {
	switch nodeType {
	case types.EpisodicNodeType:
		return "Episodic"
	case types.EntityNodeType:
		return "Entity"
	case types.CommunityNodeType:
		return "Community"
	default:
		return "Entity"
	}
}

func (k *KuzuDriver) mapToNode(data map[string]interface{}, tableName string) (*types.Node, error) {
	node := &types.Node{}

	if id, ok := data["uuid"]; ok {
		node.ID = fmt.Sprintf("%v", id)
	}
	if name, ok := data["name"]; ok {
		node.Name = fmt.Sprintf("%v", name)
	}
	if groupID, ok := data["group_id"]; ok {
		node.GroupID = fmt.Sprintf("%v", groupID)
	}
	if summary, ok := data["summary"]; ok {
		node.Summary = fmt.Sprintf("%v", summary)
	}
	if content, ok := data["content"]; ok {
		node.Content = fmt.Sprintf("%v", content)
	}

	// Set node type based on table
	switch tableName {
	case "Episodic":
		node.Type = types.EpisodicNodeType
	case "Entity":
		node.Type = types.EntityNodeType
	case "Community":
		node.Type = types.CommunityNodeType
	default:
		node.Type = types.EntityNodeType
	}

	// Set default timestamps
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

func (k *KuzuDriver) mapToEdge(data map[string]interface{}) (*types.Edge, error) {
	edge := &types.Edge{}

	if id, ok := data["uuid"]; ok {
		edge.ID = fmt.Sprintf("%v", id)
	}
	if groupID, ok := data["group_id"]; ok {
		edge.GroupID = fmt.Sprintf("%v", groupID)
	}
	if name, ok := data["name"]; ok {
		edge.Name = fmt.Sprintf("%v", name)
	}
	if fact, ok := data["fact"]; ok {
		edge.Summary = fmt.Sprintf("%v", fact)
	}
	if sourceID, ok := data["source_id"]; ok {
		edge.SourceID = fmt.Sprintf("%v", sourceID)
	}
	if targetID, ok := data["target_id"]; ok {
		edge.TargetID = fmt.Sprintf("%v", targetID)
	}

	edge.Type = types.EntityEdgeType

	// Set default timestamps
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

func (k *KuzuDriver) executeNodeCreateQuery(node *types.Node, tableName string) error {
	var metadataJSON string
	if node.Metadata != nil {
		if data, err := json.Marshal(node.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	var query string
	params := make(map[string]interface{})

	switch tableName {
	case "Episodic":
		query = `
			CREATE (n:Episodic {
				uuid: $uuid,
				name: $name,
				group_id: $group_id,
				created_at: $created_at,
				source: $source,
				source_description: $source_description,
				content: $content,
				valid_at: $valid_at,
				entity_edges: []
			})
		`
		params["uuid"] = node.ID
		params["name"] = node.Name
		params["group_id"] = node.GroupID
		params["created_at"] = node.CreatedAt
		params["source"] = string(node.EpisodeType)
		params["source_description"] = ""
		params["content"] = node.Content
		params["valid_at"] = node.ValidFrom
	case "Entity":
		query = `
			CREATE (n:Entity {
				uuid: $uuid,
				name: $name,
				group_id: $group_id,
				labels: [],
				created_at: $created_at,
				name_embedding: [],
				summary: $summary,
				attributes: $attributes
			})
		`
		params["uuid"] = node.ID
		params["name"] = node.Name
		params["group_id"] = node.GroupID
		params["created_at"] = node.CreatedAt
		params["summary"] = node.Summary
		params["attributes"] = metadataJSON
	case "Community":
		query = `
			CREATE (n:Community {
				uuid: $uuid,
				name: $name,
				group_id: $group_id,
				created_at: $created_at,
				name_embedding: [],
				summary: $summary
			})
		`
		params["uuid"] = node.ID
		params["name"] = node.Name
		params["group_id"] = node.GroupID
		params["created_at"] = node.CreatedAt
		params["summary"] = node.Summary
	default:
		return fmt.Errorf("unknown table: %s", tableName)
	}

	_, _, _, err := k.ExecuteQuery(query, params)
	return err
}

func (k *KuzuDriver) executeNodeUpdateQuery(node *types.Node, tableName string) error {
	var metadataJSON string
	if node.Metadata != nil {
		if data, err := json.Marshal(node.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	var query string
	params := make(map[string]interface{})

	switch tableName {
	case "Episodic":
		query = `
			MATCH (n:Episodic)
			WHERE n.uuid = $uuid AND n.group_id = $group_id
			SET n.name = $name,
				n.content = $content,
				n.valid_at = $valid_at
		`
		params["uuid"] = node.ID
		params["group_id"] = node.GroupID
		params["name"] = node.Name
		params["content"] = node.Content
		params["valid_at"] = node.ValidFrom
	case "Entity":
		query = `
			MATCH (n:Entity)
			WHERE n.uuid = $uuid AND n.group_id = $group_id
			SET n.name = $name,
				n.summary = $summary,
				n.attributes = $attributes
		`
		params["uuid"] = node.ID
		params["group_id"] = node.GroupID
		params["name"] = node.Name
		params["summary"] = node.Summary
		params["attributes"] = metadataJSON
	case "Community":
		query = `
			MATCH (n:Community)
			WHERE n.uuid = $uuid AND n.group_id = $group_id
			SET n.name = $name,
				n.summary = $summary
		`
		params["uuid"] = node.ID
		params["group_id"] = node.GroupID
		params["name"] = node.Name
		params["summary"] = node.Summary
	default:
		return fmt.Errorf("unknown table: %s", tableName)
	}

	_, _, _, err := k.ExecuteQuery(query, params)
	return err
}

func (k *KuzuDriver) executeEdgeCreateQuery(edge *types.Edge) error {
	var metadataJSON string
	if edge.Metadata != nil {
		if data, err := json.Marshal(edge.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	query := `
		MATCH (a:Entity {uuid: $source_uuid, group_id: $group_id})
		MATCH (b:Entity {uuid: $target_uuid, group_id: $group_id})
		CREATE (rel:RelatesToNode_ {
			uuid: $uuid,
			group_id: $group_id,
			created_at: $created_at,
			name: $name,
			fact: $fact,
			fact_embedding: [],
			episodes: [],
			expired_at: $expired_at,
			valid_at: $valid_at,
			invalid_at: $invalid_at,
			attributes: $attributes
		})
		CREATE (a)-[:RELATES_TO]->(rel)
		CREATE (rel)-[:RELATES_TO]->(b)
	`

	params := make(map[string]interface{})
	params["source_uuid"] = edge.SourceID
	params["target_uuid"] = edge.TargetID
	params["group_id"] = edge.GroupID
	params["uuid"] = edge.ID
	params["created_at"] = edge.CreatedAt
	params["name"] = edge.Name
	params["fact"] = edge.Summary
	params["attributes"] = metadataJSON
	params["valid_at"] = edge.ValidFrom

	if edge.ValidTo != nil {
		params["expired_at"] = edge.ValidTo
		params["invalid_at"] = edge.ValidTo
	} else {
		params["expired_at"] = nil
		params["invalid_at"] = nil
	}

	_, _, _, err := k.ExecuteQuery(query, params)
	return err
}

func (k *KuzuDriver) executeEdgeUpdateQuery(edge *types.Edge) error {
	var metadataJSON string
	if edge.Metadata != nil {
		if data, err := json.Marshal(edge.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	query := `
		MATCH (rel:RelatesToNode_)
		WHERE rel.uuid = $uuid AND rel.group_id = $group_id
		SET rel.name = $name,
			rel.fact = $fact,
			rel.expired_at = $expired_at,
			rel.valid_at = $valid_at,
			rel.invalid_at = $invalid_at,
			rel.attributes = $attributes
	`

	params := make(map[string]interface{})
	params["uuid"] = edge.ID
	params["group_id"] = edge.GroupID
	params["name"] = edge.Name
	params["fact"] = edge.Summary
	params["attributes"] = metadataJSON
	params["valid_at"] = edge.ValidFrom

	if edge.ValidTo != nil {
		params["expired_at"] = edge.ValidTo
		params["invalid_at"] = edge.ValidTo
	} else {
		params["expired_at"] = nil
		params["invalid_at"] = nil
	}

	_, _, _, err := k.ExecuteQuery(query, params)
	return err
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

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// KuzuDriverSession implements GraphDriverSession for Kuzu exactly like Python
type KuzuDriverSession struct {
	provider GraphProvider
	driver   *KuzuDriver
}

// NewKuzuDriverSession creates a new KuzuDriverSession
func NewKuzuDriverSession(driver *KuzuDriver) *KuzuDriverSession {
	return &KuzuDriverSession{
		provider: GraphProviderKuzu,
		driver:   driver,
	}
}

// Provider returns the provider type
func (s *KuzuDriverSession) Provider() GraphProvider {
	return s.provider
}

// Close implements session close (no cleanup needed for Kuzu, matching Python comment)
func (s *KuzuDriverSession) Close() error {
	// Do not close the session here, as we're reusing the driver connection (matching Python comment)
	return nil
}

// ExecuteWrite executes a write function exactly like Python implementation
func (s *KuzuDriverSession) ExecuteWrite(ctx context.Context, fn func(context.Context, GraphDriverSession, ...interface{}) (interface{}, error), args ...interface{}) (interface{}, error) {
	// Directly await the provided function with `self` as the transaction/session (matching Python comment)
	return fn(ctx, s, args...)
}

// Run executes a query or list of queries exactly like Python implementation
func (s *KuzuDriverSession) Run(ctx context.Context, query interface{}, kwargs map[string]interface{}) error {
	if queryList, ok := query.([][]interface{}); ok {
		// Handle list of [cypher, params] pairs
		for _, queryPair := range queryList {
			if len(queryPair) >= 2 {
				cypher := fmt.Sprintf("%v", queryPair[0])
				params, ok := queryPair[1].(map[string]interface{})
				if !ok {
					params = make(map[string]interface{})
				}
				_, _, _, err := s.driver.ExecuteQuery(cypher, params)
				if err != nil {
					return err
				}
			}
		}
	} else {
		// Handle single query string
		cypherQuery := fmt.Sprintf("%v", query)
		if kwargs == nil {
			kwargs = make(map[string]interface{})
		}
		_, _, _, err := s.driver.ExecuteQuery(cypherQuery, kwargs)
		if err != nil {
			return err
		}
	}
	return nil
}

// Enter implements context manager entry (for async with in Python)
func (s *KuzuDriverSession) Enter(ctx context.Context) (GraphDriverSession, error) {
	return s, nil
}

// Exit implements context manager exit (for async with in Python)
func (s *KuzuDriverSession) Exit(ctx context.Context, excType, excVal, excTb interface{}) error {
	// No cleanup needed for Kuzu, but method must exist (matching Python comment)
	return nil
}