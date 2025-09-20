package types

import (
	"context"
)

// EdgeOperations provides methods for edge-related database operations
type EdgeOperations interface {
	ExecuteQuery(query string, params map[string]interface{}) (interface{}, interface{}, interface{}, error)
}

// DeleteByUUIDs replicates Edge.delete_by_uuids functionality from Python
func DeleteEdgesByUUIDs(ctx context.Context, driver EdgeOperations, uuids []string) error {
	if len(uuids) == 0 {
		return nil
	}

	// Match the Python Edge.delete_by_uuids implementation
	query := `
		MATCH (n)-[e:RELATES_TO|MENTIONS|HAS_MEMBER]->(m)
		WHERE e.uuid IN $uuids
		DELETE e
	`

	_, _, _, err := driver.ExecuteQuery(query, map[string]interface{}{
		"uuids": uuids,
	})

	return err
}

// GetEntityEdgesByUUIDs replicates EntityEdge.get_by_uuids functionality from Python
func GetEntityEdgesByUUIDs(ctx context.Context, driver EdgeOperations, uuids []string) ([]*Edge, error) {
	if len(uuids) == 0 {
		return []*Edge{}, nil
	}

	// Match the Python EntityEdge.get_by_uuids query
	query := `
		MATCH (n:Entity)-[e:RELATES_TO]->(m:Entity)
		WHERE e.uuid IN $uuids
		RETURN e.uuid AS uuid, e.source_id AS source_id, e.target_id AS target_id,
		       e.name AS name, e.fact AS fact, e.group_id AS group_id,
		       e.source_ids AS source_ids, e.created_at AS created_at,
		       e.expired_at AS expired_at, e.valid_at AS valid_at
	`

	records, _, _, err := driver.ExecuteQuery(query, map[string]interface{}{
		"uuids": uuids,
	})
	if err != nil {
		return nil, err
	}

	var edges []*Edge
	if recordList, ok := records.([]map[string]interface{}); ok {
		for _, record := range recordList {
			edge := &Edge{
				Type: EntityEdgeType,
			}
			
			if uuid, ok := record["uuid"].(string); ok {
				edge.ID = uuid
			}
			if sourceID, ok := record["source_id"].(string); ok {
				edge.SourceID = sourceID
			}
			if targetID, ok := record["target_id"].(string); ok {
				edge.TargetID = targetID
			}
			if name, ok := record["name"].(string); ok {
				edge.Name = name
			}
			if summary, ok := record["fact"].(string); ok {
				edge.Summary = summary
			}
			if groupID, ok := record["group_id"].(string); ok {
				edge.GroupID = groupID
			}
			// Map source_ids to SourceIDs to match Python edge.episodes
			if sourceIDs, ok := record["source_ids"].([]interface{}); ok {
				episodes := make([]string, len(sourceIDs))
				for i, ep := range sourceIDs {
					if epStr, ok := ep.(string); ok {
						episodes[i] = epStr
					}
				}
				edge.SourceIDs = episodes
			}
			
			edges = append(edges, edge)
		}
	}

	return edges, nil
}