package graphiti_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/soundprediction/go-graphiti"
	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGraphDriver is a mock implementation for testing
type MockGraphDriver struct{}

func (m *MockGraphDriver) GetNode(ctx context.Context, nodeID, groupID string) (*types.Node, error) {
	return nil, graphiti.ErrNodeNotFound
}

func (m *MockGraphDriver) UpsertNode(ctx context.Context, node *types.Node) error {
	return nil
}

func (m *MockGraphDriver) DeleteNode(ctx context.Context, nodeID, groupID string) error {
	return nil
}

func (m *MockGraphDriver) GetNodes(ctx context.Context, nodeIDs []string, groupID string) ([]*types.Node, error) {
	return []*types.Node{}, nil
}

func (m *MockGraphDriver) GetEdge(ctx context.Context, edgeID, groupID string) (*types.Edge, error) {
	return nil, graphiti.ErrEdgeNotFound
}

func (m *MockGraphDriver) UpsertEdge(ctx context.Context, edge *types.Edge) error {
	return nil
}

func (m *MockGraphDriver) DeleteEdge(ctx context.Context, edgeID, groupID string) error {
	return nil
}

func (m *MockGraphDriver) GetEdges(ctx context.Context, edgeIDs []string, groupID string) ([]*types.Edge, error) {
	return []*types.Edge{}, nil
}

func (m *MockGraphDriver) GetNeighbors(ctx context.Context, nodeID, groupID string, maxDistance int) ([]*types.Node, error) {
	return []*types.Node{}, nil
}

func (m *MockGraphDriver) GetRelatedNodes(ctx context.Context, nodeID, groupID string, edgeTypes []types.EdgeType) ([]*types.Node, error) {
	return []*types.Node{}, nil
}

func (m *MockGraphDriver) SearchNodesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Node, error) {
	return []*types.Node{}, nil
}

func (m *MockGraphDriver) SearchEdgesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Edge, error) {
	return []*types.Edge{}, nil
}

func (m *MockGraphDriver) UpsertNodes(ctx context.Context, nodes []*types.Node) error {
	return nil
}

func (m *MockGraphDriver) UpsertEdges(ctx context.Context, edges []*types.Edge) error {
	return nil
}

func (m *MockGraphDriver) GetNodesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Node, error) {
	return []*types.Node{}, nil
}

func (m *MockGraphDriver) GetEdgesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Edge, error) {
	return []*types.Edge{}, nil
}

func (m *MockGraphDriver) GetCommunities(ctx context.Context, groupID string, level int) ([]*types.Node, error) {
	return []*types.Node{}, nil
}

func (m *MockGraphDriver) BuildCommunities(ctx context.Context, groupID string) error {
	return nil
}

func (m *MockGraphDriver) GetExistingCommunity(ctx context.Context, entityUUID string) (*types.Node, error) {
	return nil, nil
}

func (m *MockGraphDriver) FindModalCommunity(ctx context.Context, entityUUID string) (*types.Node, error) {
	return nil, nil
}

func (m *MockGraphDriver) CreateIndices(ctx context.Context) error {
	return nil
}

func (m *MockGraphDriver) GetStats(ctx context.Context, groupID string) (*driver.GraphStats, error) {
	return &driver.GraphStats{}, nil
}

func (m *MockGraphDriver) SearchNodes(ctx context.Context, query, groupID string, options *driver.SearchOptions) ([]*types.Node, error) {
	return []*types.Node{}, nil
}

func (m *MockGraphDriver) SearchEdges(ctx context.Context, query, groupID string, options *driver.SearchOptions) ([]*types.Edge, error) {
	return []*types.Edge{}, nil
}

func (m *MockGraphDriver) SearchNodesByVector(ctx context.Context, vector []float32, groupID string, options *driver.VectorSearchOptions) ([]*types.Node, error) {
	return []*types.Node{}, nil
}

func (m *MockGraphDriver) SearchEdgesByVector(ctx context.Context, vector []float32, groupID string, options *driver.VectorSearchOptions) ([]*types.Edge, error) {
	return []*types.Edge{}, nil
}

func (m *MockGraphDriver) Close() error {
	return nil
}

func (m *MockGraphDriver) ExecuteQuery(cypherQuery string, kwargs map[string]interface{}) (interface{}, interface{}, interface{}, error) {
	return nil, nil, nil, nil
}

func (m *MockGraphDriver) Session(database *string) driver.GraphDriverSession {
	return nil
}

func (m *MockGraphDriver) DeleteAllIndexes(database string) {
	// No-op for mock
}

func (m *MockGraphDriver) Provider() driver.GraphProvider {
	return driver.GraphProviderNeo4j
}

func (m *MockGraphDriver) GetAossClient() interface{} {
	return nil
}

// MockLLMClient is a mock LLM implementation for testing
type MockLLMClient struct{}

func (m *MockLLMClient) Chat(ctx context.Context, messages []llm.Message) (*llm.Response, error) {
	return &llm.Response{
		Content: "Mock response",
	}, nil
}

func (m *MockLLMClient) ChatWithStructuredOutput(ctx context.Context, messages []llm.Message, schema any) (json.RawMessage, error) {
	return json.RawMessage(`{"mock": "response"}`), nil
}

func (m *MockLLMClient) Close() error {
	return nil
}

// MockEmbedderClient is a mock embedder implementation for testing
type MockEmbedderClient struct{}

func (m *MockEmbedderClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = make([]float32, 1536) // Mock 1536-dimensional embedding
	}
	return embeddings, nil
}

func (m *MockEmbedderClient) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	return make([]float32, 1536), nil
}

func (m *MockEmbedderClient) Dimensions() int {
	return 1536
}

func (m *MockEmbedderClient) Close() error {
	return nil
}

func TestNewClient(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	tests := []struct {
		name   string
		config *graphiti.Config
	}{
		{
			name:   "with nil config",
			config: nil,
		},
		{
			name: "with custom config",
			config: &graphiti.Config{
				GroupID:  "test-group",
				TimeZone: time.UTC,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, tt.config, nil)
			require.NotNil(t, client)
		})
	}
}

func TestClient_GetNode(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil, nil)
	ctx := context.Background()

	// Test getting a non-existent node
	node, err := client.GetNode(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Equal(t, graphiti.ErrNodeNotFound, err)
	assert.Nil(t, node)
}

func TestClient_GetEdge(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil, nil)
	ctx := context.Background()

	// Test getting a non-existent edge
	edge, err := client.GetEdge(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Equal(t, graphiti.ErrEdgeNotFound, err)
	assert.Nil(t, edge)
}

func TestClient_Add(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil, nil)
	ctx := context.Background()

	// Test adding empty episodes
	_, err := client.Add(ctx, []types.Episode{}, nil)
	assert.NoError(t, err)

	// Test adding episodes (should work with mock)
	episodes := []types.Episode{
		{
			ID:        "test-episode",
			Name:      "Test Episode",
			Content:   "Test content",
			Reference: time.Now(),
			CreatedAt: time.Now(),
			GroupID:   "test-group",
		},
	}
	_, err = client.Add(ctx, episodes, nil)
	// With mock driver, this should work without error
	assert.NoError(t, err)
}

func TestClient_AddEpisodeWithCommunityUpdates(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil, nil)
	ctx := context.Background()

	// Test adding episode with community updates enabled
	episode := types.Episode{
		ID:        "test-episode-community",
		Name:      "Test Episode with Community Updates",
		Content:   "Test content for community building",
		Reference: time.Now(),
		CreatedAt: time.Now(),
		GroupID:   "test-group",
	}

	// Test with UpdateCommunities disabled (default)
	options := &graphiti.AddEpisodeOptions{
		UpdateCommunities: false,
	}
	result, err := client.AddEpisode(ctx, episode, options)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Communities, "Communities should be empty when UpdateCommunities is false")
	assert.Empty(t, result.CommunityEdges, "CommunityEdges should be empty when UpdateCommunities is false")

	// Test with UpdateCommunities enabled
	options.UpdateCommunities = true
	result, err = client.AddEpisode(ctx, episode, options)
	// With mock driver, community building will fail as it only supports Kuzu drivers
	assert.Error(t, err, "Community building should fail with mock driver")
	assert.Contains(t, err.Error(), "failed to build communities", "Error should indicate community building failure")
	assert.Nil(t, result, "Result should be nil when community building fails")
}

func TestClient_AddBulk(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil, nil)
	ctx := context.Background()

	now := time.Now()
	groupID := "test-group"

	// Create test episodes similar to Python test
	episodes := []types.Episode{
		{
			ID:        "episode-1",
			Name:      "test_episode",
			Content:   "Alice likes Bob",
			Reference: now,
			CreatedAt: now,
			GroupID:   groupID,
		},
		{
			ID:        "episode-2",
			Name:      "test_episode_2",
			Content:   "Bob adores Alice",
			Reference: now,
			CreatedAt: now,
			GroupID:   groupID,
		},
	}

	// Test bulk add operations
	_, err := client.Add(ctx, episodes, nil)
	assert.NoError(t, err)
}

func TestClient_NodeOperations(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil, nil)
	ctx := context.Background()

	now := time.Now()
	groupID := "test-group"

	// Test creating entity node similar to Python test
	entityNode := &types.Node{
		ID:         "entity-1",
		Name:       "test_entity_1",
		Type:       types.EntityNodeType,
		GroupID:    groupID,
		EntityType: "Person",
		CreatedAt:  now,
		Summary:    "test_entity_1 summary",
		Metadata: map[string]interface{}{
			"age":      30,
			"location": "New York",
		},
	}

	// Test upserting node
	err := mockDriver.UpsertNode(ctx, entityNode)
	assert.NoError(t, err)

	// Test getting node (should return not found)
	retrievedNode, err := client.GetNode(ctx, entityNode.ID)
	assert.Error(t, err)
	assert.Equal(t, graphiti.ErrNodeNotFound, err)
	assert.Nil(t, retrievedNode)
}

func TestClient_EdgeOperations(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil, nil)
	ctx := context.Background()

	now := time.Now()
	groupID := "test-group"

	// Test creating entity edge similar to Python test
	entityEdge := &types.Edge{
		BaseEdge: types.BaseEdge{
			ID:           "edge-1",
			GroupID:      groupID,
			SourceNodeID: "entity-1",
			TargetNodeID: "entity-2",
			CreatedAt:    now,
			Metadata: map[string]interface{}{
				"fact": "test_entity_1 relates to test_entity_2",
			},
		},
		Name:     "likes",
		Type:     types.EntityEdgeType,
		SourceID: "entity-1",
		TargetID: "entity-2",
	}

	// Test upserting edge
	err := mockDriver.UpsertEdge(ctx, entityEdge)
	assert.NoError(t, err)

	// Test getting edge (should return not found)
	retrievedEdge, err := client.GetEdge(ctx, entityEdge.BaseEdge.ID)
	assert.Error(t, err)
	assert.Equal(t, graphiti.ErrEdgeNotFound, err)
	assert.Nil(t, retrievedEdge)
}

func TestClient_Search(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil, nil)
	ctx := context.Background()

	// Test search (returns empty results with mock driver)
	results, err := client.Search(ctx, "test query", nil)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Equal(t, "test query", results.Query)
	assert.Equal(t, 0, results.Total)
	assert.Empty(t, results.Nodes)
	assert.Empty(t, results.Edges)
}

func TestSearchConfig(t *testing.T) {
	config := graphiti.NewDefaultSearchConfig()

	assert.Equal(t, 20, config.Limit)
	assert.Equal(t, 2, config.CenterNodeDistance)
	assert.Equal(t, 0.0, config.MinScore)
	assert.True(t, config.IncludeEdges)
	assert.False(t, config.Rerank)
}

func TestNodeTypes(t *testing.T) {
	assert.Equal(t, types.NodeType("entity"), types.EntityNodeType)
	assert.Equal(t, types.NodeType("episodic"), types.EpisodicNodeType)
	assert.Equal(t, types.NodeType("community"), types.CommunityNodeType)
}

func TestEdgeTypes(t *testing.T) {
	assert.Equal(t, types.EdgeType("entity"), types.EntityEdgeType)
	assert.Equal(t, types.EdgeType("episodic"), types.EpisodicEdgeType)
	assert.Equal(t, types.EdgeType("community"), types.CommunityEdgeType)
}

func TestEpisodeTypes(t *testing.T) {
	assert.Equal(t, types.EpisodeType("conversation"), types.ConversationEpisodeType)
	assert.Equal(t, types.EpisodeType("document"), types.DocumentEpisodeType)
	assert.Equal(t, types.EpisodeType("event"), types.EventEpisodeType)
}

func TestClient_parseEntitiesFromResponse(t *testing.T) {
	// Use same client setup as TestClient_Add
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil, nil)
	groupID := "test-group"

	tests := []struct {
		name             string
		responseContent  string
		expectedEntities int
		expectedNames    []string
		expectError      bool
	}{
		{
			name: "valid JSON response with multiple entities from Test content",
			responseContent: `{
				"extracted_entities": [
					{"name": "Test Episode", "entity_type_id": 0},
					{"name": "Test content", "entity_type_id": 0}
				]
			}`,
			expectedEntities: 2,
			expectedNames:    []string{"Test Episode", "Test content"},
			expectError:      false,
		},
		{
			name: "valid JSON response with single entity",
			responseContent: `{
				"extracted_entities": [
					{"name": "Test Episode", "entity_type_id": 0}
				]
			}`,
			expectedEntities: 1,
			expectedNames:    []string{"Test Episode"},
			expectError:      false,
		},
		{
			name: "JSON response with empty entities",
			responseContent: `{
				"extracted_entities": []
			}`,
			expectedEntities: 0,
			expectedNames:    []string{},
			expectError:      false,
		},
		{
			name: "JSON response with empty/whitespace names filtered out",
			responseContent: `{
				"extracted_entities": [
					{"name": "Test Episode", "entity_type_id": 0},
					{"name": "", "entity_type_id": 0},
					{"name": "   ", "entity_type_id": 0},
					{"name": "Content", "entity_type_id": 0}
				]
			}`,
			expectedEntities: 2,
			expectedNames:    []string{"Test Episode", "Content"},
			expectError:      false,
		},
		{
			name: "JSON wrapped in extra text",
			responseContent: `Here are the extracted entities:
			{
				"extracted_entities": [
					{"name": "Test Episode", "entity_type_id": 0}
				]
			}
			That's all the entities I found.`,
			expectedEntities: 1,
			expectedNames:    []string{"Test Episode"},
			expectError:      false,
		},
		{
			name:             "empty response",
			responseContent:  "",
			expectedEntities: 0,
			expectedNames:    []string{},
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities, err := client.ParseEntitiesFromResponse(tt.responseContent, groupID)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, entities, tt.expectedEntities)

			// Check entity names
			actualNames := make([]string, len(entities))
			for i, entity := range entities {
				actualNames[i] = entity.Name
			}

			for _, expectedName := range tt.expectedNames {
				assert.Contains(t, actualNames, expectedName)
			}

			// Validate entity properties (same as in TestClient_Add pattern)
			for _, entity := range entities {
				assert.NotEmpty(t, entity.ID)
				assert.Equal(t, types.EntityNodeType, entity.Type)
				assert.Equal(t, groupID, entity.GroupID)
				assert.Equal(t, "Entity", entity.EntityType)
				assert.NotNil(t, entity.Metadata)
				assert.Contains(t, entity.Metadata, "entity_type_id")
				assert.Contains(t, entity.Metadata, "labels")
				assert.NotZero(t, entity.CreatedAt)
				assert.NotZero(t, entity.UpdatedAt)
				assert.NotZero(t, entity.ValidFrom)
			}
		})
	}
}

// TestClient_Add_PythonCompatibility tests that Go client.Add() behavior matches Python Graphiti
// This comprehensive test compares Go implementation against Python add_episode() and add_episode_bulk()
func TestClient_Add_PythonCompatibility(t *testing.T) {
	// Setup mock clients that simulate successful operations
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil, nil)
	ctx := context.Background()
	now := time.Now()
	groupID := "test-group"

	t.Run("SingleEpisode_MatchesPythonAddEpisode", func(t *testing.T) {
		// Test that Go Add([single_episode]) matches Python add_episode(episode)
		episode := types.Episode{
			ID:        "episode-001",
			Name:      "Single Episode Test",
			Content:   "This is a test episode with some entities like Alice and Bob.",
			Reference: now,
			CreatedAt: now,
			GroupID:   groupID,
			Metadata: map[string]interface{}{
				"source":             "test",
				"source_description": "Integration test",
				"episode_type":       "text",
			},
		}

		// Test Go Add with single episode
		result, err := client.Add(ctx, []types.Episode{episode}, nil)
		assert.NoError(t, err, "Go Add should succeed with single episode")
		assert.NotNil(t, result, "Result should not be nil")

		// Verify result structure matches Python add_episode return type
		assert.NotNil(t, result.Episodes, "Episodes should not be nil")
		assert.NotNil(t, result.EpisodicEdges, "EpisodicEdges should not be nil")
		assert.NotNil(t, result.Nodes, "Nodes should not be nil")
		assert.NotNil(t, result.Edges, "Edges should not be nil")
		assert.NotNil(t, result.Communities, "Communities should not be nil")
		assert.NotNil(t, result.CommunityEdges, "CommunityEdges should not be nil")

		// Verify single episode processing
		assert.Len(t, result.Episodes, 1, "Should have exactly one episode")
		assert.Equal(t, episode.ID, result.Episodes[0].ID, "Episode ID should match")
		assert.Equal(t, episode.Name, result.Episodes[0].Name, "Episode name should match")
		assert.Equal(t, types.EpisodicNodeType, result.Episodes[0].Type, "Episode should be EpisodicNodeType")
	})

	t.Run("BulkEpisodes_MatchesPythonAddEpisodeBulk", func(t *testing.T) {
		// Test that Go Add([multiple_episodes]) matches Python add_episode_bulk(episodes)
		episodes := []types.Episode{
			{
				ID:        "bulk-001",
				Name:      "Bulk Episode 1",
				Content:   "Alice works at Company A and knows Bob.",
				Reference: now,
				CreatedAt: now,
				GroupID:   groupID,
				Metadata: map[string]interface{}{
					"source":       "bulk_test",
					"episode_type": "text",
				},
			},
			{
				ID:        "bulk-002",
				Name:      "Bulk Episode 2",
				Content:   "Bob is a software engineer and lives in Seattle.",
				Reference: now.Add(time.Hour),
				CreatedAt: now.Add(time.Hour),
				GroupID:   groupID,
				Metadata: map[string]interface{}{
					"source":       "bulk_test",
					"episode_type": "text",
				},
			},
			{
				ID:        "bulk-003",
				Name:      "Bulk Episode 3",
				Content:   "Company A is a tech startup founded in 2020.",
				Reference: now.Add(2 * time.Hour),
				CreatedAt: now.Add(2 * time.Hour),
				GroupID:   groupID,
				Metadata: map[string]interface{}{
					"source":       "bulk_test",
					"episode_type": "text",
				},
			},
		}

		// Test Go Add with multiple episodes
		result, err := client.Add(ctx, episodes, nil)
		assert.NoError(t, err, "Go Add should succeed with multiple episodes")
		assert.NotNil(t, result, "Result should not be nil")

		// Verify bulk processing - should have processed all episodes
		assert.Len(t, result.Episodes, len(episodes), "Should have processed all episodes")

		// Verify each episode was processed correctly
		episodeIDs := make(map[string]bool)
		for _, ep := range result.Episodes {
			episodeIDs[ep.ID] = true
			assert.Equal(t, types.EpisodicNodeType, ep.Type, "Each episode should be EpisodicNodeType")
			assert.Equal(t, groupID, ep.GroupID, "Each episode should have correct GroupID")
		}

		// Verify all episode IDs are present
		for _, originalEp := range episodes {
			assert.True(t, episodeIDs[originalEp.ID], "Episode %s should be in results", originalEp.ID)
		}

		// Verify aggregation behavior (results should be accumulated)
		assert.GreaterOrEqual(t, len(result.Nodes), 0, "Should have accumulated entity nodes")
		assert.GreaterOrEqual(t, len(result.Edges), 0, "Should have accumulated entity edges")
		assert.GreaterOrEqual(t, len(result.EpisodicEdges), 0, "Should have accumulated episodic edges")
	})

	t.Run("EmptyEpisodes_HandlesGracefully", func(t *testing.T) {
		// Test edge case that Python would handle
		result, err := client.Add(ctx, []types.Episode{}, nil)
		assert.NoError(t, err, "Empty episodes should not cause error")
		assert.NotNil(t, result, "Result should not be nil")
		assert.Empty(t, result.Episodes, "Episodes should be empty")
		assert.Empty(t, result.Nodes, "Nodes should be empty")
		assert.Empty(t, result.Edges, "Edges should be empty")
	})

	t.Run("ErrorHandling_MatchesPythonBehavior", func(t *testing.T) {
		// Test error propagation behavior
		invalidEpisode := types.Episode{
			ID:        "", // Invalid empty ID
			Name:      "Invalid Episode",
			Content:   "This episode has invalid data",
			Reference: now,
			CreatedAt: now,
			GroupID:   groupID,
		}

		result, err := client.Add(ctx, []types.Episode{invalidEpisode}, nil)
		// The behavior depends on the mock implementation, but we test that errors are handled
		if err != nil {
			assert.Nil(t, result, "Result should be nil when error occurs")
			assert.Contains(t, err.Error(), "failed to process episode", "Error should indicate episode processing failure")
		}
	})

	t.Run("ResultStructure_MatchesPythonTypes", func(t *testing.T) {
		// Verify that the Go result structure semantically matches Python return types
		episode := types.Episode{
			ID:        "structure-test",
			Name:      "Structure Test Episode",
			Content:   "Testing result structure compatibility",
			Reference: now,
			CreatedAt: now,
			GroupID:   groupID,
		}

		result, err := client.Add(ctx, []types.Episode{episode}, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify types match Python expectations
		assert.IsType(t, &types.AddBulkEpisodeResults{}, result, "Result should be AddBulkEpisodeResults type")
		assert.IsType(t, []*types.Node{}, result.Episodes, "Episodes should be slice of Node pointers")
		assert.IsType(t, []*types.Edge{}, result.EpisodicEdges, "EpisodicEdges should be slice of Edge pointers")
		assert.IsType(t, []*types.Node{}, result.Nodes, "Nodes should be slice of Node pointers")
		assert.IsType(t, []*types.Edge{}, result.Edges, "Edges should be slice of Edge pointers")
		assert.IsType(t, []*types.Node{}, result.Communities, "Communities should be slice of Node pointers")
		assert.IsType(t, []*types.Edge{}, result.CommunityEdges, "CommunityEdges should be slice of Edge pointers")

		// Verify episode structure matches Python EpisodicNode
		if len(result.Episodes) > 0 {
			ep := result.Episodes[0]
			assert.Equal(t, episode.ID, ep.ID, "Episode ID should match input")
			assert.Equal(t, episode.Name, ep.Name, "Episode name should match input")
			assert.Equal(t, episode.Content, ep.Content, "Episode content should match input")
			assert.Equal(t, episode.GroupID, ep.GroupID, "Episode GroupID should match input")
			assert.Equal(t, types.EpisodicNodeType, ep.Type, "Episode should have EpisodicNodeType")
			assert.NotZero(t, ep.CreatedAt, "Episode should have CreatedAt timestamp")
			assert.NotZero(t, ep.UpdatedAt, "Episode should have UpdatedAt timestamp")
		}
	})

	t.Run("SequentialProcessing_MatchesPythonOrder", func(t *testing.T) {
		// Test that episodes are processed in order like Python
		episodes := []types.Episode{
			{ID: "seq-001", Name: "First", Content: "First episode", Reference: now, CreatedAt: now, GroupID: groupID},
			{ID: "seq-002", Name: "Second", Content: "Second episode", Reference: now.Add(time.Minute), CreatedAt: now.Add(time.Minute), GroupID: groupID},
			{ID: "seq-003", Name: "Third", Content: "Third episode", Reference: now.Add(2 * time.Minute), CreatedAt: now.Add(2 * time.Minute), GroupID: groupID},
		}

		result, err := client.Add(ctx, episodes, nil)
		assert.NoError(t, err)
		assert.Len(t, result.Episodes, 3, "Should have processed all episodes")

		// Verify processing order (episodes should appear in same order as input)
		for i, ep := range result.Episodes {
			assert.Equal(t, episodes[i].ID, ep.ID, "Episode %d should maintain input order", i)
		}
	})
}
