package types

import (
	"context"
	"fmt"
)

// NodeOperations provides methods for node-related database operations
type NodeOperations interface {
	ExecuteQuery(query string, params map[string]interface{}) (interface{}, interface{}, interface{}, error)
}

// GetEpisodicNodeByUUID replicates EpisodicNode.get_by_uuid functionality from Python
func GetEpisodicNodeByUUID(ctx context.Context, driver NodeOperations, uuid string) (*Node, error) {
	// Match the Python EpisodicNode.get_by_uuid query
	query := `
		MATCH (e:Episodic {uuid: $uuid})
		RETURN e.uuid AS uuid, e.name AS name, e.source AS source,
		       e.source_description AS source_description, e.content AS content,
		       e.valid_at AS valid_at, e.entity_edges AS entity_edges,
		       e.group_id AS group_id, e.created_at AS created_at
	`

	records, _, _, err := driver.ExecuteQuery(query, map[string]interface{}{
		"uuid": uuid,
	})
	if err != nil {
		return nil, err
	}

	recordList, ok := records.([]map[string]interface{})
	if !ok || len(recordList) == 0 {
		return nil, fmt.Errorf("episode with UUID %s not found", uuid)
	}

	record := recordList[0]
	episode := &Node{
		Type: EpisodicNodeType,
	}

	if id, ok := record["uuid"].(string); ok {
		episode.Uuid = id
	}
	if name, ok := record["name"].(string); ok {
		episode.Name = name
	}
	if content, ok := record["content"].(string); ok {
		episode.Content = content
	}
	if groupID, ok := record["group_id"].(string); ok {
		episode.GroupID = groupID
	}
	if sourceDesc, ok := record["source_description"].(string); ok {
		episode.Summary = sourceDesc // Map to summary field
	}

	// Handle entity_edges - this is critical for the remove_episode logic
	if entityEdges, ok := record["entity_edges"].([]interface{}); ok {
		edges := make([]string, len(entityEdges))
		for i, edge := range entityEdges {
			if edgeStr, ok := edge.(string); ok {
				edges[i] = edgeStr
			}
		}
		episode.EntityEdges = edges
	}

	return episode, nil
}

// DeleteNode replicates the Python Node.delete() method functionality
func DeleteNode(ctx context.Context, driver NodeOperations, node *Node) error {
	// Match the Python Node.delete() implementation
	query := `
		MATCH (n {uuid: $uuid})
		WHERE n:Entity OR n:Episodic OR n:Community
		OPTIONAL MATCH (n)-[r]-()
		WITH collect(r.uuid) AS edge_uuids, n
		DETACH DELETE n
		RETURN edge_uuids
	`

	_, _, _, err := driver.ExecuteQuery(query, map[string]interface{}{
		"uuid": node.ID,
	})

	return err
}

// DeleteByUUIDs replicates Node.delete_by_uuids functionality from Python
func DeleteNodesByUUIDs(ctx context.Context, driver NodeOperations, uuids []string) error {
	if len(uuids) == 0 {
		return nil
	}

	// Match the Python Node.delete_by_uuids implementation
	// Try different node labels as in the Python version
	labels := []string{"Entity", "Episodic", "Community"}

	for _, label := range labels {
		query := fmt.Sprintf(`
			MATCH (n:%s)
			WHERE n.uuid IN $uuids
			DETACH DELETE n
		`, label)

		_, _, _, err := driver.ExecuteQuery(query, map[string]interface{}{
			"uuids": uuids,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// GetMentionedNodes replicates get_mentioned_nodes functionality from Python
func GetMentionedNodes(ctx context.Context, driver NodeOperations, episodes []*Node) ([]*Node, error) {
	if len(episodes) == 0 {
		return []*Node{}, nil
	}

	episodeUUIDs := make([]string, len(episodes))
	for i, episode := range episodes {
		episodeUUIDs[i] = episode.ID
	}

	// Match the Python get_mentioned_nodes query
	query := `
		MATCH (episode:Episodic)-[:MENTIONS]->(n:Entity)
		WHERE episode.uuid IN $uuids
		RETURN DISTINCT n.uuid AS uuid, n.name AS name, n.entity_type AS entity_type,
		       n.summary AS summary, n.group_id AS group_id
	`

	records, _, _, err := driver.ExecuteQuery(query, map[string]interface{}{
		"uuids": episodeUUIDs,
	})
	if err != nil {
		return nil, err
	}

	var nodes []*Node
	if recordList, ok := records.([]map[string]interface{}); ok {
		for _, record := range recordList {
			node := &Node{
				Type: EntityNodeType,
			}

			if uuid, ok := record["uuid"].(string); ok {
				node.ID = uuid
			}
			if name, ok := record["name"].(string); ok {
				node.Name = name
			}
			if entityType, ok := record["entity_type"].(string); ok {
				node.EntityType = entityType
			}
			if summary, ok := record["summary"].(string); ok {
				node.Summary = summary
			}
			if groupID, ok := record["group_id"].(string); ok {
				node.GroupID = groupID
			}

			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}
