package community

import (
	"context"
	"fmt"

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
			maxCount := 0

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

	// Default Neo4j query
	query := `
		MATCH (n:Entity {uuid: $uuid, group_id: $group_id})-[e:RELATES_TO]-(m:Entity {group_id: $group_id})
		WITH count(e) AS count, m.uuid AS uuid
		RETURN uuid, count
	`

	params := map[string]interface{}{
		"uuid":     nodeUUID,
		"group_id": groupID,
	}

	// For other drivers, we'd need to implement the query execution
	// This is a simplified version - in a real implementation, you'd handle this properly
	return nil, fmt.Errorf("non-Kuzu drivers not yet supported for neighbor queries")
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

	return nil, fmt.Errorf("non-Kuzu drivers not yet supported for getting group IDs")
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

	return nil, fmt.Errorf("non-Kuzu drivers not yet supported for getting entity nodes")
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