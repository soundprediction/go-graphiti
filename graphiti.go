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
// This implementation follows the Python graphiti.add_episode() flow exactly.
// If the episode content exceeds MaxCharacters, it will be chunked and processed in parts.
func (c *Client) AddEpisode(ctx context.Context, episode types.Episode, options *AddEpisodeOptions) (*types.AddEpisodeResults, error) {
	if options == nil {
		options = &AddEpisodeOptions{}
	}
	maxCharacters := 80000
	if options.MaxCharacters > 0 {
		maxCharacters = options.MaxCharacters
	}

	// Check if we need to chunk the episode
	if len(episode.Content) > maxCharacters {
		return c.addEpisodeChunked(ctx, episode, options, maxCharacters)
	}

	// Process single episode (no chunking needed)
	return c.addEpisodeSingle(ctx, episode, options)
}

// addEpisodeChunked chunks long episode content and processes each chunk separately,
// then merges the results.
func (c *Client) addEpisodeChunked(ctx context.Context, episode types.Episode, options *AddEpisodeOptions, maxCharacters int) (*types.AddEpisodeResults, error) {
	// Chunk the content
	chunks := chunkText(episode.Content, maxCharacters)

	c.logger.Info("Chunking episode content",
		"episode_id", episode.ID,
		"original_length", len(episode.Content),
		"num_chunks", len(chunks),
		"max_characters", maxCharacters)

	// Create merged result
	mergedResult := &types.AddEpisodeResults{
		EpisodicEdges:  []*types.Edge{},
		Nodes:          []*types.Node{},
		Edges:          []*types.Edge{},
		Communities:    []*types.Node{},
		CommunityEdges: []*types.Edge{},
	}

	// Track unique nodes and edges to avoid duplicates
	nodeIDMap := make(map[string]*types.Node)
	edgeIDMap := make(map[string]*types.Edge)
	episodicEdgeIDMap := make(map[string]*types.Edge)
	communityIDMap := make(map[string]*types.Node)
	communityEdgeIDMap := make(map[string]*types.Edge)

	// Process each chunk
	for i, chunk := range chunks {
		// Create a new episode for this chunk
		chunkEpisode := types.Episode{
			ID:               fmt.Sprintf("%s_chunk_%d", episode.ID, i),
			Name:             fmt.Sprintf("%s (chunk %d/%d)", episode.Name, i+1, len(chunks)),
			Content:          chunk,
			Reference:        episode.Reference,
			CreatedAt:        episode.CreatedAt,
			GroupID:          episode.GroupID,
			Metadata:         episode.Metadata,
			ContentEmbedding: nil, // Will be generated for each chunk
		}

		// Process the chunk
		chunkResult, err := c.addEpisodeSingle(ctx, chunkEpisode, options)
		if err != nil {
			return nil, fmt.Errorf("failed to process chunk %d: %w", i, err)
		}

		// Set the first chunk's episode as the main episode
		if i == 0 && chunkResult.Episode != nil {
			mergedResult.Episode = chunkResult.Episode
		}

		// Merge episodic edges (avoiding duplicates)
		for _, edge := range chunkResult.EpisodicEdges {
			if _, exists := episodicEdgeIDMap[edge.ID]; !exists {
				episodicEdgeIDMap[edge.ID] = edge
				mergedResult.EpisodicEdges = append(mergedResult.EpisodicEdges, edge)
			}
		}

		// Merge nodes (avoiding duplicates)
		for _, node := range chunkResult.Nodes {
			if _, exists := nodeIDMap[node.ID]; !exists {
				nodeIDMap[node.ID] = node
				mergedResult.Nodes = append(mergedResult.Nodes, node)
			}
		}

		// Merge edges (avoiding duplicates)
		for _, edge := range chunkResult.Edges {
			if _, exists := edgeIDMap[edge.ID]; !exists {
				edgeIDMap[edge.ID] = edge
				mergedResult.Edges = append(mergedResult.Edges, edge)
			}
		}

		// Merge communities (avoiding duplicates)
		for _, community := range chunkResult.Communities {
			if _, exists := communityIDMap[community.ID]; !exists {
				communityIDMap[community.ID] = community
				mergedResult.Communities = append(mergedResult.Communities, community)
			}
		}

		// Merge community edges (avoiding duplicates)
		for _, edge := range chunkResult.CommunityEdges {
			if _, exists := communityEdgeIDMap[edge.ID]; !exists {
				communityEdgeIDMap[edge.ID] = edge
				mergedResult.CommunityEdges = append(mergedResult.CommunityEdges, edge)
			}
		}
	}

	c.logger.Info("Chunked episode processing completed",
		"episode_id", episode.ID,
		"total_chunks", len(chunks),
		"total_entities", len(mergedResult.Nodes),
		"total_relationships", len(mergedResult.Edges),
		"total_episodic_edges", len(mergedResult.EpisodicEdges),
		"total_communities", len(mergedResult.Communities))

	return mergedResult, nil
}

// addEpisodeSingle processes a single episode without chunking.
func (c *Client) addEpisodeSingle(ctx context.Context, episode types.Episode, options *AddEpisodeOptions) (*types.AddEpisodeResults, error) {
	// Use default entity types from config if not provided in options
	if options.EntityTypes == nil && c.config.EntityTypes != nil {
		options.EntityTypes = c.config.EntityTypes
	}

	now := time.Now()

	// PHASE 1: VALIDATION
	// Validate entity types
	if err := utils.ValidateEntityTypes(options.EntityTypes); err != nil {
		return nil, fmt.Errorf("invalid entity types: %w", err)
	}

	// Validate excluded entity types
	entityTypeNames := make([]string, 0, len(options.EntityTypes))
	for name := range options.EntityTypes {
		entityTypeNames = append(entityTypeNames, name)
	}
	if err := utils.ValidateExcludedEntityTypes(options.ExcludedEntityTypes, entityTypeNames); err != nil {
		return nil, fmt.Errorf("invalid excluded entity types: %w", err)
	}

	// Validate and set group ID
	if err := utils.ValidateGroupID(episode.GroupID); err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}
	if episode.GroupID == "" {
		episode.GroupID = utils.GetDefaultGroupID(c.driver.Provider())
	}

	// PHASE 2: CONTEXT RETRIEVAL
	// Get previous episodes for context
	var previousEpisodes []*types.Node
	var err error

	if len(options.PreviousEpisodeUUIDs) > 0 {
		// Get specific episodes by UUIDs
		for _, uuid := range options.PreviousEpisodeUUIDs {
			episodeNode, err := c.driver.GetNode(ctx, uuid, episode.GroupID)
			if err == nil && episodeNode != nil {
				previousEpisodes = append(previousEpisodes, episodeNode)
			}
		}
	} else {
		// Get recent episodes for context (simplified - using group ID and limit)
		previousEpisodes, err = c.GetEpisodes(ctx, episode.GroupID, search.RelevantSchemaLimit)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve previous episodes: %w", err)
		}
	}

	// Get or create episode node
	var episodeNode *types.Node
	if episode.ID != "" {
		// Check if episode with same UUID already exists
		existingNode, err := c.driver.GetNode(ctx, episode.ID, episode.GroupID)
		if err == nil && existingNode != nil {
			if !options.OverwriteExisting {
				// Default behavior: skip existing episode
				return &types.AddEpisodeResults{
					Episode:        existingNode,
					EpisodicEdges:  []*types.Edge{},
					Nodes:          []*types.Node{},
					Edges:          []*types.Edge{},
					Communities:    []*types.Node{},
					CommunityEdges: []*types.Edge{},
				}, nil
			}
			// OverwriteExisting is true - remove existing episode first
			if err := c.RemoveEpisode(ctx, episode.ID); err != nil {
				return nil, fmt.Errorf("failed to remove existing episode %s: %w", episode.ID, err)
			}
		}
		episodeNode, err = c.driver.GetNode(ctx, episode.ID, episode.GroupID)
		if err != nil || episodeNode == nil {
			// Create new episode node
			episodeNode, err = c.createEpisodeNode(ctx, episode, options)
			if err != nil {
				return nil, fmt.Errorf("failed to create episode node: %w", err)
			}
		}
	} else {
		// Create new episode node
		episodeNode, err = c.createEpisodeNode(ctx, episode, options)
		if err != nil {
			return nil, fmt.Errorf("failed to create episode node: %w", err)
		}
	}

	result := &types.AddEpisodeResults{
		Episode:        episodeNode,
		EpisodicEdges:  []*types.Edge{},
		Nodes:          []*types.Node{},
		Edges:          []*types.Edge{},
		Communities:    []*types.Node{},
		CommunityEdges: []*types.Edge{},
	}

	// Initialize maintenance operations
	nodeOps := maintenance.NewNodeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())
	edgeOps := maintenance.NewEdgeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())

	// PHASE 3: ENTITY EXTRACTION
	var extractedNodes []*types.Node
	if c.llm != nil {
		c.logger.Info("Starting entity extraction",
			"episode_id", episodeNode.ID,
			"group_id", episode.GroupID,
			"previous_episodes", len(previousEpisodes))

		extractedNodes, err = nodeOps.ExtractNodes(ctx, episodeNode, previousEpisodes,
			options.EntityTypes, options.ExcludedEntityTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to extract nodes: %w", err)
		}

		c.logger.Info("Entity extraction completed",
			"episode_id", episodeNode.ID,
			"entities_extracted", len(extractedNodes))
	}

	// PHASE 4: ENTITY RESOLUTION & DEDUPLICATION
	var resolvedNodes []*types.Node
	var uuidMap map[string]string
	var duplicatePairs []maintenance.NodePair

	if len(extractedNodes) > 0 {
		c.logger.Info("Starting entity resolution and deduplication",
			"episode_id", episodeNode.ID,
			"entities_to_resolve", len(extractedNodes))

		resolvedNodes, uuidMap, duplicatePairs, err = nodeOps.ResolveExtractedNodes(ctx,
			extractedNodes, episodeNode, previousEpisodes, options.EntityTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve nodes: %w", err)
		}

		c.logger.Info("Entity resolution completed",
			"episode_id", episodeNode.ID,
			"resolved_entities", len(resolvedNodes),
			"duplicates_found", len(duplicatePairs))

		// Build and store duplicate edges
		if len(duplicatePairs) > 0 {
			duplicateEdges, err := edgeOps.BuildDuplicateOfEdges(ctx, episodeNode, now, duplicatePairs)
			if err != nil {
				return nil, fmt.Errorf("failed to build duplicate edges: %w", err)
			}
			for _, edge := range duplicateEdges {
				if err := c.driver.UpsertEdge(ctx, edge); err != nil {
					return nil, fmt.Errorf("failed to store duplicate edge: %w", err)
				}
			}
		}
	}

	// PHASE 5: RELATIONSHIP EXTRACTION
	var extractedEdges []*types.Edge
	if c.llm != nil && len(resolvedNodes) > 0 {
		c.logger.Info("Starting relationship extraction",
			"episode_id", episodeNode.ID,
			"entity_count", len(resolvedNodes))

		// Create edge type map if needed
		edgeTypeMapInterface := options.EdgeTypeMap
		if edgeTypeMapInterface == nil && c.config.EdgeTypes != nil {
			edgeTypeMapInterface = c.config.EdgeMap
		}

		edgeTypeMap := make(map[string][][]string)

		for outerEntity, innerMap := range edgeTypeMapInterface {
			for innerEntity, relationships := range innerMap {
				for _, relation := range relationships {
					edgeTypeMap[relation.(string)] = append(edgeTypeMap[relation.(string)], []string{outerEntity, innerEntity})
				}
			}

		}

		extractedEdges, err = edgeOps.ExtractEdges(ctx, episodeNode, resolvedNodes,
			previousEpisodes, edgeTypeMap, options.EdgeTypes, episode.GroupID)
		if err != nil {
			return nil, fmt.Errorf("failed to extract edges: %w", err)
		}

		c.logger.Info("Relationship extraction completed",
			"episode_id", episodeNode.ID,
			"relationships_extracted", len(extractedEdges))
	}

	// PHASE 6: RELATIONSHIP RESOLUTION & TEMPORAL INVALIDATION
	var resolvedEdges []*types.Edge
	var invalidatedEdges []*types.Edge

	if len(extractedEdges) > 0 {
		c.logger.Info("Starting relationship resolution",
			"episode_id", episodeNode.ID,
			"relationships_to_resolve", len(extractedEdges))

		// Resolve edge pointers using uuid map from node resolution
		utils.ResolveEdgePointers(extractedEdges, uuidMap)

		// Resolve extracted edges (dedupe + invalidation)
		resolvedEdges, invalidatedEdges, err = edgeOps.ResolveExtractedEdges(ctx,
			extractedEdges, episodeNode, resolvedNodes, options.GenerateEmbeddings)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve edges: %w", err)
		}

		c.logger.Info("Relationship resolution completed",
			"episode_id", episodeNode.ID,
			"resolved_relationships", len(resolvedEdges),
			"invalidated_relationships", len(invalidatedEdges))
	}

	// PHASE 7: ATTRIBUTE EXTRACTION
	var hydratedNodes []*types.Node
	if len(resolvedNodes) > 0 {
		c.logger.Info("Starting attribute extraction",
			"episode_id", episodeNode.ID,
			"entities_to_hydrate", len(resolvedNodes))

		hydratedNodes, err = nodeOps.ExtractAttributesFromNodes(ctx,
			resolvedNodes, episodeNode, previousEpisodes, options.EntityTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to extract attributes: %w", err)
		}

		c.logger.Info("Attribute extraction completed",
			"episode_id", episodeNode.ID,
			"hydrated_entities", len(hydratedNodes))
	}

	// PHASE 8: BUILD EPISODIC EDGES
	var episodicEdges []*types.Edge
	if len(hydratedNodes) > 0 {
		episodicEdges, err = edgeOps.BuildEpisodicEdges(ctx, hydratedNodes, episodeNode.ID, now)
		if err != nil {
			return nil, fmt.Errorf("failed to build episodic edges: %w", err)
		}

		// Store entity edge UUIDs on episode node
		entityEdgeUUIDs := make([]string, 0, len(resolvedEdges)+len(invalidatedEdges))
		for _, edge := range resolvedEdges {
			entityEdgeUUIDs = append(entityEdgeUUIDs, edge.ID)
		}
		for _, edge := range invalidatedEdges {
			entityEdgeUUIDs = append(entityEdgeUUIDs, edge.ID)
		}
		if episodeNode.Metadata == nil {
			episodeNode.Metadata = make(map[string]interface{})
		}
		episodeNode.Metadata["entity_edges"] = entityEdgeUUIDs
	}

	// PHASE 9: BULK PERSISTENCE
	allEdges := append(resolvedEdges, invalidatedEdges...)

	// Use bulk operations for efficiency
	_, err = utils.AddNodesAndEdgesBulk(ctx, c.driver,
		[]*types.Node{episodeNode},
		episodicEdges,
		hydratedNodes,
		allEdges,
		c.embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk persist data: %w", err)
	}

	result.Nodes = hydratedNodes
	result.Edges = allEdges
	result.EpisodicEdges = episodicEdges

	// PHASE 10: COMMUNITY UPDATE
	if options.UpdateCommunities {
		c.logger.Info("Starting community update",
			"episode_id", episodeNode.ID,
			"group_id", episode.GroupID)

		communityResult, err := c.community.BuildCommunities(ctx, []string{episode.GroupID})
		if err != nil {
			return nil, fmt.Errorf("failed to build communities: %w", err)
		}
		result.Communities = communityResult.CommunityNodes
		result.CommunityEdges = communityResult.CommunityEdges

		c.logger.Info("Community update completed",
			"episode_id", episodeNode.ID,
			"communities", len(result.Communities),
			"community_edges", len(result.CommunityEdges))
	}

	// Final summary log
	c.logger.Info("Episode processing completed",
		"episode_id", episodeNode.ID,
		"group_id", episode.GroupID,
		"total_entities", len(result.Nodes),
		"total_relationships", len(result.Edges),
		"episodic_edges", len(result.EpisodicEdges),
		"communities", len(result.Communities))

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

// GetEpisodes retrieves recent episodes from the knowledge graph.
func (c *Client) GetEpisodes(ctx context.Context, groupID string, limit int) ([]*types.Node, error) {
	if groupID == "" {
		groupID = c.config.GroupID
	}

	if limit <= 0 {
		limit = 10
	}

	// Use driver's SearchNodes with episodic node type filter
	searchOptions := &driver.SearchOptions{
		Limit:     limit,
		NodeTypes: []types.NodeType{types.EpisodicNodeType},
	}

	return c.driver.SearchNodes(ctx, "", groupID, searchOptions)
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
	nodes, uuidMap, _, err := nodeOps.ResolveExtractedNodes(ctx, []*types.Node{sourceNode, targetNode}, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve extracted nodes: %w", err)
	}

	// Step 4: Update edge pointers to resolved node UUIDs (line 1036)
	utils.ResolveEdgePointers([]*types.Edge{edge}, uuidMap)
	updatedEdge := edge // The edge is updated in-place

	// Step 5: Get existing edges between nodes (lines 1038-1040)
	edgeOps := maintenance.NewEdgeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())
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

	// The Go implementation wraps the private resolveExtractedEdge method
	// We'll use ResolveExtractedEdges which internally calls the same logic
	resolvedEdges, invalidatedEdges, err := edgeOps.ResolveExtractedEdges(ctx, []*types.Edge{extractedEdge}, episode, []*types.Node{}, createEmbeddings)
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
