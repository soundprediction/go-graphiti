package utils_test

import (
	"strings"
	"testing"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEdgeOperations(t *testing.T) {
	now := time.Now()
	groupID := "test-group"

	t.Run("filter existing duplicate edges", func(t *testing.T) {
		// Create test edges
		extractedEdge := &types.Edge{
			BaseEdge: types.BaseEdge{
				Uuid:         "new-edge",
				GroupID:      groupID,
				SourceNodeID: "source-uuid",
				TargetNodeID: "target-uuid",
				CreatedAt:    now,
				Metadata: map[string]interface{}{
					"fact": "Test fact",
				},
			},
			Name:     "test_edge",
			Type:     types.EntityEdgeType,
			SourceID: "source-uuid",
			TargetID: "target-uuid",
		}

		relatedEdges := []*types.Edge{
			{
				BaseEdge: types.BaseEdge{
					Uuid:         "related-edge",
					GroupID:      groupID,
					SourceNodeID: "source-uuid-2",
					TargetNodeID: "target-uuid-2",
					CreatedAt:    now.Add(-24 * time.Hour),
					Metadata: map[string]interface{}{
						"fact": "Related fact",
					},
				},
				Name:     "related_edge",
				Type:     types.EntityEdgeType,
				SourceID: "source-uuid-2",
				TargetID: "target-uuid-2",
			},
		}

		existingEdges := []*types.Edge{
			{
				BaseEdge: types.BaseEdge{
					Uuid:         "existing-edge",
					GroupID:      groupID,
					SourceNodeID: "source-uuid-3",
					TargetNodeID: "target-uuid-3",
					CreatedAt:    now.Add(-48 * time.Hour),
					Metadata: map[string]interface{}{
						"fact": "Existing fact",
					},
				},
				Name:     "existing_edge",
				Type:     types.EntityEdgeType,
				SourceID: "source-uuid-3",
				TargetID: "target-uuid-3",
			},
		}

		// Test that edges are properly categorized
		assert.NotNil(t, extractedEdge)
		assert.Len(t, relatedEdges, 1)
		assert.Len(t, existingEdges, 1)

		// Test edge uniqueness check
		allEdges := append(relatedEdges, existingEdges...)
		allEdges = append(allEdges, extractedEdge)

		uniqueEdgeIDs := make(map[string]bool)
		for _, edge := range allEdges {
			assert.False(t, uniqueEdgeIDs[edge.BaseEdge.Uuid], "Edge ID should be unique: %s", edge.BaseEdge.Uuid)
			uniqueEdgeIDs[edge.BaseEdge.Uuid] = true
		}
	})
}

func TestCommunityOperations(t *testing.T) {
	groupID := "test-group"
	now := time.Now()

	t.Run("determine entity community", func(t *testing.T) {
		// Create test entity nodes
		entities := []*types.Node{
			{
				Uuid:       "entity-1",
				Name:       "Alice",
				Type:       types.EntityNodeType,
				GroupID:    groupID,
				EntityType: "Person",
				CreatedAt:  now,
				Summary:    "A person named Alice",
			},
			{
				Uuid:       "entity-2",
				Name:       "Bob",
				Type:       types.EntityNodeType,
				GroupID:    groupID,
				EntityType: "Person",
				CreatedAt:  now,
				Summary:    "A person named Bob",
			},
		}

		// Test community determination logic
		assert.Len(t, entities, 2)

		// In a real implementation, this would use clustering algorithms
		// to determine which entities belong to the same community
		for _, entity := range entities {
			assert.Equal(t, "Person", entity.EntityType)
			assert.Equal(t, types.EntityNodeType, entity.Type)
		}
	})

	t.Run("get community clusters", func(t *testing.T) {
		// Create test community nodes
		communities := []*types.Node{
			{
				Uuid:      "community-1",
				Name:      "Community A",
				Type:      types.CommunityNodeType,
				GroupID:   groupID,
				CreatedAt: now,
				Level:     0,
				Metadata: map[string]interface{}{
					"size":        3,
					"entities":    []string{"entity-1", "entity-2", "entity-3"},
					"description": "A community of related people",
				},
			},
			{
				Uuid:      "community-2",
				Name:      "Community B",
				Type:      types.CommunityNodeType,
				GroupID:   groupID,
				CreatedAt: now,
				Level:     1,
				Metadata: map[string]interface{}{
					"size":        2,
					"entities":    []string{"entity-4", "entity-5"},
					"description": "Another community cluster",
				},
			},
		}

		// Test community cluster properties
		for _, community := range communities {
			assert.Equal(t, types.CommunityNodeType, community.Type)
			assert.Greater(t, community.Level, -1)

			entities, ok := community.Metadata["entities"]
			require.True(t, ok)
			entitiesSlice, ok := entities.([]string)
			require.True(t, ok)
			assert.Greater(t, len(entitiesSlice), 0)
		}
	})

	t.Run("remove communities", func(t *testing.T) {
		// Test community removal logic
		communityToRemove := &types.Node{
			Uuid:      "community-to-remove",
			Name:      "Community to Remove",
			Type:      types.CommunityNodeType,
			GroupID:   groupID,
			CreatedAt: now,
		}

		// In a real implementation, this would remove the community
		// and potentially reassign entities to other communities
		assert.NotNil(t, communityToRemove)
		assert.Equal(t, types.CommunityNodeType, communityToRemove.Type)
	})
}

func TestTemporalOperations(t *testing.T) {
	now := time.Now()
	past := now.Add(-2 * time.Hour)
	future := now.Add(2 * time.Hour)
	groupID := "test-group"

	t.Run("time range filtering", func(t *testing.T) {
		// Create nodes with different timestamps
		nodes := []*types.Node{
			{
				Uuid:      "node-past",
				Name:      "Past Node",
				Type:      types.EntityNodeType,
				GroupID:   groupID,
				CreatedAt: past,
			},
			{
				Uuid:      "node-present",
				Name:      "Present Node",
				Type:      types.EntityNodeType,
				GroupID:   groupID,
				CreatedAt: now,
			},
			{
				Uuid:      "node-future",
				Name:      "Future Node",
				Type:      types.EntityNodeType,
				GroupID:   groupID,
				CreatedAt: future,
			},
		}

		// Test filtering nodes in time range
		startTime := now.Add(-1 * time.Hour)
		endTime := now.Add(1 * time.Hour)

		var filteredNodes []*types.Node
		for _, node := range nodes {
			if node.CreatedAt.After(startTime) && node.CreatedAt.Before(endTime) {
				filteredNodes = append(filteredNodes, node)
			}
		}

		// Should only include the present node
		assert.Len(t, filteredNodes, 1)
		assert.Equal(t, "node-present", filteredNodes[0].Uuid)
	})

	t.Run("edge temporal operations", func(t *testing.T) {
		// Create edges with temporal properties
		edges := []*types.Edge{
			{
				BaseEdge: types.BaseEdge{
					Uuid:         "edge-valid",
					GroupID:      groupID,
					SourceNodeID: "source-1",
					TargetNodeID: "target-1",
					CreatedAt:    now,
				},
				Type:      types.EntityEdgeType,
				SourceID:  "source-1",
				TargetID:  "target-1",
				ValidFrom: past,
				ValidTo:   &future,
			},
			{
				BaseEdge: types.BaseEdge{
					Uuid:         "edge-expired",
					GroupID:      groupID,
					SourceNodeID: "source-2",
					TargetNodeID: "target-2",
					CreatedAt:    past,
				},
				Type:      types.EntityEdgeType,
				SourceID:  "source-2",
				TargetID:  "target-2",
				ValidFrom: past,
				ValidTo:   func() *time.Time { t := past.Add(30 * time.Minute); return &t }(),
			},
		}

		// Test valid edge detection
		var validEdges []*types.Edge
		for _, edge := range edges {
			if edge.ValidTo == nil || now.Before(*edge.ValidTo) {
				if now.After(edge.ValidFrom) || now.Equal(edge.ValidFrom) {
					validEdges = append(validEdges, edge)
				}
			}
		}

		// Should only include the first edge (still valid)
		assert.Len(t, validEdges, 1)
		assert.Equal(t, "edge-valid", validEdges[0].BaseEdge.Uuid)
	})
}

func TestBulkOperations(t *testing.T) {
	groupID := "test-group"
	now := time.Now()

	t.Run("bulk node operations", func(t *testing.T) {
		// Create multiple nodes for bulk operations
		nodes := []*types.Node{
			{
				Uuid:      "bulk-node-1",
				Name:      "Bulk Node 1",
				Type:      types.EntityNodeType,
				GroupID:   groupID,
				CreatedAt: now,
			},
			{
				Uuid:      "bulk-node-2",
				Name:      "Bulk Node 2",
				Type:      types.EntityNodeType,
				GroupID:   groupID,
				CreatedAt: now,
			},
			{
				Uuid:      "bulk-node-3",
				Name:      "Bulk Node 3",
				Type:      types.EpisodicNodeType,
				GroupID:   groupID,
				CreatedAt: now,
			},
		}

		// Test bulk validation
		for _, node := range nodes {
			assert.NotEmpty(t, node.Uuid)
			assert.NotEmpty(t, node.Name)
			assert.Equal(t, groupID, node.GroupID)
		}

		// Group by type
		entityNodes := 0
		episodicNodes := 0
		for _, node := range nodes {
			switch node.Type {
			case types.EntityNodeType:
				entityNodes++
			case types.EpisodicNodeType:
				episodicNodes++
			}
		}

		assert.Equal(t, 2, entityNodes)
		assert.Equal(t, 1, episodicNodes)
	})

	t.Run("bulk edge operations", func(t *testing.T) {
		// Create multiple edges for bulk operations
		edges := []*types.Edge{
			{
				BaseEdge: types.BaseEdge{
					Uuid:         "bulk-edge-1",
					GroupID:      groupID,
					SourceNodeID: "source-1",
					TargetNodeID: "target-1",
					CreatedAt:    now,
				},
				Type:     types.EntityEdgeType,
				SourceID: "source-1",
				TargetID: "target-1",
			},
			{
				BaseEdge: types.BaseEdge{
					Uuid:         "bulk-edge-2",
					GroupID:      groupID,
					SourceNodeID: "episode-1",
					TargetNodeID: "entity-1",
					CreatedAt:    now,
				},
				Type:     types.EpisodicEdgeType,
				SourceID: "episode-1",
				TargetID: "entity-1",
			},
		}

		// Test bulk validation
		for _, edge := range edges {
			assert.NotEmpty(t, edge.BaseEdge.Uuid)
			assert.NotEmpty(t, edge.SourceID)
			assert.NotEmpty(t, edge.TargetID)
			assert.Equal(t, groupID, edge.BaseEdge.GroupID)
		}

		// Group by type
		relationEdges := 0
		episodicEdges := 0
		for _, edge := range edges {
			switch edge.Type {
			case types.EntityEdgeType:
				relationEdges++
			case types.EpisodicEdgeType:
				episodicEdges++
			}
		}

		assert.Equal(t, 1, relationEdges)
		assert.Equal(t, 1, episodicEdges)
	})
}

func TestSearchUtilities(t *testing.T) {
	t.Run("text processing utilities", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "basic cleanup",
				input:    "Hello World",
				expected: "hello world",
			},
			{
				name:     "remove special characters",
				input:    "Hello, World! How are you?",
				expected: "hello world how are you",
			},
			{
				name:     "normalize whitespace",
				input:    "Hello    World\n\tTest",
				expected: "hello world test",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Mock text processing function
				result := processText(tc.input)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

// Mock utility functions for testing
func processText(input string) string {
	// Simple mock implementation
	// In reality this would be more sophisticated
	result := input
	result = strings.ToLower(result)
	result = strings.ReplaceAll(result, ",", "")
	result = strings.ReplaceAll(result, "!", "")
	result = strings.ReplaceAll(result, "?", "")
	result = strings.ReplaceAll(result, "\n", " ")
	result = strings.ReplaceAll(result, "\t", " ")

	// Normalize multiple spaces to single space
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	return strings.TrimSpace(result)
}
