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
				ID:           "edge-1",
				Name:         "likes",
				Type:         types.EntityEdgeType,
				SourceID:     "entity-1",
				TargetID:     "entity-2",
				GroupID:      groupID,
				CreatedAt:    now,
				Summary:      "test_entity_1 relates to test_entity_2",
			},
		},
		{
			name: "edge with episodes",
			edge: &types.Edge{
				ID:           "edge-2",
				Name:         "knows",
				Type:         types.EntityEdgeType,
				SourceID:     "entity-3",
				TargetID:     "entity-4",
				GroupID:      groupID,
				CreatedAt:    now,
				Summary:      "entity 3 knows entity 4",
				Metadata: map[string]interface{}{
					"episodes": []string{"episode-1", "episode-2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.edge.ID)
			assert.NotEmpty(t, tt.edge.Name)
			assert.Equal(t, groupID, tt.edge.GroupID)
			assert.Equal(t, types.EntityEdgeType, tt.edge.Type)
			assert.NotZero(t, tt.edge.CreatedAt)
		})
	}
}

func TestEpisodicEdge(t *testing.T) {
	now := time.Now()
	groupID := "test-group"

	episodicEdge := &types.Edge{
		ID:           "episodic-edge-1",
		Type:         types.EpisodicEdgeType,
		SourceID:     "episode-1",
		TargetID:     "entity-1",
		GroupID:      groupID,
		CreatedAt:    now,
		Metadata: map[string]interface{}{
			"source_type": "episode",
			"target_type": "entity",
		},
	}

	assert.Equal(t, types.EpisodicEdgeType, episodicEdge.Type)
	assert.Equal(t, groupID, episodicEdge.GroupID)
	assert.NotZero(t, episodicEdge.CreatedAt)
}

func TestCommunityEdge(t *testing.T) {
	now := time.Now()
	groupID := "test-group"

	communityEdge := &types.Edge{
		ID:           "community-edge-1",
		Name:         "belongs_to",
		Type:         types.CommunityEdgeType,
		SourceID:     "entity-1",
		TargetID:     "community-1",
		GroupID:      groupID,
		CreatedAt:    now,
		Metadata: map[string]interface{}{
			"level":      0,
			"membership": 0.85,
		},
	}

	assert.Equal(t, types.CommunityEdgeType, communityEdge.Type)
	assert.Equal(t, "belongs_to", communityEdge.Name)
	assert.Equal(t, groupID, communityEdge.GroupID)
	assert.NotZero(t, communityEdge.CreatedAt)

	// Test community-specific properties
	level, ok := communityEdge.Metadata["level"]
	require.True(t, ok)
	assert.Equal(t, 0, level)

	membership, ok := communityEdge.Metadata["membership"]
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
				ID:           "valid-edge",
				Type:         types.EntityEdgeType,
				SourceID:"source",
				TargetID:"target",
				GroupID:      "group",
				CreatedAt:    time.Now(),
			},
			isValid: true,
		},
		{
			name: "missing ID",
			edge: &types.Edge{
				Type:         types.EntityEdgeType,
				SourceID:"source",
				TargetID:"target",
				GroupID:      "group",
				CreatedAt:    time.Now(),
			},
			isValid: false,
		},
		{
			name: "missing source",
			edge: &types.Edge{
				ID:           "edge-id",
				Type:         types.EntityEdgeType,
				TargetID:"target",
				GroupID:      "group",
				CreatedAt:    time.Now(),
			},
			isValid: false,
		},
		{
			name: "missing target",
			edge: &types.Edge{
				ID:           "edge-id",
				Type:         types.EntityEdgeType,
				SourceID:"source",
				GroupID:      "group",
				CreatedAt:    time.Now(),
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.edge.ID != "" &&
				tt.edge.SourceID != "" &&
				tt.edge.TargetID != "" &&
				tt.edge.GroupID != ""

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
				ID:           "test-edge",
				Type:         edgeType,
				SourceID:"source",
				TargetID:"target",
				GroupID:      "group",
				CreatedAt:    time.Now(),
			}

			assert.Equal(t, edgeType, edge.Type)
		})
	}
}

func TestEdgeTimeOperations(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)

	edge := &types.Edge{
		ID:           "time-edge",
		Type:         types.EntityEdgeType,
		SourceID:"source",
		TargetID:"target",
		GroupID:      "group",
		CreatedAt:    now,
		ValidFrom:    now,
		Metadata: map[string]interface{}{
			"invalid_at": future,
			"expired_at": nil,
		},
	}

	// Test creation time
	assert.Equal(t, now, edge.CreatedAt)

	// Test temporal properties
	assert.Equal(t, now, edge.ValidFrom)

	invalidAt, ok := edge.Metadata["invalid_at"]
	require.True(t, ok)
	assert.Equal(t, future, invalidAt)

	// Test edge is valid at creation time
	createdAt := edge.CreatedAt
	validTime := edge.ValidFrom
	invalidTime := edge.Metadata["invalid_at"].(time.Time)

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
		ID:           "embedded-edge",
		Name:         "relates_to",
		Type:         types.EntityEdgeType,
		SourceID:"source",
		TargetID:"target",
		GroupID:      "group",
		CreatedAt:    now,
		Summary:   "source relates to target",
		Embedding: embedding,
	}

	// Test embedding property
	assert.Len(t, edge.Embedding, 1536)
	assert.Equal(t, float32(0.0), edge.Embedding[0])
	assert.Equal(t, float32(0.1), edge.Embedding[1])
}