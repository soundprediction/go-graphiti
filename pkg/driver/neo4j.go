package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// Neo4jDriver implements the GraphDriver interface for Neo4j databases.
type Neo4jDriver struct {
	client   neo4j.DriverWithContext
	database string
}

// NewNeo4jDriver creates a new Neo4j driver instance.
func NewNeo4jDriver(uri, username, password, database string) (*Neo4jDriver, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	if database == "" {
		database = "neo4j"
	}

	return &Neo4jDriver{
		client:   driver,
		database: database,
	}, nil
}

// GetNode retrieves a node by ID.
func (n *Neo4jDriver) GetNode(ctx context.Context, nodeID, groupID string) (*types.Node, error) {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {id: $nodeID, group_id: $groupID})
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
	return n.nodeFromDBNode(node), nil
}

// UpsertNode creates or updates a node.
func (n *Neo4jDriver) UpsertNode(ctx context.Context, node *types.Node) error {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MERGE (n {id: $id, group_id: $group_id})
			SET n += $properties
			SET n.updated_at = $updated_at
		`

		properties := n.nodeToProperties(node)
		_, err := tx.Run(ctx, query, map[string]any{
			"id":         node.ID,
			"group_id":   node.GroupID,
			"properties": properties,
			"updated_at": time.Now().Format(time.RFC3339),
		})
		return nil, err
	})

	return err
}

// DeleteNode removes a node and its edges.
func (n *Neo4jDriver) DeleteNode(ctx context.Context, nodeID, groupID string) error {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {id: $nodeID, group_id: $groupID})
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
func (n *Neo4jDriver) GetNodes(ctx context.Context, nodeIDs []string, groupID string) ([]*types.Node, error) {
	if len(nodeIDs) == 0 {
		return []*types.Node{}, nil
	}

	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {group_id: $groupID})
			WHERE n.id IN $nodeIDs
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
		nodes = append(nodes, n.nodeFromDBNode(node))
	}

	return nodes, nil
}

// GetEdge retrieves an edge by ID.
func (n *Neo4jDriver) GetEdge(ctx context.Context, edgeID, groupID string) (*types.Edge, error) {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (s)-[r {id: $edgeID, group_id: $groupID}]->(t)
			RETURN r, s.id as source_id, t.id as target_id
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

	return n.edgeFromDBRelation(relation, sourceID.(string), targetID.(string)), nil
}

// UpsertEdge creates or updates an edge.
func (n *Neo4jDriver) UpsertEdge(ctx context.Context, edge *types.Edge) error {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (s {id: $source_id, group_id: $group_id})
			MATCH (t {id: $target_id, group_id: $group_id})
			MERGE (s)-[r:RELATES {id: $id, group_id: $group_id}]->(t)
			SET r += $properties
			SET r.updated_at = $updated_at
		`

		properties := n.edgeToProperties(edge)
		_, err := tx.Run(ctx, query, map[string]any{
			"id":         edge.ID,
			"source_id":  edge.SourceID,
			"target_id":  edge.TargetID,
			"group_id":   edge.GroupID,
			"properties": properties,
			"updated_at": time.Now().Format(time.RFC3339),
		})
		return nil, err
	})

	return err
}

// DeleteEdge removes an edge.
func (n *Neo4jDriver) DeleteEdge(ctx context.Context, edgeID, groupID string) error {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH ()-[r {id: $edgeID, group_id: $groupID}]-()
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
func (n *Neo4jDriver) GetEdges(ctx context.Context, edgeIDs []string, groupID string) ([]*types.Edge, error) {
	if len(edgeIDs) == 0 {
		return []*types.Edge{}, nil
	}

	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (s)-[r {group_id: $groupID}]->(t)
			WHERE r.id IN $edgeIDs
			RETURN r, s.id as source_id, t.id as target_id
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

		edges = append(edges, n.edgeFromDBRelation(relation, sourceID.(string), targetID.(string)))
	}

	return edges, nil
}

// Placeholder implementations for other methods
func (n *Neo4jDriver) GetNeighbors(ctx context.Context, nodeID, groupID string, maxDistance int) ([]*types.Node, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) GetRelatedNodes(ctx context.Context, nodeID, groupID string, edgeTypes []types.EdgeType) ([]*types.Node, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) SearchNodesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Node, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) SearchEdgesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Edge, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) UpsertNodes(ctx context.Context, nodes []*types.Node) error {
	return fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) UpsertEdges(ctx context.Context, edges []*types.Edge) error {
	return fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) GetNodesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Node, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) GetEdgesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Edge, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) GetCommunities(ctx context.Context, groupID string, level int) ([]*types.Node, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) BuildCommunities(ctx context.Context, groupID string) error {
	return fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) CreateIndices(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (n *Neo4jDriver) GetStats(ctx context.Context, groupID string) (*GraphStats, error) {
	return nil, fmt.Errorf("not implemented")
}

// Close closes the Neo4j driver.
func (n *Neo4jDriver) Close(ctx context.Context) error {
	return n.client.Close(ctx)
}

// Helper methods for converting between Graphiti and Neo4j types

func (n *Neo4jDriver) nodeFromDBNode(node dbtype.Node) *types.Node {
	props := node.Props
	
	result := &types.Node{}
	
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
	
	return result
}

func (n *Neo4jDriver) nodeToProperties(node *types.Node) map[string]any {
	props := map[string]any{
		"id":         node.ID,
		"name":       node.Name,
		"type":       string(node.Type),
		"group_id":   node.GroupID,
		"created_at": node.CreatedAt.Format(time.RFC3339),
	}
	
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
	if len(node.Embedding) > 0 {
		if embeddingJSON, err := json.Marshal(node.Embedding); err == nil {
			props["embedding"] = string(embeddingJSON)
		}
	}
	if len(node.SourceIDs) > 0 {
		if sourceIDsJSON, err := json.Marshal(node.SourceIDs); err == nil {
			props["source_ids"] = string(sourceIDsJSON)
		}
	}
	
	return props
}

func (n *Neo4jDriver) edgeFromDBRelation(relation dbtype.Relationship, sourceID, targetID string) *types.Edge {
	props := relation.Props
	
	result := &types.Edge{
		SourceID: sourceID,
		TargetID: targetID,
	}
	
	if id, ok := props["id"].(string); ok {
		result.ID = id
	}
	if edgeType, ok := props["type"].(string); ok {
		result.Type = types.EdgeType(edgeType)
	}
	if groupID, ok := props["group_id"].(string); ok {
		result.GroupID = groupID
	}
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
	
	return result
}

func (n *Neo4jDriver) edgeToProperties(edge *types.Edge) map[string]any {
	props := map[string]any{
		"id":         edge.ID,
		"type":       string(edge.Type),
		"group_id":   edge.GroupID,
		"created_at": edge.CreatedAt.Format(time.RFC3339),
	}
	
	if edge.Name != "" {
		props["name"] = edge.Name
	}
	if edge.Summary != "" {
		props["summary"] = edge.Summary
	}
	if edge.Strength > 0 {
		props["strength"] = edge.Strength
	}
	if len(edge.Embedding) > 0 {
		if embeddingJSON, err := json.Marshal(edge.Embedding); err == nil {
			props["embedding"] = string(embeddingJSON)
		}
	}
	if len(edge.SourceIDs) > 0 {
		if sourceIDsJSON, err := json.Marshal(edge.SourceIDs); err == nil {
			props["source_ids"] = string(sourceIDsJSON)
		}
	}
	
	return props
}