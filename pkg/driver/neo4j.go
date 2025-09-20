package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
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

// GetNeighbors retrieves neighboring nodes within a specified distance
func (n *Neo4jDriver) GetNeighbors(ctx context.Context, nodeID, groupID string, maxDistance int) ([]*types.Node, error) {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := fmt.Sprintf(`
			MATCH (start {id: $nodeID, group_id: $groupID})
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
		nodes = append(nodes, n.nodeFromDBNode(node))
	}

	return nodes, nil
}

func (n *Neo4jDriver) GetRelatedNodes(ctx context.Context, nodeID, groupID string, edgeTypes []types.EdgeType) ([]*types.Node, error) {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
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
				MATCH (start {id: $nodeID, group_id: $groupID})
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
				MATCH (start {id: $nodeID, group_id: $groupID})
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
		nodes = append(nodes, n.nodeFromDBNode(node))
	}

	return nodes, nil
}

func (n *Neo4jDriver) SearchNodesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Node, error) {
	if len(embedding) == 0 {
		return []*types.Node{}, nil
	}

	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	// Get all nodes with embeddings and compute similarity in-memory
	// In production, you might want to use Neo4j's vector index capabilities
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
		node := n.nodeFromDBNode(dbNode)

		// Parse embedding from JSON
		if embeddingStr, ok := dbNode.Props["embedding"].(string); ok {
			var nodeEmbedding []float32
			if err := json.Unmarshal([]byte(embeddingStr), &nodeEmbedding); err == nil {
				similarity := n.cosineSimilarity(embedding, nodeEmbedding)
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

func (n *Neo4jDriver) SearchEdgesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Edge, error) {
	if len(embedding) == 0 {
		return []*types.Edge{}, nil
	}

	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
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
		edge := n.edgeFromDBRelation(dbRelation, sourceID.(string), targetID.(string))

		// Parse embedding from JSON
		if embeddingStr, ok := dbRelation.Props["embedding"].(string); ok {
			var edgeEmbedding []float32
			if err := json.Unmarshal([]byte(embeddingStr), &edgeEmbedding); err == nil {
				similarity := n.cosineSimilarity(embedding, edgeEmbedding)
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

func (n *Neo4jDriver) UpsertNodes(ctx context.Context, nodes []*types.Node) error {
	if len(nodes) == 0 {
		return nil
	}

	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	// Use a transaction to batch the operations
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, node := range nodes {
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
			if err != nil {
				return nil, fmt.Errorf("failed to upsert node %s: %w", node.ID, err)
			}
		}
		return nil, nil
	})

	return err
}

func (n *Neo4jDriver) UpsertEdges(ctx context.Context, edges []*types.Edge) error {
	if len(edges) == 0 {
		return nil
	}

	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	// Use a transaction to batch the operations
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, edge := range edges {
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
			if err != nil {
				return nil, fmt.Errorf("failed to upsert edge %s: %w", edge.ID, err)
			}
		}
		return nil, nil
	})

	return err
}

func (n *Neo4jDriver) GetNodesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Node, error) {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
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
		nodes = append(nodes, n.nodeFromDBNode(node))
	}

	return nodes, nil
}

func (n *Neo4jDriver) GetEdgesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Edge, error) {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
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

		edges = append(edges, n.edgeFromDBRelation(relation, sourceID.(string), targetID.(string)))
	}

	return edges, nil
}

func (n *Neo4jDriver) GetCommunities(ctx context.Context, groupID string, level int) ([]*types.Node, error) {
	// For basic implementation, return nodes grouped by a hypothetical community property
	// In production, you might use algorithms like Louvain or Label Propagation
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
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
		nodes = append(nodes, n.nodeFromDBNode(node))
	}

	return nodes, nil
}

func (n *Neo4jDriver) BuildCommunities(ctx context.Context, groupID string) error {
	// Basic implementation that assigns community IDs based on connected components
	// In production, you would use proper community detection algorithms
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
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
			WITH n, collect(DISTINCT connected.id) + [n.id] as component
			SET n.community_id = component[0]
			SET n.community_level = 0
		`
		_, err = tx.Run(ctx, communityQuery, map[string]any{"groupID": groupID})
		return nil, err
	})

	return err
}

func (n *Neo4jDriver) CreateIndices(ctx context.Context) error {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Create indices for commonly queried properties
		indices := []string{
			"CREATE INDEX node_id_group_idx IF NOT EXISTS FOR (n) ON (n.id, n.group_id)",
			"CREATE INDEX edge_id_group_idx IF NOT EXISTS FOR ()-[r]-() ON (r.id, r.group_id)",
			"CREATE INDEX node_created_at_idx IF NOT EXISTS FOR (n) ON (n.created_at)",
			"CREATE INDEX edge_created_at_idx IF NOT EXISTS FOR ()-[r]-() ON (r.created_at)",
			"CREATE INDEX node_type_idx IF NOT EXISTS FOR (n) ON (n.type)",
			"CREATE INDEX edge_type_idx IF NOT EXISTS FOR ()-[r]-() ON (r.type)",
		}

		for _, indexQuery := range indices {
			_, err := tx.Run(ctx, indexQuery, nil)
			if err != nil {
				// Continue with other indices even if one fails
				continue
			}
		}

		return nil, nil
	})

	return err
}

func (n *Neo4jDriver) GetStats(ctx context.Context, groupID string) (*GraphStats, error) {
	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Get node count and types
		nodeQuery := `
			MATCH (n {group_id: $groupID})
			RETURN count(n) as node_count, n.type as node_type
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
			RETURN count(r) as edge_count, r.type as edge_type
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
func (n *Neo4jDriver) SearchNodes(ctx context.Context, query, groupID string, options *SearchOptions) ([]*types.Node, error) {
	if query == "" {
		return []*types.Node{}, nil
	}

	limit := 10
	if options != nil && options.Limit > 0 {
		limit = options.Limit
	}

	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Basic text search using CONTAINS (in production, use Neo4j's fulltext indexes)
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
		nodes = append(nodes, n.nodeFromDBNode(node))
	}

	return nodes, nil
}

// SearchEdges performs text-based search on edges
func (n *Neo4jDriver) SearchEdges(ctx context.Context, query, groupID string, options *SearchOptions) ([]*types.Edge, error) {
	if query == "" {
		return []*types.Edge{}, nil
	}

	limit := 10
	if options != nil && options.Limit > 0 {
		limit = options.Limit
	}

	session := n.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: n.database})
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

		edges = append(edges, n.edgeFromDBRelation(relation, sourceID.(string), targetID.(string)))
	}

	return edges, nil
}

// SearchNodesByVector performs vector similarity search on nodes
func (n *Neo4jDriver) SearchNodesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Node, error) {
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
	nodes, err := n.SearchNodesByEmbedding(ctx, vector, groupID, limit)
	if err != nil {
		return nil, err
	}

	// Apply minimum score filter if specified
	if minScore > 0 {
		var filteredNodes []*types.Node
		for _, node := range nodes {
			if len(node.Embedding) > 0 {
				similarity := n.cosineSimilarity(vector, node.Embedding)
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
func (n *Neo4jDriver) SearchEdgesByVector(ctx context.Context, vector []float32, groupID string, options *VectorSearchOptions) ([]*types.Edge, error) {
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
	edges, err := n.SearchEdgesByEmbedding(ctx, vector, groupID, limit)
	if err != nil {
		return nil, err
	}

	// Apply minimum score filter if specified
	if minScore > 0 {
		var filteredEdges []*types.Edge
		for _, edge := range edges {
			if len(edge.Embedding) > 0 {
				similarity := n.cosineSimilarity(vector, edge.Embedding)
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
func (n *Neo4jDriver) ExecuteQuery(cypherQuery string, kwargs map[string]interface{}) (interface{}, interface{}, interface{}, error) {
	session := n.client.NewSession(context.Background(), neo4j.SessionConfig{DatabaseName: n.database})
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
func (n *Neo4jDriver) Session(database *string) GraphDriverSession {
	dbName := n.database
	if database != nil {
		dbName = *database
	}
	return &Neo4jDriverSession{
		driver:   n,
		database: dbName,
	}
}

// DeleteAllIndexes deletes all indexes in the specified database.
func (n *Neo4jDriver) DeleteAllIndexes(database string) {
	// Implementation for deleting indexes
	session := n.client.NewSession(context.Background(), neo4j.SessionConfig{DatabaseName: database})
	defer session.Close(context.Background())

	// Get all indexes
	result, err := session.Run(context.Background(), "SHOW INDEXES", nil)
	if err != nil {
		return
	}

	records, err := result.Collect(context.Background())
	if err != nil {
		return
	}

	// Drop each index
	for _, record := range records {
		if name, ok := record.Values[1].(string); ok {
			session.Run(context.Background(), fmt.Sprintf("DROP INDEX %s IF EXISTS", name), nil)
		}
	}
}

// Provider returns the provider type.
func (n *Neo4jDriver) Provider() GraphProvider {
	return GraphProviderNeo4j
}

// GetAossClient returns nil for Neo4j (Amazon OpenSearch not applicable).
func (n *Neo4jDriver) GetAossClient() interface{} {
	return nil
}

// Close closes the Neo4j driver.
func (n *Neo4jDriver) Close() error {
	return n.client.Close(context.Background())
}

// Neo4jDriverSession implements GraphDriverSession for Neo4j.
type Neo4jDriverSession struct {
	driver   *Neo4jDriver
	database string
	session  neo4j.SessionWithContext
}

// Enter implements the context manager pattern.
func (s *Neo4jDriverSession) Enter(ctx context.Context) (GraphDriverSession, error) {
	s.session = s.driver.client.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	return s, nil
}

// Exit implements the context manager pattern.
func (s *Neo4jDriverSession) Exit(ctx context.Context, excType, excVal, excTb interface{}) error {
	if s.session != nil {
		return s.session.Close(ctx)
	}
	return nil
}

// Close closes the session.
func (s *Neo4jDriverSession) Close() error {
	if s.session != nil {
		return s.session.Close(context.Background())
	}
	return nil
}

// Run executes a query in this session.
func (s *Neo4jDriverSession) Run(ctx context.Context, query interface{}, kwargs map[string]interface{}) error {
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
func (s *Neo4jDriverSession) ExecuteWrite(ctx context.Context, fn func(context.Context, GraphDriverSession, ...interface{}) (interface{}, error), args ...interface{}) (interface{}, error) {
	if s.session == nil {
		return nil, fmt.Errorf("session not entered")
	}

	return s.session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		return fn(ctx, s, args...)
	})
}

// Provider returns the provider type.
func (s *Neo4jDriverSession) Provider() GraphProvider {
	return GraphProviderNeo4j
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

// cosineSimilarity computes the cosine similarity between two vectors
func (n *Neo4jDriver) cosineSimilarity(a, b []float32) float32 {
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