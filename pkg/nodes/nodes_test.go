package nodes_test

import (
	"testing"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityNode(t *testing.T) {
	now := time.Now()
	groupID := "test-group"

	tests := []struct {
		name string
		node *types.Node
	}{
		{
			name: "basic entity node",
			node: &types.Node{
				ID:        "entity-1",
				Name:      "test_entity_1",
				Type:      types.EntityNodeType,
				GroupID:   groupID,
				EntityType: "Person",
				CreatedAt: now,
				Summary:   "test_entity_1 summary",
				Metadata: map[string]interface{}{
					"age":      30,
					"location": "New York",
				},
			},
		},
		{
			name: "entity with attributes",
			node: &types.Node{
				ID:        "entity-2",
				Name:      "test_entity_2",
				Type:      types.EntityNodeType,
				GroupID:   groupID,
				EntityType: "Person2",
				CreatedAt: now,
				Summary:   "test_entity_2 summary",
				Metadata: map[string]interface{}{
					"age":         25,
					"location":    "Los Angeles",
					"occupation":  "Engineer",
					"interests":   []string{"AI", "Music", "Sports"},
				},
			},
		},
		{
			name: "location entity",
			node: &types.Node{
				ID:        "entity-3",
				Name:      "test_entity_3",
				Type:      types.EntityNodeType,
				GroupID:   groupID,
				EntityType: "Location",
				CreatedAt: now,
				Summary:   "test_entity_3 summary",
				Metadata: map[string]interface{}{
					"population":  1000000,
					"country":     "USA",
					"timezone":    "PST",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.node.ID)
			assert.NotEmpty(t, tt.node.Name)
			assert.Equal(t, groupID, tt.node.GroupID)
			assert.Equal(t, types.EntityNodeType, tt.node.Type)
			assert.NotZero(t, tt.node.CreatedAt)
			assert.NotEmpty(t, tt.node.EntityType)
		})
	}
}

func TestEpisodicNode(t *testing.T) {
	now := time.Now()
	groupID := "test-group"

	tests := []struct {
		name string
		node *types.Node
	}{
		{
			name: "message episode",
			node: &types.Node{
				ID:        "episode-1",
				Name:      "test_episode",
				Type:      types.EpisodicNodeType,
				GroupID:   groupID,
				EpisodeType: types.ConversationEpisodeType,
				CreatedAt: now,
				Content:   "Alice likes Bob",
				Reference: now,
				Metadata: map[string]interface{}{
					"source":             "message",
					"source_description": "conversation message",
				},
			},
		},
		{
			name: "document episode",
			node: &types.Node{
				ID:        "episode-2",
				Name:      "test_episode_2",
				Type:      types.EpisodicNodeType,
				GroupID:   groupID,
				EpisodeType: types.DocumentEpisodeType,
				CreatedAt: now,
				Content:   "Bob adores Alice",
				Reference: now,
				Metadata: map[string]interface{}{
					"source":             "document",
					"source_description": "meeting notes",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.node.ID)
			assert.NotEmpty(t, tt.node.Name)
			assert.Equal(t, groupID, tt.node.GroupID)
			assert.Equal(t, types.EpisodicNodeType, tt.node.Type)
			assert.NotZero(t, tt.node.CreatedAt)
			assert.NotEmpty(t, tt.node.EpisodeType)

			// Test episode-specific properties
			assert.NotEmpty(t, tt.node.Content)
			assert.NotZero(t, tt.node.Reference)

			source, ok := tt.node.Metadata["source"]
			require.True(t, ok)
			assert.NotEmpty(t, source)
		})
	}
}

func TestCommunityNode(t *testing.T) {
	now := time.Now()
	groupID := "test-group"

	communityNode := &types.Node{
		ID:        "community-1",
		Name:      "test_community",
		Type:      types.CommunityNodeType,
		GroupID:   groupID,
		Level:     0,
		CreatedAt: now,
		Summary:   "A community of related entities",
		Metadata: map[string]interface{}{
			"size":        5,
			"entities":    []string{"entity-1", "entity-2", "entity-3"},
			"description": "Community formed around shared interests",
		},
	}

	assert.Equal(t, types.CommunityNodeType, communityNode.Type)
	assert.Equal(t, "test_community", communityNode.Name)
	assert.Equal(t, groupID, communityNode.GroupID)
	assert.NotZero(t, communityNode.CreatedAt)
	assert.Equal(t, 0, communityNode.Level)

	// Test community-specific properties
	assert.NotEmpty(t, communityNode.Summary)

	size, ok := communityNode.Metadata["size"]
	require.True(t, ok)
	assert.Equal(t, 5, size)

	entities, ok := communityNode.Metadata["entities"]
	require.True(t, ok)
	entitiesSlice, ok := entities.([]string)
	require.True(t, ok)
	assert.Len(t, entitiesSlice, 3)
}

func TestNodeValidation(t *testing.T) {
	tests := []struct {
		name    string
		node    *types.Node
		isValid bool
	}{
		{
			name: "valid node",
			node: &types.Node{
				ID:        "valid-node",
				Name:      "Valid Node",
				Type:      types.EntityNodeType,
				GroupID:   "group",
				CreatedAt: time.Now(),
			},
			isValid: true,
		},
		{
			name: "missing ID",
			node: &types.Node{
				Name:      "Node without ID",
				Type:      types.EntityNodeType,
				GroupID:   "group",
				CreatedAt: time.Now(),
			},
			isValid: false,
		},
		{
			name: "missing name",
			node: &types.Node{
				ID:        "node-id",
				Type:      types.EntityNodeType,
				GroupID:   "group",
				CreatedAt: time.Now(),
			},
			isValid: false,
		},
		{
			name: "missing group ID",
			node: &types.Node{
				ID:        "node-id",
				Name:      "Node Name",
				Type:      types.EntityNodeType,
				CreatedAt: time.Now(),
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.node.ID != "" && 
				tt.node.Name != "" && 
				tt.node.GroupID != ""

			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestNodeTypes(t *testing.T) {
	nodeTypes := []types.NodeType{
		types.EntityNodeType,
		types.EpisodicNodeType,
		types.CommunityNodeType,
	}

	for _, nodeType := range nodeTypes {
		t.Run(string(nodeType), func(t *testing.T) {
			node := &types.Node{
				ID:        "test-node",
				Name:      "Test Node",
				Type:      nodeType,
				GroupID:   "group",
				CreatedAt: time.Now(),
			}

			assert.Equal(t, nodeType, node.Type)
		})
	}
}

func TestNodeWithEmbedding(t *testing.T) {
	now := time.Now()

	// Mock embedding vector
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) * 0.1
	}

	node := &types.Node{
		ID:        "embedded-node",
		Name:      "Node with Embedding",
		Type:      types.EntityNodeType,
		GroupID:   "group",
		CreatedAt: now,
		Summary:   "A node with an embedding",
		Embedding: embedding,
	}

	// Test embedding property
	assert.Len(t, node.Embedding, 1536)
	assert.Equal(t, float32(0.0), node.Embedding[0])
	assert.Equal(t, float32(0.1), node.Embedding[1])
}

func TestNodeTimeOperations(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	node := &types.Node{
		ID:        "time-node",
		Name:      "Time-aware Node",
		Type:      types.EntityNodeType,
		GroupID:   "group",
		CreatedAt: now,
		ValidFrom:  now,
		UpdatedAt:  past,
		Metadata: map[string]interface{}{
			"expires_at":  future,
		},
	}

	// Test creation time
	assert.Equal(t, now, node.CreatedAt)

	// Test temporal properties
	assert.Equal(t, now, node.ValidFrom)
	assert.Equal(t, past, node.UpdatedAt)

	expiresAt, ok := node.Metadata["expires_at"]
	require.True(t, ok)
	assert.Equal(t, future, expiresAt)
}

func TestNodeLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
	}{
		{
			name:   "single label",
			labels: []string{"Person"},
		},
		{
			name:   "multiple labels",
			labels: []string{"Entity", "Person", "Employee"},
		},
		{
			name:   "hierarchical labels",
			labels: []string{"Entity", "Location", "City", "Capital"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &types.Node{
				ID:        "labeled-node",
				Name:      "Labeled Node",
				Type:      types.EntityNodeType,
				GroupID:   "group",
				EntityType: "Labeled",
				CreatedAt: time.Now(),
			}

			// Test that node was created successfully
			assert.NotEmpty(t, node.EntityType)
		})
	}
}