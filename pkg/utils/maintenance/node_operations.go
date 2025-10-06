package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	jsonrepair "github.com/RealAlexandreAI/json-repair"
	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/prompts"
	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/soundprediction/go-graphiti/pkg/utils"
)

// NodeOperations provides node-related maintenance operations
type NodeOperations struct {
	driver   driver.GraphDriver
	llm      llm.Client
	embedder embedder.Client
	prompts  prompts.Library
}

// NewNodeOperations creates a new NodeOperations instance
func NewNodeOperations(driver driver.GraphDriver, llm llm.Client, embedder embedder.Client, prompts prompts.Library) *NodeOperations {
	return &NodeOperations{
		driver:   driver,
		llm:      llm,
		embedder: embedder,
		prompts:  prompts,
	}
}

// ExtractNodes extracts entity nodes from episode content using LLM
func (no *NodeOperations) ExtractNodes(ctx context.Context, episode *types.Node, previousEpisodes []*types.Node, entityTypes map[string]interface{}, excludedEntityTypes []string) ([]*types.Node, error) {
	start := time.Now()

	// Prepare entity types context
	entityTypesContext := []map[string]interface{}{
		{
			"entity_type_id":          0,
			"entity_type_name":        "Entity",
			"entity_type_description": "Default classification. Use this entity type if the entity is not one of the other listed types.",
		},
	}

	if entityTypes != nil {
		id := 1
		for typeName := range entityTypes {
			entityTypesContext = append(entityTypesContext, map[string]interface{}{
				"entity_type_id":          id,
				"entity_type_name":        typeName,
				"entity_type_description": fmt.Sprintf("custom type: %s", typeName),
			})
			id++
		}
	}

	// Prepare previous episodes content
	previousEpisodeContents := make([]string, len(previousEpisodes))
	for i, ep := range previousEpisodes {
		previousEpisodeContents[i] = ep.Summary
	}

	// Convert entity types context to JSON string
	entityTypesJson, _ := json.Marshal(entityTypesContext)
	entityTypesStr := string(entityTypesJson)
	// Prepare context for LLM
	promptContext := map[string]interface{}{
		"episode_content":    episode.Content,
		"episode_timestamp":  episode.ValidFrom.Format(time.RFC3339),
		"previous_episodes":  previousEpisodeContents,
		"custom_prompt":      "",
		"entity_types":       entityTypesStr,
		"source_description": string(episode.EpisodeType),
		"ensure_ascii":       true,
	}

	// Extract entities with reflexion
	entitiesMissed := true
	reflexionIterations := 0
	maxReflexionIterations := utils.GetMaxReflexionIterations()

	var extractedEntities prompts.ExtractedEntities

	for entitiesMissed && reflexionIterations <= maxReflexionIterations {
		// Choose the appropriate extraction method based on episode source
		var messages []llm.Message
		var err error

		switch strings.ToLower(string(episode.EpisodeType)) {
		case "message":
			messages, err = no.prompts.ExtractNodes().ExtractMessage().Call(promptContext)
		case "text":
			messages, err = no.prompts.ExtractNodes().ExtractText().Call(promptContext)
		case "json":
			messages, err = no.prompts.ExtractNodes().ExtractJSON().Call(promptContext)
		default:
			messages, err = no.prompts.ExtractNodes().ExtractText().Call(promptContext)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create extraction prompt: %w", err)
		}

		response, err := no.llm.ChatWithStructuredOutput(ctx, messages, &prompts.ExtractedEntities{})
		if err != nil {
			return nil, fmt.Errorf("failed to extract entities: %w", err)
		}
		// fmt.Printf("string(response): %v\n", string(response))
		// Repair JSON before unmarshaling
		repairedResponse, _ := jsonrepair.RepairJSON(string(response))

		// Try to unmarshal - if it's a quoted JSON string, unmarshal twice
		var rawJSON json.RawMessage
		if err := json.Unmarshal([]byte(repairedResponse), &rawJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal repaired response: %w", err)
		}

		if err := json.Unmarshal(rawJSON, &extractedEntities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entities response: %w", err)
		}

		reflexionIterations++
		if reflexionIterations < maxReflexionIterations {
			// Run reflexion to check for missed entities
			missedEntities, err := no.extractNodesReflexion(ctx, episode, previousEpisodes, extractedEntities)
			if err != nil {
				log.Printf("Warning: reflexion failed: %v", err)
				break
			}

			entitiesMissed = len(missedEntities) > 0
			if entitiesMissed {
				customPrompt := "Make sure that the following entities are extracted:"
				for _, entity := range missedEntities {
					customPrompt += fmt.Sprintf("\n%s,", entity)
				}
				promptContext["custom_prompt"] = customPrompt
			}
		} else {
			entitiesMissed = false
		}
	}

	// Filter out empty entity names
	var filteredEntities []prompts.ExtractedEntity
	for _, entity := range extractedEntities.ExtractedEntities {
		if strings.TrimSpace(entity.Name) != "" {
			filteredEntities = append(filteredEntities, entity)
		}
	}

	log.Printf("Extracted %d entities in %v", len(filteredEntities), time.Since(start))

	// Convert to Node objects
	var extractedNodes []*types.Node
	for _, extractedEntity := range filteredEntities {
		// Determine entity type
		var entityTypeName string
		if extractedEntity.EntityTypeID >= 0 && extractedEntity.EntityTypeID < len(entityTypesContext) {
			entityTypeName = entityTypesContext[extractedEntity.EntityTypeID]["entity_type_name"].(string)
		} else {
			entityTypeName = "Entity"
		}

		// Check if this entity type should be excluded
		if len(excludedEntityTypes) > 0 {
			excluded := false
			for _, excludedType := range excludedEntityTypes {
				if entityTypeName == excludedType {
					excluded = true
					break
				}
			}
			if excluded {
				log.Printf("Excluding entity %s of type %s", extractedEntity.Name, entityTypeName)
				continue
			}
		}

		node := &types.Node{
			ID:         utils.GenerateUUID(),
			Type:       types.EntityNodeType,
			GroupID:    episode.GroupID,
			Name:       extractedEntity.Name,
			Summary:    extractedEntity.Name,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
			ValidFrom:  episode.ValidFrom,
			EntityType: entityTypeName,
			Metadata:   make(map[string]interface{}),
		}

		extractedNodes = append(extractedNodes, node)
		log.Printf("Created entity node: %s (UUID: %s)", node.Name, node.ID)
	}

	return extractedNodes, nil
}

// extractNodesReflexion performs reflexion to identify missed entities
func (no *NodeOperations) extractNodesReflexion(ctx context.Context, episode *types.Node, previousEpisodes []*types.Node, extractedEntities prompts.ExtractedEntities) ([]string, error) {
	// Get entity names
	var entityNames []string
	for _, entity := range extractedEntities.ExtractedEntities {
		entityNames = append(entityNames, entity.Name)
	}

	// Prepare previous episodes content
	previousEpisodeContents := make([]string, len(previousEpisodes))
	for i, ep := range previousEpisodes {
		previousEpisodeContents[i] = ep.Summary
	}

	// Prepare context for reflexion
	promptContext := map[string]interface{}{
		"episode_content":    episode.Summary,
		"previous_episodes":  previousEpisodeContents,
		"extracted_entities": entityNames,
		"ensure_ascii":       true,
	}

	messages, err := no.prompts.ExtractNodes().Reflexion().Call(promptContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create reflexion prompt: %w", err)
	}

	response, err := no.llm.ChatWithStructuredOutput(ctx, messages, &prompts.MissedEntities{})
	if err != nil {
		return nil, fmt.Errorf("failed to run reflexion: %w", err)
	}

	// Repair JSON before unmarshaling
	repairedResponse, _ := jsonrepair.RepairJSON(string(response))

	// Try to unmarshal - if it's a quoted JSON string, unmarshal twice
	var rawJSON json.RawMessage
	if err := json.Unmarshal([]byte(repairedResponse), &rawJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repaired response: %w", err)
	}

	var missedEntities prompts.MissedEntities
	if err := json.Unmarshal(rawJSON, &missedEntities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reflexion response: %w", err)
	}

	return missedEntities.MissedEntities, nil
}

// ResolveExtractedNodes resolves newly extracted nodes against existing ones in the graph
func (no *NodeOperations) ResolveExtractedNodes(ctx context.Context, extractedNodes []*types.Node, episode *types.Node, previousEpisodes []*types.Node, entityTypes map[string]interface{}) ([]*types.Node, map[string]string, []NodePair, error) {
	if len(extractedNodes) == 0 {
		return []*types.Node{}, make(map[string]string), []NodePair{}, nil
	}

	// Search for existing nodes that might be duplicates
	var candidateNodes []*types.Node
	searchResults := make(map[string][]*types.Node)

	for _, node := range extractedNodes {
		// Search for nodes with similar names
		options := &driver.SearchOptions{
			Limit:     50,
			NodeTypes: []types.NodeType{types.EntityNodeType},
		}

		nodes, err := no.driver.SearchNodes(ctx, node.Name, node.GroupID, options)
		if err != nil {
			log.Printf("Warning: failed to search for similar nodes: %v", err)
			nodes = []*types.Node{}
		}

		searchResults[node.ID] = nodes
		candidateNodes = append(candidateNodes, nodes...)
	}

	// Remove duplicates from candidate nodes
	candidateMap := make(map[string]*types.Node)
	for _, node := range candidateNodes {
		candidateMap[node.ID] = node
	}

	var existingNodes []*types.Node
	for _, node := range candidateMap {
		existingNodes = append(existingNodes, node)
	}

	// Prepare context for LLM deduplication
	extractedNodesContext := make([]map[string]interface{}, len(extractedNodes))
	for i, node := range extractedNodes {
		extractedNodesContext[i] = map[string]interface{}{
			"id":                      i,
			"name":                    node.Name,
			"entity_type":             []string{"Entity", node.EntityType},
			"entity_type_description": "Entity description", // Simplified
		}
	}

	existingNodesContext := make([]map[string]interface{}, len(existingNodes))
	for i, node := range existingNodes {
		existingNodesContext[i] = map[string]interface{}{
			"idx":          i,
			"name":         node.Name,
			"entity_types": []string{"Entity", node.EntityType},
			"summary":      node.Summary,
		}
		// Add metadata as attributes
		for k, v := range node.Metadata {
			existingNodesContext[i][k] = v
		}
	}

	// Prepare previous episodes content
	previousEpisodeContents := make([]string, len(previousEpisodes))
	for i, ep := range previousEpisodes {
		previousEpisodeContents[i] = ep.Summary
	}

	promptContext := map[string]interface{}{
		"extracted_nodes":   extractedNodesContext,
		"existing_nodes":    existingNodesContext,
		"episode_content":   episode.Content,
		"previous_episodes": previousEpisodeContents,
		"ensure_ascii":      true,
	}

	// Use LLM to resolve duplicates
	messages, err := no.prompts.DedupeNodes().Nodes().Call(promptContext)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create dedupe prompt: %w", err)
	}

	response, err := no.llm.ChatWithStructuredOutput(ctx, messages, &prompts.NodeResolutions{})

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to resolve nodes: %w", err)
	}

	// Repair JSON before unmarshaling
	repairedResponse, _ := jsonrepair.RepairJSON(string(response))

	// Try to unmarshal - if it's a quoted JSON string, unmarshal twice
	var rawJSON json.RawMessage
	if err := json.Unmarshal([]byte(repairedResponse), &rawJSON); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal repaired response: %w", err)
	}

	var nodeResolutions prompts.NodeResolutions
	if err := json.Unmarshal(rawJSON, &nodeResolutions); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal node resolutions: %w", err)
	}

	// Process the resolutions
	var resolvedNodes []*types.Node
	uuidMap := make(map[string]string)
	var nodeDuplicates []NodePair

	for _, resolution := range nodeResolutions.EntityResolutions {
		if resolution.ID < 0 || resolution.ID >= len(extractedNodes) {
			continue
		}

		extractedNode := extractedNodes[resolution.ID]
		var resolvedNode *types.Node

		// Check if it's a duplicate of an existing node
		if resolution.DuplicateIdx >= 0 && resolution.DuplicateIdx < len(existingNodes) {
			resolvedNode = existingNodes[resolution.DuplicateIdx]
		} else {
			resolvedNode = extractedNode
		}

		resolvedNodes = append(resolvedNodes, resolvedNode)
		uuidMap[extractedNode.ID] = resolvedNode.ID

		// Track duplicates for edge creation
		for _, duplicateIdx := range resolution.Duplicates {
			if duplicateIdx >= 0 && duplicateIdx < len(existingNodes) {
				existingNode := existingNodes[duplicateIdx]
				nodeDuplicates = append(nodeDuplicates, NodePair{
					Source: extractedNode,
					Target: existingNode,
				})
			}
		}
	}

	log.Printf("Resolved %d nodes, found %d duplicates", len(resolvedNodes), len(nodeDuplicates))

	// Filter duplicates using edge operations to remove those that already have IS_DUPLICATE_OF edges
	edgeOps := NewEdgeOperations(no.driver, no.llm, no.embedder, no.prompts)
	filteredDuplicates, err := edgeOps.FilterExistingDuplicateOfEdges(ctx, nodeDuplicates)
	if err != nil {
		log.Printf("Warning: failed to filter existing duplicate edges: %v", err)
		filteredDuplicates = nodeDuplicates
	}

	return resolvedNodes, uuidMap, filteredDuplicates, nil
}

// ExtractAttributesFromNodes extracts and updates attributes for nodes using LLM
func (no *NodeOperations) ExtractAttributesFromNodes(ctx context.Context, nodes []*types.Node, episode *types.Node, previousEpisodes []*types.Node, entityTypes map[string]interface{}) ([]*types.Node, error) {
	var updatedNodes []*types.Node

	for _, node := range nodes {
		updatedNode, err := no.extractAttributesFromNode(ctx, node, episode, previousEpisodes, entityTypes)
		if err != nil {
			log.Printf("Warning: failed to extract attributes for node %s: %v", node.Name, err)
			updatedNodes = append(updatedNodes, node) // Use original node if extraction fails
		} else {
			updatedNodes = append(updatedNodes, updatedNode)
		}
	}

	// Create embeddings for all updated nodes
	for _, node := range updatedNodes {
		if err := no.createNodeEmbedding(ctx, node); err != nil {
			log.Printf("Warning: failed to create embedding for node %s: %v", node.Name, err)
		}
	}

	return updatedNodes, nil
}

// extractAttributesFromNode extracts attributes and summary for a single node
func (no *NodeOperations) extractAttributesFromNode(ctx context.Context, node *types.Node, episode *types.Node, previousEpisodes []*types.Node, entityTypes map[string]interface{}) (*types.Node, error) {
	// Prepare node context
	nodeContext := map[string]interface{}{
		"name":         node.Name,
		"summary":      node.Summary,
		"entity_types": []string{"Entity", node.EntityType},
		"attributes":   node.Metadata,
	}

	// Prepare previous episodes content
	previousEpisodeContents := make([]string, len(previousEpisodes))
	for i, ep := range previousEpisodes {
		previousEpisodeContents[i] = ep.Summary
	}

	// Extract summary
	summaryContext := map[string]interface{}{
		"node":              nodeContext,
		"episode_content":   episode.Summary,
		"previous_episodes": previousEpisodeContents,
		"ensure_ascii":      true,
	}

	summaryMessages, err := no.prompts.ExtractNodes().ExtractSummary().Call(summaryContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create summary prompt: %w", err)
	}

	summaryResponse, err := no.llm.ChatWithStructuredOutput(ctx, summaryMessages, &prompts.EntitySummary{})
	if err != nil {
		return nil, fmt.Errorf("failed to extract summary: %w", err)
	}

	var entitySummary prompts.EntitySummary
	if err := json.Unmarshal(summaryResponse, &entitySummary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal summary response: %w", err)
	}

	// Update node with new summary
	updatedNode := *node // Copy the node
	updatedNode.Summary = entitySummary.Summary
	updatedNode.UpdatedAt = time.Now().UTC()

	// Extract attributes if entity type is defined
	if entityTypes != nil {
		// Find the entity type for this node
		entityTypeName := node.EntityType
		if entityTypeName == "" {
			entityTypeName = "Entity"
		}

		if entityTypeName != "" && entityTypes[entityTypeName] != nil {
			attributesContext := map[string]interface{}{
				"node":              nodeContext,
				"episode_content":   episode.Content,
				"previous_episodes": previousEpisodeContents,
				"ensure_ascii":      true,
			}

			attributesMessages, err := no.prompts.ExtractNodes().ExtractAttributes().Call(attributesContext)
			if err != nil {
				log.Printf("Warning: failed to create attributes prompt: %v", err)
			} else {
				// For now, we'll use a generic map for attributes since we don't have the specific type
				attributesResponse, err := no.llm.Chat(ctx, attributesMessages)
				if err != nil {
					log.Printf("Warning: failed to extract attributes: %v", err)
				} else {
					// Parse the response as a simple string and store in metadata
					if updatedNode.Metadata == nil {
						updatedNode.Metadata = make(map[string]interface{})
					}
					updatedNode.Metadata["llm_attributes"] = attributesResponse.Content
				}
			}
		}
	}

	return &updatedNode, nil
}

// createNodeEmbedding creates an embedding for a node based on its name and summary
func (no *NodeOperations) createNodeEmbedding(ctx context.Context, node *types.Node) error {
	// Create text for embedding from name and summary
	text := node.Name
	if node.Summary != "" {
		text += " " + node.Summary
	}

	embedding, err := no.embedder.EmbedSingle(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	node.Embedding = embedding
	return nil
}
