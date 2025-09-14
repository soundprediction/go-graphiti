package graphiti

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/search"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// Graphiti is the main interface for interacting with temporal knowledge graphs.
// It provides methods for building, querying, and maintaining temporally-aware
// knowledge graphs designed for AI agents.
type Graphiti interface {
	// Add processes and adds new episodes to the knowledge graph.
	// Episodes can be text, conversations, or any temporal data.
	Add(ctx context.Context, episodes []types.Episode) error

	// Search performs hybrid search across the knowledge graph combining
	// semantic embeddings, keyword search, and graph traversal.
	Search(ctx context.Context, query string, config *types.SearchConfig) (*types.SearchResults, error)

	// GetNode retrieves a specific node from the knowledge graph.
	GetNode(ctx context.Context, nodeID string) (*types.Node, error)

	// GetEdge retrieves a specific edge from the knowledge graph.
	GetEdge(ctx context.Context, edgeID string) (*types.Edge, error)

	// Close closes all connections and cleans up resources.
	Close(ctx context.Context) error
}

// Client is the main implementation of the Graphiti interface.
type Client struct {
	driver   driver.GraphDriver
	llm      llm.Client
	embedder embedder.Client
	searcher *search.Searcher
	config   *Config
}

// Config holds configuration for the Graphiti client.
type Config struct {
	// GroupID is used to isolate data for multi-tenant scenarios
	GroupID string
	// TimeZone for temporal operations
	TimeZone *time.Location
	// Search configuration
	SearchConfig *types.SearchConfig
}

// NewClient creates a new Graphiti client with the provided configuration.
func NewClient(driver driver.GraphDriver, llmClient llm.Client, embedderClient embedder.Client, config *Config) *Client {
	if config == nil {
		config = &Config{
			GroupID:  "default",
			TimeZone: time.UTC,
		}
	}
	if config.SearchConfig == nil {
		config.SearchConfig = NewDefaultSearchConfig()
	}

	searcher := search.NewSearcher(driver, embedderClient, llmClient)

	return &Client{
		driver:   driver,
		llm:      llmClient,
		embedder: embedderClient,
		searcher: searcher,
		config:   config,
	}
}

// Add processes episodes and adds them to the knowledge graph.
func (c *Client) Add(ctx context.Context, episodes []types.Episode) error {
	if len(episodes) == 0 {
		return nil
	}

	for _, episode := range episodes {
		if err := c.processEpisode(ctx, episode); err != nil {
			return fmt.Errorf("failed to process episode %s: %w", episode.ID, err)
		}
	}

	return nil
}

// processEpisode processes a single episode through the knowledge extraction pipeline.
func (c *Client) processEpisode(ctx context.Context, episode types.Episode) error {
	// 1. Create episode node in graph
	episodeNode, err := c.createEpisodeNode(ctx, episode)
	if err != nil {
		return fmt.Errorf("failed to create episode node: %w", err)
	}

	// 2. Extract entities from episode content if LLM is available
	var extractedNodes []*types.Node
	if c.llm != nil {
		extractedNodes, err = c.extractEntities(ctx, episode)
		if err != nil {
			return fmt.Errorf("failed to extract entities: %w", err)
		}
	}

	// 3. Deduplicate and store nodes
	finalNodes, err := c.deduplicateAndStoreNodes(ctx, extractedNodes, episode.GroupID)
	if err != nil {
		return fmt.Errorf("failed to deduplicate and store nodes: %w", err)
	}

	// 4. Extract relationships between entities if LLM is available
	var extractedEdges []*types.Edge
	if c.llm != nil && len(finalNodes) > 1 {
		extractedEdges, err = c.extractRelationships(ctx, episode, finalNodes)
		if err != nil {
			return fmt.Errorf("failed to extract relationships: %w", err)
		}
	}

	// 5. Store edges in graph
	for _, edge := range extractedEdges {
		if err := c.driver.UpsertEdge(ctx, edge); err != nil {
			return fmt.Errorf("failed to store edge %s: %w", edge.ID, err)
		}
	}

	// 6. Create episodic edges connecting episode to extracted entities
	for _, node := range finalNodes {
		episodeEdge := &types.Edge{
			ID:        generateID(),
			Type:      types.EpisodicEdgeType,
			SourceID:  episodeNode.ID,
			TargetID:  node.ID,
			GroupID:   episode.GroupID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			ValidFrom: episode.Reference,
			Name:      "MENTIONED_IN",
			Summary:   "Entity mentioned in episode",
		}

		if err := c.driver.UpsertEdge(ctx, episodeEdge); err != nil {
			return fmt.Errorf("failed to create episodic edge: %w", err)
		}
	}

	return nil
}

// createEpisodeNode creates an episode node in the graph.
func (c *Client) createEpisodeNode(ctx context.Context, episode types.Episode) (*types.Node, error) {
	now := time.Now()

	// Create embedding for episode content if embedder is available
	var embedding []float32
	if c.embedder != nil {
		var err error
		embedding, err = c.embedder.EmbedSingle(ctx, episode.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to create episode embedding: %w", err)
		}
	}

	episodeNode := &types.Node{
		ID:          episode.ID,
		Name:        episode.Name,
		Type:        types.EpisodicNodeType,
		GroupID:     episode.GroupID,
		CreatedAt:   now,
		UpdatedAt:   now,
		EpisodeType: types.ConversationEpisodeType, // Default to conversation type
		Content:     episode.Content,
		Reference:   episode.Reference,
		ValidFrom:   episode.Reference,
		Embedding:   embedding,
		Metadata:    episode.Metadata,
	}

	if err := c.driver.UpsertNode(ctx, episodeNode); err != nil {
		return nil, fmt.Errorf("failed to create episode node: %w", err)
	}

	return episodeNode, nil
}

// extractEntities uses LLM to extract entities from episode content.
func (c *Client) extractEntities(ctx context.Context, episode types.Episode) ([]*types.Node, error) {
	// This is a simplified implementation. In a full implementation, you would:
	// 1. Use the prompt library to generate entity extraction prompts
	// 2. Call the LLM to extract entities
	// 3. Parse the LLM response into Node structures

	// For now, return empty slice - this would be implemented with proper prompt engineering
	return []*types.Node{}, nil
}

// deduplicateAndStoreNodes deduplicates nodes against existing nodes and stores them.
func (c *Client) deduplicateAndStoreNodes(ctx context.Context, nodes []*types.Node, groupID string) ([]*types.Node, error) {
	var finalNodes []*types.Node

	for _, node := range nodes {
		// Check if node already exists (simple name-based deduplication for now)
		existingNode, err := c.findNodeByName(ctx, node.Name, node.EntityType, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to search for existing node: %w", err)
		}

		if existingNode != nil {
			// Node already exists, use existing one
			finalNodes = append(finalNodes, existingNode)
		} else {
			// Create new node
			node.ID = generateID()
			node.GroupID = groupID
			now := time.Now()
			node.CreatedAt = now
			node.UpdatedAt = now
			node.ValidFrom = now

			// Create embedding if embedder available
			if c.embedder != nil && node.Summary != "" {
				embedding, err := c.embedder.EmbedSingle(ctx, node.Summary)
				if err != nil {
					return nil, fmt.Errorf("failed to create node embedding: %w", err)
				}
				node.Embedding = embedding
			}

			if err := c.driver.UpsertNode(ctx, node); err != nil {
				return nil, fmt.Errorf("failed to create node: %w", err)
			}

			finalNodes = append(finalNodes, node)
		}
	}

	return finalNodes, nil
}

// extractRelationships uses LLM to extract relationships between entities.
func (c *Client) extractRelationships(ctx context.Context, episode types.Episode, nodes []*types.Node) ([]*types.Edge, error) {
	// This is a simplified implementation. In a full implementation, you would:
	// 1. Use the prompt library to generate relationship extraction prompts
	// 2. Call the LLM to extract relationships
	// 3. Parse the LLM response into Edge structures

	// For now, return empty slice - this would be implemented with proper prompt engineering
	return []*types.Edge{}, nil
}

// findNodeByName searches for an existing node by name and entity type.
func (c *Client) findNodeByName(ctx context.Context, name, entityType, groupID string) (*types.Node, error) {
	// This is a placeholder implementation. In a real implementation, you would
	// search the graph database for nodes with matching name and entity type.
	// For now, we'll return nil (no existing node found).
	return nil, nil
}

// generateID generates a unique ID for nodes and edges.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// Search performs hybrid search across the knowledge graph.
func (c *Client) Search(ctx context.Context, query string, config *types.SearchConfig) (*types.SearchResults, error) {
	if config == nil {
		config = c.config.SearchConfig
	}

	// Convert types.SearchConfig to search.SearchConfig
	searchConfig := &search.SearchConfig{
		Limit:    config.Limit,
		MinScore: config.MinScore,
	}

	// Convert node config if present
	if config.NodeConfig != nil {
		searchConfig.NodeConfig = &search.NodeSearchConfig{
			SearchMethods: convertSearchMethods(config.NodeConfig.SearchMethods),
			Reranker:      convertReranker(config.NodeConfig.Reranker),
			MinScore:      config.NodeConfig.MinScore,
			MMRLambda:     0.5, // Default MMR lambda
			MaxDepth:      config.CenterNodeDistance,
		}
	}

	// Convert edge config if present
	if config.EdgeConfig != nil {
		searchConfig.EdgeConfig = &search.EdgeSearchConfig{
			SearchMethods: convertSearchMethods(config.EdgeConfig.SearchMethods),
			Reranker:      convertReranker(config.EdgeConfig.Reranker),
			MinScore:      config.EdgeConfig.MinScore,
			MMRLambda:     0.5, // Default MMR lambda
			MaxDepth:      config.CenterNodeDistance,
		}
	}

	// Create search filters
	filters := &search.SearchFilters{}

	// Perform the search
	result, err := c.searcher.Search(ctx, query, searchConfig, filters, c.config.GroupID)
	if err != nil {
		return nil, err
	}

	// Convert back to types.SearchResults
	searchResults := &types.SearchResults{
		Nodes: result.Nodes,
		Edges: result.Edges,
		Query: result.Query,
		Total: result.Total,
	}

	return searchResults, nil
}

// GetNode retrieves a node by ID.
func (c *Client) GetNode(ctx context.Context, nodeID string) (*types.Node, error) {
	return c.driver.GetNode(ctx, nodeID, c.config.GroupID)
}

// GetEdge retrieves an edge by ID.
func (c *Client) GetEdge(ctx context.Context, edgeID string) (*types.Edge, error) {
	return c.driver.GetEdge(ctx, edgeID, c.config.GroupID)
}

// Close closes the client and all its connections.
func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// NewDefaultSearchConfig creates a default search configuration.
func NewDefaultSearchConfig() *types.SearchConfig {
	return &types.SearchConfig{
		Limit:              20,
		CenterNodeDistance: 2,
		MinScore:           0.0,
		IncludeEdges:       true,
		Rerank:             false,
	}
}

// Helper functions for converting between type systems

func convertSearchMethods(methods []string) []search.SearchMethod {
	converted := make([]search.SearchMethod, len(methods))
	for i, method := range methods {
		switch method {
		case "cosine_similarity":
			converted[i] = search.CosineSimilarity
		case "bm25":
			converted[i] = search.BM25
		case "bfs", "breadth_first_search":
			converted[i] = search.BreadthFirstSearch
		default:
			converted[i] = search.BM25 // Default fallback
		}
	}
	return converted
}

func convertReranker(reranker string) search.RerankerType {
	switch reranker {
	case "rrf":
		return search.RRFRerankType
	case "mmr":
		return search.MMRRerankType
	case "cross_encoder":
		return search.CrossEncoderRerankType
	case "node_distance":
		return search.NodeDistanceRerankType
	case "episode_mentions":
		return search.EpisodeMentionsRerankType
	default:
		return search.RRFRerankType // Default fallback
	}
}

var (
	// ErrNodeNotFound is returned when a node is not found.
	ErrNodeNotFound = errors.New("node not found")
	// ErrEdgeNotFound is returned when an edge is not found.
	ErrEdgeNotFound = errors.New("edge not found")
	// ErrInvalidEpisode is returned when an episode is invalid.
	ErrInvalidEpisode = errors.New("invalid episode")
)
