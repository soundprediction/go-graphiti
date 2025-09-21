package edges_test

import (
	"testing"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityEdge(t *testing.T) {
	now := time.Now()
	groupID := "test-group"

	tests := []struct {
		name string
		edge *types.Edge
	}{
		{
			name: "basic entity edge",
			edge: &types.Edge{
				BaseEdge: types.BaseEdge{
					ID:           "edge-1",
					GroupID:      groupID,
					SourceNodeID: "entity-1",
					TargetNodeID: "entity-2",
					CreatedAt:    now,
				},
				Name:     "likes",
				Type:     types.EntityEdgeType,
				SourceID: "entity-1",
				TargetID: "entity-2",
				Summary:  "test_entity_1 relates to test_entity_2",
			},
		},
		{
			name: "edge with episodes",
			edge: &types.Edge{
				BaseEdge: types.BaseEdge{
					ID:           "edge-2",
					GroupID:      groupID,
					SourceNodeID: "entity-3",
					TargetNodeID: "entity-4",
					CreatedAt:    now,
					Metadata: map[string]interface{}{
						"episodes": []string{"episode-1", "episode-2"},
					},
				},
				Name:     "knows",
				Type:     types.EntityEdgeType,
				SourceID: "entity-3",
				TargetID: "entity-4",
				Summary:  "entity 3 knows entity 4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.edge.BaseEdge.ID)
			assert.NotEmpty(t, tt.edge.Name)
			assert.Equal(t, groupID, tt.edge.BaseEdge.GroupID)
			assert.Equal(t, types.EntityEdgeType, tt.edge.Type)
			assert.NotZero(t, tt.edge.BaseEdge.CreatedAt)
		})
	}
}

func TestEpisodicEdge(t *testing.T) {
	now := time.Now()
	groupID := "test-group"

	episodicEdge := &types.Edge{
		BaseEdge: types.BaseEdge{
			ID:           "episodic-edge-1",
			GroupID:      groupID,
			SourceNodeID: "episode-1",
			TargetNodeID: "entity-1",
			CreatedAt:    now,
			Metadata: map[string]interface{}{
				"source_type": "episode",
				"target_type": "entity",
			},
		},
		Type:     types.EpisodicEdgeType,
		SourceID: "episode-1",
		TargetID: "entity-1",
	}

	assert.Equal(t, types.EpisodicEdgeType, episodicEdge.Type)
	assert.Equal(t, groupID, episodicEdge.BaseEdge.GroupID)
	assert.NotZero(t, episodicEdge.BaseEdge.CreatedAt)
}

func TestCommunityEdge(t *testing.T) {
	now := time.Now()
	groupID := "test-group"

	communityEdge := &types.Edge{
		BaseEdge: types.BaseEdge{
			ID:           "community-edge-1",
			GroupID:      groupID,
			SourceNodeID: "entity-1",
			TargetNodeID: "community-1",
			CreatedAt:    now,
			Metadata: map[string]interface{}{
				"level":      0,
				"membership": 0.85,
			},
		},
		Name:     "belongs_to",
		Type:     types.CommunityEdgeType,
		SourceID: "entity-1",
		TargetID: "community-1",
	}

	assert.Equal(t, types.CommunityEdgeType, communityEdge.Type)
	assert.Equal(t, "belongs_to", communityEdge.Name)
	assert.Equal(t, groupID, communityEdge.BaseEdge.GroupID)
	assert.NotZero(t, communityEdge.BaseEdge.CreatedAt)

	// Test community-specific properties
	level, ok := communityEdge.BaseEdge.Metadata["level"]
	require.True(t, ok)
	assert.Equal(t, 0, level)

	membership, ok := communityEdge.BaseEdge.Metadata["membership"]
	require.True(t, ok)
	assert.Equal(t, 0.85, membership)
}

func TestEdgeValidation(t *testing.T) {
	tests := []struct {
		name    string
		edge    *types.Edge
		isValid bool
	}{
		{
			name: "valid edge",
			edge: &types.Edge{
				BaseEdge: types.BaseEdge{
					ID:           "valid-edge",
					GroupID:      "group",
					SourceNodeID: "source",
					TargetNodeID: "target",
					CreatedAt:    time.Now(),
				},
				Type:     types.EntityEdgeType,
				SourceID: "source",
				TargetID: "target",
			},
			isValid: true,
		},
		{
			name: "missing ID",
			edge: &types.Edge{
				BaseEdge: types.BaseEdge{
					GroupID:      "group",
					SourceNodeID: "source",
					TargetNodeID: "target",
					CreatedAt:    time.Now(),
				},
				Type:     types.EntityEdgeType,
				SourceID: "source",
				TargetID: "target",
			},
			isValid: false,
		},
		{
			name: "missing source",
			edge: &types.Edge{
				BaseEdge: types.BaseEdge{
					ID:           "edge-id",
					GroupID:      "group",
					TargetNodeID: "target",
					CreatedAt:    time.Now(),
				},
				Type:     types.EntityEdgeType,
				TargetID: "target",
			},
			isValid: false,
		},
		{
			name: "missing target",
			edge: &types.Edge{
				BaseEdge: types.BaseEdge{
					ID:           "edge-id",
					GroupID:      "group",
					SourceNodeID: "source",
					CreatedAt:    time.Now(),
				},
				Type:     types.EntityEdgeType,
				SourceID: "source",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.edge.BaseEdge.ID != "" &&
				tt.edge.BaseEdge.SourceNodeID != "" &&
				tt.edge.BaseEdge.TargetNodeID != "" &&
				tt.edge.BaseEdge.GroupID != ""

			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestEdgeTypes(t *testing.T) {
	edgeTypes := []types.EdgeType{
		types.EntityEdgeType,
		types.EpisodicEdgeType,
		types.CommunityEdgeType,
	}

	for _, edgeType := range edgeTypes {
		t.Run(string(edgeType), func(t *testing.T) {
			edge := &types.Edge{
				BaseEdge: types.BaseEdge{
					ID:           "test-edge",
					GroupID:      "group",
					SourceNodeID: "source",
					TargetNodeID: "target",
					CreatedAt:    time.Now(),
				},
				Type:     edgeType,
				SourceID: "source",
				TargetID: "target",
			}

			assert.Equal(t, edgeType, edge.Type)
		})
	}
}

func TestEdgeTimeOperations(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)

	edge := &types.Edge{
		BaseEdge: types.BaseEdge{
			ID:           "time-edge",
			GroupID:      "group",
			SourceNodeID: "source",
			TargetNodeID: "target",
			CreatedAt:    now,
			Metadata: map[string]interface{}{
				"invalid_at": future,
				"expired_at": nil,
			},
		},
		Type:      types.EntityEdgeType,
		SourceID:  "source",
		TargetID:  "target",
		ValidFrom: now,
	}

	// Test creation time
	assert.Equal(t, now, edge.BaseEdge.CreatedAt)

	// Test temporal properties
	assert.Equal(t, now, edge.ValidFrom)

	invalidAt, ok := edge.BaseEdge.Metadata["invalid_at"]
	require.True(t, ok)
	assert.Equal(t, future, invalidAt)

	// Test edge is valid at creation time
	createdAt := edge.BaseEdge.CreatedAt
	validTime := edge.ValidFrom
	invalidTime := edge.BaseEdge.Metadata["invalid_at"].(time.Time)

	assert.True(t, createdAt.Equal(validTime) || createdAt.After(validTime))
	assert.True(t, createdAt.Before(invalidTime))
}

func TestEdgeWithEmbedding(t *testing.T) {
	now := time.Now()

	// Mock embedding vector
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) * 0.1
	}

	edge := &types.Edge{
		BaseEdge: types.BaseEdge{
			ID:           "embedded-edge",
			GroupID:      "group",
			SourceNodeID: "source",
			TargetNodeID: "target",
			CreatedAt:    now,
		},
		Name:      "relates_to",
		Type:      types.EntityEdgeType,
		SourceID:  "source",
		TargetID:  "target",
		Summary:   "source relates to target",
		Embedding: embedding,
	}

	// Test embedding property
	assert.Len(t, edge.Embedding, 1536)
	assert.Equal(t, float32(0.0), edge.Embedding[0])
	assert.Equal(t, float32(0.1), edge.Embedding[1])
}