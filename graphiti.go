package graphiti

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	jsonrepair "github.com/RealAlexandreAI/json-repair"
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
	Add(ctx context.Context, episodes []types.Episode) (*types.AddBulkEpisodeResults, error)

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
	AddTriplet(ctx context.Context, sourceNode *types.Node, edge *types.Edge, targetNode *types.Node) (*types.AddTripletResults, error)

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
	EdgeTypeMap map[string][]string
	// OverwriteExisting whether to overwrite an existing episode with the same UUID
	// Default behavior is false (skip if exists)
	OverwriteExisting  bool
	GenerateEmbeddings bool
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
	communityBuilder := community.NewBuilder(driver, llmClient, embedderClient)

	return &Client{
		driver:    driver,
		llm:       llmClient,
		embedder:  embedderClient,
		searcher:  searcher,
		community: communityBuilder,
		config:    config,
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
func (c *Client) AddEpisode(ctx context.Context, episode types.Episode, options *AddEpisodeOptions) (*types.AddEpisodeResults, error) {
	if options == nil {
		options = &AddEpisodeOptions{}
	}

	// Check if episode with same UUID already exists
	existingNode, err := c.driver.GetNode(ctx, episode.ID, episode.GroupID)
	if err == nil && existingNode != nil {
		// Episode exists - check overwrite option
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

	result := &types.AddEpisodeResults{
		EpisodicEdges:  []*types.Edge{},
		Nodes:          []*types.Node{},
		Edges:          []*types.Edge{},
		Communities:    []*types.Node{},
		CommunityEdges: []*types.Edge{},
	}

	// 1. Create episode node in graph
	episodeNode, err := c.createEpisodeNode(ctx, episode, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create episode node: %w", err)
	}
	result.Episode = episodeNode

	// 2. Extract entities from episode content if LLM is available
	var extractedNodes []*types.Node
	if c.llm != nil {
		extractedNodes, err = c.extractEntities(ctx, episode)
		if err != nil {
			return nil, fmt.Errorf("failed to extract entities: %w", err)
		}
	}

	// 3. Deduplicate and store nodes (with embedding generation if enabled)
	finalNodes, err := c.deduplicateAndStoreNodes(ctx, extractedNodes, episode.GroupID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to deduplicate and store nodes: %w", err)
	}
	result.Nodes = finalNodes

	// 4. Extract relationships between entities if LLM is available
	var extractedEdges []*types.Edge
	if c.llm != nil && len(finalNodes) > 1 {
		extractedEdges, err = c.extractRelationships(ctx, episode, finalNodes)
		if err != nil {
			return nil, fmt.Errorf("failed to extract relationships: %w", err)
		}
	}

	// 5. Store edges in graph (with fact embedding generation if enabled)
	for _, edge := range extractedEdges {
		// Generate fact_embedding if GenerateEmbeddings is enabled
		if options.GenerateEmbeddings && c.embedder != nil && edge.Fact != "" && len(edge.FactEmbedding) == 0 {
			factEmbedding, err := c.embedder.EmbedSingle(ctx, edge.Fact)
			if err != nil {
				return nil, fmt.Errorf("failed to generate fact embedding for edge %s: %w", edge.BaseEdge.ID, err)
			}
			edge.FactEmbedding = factEmbedding
		}

		if err := c.driver.UpsertEdge(ctx, edge); err != nil {
			return nil, fmt.Errorf("failed to store edge %s: %w", edge.BaseEdge.ID, err)
		}
	}
	result.Edges = extractedEdges

	// 6. Create episodic edges connecting episode to extracted entities
	for _, node := range finalNodes {
		episodeEdge := types.NewEntityEdge(
			generateID(),
			episodeNode.ID,
			node.ID,
			episode.GroupID,
			"MENTIONED_IN",
			types.EpisodicEdgeType,
		)
		episodeEdge.UpdatedAt = time.Now()
		episodeEdge.ValidFrom = episode.Reference
		episodeEdge.Summary = "Entity mentioned in episode"

		if err := c.driver.UpsertEdge(ctx, episodeEdge); err != nil {
			return nil, fmt.Errorf("failed to create episodic edge: %w", err)
		}
		result.EpisodicEdges = append(result.EpisodicEdges, episodeEdge)
	}

	// 7. Update communities if requested
	if options.UpdateCommunities {
		// Build communities for the current group using the community builder
		communityResult, err := c.community.BuildCommunities(ctx, []string{episode.GroupID})
		if err != nil {
			return nil, fmt.Errorf("failed to build communities: %w", err)
		}

		// Add the community results to the episode results
		result.Communities = communityResult.CommunityNodes
		result.CommunityEdges = communityResult.CommunityEdges
	}

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
	responseContent, _ = jsonrepair.RepairJSON(responseContent)

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

// deduplicateAndStoreNodes deduplicates nodes against existing nodes and stores them.
func (c *Client) deduplicateAndStoreNodes(ctx context.Context, nodes []*types.Node, groupID string, options *AddEpisodeOptions) ([]*types.Node, error) {
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

			// Generate name_embedding if GenerateEmbeddings is enabled and embedder is available
			if options != nil && options.GenerateEmbeddings && c.embedder != nil && node.Name != "" && len(node.NameEmbedding) == 0 {
				nameEmbedding, err := c.embedder.EmbedSingle(ctx, node.Name)
				if err != nil {
					return nil, fmt.Errorf("failed to generate name embedding for node %s: %w", node.Name, err)
				}
				node.NameEmbedding = nameEmbedding
			}

			// Create summary embedding if embedder available (existing behavior)
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
	if c.llm == nil || len(nodes) < 2 {
		// If no LLM client is available or insufficient nodes, return empty slice
		return []*types.Edge{}, nil
	}

	// 1. Use the prompt library to generate relationship extraction prompts
	library := prompts.DefaultLibrary
	extractEdgesPrompt := library.ExtractEdges()

	// Get previous episodes for context (simplified - in full implementation would get recent episodes)
	previousEpisodes := []string{} // Would be populated with recent episode content

	// Default edge types context (simplified - would be configurable)
	edgeTypesContext := []map[string]interface{}{} // Could be populated with custom edge types

	// Prepare nodes context with IDs for reference
	nodesContext := make([]map[string]interface{}, len(nodes))
	for idx, node := range nodes {
		nodesContext[idx] = map[string]interface{}{
			"id":           idx,
			"name":         node.Name,
			"entity_types": []string{node.EntityType},
		}
	}

	// Prepare context for the prompt
	context := map[string]interface{}{
		"episode_content":   episode.Content,
		"nodes":             nodesContext,
		"previous_episodes": previousEpisodes,
		"reference_time":    episode.Reference,
		"edge_types":        edgeTypesContext,
		"custom_prompt":     "",
		"ensure_ascii":      false,
	}

	// 2. Call the LLM to extract relationships
	messages, err := extractEdgesPrompt.Edge().Call(context)
	if err != nil {
		return nil, fmt.Errorf("failed to generate edge extraction prompt: %w", err)
	}

	// Generate LLM response
	response, err := c.llm.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response for edge extraction: %w", err)
	}
	// fmt.Printf("extractRelationships response.Content: %v\n", response.Content)
	// 3. Parse the LLM response into Edge structures
	edges, err := c.parseRelationshipsFromResponse(response.Content, episode, nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse relationships from LLM response: %w", err)
	}

	return edges, nil
}

// ExtractedEdge represents an edge extracted by the LLM (matching Python model)
type ExtractedEdge struct {
	RelationType   string `json:"relation_type"`
	SourceEntityID int    `json:"source_entity_id"`
	TargetEntityID int    `json:"target_entity_id"`
	Fact           string `json:"fact"`
	ValidAt        string `json:"valid_at,omitempty"`
	InvalidAt      string `json:"invalid_at,omitempty"`
	// Alternative field names that some LLMs might use
	SubjectID int    `json:"subject_id"`
	ObjectID  int    `json:"object_id"`
	FactText  string `json:"fact_text"`
}

// ExtractedEdgesResponse represents the response from edge extraction (matching Python model)
type ExtractedEdgesResponse struct {
	Edges []ExtractedEdge `json:"edges"`
}

// parseRelationshipsFromResponse parses the LLM response and converts it to Edge structures
func (c *Client) parseRelationshipsFromResponse(responseContent string, episode types.Episode, nodes []*types.Node) ([]*types.Edge, error) {
	// 1. Parse the structured JSON response from the LLM
	var extractedEdges ExtractedEdgesResponse
	responseContent, _ = jsonrepair.RepairJSON(responseContent)

	// First try to parse as object with "edges" field
	if err := json.Unmarshal([]byte(responseContent), &extractedEdges); err != nil {
		// If that fails, try to parse as direct array
		var edgeArray []ExtractedEdge
		if err := json.Unmarshal([]byte(responseContent), &edgeArray); err != nil {
			// If JSON parsing fails, try to extract JSON from response
			jsonStart := strings.Index(responseContent, "[")
			jsonEnd := strings.LastIndex(responseContent, "]")

			if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
				jsonContent := responseContent[jsonStart : jsonEnd+1]
				if err := json.Unmarshal([]byte(jsonContent), &edgeArray); err != nil {
					// Try object format extraction
					jsonStart = strings.Index(responseContent, "{")
					jsonEnd = strings.LastIndex(responseContent, "}")
					if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
						jsonContent = responseContent[jsonStart : jsonEnd+1]
						if err := json.Unmarshal([]byte(jsonContent), &extractedEdges); err != nil {
							// If still fails, fall back to simple text parsing
							return c.parseRelationshipsFromText(responseContent, episode, nodes)
						}
					} else {
						// Fall back to simple text parsing
						return c.parseRelationshipsFromText(responseContent, episode, nodes)
					}
				} else {
					extractedEdges.Edges = edgeArray
				}
			} else {
				// Fall back to simple text parsing
				return c.parseRelationshipsFromText(responseContent, episode, nodes)
			}
		} else {
			extractedEdges.Edges = edgeArray
		}
	}

	// Normalize field values for alternative field names
	for i := range extractedEdges.Edges {
		edge := &extractedEdges.Edges[i]
		if edge.SourceEntityID == 0 && edge.SubjectID != 0 {
			edge.SourceEntityID = edge.SubjectID
		}
		if edge.TargetEntityID == 0 && edge.ObjectID != 0 {
			edge.TargetEntityID = edge.ObjectID
		}
		if edge.Fact == "" && edge.FactText != "" {
			edge.Fact = edge.FactText
		}
	}

	// 2. Handle the ExtractedEdges response model
	edges := make([]*types.Edge, 0, len(extractedEdges.Edges))
	now := time.Now()

	// 3. Create proper EntityEdge objects with all attributes
	for _, extractedEdge := range extractedEdges.Edges {
		// Validate node indices
		if extractedEdge.SourceEntityID < 0 || extractedEdge.SourceEntityID >= len(nodes) ||
			extractedEdge.TargetEntityID < 0 || extractedEdge.TargetEntityID >= len(nodes) {
			continue // Skip invalid node references
		}

		// Skip self-referential edges
		if extractedEdge.SourceEntityID == extractedEdge.TargetEntityID {
			continue
		}

		sourceNode := nodes[extractedEdge.SourceEntityID]
		targetNode := nodes[extractedEdge.TargetEntityID]

		// Parse temporal information
		var validFrom, validTo *time.Time
		if extractedEdge.ValidAt != "" {
			if t, err := time.Parse(time.RFC3339, extractedEdge.ValidAt); err == nil {
				validFrom = &t
			}
		}
		if extractedEdge.InvalidAt != "" {
			if t, err := time.Parse(time.RFC3339, extractedEdge.InvalidAt); err == nil {
				validTo = &t
			}
		}

		edge := types.NewEntityEdge(
			generateID(),
			sourceNode.ID,
			targetNode.ID,
			episode.GroupID,
			extractedEdge.RelationType,
			types.EntityEdgeType,
		)
		edge.CreatedAt = now
		edge.UpdatedAt = now
		edge.Fact = extractedEdge.Fact
		edge.ValidFrom = now
		edge.ValidTo = validTo
		edge.Episodes = []string{episode.ID}
		edge.Metadata = make(map[string]interface{})

		// Add extracted metadata
		edge.Metadata["fact"] = extractedEdge.Fact
		edge.Metadata["source_entity_name"] = sourceNode.Name
		edge.Metadata["target_entity_name"] = targetNode.Name
		if validFrom != nil {
			edge.Metadata["valid_at"] = extractedEdge.ValidAt
		}
		if extractedEdge.InvalidAt != "" {
			edge.Metadata["invalid_at"] = extractedEdge.InvalidAt
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// parseRelationshipsFromText provides fallback text-based parsing when JSON parsing fails
func (c *Client) parseRelationshipsFromText(responseContent string, episode types.Episode, nodes []*types.Node) ([]*types.Edge, error) {
	edges := []*types.Edge{}
	now := time.Now()

	// Create a map of node names to indices for lookup
	nodeNameToIndex := make(map[string]int)
	for i, node := range nodes {
		nodeNameToIndex[strings.ToLower(node.Name)] = i
	}

	// Simple text-based extraction as fallback
	lines := strings.Split(responseContent, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for relationship patterns in various formats
		if strings.Contains(line, "relation") || strings.Contains(line, "fact") ||
			strings.Contains(line, "->") || strings.Contains(line, "relates to") {

			// Try to extract entity references and relationship
			for sourceName, sourceIdx := range nodeNameToIndex {
				for targetName, targetIdx := range nodeNameToIndex {
					if sourceIdx != targetIdx &&
						strings.Contains(strings.ToLower(line), sourceName) &&
						strings.Contains(strings.ToLower(line), targetName) {

						// Create a basic relationship
						edge := types.NewEntityEdge(
							generateID(),
							nodes[sourceIdx].ID,
							nodes[targetIdx].ID,
							episode.GroupID,
							"RELATED_TO", // Default relationship type
							types.EntityEdgeType,
						)
						edge.CreatedAt = now
						edge.UpdatedAt = now
						edge.Fact = line
						edge.ValidFrom = now
						edge.Episodes = []string{episode.ID}
						edge.Metadata = make(map[string]interface{})

						edge.Metadata["fact"] = line
						edge.Metadata["source_entity_name"] = nodes[sourceIdx].Name
						edge.Metadata["target_entity_name"] = nodes[targetIdx].Name
						edge.Metadata["extraction_method"] = "text_fallback"

						edges = append(edges, edge)
						break
					}
				}
			}
		}
	}

	return edges, nil
}

// findNodeByName searches for an existing node by name and entity type.
// This function implements node deduplication logic similar to Python's resolve_extracted_nodes.
func (c *Client) findNodeByName(ctx context.Context, name, entityType, groupID string) (*types.Node, error) {
	if c.driver == nil {
		return nil, nil
	}

	// Normalize search query
	searchQuery := strings.TrimSpace(name)
	if searchQuery == "" {
		return nil, nil
	}

	// Strategy 1: Try exact text search first
	exactMatches, err := c.searchNodesByExactName(ctx, searchQuery, entityType, groupID)
	if err == nil && len(exactMatches) > 0 {
		return c.selectBestMatch(exactMatches, searchQuery, entityType), nil
	}

	// Strategy 2: Try fuzzy/fulltext search
	searchOptions := &driver.SearchOptions{
		Limit:       20, // Search for more candidates for better matching
		UseFullText: true,
		NodeTypes:   []types.NodeType{types.EntityNodeType}, // Only search entity nodes
	}

	candidateNodes, err := c.driver.SearchNodes(ctx, searchQuery, groupID, searchOptions)
	if err != nil {
		// If search fails, it's not an error - just no nodes found
		return nil, nil
	}

	if len(candidateNodes) == 0 {
		return nil, nil
	}

	// Strategy 3: Use embedding similarity search if embedder is available
	if c.embedder != nil {
		embeddingMatches, err := c.searchNodesByEmbedding(ctx, searchQuery, groupID)
		if err == nil && len(embeddingMatches) > 0 {
			// Combine text and embedding results
			candidateNodes = c.combineSearchResults(candidateNodes, embeddingMatches)
		}
	}

	// Find the best match using similarity scoring
	return c.selectBestMatch(candidateNodes, searchQuery, entityType), nil
}

// searchNodesByExactName performs exact name matching
func (c *Client) searchNodesByExactName(ctx context.Context, name, entityType, groupID string) ([]*types.Node, error) {
	// Try to use driver's direct search if available
	searchOptions := &driver.SearchOptions{
		Limit:       5,
		UseFullText: false, // Exact matching
		NodeTypes:   []types.NodeType{types.EntityNodeType},
	}

	// Search with exact query
	nodes, err := c.driver.SearchNodes(ctx, fmt.Sprintf("\"%s\"", name), groupID, searchOptions)
	if err != nil {
		return nil, err
	}

	// Filter for exact name matches
	var exactMatches []*types.Node
	for _, node := range nodes {
		if node != nil && strings.EqualFold(node.Name, name) {
			// Additional entity type filtering if specified
			if entityType == "" || node.EntityType == entityType {
				exactMatches = append(exactMatches, node)
			}
		}
	}

	return exactMatches, nil
}

// searchNodesByEmbedding performs semantic similarity search
func (c *Client) searchNodesByEmbedding(ctx context.Context, query, groupID string) ([]*types.Node, error) {
	if c.embedder == nil {
		return nil, nil
	}

	// Generate embedding for the query
	embedding, err := c.embedder.EmbedSingle(ctx, query)
	if err != nil {
		return nil, err
	}

	// Search using embedding similarity
	return c.driver.SearchNodesByEmbedding(ctx, embedding, groupID, 10)
}

// combineSearchResults merges and deduplicates search results from multiple strategies
func (c *Client) combineSearchResults(textResults, embeddingResults []*types.Node) []*types.Node {
	seen := make(map[string]*types.Node)
	var combined []*types.Node

	// Add text results first (higher priority)
	for _, node := range textResults {
		if node != nil {
			seen[node.ID] = node
			combined = append(combined, node)
		}
	}

	// Add embedding results if not already seen
	for _, node := range embeddingResults {
		if node != nil {
			if _, exists := seen[node.ID]; !exists {
				seen[node.ID] = node
				combined = append(combined, node)
			}
		}
	}

	return combined
}

// selectBestMatch chooses the best matching node from candidates
func (c *Client) selectBestMatch(candidates []*types.Node, searchName, entityType string) *types.Node {
	if len(candidates) == 0 {
		return nil
	}

	var exactMatch *types.Node
	var bestMatch *types.Node
	var bestScore float64

	searchNameLower := strings.ToLower(searchName)

	for _, node := range candidates {
		if node == nil {
			continue
		}

		nodeNameLower := strings.ToLower(node.Name)

		// Calculate similarity score
		score := c.calculateNameSimilarity(searchNameLower, nodeNameLower)

		// Check for exact name match
		if strings.EqualFold(node.Name, searchName) {
			// Check entity type compatibility
			if entityType == "" || node.EntityType == entityType || node.EntityType == "" {
				if node.EntityType == entityType {
					// Perfect match: exact name and entity type
					return node
				} else if exactMatch == nil {
					// Exact name but different/missing entity type
					exactMatch = node
				}
			}
		}

		// Track best scoring match
		if score > bestScore {
			bestScore = score
			bestMatch = node
		}
	}

	// Return exact match if found, otherwise best scoring match
	if exactMatch != nil {
		return exactMatch
	}

	// Only return best match if score is above threshold
	if bestScore > 0.8 { // 80% similarity threshold
		return bestMatch
	}

	return nil
}

// calculateNameSimilarity computes similarity between two names
func (c *Client) calculateNameSimilarity(name1, name2 string) float64 {
	// Simple similarity calculation - could be enhanced with more sophisticated algorithms
	if name1 == name2 {
		return 1.0
	}

	// Check if one name contains the other
	if strings.Contains(name1, name2) || strings.Contains(name2, name1) {
		shorter := min(len(name1), len(name2))
		longer := max(len(name1), len(name2))
		return float64(shorter) / float64(longer)
	}

	// Check common prefix/suffix
	commonPrefix := 0
	minLen := min(len(name1), len(name2))
	for i := 0; i < minLen; i++ {
		if name1[i] == name2[i] {
			commonPrefix++
		} else {
			break
		}
	}

	if commonPrefix > 0 {
		return float64(commonPrefix) / float64(max(len(name1), len(name2)))
	}

	return 0.0
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
	} else {
		searchConfig.NodeConfig = &search.NodeSearchConfig{
			SearchMethods: []search.SearchMethod{search.CosineSimilarity},
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
			SearchMethods: []search.SearchMethod{search.CosineSimilarity},
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
func (c *Client) AddTriplet(ctx context.Context, sourceNode *types.Node, edge *types.Edge, targetNode *types.Node) (*types.AddTripletResults, error) {
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
	resolvedEdge, invalidatedEdges, err := c.resolveExtractedEdgeExact(ctx, updatedEdge, relatedEdges, existingEdges, episodicNode)
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
func (c *Client) resolveExtractedEdgeExact(ctx context.Context, extractedEdge *types.Edge, relatedEdges []*types.Edge, existingEdges []*types.Edge, episode *types.Node) (*types.Edge, []*types.Edge, error) {
	// Use the EdgeOperations to resolve the edge exactly as in Python
	edgeOps := maintenance.NewEdgeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())

	// The Go implementation wraps the private resolveExtractedEdge method
	// We'll use ResolveExtractedEdges which internally calls the same logic
	resolvedEdges, invalidatedEdges, err := edgeOps.ResolveExtractedEdges(ctx, []*types.Edge{extractedEdge}, episode, []*types.Node{})
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
