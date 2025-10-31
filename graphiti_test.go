package graphiti_test

import (
	"context"
	"encoding/json"
	"time"

	"github.com/soundprediction/go-graphiti"
	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/types"
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

func (m *MockLLMClient) Chat(ctx context.Context, messages []types.Message) (*llm.Response, error) {
	return &llm.Response{
		Content: "Mock response",
	}, nil
}

func (m *MockLLMClient) ChatWithStructuredOutput(ctx context.Context, messages []types.Message, schema any) (json.RawMessage, error) {
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
