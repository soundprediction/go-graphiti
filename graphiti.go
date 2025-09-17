package graphiti

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/prompts"
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

	// GetEpisodes retrieves recent episodes from the knowledge graph.
	GetEpisodes(ctx context.Context, groupID string, limit int) ([]*types.Node, error)

	// ClearGraph removes all nodes and edges from the knowledge graph for a specific group.
	ClearGraph(ctx context.Context, groupID string) error

	// CreateIndices creates database indices and constraints for optimal performance.
	CreateIndices(ctx context.Context) error

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
		"episode_content":    episode.Content,
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

	// 3. Parse the LLM response into Node structures
	entities, err := c.parseEntitiesFromResponse(response.Content, episode.GroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse entities from LLM response: %w", err)
	}

	return entities, nil
}

// ExtractedEntity represents an entity extracted by the LLM (matching Python model)
type ExtractedEntity struct {
	Name         string `json:"name"`
	EntityTypeID int    `json:"entity_type_id"`
}

// ExtractedEntities represents the response from entity extraction (matching Python model)
type ExtractedEntities struct {
	ExtractedEntities []ExtractedEntity `json:"extracted_entities"`
}

// parseEntitiesFromResponse parses the LLM response and converts it to Node structures
func (c *Client) parseEntitiesFromResponse(responseContent, groupID string) ([]*types.Node, error) {
	// 1. Parse the structured JSON response from the LLM
	var extractedEntities ExtractedEntities
	if err := json.Unmarshal([]byte(responseContent), &extractedEntities); err != nil {
		// If JSON parsing fails, try to extract JSON from response
		// Sometimes LLM responses include extra text around the JSON
		jsonStart := strings.Index(responseContent, "{")
		jsonEnd := strings.LastIndex(responseContent, "}")

		if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
			jsonContent := responseContent[jsonStart : jsonEnd+1]
			if err := json.Unmarshal([]byte(jsonContent), &extractedEntities); err != nil {
				// If still fails, fall back to simple text parsing
				return c.parseEntitiesFromText(responseContent, groupID)
			}
		} else {
			// Fall back to simple text parsing
			return c.parseEntitiesFromText(responseContent, groupID)
		}
	}

	// 2. Handle the ExtractedEntities response model
	entities := make([]*types.Node, 0, len(extractedEntities.ExtractedEntities))
	now := time.Now()

	// Default entity types (matching Python implementation)
	entityTypes := map[int]string{
		0: "Entity", // Default entity type
	}

	// 3. Create proper EntityNode objects with all attributes
	for _, extractedEntity := range extractedEntities.ExtractedEntities {
		// Skip empty names
		if strings.TrimSpace(extractedEntity.Name) == "" {
			continue
		}

		// Determine entity type from ID
		entityType := "Entity" // Default
		if entityTypeName, exists := entityTypes[extractedEntity.EntityTypeID]; exists {
			entityType = entityTypeName
		}

		entity := &types.Node{
			ID:         generateID(),
			Name:       strings.TrimSpace(extractedEntity.Name),
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
		"episode_content":    episode.Content,
		"nodes":              nodesContext,
		"previous_episodes":  previousEpisodes,
		"reference_time":     episode.Reference,
		"edge_types":         edgeTypesContext,
		"custom_prompt":      "",
		"ensure_ascii":       false,
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

	// 3. Parse the LLM response into Edge structures
	edges, err := c.parseRelationshipsFromResponse(response.Content, episode, nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse relationships from LLM response: %w", err)
	}

	return edges, nil
}

// ExtractedEdge represents an edge extracted by the LLM (matching Python model)
type ExtractedEdge struct {
	RelationType     string `json:"relation_type"`
	SourceEntityID   int    `json:"source_entity_id"`
	TargetEntityID   int    `json:"target_entity_id"`
	Fact             string `json:"fact"`
	ValidAt          string `json:"valid_at,omitempty"`
	InvalidAt        string `json:"invalid_at,omitempty"`
}

// ExtractedEdgesResponse represents the response from edge extraction (matching Python model)
type ExtractedEdgesResponse struct {
	Edges []ExtractedEdge `json:"edges"`
}

// parseRelationshipsFromResponse parses the LLM response and converts it to Edge structures
func (c *Client) parseRelationshipsFromResponse(responseContent string, episode types.Episode, nodes []*types.Node) ([]*types.Edge, error) {
	// 1. Parse the structured JSON response from the LLM
	var extractedEdges ExtractedEdgesResponse
	if err := json.Unmarshal([]byte(responseContent), &extractedEdges); err != nil {
		// If JSON parsing fails, try to extract JSON from response
		jsonStart := strings.Index(responseContent, "{")
		jsonEnd := strings.LastIndex(responseContent, "}")

		if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
			jsonContent := responseContent[jsonStart : jsonEnd+1]
			if err := json.Unmarshal([]byte(jsonContent), &extractedEdges); err != nil {
				// If still fails, fall back to simple text parsing
				return c.parseRelationshipsFromText(responseContent, episode, nodes)
			}
		} else {
			// Fall back to simple text parsing
			return c.parseRelationshipsFromText(responseContent, episode, nodes)
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

		edge := &types.Edge{
			ID:           generateID(),
			Type:         types.EntityEdgeType,
			SourceID:     sourceNode.ID,
			TargetID:     targetNode.ID,
			GroupID:      episode.GroupID,
			CreatedAt:    now,
			UpdatedAt:    now,
			Name:         extractedEdge.RelationType,
			Summary:      extractedEdge.Fact,
			ValidFrom:    now,
			ValidTo:      validTo,
			Metadata:     make(map[string]interface{}),
			SourceIDs:    []string{episode.ID},
		}

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
						edge := &types.Edge{
							ID:        generateID(),
							Type:      types.EntityEdgeType,
							SourceID:  nodes[sourceIdx].ID,
							TargetID:  nodes[targetIdx].ID,
							GroupID:   episode.GroupID,
							CreatedAt: now,
							UpdatedAt: now,
							Name:      "RELATED_TO", // Default relationship type
							Summary:   line,
							ValidFrom: now,
							Metadata:  make(map[string]interface{}),
							SourceIDs: []string{episode.ID},
						}

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
