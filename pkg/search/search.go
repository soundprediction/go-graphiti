package search

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

type SearchMethod string

const (
	CosineSimilarity  SearchMethod = "cosine_similarity"
	BM25             SearchMethod = "bm25"
	BreadthFirstSearch SearchMethod = "bfs"
)

type RerankerType string

const (
	RRFRerankType            RerankerType = "rrf"
	MMRRerankType            RerankerType = "mmr" 
	CrossEncoderRerankType   RerankerType = "cross_encoder"
	NodeDistanceRerankType   RerankerType = "node_distance"
	EpisodeMentionsRerankType RerankerType = "episode_mentions"
)

type SearchConfig struct {
	NodeConfig      *NodeSearchConfig      `json:"node_config,omitempty"`
	EdgeConfig      *EdgeSearchConfig      `json:"edge_config,omitempty"`
	EpisodeConfig   *EpisodeSearchConfig   `json:"episode_config,omitempty"`
	CommunityConfig *CommunitySearchConfig `json:"community_config,omitempty"`
	Limit           int                    `json:"limit"`
	MinScore        float64                `json:"min_score"`
}

type NodeSearchConfig struct {
	SearchMethods []SearchMethod `json:"search_methods"`
	Reranker      RerankerType   `json:"reranker"`
	MinScore      float64        `json:"min_score"`
	MMRLambda     float64        `json:"mmr_lambda"`
	MaxDepth      int            `json:"max_depth"`
}

type EdgeSearchConfig struct {
	SearchMethods []SearchMethod `json:"search_methods"`
	Reranker      RerankerType   `json:"reranker"`
	MinScore      float64        `json:"min_score"`
	MMRLambda     float64        `json:"mmr_lambda"`
	MaxDepth      int            `json:"max_depth"`
}

type EpisodeSearchConfig struct {
	SearchMethods []SearchMethod `json:"search_methods"`
	Reranker      RerankerType   `json:"reranker"`
	MinScore      float64        `json:"min_score"`
}

type CommunitySearchConfig struct {
	SearchMethods []SearchMethod `json:"search_methods"`
	Reranker      RerankerType   `json:"reranker"`
	MinScore      float64        `json:"min_score"`
	MMRLambda     float64        `json:"mmr_lambda"`
}

type SearchFilters struct {
	GroupIDs     []string           `json:"group_ids,omitempty"`
	NodeTypes    []types.NodeType   `json:"node_types,omitempty"`
	EdgeTypes    []types.EdgeType   `json:"edge_types,omitempty"`
	EntityTypes  []string           `json:"entity_types,omitempty"`
	TimeRange    *types.TimeRange   `json:"time_range,omitempty"`
}

type HybridSearchResult struct {
	Nodes      []*types.Node  `json:"nodes"`
	Edges      []*types.Edge  `json:"edges"`
	NodeScores []float64      `json:"node_scores"`
	EdgeScores []float64      `json:"edge_scores"`
	Query      string         `json:"query"`
	Total      int            `json:"total"`
}

type Searcher struct {
	driver    driver.GraphDriver
	embedder  embedder.Client
	llm       llm.Client
}

func NewSearcher(driver driver.GraphDriver, embedder embedder.Client, llm llm.Client) *Searcher {
	return &Searcher{
		driver:    driver,
		embedder:  embedder,
		llm:       llm,
	}
}

func (s *Searcher) Search(ctx context.Context, query string, config *SearchConfig, filters *SearchFilters, groupID string) (*HybridSearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return &HybridSearchResult{}, nil
	}

	// Generate query embedding if needed for semantic search
	var queryVector []float32
	needsEmbedding := s.needsEmbedding(config)
	
	if needsEmbedding {
		vectors, err := s.embedder.Embed(ctx, []string{strings.ReplaceAll(query, "\n", " ")})
		if err != nil {
			return nil, fmt.Errorf("failed to create query embedding: %w", err)
		}
		if len(vectors) > 0 {
			queryVector = vectors[0]
		}
	}

	// Perform searches concurrently
	nodeResults := make([]*types.Node, 0)
	edgeResults := make([]*types.Edge, 0)
	nodeScores := make([]float64, 0)
	edgeScores := make([]float64, 0)

	// Node search
	if config.NodeConfig != nil {
		nodes, scores, err := s.searchNodes(ctx, query, queryVector, config.NodeConfig, filters, groupID, config.Limit)
		if err != nil {
			return nil, fmt.Errorf("node search failed: %w", err)
		}
		nodeResults = nodes
		nodeScores = scores
	}

	// Edge search
	if config.EdgeConfig != nil {
		edges, scores, err := s.searchEdges(ctx, query, queryVector, config.EdgeConfig, filters, groupID, config.Limit)
		if err != nil {
			return nil, fmt.Errorf("edge search failed: %w", err)
		}
		edgeResults = edges
		edgeScores = scores
	}

	return &HybridSearchResult{
		Nodes:      nodeResults,
		Edges:      edgeResults,
		NodeScores: nodeScores,
		EdgeScores: edgeScores,
		Query:      query,
		Total:      len(nodeResults) + len(edgeResults),
	}, nil
}

func (s *Searcher) needsEmbedding(config *SearchConfig) bool {
	if config.NodeConfig != nil {
		for _, method := range config.NodeConfig.SearchMethods {
			if method == CosineSimilarity {
				return true
			}
		}
		if config.NodeConfig.Reranker == MMRRerankType {
			return true
		}
	}

	if config.EdgeConfig != nil {
		for _, method := range config.EdgeConfig.SearchMethods {
			if method == CosineSimilarity {
				return true
			}
		}
		if config.EdgeConfig.Reranker == MMRRerankType {
			return true
		}
	}

	if config.CommunityConfig != nil {
		for _, method := range config.CommunityConfig.SearchMethods {
			if method == CosineSimilarity {
				return true
			}
		}
		if config.CommunityConfig.Reranker == MMRRerankType {
			return true
		}
	}

	return false
}

func (s *Searcher) searchNodes(ctx context.Context, query string, queryVector []float32, config *NodeSearchConfig, filters *SearchFilters, groupID string, limit int) ([]*types.Node, []float64, error) {
	searchResults := make([][]*types.Node, 0)

	// Execute different search methods
	for _, method := range config.SearchMethods {
		switch method {
		case BM25:
			nodes, err := s.nodeFulltextSearch(ctx, query, filters, groupID, limit*2)
			if err != nil {
				return nil, nil, fmt.Errorf("BM25 node search failed: %w", err)
			}
			searchResults = append(searchResults, nodes)

		case CosineSimilarity:
			if len(queryVector) == 0 {
				continue
			}
			nodes, err := s.nodeSimilaritySearch(ctx, queryVector, filters, groupID, limit*2, config.MinScore)
			if err != nil {
				return nil, nil, fmt.Errorf("similarity node search failed: %w", err)
			}
			searchResults = append(searchResults, nodes)

		case BreadthFirstSearch:
			// BFS requires origin nodes, implement if needed
			continue
		}
	}

	// Combine and rerank results
	return s.rerankNodes(ctx, query, queryVector, searchResults, config, limit)
}

func (s *Searcher) searchEdges(ctx context.Context, query string, queryVector []float32, config *EdgeSearchConfig, filters *SearchFilters, groupID string, limit int) ([]*types.Edge, []float64, error) {
	searchResults := make([][]*types.Edge, 0)

	// Execute different search methods
	for _, method := range config.SearchMethods {
		switch method {
		case BM25:
			edges, err := s.edgeFulltextSearch(ctx, query, filters, groupID, limit*2)
			if err != nil {
				return nil, nil, fmt.Errorf("BM25 edge search failed: %w", err)
			}
			searchResults = append(searchResults, edges)

		case CosineSimilarity:
			if len(queryVector) == 0 {
				continue
			}
			edges, err := s.edgeSimilaritySearch(ctx, queryVector, filters, groupID, limit*2, config.MinScore)
			if err != nil {
				return nil, nil, fmt.Errorf("similarity edge search failed: %w", err)
			}
			searchResults = append(searchResults, edges)

		case BreadthFirstSearch:
			// BFS requires origin nodes, implement if needed
			continue
		}
	}

	// Combine and rerank results
	return s.rerankEdges(ctx, query, queryVector, searchResults, config, limit)
}

func (s *Searcher) nodeFulltextSearch(ctx context.Context, query string, filters *SearchFilters, groupID string, limit int) ([]*types.Node, error) {
	// This would use the driver's fulltext search capabilities
	// For now, return a basic implementation
	return s.driver.SearchNodes(ctx, query, groupID, &driver.SearchOptions{
		Limit:     limit,
		UseFullText: true,
		NodeTypes: filters.NodeTypes,
	})
}

func (s *Searcher) nodeSimilaritySearch(ctx context.Context, queryVector []float32, filters *SearchFilters, groupID string, limit int, minScore float64) ([]*types.Node, error) {
	// This would use vector similarity search
	return s.driver.SearchNodesByVector(ctx, queryVector, groupID, &driver.VectorSearchOptions{
		Limit:     limit,
		MinScore:  minScore,
		NodeTypes: filters.NodeTypes,
	})
}

func (s *Searcher) edgeFulltextSearch(ctx context.Context, query string, filters *SearchFilters, groupID string, limit int) ([]*types.Edge, error) {
	return s.driver.SearchEdges(ctx, query, groupID, &driver.SearchOptions{
		Limit:     limit,
		UseFullText: true,
		EdgeTypes: filters.EdgeTypes,
	})
}

func (s *Searcher) edgeSimilaritySearch(ctx context.Context, queryVector []float32, filters *SearchFilters, groupID string, limit int, minScore float64) ([]*types.Edge, error) {
	return s.driver.SearchEdgesByVector(ctx, queryVector, groupID, &driver.VectorSearchOptions{
		Limit:     limit,
		MinScore:  minScore,
		EdgeTypes: filters.EdgeTypes,
	})
}

func (s *Searcher) rerankNodes(ctx context.Context, query string, queryVector []float32, searchResults [][]*types.Node, config *NodeSearchConfig, limit int) ([]*types.Node, []float64, error) {
	if len(searchResults) == 0 {
		return []*types.Node{}, []float64{}, nil
	}

	// Create node map for deduplication
	nodeMap := make(map[string]*types.Node)
	for _, results := range searchResults {
		for _, node := range results {
			nodeMap[node.ID] = node
		}
	}

	nodes := make([]*types.Node, 0, len(nodeMap))
	for _, node := range nodeMap {
		nodes = append(nodes, node)
	}

	switch config.Reranker {
	case RRFRerankType:
		return s.rrfRerankNodes(searchResults, limit)
	case MMRRerankType:
		return s.mmrRerankNodes(ctx, queryVector, nodes, config.MMRLambda, config.MinScore, limit)
	case CrossEncoderRerankType:
		return s.crossEncoderRerankNodes(ctx, query, nodes, config.MinScore, limit)
	default:
		// Default to simple score-based ranking
		scores := make([]float64, len(nodes))
		for i := range scores {
			scores[i] = 1.0 // Default score
		}
		return nodes[:min(limit, len(nodes))], scores[:min(limit, len(scores))], nil
	}
}

func (s *Searcher) rerankEdges(ctx context.Context, query string, queryVector []float32, searchResults [][]*types.Edge, config *EdgeSearchConfig, limit int) ([]*types.Edge, []float64, error) {
	if len(searchResults) == 0 {
		return []*types.Edge{}, []float64{}, nil
	}

	// Create edge map for deduplication
	edgeMap := make(map[string]*types.Edge)
	for _, results := range searchResults {
		for _, edge := range results {
			edgeMap[edge.ID] = edge
		}
	}

	edges := make([]*types.Edge, 0, len(edgeMap))
	for _, edge := range edgeMap {
		edges = append(edges, edge)
	}

	switch config.Reranker {
	case RRFRerankType:
		return s.rrfRerankEdges(searchResults, limit)
	case MMRRerankType:
		return s.mmrRerankEdges(ctx, queryVector, edges, config.MMRLambda, config.MinScore, limit)
	case CrossEncoderRerankType:
		return s.crossEncoderRerankEdges(ctx, query, edges, config.MinScore, limit)
	default:
		// Default to simple score-based ranking
		scores := make([]float64, len(edges))
		for i := range scores {
			scores[i] = 1.0 // Default score
		}
		return edges[:min(limit, len(edges))], scores[:min(limit, len(scores))], nil
	}
}

// RRF (Reciprocal Rank Fusion) reranking
func (s *Searcher) rrfRerankNodes(searchResults [][]*types.Node, limit int) ([]*types.Node, []float64, error) {
	scoreMap := make(map[string]float64)
	nodeMap := make(map[string]*types.Node)

	for _, results := range searchResults {
		for rank, node := range results {
			if _, exists := scoreMap[node.ID]; !exists {
				scoreMap[node.ID] = 0
			}
			// RRF formula: 1 / (rank + k), where k is typically 60
			scoreMap[node.ID] += 1.0 / float64(rank+60)
			nodeMap[node.ID] = node
		}
	}

	// Sort by score
	type nodeScore struct {
		node  *types.Node
		score float64
	}

	nodeScores := make([]nodeScore, 0, len(scoreMap))
	for id, score := range scoreMap {
		nodeScores = append(nodeScores, nodeScore{
			node:  nodeMap[id],
			score: score,
		})
	}

	sort.Slice(nodeScores, func(i, j int) bool {
		return nodeScores[i].score > nodeScores[j].score
	})

	// Extract results
	nodes := make([]*types.Node, 0, min(limit, len(nodeScores)))
	scores := make([]float64, 0, min(limit, len(nodeScores)))

	for i := 0; i < min(limit, len(nodeScores)); i++ {
		nodes = append(nodes, nodeScores[i].node)
		scores = append(scores, nodeScores[i].score)
	}

	return nodes, scores, nil
}

func (s *Searcher) rrfRerankEdges(searchResults [][]*types.Edge, limit int) ([]*types.Edge, []float64, error) {
	scoreMap := make(map[string]float64)
	edgeMap := make(map[string]*types.Edge)

	for _, results := range searchResults {
		for rank, edge := range results {
			if _, exists := scoreMap[edge.ID]; !exists {
				scoreMap[edge.ID] = 0
			}
			// RRF formula: 1 / (rank + k), where k is typically 60
			scoreMap[edge.ID] += 1.0 / float64(rank+60)
			edgeMap[edge.ID] = edge
		}
	}

	// Sort by score
	type edgeScore struct {
		edge  *types.Edge
		score float64
	}

	edgeScores := make([]edgeScore, 0, len(scoreMap))
	for id, score := range scoreMap {
		edgeScores = append(edgeScores, edgeScore{
			edge:  edgeMap[id],
			score: score,
		})
	}

	sort.Slice(edgeScores, func(i, j int) bool {
		return edgeScores[i].score > edgeScores[j].score
	})

	// Extract results
	edges := make([]*types.Edge, 0, min(limit, len(edgeScores)))
	scores := make([]float64, 0, min(limit, len(edgeScores)))

	for i := 0; i < min(limit, len(edgeScores)); i++ {
		edges = append(edges, edgeScores[i].edge)
		scores = append(scores, edgeScores[i].score)
	}

	return edges, scores, nil
}

// MMR (Maximal Marginal Relevance) reranking
func (s *Searcher) mmrRerankNodes(ctx context.Context, queryVector []float32, nodes []*types.Node, lambda float64, minScore float64, limit int) ([]*types.Node, []float64, error) {
	if len(queryVector) == 0 {
		return nodes[:min(limit, len(nodes))], make([]float64, min(limit, len(nodes))), nil
	}

	// TODO: Implement MMR algorithm
	// For now, return nodes with default scores
	scores := make([]float64, min(limit, len(nodes)))
	for i := range scores {
		scores[i] = 1.0
	}
	
	return nodes[:min(limit, len(nodes))], scores, nil
}

func (s *Searcher) mmrRerankEdges(ctx context.Context, queryVector []float32, edges []*types.Edge, lambda float64, minScore float64, limit int) ([]*types.Edge, []float64, error) {
	if len(queryVector) == 0 {
		return edges[:min(limit, len(edges))], make([]float64, min(limit, len(edges))), nil
	}

	// TODO: Implement MMR algorithm
	// For now, return edges with default scores
	scores := make([]float64, min(limit, len(edges)))
	for i := range scores {
		scores[i] = 1.0
	}
	
	return edges[:min(limit, len(edges))], scores, nil
}

// Cross-encoder reranking
func (s *Searcher) crossEncoderRerankNodes(ctx context.Context, query string, nodes []*types.Node, minScore float64, limit int) ([]*types.Node, []float64, error) {
	// TODO: Implement cross-encoder reranking using LLM
	// For now, return nodes with default scores
	scores := make([]float64, min(limit, len(nodes)))
	for i := range scores {
		scores[i] = 1.0
	}
	
	return nodes[:min(limit, len(nodes))], scores, nil
}

func (s *Searcher) crossEncoderRerankEdges(ctx context.Context, query string, edges []*types.Edge, minScore float64, limit int) ([]*types.Edge, []float64, error) {
	// TODO: Implement cross-encoder reranking using LLM
	// For now, return edges with default scores
	scores := make([]float64, min(limit, len(edges)))
	for i := range scores {
		scores[i] = 1.0
	}
	
	return edges[:min(limit, len(edges))], scores, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}