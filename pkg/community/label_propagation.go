package community

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// labelPropagation implements the label propagation community detection algorithm
func (b *Builder) labelPropagation(projection map[string][]types.Neighbor) [][]string {
	if len(projection) == 0 {
		return nil
	}

	// Initialize each node to its own community
	communityMap := make(map[string]int)
	nodeIndex := 0
	for uuid := range projection {
		communityMap[uuid] = nodeIndex
		nodeIndex++
	}

	maxIterations := 100 // Prevent infinite loops
	for iteration := 0; iteration < maxIterations; iteration++ {
		noChange := true
		newCommunityMap := make(map[string]int)

		for uuid, neighbors := range projection {
			currentCommunity := communityMap[uuid]

			// Count community occurrences among neighbors, weighted by edge count
			communityCandidates := make(map[int]int)
			for _, neighbor := range neighbors {
				if neighborCommunity, exists := communityMap[neighbor.NodeUUID]; exists {
					communityCandidates[neighborCommunity] += neighbor.EdgeCount
				}
			}

			// Find the community with highest weighted count
			newCommunity := currentCommunity

			// Convert to slice for sorting
			type communityScore struct {
				community int
				count     int
			}

			var scores []communityScore
			for community, count := range communityCandidates {
				scores = append(scores, communityScore{community: community, count: count})
			}

			// Sort by count (descending), then by community ID for tie-breaking
			for i := 0; i < len(scores); i++ {
				for j := i + 1; j < len(scores); j++ {
					if scores[j].count > scores[i].count ||
						(scores[j].count == scores[i].count && scores[j].community > scores[i].community) {
						scores[i], scores[j] = scores[j], scores[i]
					}
				}
			}

			if len(scores) > 0 {
				topScore := scores[0]
				if topScore.count > 1 { // Only change if there's significant support
					newCommunity = topScore.community
				} else {
					// Keep the maximum of current and candidate
					if topScore.community > currentCommunity {
						newCommunity = topScore.community
					}
				}
			}

			newCommunityMap[uuid] = newCommunity

			if newCommunity != currentCommunity {
				noChange = false
			}
		}

		if noChange {
			break
		}

		communityMap = newCommunityMap
	}

	// Group nodes by community
	communityClusterMap := make(map[int][]string)
	for uuid, community := range communityMap {
		communityClusterMap[community] = append(communityClusterMap[community], uuid)
	}

	// Convert to slice of clusters
	var clusters [][]string
	for _, cluster := range communityClusterMap {
		if len(cluster) > 1 { // Only include clusters with more than one node
			clusters = append(clusters, cluster)
		}
	}

	return clusters
}

// buildProjection builds the neighbor projection for community detection
func (b *Builder) buildProjection(ctx context.Context, nodes []*types.Node, groupID string) (map[string][]types.Neighbor, error) {
	projection := make(map[string][]types.Neighbor)

	for _, node := range nodes {
		neighbors, err := b.getNodeNeighbors(ctx, node.ID, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to get neighbors for node %s: %w", node.ID, err)
		}
		projection[node.ID] = neighbors
	}

	return projection, nil
}

// getNodeNeighbors gets the neighbors of a node with edge counts
func (b *Builder) getNodeNeighbors(ctx context.Context, nodeUUID, groupID string) ([]types.Neighbor, error) {
	// Check if this is a Kuzu driver to use the appropriate query
	return b.driver.GetNodeNeighbors(ctx, nodeUUID, groupID)
}

// getAllGroupIDs gets all distinct group IDs from entity nodes
func (b *Builder) getAllGroupIDs(ctx context.Context) ([]string, error) {
	if kuzuDriver, ok := b.driver.(*driver.KuzuDriver); ok {
		return b.getAllGroupIDsKuzu(ctx, kuzuDriver)
	}

	// For Neo4j/Memgraph drivers
	return b.getAllGroupIDsNeo4j(ctx)
}

// getAllGroupIDsKuzu gets all group IDs specifically for Kuzu
func (b *Builder) getAllGroupIDsKuzu(ctx context.Context, kuzuDriver *driver.KuzuDriver) ([]string, error) {
	query := `
		MATCH (n:Entity)
		WHERE n.group_id IS NOT NULL
		RETURN collect(DISTINCT n.group_id) AS group_ids
	`

	records, _, _, err := kuzuDriver.ExecuteQuery(query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute group IDs query: %w", err)
	}

	recordSlice, ok := records.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected records type: %T", records)
	}

	if len(recordSlice) == 0 {
		return []string{}, nil
	}

	// Extract group IDs from the result
	if groupIDsInterface, ok := recordSlice[0]["group_ids"]; ok {
		if groupIDs, ok := groupIDsInterface.([]interface{}); ok {
			var result []string
			for _, gid := range groupIDs {
				if gidStr, ok := gid.(string); ok {
					result = append(result, gidStr)
				}
			}
			return result, nil
		}
	}

	return []string{}, nil
}

// getEntityNodesByGroup gets all entity nodes for a specific group
func (b *Builder) getEntityNodesByGroup(ctx context.Context, groupID string) ([]*types.Node, error) {
	if kuzuDriver, ok := b.driver.(*driver.KuzuDriver); ok {
		return b.getEntityNodesByGroupKuzu(ctx, kuzuDriver, groupID)
	}

	// For Neo4j/Memgraph drivers
	return b.getEntityNodesByGroupNeo4j(ctx, groupID)
}

// getEntityNodesByGroupKuzu gets entity nodes specifically for Kuzu
func (b *Builder) getEntityNodesByGroupKuzu(ctx context.Context, kuzuDriver *driver.KuzuDriver, groupID string) ([]*types.Node, error) {
	query := `
		MATCH (n:Entity {group_id: $group_id})
		RETURN n.uuid AS uuid, n.name AS name, n.summary AS summary, n.created_at AS created_at
	`
	params := map[string]interface{}{
		"group_id": groupID,
	}

	records, _, _, err := kuzuDriver.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute entity nodes query: %w", err)
	}

	var nodes []*types.Node
	recordSlice, ok := records.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected records type: %T", records)
	}
	for _, record := range recordSlice {
		node := &types.Node{
			Type:    types.EntityNodeType,
			GroupID: groupID,
		}

		if uuid, ok := record["uuid"].(string); ok {
			node.ID = uuid
		}
		if name, ok := record["name"].(string); ok {
			node.Name = name
		}
		if summary, ok := record["summary"].(string); ok {
			node.Summary = summary
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// getNodesByUUIDs gets nodes by their UUIDs
func (b *Builder) getNodesByUUIDs(ctx context.Context, uuids []string, groupID string) ([]*types.Node, error) {
	var nodes []*types.Node

	for _, uuid := range uuids {
		node, err := b.driver.GetNode(ctx, uuid, groupID)
		if err != nil {
			continue // Skip nodes that can't be found
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// ====== Neo4j/Memgraph Implementations ======

// getAllGroupIDsNeo4j gets all group IDs for Neo4j/Memgraph
func (b *Builder) getAllGroupIDsNeo4j(ctx context.Context) ([]string, error) {
	query := `
		MATCH (n:Entity)
		WHERE n.group_id IS NOT NULL
		RETURN collect(DISTINCT n.group_id) AS group_ids
	`

	result, _, _, err := b.driver.ExecuteQuery(query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute group IDs query: %w", err)
	}

	return b.parseGroupIDsFromRecords(result)
}

// getEntityNodesByGroupNeo4j gets entity nodes for Neo4j/Memgraph
func (b *Builder) getEntityNodesByGroupNeo4j(ctx context.Context, groupID string) ([]*types.Node, error) {
	query := `
		MATCH (n:Entity {group_id: $group_id})
		RETURN n.uuid AS uuid, n.name AS name, n.type AS type, n.entity_type AS entity_type, n.group_id AS group_id, n.summary AS summary, n.created_at AS created_at
	`

	params := map[string]interface{}{
		"group_id": groupID,
	}

	result, _, _, err := b.driver.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute entity nodes query: %w", err)
	}
	nodes, _ := driver.ParseNeo4JRecords[*types.Node](result)
	return nodes, nil
}

// ====== Record Parsing Helpers ======

// parseGroupIDsFromRecords parses group IDs from Neo4j/Memgraph records
func (b *Builder) parseGroupIDsFromRecords(result interface{}) ([]string, error) {
	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %T", result)
	}

	if value.Len() == 0 {
		return []string{}, nil
	}

	// Get first record
	record := value.Index(0)
	getMethod := record.MethodByName("Get")
	if !getMethod.IsValid() {
		return []string{}, nil
	}

	// Get group_ids field
	results := getMethod.Call([]reflect.Value{reflect.ValueOf("group_ids")})
	if len(results) < 1 {
		return []string{}, nil
	}

	groupIDsInterface := results[0].Interface()

	// Handle different types
	switch gids := groupIDsInterface.(type) {
	case []interface{}:
		var groupIDs []string
		for _, gid := range gids {
			if gidStr, ok := gid.(string); ok {
				groupIDs = append(groupIDs, gidStr)
			}
		}
		return groupIDs, nil
	case []string:
		return gids, nil
	}

	return []string{}, nil
}

// DummyMemgraphNode mimics a memgraph.Node struct
type DummyMemgraphNode struct {
	ID    int64
	Props map[string]interface{}
}

func (n DummyMemgraphNode) GetID() int64                     { return n.ID }
func (n DummyMemgraphNode) GetProps() map[string]interface{} { return n.Props }

// DummyNeo4jNode mimics a neo4j.Node struct
type DummyNeo4jNode struct {
	Id    int64 // Note the lowercase 'd'
	Props map[string]interface{}
}

func (n DummyNeo4jNode) GetID() int64                     { return n.Id }
func (n DummyNeo4jNode) GetProps() map[string]interface{} { return n.Props }

// parseEntityNodesFromRecords parses entity nodes from Neo4j/Memgraph records
func parseEntityNodesFromRecords(result interface{}, groupID string) ([]*types.Node, error) {
	var records []map[string]interface{}
	var isListResult bool

	// Step 1: Type-assert the result to a common structure.
	if dictRecords, ok := result.([]map[string]interface{}); ok {
		records = dictRecords
	} else if listRecords, okList := result.([][]interface{}); okList {
		isListResult = true
		// Convert list result (e.g., [][]interface{}) to a dict result
		// for uniform processing. We assume the node is in the first column.
		records = make([]map[string]interface{}, len(listRecords))
		for i, row := range listRecords {
			if len(row) > 0 {
				records[i] = map[string]interface{}{"col_0": row[0]}
			} else {
				records[i] = make(map[string]interface{}) // Handle empty row
			}
		}
	} else {
		return nil, fmt.Errorf("expected []map[string]interface{} (dict result) or [][]interface{} (list result), got %T", result)
	}

	var nodes []*types.Node
	var seenIDs = make(map[string]struct{})

	// Step 2: Iterate over each record (row)
	for i, record := range records {

		// Step 3: Iterate over the columns in the record (e.g., {"n": <node>, "r": <rel>})
		for key, value := range record {
			var graphNode GraphNode
			var isNode bool

			// Step 4: Check if the column value is a graph node.
			// In a real implementation, you'd check for concrete driver types:
			//
			// if n, ok := value.(memgraph.Node); ok {
			//     graphNode = n // memgraph.Node must implement GraphNode
			//     isNode = true
			// } else if n, ok := value.(neo4j.Node); ok {
			//     graphNode = n // neo4j.Node must implement GraphNode
			//     isNode = true
			// }

			// For this example, we check our interface and dummy types
			if n, ok := value.(GraphNode); ok {
				graphNode = n
				isNode = true
			} else if n, ok := value.(DummyMemgraphNode); ok {
				graphNode = n
				isNode = true
			} else if n, ok := value.(DummyNeo4jNode); ok {
				graphNode = n
				isNode = true
			}

			// If this value is a node, parse it
			if isNode {
				node, err := parseNodeFromProps(graphNode.GetProps(), groupID, graphNode.GetID())
				if err != nil {
					// Log the error and continue parsing other records
					fmt.Printf("Warning: failed to parse node (id: %d) in record %d, key %s: %v\n", graphNode.GetID(), i, key, err)
					continue
				}

				// Avoid adding duplicate nodes if returned in multiple rows
				if _, seen := seenIDs[node.ID]; !seen {
					nodes = append(nodes, node)
					seenIDs[node.ID] = struct{}{}
				}

				// If the original result was a list, we assume only one
				// column (which we found), so we can break.
				// If it was a dict, we parse the first node found and break.
				if isListResult || !isListResult {
					break
				}
			}
		}
	}

	return nodes, nil
}

// GraphNode defines the methods we expect a node object from the driver to have.
type GraphNode interface {
	GetID() int64
	GetProps() map[string]interface{}
}

// parseNodeFromProps is a helper to populate types.Node from a property map
func parseNodeFromProps(props map[string]interface{}, groupID string, graphID int64) (*types.Node, error) {
	if props == nil {
		return nil, fmt.Errorf("node properties map is nil for ID %d", graphID)
	}

	node := &types.Node{
		ID:      fmt.Sprintf("%d", graphID), // Convert graph int64 ID to string
		GroupID: groupID,
		Type:    "Entity", // This function is for *Entity* nodes
	}

	// --- Populate fields using safe type assertions ---

	if val, ok := props["name"].(string); ok {
		node.Name = val
	}

	// Entity-specific fields
	if val, ok := props["entity_type"].(string); ok {
		node.EntityType = val
	}
	if val, ok := props["summary"].(string); ok {
		node.Summary = val
	}

	// Timestamps
	node.CreatedAt = parseTimeProp(props["created_at"], time.Now().UTC()) // Default to Now
	node.UpdatedAt = parseTimeProp(props["updated_at"], time.Now().UTC()) // Default to Now
	node.ValidFrom = parseTimeProp(props["valid_from"], time.Time{})      // Default to zero time

	if props["valid_to"] != nil && props["valid_to"] != "" {
		vt := parseTimeProp(props["valid_to"], time.Time{})
		if !vt.IsZero() {
			node.ValidTo = &vt
		}
	}

	// Source IDs
	if val, ok := props["source_ids"].([]interface{}); ok {
		for _, v := range val {
			if str, ok := v.(string); ok {
				node.SourceIDs = append(node.SourceIDs, str)
			}
		}
	} else if val, ok := props["source_ids"].([]string); ok { // Handle if already string slice
		node.SourceIDs = val
	}

	// Embeddings
	node.Embedding = parseFloat32Slice(props["embedding"])
	node.NameEmbedding = parseFloat32Slice(props["name_embedding"])

	// Metadata
	if val, ok := props["metadata"].(map[string]interface{}); ok {
		node.Metadata = val
	}

	// Note: Episode/Community fields are intentionally left blank
	// as this function is for parsing Entity nodes.

	return node, nil
}

// parseTimeProp is a helper to safely parse time from various formats
func parseTimeProp(prop interface{}, defaultVal time.Time) time.Time {
	if prop == nil {
		return defaultVal
	}

	// Case 1: Already time.Time
	if t, ok := prop.(time.Time); ok {
		return t.UTC()
	}

	// Case 2: String (ISO 8601 / RFC3339)
	if s, ok := prop.(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.UTC()
		}
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t.UTC()
		}
		// Add more string formats if needed
	}

	// Case 3: int64 (assume epoch milliseconds)
	if i, ok := prop.(int64); ok {
		return time.Unix(0, i*int64(time.Millisecond)).UTC()
	}

	// Case 4: float64 (some drivers use this, assume epoch milliseconds)
	if f, ok := prop.(float64); ok {
		return time.Unix(0, int64(f)*int64(time.Millisecond)).UTC()
	}

	// Case 5: Driver-specific types (e.g., neo4j.LocalDateTime)
	// You would need to check for these concrete types here.
	// Example (conceptual):
	// if neoTime, ok := prop.(neo4j.LocalDateTime); ok {
	//     return neoTime.Time().UTC()
	// }

	return defaultVal
}

// parseFloat32Slice is a helper to safely parse embedding vectors
func parseFloat32Slice(prop interface{}) []float32 {
	if prop == nil {
		return nil
	}

	// Case 1: []interface{} (most common from drivers)
	if slice, ok := prop.([]interface{}); ok {
		result := make([]float32, 0, len(slice))
		for _, v := range slice {
			if f64, ok := v.(float64); ok { // JSON unmarshals numbers to float64
				result = append(result, float32(f64))
			} else if f32, ok := v.(float32); ok {
				result = append(result, f32)
			}
		}
		return result
	}

	// Case 2: Already []float32
	if slice, ok := prop.([]float32); ok {
		return slice
	}

	// Case 3: Already []float64
	if slice, ok := prop.([]float64); ok {
		result := make([]float32, len(slice))
		for i, v := range slice {
			result[i] = float32(v)
		}
		return result
	}

	return nil
}
