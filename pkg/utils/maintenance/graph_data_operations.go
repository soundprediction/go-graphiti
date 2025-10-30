package maintenance

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

const (
	EpisodeWindowLen = 3
)

// GraphDataOperations provides graph data maintenance operations
type GraphDataOperations struct {
	driver driver.GraphDriver
}

// NewGraphDataOperations creates a new GraphDataOperations instance
func NewGraphDataOperations(driver driver.GraphDriver) *GraphDataOperations {
	return &GraphDataOperations{
		driver: driver,
	}
}

// BuildIndicesAndConstraints creates necessary indices and constraints for the graph database
func (gdo *GraphDataOperations) BuildIndicesAndConstraints(ctx context.Context, deleteExisting bool) error {
	log.Printf("Building indices and constraints (delete_existing: %v)", deleteExisting)

	// For now, use the driver's CreateIndices method which should handle the database-specific logic
	return gdo.driver.CreateIndices(ctx)
}

// RetrieveEpisodes retrieves the last n episodic nodes from the graph
func (gdo *GraphDataOperations) RetrieveEpisodes(ctx context.Context, referenceTime time.Time, lastN int, groupIDs []string, source string) ([]*types.Node, error) {
	if lastN <= 0 {
		lastN = EpisodeWindowLen
	}

	log.Printf("Retrieving %d episodes before %v for groups %v with source %s", lastN, referenceTime, groupIDs, source)

	// Use the driver's temporal operations to get nodes in time range
	// We'll get all nodes up to the reference time and then filter
	startTime := time.Time{} // Beginning of time
	nodes, err := gdo.driver.GetNodesInTimeRange(ctx, startTime, referenceTime, "")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve nodes in time range: %w", err)
	}

	// Filter for episodic nodes
	var episodic []*types.Node
	for _, node := range nodes {
		if node.Type == types.EpisodicNodeType {
			// Apply group ID filter if specified
			if len(groupIDs) > 0 {
				found := false
				for _, groupID := range groupIDs {
					if node.GroupID == groupID {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			// Apply source filter if specified
			if source != "" && string(node.EpisodeType) != source {
				continue
			}

			episodic = append(episodic, node)
		}
	}

	// Sort by ValidFrom time (most recent first) and limit
	// This is a simple bubble sort for small arrays
	for i := 0; i < len(episodic)-1; i++ {
		for j := 0; j < len(episodic)-i-1; j++ {
			if episodic[j].ValidFrom.Before(episodic[j+1].ValidFrom) {
				episodic[j], episodic[j+1] = episodic[j+1], episodic[j]
			}
		}
	}

	// Take the last N episodes
	if len(episodic) > lastN {
		episodic = episodic[:lastN]
	}

	// Reverse to return in chronological order
	for i, j := 0, len(episodic)-1; i < j; i, j = i+1, j-1 {
		episodic[i], episodic[j] = episodic[j], episodic[i]
	}

	log.Printf("Retrieved %d episodes", len(episodic))
	return episodic, nil
}

// ClearData removes all data from the graph or specific group IDs
func (gdo *GraphDataOperations) ClearData(ctx context.Context, groupIDs []string) error {
	log.Printf("Clearing data for groups: %v", groupIDs)

	if len(groupIDs) == 0 {
		// Clear all data - this is a dangerous operation, so we'll be cautious
		log.Println("Warning: Clearing all data from the graph")

		// Get all nodes and edges and delete them
		// This is a simplified approach - in production you might want a more efficient method
		allNodes, err := gdo.driver.SearchNodes(ctx, "", "", &driver.SearchOptions{Limit: 10000})
		if err != nil {
			return fmt.Errorf("failed to get all nodes: %w", err)
		}

		for _, node := range allNodes {
			if err := gdo.driver.DeleteNode(ctx, node.Uuid, node.GroupID); err != nil {
				log.Printf("Warning: failed to delete node %s: %v", node.Uuid, err)
			}
		}

		allEdges, err := gdo.driver.SearchEdges(ctx, "", "", &driver.SearchOptions{Limit: 10000})
		if err != nil {
			return fmt.Errorf("failed to get all edges: %w", err)
		}

		for _, edge := range allEdges {
			if err := gdo.driver.DeleteEdge(ctx, edge.Uuid, edge.GroupID); err != nil {
				log.Printf("Warning: failed to delete edge %s: %v", edge.Uuid, err)
			}
		}
	} else {
		// Clear data for specific group IDs
		for _, groupID := range groupIDs {
			// Get all nodes for this group
			nodes, err := gdo.driver.SearchNodes(ctx, "", groupID, &driver.SearchOptions{Limit: 10000})
			if err != nil {
				log.Printf("Warning: failed to get nodes for group %s: %v", groupID, err)
				continue
			}

			for _, node := range nodes {
				if err := gdo.driver.DeleteNode(ctx, node.ID, groupID); err != nil {
					log.Printf("Warning: failed to delete node %s: %v", node.ID, err)
				}
			}

			// Get all edges for this group
			edges, err := gdo.driver.SearchEdges(ctx, "", groupID, &driver.SearchOptions{Limit: 10000})
			if err != nil {
				log.Printf("Warning: failed to get edges for group %s: %v", groupID, err)
				continue
			}

			for _, edge := range edges {
				if err := gdo.driver.DeleteEdge(ctx, edge.ID, groupID); err != nil {
					log.Printf("Warning: failed to delete edge %s: %v", edge.ID, err)
				}
			}
		}
	}

	log.Printf("Data clearing completed for groups: %v", groupIDs)
	return nil
}

// GetStats returns basic statistics about the graph
func (gdo *GraphDataOperations) GetStats(ctx context.Context, groupID string) (*driver.GraphStats, error) {
	return gdo.driver.GetStats(ctx, groupID)
}
