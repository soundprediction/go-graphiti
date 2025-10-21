package graphiti

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	jsonrepair "github.com/kaptinlin/jsonrepair"
	"github.com/soundprediction/go-graphiti/pkg/community"
	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/prompts"
	"github.com/soundprediction/go-graphiti/pkg/search"
	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/soundprediction/go-graphiti/pkg/utils"
	"github.com/soundprediction/go-graphiti/pkg/utils/maintenance"
)

// driverWrapper wraps driver.GraphDriver to implement types.EdgeOperations
type driverWrapper struct {
	driver.GraphDriver
}

// Provider converts driver.GraphProvider to types.GraphProvider
func (w *driverWrapper) Provider() types.GraphProvider {
	switch w.GraphDriver.Provider() {
	case driver.GraphProviderKuzu:
		return types.GraphProviderKuzu
	case driver.GraphProviderNeo4j:
		return types.GraphProviderNeo4j
	case driver.GraphProviderFalkorDB:
		return types.GraphProviderFalkorDB
	case driver.GraphProviderNeptune:
		return types.GraphProviderNeptune
	default:
		return types.GraphProviderKuzu // default fallback
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
	// UpdateCommunities whether to update community structures
	UpdateCommunities bool
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
	OverwriteExisting   bool
	GenerateEmbeddings  bool
	MaxCharacters       int
	DeferGraphIngestion bool
	// DuckDBPath is the path to the DuckDB file for deferred ingestion
	// If empty and DeferGraphIngestion is true, defaults to "./graphiti_deferred.duckdb"
	DuckDBPath string
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

// Add processes episodes and adds them to the knowledge graph.
func (c *Client) Add(ctx context.Context, episodes []types.Episode, options *AddEpisodeOptions) (*types.AddBulkEpisodeResults, error) {
	if len(episodes) == 0 {
		return &types.AddBulkEpisodeResults{}, nil
	}

	result := &types.AddBulkEpisodeResults{
		Episodes:       []*types.Node{},
		EpisodicEdges:  []*types.Edge{},
		Nodes:          []*types.Node{},
		Edges:          []*types.Edge{},
		Communities:    []*types.Node{},
		CommunityEdges: []*types.Edge{},
	}

	for _, episode := range episodes {
		episodeResult, err := c.AddEpisode(ctx, episode, options)
		if err != nil {
			return nil, fmt.Errorf("failed to process episode %s: %w", episode.ID, err)
		}

		// Aggregate results
		if episodeResult.Episode != nil {
			result.Episodes = append(result.Episodes, episodeResult.Episode)
		}
		result.EpisodicEdges = append(result.EpisodicEdges, episodeResult.EpisodicEdges...)
		result.Nodes = append(result.Nodes, episodeResult.Nodes...)
		result.Edges = append(result.Edges, episodeResult.Edges...)
		result.Communities = append(result.Communities, episodeResult.Communities...)
		result.CommunityEdges = append(result.CommunityEdges, episodeResult.CommunityEdges...)
	}

	return result, nil
}

// AddEpisode processes and adds a single episode to the knowledge graph.
// This implementation uses bulk processing with sophisticated deduplication.
// Content is automatically chunked if it exceeds MaxCharacters, but the same
// efficient bulk processing path is used for both single and multi-chunk episodes.
func (c *Client) AddEpisode(ctx context.Context, episode types.Episode, options *AddEpisodeOptions) (*types.AddEpisodeResults, error) {
	if options == nil {
		options = &AddEpisodeOptions{}
	}
	maxCharacters := 8192
	if options.MaxCharacters > 0 {
		maxCharacters = options.MaxCharacters
	}

	// Always use the bulk processing path for consistent, sophisticated deduplication
	// If content is small, it will be processed as a single chunk
	return c.addEpisodeChunked(ctx, episode, options, maxCharacters)
}

// addEpisodeChunked chunks long episode content and uses bulk deduplication
// processing across all chunks to efficiently handle large episodes.
func (c *Client) addEpisodeChunked(ctx context.Context, episode types.Episode, options *AddEpisodeOptions, maxCharacters int) (*types.AddEpisodeResults, error) {
	now := time.Now()

	// Chunk the content
	chunks := chunkText(episode.Content, maxCharacters)

	c.logger.Info("Chunking episode content",
		"episode_id", episode.ID,
		"original_length", len(episode.Content),
		"num_chunks", len(chunks),
		"max_characters", maxCharacters)

	// Validate entity types
	if err := utils.ValidateEntityTypes(options.EntityTypes); err != nil {
		return nil, fmt.Errorf("invalid entity types: %w", err)
	}

	// Validate and set group ID
	if err := utils.ValidateGroupID(episode.GroupID); err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}
	if episode.GroupID == "" {
		episode.GroupID = utils.GetDefaultGroupID(c.driver.Provider())
	}

	// Get previous episodes for context
	var previousEpisodes []*types.Node
	var err error
	if len(options.PreviousEpisodeUUIDs) > 0 {
		for _, uuid := range options.PreviousEpisodeUUIDs {
			episodeNode, err := c.driver.GetNode(ctx, uuid, episode.GroupID)
			if err == nil && episodeNode != nil {
				previousEpisodes = append(previousEpisodes, episodeNode)
			}
		}
	} else {
		previousEpisodes, err = c.RetrieveEpisodes(
			ctx,
			episode.Reference,
			[]string{episode.GroupID},
			search.RelevantSchemaLimit,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve previous episodes: %w", err)
		}
	}

	// Create episode nodes for each chunk
	chunkEpisodes := make([]*types.Node, len(chunks))
	episodeTuples := make([]utils.EpisodeTuple, len(chunks))

	for i, chunk := range chunks {
		// Create episode node for this chunk
		chunkEpisode := types.Episode{
			ID:        fmt.Sprintf("%s_chunk_%d", episode.ID, i),
			Name:      fmt.Sprintf("%s (chunk %d/%d)", episode.Name, i+1, len(chunks)),
			Content:   chunk,
			Reference: episode.Reference,
			CreatedAt: episode.CreatedAt,
			GroupID:   episode.GroupID,
			Metadata:  episode.Metadata,
		}

		chunkNode, err := c.createEpisodeNode(ctx, chunkEpisode, options)
		if err != nil {
			return nil, fmt.Errorf("failed to create episode node for chunk %d: %w", i, err)
		}
		chunkEpisodes[i] = chunkNode

		// Convert previous episodes to Episode type for EpisodeTuple
		prevEps := make([]*types.Episode, len(previousEpisodes))
		for j, prevNode := range previousEpisodes {
			prevEps[j] = &types.Episode{
				ID:        prevNode.ID,
				Name:      prevNode.Name,
				Content:   prevNode.Content,
				Reference: prevNode.ValidFrom,
				CreatedAt: prevNode.CreatedAt,
				GroupID:   prevNode.GroupID,
				Metadata:  prevNode.Metadata,
			}
		}

		episodeTuples[i] = utils.EpisodeTuple{
			Episode:          &chunkEpisode,
			PreviousEpisodes: prevEps,
		}
	}

	// Initialize maintenance operations
	nodeOps := maintenance.NewNodeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())
	nodeOps.SetLogger(c.logger)
	edgeOps := maintenance.NewEdgeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())
	edgeOps.SetLogger(c.logger)

	// PHASE 1: ENTITY EXTRACTION for all chunks
	c.logger.Info("Starting bulk entity extraction",
		"episode_id", episode.ID,
		"num_chunks", len(chunks))

	extractedNodesByChunk := make([][]*types.Node, len(chunks))
	for i, chunkNode := range chunkEpisodes {
		extractedNodes, err := nodeOps.ExtractNodes(ctx, chunkNode, previousEpisodes,
			options.EntityTypes, options.ExcludedEntityTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to extract nodes from chunk %d: %w", i, err)
		}
		extractedNodesByChunk[i] = extractedNodes
	}

	c.logger.Info("Bulk entity extraction completed",
		"episode_id", episode.ID,
		"total_entities_extracted", func() int {
			total := 0
			for _, nodes := range extractedNodesByChunk {
				total += len(nodes)
			}
			return total
		}())

	// PHASE 2: BULK ENTITY DEDUPLICATION across all chunks
	c.logger.Info("Starting bulk entity deduplication",
		"episode_id", episode.ID,
		"num_chunks", len(chunks))

	clients := &utils.Clients{
		Driver:   c.driver,
		LLM:      c.llm,
		Embedder: c.embedder,
		Prompts:  prompts.NewLibrary(),
	}

	dedupeResult, err := utils.DedupeNodesBulk(
		ctx,
		clients,
		extractedNodesByChunk,
		episodeTuples,
		options.EntityTypes,
		&nodeOpsWrapper{nodeOps},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate nodes in bulk: %w", err)
	}

	c.logger.Info("Bulk entity deduplication completed",
		"episode_id", episode.ID,
		"uuid_mappings", len(dedupeResult.UUIDMap))

	// Collect all resolved nodes across chunks
	var allResolvedNodes []*types.Node
	seenNodeIDs := make(map[string]bool)
	for _, nodes := range dedupeResult.NodesByEpisode {
		for _, node := range nodes {
			if !seenNodeIDs[node.ID] {
				allResolvedNodes = append(allResolvedNodes, node)
				seenNodeIDs[node.ID] = true
			}
		}
	}

	// EARLY WRITE: Persist deduplicated nodes to enable cross-parallel-run deduplication
	c.logger.Info("Persisting deduplicated nodes early",
		"episode_id", episode.ID,
		"num_nodes", len(allResolvedNodes))

	for _, node := range allResolvedNodes {
		if err := c.driver.UpsertNode(ctx, node); err != nil {
			c.logger.Warn("Failed to persist deduplicated node",
				"episode_id", episode.ID,
				"node_id", node.ID,
				"error", err)
		}
	}

	c.logger.Info("Deduplicated nodes persisted",
		"episode_id", episode.ID,
		"num_nodes", len(allResolvedNodes))

	// PHASE 3: RELATIONSHIP EXTRACTION for all chunks
	c.logger.Info("Starting bulk relationship extraction",
		"episode_id", episode.ID,
		"num_chunks", len(chunks))

	var allExtractedEdges []*types.Edge
	edgeTypeMap := make(map[string][][]string)
	if options.EdgeTypeMap != nil {
		for outerEntity, innerMap := range options.EdgeTypeMap {
			for innerEntity, relationships := range innerMap {
				for _, relation := range relationships {
					edgeTypeMap[relation.(string)] = append(edgeTypeMap[relation.(string)], []string{outerEntity, innerEntity})
				}
			}
		}
	}

	for i, chunkNode := range chunkEpisodes {
		// Get resolved nodes for this chunk
		chunkNodes := dedupeResult.NodesByEpisode[chunkNode.ID]
		if len(chunkNodes) == 0 {
			continue
		}

		extractedEdges, err := edgeOps.ExtractEdges(ctx, chunkNode, chunkNodes,
			previousEpisodes, edgeTypeMap, options.EdgeTypes, episode.GroupID)
		if err != nil {
			return nil, fmt.Errorf("failed to extract edges from chunk %d: %w", i, err)
		}

		// Apply UUID mapping to edge pointers
		utils.ResolveEdgePointers(extractedEdges, dedupeResult.UUIDMap)
		allExtractedEdges = append(allExtractedEdges, extractedEdges...)
	}

	c.logger.Info("Bulk relationship extraction completed",
		"episode_id", episode.ID,
		"total_relationships_extracted", len(allExtractedEdges))

	// PHASE 4: RELATIONSHIP RESOLUTION & TEMPORAL INVALIDATION
	c.logger.Info("Starting bulk relationship resolution",
		"episode_id", episode.ID,
		"relationships_to_resolve", len(allExtractedEdges))

	var resolvedEdges []*types.Edge
	var invalidatedEdges []*types.Edge
	if len(allExtractedEdges) > 0 {
		// Use the first chunk's episode node as the episode context
		resolvedEdges, invalidatedEdges, err = edgeOps.ResolveExtractedEdges(ctx,
			allExtractedEdges, chunkEpisodes[0], allResolvedNodes, options.GenerateEmbeddings, options.EdgeTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve edges: %w", err)
		}
	}

	c.logger.Info("Bulk relationship resolution completed",
		"episode_id", episode.ID,
		"resolved_relationships", len(resolvedEdges),
		"invalidated_relationships", len(invalidatedEdges))

	// EARLY WRITE: Persist resolved edges to enable cross-parallel-run deduplication
	c.logger.Info("Persisting resolved edges early",
		"episode_id", episode.ID,
		"num_edges", len(resolvedEdges)+len(invalidatedEdges))

	allResolvedEdges := append(resolvedEdges, invalidatedEdges...)
	for _, edge := range allResolvedEdges {
		if err := c.driver.UpsertEdge(ctx, edge); err != nil {
			c.logger.Warn("Failed to persist resolved edge",
				"episode_id", episode.ID,
				"edge_id", edge.ID,
				"error", err)
		}
	}

	c.logger.Info("Resolved edges persisted",
		"episode_id", episode.ID,
		"num_edges", len(allResolvedEdges))

	// PHASE 5: ATTRIBUTE EXTRACTION
	c.logger.Info("Starting bulk attribute extraction",
		"episode_id", episode.ID,
		"entities_to_hydrate", len(allResolvedNodes))

	hydratedNodes, err := nodeOps.ExtractAttributesFromNodes(ctx,
		allResolvedNodes, chunkEpisodes[0], previousEpisodes, options.EntityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to extract attributes: %w", err)
	}

	c.logger.Info("Bulk attribute extraction completed",
		"episode_id", episode.ID,
		"hydrated_entities", len(hydratedNodes))

	// PHASE 6: BUILD EPISODIC EDGES for all chunks
	var allEpisodicEdges []*types.Edge
	for _, chunkNode := range chunkEpisodes {
		episodicEdges, err := edgeOps.BuildEpisodicEdges(ctx, hydratedNodes, chunkNode.ID, now)
		if err != nil {
			return nil, fmt.Errorf("failed to build episodic edges for chunk %s: %w", chunkNode.ID, err)
		}
		allEpisodicEdges = append(allEpisodicEdges, episodicEdges...)
	}

	// PHASE 7: FINAL UPDATES
	// Note: Entity nodes and edges were already persisted after deduplication/resolution
	// This phase updates them with hydrated attributes and adds episodic edges
	allEdges := append(resolvedEdges, invalidatedEdges...)
	c.logger.Info("Starting final updates",
		"episode_id", episode.ID,
		"episodic_nodes", len(chunkEpisodes),
		"entity_nodes_to_update", len(hydratedNodes),
		"entity_edges_to_update", len(allEdges),
		"episodic_edges_to_add", len(allEpisodicEdges))

	_, err = utils.AddNodesAndEdgesBulk(ctx, c.driver,
		chunkEpisodes,
		allEpisodicEdges,
		hydratedNodes,
		allEdges,
		c.embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to perform final updates: %w", err)
	}

	// PHASE 8: COMMUNITY UPDATE (optional)
	result := &types.AddEpisodeResults{
		Episode:        chunkEpisodes[0], // Return first chunk as main episode
		EpisodicEdges:  allEpisodicEdges,
		Nodes:          hydratedNodes,
		Edges:          allEdges,
		Communities:    []*types.Node{},
		CommunityEdges: []*types.Edge{},
	}

	if options.UpdateCommunities {
		c.logger.Info("Starting community update",
			"episode_id", episode.ID,
			"group_id", episode.GroupID)

		communityResult, err := c.community.BuildCommunities(ctx, []string{episode.GroupID})
		if err != nil {
			return nil, fmt.Errorf("failed to build communities: %w", err)
		}
		result.Communities = communityResult.CommunityNodes
		result.CommunityEdges = communityResult.CommunityEdges

		c.logger.Info("Community update completed",
			"episode_id", episode.ID,
			"communities", len(result.Communities),
			"community_edges", len(result.CommunityEdges))
	}

	c.logger.Info("Chunked episode processing completed with bulk deduplication",
		"episode_id", episode.ID,
		"total_chunks", len(chunks),
		"total_entities", len(result.Nodes),
		"total_relationships", len(result.Edges),
		"total_episodic_edges", len(result.EpisodicEdges),
		"total_communities", len(result.Communities))

	return result, nil
}

// createEpisodeNode creates an episode node in the graph.
func (c *Client) createEpisodeNode(ctx context.Context, episode types.Episode, options *AddEpisodeOptions) (*types.Node, error) {
	now := time.Now()

	// Use existing embedding or create new one if embedder is available
	var embedding []float32
	if len(episode.ContentEmbedding) > 0 {
		// Use pre-computed embedding if available
		embedding = episode.ContentEmbedding
	} else if c.embedder != nil {
		// Generate embedding if not provided and embedder is available
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
	if c.llm == nil {
		// If no LLM client is available, return empty slice
		return []*types.Node{}, nil
	}

	// 1. Use the prompt library to generate entity extraction prompts
	library := prompts.DefaultLibrary
	extractNodesPrompt := library.ExtractNodes()

	// Get previous episodes for context (simplified - in full implementation would get recent episodes)
	previousEpisodes := []string{} // Would be populated with recent episode content

	// Default entity types context (matching Python implementation)
	entityTypesContext := []map[string]interface{}{
		{
			"entity_type_id":          0,
			"entity_type_name":        "Entity",
			"entity_type_description": "Default entity classification. Use this entity type if the entity is not one of the other listed types.",
		},
	}

	// Prepare context for the prompt
	context := map[string]interface{}{
		"episode_content":    strings.ReplaceAll(episode.Content, "'", "\\'"),
		"previous_episodes":  previousEpisodes,
		"entity_types":       entityTypesContext,
		"custom_prompt":      "",
		"source_description": episode.Metadata["source_description"],
		"ensure_ascii":       false,
		"logger":             c.logger, // Add logger for debug logging
	}

	// 2. Call the LLM to extract entities
	var messages []llm.Message
	var err error

	// Determine episode type and use appropriate prompt (matching Python logic)
	episodeType := episode.Metadata["episode_type"]
	switch episodeType {
	case "message", "conversation":
		messages, err = extractNodesPrompt.ExtractMessage().Call(context)
	case "text", "document":
		messages, err = extractNodesPrompt.ExtractText().Call(context)
	case "json":
		messages, err = extractNodesPrompt.ExtractJSON().Call(context)
	default:
		// Default to text extraction
		messages, err = extractNodesPrompt.ExtractText().Call(context)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate extraction prompt: %w", err)
	}

	// Generate LLM response
	response, err := c.llm.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response for entity extraction: %w", err)
	}
	// fmt.Printf("response: %v\n", response)
	// 3. Parse the LLM response into Node structures
	entities, err := c.ParseEntitiesFromResponse(response.Content, episode.GroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse entities from LLM response: %w", err)
	}

	return entities, nil
}

// ExtractedEntity represents an entity extracted by the LLM
// Supports multiple field names for compatibility with different LLM response formats
type ExtractedEntity struct {
	Name         string `json:"name"`        // Expected format
	Entity       string `json:"entity"`      // Common LLM format
	EntityName   string `json:"entity_name"` // Alternative LLM format
	EntityTypeID int    `json:"entity_type_id"`
}

// GetEntityName returns the entity name, checking all possible field names
func (e *ExtractedEntity) GetEntityName() string {
	if e.Name != "" {
		return e.Name
	}
	if e.Entity != "" {
		return e.Entity
	}
	return e.EntityName
}

// ExtractedEntities represents the response from entity extraction (multiple formats)
type ExtractedEntities struct {
	ExtractedEntities []ExtractedEntity `json:"extracted_entities"` // Expected format
	Entities          []ExtractedEntity `json:"entities"`           // Alternative LLM format
}

// GetEntitiesList returns the entities list, checking all possible field names
func (e *ExtractedEntities) GetEntitiesList() []ExtractedEntity {
	if len(e.ExtractedEntities) > 0 {
		return e.ExtractedEntities
	}
	return e.Entities
}

// ParseEntitiesFromResponse parses the LLM response and converts it to Node structures
func (c *Client) ParseEntitiesFromResponse(responseContent, groupID string) ([]*types.Node, error) {
	// 1. Parse the structured JSON response from the LLM
	responseContent, _ = jsonrepair.JSONRepair(responseContent)

	var entitiesList []ExtractedEntity

	// Try multiple parsing strategies to handle different LLM response formats

	// Strategy 1: Try to parse as wrapped format {"extracted_entities": [...]} or {"entities": [...]}
	var extractedEntities ExtractedEntities
	if err := json.Unmarshal([]byte(responseContent), &extractedEntities); err == nil {
		entitiesList = extractedEntities.GetEntitiesList()
	}

	// Strategy 2: If wrapped format didn't work or was empty, try direct array
	if len(entitiesList) == 0 {
		if err := json.Unmarshal([]byte(responseContent), &entitiesList); err != nil {
			// Strategy 3: Try to extract JSON from response text
			jsonStart := strings.Index(responseContent, "[")
			if jsonStart == -1 {
				jsonStart = strings.Index(responseContent, "{")
			}
			jsonEnd := strings.LastIndex(responseContent, "]")
			if jsonEnd == -1 {
				jsonEnd = strings.LastIndex(responseContent, "}")
			}

			if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
				jsonContent := responseContent[jsonStart : jsonEnd+1]

				// Try direct array first
				if err := json.Unmarshal([]byte(jsonContent), &entitiesList); err != nil {
					// Try wrapped format
					var wrappedEntities ExtractedEntities
					if err := json.Unmarshal([]byte(jsonContent), &wrappedEntities); err != nil {
						// If all JSON parsing fails, fall back to simple text parsing
						return c.parseEntitiesFromText(responseContent, groupID)
					} else {
						entitiesList = wrappedEntities.GetEntitiesList()
					}
				}
			} else {
				// Fall back to simple text parsing
				return c.parseEntitiesFromText(responseContent, groupID)
			}
		}
	}

	// 2. Process the extracted entities list
	entities := make([]*types.Node, 0, len(entitiesList))
	now := time.Now()

	// Default entity types (matching Python implementation)
	entityTypes := map[int]string{
		0: "Entity", // Default entity type
	}

	// 3. Create proper EntityNode objects with all attributes
	for _, extractedEntity := range entitiesList {
		// Get entity name using flexible field mapping
		entityName := strings.TrimSpace(extractedEntity.GetEntityName())

		// Skip empty names
		if entityName == "" {
			continue
		}

		// Determine entity type from ID
		entityType := "Entity" // Default
		if entityTypeName, exists := entityTypes[extractedEntity.EntityTypeID]; exists {
			entityType = entityTypeName
		}

		entity := &types.Node{
			ID:         generateID(),
			Name:       entityName,
			Type:       types.EntityNodeType,
			GroupID:    groupID,
			CreatedAt:  now,
			UpdatedAt:  now,
			ValidFrom:  now,
			EntityType: entityType,
			Summary:    "", // Will be populated later if needed
			Metadata:   make(map[string]interface{}),
		}

		// Add entity type information to metadata
		entity.Metadata["entity_type_id"] = extractedEntity.EntityTypeID
		entity.Metadata["labels"] = []string{"Entity", entityType}

		entities = append(entities, entity)
	}

	return entities, nil
}

// parseEntitiesFromText provides fallback text-based parsing when JSON parsing fails
func (c *Client) parseEntitiesFromText(responseContent, groupID string) ([]*types.Node, error) {
	entities := []*types.Node{}
	now := time.Now()

	// Simple text-based extraction as fallback
	lines := strings.Split(responseContent, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for entity patterns in various formats
		patterns := []string{
			"entity:", "Entity:", "name:", "Name:",
			"- entity:", "- Entity:", "* entity:", "* Entity:",
		}

		for _, pattern := range patterns {
			if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
				// Extract entity name from the line
				if strings.Contains(line, ":") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						entityName := strings.TrimSpace(parts[1])
						entityName = strings.Trim(entityName, `"'.,`)

						if entityName != "" && len(entityName) > 2 {
							entity := &types.Node{
								ID:         generateID(),
								Name:       entityName,
								Type:       types.EntityNodeType,
								GroupID:    groupID,
								CreatedAt:  now,
								UpdatedAt:  now,
								ValidFrom:  now,
								EntityType: "Entity",
								Summary:    "",
								Metadata:   make(map[string]interface{}),
							}
							entities = append(entities, entity)
						}
					}
				}
				break
			}
		}
	}

	return entities, nil
}

// generateID generates a unique ID for nodes and edges.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// chunkText splits text into chunks of approximately maxChars size,
// preserving paragraph boundaries when possible. It prioritizes keeping
// complete paragraphs together and only splits within paragraphs when necessary.
func chunkText(text string, maxChars int) []string {
	if len(text) <= maxChars {
		return []string{text}
	}

	// Split text into paragraphs first (preserve paragraph structure)
	paragraphs := strings.Split(text, "\n\n")

	var chunks []string
	var currentChunk strings.Builder
	currentLen := 0

	for i, para := range paragraphs {
		paraLen := len(para)

		// If this single paragraph is longer than maxChars, we need to split it
		if paraLen > maxChars {
			// Flush current chunk if it has content
			if currentChunk.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
				currentLen = 0
			}

			// Split the large paragraph into smaller chunks
			subChunks := chunkParagraph(para, maxChars)
			chunks = append(chunks, subChunks...)
			continue
		}

		// Will adding this paragraph exceed maxChars?
		separator := ""
		if currentChunk.Len() > 0 {
			separator = "\n\n"
		}
		newLen := currentLen + len(separator) + paraLen

		if newLen > maxChars && currentChunk.Len() > 0 {
			// Adding this paragraph would exceed limit, flush current chunk
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
			currentChunk.WriteString(para)
			currentLen = paraLen
		} else {
			// Add paragraph to current chunk
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(para)
			currentLen = newLen
		}

		// If this is the last paragraph, flush the chunk
		if i == len(paragraphs)-1 && currentChunk.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
		}
	}

	return chunks
}

// chunkParagraph splits a single paragraph that's too large into smaller chunks,
// breaking at sentence or word boundaries.
func chunkParagraph(para string, maxChars int) []string {
	var chunks []string
	remaining := para

	for len(remaining) > 0 {
		if len(remaining) <= maxChars {
			chunks = append(chunks, strings.TrimSpace(remaining))
			break
		}

		// Try to find a good break point within maxChars
		chunkEnd := maxChars
		breakPoint := -1

		// Minimum chunk size to avoid tiny fragments (at least 1/3 of maxChars)
		minChunkSize := maxChars / 3

		// Try to break at a sentence boundary first
		if idx := strings.LastIndex(remaining[:chunkEnd], ". "); idx > minChunkSize {
			breakPoint = idx + 2
		} else if idx := strings.LastIndex(remaining[:chunkEnd], "! "); idx > minChunkSize {
			breakPoint = idx + 2
		} else if idx := strings.LastIndex(remaining[:chunkEnd], "? "); idx > minChunkSize {
			breakPoint = idx + 2
		} else if idx := strings.LastIndex(remaining[:chunkEnd], "\n"); idx > minChunkSize {
			// Try to break at a newline
			breakPoint = idx + 1
		} else if idx := strings.LastIndex(remaining[:chunkEnd], " "); idx > minChunkSize {
			// Try to break at a word boundary
			breakPoint = idx + 1
		} else {
			// No good break point found, just split at maxChars
			breakPoint = maxChars
		}

		chunks = append(chunks, strings.TrimSpace(remaining[:breakPoint]))
		remaining = remaining[breakPoint:]
	}

	return chunks
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
	} else {
		// Default: use all search methods for comprehensive results
		searchConfig.NodeConfig = &search.NodeSearchConfig{
			SearchMethods: []search.SearchMethod{search.CosineSimilarity, search.BM25, search.BreadthFirstSearch},
			Reranker:      search.RRFRerankType,
			MinScore:      0.0,
			MMRLambda:     0.5,
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
	} else {
		searchConfig.EdgeConfig = &search.EdgeSearchConfig{
			SearchMethods: []search.SearchMethod{search.CosineSimilarity, search.BM25, search.BreadthFirstSearch},
			Reranker:      search.RRFRerankType,
			MinScore:      0.0,
			MMRLambda:     0.5,
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

// RetrieveEpisodes retrieves episodes from the knowledge graph with temporal filtering.
// This is an exact translation of the Python retrieve_episodes() function from
// graphiti_core/utils/maintenance/graph_data_operations.py:122-181
//
// Parameters:
//   - referenceTime: Only episodes with valid_at <= referenceTime will be retrieved
//   - groupIDs: List of group IDs to filter by (can be nil for all groups)
//   - limit: Maximum number of episodes to retrieve
//   - episodeType: Optional episode type filter (nil for all types)
//
// Returns episodes in chronological order (oldest first).
func (c *Client) RetrieveEpisodes(
	ctx context.Context,
	referenceTime time.Time,
	groupIDs []string,
	limit int,
	episodeType *types.EpisodeType,
) ([]*types.Node, error) {
	if limit <= 0 {
		limit = 10
	}

	// Build query parameters
	queryParams := make(map[string]interface{})
	queryParams["reference_time"] = referenceTime
	queryParams["num_episodes"] = limit

	// Build conditional filters
	queryFilter := ""

	// Group ID filter
	if groupIDs != nil && len(groupIDs) > 0 {
		queryFilter += "\nAND e.group_id IN $group_ids"
		queryParams["group_ids"] = groupIDs
	}

	// Optional episode type filter
	if episodeType != nil {
		queryFilter += "\nAND e.episode_type = $source"
		queryParams["source"] = string(*episodeType)
	}

	// Build complete query
	// Match Python's query structure exactly from graph_data_operations.py:154-171
	// Python uses 'valid_at' not 'valid_from'
	query := fmt.Sprintf(`
		MATCH (e:Episodic)
		WHERE e.valid_at <= $reference_time
		%s
		RETURN e
		ORDER BY e.valid_at DESC
		LIMIT $num_episodes
	`, queryFilter)

	// Execute query
	result, _, _, err := c.driver.ExecuteQuery(query, queryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve episodes: %w", err)
	}

	// Parse results - the exact format depends on the driver implementation
	episodes, err := c.parseEpisodicNodesFromQueryResult(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse episodes: %w", err)
	}

	// Reverse to return in chronological order (oldest first)
	// This matches Python's: return list(reversed(episodes))
	c.reverseNodes(episodes)

	return episodes, nil
}

// GetEpisodes retrieves recent episodes from the knowledge graph.
// This is a simplified wrapper around RetrieveEpisodes for backward compatibility.
func (c *Client) GetEpisodes(ctx context.Context, groupID string, limit int) ([]*types.Node, error) {
	if groupID == "" {
		groupID = c.config.GroupID
	}

	// Use current time as reference time
	referenceTime := time.Now()

	// Call the full RetrieveEpisodes with temporal filtering
	return c.RetrieveEpisodes(ctx, referenceTime, []string{groupID}, limit, nil)
}

// parseEpisodicNodesFromQueryResult parses query results into episodic nodes
func (c *Client) parseEpisodicNodesFromQueryResult(result interface{}) ([]*types.Node, error) {
	var episodes []*types.Node

	// Handle different result formats from ExecuteQuery
	switch v := result.(type) {
	case []map[string]interface{}:
		// Result is a list of records
		for _, record := range v {
			if nodeData, ok := record["e"].(map[string]interface{}); ok {
				node, err := c.parseNodeFromMap(nodeData)
				if err != nil {
					continue // Skip malformed nodes
				}
				episodes = append(episodes, node)
			}
		}
	case []interface{}:
		// Result is a list of interfaces
		for _, item := range v {
			if record, ok := item.(map[string]interface{}); ok {
				if nodeData, ok := record["e"].(map[string]interface{}); ok {
					node, err := c.parseNodeFromMap(nodeData)
					if err != nil {
						continue // Skip malformed nodes
					}
					episodes = append(episodes, node)
				}
			}
		}
	default:
		return nil, fmt.Errorf("unexpected query result type: %T", result)
	}

	return episodes, nil
}

// parseNodeFromMap converts a map to a Node
func (c *Client) parseNodeFromMap(data map[string]interface{}) (*types.Node, error) {
	node := &types.Node{
		Metadata: make(map[string]interface{}),
	}

	// Parse basic fields
	if id, ok := data["uuid"].(string); ok {
		node.ID = id
	} else if id, ok := data["id"].(string); ok {
		node.ID = id
	}

	if name, ok := data["name"].(string); ok {
		node.Name = name
	}

	if groupID, ok := data["group_id"].(string); ok {
		node.GroupID = groupID
	}

	if content, ok := data["content"].(string); ok {
		node.Content = content
	}

	if summary, ok := data["summary"].(string); ok {
		node.Summary = summary
	}

	// Parse timestamps
	// Python uses 'valid_at' but Go Node struct uses 'ValidFrom'
	if validAt, ok := data["valid_at"].(time.Time); ok {
		node.ValidFrom = validAt
	} else if validFrom, ok := data["valid_from"].(time.Time); ok {
		node.ValidFrom = validFrom
	}

	if createdAt, ok := data["created_at"].(time.Time); ok {
		node.CreatedAt = createdAt
	}

	if updatedAt, ok := data["updated_at"].(time.Time); ok {
		node.UpdatedAt = updatedAt
	}

	// Set type
	node.Type = types.EpisodicNodeType

	// Parse episode type
	if episodeTypeStr, ok := data["episode_type"].(string); ok {
		node.EpisodeType = types.EpisodeType(episodeTypeStr)
	}

	return node, nil
}

// reverseNodes reverses a slice of nodes in place
func (c *Client) reverseNodes(nodes []*types.Node) {
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
}

// ClearGraph removes all nodes and edges from the knowledge graph for a specific group.
func (c *Client) ClearGraph(ctx context.Context, groupID string) error {
	if groupID == "" {
		groupID = c.config.GroupID
	}

	// First, get all nodes for this group
	allNodes, err := c.getAllNodesForGroup(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to get nodes for clearing: %w", err)
	}

	// Delete all nodes (this will also delete associated edges in most graph databases)
	for _, node := range allNodes {
		if err := c.driver.DeleteNode(ctx, node.ID, groupID); err != nil {
			return fmt.Errorf("failed to delete node %s: %w", node.ID, err)
		}
	}

	return nil
}

// getAllNodesForGroup retrieves all nodes for a specific group
func (c *Client) getAllNodesForGroup(ctx context.Context, groupID string) ([]*types.Node, error) {
	// Search for all nodes with a high limit and no type filter
	searchOptions := &driver.SearchOptions{
		Limit: 100000, // Large limit to get all nodes
	}

	return c.driver.SearchNodes(ctx, "", groupID, searchOptions)
}

// CreateIndices creates database indices and constraints for optimal performance.
func (c *Client) CreateIndices(ctx context.Context) error {
	return c.driver.CreateIndices(ctx)
}

// RemoveEpisode removes an episode and its associated nodes and edges from the knowledge graph.
// This is an exact translation of the Python Graphiti.remove_episode() method.
func (c *Client) RemoveEpisode(ctx context.Context, episodeUUID string) error {
	// Find the episode to be deleted
	// Equivalent to: episode = await EpisodicNode.get_by_uuid(self.driver, episode_uuid)
	episode, err := types.GetEpisodicNodeByUUID(ctx, c.driver, episodeUUID)
	if err != nil {
		return fmt.Errorf("failed to get episode: %w", err)
	}

	// Find edges mentioned by the episode
	// Equivalent to: edges = await EntityEdge.get_by_uuids(self.driver, episode.entity_edges)
	wrapper := &driverWrapper{c.driver}
	edges, err := types.GetEntityEdgesByUUIDs(ctx, wrapper, episode.EntityEdges)
	if err != nil {
		return fmt.Errorf("failed to get entity edges: %w", err)
	}

	// We should only delete edges created by the episode
	// Equivalent to: if edge.episodes and edge.episodes[0] == episode.uuid:
	var edgesToDelete []*types.Edge
	for _, edge := range edges {
		if len(edge.Episodes) > 0 && edge.Episodes[0] == episode.ID {
			edgesToDelete = append(edgesToDelete, edge)
		}
	}

	// Find nodes mentioned by the episode
	// Equivalent to: nodes = await get_mentioned_nodes(self.driver, [episode])
	mentionedNodes, err := types.GetMentionedNodes(ctx, c.driver, []*types.Node{episode})
	if err != nil {
		return fmt.Errorf("failed to get mentioned nodes: %w", err)
	}

	// We should delete all nodes that are only mentioned in the deleted episode
	var nodesToDelete []*types.Node
	for _, node := range mentionedNodes {
		// Equivalent to: query: LiteralString = 'MATCH (e:Episodic)-[:MENTIONS]->(n:Entity {uuid: $uuid}) RETURN count(*) AS episode_count'
		query := `MATCH (e:Episodic)-[:MENTIONS]->(n:Entity {uuid: $uuid}) RETURN count(*) AS episode_count`
		records, _, _, err := c.driver.ExecuteQuery(query, map[string]interface{}{
			"uuid": node.ID,
		})
		if err != nil {
			continue // Skip on error, don't delete
		}

		// Check if only one episode mentions this node
		if recordList, ok := records.([]map[string]interface{}); ok {
			for _, record := range recordList {
				if count, ok := record["episode_count"].(int64); ok && count == 1 {
					nodesToDelete = append(nodesToDelete, node)
				}
			}
		}
	}

	// Delete edges first
	// Equivalent to: await Edge.delete_by_uuids(self.driver, [edge.uuid for edge in edges_to_delete])
	if len(edgesToDelete) > 0 {
		edgeUUIDs := make([]string, len(edgesToDelete))
		for i, edge := range edgesToDelete {
			edgeUUIDs[i] = edge.ID
		}
		if err := types.DeleteEdgesByUUIDs(ctx, wrapper, edgeUUIDs); err != nil {
			return fmt.Errorf("failed to delete edges: %w", err)
		}
	}

	// Delete nodes
	// Equivalent to: await Node.delete_by_uuids(self.driver, [node.uuid for node in nodes_to_delete])
	if len(nodesToDelete) > 0 {
		nodeUUIDs := make([]string, len(nodesToDelete))
		for i, node := range nodesToDelete {
			nodeUUIDs[i] = node.ID
		}
		if err := types.DeleteNodesByUUIDs(ctx, c.driver, nodeUUIDs); err != nil {
			return fmt.Errorf("failed to delete nodes: %w", err)
		}
	}

	// Finally, delete the episode itself
	// Equivalent to: await episode.delete(self.driver)
	if err := types.DeleteNode(ctx, c.driver, episode); err != nil {
		return fmt.Errorf("failed to delete episode: %w", err)
	}

	return nil
}

// GetNodesAndEdgesByEpisode retrieves all nodes and edges associated with a specific episode.
// This is a port of the Python Graphiti.get_nodes_and_edges_by_episode() method.
func (c *Client) GetNodesAndEdgesByEpisode(ctx context.Context, episodeUUID string) ([]*types.Node, []*types.Edge, error) {
	// Get the episode first
	episode, err := c.GetNode(ctx, episodeUUID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get episode: %w", err)
	}
	if episode.Type != types.EpisodicNodeType {
		return nil, nil, fmt.Errorf("node %s is not an episode", episodeUUID)
	}

	// Find nodes mentioned by the episode
	mentionedNodes, err := types.GetMentionedNodes(ctx, c.driver, []*types.Node{episode})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get mentioned nodes: %w", err)
	}

	// Find edges mentioned by the episode
	wrapper := &driverWrapper{c.driver}
	edges, err := types.GetEntityEdgesByUUIDs(ctx, wrapper, episode.EntityEdges)
	if err != nil {
		return mentionedNodes, nil, fmt.Errorf("failed to get entity edges: %w", err)
	}

	return mentionedNodes, edges, nil
}

// Close closes the client and all its connections.
func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close()
}

// AddTriplet adds a triplet (subject-predicate-object) directly to the knowledge graph.
// This is an exact translation of the Python Graphiti.add_triplet() method.
func (c *Client) AddTriplet(ctx context.Context, sourceNode *types.Node, edge *types.Edge, targetNode *types.Node, createEmbeddings bool) (*types.AddTripletResults, error) {
	if sourceNode == nil || edge == nil || targetNode == nil {
		return nil, fmt.Errorf("source node, edge, and target node must not be nil")
	}

	// Step 1: Generate name embeddings for nodes if missing (lines 1024-1027)
	// Equivalent to: if source_node.name_embedding is None: await source_node.generate_name_embedding(self.embedder)
	if len(sourceNode.NameEmbedding) == 0 && c.embedder != nil {
		embedding, err := c.embedder.EmbedSingle(ctx, sourceNode.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to generate name embedding for source node: %w", err)
		}
		sourceNode.NameEmbedding = embedding
	}

	if len(targetNode.NameEmbedding) == 0 && c.embedder != nil {
		embedding, err := c.embedder.EmbedSingle(ctx, targetNode.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to generate name embedding for target node: %w", err)
		}
		targetNode.NameEmbedding = embedding
	}

	// Step 2: Generate fact embedding for edge if missing (lines 1028-1029)
	// Equivalent to: if edge.fact_embedding is None: await edge.generate_embedding(self.embedder)
	if len(edge.FactEmbedding) == 0 && c.embedder != nil {
		embedding, err := c.embedder.EmbedSingle(ctx, edge.Fact)
		if err != nil {
			return nil, fmt.Errorf("failed to generate fact embedding for edge: %w", err)
		}
		edge.FactEmbedding = embedding
	}

	// Step 3: Resolve extracted nodes (lines 1031-1034)
	nodeOps := maintenance.NewNodeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())
	nodeOps.SetLogger(c.logger)
	nodes, uuidMap, _, err := nodeOps.ResolveExtractedNodes(ctx, []*types.Node{sourceNode, targetNode}, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve extracted nodes: %w", err)
	}

	// Step 4: Update edge pointers to resolved node UUIDs (line 1036)
	utils.ResolveEdgePointers([]*types.Edge{edge}, uuidMap)
	updatedEdge := edge // The edge is updated in-place

	// Step 5: Get existing edges between nodes (lines 1038-1040)
	edgeOps := maintenance.NewEdgeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())
	edgeOps.SetLogger(c.logger)
	validEdges, err := edgeOps.GetBetweenNodes(ctx, updatedEdge.SourceID, updatedEdge.TargetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get edges between nodes: %w", err)
	}

	// Step 6: Search for related edges with edge UUID filters (lines 1042-1050)
	var edgeUUIDs []string
	for _, validEdge := range validEdges {
		edgeUUIDs = append(edgeUUIDs, validEdge.ID)
	}

	searchFilters := &search.SearchFilters{
		EdgeTypes: []types.EdgeType{types.EntityEdgeType}, // Filter for entity edges
	}

	// Use edge hybrid search RRF config
	edgeSearchConfig := &search.SearchConfig{
		EdgeConfig: &search.EdgeSearchConfig{
			SearchMethods: []search.SearchMethod{search.BM25, search.CosineSimilarity},
			Reranker:      search.RRFRerankType,
			MinScore:      0.0,
		},
		Limit:    20,
		MinScore: 0.0,
	}

	relatedResults, err := c.searcher.Search(ctx, updatedEdge.Summary, edgeSearchConfig, searchFilters, updatedEdge.GroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to search for related edges: %w", err)
	}
	relatedEdges := relatedResults.Edges

	// Step 7: Search for existing edges without filters (lines 1051-1059)
	existingResults, err := c.searcher.Search(ctx, updatedEdge.Summary, edgeSearchConfig, &search.SearchFilters{}, updatedEdge.GroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to search for existing edges: %w", err)
	}
	existingEdges := existingResults.Edges

	// Step 8: Create EpisodicNode exactly as in Python (lines 1066-1074)
	var validAt time.Time
	if !updatedEdge.ValidFrom.IsZero() {
		validAt = updatedEdge.ValidFrom
	} else {
		validAt = time.Now()
	}

	episodicNode := &types.Node{
		Name:        "",
		Type:        types.EpisodicNodeType,
		EpisodeType: types.DocumentEpisodeType, // Equivalent to Python's EpisodeType.text
		Content:     "",
		Summary:     "",
		ValidFrom:   validAt,
		GroupID:     updatedEdge.GroupID,
	}

	// Step 9: Resolve extracted edge (lines 1061-1077)
	resolvedEdge, invalidatedEdges, err := c.resolveExtractedEdgeExact(ctx, updatedEdge, relatedEdges, existingEdges, episodicNode, createEmbeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve extracted edge: %w", err)
	}

	// Step 10: Combine all edges (line 1079)
	allEdges := []*types.Edge{resolvedEdge}
	allEdges = append(allEdges, invalidatedEdges...)

	// Step 11: Create entity edge embeddings (line 1081)
	err = c.createEntityEdgeEmbeddings(ctx, allEdges)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity edge embeddings: %w", err)
	}

	// Step 12: Create entity node embeddings (line 1082)
	err = c.createEntityNodeEmbeddings(ctx, nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity node embeddings: %w", err)
	}

	// Step 13: Add nodes and edges in bulk (line 1084)
	_, err = utils.AddNodesAndEdgesBulk(ctx, c.driver, []*types.Node{}, []*types.Edge{}, nodes, allEdges, c.embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to add nodes and edges to database: %w", err)
	}

	// Step 14: Return results (line 1085)
	return &types.AddTripletResults{
		Edges: allEdges,
		Nodes: nodes,
	}, nil
}

// resolveExtractedEdgeExact is an exact translation of Python's resolve_extracted_edge function
func (c *Client) resolveExtractedEdgeExact(ctx context.Context, extractedEdge *types.Edge, relatedEdges []*types.Edge, existingEdges []*types.Edge, episode *types.Node, createEmbeddings bool) (*types.Edge, []*types.Edge, error) {
	// Use the EdgeOperations to resolve the edge exactly as in Python
	edgeOps := maintenance.NewEdgeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())
	edgeOps.SetLogger(c.logger)

	// The Go implementation wraps the private resolveExtractedEdge method
	// We'll use ResolveExtractedEdges which internally calls the same logic
	resolvedEdges, invalidatedEdges, err := edgeOps.ResolveExtractedEdges(ctx, []*types.Edge{extractedEdge}, episode, []*types.Node{}, createEmbeddings, c.config.EdgeTypes)
	if err != nil {
		return nil, nil, err
	}

	var resolvedEdge *types.Edge
	if len(resolvedEdges) > 0 {
		resolvedEdge = resolvedEdges[0]
	} else {
		resolvedEdge = extractedEdge
	}

	return resolvedEdge, invalidatedEdges, nil
}

// createEntityEdgeEmbeddings creates embeddings for entity edges (equivalent to Python's create_entity_edge_embeddings)
func (c *Client) createEntityEdgeEmbeddings(ctx context.Context, edges []*types.Edge) error {
	if c.embedder == nil {
		return nil
	}

	for _, edge := range edges {
		if edge.Type == types.EntityEdgeType && len(edge.Embedding) == 0 && edge.Summary != "" {
			embedding, err := c.embedder.EmbedSingle(ctx, edge.Summary)
			if err != nil {
				return fmt.Errorf("failed to create embedding for edge %s: %w", edge.ID, err)
			}
			edge.Embedding = embedding
		}
	}

	return nil
}

// createEntityNodeEmbeddings creates embeddings for entity nodes (equivalent to Python's create_entity_node_embeddings)
func (c *Client) createEntityNodeEmbeddings(ctx context.Context, nodes []*types.Node) error {
	if c.embedder == nil {
		return nil
	}

	for _, node := range nodes {
		if node.Type == types.EntityNodeType && len(node.Embedding) == 0 && node.Name != "" {
			embedding, err := c.embedder.EmbedSingle(ctx, node.Name)
			if err != nil {
				return fmt.Errorf("failed to create embedding for node %s: %w", node.ID, err)
			}
			node.Embedding = embedding
		}
	}

	return nil
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
