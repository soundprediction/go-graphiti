package search

import (
	"context"
	"testing"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// MockGraphDriver implements driver.GraphDriver for testing
type MockGraphDriver struct {
	nodes []*types.Node
	edges []*types.Edge
}

func NewMockGraphDriver() *MockGraphDriver {
	return &MockGraphDriver{
		nodes: createMockNodes(),
		edges: createMockEdges(),
	}
}

// Implement driver.GraphDriver interface methods
func (m *MockGraphDriver) GetNode(ctx context.Context, id string, groupID string) (*types.Node, error) {
	for _, node := range m.nodes {
		if node.ID == id {
			return node, nil
		}
	}
	return nil, nil
}

func (m *MockGraphDriver) GetEdge(ctx context.Context, id string, groupID string) (*types.Edge, error) {
	for _, edge := range m.edges {
		if edge.BaseEdge.ID == id {
			return edge, nil
		}
	}
	return nil, nil
}

// Stub implementations for other required methods
func (m *MockGraphDriver) UpsertNode(ctx context.Context, node *types.Node) error { return nil }
func (m *MockGraphDriver) UpsertEdge(ctx context.Context, edge *types.Edge) error { return nil }
func (m *MockGraphDriver) DeleteNode(ctx context.Context, id string, groupID string) error { return nil }
func (m *MockGraphDriver) DeleteEdge(ctx context.Context, id string, groupID string) error { return nil }
func (m *MockGraphDriver) GetNodes(ctx context.Context, nodeIDs []string, groupID string) ([]*types.Node, error) {
	return m.nodes, nil
}
func (m *MockGraphDriver) GetEdges(ctx context.Context, edgeIDs []string, groupID string) ([]*types.Edge, error) {
	return m.edges, nil
}
func (m *MockGraphDriver) GetNeighbors(ctx context.Context, nodeID string, groupID string, maxDistance int) ([]*types.Node, error) {
	return []*types.Node{}, nil
}
func (m *MockGraphDriver) GetRelatedNodes(ctx context.Context, nodeID string, groupID string, edgeTypes []types.EdgeType) ([]*types.Node, error) {
	return []*types.Node{}, nil
}
func (m *MockGraphDriver) SearchNodesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Node, error) {
	return m.nodes, nil
}
func (m *MockGraphDriver) SearchEdgesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Edge, error) {
	return m.edges, nil
}
func (m *MockGraphDriver) UpsertNodes(ctx context.Context, nodes []*types.Node) error { return nil }
func (m *MockGraphDriver) UpsertEdges(ctx context.Context, edges []*types.Edge) error { return nil }
func (m *MockGraphDriver) GetNodesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Node, error) {
	return []*types.Node{}, nil
}
func (m *MockGraphDriver) GetEdgesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Edge, error) {
	return []*types.Edge{}, nil
}
func (m *MockGraphDriver) GetCommunities(ctx context.Context, groupID string, level int) ([]*types.Node, error) {
	return []*types.Node{}, nil
}
func (m *MockGraphDriver) BuildCommunities(ctx context.Context, groupID string) error { return nil }
func (m *MockGraphDriver) CreateIndices(ctx context.Context) error                    { return nil }
func (m *MockGraphDriver) GetStats(ctx context.Context, groupID string) (*driver.GraphStats, error) {
	return &driver.GraphStats{}, nil
}
func (m *MockGraphDriver) Close() error { return nil }

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

// New search interface methods
func (m *MockGraphDriver) SearchNodes(ctx context.Context, query, groupID string, options *driver.SearchOptions) ([]*types.Node, error) {
	return m.nodes, nil
}

func (m *MockGraphDriver) SearchEdges(ctx context.Context, query, groupID string, options *driver.SearchOptions) ([]*types.Edge, error) {
	return m.edges, nil
}

func (m *MockGraphDriver) SearchNodesByVector(ctx context.Context, vector []float32, groupID string, options *driver.VectorSearchOptions) ([]*types.Node, error) {
	return m.nodes, nil
}

func (m *MockGraphDriver) SearchEdgesByVector(ctx context.Context, vector []float32, groupID string, options *driver.VectorSearchOptions) ([]*types.Edge, error) {
	return m.edges, nil
}

// Create mock data
func createMockNodes() []*types.Node {
	now := time.Now()
	return []*types.Node{
		{
			ID:        "node-1",
			Name:      "Test Node 1",
			Summary:   "This is a test node for entity search",
			Type:      types.EntityNodeType,
			GroupID:   "test-group",
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"name_embedding": []float32{0.1, 0.2, 0.3, 0.4, 0.5},
			},
		},
		{
			ID:        "node-2",
			Name:      "Test Node 2",
			Summary:   "Another test node for searching",
			Type:      types.EntityNodeType,
			GroupID:   "test-group",
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"name_embedding": []float32{0.2, 0.3, 0.4, 0.5, 0.6},
			},
		},
		{
			ID:        "episode-1",
			Name:      "Test Episode",
			Summary:   "This is an episodic node for testing",
			Type:      types.EpisodicNodeType,
			GroupID:   "test-group",
			CreatedAt: now,
			Metadata:  map[string]interface{}{},
		},
		{
			ID:        "community-1",
			Name:      "Test Community",
			Summary:   "This is a community node for testing",
			Type:      types.CommunityNodeType,
			GroupID:   "test-group",
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"name_embedding": []float32{0.3, 0.4, 0.5, 0.6, 0.7},
			},
		},
	}
}

func createMockEdges() []*types.Edge {
	now := time.Now()
	return []*types.Edge{
		{
			BaseEdge: types.BaseEdge{
				ID:           "edge-1",
				GroupID:      "test-group",
				SourceNodeID: "node-1",
				TargetNodeID: "node-2",
				CreatedAt:    now,
				Metadata: map[string]interface{}{
					"fact_embedding": []float32{0.4, 0.5, 0.6, 0.7, 0.8},
				},
			},
			Name:      "Test Relationship",
			Fact:      "This is a test relationship between nodes",
			Type:      types.EntityEdgeType,
			SourceID:  "node-1",
			TargetID:  "node-2",
			Summary:   "This is a test relationship between nodes",
			UpdatedAt: now,
			ValidFrom: now,
		},
	}
}


// Test functions

func TestCalculateCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		vector1  []float32
		vector2  []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			vector1:  []float32{1, 0, 0},
			vector2:  []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			vector1:  []float32{1, 0, 0},
			vector2:  []float32{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			vector1:  []float32{1, 0, 0},
			vector2:  []float32{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "different lengths",
			vector1:  []float32{1, 0},
			vector2:  []float32{1, 0, 0},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			vector1:  []float32{0, 0, 0},
			vector2:  []float32{1, 0, 0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCosineSimilarity(tt.vector1, tt.vector2)
			if abs(result-tt.expected) > 1e-6 {
				t.Errorf("CalculateCosineSimilarity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestFulltextQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		groupIDs []string
		expected string
	}{
		{
			name:     "simple query no groups",
			query:    "test query",
			groupIDs: nil,
			expected: "test query",
		},
		{
			name:     "empty query",
			query:    "",
			groupIDs: []string{"group1"},
			expected: "",
		},
		{
			name:     "query with groups",
			query:    "test",
			groupIDs: []string{"group1", "group2"},
			expected: `(group_id:"group1" OR group_id:"group2") AND (test)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FulltextQuery(tt.query, tt.groupIDs)
			if result != tt.expected {
				t.Errorf("FulltextQuery() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRRF(t *testing.T) {
	tests := []struct {
		name         string
		results      [][]string
		rankConstant int
		minScore     float64
		expectUUIDs  int // Just check length for simplicity
	}{
		{
			name: "basic RRF",
			results: [][]string{
				{"uuid1", "uuid2", "uuid3"},
				{"uuid2", "uuid1", "uuid4"},
			},
			rankConstant: 60,
			minScore:     0,
			expectUUIDs:  4,
		},
		{
			name: "empty results",
			results: [][]string{
				{},
				{},
			},
			rankConstant: 60,
			minScore:     0,
			expectUUIDs:  0,
		},
		{
			name: "with min score",
			results: [][]string{
				{"uuid1", "uuid2", "uuid3"},
				{"uuid2", "uuid1", "uuid4"},
			},
			rankConstant: 60,
			minScore:     0.02, // High threshold
			expectUUIDs:  2,    // Only top results should pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uuids, scores := RRF(tt.results, tt.rankConstant, tt.minScore)
			
			if len(uuids) != tt.expectUUIDs {
				t.Errorf("RRF() returned %d UUIDs, expected %d", len(uuids), tt.expectUUIDs)
			}
			
			if len(uuids) != len(scores) {
				t.Errorf("RRF() returned %d UUIDs but %d scores", len(uuids), len(scores))
			}
			
			// Check that scores are in descending order
			for i := 1; i < len(scores); i++ {
				if scores[i] > scores[i-1] {
					t.Errorf("RRF() scores not in descending order: %v", scores)
					break
				}
			}
		})
	}
}

func TestMaximalMarginalRelevance(t *testing.T) {
	queryVector := []float32{1.0, 0.0, 0.0}
	candidates := map[string][]float32{
		"uuid1": {1.0, 0.0, 0.0},   // Same as query
		"uuid2": {0.0, 1.0, 0.0},   // Orthogonal to query
		"uuid3": {0.5, 0.5, 0.0},   // Between query and uuid2
		"uuid4": {1.0, 0.1, 0.0},   // Similar to query but slightly different
	}

	uuids, scores := MaximalMarginalRelevance(queryVector, candidates, 0.5, -2.0)

	if len(uuids) != 4 {
		t.Errorf("MaximalMarginalRelevance() returned %d UUIDs, expected 4", len(uuids))
	}

	if len(uuids) != len(scores) {
		t.Errorf("MaximalMarginalRelevance() returned %d UUIDs but %d scores", len(uuids), len(scores))
	}

	// Check that scores are in descending order
	for i := 1; i < len(scores); i++ {
		if scores[i] > scores[i-1] {
			t.Errorf("MMR() scores not in descending order: %v", scores)
			break
		}
	}
}

func TestMMRIntegrationWithSearch(t *testing.T) {
	mockDriver := NewMockGraphDriver()
	searcher := NewSearcher(mockDriver, nil, nil)
	ctx := context.Background()

	t.Run("MMR Node Reranking", func(t *testing.T) {
		nodes := mockDriver.nodes
		queryVector := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

		rerankedNodes, scores, err := searcher.mmrRerankNodes(ctx, queryVector, nodes, 0.5, -2.0, 10)
		if err != nil {
			t.Errorf("mmrRerankNodes() error = %v", err)
			return
		}

		// Should return nodes (may be fewer if some don't have embeddings)
		if len(rerankedNodes) == 0 {
			t.Error("mmrRerankNodes() returned no nodes")
		}

		if len(rerankedNodes) != len(scores) {
			t.Errorf("mmrRerankNodes() returned %d nodes but %d scores", len(rerankedNodes), len(scores))
		}

		// Verify all returned nodes are from the original set
		originalNodeMap := make(map[string]bool)
		for _, node := range nodes {
			originalNodeMap[node.ID] = true
		}

		for _, node := range rerankedNodes {
			if !originalNodeMap[node.ID] {
				t.Errorf("mmrRerankNodes() returned unexpected node ID: %s", node.ID)
			}
		}
	})

	t.Run("MMR Edge Reranking", func(t *testing.T) {
		edges := mockDriver.edges
		queryVector := []float32{0.4, 0.5, 0.6, 0.7, 0.8}

		rerankedEdges, scores, err := searcher.mmrRerankEdges(ctx, queryVector, edges, 0.5, -2.0, 10)
		if err != nil {
			t.Errorf("mmrRerankEdges() error = %v", err)
			return
		}

		if len(rerankedEdges) != len(scores) {
			t.Errorf("mmrRerankEdges() returned %d edges but %d scores", len(rerankedEdges), len(scores))
		}

		// Should handle empty query vector gracefully
		rerankedEdges2, scores2, err := searcher.mmrRerankEdges(ctx, []float32{}, edges, 0.5, -2.0, 10)
		if err != nil {
			t.Errorf("mmrRerankEdges() with empty query vector error = %v", err)
			return
		}

		if len(rerankedEdges2) == 0 {
			t.Error("mmrRerankEdges() with empty query vector returned no edges")
		}

		if len(rerankedEdges2) != len(scores2) {
			t.Errorf("mmrRerankEdges() with empty query returned %d edges but %d scores", len(rerankedEdges2), len(scores2))
		}
	})

	t.Run("MMR with different lambda values", func(t *testing.T) {
		nodes := mockDriver.nodes
		queryVector := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

		// Test with lambda = 0.0 (pure diversity)
		reranked1, scores1, err := searcher.mmrRerankNodes(ctx, queryVector, nodes, 0.0, -2.0, 10)
		if err != nil {
			t.Errorf("mmrRerankNodes() lambda=0.0 error = %v", err)
			return
		}

		// Test with lambda = 1.0 (pure relevance)
		reranked2, scores2, err := searcher.mmrRerankNodes(ctx, queryVector, nodes, 1.0, -2.0, 10)
		if err != nil {
			t.Errorf("mmrRerankNodes() lambda=1.0 error = %v", err)
			return
		}

		// Both should return same number of results but potentially different rankings
		if len(reranked1) != len(reranked2) {
			t.Errorf("Different lambda values returned different result counts: %d vs %d", len(reranked1), len(reranked2))
		}

		if len(scores1) != len(scores2) {
			t.Errorf("Different lambda values returned different score counts: %d vs %d", len(scores1), len(scores2))
		}
	})
}

func TestSearchUtilities(t *testing.T) {
	mockDriver := NewMockGraphDriver()
	su := NewSearchUtilities(mockDriver)
	ctx := context.Background()

	t.Run("NodeFulltextSearch", func(t *testing.T) {
		nodes, err := su.NodeFulltextSearch(ctx, "test", nil, []string{"test-group"}, 10)
		if err != nil {
			t.Errorf("NodeFulltextSearch() error = %v", err)
			return
		}
		
		if len(nodes) == 0 {
			t.Error("NodeFulltextSearch() returned no nodes")
		}
	})

	t.Run("NodeSimilaritySearch", func(t *testing.T) {
		searchVector := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		nodes, err := su.NodeSimilaritySearch(ctx, searchVector, nil, []string{"test-group"}, 10, 0.0)
		if err != nil {
			t.Errorf("NodeSimilaritySearch() error = %v", err)
			return
		}
		
		if len(nodes) == 0 {
			t.Error("NodeSimilaritySearch() returned no nodes")
		}
	})

	t.Run("HybridNodeSearch", func(t *testing.T) {
		queries := []string{"test"}
		embeddings := [][]float32{{0.1, 0.2, 0.3, 0.4, 0.5}}
		
		nodes, err := su.HybridNodeSearch(ctx, queries, embeddings, nil, []string{"test-group"}, 10)
		if err != nil {
			t.Errorf("HybridNodeSearch() error = %v", err)
			return
		}
		
		if len(nodes) == 0 {
			t.Error("HybridNodeSearch() returned no nodes")
		}
	})

	t.Run("EdgeFulltextSearch", func(t *testing.T) {
		edges, err := su.EdgeFulltextSearch(ctx, "test", nil, []string{"test-group"}, 10)
		if err != nil {
			t.Errorf("EdgeFulltextSearch() error = %v", err)
			return
		}
		
		if len(edges) == 0 {
			t.Error("EdgeFulltextSearch() returned no edges")
		}
	})
}

func TestSpecializedSearch(t *testing.T) {
	mockDriver := NewMockGraphDriver()
	su := NewSearchUtilities(mockDriver)
	ctx := context.Background()

	t.Run("EpisodeFulltextSearch", func(t *testing.T) {
		options := &EpisodeSearchOptions{
			Limit:    10,
			GroupIDs: []string{"test-group"},
		}
		
		episodes, err := su.EpisodeFulltextSearch(ctx, "test", options)
		if err != nil {
			t.Errorf("EpisodeFulltextSearch() error = %v", err)
		}
		
		// Should return mock nodes even if they're not actual episodes
		if len(episodes) == 0 {
			t.Error("EpisodeFulltextSearch() returned no episodes")
		}
	})

	t.Run("CommunityFulltextSearch", func(t *testing.T) {
		options := &CommunitySearchOptions{
			Limit:    10,
			GroupIDs: []string{"test-group"},
		}
		
		communities, err := su.CommunityFulltextSearch(ctx, "test", options)
		if err != nil {
			t.Errorf("CommunityFulltextSearch() error = %v", err)
		}
		
		// Should return mock nodes even if they're not actual communities
		if len(communities) == 0 {
			t.Error("CommunityFulltextSearch() returned no communities")
		}
	})

	t.Run("MultiModalSearch", func(t *testing.T) {
		searchVector := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		
		result, err := su.MultiModalSearch(ctx, "test", searchVector, []string{"test-group"}, 10)
		if err != nil {
			t.Errorf("MultiModalSearch() error = %v", err)
			return
		}
		
		if result.TotalResults == 0 {
			t.Error("MultiModalSearch() returned no results")
		}
		
		if len(result.NodeScores) != len(result.Nodes) {
			t.Errorf("MultiModalSearch() node scores length mismatch: %d scores for %d nodes", 
				len(result.NodeScores), len(result.Nodes))
		}
		
		if len(result.EdgeScores) != len(result.Edges) {
			t.Errorf("MultiModalSearch() edge scores length mismatch: %d scores for %d edges", 
				len(result.EdgeScores), len(result.Edges))
		}
	})
}

func TestRerankers(t *testing.T) {
	mockDriver := NewMockGraphDriver()
	ctx := context.Background()

	t.Run("NodeDistanceReranker", func(t *testing.T) {
		nodeUUIDs := []string{"node-1", "node-2", "node-3"}
		centerUUID := "node-1"
		
		uuids, scores, err := NodeDistanceReranker(ctx, mockDriver, nodeUUIDs, centerUUID, 0.0)
		if err != nil {
			t.Errorf("NodeDistanceReranker() error = %v", err)
			return
		}
		
		if len(uuids) == 0 {
			t.Error("NodeDistanceReranker() returned no results")
		}
		
		if len(uuids) != len(scores) {
			t.Errorf("NodeDistanceReranker() returned %d UUIDs but %d scores", len(uuids), len(scores))
		}
		
		// Center node should be first if present
		if uuids[0] != centerUUID {
			t.Errorf("NodeDistanceReranker() center node should be first, got %v", uuids[0])
		}
	})

	t.Run("EpisodeMentionsReranker", func(t *testing.T) {
		nodeUUIDs := [][]string{
			{"node-1", "node-2"},
			{"node-2", "node-3"},
		}
		
		uuids, scores, err := EpisodeMentionsReranker(ctx, mockDriver, nodeUUIDs, 0.0)
		if err != nil {
			t.Errorf("EpisodeMentionsReranker() error = %v", err)
			return
		}
		
		if len(uuids) != len(scores) {
			t.Errorf("EpisodeMentionsReranker() returned %d UUIDs but %d scores", len(uuids), len(scores))
		}
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("normalizeL2", func(t *testing.T) {
		vector := []float32{3.0, 4.0}
		normalized := normalizeL2(vector)
		
		// Check that the normalized vector has unit length
		var sumSquares float32
		for _, val := range normalized {
			sumSquares += val * val
		}
		
		if abs(float64(sumSquares)-1.0) > 1e-6 {
			t.Errorf("normalizeL2() result not unit vector: sum of squares = %f", sumSquares)
		}
	})

	t.Run("toFloat32Slice", func(t *testing.T) {
		tests := []struct {
			name     string
			input    interface{}
			expected []float32
		}{
			{
				name:     "float32 slice",
				input:    []float32{1.0, 2.0, 3.0},
				expected: []float32{1.0, 2.0, 3.0},
			},
			{
				name:     "float64 slice",
				input:    []float64{1.0, 2.0, 3.0},
				expected: []float32{1.0, 2.0, 3.0},
			},
			{
				name:     "string slice with numbers",
				input:    []interface{}{"1.5", "2.5", "3.5"},
				expected: []float32{1.5, 2.5, 3.5},
			},
			{
				name:     "nil input",
				input:    nil,
				expected: nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := toFloat32Slice(tt.input)
				
				if tt.expected == nil && result != nil {
					t.Errorf("toFloat32Slice() = %v, want nil", result)
					return
				}
				
				if tt.expected != nil && result == nil {
					t.Errorf("toFloat32Slice() = nil, want %v", tt.expected)
					return
				}
				
				if len(result) != len(tt.expected) {
					t.Errorf("toFloat32Slice() length = %d, want %d", len(result), len(tt.expected))
					return
				}
				
				for i := range result {
					if abs(float64(result[i]-tt.expected[i])) > 1e-6 {
						t.Errorf("toFloat32Slice()[%d] = %f, want %f", i, result[i], tt.expected[i])
					}
				}
			})
		}
	})
}