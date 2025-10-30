package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// MemgraphDriver implements the GraphDriver interface for Memgraph databases.
// Memgraph is compatible with Neo4j's Bolt protocol and Cypher query language.
type MemgraphDriver struct {
	client   neo4j.DriverWithContext
	database string
}

// NewMemgraphDriver creates a new Memgraph driver instance.
func NewMemgraphDriver(uri, username, password, database string) (*MemgraphDriver, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create memgraph driver: %w", err)
	}

	if database == "" {
		database = "memgraph"
	}

	return &MemgraphDriver{
		client:   driver,
		database: database,
	}, nil
}

// GetNode retrieves a node by ID.
func (m *MemgraphDriver) GetNode(ctx context.Context, nodeID, groupID string) (*types.Node, error) {
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {uuid: $nodeID, group_id: $groupID})
			RETURN n
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"nodeID":  nodeID,
			"groupID": groupID,
		})
		if err != nil {
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			if err.Error() == "Result contains no more records" {
				return nil, fmt.Errorf("node not found")
			}
			return nil, err
		}

		return record, nil
	})
	if err != nil {
		return nil, err
	}

	record := result.(*db.Record)
	nodeValue, found := record.Get("n")
	if !found {
		return nil, fmt.Errorf("node not found")
	}

	node := nodeValue.(dbtype.Node)
	return m.nodeFromDBNode(node), nil
}

// NodeExists checks if a node exists in the database.
func (m *MemgraphDriver) NodeExists(ctx context.Context, node *types.Node) bool {
	// Handle nil node
	if node == nil {
		return false
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {uuid: $uuid, group_id: $group_id})
			RETURN n.uuid
			LIMIT 1
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"uuid":     node.ID,
			"group_id": node.GroupID,
		})
		if err != nil {
			return false, err
		}

		return res.Single(ctx)
	})

	if err != nil {
		return false
	}

	return result != nil
}

// getLabelForNodeType returns the appropriate node label for a given node type.
func (m *MemgraphDriver) getLabelForNodeType(nodeType types.NodeType) string {
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

// UpsertNode creates or updates a node.
func (m *MemgraphDriver) UpsertNode(ctx context.Context, node *types.Node) error {
	// Handle nil node
	if node == nil {
		return fmt.Errorf("cannot upsert nil node")
	}

	// Set timestamps if not already set
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	node.UpdatedAt = time.Now()
	if node.ValidFrom.IsZero() {
		node.ValidFrom = node.CreatedAt
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MERGE (n:$label {id: $id, group_id: $group_id})
			SET n += $properties
			SET n.updated_at = $updated_at
		`

		properties := m.nodeToProperties(node)
		_, err := tx.Run(ctx, query, map[string]any{
			"id":         node.ID,
			"group_id":   node.GroupID,
			"properties": properties,
			"updated_at": time.Now().Format(time.RFC3339),
			"label":      m.getLabelForNodeType(node.Type),
		})
		return nil, err
	})

	return err
}

// UpsertNode creates or updates a node.
func (m *MemgraphDriver) UpsertNode2(ctx context.Context, node *types.Node) error {
	// Handle nil node
	if node == nil {
		return fmt.Errorf("cannot upsert nil node")
	}

	// Set timestamps if not already set
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	node.UpdatedAt = time.Now()
	if node.ValidFrom.IsZero() {
		node.ValidFrom = node.CreatedAt
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)
	label := m.getLabelForNodeType(node.Type)

	// Try to create first
	if !m.NodeExists(ctx, node) {
		err := m.executeNodeCreateQuery(ctx, node, label)
		if err != nil {
			return fmt.Errorf("failed to create node %w", err)
		}
		return err

	}

	updateErr := m.executeNodeUpdateQuery(node, label)
	if updateErr != nil {
		return fmt.Errorf("failed to update node %w", updateErr)
	}

	return nil
}
func (m *MemgraphDriver) executeNodeUpdateQuery(node *types.Node, tableName string) error {
	// Defensive nil check for node
	if node == nil {
		return fmt.Errorf("cannot update nil node")
	}

	var metadataJSON string
	var err error
	if node.Metadata != nil && len(node.Metadata) > 0 {
		data, marshalErr := json.Marshal(node.Metadata)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal node metadata: %w", marshalErr)
		}
		metadataJSON = string(data)
	}

	var query string
	params := make(map[string]interface{})
	setClauses := []string{}

	params["uuid"] = node.ID
	params["group_id"] = node.GroupID

	switch tableName {
	case "Episodic":
		// Always update name, content, and valid_at for episodic nodes
		setClauses = append(setClauses, "n.name = $name")
		params["name"] = node.Name

		setClauses = append(setClauses, "n.content = $content")
		params["content"] = node.Content

		setClauses = append(setClauses, "n.valid_at = $valid_at")
		params["valid_at"] = node.ValidFrom

		// Update entity_edges - Memgraph handles empty arrays naturally
		setClauses = append(setClauses, "n.entity_edges = $entity_edges")
		if len(node.EntityEdges) > 0 {
			params["entity_edges"] = node.EntityEdges
		} else {
			params["entity_edges"] = []string{}
		}

	case "Entity":
		// Dynamically add SET clauses for non-empty fields
		if node.Name != "" {
			setClauses = append(setClauses, "n.name = $name")
			params["name"] = node.Name
		}
		if node.Summary != "" {
			setClauses = append(setClauses, "n.summary = $summary")
			params["summary"] = node.Summary
		}
		if metadataJSON != "" {
			setClauses = append(setClauses, "n.attributes = $attributes")
			params["attributes"] = metadataJSON
		}

		// Update labels - Memgraph handles empty arrays naturally
		setClauses = append(setClauses, "n.labels = $labels")
		if node.EntityType != "" {
			params["labels"] = []string{node.EntityType}
		} else {
			params["labels"] = []string{}
		}

		// Update name_embedding - Memgraph accepts float32 directly
		setClauses = append(setClauses, "n.name_embedding = $name_embedding")
		if len(node.NameEmbedding) > 0 {
			params["name_embedding"] = node.NameEmbedding
		} else {
			params["name_embedding"] = []float32{}
		}

	case "Community":
		// Dynamically add SET clauses for non-empty fields
		if node.Name != "" {
			setClauses = append(setClauses, "n.name = $name")
			params["name"] = node.Name
		}
		if node.Summary != "" {
			setClauses = append(setClauses, "n.summary = $summary")
			params["summary"] = node.Summary
		}

		// Update name_embedding - Memgraph accepts float32 directly
		setClauses = append(setClauses, "n.name_embedding = $name_embedding")
		if len(node.NameEmbedding) > 0 {
			params["name_embedding"] = node.NameEmbedding
		} else {
			params["name_embedding"] = []float32{}
		}

	default:
		return fmt.Errorf("unknown table: %s", tableName)
	}

	// Only execute query if there are fields to update
	if len(setClauses) == 0 {
		return nil // Nothing to update
	}

	query = fmt.Sprintf(`
          MATCH (n:%s)
          WHERE n.uuid = $uuid AND n.group_id = $group_id
          SET %s
      `, tableName, strings.Join(setClauses, ", "))

	_, _, _, err = m.ExecuteQuery(query, params)
	return err
}
func (m *MemgraphDriver) executeNodeCreateQuery(ctx context.Context, node *types.Node, tableName string) error {
	if node == nil {
		return fmt.Errorf("cannot create nil node")
	}

	// Create a new session (always close after use)
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	// Build node properties
	properties := make(map[string]interface{})

	// Common fields
	properties["uuid"] = node.ID
	properties["name"] = node.Name
	properties["group_id"] = node.GroupID
	properties["created_at"] = node.CreatedAt
	properties["updated_at"] = time.Now().Format(time.RFC3339)

	switch tableName {
	case "Episodic":
		properties["source"] = string(node.EpisodeType)
		properties["source_description"] = ""
		properties["content"] = node.Content
		properties["valid_at"] = node.ValidFrom

		if len(node.EntityEdges) > 0 {
			properties["entity_edges"] = node.EntityEdges
		} else {
			properties["entity_edges"] = []string{}
		}

	case "Entity":
		// Convert Metadata → map
		if node.Metadata != nil {
			data, err := json.Marshal(node.Metadata)
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			var metadata map[string]interface{}
			if err := json.Unmarshal(data, &metadata); err != nil {
				return fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
			properties["attributes"] = metadata
		} else {
			properties["attributes"] = map[string]interface{}{}
		}

		// Handle labels
		if node.EntityType != "" {
			properties["labels"] = []string{node.EntityType}
		} else {
			properties["labels"] = []string{}
		}

		properties["summary"] = node.Summary

		// Convert []float32 → []float64 for Memgraph compatibility
		embedding := make([]float64, len(node.NameEmbedding))
		for i, v := range node.NameEmbedding {
			embedding[i] = float64(v)
		}
		properties["name_embedding"] = embedding

	case "Community":
		properties["summary"] = node.Summary
		embedding := make([]float64, len(node.NameEmbedding))
		for i, v := range node.NameEmbedding {
			embedding[i] = float64(v)
		}
		properties["name_embedding"] = embedding

	default:
		return fmt.Errorf("unknown table: %s", tableName)
	}

	// Build query dynamically based on label (tableName)
	query := fmt.Sprintf(`
		MERGE (n:%s {uuid: $uuid, group_id: $group_id})
		SET n += $properties
	`, tableName)

	// Execute in write transaction
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, map[string]any{
			"uuid":       node.ID,
			"group_id":   node.GroupID,
			"properties": properties,
		})
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("failed to execute write for %s: %w", tableName, err)
	}

	return nil
}

// DeleteNode removes a node and its edges.
func (m *MemgraphDriver) DeleteNode(ctx context.Context, nodeID, groupID string) error {
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {uuid: $nodeID, group_id: $groupID})
			DETACH DELETE n
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"nodeID":  nodeID,
			"groupID": groupID,
		})
		return nil, err
	})

	return err
}

// GetNodes retrieves multiple nodes by their IDs.
func (m *MemgraphDriver) GetNodes(ctx context.Context, nodeIDs []string, groupID string) ([]*types.Node, error) {
	if len(nodeIDs) == 0 {
		return []*types.Node{}, nil
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {group_id: $groupID})
			WHERE n.uuid IN $nodeIDs
			RETURN n
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"nodeIDs": nodeIDs,
			"groupID": groupID,
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	nodes := make([]*types.Node, 0, len(records))

	for _, record := range records {
		nodeValue, found := record.Get("n")
		if !found {
			continue
		}
		node := nodeValue.(dbtype.Node)
		nodes = append(nodes, m.nodeFromDBNode(node))
	}

	return nodes, nil
}

// GetEdge retrieves an edge by ID.
func (m *MemgraphDriver) GetEdge(ctx context.Context, edgeID, groupID string) (*types.Edge, error) {
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (s)-[r {uuid: $edgeID, group_id: $groupID}]->(t)
			RETURN r, s.uuid as source_id, t.uuid as target_id
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"edgeID":  edgeID,
			"groupID": groupID,
		})
		if err != nil {
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			if err.Error() == "Result contains no more records" {
				return nil, fmt.Errorf("edge not found")
			}
			return nil, err
		}

		return record, nil
	})
	if err != nil {
		return nil, err
	}

	record := result.(*db.Record)
	relationValue, found := record.Get("r")
	if !found {
		return nil, fmt.Errorf("edge not found")
	}

	relation := relationValue.(dbtype.Relationship)
	sourceID, _ := record.Get("source_id")
	targetID, _ := record.Get("target_id")

	return m.edgeFromDBRelation(relation, sourceID.(string), targetID.(string)), nil
}

// EdgeExists checks if an edge exists in the database.
func (m *MemgraphDriver) EdgeExists(ctx context.Context, edge *types.Edge) bool {
	// Handle nil edge
	if edge == nil {
		return false
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH ()-[r {uuid: $id, group_id: $group_id}]-()
			RETURN r.uuid
			LIMIT 1
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"id":       edge.ID,
			"group_id": edge.GroupID,
		})
		if err != nil {
			return false, err
		}

		return res.Single(ctx)
	})

	if err != nil {
		return false
	}

	return result != nil
}

// UpsertEdge creates or updates an edge.
func (m *MemgraphDriver) UpsertEdge(ctx context.Context, edge *types.Edge) error {
	// Ensure timestamps are initialized
	if edge.CreatedAt.IsZero() {
		edge.CreatedAt = time.Now()
	}
	edge.UpdatedAt = time.Now()

	if edge.ValidFrom.IsZero() {
		edge.ValidFrom = edge.CreatedAt
	}

	// Check if the edge already exists in Memgraph
	if !m.EdgeExists(ctx, edge) {
		// Create a new edge if not found
		if err := m.executeEdgeCreateQuery(edge); err != nil {
			return fmt.Errorf("failed to create edge: %w", err)
		}
		return nil
	}

	// Otherwise, update the existing edge
	if err := m.executeEdgeUpdateQuery(edge); err != nil {
		return fmt.Errorf("failed to update edge: %w", err)
	}

	return nil
}

func (m *MemgraphDriver) executeEdgeCreateQuery(edge *types.Edge) error {
	var metadataJSON string
	if edge.Metadata != nil {
		if data, err := json.Marshal(edge.Metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	params := make(map[string]interface{})

	// Handle fact_embedding
	if len(edge.FactEmbedding) > 0 {
		// Memgraph supports list parameters directly, no need for CAST
		embedding := make([]float64, len(edge.FactEmbedding))
		for i, v := range edge.FactEmbedding {
			embedding[i] = float64(v)
		}
		params["fact_embedding"] = embedding
	} else {
		// Provide an empty list directly
		params["fact_embedding"] = []float64{}
	}

	// Handle episodes
	if len(edge.Episodes) > 0 {
		params["episodes"] = edge.Episodes
	} else {
		params["episodes"] = []string{}
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
			fact_embedding: $fact_embedding,
			episodes: $episodes,
			expired_at: $expired_at,
			valid_at: $valid_at,
			invalid_at: $invalid_at,
			attributes: $attributes
		})
		CREATE (a)-[:RELATES_TO]->(rel)
		CREATE (rel)-[:RELATES_TO]->(b)
		RETURN rel
	`

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

	_, _, _, err := m.ExecuteQuery(query, params)
	return err
}

func (m *MemgraphDriver) executeEdgeUpdateQuery(edge *types.Edge) error {
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
		RETURN rel
	`

	params := map[string]interface{}{
		"uuid":       edge.ID,
		"group_id":   edge.GroupID,
		"name":       edge.Name,
		"fact":       edge.Fact,
		"attributes": metadataJSON,
		"valid_at":   edge.ValidFrom,
	}

	if edge.ValidTo != nil {
		params["expired_at"] = edge.ValidTo
		params["invalid_at"] = edge.ValidTo
	} else {
		params["expired_at"] = nil
		params["invalid_at"] = nil
	}

	_, _, _, err := m.ExecuteQuery(query, params)
	return err
}

// DeleteEdge removes an edge.
func (m *MemgraphDriver) DeleteEdge(ctx context.Context, edgeID, groupID string) error {
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH ()-[r {uuid: $edgeID, group_id: $groupID}]-()
			DELETE r
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"edgeID":  edgeID,
			"groupID": groupID,
		})
		return nil, err
	})

	return err
}

// GetEdges retrieves multiple edges by their IDs.
func (m *MemgraphDriver) GetEdges(ctx context.Context, edgeIDs []string, groupID string) ([]*types.Edge, error) {
	if len(edgeIDs) == 0 {
		return []*types.Edge{}, nil
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (s)-[r {group_id: $groupID}]->(t)
			WHERE r.uuid IN $edgeIDs
			RETURN r, s.uuid as source_id, t.uuid as target_id
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"edgeIDs": edgeIDs,
			"groupID": groupID,
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	edges := make([]*types.Edge, 0, len(records))

	for _, record := range records {
		relationValue, found := record.Get("r")
		if !found {
			continue
		}
		relation := relationValue.(dbtype.Relationship)
		sourceID, _ := record.Get("source_id")
		targetID, _ := record.Get("target_id")

		edges = append(edges, m.edgeFromDBRelation(relation, sourceID.(string), targetID.(string)))
	}

	return edges, nil
}

// GetNeighbors retrieves neighboring nodes within a specified distance
func (m *MemgraphDriver) GetNeighbors(ctx context.Context, nodeID, groupID string, maxDistance int) ([]*types.Node, error) {
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := fmt.Sprintf(`
			MATCH (start {uuid: $nodeID, group_id: $groupID})
			MATCH (start)-[*1..%d]-(neighbor)
			WHERE neighbor.group_id = $groupID AND neighbor.id <> $nodeID
			RETURN DISTINCT neighbor
		`, maxDistance)

		res, err := tx.Run(ctx, query, map[string]any{
			"nodeID":  nodeID,
			"groupID": groupID,
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	nodes := make([]*types.Node, 0, len(records))

	for _, record := range records {
		nodeValue, found := record.Get("neighbor")
		if !found {
			continue
		}
		node := nodeValue.(dbtype.Node)
		nodes = append(nodes, m.nodeFromDBNode(node))
	}

	return nodes, nil
}

func (m *MemgraphDriver) GetRelatedNodes(ctx context.Context, nodeID, groupID string, edgeTypes []types.EdgeType) ([]*types.Node, error) {
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		var query string
		params := map[string]any{
			"nodeID":  nodeID,
			"groupID": groupID,
		}

		if len(edgeTypes) == 0 {
			// Get all related nodes regardless of edge type
			query = `
				MATCH (start {uuid: $nodeID, group_id: $groupID})
				MATCH (start)-[r]-(related)
				WHERE related.group_id = $groupID AND related.id <> $nodeID
				RETURN DISTINCT related
			`
		} else {
			// Filter by specific edge types
			edgeTypeStrings := make([]string, len(edgeTypes))
			for i, edgeType := range edgeTypes {
				edgeTypeStrings[i] = string(edgeType)
			}
			params["edgeTypes"] = edgeTypeStrings

			query = `
				MATCH (start {uuid: $nodeID, group_id: $groupID})
				MATCH (start)-[r]-(related)
				WHERE related.group_id = $groupID
				  AND related.id <> $nodeID
				  AND r.type IN $edgeTypes
				RETURN DISTINCT related
			`
		}

		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	nodes := make([]*types.Node, 0, len(records))

	for _, record := range records {
		nodeValue, found := record.Get("related")
		if !found {
			continue
		}
		node := nodeValue.(dbtype.Node)
		nodes = append(nodes, m.nodeFromDBNode(node))
	}

	return nodes, nil
}

func (m *MemgraphDriver) SearchNodesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Node, error) {
	if len(embedding) == 0 {
		return []*types.Node{}, nil
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	// Get all nodes with embeddings and compute similarity in-memory
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {group_id: $groupID})
			WHERE n.embedding IS NOT NULL
			RETURN n
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"groupID": groupID,
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	type nodeWithSimilarity struct {
		node       *types.Node
		similarity float32
	}

	var candidates []nodeWithSimilarity

	for _, record := range records {
		nodeValue, found := record.Get("n")
		if !found {
			continue
		}
		dbNode := nodeValue.(dbtype.Node)
		node := m.nodeFromDBNode(dbNode)

		// Parse embedding from JSON
		if embeddingStr, ok := dbNode.Props["embedding"].(string); ok {
			var nodeEmbedding []float32
			if err := json.Unmarshal([]byte(embeddingStr), &nodeEmbedding); err == nil {
				similarity := m.cosineSimilarity(embedding, nodeEmbedding)
				candidates = append(candidates, nodeWithSimilarity{
					node:       node,
					similarity: similarity,
				})
			}
		}
	}

	// Sort by similarity (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].similarity > candidates[j].similarity
	})

	// Apply limit
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}

	// Extract nodes
	nodes := make([]*types.Node, len(candidates))
	for i, candidate := range candidates {
		nodes[i] = candidate.node
	}

	return nodes, nil
}

func (m *MemgraphDriver) SearchEdgesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Edge, error) {
	if len(embedding) == 0 {
		return []*types.Edge{}, nil
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	// Get all edges with embeddings and compute similarity in-memory
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (s)-[r {group_id: $groupID}]->(t)
			WHERE r.embedding IS NOT NULL
			RETURN r, s.id as source_id, t.id as target_id
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"groupID": groupID,
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	type edgeWithSimilarity struct {
		edge       *types.Edge
		similarity float32
	}

	var candidates []edgeWithSimilarity

	for _, record := range records {
		relationValue, found := record.Get("r")
		if !found {
			continue
		}
		dbRelation := relationValue.(dbtype.Relationship)
		sourceID, _ := record.Get("source_id")
		targetID, _ := record.Get("target_id")
		edge := m.edgeFromDBRelation(dbRelation, sourceID.(string), targetID.(string))

		// Parse embedding from JSON
		if embeddingStr, ok := dbRelation.Props["embedding"].(string); ok {
			var edgeEmbedding []float32
			if err := json.Unmarshal([]byte(embeddingStr), &edgeEmbedding); err == nil {
				similarity := m.cosineSimilarity(embedding, edgeEmbedding)
				candidates = append(candidates, edgeWithSimilarity{
					edge:       edge,
					similarity: similarity,
				})
			}
		}
	}

	// Sort by similarity (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].similarity > candidates[j].similarity
	})

	// Apply limit
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}

	// Extract edges
	edges := make([]*types.Edge, len(candidates))
	for i, candidate := range candidates {
		edges[i] = candidate.edge
	}

	return edges, nil
}

func (m *MemgraphDriver) UpsertNodes(ctx context.Context, nodes []*types.Node) error {
	if len(nodes) == 0 {
		return nil
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	// Use a transaction to batch the operations
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, node := range nodes {
			query := `
				MERGE (n {uuid: $id, group_id: $group_id})
				SET n += $properties
				SET n.updated_at = $updated_at
			`

			properties := m.nodeToProperties(node)
			_, err := tx.Run(ctx, query, map[string]any{
				"id":         node.ID,
				"group_id":   node.GroupID,
				"properties": properties,
				"updated_at": time.Now().Format(time.RFC3339),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to upsert node %s: %w", node.ID, err)
			}
		}
		return nil, nil
	})

	return err
}

func (m *MemgraphDriver) UpsertEdges(ctx context.Context, edges []*types.Edge) error {
	if len(edges) == 0 {
		return nil
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	// Use a transaction to batch the operations
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, edge := range edges {
			query := `
				MATCH (s {uuid: $source_id, group_id: $group_id})
				MATCH (t {uuid: $target_id, group_id: $group_id})
				MERGE (s)-[r:RELATES {uuid: $id, group_id: $group_id}]->(t)
				SET r += $properties
				SET r.updated_at = $updated_at
			`

			properties := m.edgeToProperties(edge)
			_, err := tx.Run(ctx, query, map[string]any{
				"id":         edge.ID,
				"source_id":  edge.SourceID,
				"target_id":  edge.TargetID,
				"group_id":   edge.GroupID,
				"properties": properties,
				"updated_at": time.Now().Format(time.RFC3339),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to upsert edge %s: %w", edge.ID, err)
			}
		}
		return nil, nil
	})

	return err
}

func (m *MemgraphDriver) GetNodesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Node, error) {
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {group_id: $groupID})
			WHERE n.created_at >= $start AND n.created_at <= $end
			RETURN n
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"groupID": groupID,
			"start":   start.Format(time.RFC3339),
			"end":     end.Format(time.RFC3339),
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	nodes := make([]*types.Node, 0, len(records))

	for _, record := range records {
		nodeValue, found := record.Get("n")
		if !found {
			continue
		}
		node := nodeValue.(dbtype.Node)
		nodes = append(nodes, m.nodeFromDBNode(node))
	}

	return nodes, nil
}

func (m *MemgraphDriver) GetEdgesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Edge, error) {
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (s)-[r {group_id: $groupID}]->(t)
			WHERE r.created_at >= $start AND r.created_at <= $end
			RETURN r, s.id as source_id, t.id as target_id
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"groupID": groupID,
			"start":   start.Format(time.RFC3339),
			"end":     end.Format(time.RFC3339),
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	edges := make([]*types.Edge, 0, len(records))

	for _, record := range records {
		relationValue, found := record.Get("r")
		if !found {
			continue
		}
		relation := relationValue.(dbtype.Relationship)
		sourceID, _ := record.Get("source_id")
		targetID, _ := record.Get("target_id")

		edges = append(edges, m.edgeFromDBRelation(relation, sourceID.(string), targetID.(string)))
	}

	return edges, nil
}

func (m *MemgraphDriver) GetCommunities(ctx context.Context, groupID string, level int) ([]*types.Node, error) {
	// For basic implementation, return nodes grouped by a hypothetical community property
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {group_id: $groupID})
			WHERE n.community_level = $level
			RETURN n
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"groupID": groupID,
			"level":   level,
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	nodes := make([]*types.Node, 0, len(records))

	for _, record := range records {
		nodeValue, found := record.Get("n")
		if !found {
			continue
		}
		node := nodeValue.(dbtype.Node)
		nodes = append(nodes, m.nodeFromDBNode(node))
	}

	return nodes, nil
}

func (m *MemgraphDriver) BuildCommunities(ctx context.Context, groupID string) error {
	// Basic implementation that assigns community IDs based on connected components
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Reset existing community assignments
		resetQuery := `
			MATCH (n {group_id: $groupID})
			REMOVE n.community_id, n.community_level
		`
		_, err := tx.Run(ctx, resetQuery, map[string]any{"groupID": groupID})
		if err != nil {
			return nil, err
		}

		// Simple community detection using connected components
		communityQuery := `
			MATCH (n {group_id: $groupID})
			OPTIONAL MATCH (n)-[*]-(connected {group_id: $groupID})
			WITH n, collect(DISTINCT connected.id) + [n.uuid] as component
			SET n.community_id = component[0]
			SET n.community_level = 0
		`
		_, err = tx.Run(ctx, communityQuery, map[string]any{"groupID": groupID})
		return nil, err
	})

	return err
}

func (m *MemgraphDriver) GetExistingCommunity(ctx context.Context, entityUUID string) (*types.Node, error) {
	query := `
		MATCH (e:Entity {uuid: $entity_uuid})-[:MEMBER_OF]->(c:Community)
		RETURN c
		LIMIT 1
	`

	params := map[string]interface{}{
		"entity_uuid": entityUUID,
	}

	result, _, _, err := m.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing community: %w", err)
	}

	// Parse result - expecting Neo4j record format
	nodes, err := m.parseCommunityNodesFromRecords(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse existing community: %w", err)
	}

	if len(nodes) > 0 {
		return nodes[0], nil
	}

	return nil, nil
}

func (m *MemgraphDriver) FindModalCommunity(ctx context.Context, entityUUID string) (*types.Node, error) {
	query := `
		MATCH (e:Entity {uuid: $entity_uuid})-[:RELATES_TO]-(rel)-[:RELATES_TO]-(neighbor:Entity)
		MATCH (neighbor)-[:MEMBER_OF]->(c:Community)
		WITH c, count(*) AS count
		ORDER BY count DESC
		LIMIT 1
		RETURN c
	`

	params := map[string]interface{}{
		"entity_uuid": entityUUID,
	}

	result, _, _, err := m.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query modal community: %w", err)
	}

	// Parse result - expecting Neo4j record format
	nodes, err := m.parseCommunityNodesFromRecords(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse modal community: %w", err)
	}

	if len(nodes) > 0 {
		return nodes[0], nil
	}

	return nil, nil
}

// parseCommunityNodesFromRecords parses community nodes from Neo4j/Memgraph records
func (m *MemgraphDriver) parseCommunityNodesFromRecords(result interface{}) ([]*types.Node, error) {
	var nodes []*types.Node

	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %T", result)
	}

	for i := 0; i < value.Len(); i++ {
		record := value.Index(i)

		getMethod := record.MethodByName("Get")
		if !getMethod.IsValid() {
			continue
		}

		// Get the community node
		results := getMethod.Call([]reflect.Value{reflect.ValueOf("c")})
		if len(results) < 1 {
			continue
		}

		nodeInterface := results[0].Interface()

		// Convert to Node - this will need to extract properties from the Neo4j node
		node := &types.Node{
			Type: types.CommunityNodeType,
		}

		// Use reflection to get node properties
		nodeValue := reflect.ValueOf(nodeInterface)
		if nodeValue.Kind() == reflect.Ptr {
			nodeValue = nodeValue.Elem()
		}

		// Try to get Props method
		propsMethod := nodeValue.MethodByName("Props")
		if !propsMethod.IsValid() {
			propsMethod = nodeValue.MethodByName("Properties")
		}

		if propsMethod.IsValid() {
			propsResults := propsMethod.Call(nil)
			if len(propsResults) > 0 {
				if props, ok := propsResults[0].Interface().(map[string]interface{}); ok {
					if uuid, ok := props["uuid"].(string); ok {
						node.ID = uuid
					}
					if name, ok := props["name"].(string); ok {
						node.Name = name
					}
					if summary, ok := props["summary"].(string); ok {
						node.Summary = summary
					}
				}
			}
		}

		if node.ID != "" {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

func (m *MemgraphDriver) CreateIndices(ctx context.Context) error {
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	// Create indices for commonly queried properties
	// Note: Memgraph requires index creation to use auto-commit (implicit) transactions
	// rather than explicit transactions. Using session.Run() directly instead of ExecuteWrite.
	indices := []string{
		"CREATE INDEX ON :Entity(uuid, group_id)",
		"CREATE INDEX ON :Episodic(uuid, group_id)",
		"CREATE INDEX ON :Community(uuid, group_id)",
		"CREATE INDEX ON :Entity(created_at)",
		"CREATE INDEX ON :Episodic(created_at)",
		"CREATE INDEX ON :Community(created_at)",
	}

	for _, indexQuery := range indices {
		_, err := session.Run(ctx, indexQuery, nil)
		if err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
		}
	}

	return nil
}

func (m *MemgraphDriver) GetStats(ctx context.Context, groupID string) (*GraphStats, error) {
	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Get node count and types
		nodeQuery := `
			MATCH (n {group_id: $groupID})
			WITH n.type as node_type, count(*) as node_count
			RETURN node_type, node_count
			ORDER BY node_type
		`
		nodeRes, err := tx.Run(ctx, nodeQuery, map[string]any{"groupID": groupID})
		if err != nil {
			return nil, err
		}
		nodeRecords, err := nodeRes.Collect(ctx)
		if err != nil {
			return nil, err
		}

		// Get edge count and types
		edgeQuery := `
			MATCH ()-[r {group_id: $groupID}]-()
			WITH r.type as edge_type, count(*) as edge_count
			RETURN edge_type, edge_count
			ORDER BY edge_type
		`
		edgeRes, err := tx.Run(ctx, edgeQuery, map[string]any{"groupID": groupID})
		if err != nil {
			return nil, err
		}
		edgeRecords, err := edgeRes.Collect(ctx)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"nodes": nodeRecords,
			"edges": edgeRecords,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	data := result.(map[string]interface{})
	nodeRecords := data["nodes"].([]*db.Record)
	edgeRecords := data["edges"].([]*db.Record)

	stats := &GraphStats{
		NodesByType: make(map[string]int64),
		EdgesByType: make(map[string]int64),
		LastUpdated: time.Now(),
	}

	// Process node stats
	for _, record := range nodeRecords {
		if nodeCount, found := record.Get("node_count"); found {
			stats.NodeCount += nodeCount.(int64)
		}
		if nodeType, found := record.Get("node_type"); found && nodeType != nil {
			if nodeCount, found := record.Get("node_count"); found {
				stats.NodesByType[nodeType.(string)] = nodeCount.(int64)
			}
		}
	}

	// Process edge stats
	for _, record := range edgeRecords {
		if edgeCount, found := record.Get("edge_count"); found {
			stats.EdgeCount += edgeCount.(int64)
		}
		if edgeType, found := record.Get("edge_type"); found && edgeType != nil {
			if edgeCount, found := record.Get("edge_count"); found {
				stats.EdgesByType[edgeType.(string)] = edgeCount.(int64)
			}
		}
	}

	return stats, nil
}

// SearchNodes performs text-based search on nodes
func (m *MemgraphDriver) SearchNodes(ctx context.Context, query, groupID string, options *SearchOptions) ([]*types.Node, error) {
	if query == "" {
		return []*types.Node{}, nil
	}

	limit := 10
	if options != nil && options.Limit > 0 {
		limit = options.Limit
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Basic text search using CONTAINS
		searchQuery := `
			MATCH (n {group_id: $groupID})
			WHERE n.name CONTAINS $query OR n.summary CONTAINS $query OR n.content CONTAINS $query
			RETURN n
			LIMIT $limit
		`
		res, err := tx.Run(ctx, searchQuery, map[string]any{
			"groupID": groupID,
			"query":   query,
			"limit":   limit,
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	nodes := make([]*types.Node, 0, len(records))

	for _, record := range records {
		nodeValue, found := record.Get("n")
		if !found {
			continue
		}
		node := nodeValue.(dbtype.Node)
		nodes = append(nodes, m.nodeFromDBNode(node))
	}

	return nodes, nil
}

// SearchEdges performs text-based search on edges
func (m *MemgraphDriver) SearchEdges(ctx context.Context, query, groupID string, options *SearchOptions) ([]*types.Edge, error) {
	if query == "" {
		return []*types.Edge{}, nil
	}

	limit := 10
	if options != nil && options.Limit > 0 {
		limit = options.Limit
	}

	session := m.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Basic text search using CONTAINS
		searchQuery := `
			MATCH (s)-[r {group_id: $groupID}]->(t)
			WHERE r.name CONTAINS $query OR r.summary CONTAINS $query
			RETURN r, s.id as source_id, t.id as target_id
			LIMIT $limit
		`
		res, err := tx.Run(ctx, searchQuery, map[string]any{
			"groupID": groupID,
			"query":   query,
			"limit":   limit,
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		return records, err
	})
	if err != nil {
		return nil, err
	}

	records := result.([]*db.Record)
	edges := make([]*types.Edge, 0, len(records))

	for _, record := range records {
		relationValue, found := record.Get("r")
		if !found {
			continue
		}
		relation := relationValue.(dbtype.Relationship)
		sourceID, _ := record.Get("source_id")
		targetID, _ := record.Get("target_id")

		edges = append(edges, m.edgeFromDBRelation(relation, sourceID.(string), targetID.(string)))
	}

	return edges, nil
}

// SearchNodesByVector performs vector similarity search on nodes
func (m *MemgraphDriver) SearchNodesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Node, error) {
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

	// Use the existing SearchNodesByEmbedding method for compatibility
	// Filter by minimum score if needed
	nodes, err := m.SearchNodesByEmbedding(ctx, vector, groupID, limit)
	if err != nil {
		return nil, err
	}

	// Apply minimum score filter if specified
	if minScore > 0 {
		var filteredNodes []*types.Node
		for _, node := range nodes {
			if len(node.Embedding) > 0 {
				similarity := m.cosineSimilarity(vector, node.Embedding)
				if float64(similarity) >= minScore {
					filteredNodes = append(filteredNodes, node)
				}
			}
		}
		nodes = filteredNodes
	}

	return nodes, nil
}

// SearchEdgesByVector performs vector similarity search on edges
func (m *MemgraphDriver) SearchEdgesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Edge, error) {
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

	// Use the existing SearchEdgesByEmbedding method for compatibility
	// Filter by minimum score if needed
	edges, err := m.SearchEdgesByEmbedding(ctx, vector, groupID, limit)
	if err != nil {
		return nil, err
	}

	// Apply minimum score filter if specified
	if minScore > 0 {
		var filteredEdges []*types.Edge
		for _, edge := range edges {
			if len(edge.Embedding) > 0 {
				similarity := m.cosineSimilarity(vector, edge.Embedding)
				if float64(similarity) >= minScore {
					filteredEdges = append(filteredEdges, edge)
				}
			}
		}
		edges = filteredEdges
	}

	return edges, nil
}

// ExecuteQuery executes a Cypher query and returns records, summary, and keys (matching Python interface).
func (m *MemgraphDriver) ExecuteQuery(cypherQuery string, kwargs map[string]interface{}) (interface{}, interface{}, interface{}, error) {
	session := m.client.NewSession(context.Background(), neo4j.SessionConfig{DatabaseName: m.database})
	defer session.Close(context.Background())

	result, err := session.Run(context.Background(), cypherQuery, kwargs)
	if err != nil {
		return nil, nil, nil, err
	}

	records, err := result.Collect(context.Background())
	if err != nil {
		return nil, nil, nil, err
	}

	summary, err := result.Consume(context.Background())
	if err != nil {
		return nil, nil, nil, err
	}
	keys, err := result.Keys()
	if err != nil {
		return nil, nil, nil, err
	}

	return records, summary, keys, nil
}

// Session creates a new database session.
func (m *MemgraphDriver) Session(database *string) GraphDriverSession {
	dbName := m.database
	if database != nil {
		dbName = *database
	}
	return &MemgraphDriverSession{
		driver:   m,
		database: dbName,
	}
}

// DeleteAllIndexes deletes all indexes in the specified database.
func (m *MemgraphDriver) DeleteAllIndexes(database string) {
	// Implementation for deleting indexes
	session := m.client.NewSession(context.Background(), neo4j.SessionConfig{DatabaseName: database})
	defer session.Close(context.Background())

	// Get all indexes (Memgraph syntax)
	result, err := session.Run(context.Background(), "SHOW INDEX INFO", nil)
	if err != nil {
		return
	}

	records, err := result.Collect(context.Background())
	if err != nil {
		return
	}

	// Drop each index
	for _, record := range records {
		if label, ok := record.Values[0].(string); ok {
			if property, ok := record.Values[1].(string); ok {
				session.Run(context.Background(), fmt.Sprintf("DROP INDEX ON :%s(%s)", label, property), nil)
			}
		}
	}
}

// Provider returns the provider type.
func (m *MemgraphDriver) Provider() GraphProvider {
	return GraphProviderMemgraph
}

// GetAossClient returns nil for Memgraph (Amazon OpenSearch not applicable).
func (m *MemgraphDriver) GetAossClient() interface{} {
	return nil
}

// Close closes the Memgraph driver.
func (m *MemgraphDriver) Close() error {
	return m.client.Close(context.Background())
}

// VerifyConnectivity checks if the driver can connect to the database.
func (m *MemgraphDriver) VerifyConnectivity(ctx context.Context) error {
	return m.client.VerifyConnectivity(ctx)
}

// MemgraphDriverSession implements GraphDriverSession for Memgraph.
type MemgraphDriverSession struct {
	driver   *MemgraphDriver
	database string
	session  neo4j.SessionWithContext
}

// Enter implements the context manager pattern.
func (s *MemgraphDriverSession) Enter(ctx context.Context) (GraphDriverSession, error) {
	s.session = s.driver.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	return s, nil
}

// Exit implements the context manager pattern.
func (s *MemgraphDriverSession) Exit(ctx context.Context, excType, excVal, excTb interface{}) error {
	if s.session != nil {
		return s.session.Close(ctx)
	}
	return nil
}

// Close closes the session.
func (s *MemgraphDriverSession) Close() error {
	if s.session != nil {
		return s.session.Close(context.Background())
	}
	return nil
}

// Run executes a query in this session.
func (s *MemgraphDriverSession) Run(ctx context.Context, query interface{}, kwargs map[string]interface{}) error {
	if s.session == nil {
		return fmt.Errorf("session not entered")
	}

	queryStr, ok := query.(string)
	if !ok {
		return fmt.Errorf("query must be a string")
	}

	_, err := s.session.Run(ctx, queryStr, kwargs)
	return err
}

// ExecuteWrite executes a write transaction.
func (s *MemgraphDriverSession) ExecuteWrite(ctx context.Context, fn func(context.Context, GraphDriverSession, ...interface{}) (interface{}, error), args ...interface{}) (interface{}, error) {
	if s.session == nil {
		return nil, fmt.Errorf("session not entered")
	}

	return s.session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return fn(ctx, s, args...)
	})
}

// Provider returns the provider type.
func (s *MemgraphDriverSession) Provider() GraphProvider {
	return GraphProviderMemgraph
}

// Helper methods for converting between Graphiti and Memgraph types

func (m *MemgraphDriver) nodeFromDBNode(node dbtype.Node) *types.Node {
	props := node.Props

	result := &types.Node{}

	// Core fields
	if id, ok := props["id"].(string); ok {
		result.ID = id
	}
	if name, ok := props["name"].(string); ok {
		result.Name = name
	}
	if nodeType, ok := props["type"].(string); ok {
		result.Type = types.NodeType(nodeType)
	}
	if groupID, ok := props["group_id"].(string); ok {
		result.GroupID = groupID
	}

	// Timestamps
	if createdAtStr, ok := props["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			result.CreatedAt = t
		}
	}
	if updatedAtStr, ok := props["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			result.UpdatedAt = t
		}
	}

	// Temporal fields
	if validFromStr, ok := props["valid_from"].(string); ok {
		if t, err := time.Parse(time.RFC3339, validFromStr); err == nil {
			result.ValidFrom = t
		}
	}
	if validToStr, ok := props["valid_to"].(string); ok {
		if t, err := time.Parse(time.RFC3339, validToStr); err == nil {
			result.ValidTo = &t
		}
	}

	// Content fields
	if entityType, ok := props["entity_type"].(string); ok {
		result.EntityType = entityType
	}
	if summary, ok := props["summary"].(string); ok {
		result.Summary = summary
	}
	if content, ok := props["content"].(string); ok {
		result.Content = content
	}
	if refStr, ok := props["reference"].(string); ok {
		if t, err := time.Parse(time.RFC3339, refStr); err == nil {
			result.Reference = t
		}
	}
	if level, ok := props["level"].(int64); ok {
		result.Level = int(level)
	}

	// Episode-specific fields
	if episodeType, ok := props["episode_type"].(string); ok {
		result.EpisodeType = types.EpisodeType(episodeType)
	}
	if entityEdgesJSON, ok := props["entity_edges"].(string); ok {
		var entityEdges []string
		if err := json.Unmarshal([]byte(entityEdgesJSON), &entityEdges); err == nil {
			result.EntityEdges = entityEdges
		}
	}

	// Embeddings
	if nameEmbeddingJSON, ok := props["name_embedding"].(string); ok {
		var embedding []float32
		if err := json.Unmarshal([]byte(nameEmbeddingJSON), &embedding); err == nil {
			result.NameEmbedding = embedding
		}
	}
	if embeddingJSON, ok := props["embedding"].(string); ok {
		var embedding []float32
		if err := json.Unmarshal([]byte(embeddingJSON), &embedding); err == nil {
			result.Embedding = embedding
		}
	}

	// Source tracking
	if sourceIDsJSON, ok := props["source_ids"].(string); ok {
		var sourceIDs []string
		if err := json.Unmarshal([]byte(sourceIDsJSON), &sourceIDs); err == nil {
			result.SourceIDs = sourceIDs
		}
	}

	// Metadata
	if metadataJSON, ok := props["metadata"].(string); ok {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err == nil {
			result.Metadata = metadata
		}
	}

	return result
}

func (m *MemgraphDriver) nodeToProperties(node *types.Node) map[string]any {
	props := map[string]any{
		"uuid":       node.ID,
		"name":       node.Name,
		"type":       string(node.Type),
		"group_id":   node.GroupID,
		"created_at": node.CreatedAt.Format(time.RFC3339),
	}

	// Temporal fields
	if !node.ValidFrom.IsZero() {
		props["valid_from"] = node.ValidFrom.Format(time.RFC3339)
	}
	if node.ValidTo != nil && !node.ValidTo.IsZero() {
		props["valid_to"] = node.ValidTo.Format(time.RFC3339)
	}

	// Content fields
	if node.EntityType != "" {
		props["entity_type"] = node.EntityType
	}
	if node.Summary != "" {
		props["summary"] = node.Summary
	}
	if node.Content != "" {
		props["content"] = node.Content
	}
	if !node.Reference.IsZero() {
		props["reference"] = node.Reference.Format(time.RFC3339)
	}
	if node.Level > 0 {
		props["level"] = node.Level
	}

	// Episode-specific fields
	if node.EpisodeType != "" {
		props["episode_type"] = string(node.EpisodeType)
	}
	if len(node.EntityEdges) > 0 {
		if entityEdgesJSON, err := json.Marshal(node.EntityEdges); err == nil {
			props["entity_edges"] = string(entityEdgesJSON)
		}
	}

	// Embeddings - distinguish between name and generic embeddings
	if len(node.NameEmbedding) > 0 {
		if embeddingJSON, err := json.Marshal(node.NameEmbedding); err == nil {
			props["name_embedding"] = string(embeddingJSON)
		}
	}
	if len(node.Embedding) > 0 {
		if embeddingJSON, err := json.Marshal(node.Embedding); err == nil {
			props["embedding"] = string(embeddingJSON)
		}
	}

	// Source tracking
	if len(node.SourceIDs) > 0 {
		if sourceIDsJSON, err := json.Marshal(node.SourceIDs); err == nil {
			props["source_ids"] = string(sourceIDsJSON)
		}
	}

	// Metadata
	if node.Metadata != nil {
		if metadataJSON, err := json.Marshal(node.Metadata); err == nil {
			props["metadata"] = string(metadataJSON)
		}
	}

	return props
}

func (m *MemgraphDriver) edgeFromDBRelation(relation dbtype.Relationship, sourceID, targetID string) *types.Edge {
	props := relation.Props

	result := &types.Edge{
		BaseEdge: types.BaseEdge{
			SourceNodeID: sourceID,
			TargetNodeID: targetID,
		},
		SourceID: sourceID,
		TargetID: targetID,
	}

	// Core fields
	if id, ok := props["id"].(string); ok {
		result.ID = id
	}
	if edgeType, ok := props["type"].(string); ok {
		result.Type = types.EdgeType(edgeType)
	}
	if groupID, ok := props["group_id"].(string); ok {
		result.GroupID = groupID
	}

	// Timestamps
	if createdAtStr, ok := props["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			result.CreatedAt = t
		}
	}
	if updatedAtStr, ok := props["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			result.UpdatedAt = t
		}
	}

	// Temporal fields
	if validFromStr, ok := props["valid_from"].(string); ok {
		if t, err := time.Parse(time.RFC3339, validFromStr); err == nil {
			result.ValidFrom = t
		}
	}
	if validToStr, ok := props["valid_to"].(string); ok {
		if t, err := time.Parse(time.RFC3339, validToStr); err == nil {
			result.ValidTo = &t
		}
	}
	if expiredAtStr, ok := props["expired_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, expiredAtStr); err == nil {
			result.ExpiredAt = &t
		}
	}
	if validAtStr, ok := props["valid_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, validAtStr); err == nil {
			result.ValidAt = &t
		}
	}
	if invalidAtStr, ok := props["invalid_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, invalidAtStr); err == nil {
			result.InvalidAt = &t
		}
	}

	// Content fields
	if name, ok := props["name"].(string); ok {
		result.Name = name
	}
	if summary, ok := props["summary"].(string); ok {
		result.Summary = summary
	}
	if fact, ok := props["fact"].(string); ok {
		result.Fact = fact
	}
	if strength, ok := props["strength"].(float64); ok {
		result.Strength = strength
	}

	// Episodes tracking
	if episodesJSON, ok := props["episodes"].(string); ok {
		var episodes []string
		if err := json.Unmarshal([]byte(episodesJSON), &episodes); err == nil {
			result.Episodes = episodes
		}
	}

	// Embeddings
	if factEmbeddingJSON, ok := props["fact_embedding"].(string); ok {
		var embedding []float32
		if err := json.Unmarshal([]byte(factEmbeddingJSON), &embedding); err == nil {
			result.FactEmbedding = embedding
		}
	}
	if embeddingJSON, ok := props["embedding"].(string); ok {
		var embedding []float32
		if err := json.Unmarshal([]byte(embeddingJSON), &embedding); err == nil {
			result.Embedding = embedding
		}
	}

	// Source tracking
	if sourceIDsJSON, ok := props["source_ids"].(string); ok {
		var sourceIDs []string
		if err := json.Unmarshal([]byte(sourceIDsJSON), &sourceIDs); err == nil {
			result.SourceIDs = sourceIDs
		}
	}

	// Metadata
	if metadataJSON, ok := props["metadata"].(string); ok {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err == nil {
			result.Metadata = metadata
		}
	}

	return result
}

func (m *MemgraphDriver) edgeToProperties(edge *types.Edge) map[string]any {
	props := map[string]any{
		"id":         edge.ID,
		"type":       string(edge.Type),
		"group_id":   edge.GroupID,
		"created_at": edge.CreatedAt.Format(time.RFC3339),
	}

	// Temporal fields
	if !edge.ValidFrom.IsZero() {
		props["valid_from"] = edge.ValidFrom.Format(time.RFC3339)
	}
	if edge.ValidTo != nil && !edge.ValidTo.IsZero() {
		props["valid_to"] = edge.ValidTo.Format(time.RFC3339)
	}
	if edge.ExpiredAt != nil && !edge.ExpiredAt.IsZero() {
		props["expired_at"] = edge.ExpiredAt.Format(time.RFC3339)
	}
	if edge.ValidAt != nil && !edge.ValidAt.IsZero() {
		props["valid_at"] = edge.ValidAt.Format(time.RFC3339)
	}
	if edge.InvalidAt != nil && !edge.InvalidAt.IsZero() {
		props["invalid_at"] = edge.InvalidAt.Format(time.RFC3339)
	}

	// Content fields
	if edge.Name != "" {
		props["name"] = edge.Name
	}
	if edge.Summary != "" {
		props["summary"] = edge.Summary
	}
	if edge.Fact != "" {
		props["fact"] = edge.Fact
	}
	if edge.Strength > 0 {
		props["strength"] = edge.Strength
	}

	// Episodes tracking
	if len(edge.Episodes) > 0 {
		if episodesJSON, err := json.Marshal(edge.Episodes); err == nil {
			props["episodes"] = string(episodesJSON)
		}
	}

	// Embeddings - distinguish between fact and generic embeddings
	if len(edge.FactEmbedding) > 0 {
		if embeddingJSON, err := json.Marshal(edge.FactEmbedding); err == nil {
			props["fact_embedding"] = string(embeddingJSON)
		}
	}
	if len(edge.Embedding) > 0 {
		if embeddingJSON, err := json.Marshal(edge.Embedding); err == nil {
			props["embedding"] = string(embeddingJSON)
		}
	}

	// Source tracking
	if len(edge.SourceIDs) > 0 {
		if sourceIDsJSON, err := json.Marshal(edge.SourceIDs); err == nil {
			props["source_ids"] = string(sourceIDsJSON)
		}
	}

	// Metadata
	if edge.Metadata != nil {
		if metadataJSON, err := json.Marshal(edge.Metadata); err == nil {
			props["metadata"] = string(metadataJSON)
		}
	}

	return props
}

// cosineSimilarity computes the cosine similarity between two vectors
func (m *MemgraphDriver) cosineSimilarity(a, b []float32) float32 {
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

func (k *MemgraphDriver) GetBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string) ([]*types.Edge, error) {
	query := `
		MATCH (a:Entity {uuid: $source_uuid})-[:RELATES_TO]->(rel:RelatesToNode_)-[:RELATES_TO]->(b:Entity {uuid: $target_uuid})
		RETURN rel.uuid AS uuid, rel.name AS name, rel.fact AS fact, rel.group_id AS group_id,
		       rel.created_at AS created_at, rel.valid_at AS valid_at, rel.invalid_at AS invalid_at,
		       rel.expired_at AS expired_at, rel.episodes AS episodes, rel.attributes AS attributes,
		       a.uuid AS source_id, b.uuid AS target_id
		UNION
		MATCH (a:Entity {uuid: $target_uuid})-[:RELATES_TO]->(rel:RelatesToNode_)-[:RELATES_TO]->(b:Entity {uuid: $source_uuid})
		RETURN rel.uuid AS uuid, rel.name AS name, rel.fact AS fact, rel.group_id AS group_id,
		       rel.created_at AS created_at, rel.valid_at AS valid_at, rel.invalid_at AS invalid_at,
		       rel.expired_at AS expired_at, rel.episodes AS episodes, rel.attributes AS attributes,
		       a.uuid AS source_id, b.uuid AS target_id
	`

	params := map[string]interface{}{
		"source_uuid": sourceNodeID,
		"target_uuid": targetNodeID,
	}

	result, _, _, err := k.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetBetweenNodes query: %w", err)
	}

	// Convert result to Edge objects
	var edges []*types.Edge
	recordSlice, ok := result.([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected query result type: %T", result)
	}

	for _, record := range recordSlice {
		edge, err := convertRecordToEdge(record)
		if err != nil {
			log.Printf("Warning: failed to convert record to edge: %v", err)
			continue
		}
		edges = append(edges, edge)
	}

	return edges, nil
}

func (m *MemgraphDriver) GetNodeNeighbors(ctx context.Context, nodeUUID, groupID string) ([]types.Neighbor, error) {
	query := `
      MATCH (n:Entity {uuid: $id, group_id: $group_id})-[:RELATES_TO]->(e:RelatesToNode_)<-[:RELATES_TO]-
      (m:Entity {group_id: $group_id})
	  WITH m.uuid AS uuid, count(e) AS count
	  RETURN uuid, count
	`

	params := map[string]any{
		"id":       nodeUUID,
		"group_id": groupID,
	}

	result, _, _, err := m.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute neighbor query: %w", err)
	}

	return m.parseNeighborsFromRecords(result)
}

// parseNeighborsFromRecords parses Neo4j/Memgraph records into neighbors
func (m *MemgraphDriver) parseNeighborsFromRecords(result interface{}) ([]types.Neighbor, error) {
	var neighbors []types.Neighbor

	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %T", result)
	}

	for i := 0; i < value.Len(); i++ {
		record := value.Index(i)

		// Ensure we are dealing with a pointer to a struct (e.g. *db.Record)
		if record.Kind() == reflect.Interface {
			record = record.Elem()
		}

		if !record.IsValid() {
			continue
		}

		getMethod := record.MethodByName("Get")
		if !getMethod.IsValid() {
			return nil, fmt.Errorf("record type %T does not have a Get method", record.Interface())
		}

		// Safely call Get("uuid") and Get("count")
		uuidResults := getMethod.Call([]reflect.Value{reflect.ValueOf("uuid")})
		countResults := getMethod.Call([]reflect.Value{reflect.ValueOf("count")})

		if len(uuidResults) == 0 || len(countResults) == 0 {
			continue
		}

		uuidInterface := uuidResults[0].Interface()
		countInterface := countResults[0].Interface()

		uuid, ok := uuidInterface.(string)
		if !ok || uuid == "" {
			continue
		}

		var edgeCount int
		switch v := countInterface.(type) {
		case int:
			edgeCount = v
		case int64:
			edgeCount = int(v)
		case float64:
			edgeCount = int(v)
		default:
			continue
		}

		neighbors = append(neighbors, types.Neighbor{
			NodeUUID:  uuid,
			EdgeCount: edgeCount,
		})
	}

	return neighbors, nil
}
