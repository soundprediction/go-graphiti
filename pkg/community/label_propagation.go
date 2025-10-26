package community

import (
	"context"
	"fmt"
	"reflect"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// labelPropagation implements the label propagation community detection algorithm
func (b *Builder) labelPropagation(projection map[string][]Neighbor) [][]string {
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
func (b *Builder) buildProjection(ctx context.Context, nodes []*types.Node, groupID string) (map[string][]Neighbor, error) {
	projection := make(map[string][]Neighbor)

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
func (b *Builder) getNodeNeighbors(ctx context.Context, nodeUUID, groupID string) ([]Neighbor, error) {
	// Check if this is a Kuzu driver to use the appropriate query
	if kuzuDriver, ok := b.driver.(*driver.KuzuDriver); ok {
		return b.getNodeNeighborsKuzu(ctx, kuzuDriver, nodeUUID, groupID)
	}

	// For Neo4j/Memgraph drivers
	return b.getNodeNeighborsNeo4j(ctx, nodeUUID, groupID)
}

// getNodeNeighborsKuzu gets neighbors specifically for Kuzu database
func (b *Builder) getNodeNeighborsKuzu(ctx context.Context, kuzuDriver *driver.KuzuDriver, nodeUUID, groupID string) ([]Neighbor, error) {
	query := `
		MATCH (n:Entity {uuid: $uuid, group_id: $group_id})-[:RELATES_TO]-(e:RelatesToNode_)-[:RELATES_TO]-(m:Entity {group_id: $group_id})
		WITH count(e) AS count, m.uuid AS uuid
		RETURN uuid, count
	`

	params := map[string]interface{}{
		"uuid":     nodeUUID,
		"group_id": groupID,
	}

	records, _, _, err := kuzuDriver.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute neighbor query: %w", err)
	}

	var neighbors []Neighbor
	recordSlice, ok := records.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected records type: %T", records)
	}
	for _, record := range recordSlice {
		if uuid, ok := record["uuid"].(string); ok {
			if count, ok := record["count"].(int64); ok {
				neighbors = append(neighbors, Neighbor{
					NodeUUID:  uuid,
					EdgeCount: int(count),
				})
			}
		}
	}

	return neighbors, nil
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

// getNodeNeighborsNeo4j gets neighbors for Neo4j/Memgraph databases
func (b *Builder) getNodeNeighborsNeo4j(ctx context.Context, nodeUUID, groupID string) ([]Neighbor, error) {
	query := `
		MATCH (n:Entity {uuid: $uuid, group_id: $group_id})-[:RELATES_TO]-(e:RelatesToNode)-[:RELATES_TO]-(m:Entity {group_id: $group_id})
		WITH count(e) AS count, m.uuid AS uuid
		RETURN uuid, count
	`

	params := map[string]interface{}{
		"uuid":     nodeUUID,
		"group_id": groupID,
	}

	result, _, _, err := b.driver.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute neighbor query: %w", err)
	}

	return b.parseNeighborsFromRecords(result)
}

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
		RETURN n.uuid AS uuid, n.name AS name, n.summary AS summary, n.created_at AS created_at
	`

	params := map[string]interface{}{
		"group_id": groupID,
	}

	result, _, _, err := b.driver.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute entity nodes query: %w", err)
	}

	return b.parseEntityNodesFromRecords(result, groupID)
}

// ====== Record Parsing Helpers ======

// parseNeighborsFromRecords parses Neo4j/Memgraph records into neighbors
func (b *Builder) parseNeighborsFromRecords(result interface{}) ([]Neighbor, error) {
	var neighbors []Neighbor

	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %T", result)
	}

	for i := 0; i < value.Len(); i++ {
		record := value.Index(i)

		// Get uuid field
		getMethod := record.MethodByName("Get")
		if !getMethod.IsValid() {
			continue
		}

		// Get uuid
		uuidResults := getMethod.Call([]reflect.Value{reflect.ValueOf("uuid")})
		if len(uuidResults) < 1 {
			continue
		}

		// Get count
		countResults := getMethod.Call([]reflect.Value{reflect.ValueOf("count")})
		if len(countResults) < 1 {
			continue
		}

		uuidInterface := uuidResults[0].Interface()
		countInterface := countResults[0].Interface()

		if uuid, ok := uuidInterface.(string); ok {
			count := int64(0)
			switch c := countInterface.(type) {
			case int64:
				count = c
			case int:
				count = int64(c)
			}

			neighbors = append(neighbors, Neighbor{
				NodeUUID:  uuid,
				EdgeCount: int(count),
			})
		}
	}

	return neighbors, nil
}

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

// parseEntityNodesFromRecords parses entity nodes from Neo4j/Memgraph records
func (b *Builder) parseEntityNodesFromRecords(result interface{}, groupID string) ([]*types.Node, error) {
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

		node := &types.Node{
			Type:    types.EntityNodeType,
			GroupID: groupID,
		}

		// Get uuid
		if uuidResults := getMethod.Call([]reflect.Value{reflect.ValueOf("uuid")}); len(uuidResults) > 0 {
			if uuid, ok := uuidResults[0].Interface().(string); ok {
				node.ID = uuid
			}
		}

		// Get name
		if nameResults := getMethod.Call([]reflect.Value{reflect.ValueOf("name")}); len(nameResults) > 0 {
			if name, ok := nameResults[0].Interface().(string); ok {
				node.Name = name
			}
		}

		// Get summary
		if summaryResults := getMethod.Call([]reflect.Value{reflect.ValueOf("summary")}); len(summaryResults) > 0 {
			if summary, ok := summaryResults[0].Interface().(string); ok {
				node.Summary = summary
			}
		}

		if node.ID != "" {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}
