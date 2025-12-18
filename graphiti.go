package graphiti

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/community"
	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/search"
	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/soundprediction/go-graphiti/pkg/utils/maintenance"
)

// driverWrapper wraps driver.GraphDriver to implement types.EdgeOperations
type driverWrapper struct {
	driver.GraphDriver
}

// Provider converts driver.GraphProvider to types.GraphProvider
func (w *driverWrapper) Provider() types.GraphProvider {
	switch w.GraphDriver.Provider() {
	case driver.GraphProviderLadybug:
		return types.GraphProviderLadybug
	case driver.GraphProviderNeo4j:
		return types.GraphProviderNeo4j
	case driver.GraphProviderFalkorDB:
		return types.GraphProviderFalkorDB
	case driver.GraphProviderNeptune:
		return types.GraphProviderNeptune
	default:
		return types.GraphProviderLadybug // default fallback
	}
}

// nodeOpsWrapper adapts maintenance.NodeOperations to utils.NodeOperations interface
type nodeOpsWrapper struct {
	*maintenance.NodeOperations
}

// ResolveExtractedNodes wraps maintenance.NodeOperations.ResolveExtractedNodes to match the interface
func (w *nodeOpsWrapper) ResolveExtractedNodes(ctx context.Context, extractedNodes []*types.Node, episode *types.Node, previousEpisodes []*types.Node, entityTypes map[string]interface{}) ([]*types.Node, map[string]string, interface{}, error) {
	nodes, uuidMap, pairs, err := w.NodeOperations.ResolveExtractedNodes(ctx, extractedNodes, episode, previousEpisodes, entityTypes)
	// Return pairs as interface{} to satisfy the interface
	return nodes, uuidMap, pairs, err
}

// Graphiti is the main interface for interacting with temporal knowledge graphs.
// It provides methods for building, querying, and maintaining temporally-aware
// knowledge graphs designed for AI agents.
type Graphiti interface {
	// Add processes and adds new episodes to the knowledge graph.
	// Episodes can be text, conversations, or any temporal data.
	// Options parameter is optional and can be nil for default behavior.
	Add(ctx context.Context, episodes []types.Episode, options *AddEpisodeOptions) (*types.AddBulkEpisodeResults, error)

	// AddEpisode processes and adds a single episode to the knowledge graph.
	// This is equivalent to the Python add_episode method.
	AddEpisode(ctx context.Context, episode types.Episode, options *AddEpisodeOptions) (*types.AddEpisodeResults, error)

	// Search performs hybrid search across the knowledge graph combining
	// semantic embeddings, keyword search, and graph traversal.
	Search(ctx context.Context, query string, config *types.SearchConfig) (*types.SearchResults, error)

	// GetNode retrieves a specific node from the knowledge graph.
	GetNode(ctx context.Context, nodeID string) (*types.Node, error)

	// GetEdge retrieves a specific edge from the knowledge graph.
	GetEdge(ctx context.Context, edgeID string) (*types.Edge, error)

	// GetEpisodes retrieves recent episodes from the knowledge graph.
	GetEpisodes(ctx context.Context, groupID string, limit int) ([]*types.Node, error)

	// ClearGraph removes all nodes and edges from the knowledge graph for a specific group.
	ClearGraph(ctx context.Context, groupID string) error

	// CreateIndices creates database indices and constraints for optimal performance.
	CreateIndices(ctx context.Context) error

	// AddTriplet adds a triplet (subject-predicate-object) directly to the knowledge graph.
	AddTriplet(ctx context.Context, sourceNode *types.Node, edge *types.Edge, targetNode *types.Node, createEmbeddings bool) (*types.AddTripletResults, error)

	// RemoveEpisode removes an episode and its associated nodes and edges from the knowledge graph.
	RemoveEpisode(ctx context.Context, episodeUUID string) error

	// GetNodesAndEdgesByEpisode retrieves all nodes and edges associated with a specific episode.
	GetNodesAndEdgesByEpisode(ctx context.Context, episodeUUID string) ([]*types.Node, []*types.Edge, error)

	// Close closes all connections and cleans up resources.
	Close(ctx context.Context) error

	UpdateCommunities(ctx context.Context, episodeUUID string, groupID string) ([]*types.Node, []*types.Edge, error)
}

// Client is the main implementation of the Graphiti interface.
type Client struct {
	driver    driver.GraphDriver
	llm       llm.Client
	embedder  embedder.Client
	searcher  *search.Searcher
	community *community.Builder
	config    *Config
	logger    *slog.Logger
}

// Config holds configuration for the Graphiti client.
type Config struct {
	// GroupID is used to isolate data for multi-tenant scenarios
	GroupID string
	// TimeZone for temporal operations
	TimeZone *time.Location
	// Search configuration
	SearchConfig *types.SearchConfig
	// DefaultEntityTypes defines the default entity types to use when AddEpisodeOptions.EntityTypes is nil
	EntityTypes map[string]interface{}
	EdgeTypes   map[string]interface{}
	EdgeMap     map[string]map[string][]interface{}
}

// AddEpisodeOptions holds options for adding a single episode.
// This matches the optional parameters from the Python add_episode method.
type AddEpisodeOptions struct {
	// EntityTypes custom entity type definitions
	EntityTypes map[string]interface{}
	// ExcludedEntityTypes entity types to exclude from extraction
	ExcludedEntityTypes []string
	// PreviousEpisodeUUIDs UUIDs of previous episodes for context
	PreviousEpisodeUUIDs []string
	// EdgeTypes custom edge type definitions
	EdgeTypes map[string]interface{}
	// EdgeTypeMap mapping of entity pairs to edge types
	EdgeTypeMap map[string]map[string][]interface{}
	// OverwriteExisting whether to overwrite an existing episode with the same UUID
	// Default behavior is false (skip if exists)
	OverwriteExisting  bool
	GenerateEmbeddings bool
	MaxCharacters      int
}

// NewClient creates a new Graphiti client with the provided configuration.
func NewClient(driver driver.GraphDriver, llmClient llm.Client, embedderClient embedder.Client, config *Config, logger *slog.Logger) *Client {
	if config == nil {
		config = &Config{
			GroupID:  "default",
			TimeZone: time.UTC,
		}
	}
	if config.SearchConfig == nil {
		config.SearchConfig = NewDefaultSearchConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	searcher := search.NewSearcher(driver, embedderClient, llmClient)
	communityBuilder := community.NewBuilder(driver, llmClient, embedderClient)

	return &Client{
		driver:    driver,
		llm:       llmClient,
		embedder:  embedderClient,
		searcher:  searcher,
		community: communityBuilder,
		config:    config,
		logger:    logger,
	}
}

// GetDriver returns the underlying graph driver
func (c *Client) GetDriver() driver.GraphDriver {
	return c.driver
}

// GetLLM returns the LLM client
func (c *Client) GetLLM() llm.Client {
	return c.llm
}

// GetEmbedder returns the embedder client
func (c *Client) GetEmbedder() embedder.Client {
	return c.embedder
}

// GetCommunityBuilder returns the community builder
func (c *Client) GetCommunityBuilder() *community.Builder {
	return c.community
}

var (
	// ErrNodeNotFound is returned when a node is not found.
	ErrNodeNotFound = errors.New("node not found")
	// ErrEdgeNotFound is returned when an edge is not found.
	ErrEdgeNotFound = errors.New("edge not found")
	// ErrInvalidEpisode is returned when an episode is invalid.
	ErrInvalidEpisode = errors.New("invalid episode")
)
