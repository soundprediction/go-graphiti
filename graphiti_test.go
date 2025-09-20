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
			client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, tt.config)
			require.NotNil(t, client)
		})
	}
}

func TestClient_GetNode(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil)
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

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil)
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

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil)
	ctx := context.Background()

	// Test adding empty episodes
	err := client.Add(ctx, []types.Episode{})
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
	err = client.Add(ctx, episodes)
	// With mock driver, this should work without error
	assert.NoError(t, err)
}

func TestClient_AddBulk(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil)
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
	err := client.Add(ctx, episodes)
	assert.NoError(t, err)
}

func TestClient_NodeOperations(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil)
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

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil)
	ctx := context.Background()

	now := time.Now()
	groupID := "test-group"

	// Test creating entity edge similar to Python test
	entityEdge := &types.Edge{
		ID:        "edge-1",
		Name:      "likes",
		Type:      types.EntityEdgeType,
		SourceID:  "entity-1",
		TargetID:  "entity-2", 
		GroupID:   groupID,
		CreatedAt: now,
		Metadata: map[string]interface{}{
			"fact": "test_entity_1 relates to test_entity_2",
		},
	}

	// Test upserting edge
	err := mockDriver.UpsertEdge(ctx, entityEdge)
	assert.NoError(t, err)

	// Test getting edge (should return not found)
	retrievedEdge, err := client.GetEdge(ctx, entityEdge.ID)
	assert.Error(t, err)
	assert.Equal(t, graphiti.ErrEdgeNotFound, err)
	assert.Nil(t, retrievedEdge)
}

func TestClient_Search(t *testing.T) {
	mockDriver := &MockGraphDriver{}
	mockLLM := &MockLLMClient{}
	mockEmbedder := &MockEmbedderClient{}

	client := graphiti.NewClient(mockDriver, mockLLM, mockEmbedder, nil)
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
