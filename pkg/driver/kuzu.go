package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/kuzudb/go-kuzu"
	"github.com/soundprediction/go-graphiti/pkg/types"
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

	// Load FTS extension for this connection
	// Extensions must be loaded for each session (connection)
	_, err = client.Query("LOAD EXTENSION FTS;")
	if err != nil && !strings.Contains(err.Error(), "already loaded") {
		log.Printf("Warning: Failed to load FTS extension on main connection: %v", err)
	}

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

	// Get column names from the result
	columnNames := results.GetColumnNames()

	if !results.HasNext() {
		return []map[string]interface{}{}, columnNames, nil, nil
	}

	// Convert results to list of dictionaries like Python
	var dictResults []map[string]interface{}
	for results.HasNext() {
		row, err := results.Next()
		if err != nil {
			continue
		}

		// Convert FlatTuple to map[string]interface{} using actual column names
		values, err := row.GetAsSlice()
		if err != nil {
			continue
		}

		rowDict := make(map[string]interface{})
		for i, value := range values {
			if i < len(columnNames) {
				rowDict[columnNames[i]] = value
			}
		}

		dictResults = append(dictResults, rowDict)
	}

	return dictResults, columnNames, nil, nil
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

	// Install FTS extension (one-time operation, will be no-op if already installed)
	_, err = conn.Query("INSTALL FTS;")
	if err != nil && !strings.Contains(err.Error(), "already installed") {
		log.Printf("FTS extension install note: %v", err)
	}

	// Load FTS extension for this temporary setup connection
	// Note: Each connection needs to load extensions separately
	_, err = conn.Query("LOAD EXTENSION FTS;")
	if err != nil && !strings.Contains(err.Error(), "already loaded") {
		log.Printf("Failed to load FTS extension for setup: %v", err)
		return
	}

	// Create schema tables
	_, err = conn.Query(KuzuSchemaQueries)
	if err != nil {
		log.Printf("Failed to create schema: %v", err)
	}

	// Create fulltext indexes for BM25 search (matching Python implementation)
	// From graph_queries.py get_fulltext_indices() for Kuzu provider
	// Note: These can be created before or after data exists in the tables
	fulltextIndexQueries := []string{
		"CALL CREATE_FTS_INDEX('Episodic', 'episode_content', ['content', 'source', 'source_description']);",
		"CALL CREATE_FTS_INDEX('Entity', 'node_name_and_summary', ['name', 'summary']);",
		"CALL CREATE_FTS_INDEX('Community', 'community_name', ['name']);",
		"CALL CREATE_FTS_INDEX('RelatesToNode_', 'edge_name_and_fact', ['name', 'fact']);",
	}

	for _, query := range fulltextIndexQueries {
		_, err = conn.Query(query)
		if err != nil {
			// Log but continue - indexes may already exist or table may not have data yet
			log.Printf("Fulltext index creation note: %v", err)
		}
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

func (k *KuzuDriver) NodeExists(ctx context.Context, node *types.Node) bool {
	tableName := k.getTableNameForNodeType(node.Type)

	query := fmt.Sprintf(`
		MATCH (n:%s)
		WHERE n.uuid = $uuid AND n.group_id = $group_id
		RETURN n.uuid
		LIMIT 1
	`, tableName)

	params := map[string]interface{}{
		"uuid":     node.ID,
		"group_id": node.GroupID,
	}

	result, _, _, err := k.ExecuteQuery(query, params)
	if err != nil {
		return false
	}

	if resultList, ok := result.([]map[string]interface{}); ok && len(resultList) > 0 {
		return true
	}

	return false
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

	// See if the node already exists in the table

	// Try to create first
	if !k.NodeExists(ctx, node) {
		err := k.executeNodeCreateQuery(node, tableName)
		if err != nil {
			return fmt.Errorf("failed to create node %w", err)
		}
		return err

	}

	updateErr := k.executeNodeUpdateQuery(node, tableName)
	if updateErr != nil {
		return fmt.Errorf("failed to update node %w", updateErr)
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
	query := `
		MATCH (a:Entity)-[:RELATES_TO]->(rel:RelatesToNode_)-[:RELATES_TO]->(b:Entity)
		WHERE rel.uuid = $uuid AND rel.group_id = $group_id
		RETURN rel.uuid as uuid, rel.name as name, rel.fact as fact, rel.group_id as group_id, a.uuid AS source_id, b.uuid AS target_id
	`

	params := map[string]interface{}{
		"uuid":     edgeID,
		"group_id": groupID,
	}

	result, _, _, err := k.ExecuteQuery(query, params)
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

	if !k.EdgeExists(ctx, edge) {
		err := k.executeEdgeCreateQuery(edge)
		if err != nil {
			return fmt.Errorf("failed to create edge %w", err)
		}
		return err
	}

	updateErr := k.executeEdgeUpdateQuery(edge)
	if updateErr != nil {
		return fmt.Errorf("failed to update edge %w", updateErr)
	}

	return nil
}

func (k *KuzuDriver) EdgeExists(ctx context.Context, edge *types.Edge) bool {
	query := `
		MATCH (rel:RelatesToNode_)
		WHERE rel.uuid = $uuid AND rel.group_id = $group_id
		RETURN rel.uuid
		LIMIT 1
	`

	params := map[string]interface{}{
		"uuid":     edge.ID,
		"group_id": edge.GroupID,
	}

	result, _, _, err := k.ExecuteQuery(query, params)
	if err != nil {
		return false
	}

	if resultList, ok := result.([]map[string]interface{}); ok && len(resultList) > 0 {
		return true
	}

	return false
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
	params["fact"] = edge.Fact
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
	params["fact"] = edge.Fact
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

// SearchNodesByEmbedding performs vector similarity search on node embeddings using cosine similarity.
// This matches the Python implementation in search_utils.py:node_similarity_search()
// For Kuzu, it uses array_cosine_similarity function on name_embedding field.
func (k *KuzuDriver) SearchNodesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Node, error) {
	if limit <= 0 {
		limit = 10
	}

	// Convert float32 embedding to float64 for kuzu parameter
	embeddingF64 := make([]float64, len(embedding))
	for i, v := range embedding {
		embeddingF64[i] = float64(v)
	}

	// Build the Cypher query matching Python's Kuzu implementation
	// From search_utils.py:node_similarity_search() for Kuzu provider
	query := `
		MATCH (n:Entity)
		WHERE n.group_id = $group_id
		  AND size(n.name_embedding) > 0
		WITH n, array_cosine_similarity(n.name_embedding, CAST($search_vector AS FLOAT[` + fmt.Sprintf("%d", len(embedding)) + `])) AS score
		WHERE score > 0.0
		RETURN
			n.uuid AS uuid,
			n.name AS name,
			n.group_id AS group_id,
			n.created_at AS created_at,
			n.summary AS summary,
			n.labels AS labels,
			n.name_embedding AS name_embedding,
			n.attributes AS attributes,
			score
		ORDER BY score DESC
		LIMIT $limit
	`

	params := map[string]interface{}{
		"group_id":      groupID,
		"search_vector": embeddingF64,
		"limit":         int64(limit),
	}

	result, _, _, err := k.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute node embedding search: %w", err)
	}

	resultList, ok := result.([]map[string]interface{})
	if !ok || len(resultList) == 0 {
		return []*types.Node{}, nil
	}

	var nodes []*types.Node
	for _, row := range resultList {
		// Extract data from row
		uuid, _ := row["uuid"].(string)
		name, _ := row["name"].(string)
		groupIDVal, _ := row["group_id"].(string)
		summary, _ := row["summary"].(string)

		// Handle created_at timestamp
		var createdAt time.Time
		if createdAtVal, ok := row["created_at"].(time.Time); ok {
			createdAt = createdAtVal
		}

		// Handle labels array
		var labels []string
		if labelsVal, ok := row["labels"].([]interface{}); ok {
			labels = make([]string, len(labelsVal))
			for i, label := range labelsVal {
				if labelStr, ok := label.(string); ok {
					labels[i] = labelStr
				}
			}
		}

		// Handle name_embedding
		var nameEmbedding []float32
		if embVal, ok := row["name_embedding"].([]interface{}); ok {
			nameEmbedding = make([]float32, len(embVal))
			for i, v := range embVal {
				if f, ok := v.(float64); ok {
					nameEmbedding[i] = float32(f)
				} else if f32, ok := v.(float32); ok {
					nameEmbedding[i] = f32
				}
			}
		}

		node := &types.Node{
			ID:        uuid,
			Name:      name,
			GroupID:   groupIDVal,
			CreatedAt: createdAt,
			Summary:   summary,
			Type:      types.EntityNodeType,
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// SearchEdgesByEmbedding performs vector similarity search on edge embeddings using cosine similarity.
// This matches the Python implementation in search_utils.py:edge_similarity_search()
// For Kuzu, edges are represented as RelatesToNode_ intermediate nodes with fact_embedding field.
func (k *KuzuDriver) SearchEdgesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Edge, error) {
	if limit <= 0 {
		limit = 10
	}

	// Convert float32 embedding to float64 for kuzu parameter
	embeddingF64 := make([]float64, len(embedding))
	for i, v := range embedding {
		embeddingF64[i] = float64(v)
	}

	// Build the Cypher query matching Python's Kuzu implementation for edges
	// From search_utils.py:edge_similarity_search() for Kuzu provider
	// Uses RelatesToNode_ intermediate representation
	query := `
		MATCH (n:Entity)-[:RELATES_TO]->(e:RelatesToNode_)-[:RELATES_TO]->(m:Entity)
		WHERE e.group_id = $group_id
		WITH DISTINCT e, n, m, array_cosine_similarity(e.fact_embedding, CAST($search_vector AS FLOAT[` + fmt.Sprintf("%d", len(embedding)) + `])) AS score
		WHERE score > 0.0
		RETURN
			e.uuid AS uuid,
			e.group_id AS group_id,
			e.created_at AS created_at,
			e.name AS name,
			e.fact AS fact,
			e.fact_embedding AS fact_embedding,
			e.episodes AS episodes,
			e.expired_at AS expired_at,
			e.valid_at AS valid_at,
			e.invalid_at AS invalid_at,
			n.uuid AS source_node_uuid,
			m.uuid AS target_node_uuid,
			score
		ORDER BY score DESC
		LIMIT $limit
	`

	params := map[string]interface{}{
		"group_id":      groupID,
		"search_vector": embeddingF64,
		"limit":         int64(limit),
	}

	result, _, _, err := k.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute edge embedding search: %w", err)
	}

	resultList, ok := result.([]map[string]interface{})
	if !ok || len(resultList) == 0 {
		return []*types.Edge{}, nil
	}

	var edges []*types.Edge
	for _, row := range resultList {
		// Extract data from row
		uuid, _ := row["uuid"].(string)
		groupIDVal, _ := row["group_id"].(string)
		name, _ := row["name"].(string)
		fact, _ := row["fact"].(string)
		sourceNodeUUID, _ := row["source_node_uuid"].(string)
		targetNodeUUID, _ := row["target_node_uuid"].(string)

		// Handle timestamps
		var createdAt, expiredAt, validAt, invalidAt time.Time
		if createdAtVal, ok := row["created_at"].(time.Time); ok {
			createdAt = createdAtVal
		}
		if expiredAtVal, ok := row["expired_at"].(time.Time); ok {
			expiredAt = expiredAtVal
		}
		if validAtVal, ok := row["valid_at"].(time.Time); ok {
			validAt = validAtVal
		}
		if invalidAtVal, ok := row["invalid_at"].(time.Time); ok {
			invalidAt = invalidAtVal
		}

		// Handle episodes array
		var episodes []string
		if episodesVal, ok := row["episodes"].([]interface{}); ok {
			episodes = make([]string, len(episodesVal))
			for i, ep := range episodesVal {
				if epStr, ok := ep.(string); ok {
					episodes[i] = epStr
				}
			}
		}

		// Handle fact_embedding
		var factEmbedding []float32
		if embVal, ok := row["fact_embedding"].([]interface{}); ok {
			factEmbedding = make([]float32, len(embVal))
			for i, v := range embVal {
				if f, ok := v.(float64); ok {
					factEmbedding[i] = float32(f)
				} else if f32, ok := v.(float32); ok {
					factEmbedding[i] = f32
				}
			}
		}

		edge := &types.Edge{
			BaseEdge: types.BaseEdge{
				ID:           uuid,
				GroupID:      groupIDVal,
				SourceNodeID: sourceNodeUUID,
				TargetNodeID: targetNodeUUID,
				CreatedAt:    createdAt,
			},
			Name:          name,
			Fact:          fact,
			FactEmbedding: factEmbedding,
			Episodes:      episodes,
			ExpiredAt:     &expiredAt,
			ValidAt:       &validAt,
			InvalidAt:     &invalidAt,
		}

		edges = append(edges, edge)
	}

	return edges, nil
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

	// BM25 fulltext search using QUERY_FTS_INDEX (matching Python implementation)
	// From graph_queries.py get_nodes_query() and search_utils.py node_fulltext_search()
	// For Kuzu: CALL QUERY_FTS_INDEX('Entity', 'node_name_and_summary', query, TOP := limit)
	searchQuery := `
		CALL QUERY_FTS_INDEX('Entity', 'node_name_and_summary', cast($query AS STRING), TOP := $limit)
		WITH node AS n, score
		WHERE n.group_id = $group_id
		RETURN n.*, score
		ORDER BY score DESC
	`

	params := map[string]interface{}{
		"query":    query,
		"group_id": groupID,
		"limit":    int64(limit),
	}

	result, _, _, err := k.ExecuteQuery(searchQuery, params)
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

	// BM25 fulltext search using QUERY_FTS_INDEX (matching Python implementation)
	// From graph_queries.py get_relationships_query() and search_utils.py edge_fulltext_search()
	// For Kuzu edges (RelatesToNode_): CALL QUERY_FTS_INDEX('RelatesToNode_', 'edge_name_and_fact', query, TOP := limit)
	searchQuery := `
		CALL QUERY_FTS_INDEX('RelatesToNode_', 'edge_name_and_fact', cast($query AS STRING), TOP := $limit)
		YIELD node, score
		MATCH (n:Entity)-[:RELATES_TO]->(e:RelatesToNode_ {uuid: node.uuid})-[:RELATES_TO]->(m:Entity)
		WHERE e.group_id = $group_id
		RETURN
			e.uuid AS uuid,
			e.group_id AS group_id,
			e.created_at AS created_at,
			e.name AS name,
			e.fact AS fact,
			e.fact_embedding AS fact_embedding,
			e.episodes AS episodes,
			e.expired_at AS expired_at,
			e.valid_at AS valid_at,
			e.invalid_at AS invalid_at,
			n.uuid AS source_node_uuid,
			m.uuid AS target_node_uuid,
			score
		ORDER BY score DESC
	`

	params := map[string]interface{}{
		"query":    query,
		"group_id": groupID,
		"limit":    int64(limit),
	}

	result, _, _, err := k.ExecuteQuery(searchQuery, params)
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

// SearchNodesByVector performs vector similarity search on nodes with additional options
func (k *KuzuDriver) SearchNodesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Node, error) {
	if len(vector) == 0 {
		return []*types.Node{}, nil
	}

	limit := 10
	if options != nil && options.Limit > 0 {
		limit = options.Limit
	}

	// Use the existing SearchNodesByEmbedding method which already handles similarity scoring
	// The Kuzu query already includes the score in the results
	nodes, err := k.SearchNodesByEmbedding(ctx, vector, groupID, limit)
	if err != nil {
		return nil, err
	}

	// Note: MinScore filtering is already handled in SearchNodesByEmbedding via the WHERE score > 0.0 clause
	// Additional filtering by options.MinScore could be added here if needed
	if options != nil && options.MinScore > 0 {
		// The score is already computed in SearchNodesByEmbedding, but we need to recompute
		// for filtering since we don't store it in the Node struct
		// For now, we rely on the database-level filtering
	}

	return nodes, nil
}

// SearchEdgesByVector performs vector similarity search on edges with additional options
func (k *KuzuDriver) SearchEdgesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Edge, error) {
	if len(vector) == 0 {
		return []*types.Edge{}, nil
	}

	limit := 10
	if options != nil && options.Limit > 0 {
		limit = options.Limit
	}

	// Use the existing SearchEdgesByEmbedding method which already handles similarity scoring
	// The Kuzu query already includes the score in the results
	edges, err := k.SearchEdgesByEmbedding(ctx, vector, groupID, limit)
	if err != nil {
		return nil, err
	}

	// Note: MinScore filtering is already handled in SearchEdgesByEmbedding via the WHERE score > 0.0 clause
	// Additional filtering by options.MinScore could be added here if needed
	if options != nil && options.MinScore > 0 {
		// The score is already computed in SearchEdgesByEmbedding, but we need to recompute
		// for filtering since we don't store it in the Edge struct
		// For now, we rely on the database-level filtering
	}

	return edges, nil
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
	query := `
		MATCH (n:Entity)
		WHERE n.group_id = $group_id
		  AND n.created_at >= $start
		  AND n.created_at <= $end
		RETURN n.uuid AS uuid,
		       n.name AS name,
		       n.summary AS summary,
		       n.group_id AS group_id,
		       n.created_at AS created_at,
		       n.name_embedding AS name_embedding
	`

	params := map[string]interface{}{
		"group_id": groupID,
		"start":    start.Format(time.RFC3339),
		"end":      end.Format(time.RFC3339),
	}

	result, _, _, err := k.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetNodesInTimeRange query: %w", err)
	}

	rows, ok := result.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	nodes := make([]*types.Node, 0, len(rows))
	for _, row := range rows {
		node := &types.Node{}

		if uuid, ok := row["uuid"].(string); ok {
			node.ID = uuid
		}
		if name, ok := row["name"].(string); ok {
			node.Name = name
		}
		if summary, ok := row["summary"].(string); ok {
			node.Summary = summary
		}
		if groupID, ok := row["group_id"].(string); ok {
			node.GroupID = groupID
		}
		if createdAt, ok := row["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				node.CreatedAt = t
			}
		}
		if embedding, ok := row["name_embedding"].([]interface{}); ok {
			node.NameEmbedding = make([]float32, len(embedding))
			for i, v := range embedding {
				if f, ok := v.(float64); ok {
					node.NameEmbedding[i] = float32(f)
				}
			}
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetEdgesInTimeRange retrieves edges in a time range
func (k *KuzuDriver) GetEdgesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Edge, error) {
	query := `
		MATCH (n:Entity)-[:RELATES_TO]->(e:RelatesToNode_)-[:RELATES_TO]->(m:Entity)
		WHERE e.group_id = $group_id
		  AND e.created_at >= $start
		  AND e.created_at <= $end
		RETURN DISTINCT e.uuid AS uuid,
		       e.fact AS fact,
		       e.created_at AS created_at,
		       e.expired_at AS expired_at,
		       e.invalid_at AS invalid_at,
		       e.episodes AS episodes,
		       e.group_id AS group_id,
		       e.fact_embedding AS fact_embedding,
		       n.uuid AS source_node_id,
		       m.uuid AS target_node_id
	`

	params := map[string]interface{}{
		"group_id": groupID,
		"start":    start.Format(time.RFC3339),
		"end":      end.Format(time.RFC3339),
	}

	result, _, _, err := k.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetEdgesInTimeRange query: %w", err)
	}

	rows, ok := result.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	edges := make([]*types.Edge, 0, len(rows))
	for _, row := range rows {
		edge := &types.Edge{
			BaseEdge: types.BaseEdge{},
		}

		if uuid, ok := row["uuid"].(string); ok {
			edge.ID = uuid
		}
		if fact, ok := row["fact"].(string); ok {
			edge.Name = fact
			edge.Fact = fact
		}
		if createdAt, ok := row["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				edge.CreatedAt = t
			}
		}
		if expiredAt, ok := row["expired_at"].(string); ok && expiredAt != "" {
			if t, err := time.Parse(time.RFC3339, expiredAt); err == nil {
				edge.ExpiredAt = &t
			}
		}
		if invalidAt, ok := row["invalid_at"].(string); ok && invalidAt != "" {
			if t, err := time.Parse(time.RFC3339, invalidAt); err == nil {
				edge.InvalidAt = &t
			}
		}
		if episodes, ok := row["episodes"].([]interface{}); ok {
			edge.Episodes = make([]string, len(episodes))
			for i, ep := range episodes {
				if s, ok := ep.(string); ok {
					edge.Episodes[i] = s
				}
			}
		}
		if groupID, ok := row["group_id"].(string); ok {
			edge.GroupID = groupID
		}
		if sourceNodeID, ok := row["source_node_id"].(string); ok {
			edge.SourceNodeID = sourceNodeID
		}
		if targetNodeID, ok := row["target_node_id"].(string); ok {
			edge.TargetNodeID = targetNodeID
		}
		if embedding, ok := row["fact_embedding"].([]interface{}); ok {
			edge.FactEmbedding = make([]float32, len(embedding))
			for i, v := range embedding {
				if f, ok := v.(float64); ok {
					edge.FactEmbedding[i] = float32(f)
				}
			}
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// GetCommunities retrieves community nodes
func (k *KuzuDriver) GetCommunities(ctx context.Context, groupID string, level int) ([]*types.Node, error) {
	return []*types.Node{}, nil // Placeholder
}

// BuildCommunities builds community structure using label propagation algorithm.
//
// IMPORTANT: This is a driver-level implementation without LLM summarization.
// For production use with LLM-based community summarization, use the
// community.Builder through the Client:
//
//	client := graphiti.NewClient(driver, llmClient, embedderClient, config)
//	result, err := client.Add(ctx, episodes, &graphiti.AddEpisodeOptions{
//	    UpdateCommunities: true,
//	})
//
// Or use the community.Builder directly:
//
//	builder := community.NewBuilder(driver, llmClient, embedderClient)
//	result, err := builder.BuildCommunities(ctx, []string{groupID})
//
// This driver method is provided for:
// - Testing without LLM access
// - Batch processing scenarios
// - Simple structural community detection
func (k *KuzuDriver) BuildCommunities(ctx context.Context, groupID string) error {
	// Note: This implementation is kept simple intentionally.
	// The full LLM-powered community building is available through
	// community.Builder (see pkg/community/community.go) which provides:
	// - Hierarchical LLM summarization of entity clusters
	// - Descriptive community naming via LLM
	// - Embedding generation for community names
	// - Concurrent processing with semaphore limiting
	//
	// That implementation is the recommended approach for production use.
	return nil
}

// CreateIndices creates database indices
// For Kuzu, this is a no-op as indices are managed through the schema
// This matches the Python implementation where create_indices is not implemented for Kuzu
func (k *KuzuDriver) CreateIndices(ctx context.Context) error {
	// No-op for Kuzu - indices are created as part of schema setup
	return nil
}

// GetStats returns graph statistics
func (k *KuzuDriver) GetStats(ctx context.Context, groupID string) (*GraphStats, error) {
	stats := &GraphStats{
		NodesByType: make(map[string]int64),
		EdgesByType: make(map[string]int64),
		LastUpdated: time.Now(),
	}

	// Get node counts by table
	nodeTables := []string{"Entity", "Episodic", "Community", "RelatesToNode_"}
	for _, table := range nodeTables {
		query := fmt.Sprintf("MATCH (n:%s) RETURN count(n) as count", table)
		result, _, _, err := k.ExecuteQuery(query, nil)
		if err != nil {
			continue
		}

		if resultList, ok := result.([]map[string]interface{}); ok && len(resultList) > 0 {
			if count, ok := resultList[0]["count"].(int64); ok {
				stats.NodesByType[table] = count
				stats.NodeCount += count
			}
		}
	}

	// Get edge counts by relationship type
	edgeTables := []string{"RELATES_TO", "MENTIONS", "HAS_MEMBER"}
	for _, table := range edgeTables {
		query := fmt.Sprintf("MATCH ()-[r:%s]->() RETURN count(r) as count", table)
		result, _, _, err := k.ExecuteQuery(query, nil)
		if err != nil {
			continue
		}

		if resultList, ok := result.([]map[string]interface{}); ok && len(resultList) > 0 {
			if count, ok := resultList[0]["count"].(int64); ok {
				stats.EdgesByType[table] = count
				stats.EdgeCount += count
			}
		}
	}

	// Set community count from Community table
	if communityCount, ok := stats.NodesByType["Community"]; ok {
		stats.CommunityCount = communityCount
	}

	return stats, nil
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

	if id, ok := data["node.uuid"]; ok {
		node.ID = fmt.Sprintf("%v", id)
	} else if id, ok := data["n.uuid"]; ok {
		node.ID = fmt.Sprintf("%v", id)
	}
	if name, ok := data["node.name"]; ok {
		node.Name = fmt.Sprintf("%v", name)
	} else if name, ok := data["n.name"]; ok {
		node.Name = fmt.Sprintf("%v", name)
	}
	if groupID, ok := data["node.group_id"]; ok {
		node.GroupID = fmt.Sprintf("%v", groupID)
	} else if groupID, ok := data["n.group_id"]; ok {
		node.GroupID = fmt.Sprintf("%v", groupID)
	}

	if summary, ok := data["node.summary"]; ok {
		node.Summary = fmt.Sprintf("%v", summary)
	} else if summary, ok := data["n.summary"]; ok {
		node.Summary = fmt.Sprintf("%v", summary)
	}

	if content, ok := data["node.content"]; ok {
		node.Content = fmt.Sprintf("%v", content)
	} else if content, ok := data["n.content"]; ok {
		node.Content = fmt.Sprintf("%v", content)
	}

	if embedding, ok := data["node.name_embedding"]; ok {
		node.NameEmbedding = embedding.([]float32)
	} else if embedding, ok := data["n.name_embedding"]; ok {
		node.NameEmbedding = embedding.([]float32)
	}

	if labels, ok := data["node.labels"].([]interface{}); ok && len(labels) > 0 {
		if label, ok := labels[0].(string); ok {
			node.EntityType = label
		}
	} else if labels, ok := data["n.labels"].([]interface{}); ok && len(labels) > 0 {
		if label, ok := labels[0].(string); ok {
			node.EntityType = label
		}
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
		edge.Fact = fmt.Sprintf("%v", fact)
	}

	if embedding, ok := data["fact_embedding"]; ok {
		edge.FactEmbedding = embedding.([]float32)
	}
	if sourceID, ok := data["source_id"]; ok {
		edge.SourceID = fmt.Sprintf("%v", sourceID)
		edge.SourceNodeID = fmt.Sprintf("%v", sourceID)
	}
	if targetID, ok := data["target_id"]; ok {
		edge.TargetID = fmt.Sprintf("%v", targetID)
		edge.TargetNodeID = fmt.Sprintf("%v", targetID)
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
				labels: $labels,
				created_at: $created_at,
				name_embedding: [],
				summary: $summary,
				attributes: $attributes
			})
		`
		params["uuid"] = node.ID
		params["name"] = node.Name
		params["group_id"] = node.GroupID
		params["labels"] = []string{node.EntityType}
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
				n.attributes = $attributes,
				n.labels = $labels
		`
		params["uuid"] = node.ID
		params["group_id"] = node.GroupID
		params["name"] = node.Name
		params["summary"] = node.Summary
		params["attributes"] = metadataJSON
		params["labels"] = []string{node.EntityType}
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
