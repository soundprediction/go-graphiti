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
				Type:      types.NodeTypeEntity,
				GroupID:   groupID,
				Labels:    []string{"Entity", "Person"},
				CreatedAt: now,
				Properties: map[string]interface{}{
					"summary":  "test_entity_1 summary",
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
				Type:      types.NodeTypeEntity,
				GroupID:   groupID,
				Labels:    []string{"Entity", "Person2"},
				CreatedAt: now,
				Properties: map[string]interface{}{
					"summary":     "test_entity_2 summary",
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
				Type:      types.NodeTypeEntity,
				GroupID:   groupID,
				Labels:    []string{"Entity", "City", "Location"},
				CreatedAt: now,
				Properties: map[string]interface{}{
					"summary":     "test_entity_3 summary",
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
			assert.Equal(t, types.NodeTypeEntity, tt.node.Type)
			assert.NotZero(t, tt.node.CreatedAt)
			assert.Contains(t, tt.node.Labels, "Entity")
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
				Type:      types.NodeTypeEpisodic,
				GroupID:   groupID,
				Labels:    []string{"Episode", "Message"},
				CreatedAt: now,
				Properties: map[string]interface{}{
					"content":            "Alice likes Bob",
					"source":             "message",
					"source_description": "conversation message",
					"valid_at":          now,
					"episode_type":      "message",
				},
			},
		},
		{
			name: "document episode",
			node: &types.Node{
				ID:        "episode-2",
				Name:      "test_episode_2",
				Type:      types.NodeTypeEpisodic,
				GroupID:   groupID,
				Labels:    []string{"Episode", "Document"},
				CreatedAt: now,
				Properties: map[string]interface{}{
					"content":            "Bob adores Alice",
					"source":             "document",
					"source_description": "meeting notes",
					"valid_at":          now,
					"episode_type":      "document",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.node.ID)
			assert.NotEmpty(t, tt.node.Name)
			assert.Equal(t, groupID, tt.node.GroupID)
			assert.Equal(t, types.NodeTypeEpisodic, tt.node.Type)
			assert.NotZero(t, tt.node.CreatedAt)
			assert.Contains(t, tt.node.Labels, "Episode")

			// Test episode-specific properties
			content, ok := tt.node.Properties["content"]
			require.True(t, ok)
			assert.NotEmpty(t, content)

			source, ok := tt.node.Properties["source"]
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
		Type:      types.NodeTypeCommunity,
		GroupID:   groupID,
		Labels:    []string{"Community"},
		CreatedAt: now,
		Properties: map[string]interface{}{
			"summary":     "A community of related entities",
			"level":       0,
			"size":        5,
			"entities":    []string{"entity-1", "entity-2", "entity-3"},
			"description": "Community formed around shared interests",
		},
	}

	assert.Equal(t, types.NodeTypeCommunity, communityNode.Type)
	assert.Equal(t, "test_community", communityNode.Name)
	assert.Equal(t, groupID, communityNode.GroupID)
	assert.NotZero(t, communityNode.CreatedAt)
	assert.Contains(t, communityNode.Labels, "Community")

	// Test community-specific properties
	level, ok := communityNode.Properties["level"]
	require.True(t, ok)
	assert.Equal(t, 0, level)

	size, ok := communityNode.Properties["size"]
	require.True(t, ok)
	assert.Equal(t, 5, size)

	entities, ok := communityNode.Properties["entities"]
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
				Type:      types.NodeTypeEntity,
				GroupID:   "group",
				CreatedAt: time.Now(),
			},
			isValid: true,
		},
		{
			name: "missing ID",
			node: &types.Node{
				Name:      "Node without ID",
				Type:      types.NodeTypeEntity,
				GroupID:   "group",
				CreatedAt: time.Now(),
			},
			isValid: false,
		},
		{
			name: "missing name",
			node: &types.Node{
				ID:        "node-id",
				Type:      types.NodeTypeEntity,
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
				Type:      types.NodeTypeEntity,
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
		types.NodeTypeEntity,
		types.NodeTypeEpisodic,
		types.NodeTypeCommunity,
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
		Type:      types.NodeTypeEntity,
		GroupID:   "group",
		CreatedAt: now,
		Properties: map[string]interface{}{
			"summary":        "A node with an embedding",
			"name_embedding": embedding,
		},
	}

	// Test embedding property
	embeddingProp, ok := node.Properties["name_embedding"]
	require.True(t, ok)
	
	embeddingSlice, ok := embeddingProp.([]float32)
	require.True(t, ok)
	assert.Len(t, embeddingSlice, 1536)
	assert.Equal(t, float32(0.0), embeddingSlice[0])
	assert.Equal(t, float32(0.1), embeddingSlice[1])
}

func TestNodeTimeOperations(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	node := &types.Node{
		ID:        "time-node",
		Name:      "Time-aware Node",
		Type:      types.NodeTypeEntity,
		GroupID:   "group",
		CreatedAt: now,
		Properties: map[string]interface{}{
			"valid_at":    now,
			"updated_at":  past,
			"expires_at":  future,
		},
	}

	// Test creation time
	assert.Equal(t, now, node.CreatedAt)

	// Test temporal properties
	validAt, ok := node.Properties["valid_at"]
	require.True(t, ok)
	assert.Equal(t, now, validAt)

	updatedAt, ok := node.Properties["updated_at"]
	require.True(t, ok)
	assert.Equal(t, past, updatedAt)

	expiresAt, ok := node.Properties["expires_at"]
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
				Type:      types.NodeTypeEntity,
				GroupID:   "group",
				Labels:    tt.labels,
				CreatedAt: time.Now(),
			}

			assert.Equal(t, tt.labels, node.Labels)
			for _, label := range tt.labels {
				assert.Contains(t, node.Labels, label)
			}
		})
	}
}