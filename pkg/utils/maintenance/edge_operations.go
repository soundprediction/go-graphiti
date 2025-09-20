package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/prompts"
	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/soundprediction/go-graphiti/pkg/utils"
)

// EdgeOperations provides edge-related maintenance operations
type EdgeOperations struct {
	driver   driver.GraphDriver
	llm      llm.Client
	embedder embedder.Client
	prompts  prompts.Library
}

// NewEdgeOperations creates a new EdgeOperations instance
func NewEdgeOperations(driver driver.GraphDriver, llm llm.Client, embedder embedder.Client, prompts prompts.Library) *EdgeOperations {
	return &EdgeOperations{
		driver:   driver,
		llm:      llm,
		embedder: embedder,
		prompts:  prompts,
	}
}

// BuildEpisodicEdges creates episodic edges from entity nodes to an episode
func (eo *EdgeOperations) BuildEpisodicEdges(ctx context.Context, entityNodes []*types.Node, episodeUUID string, createdAt time.Time) ([]*types.Edge, error) {
	if len(entityNodes) == 0 {
		return []*types.Edge{}, nil
	}

	episodicEdges := make([]*types.Edge, 0, len(entityNodes))

	for _, node := range entityNodes {
		edge := types.NewEntityEdge(
			utils.GenerateUUID(),
			episodeUUID,
			node.ID,
			node.GroupID,
			"MENTIONED_IN",
			types.EpisodicEdgeType,
		)
		edge.UpdatedAt = createdAt
		edge.ValidFrom = createdAt
		episodicEdges = append(episodicEdges, edge)
	}

	log.Printf("Built %d episodic edges", len(episodicEdges))
	return episodicEdges, nil
}

// BuildDuplicateOfEdges creates IS_DUPLICATE_OF edges between duplicate node pairs
func (eo *EdgeOperations) BuildDuplicateOfEdges(ctx context.Context, episode *types.Node, createdAt time.Time, duplicateNodes []NodePair) ([]*types.Edge, error) {
	duplicateEdges := make([]*types.Edge, 0, len(duplicateNodes))

	for _, pair := range duplicateNodes {
		if pair.Source.ID == pair.Target.ID {
			continue
		}

		fact := fmt.Sprintf("%s is a duplicate of %s", pair.Source.Name, pair.Target.Name)

		edge := types.NewEntityEdge(
			utils.GenerateUUID(),
			pair.Source.ID,
			pair.Target.ID,
			episode.GroupID,
			"IS_DUPLICATE_OF",
			types.EntityEdgeType,
		)
		edge.Summary = fact
		edge.Fact = fact
		edge.UpdatedAt = createdAt
		edge.ValidFrom = createdAt
		edge.SourceIDs = []string{episode.ID}

		duplicateEdges = append(duplicateEdges, edge)
	}

	return duplicateEdges, nil
}

// ExtractEdges extracts relationship edges from episode content using LLM
func (eo *EdgeOperations) ExtractEdges(ctx context.Context, episode *types.Node, nodes []*types.Node, previousEpisodes []*types.Node, edgeTypeMap map[string][]string, groupID string) ([]*types.Edge, error) {
	start := time.Now()

	if len(nodes) == 0 {
		return []*types.Edge{}, nil
	}

	// Prepare context for LLM
	nodeContexts := make([]map[string]interface{}, len(nodes))
	for i, node := range nodes {
		nodeContexts[i] = map[string]interface{}{
			"id":           i,
			"name":         node.Name,
			"entity_types": []string{string(node.Type)}, // Simplified for now
		}
	}

	previousEpisodeContents := make([]string, len(previousEpisodes))
	for i, ep := range previousEpisodes {
		previousEpisodeContents[i] = ep.Summary
	}

	promptContext := map[string]interface{}{
		"episode_content":   episode.Summary,
		"nodes":             nodeContexts,
		"previous_episodes": previousEpisodeContents,
		"reference_time":    episode.ValidFrom,
		"edge_types":        []map[string]interface{}{}, // Simplified for now
		"custom_prompt":     "",
		"ensure_ascii":      true,
	}

	// Extract edges using LLM
	messages, err := eo.prompts.ExtractEdges().Edge().Call(promptContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt: %w", err)
	}

	response, err := eo.llm.ChatWithStructuredOutput(ctx, messages, &prompts.ExtractedEdges{})
	if err != nil {
		return nil, fmt.Errorf("failed to extract edges: %w", err)
	}

	var extractedEdges prompts.ExtractedEdges
	if err := json.Unmarshal(response, &extractedEdges); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Printf("Extracted %d edges in %v", len(extractedEdges.Edges), time.Since(start))

	if len(extractedEdges.Edges) == 0 {
		return []*types.Edge{}, nil
	}

	// Convert to Edge objects
	edges := make([]*types.Edge, 0, len(extractedEdges.Edges))
	for _, edgeData := range extractedEdges.Edges {
		// Validate node indices
		if edgeData.SourceEntityID < 0 || edgeData.SourceEntityID >= len(nodes) ||
			edgeData.TargetEntityID < 0 || edgeData.TargetEntityID >= len(nodes) {
			log.Printf("Warning: invalid node indices for edge %s", edgeData.RelationType)
			continue
		}

		sourceNode := nodes[edgeData.SourceEntityID]
		targetNode := nodes[edgeData.TargetEntityID]

		// Parse temporal information
		var validAt time.Time
		var validTo *time.Time

		if edgeData.ValidAt != nil && *edgeData.ValidAt != "" {
			if parsed, err := time.Parse(time.RFC3339, strings.ReplaceAll(*edgeData.ValidAt, "Z", "+00:00")); err == nil {
				validAt = parsed.UTC()
			} else {
				log.Printf("Warning: failed to parse valid_at date: %v", err)
				validAt = episode.ValidFrom
			}
		} else {
			validAt = episode.ValidFrom
		}

		if edgeData.InvalidAt != nil && *edgeData.InvalidAt != "" {
			if parsed, err := time.Parse(time.RFC3339, strings.ReplaceAll(*edgeData.InvalidAt, "Z", "+00:00")); err == nil {
				parsedUTC := parsed.UTC()
				validTo = &parsedUTC
			} else {
				log.Printf("Warning: failed to parse invalid_at date: %v", err)
			}
		}

		edge := types.NewEntityEdge(
			utils.GenerateUUID(),
			sourceNode.ID,
			targetNode.ID,
			groupID,
			edgeData.RelationType,
			types.EntityEdgeType,
		)
		edge.Summary = edgeData.Fact
		edge.Fact = edgeData.Fact
		edge.UpdatedAt = time.Now().UTC()
		edge.ValidFrom = validAt
		edge.ValidTo = validTo
		edge.SourceIDs = []string{episode.ID}

		edges = append(edges, edge)
		log.Printf("Created edge: %s from %s to %s", edge.Name, sourceNode.Name, targetNode.Name)
	}

	return edges, nil
}

// GetBetweenNodes retrieves edges between two specific nodes using the proper Kuzu query pattern
func (eo *EdgeOperations) GetBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string) ([]*types.Edge, error) {
	query := `
		MATCH (a:Entity {uuid: $source_uuid})-[:RELATES_TO]->(rel:RelatesToNode_)-[:RELATES_TO]->(b:Entity {uuid: $target_uuid})
		RETURN rel.*, a.uuid AS source_id, b.uuid AS target_id
		UNION
		MATCH (a:Entity {uuid: $target_uuid})-[:RELATES_TO]->(rel:RelatesToNode_)-[:RELATES_TO]->(b:Entity {uuid: $source_uuid})
		RETURN rel.*, a.uuid AS source_id, b.uuid AS target_id
	`

	params := map[string]interface{}{
		"source_uuid": sourceNodeID,
		"target_uuid": targetNodeID,
	}

	result, _, _, err := eo.driver.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetBetweenNodes query: %w", err)
	}

	// Convert result to Edge objects
	var edges []*types.Edge
	if result != nil {
		// Handle different result types based on driver implementation
		switch records := result.(type) {
		case []map[string]interface{}:
			for _, record := range records {
				edge, err := eo.convertRecordToEdge(record)
				if err != nil {
					log.Printf("Warning: failed to convert record to edge: %v", err)
					continue
				}
				edges = append(edges, edge)
			}
		default:
			log.Printf("Warning: unexpected result type from GetBetweenNodes query: %T", result)
		}
	}

	return edges, nil
}

// convertRecordToEdge converts a database record to an Edge object
func (eo *EdgeOperations) convertRecordToEdge(record map[string]interface{}) (*types.Edge, error) {
	edge := &types.Edge{}

	// Extract basic fields
	if uuid, ok := record["uuid"].(string); ok {
		edge.ID = uuid
	} else {
		return nil, fmt.Errorf("missing or invalid uuid field")
	}

	if name, ok := record["name"].(string); ok {
		edge.Name = name
	}

	if fact, ok := record["fact"].(string); ok {
		edge.Summary = fact
	}

	if groupID, ok := record["group_id"].(string); ok {
		edge.GroupID = groupID
	}

	// Extract source and target IDs
	if sourceID, ok := record["source_id"].(string); ok {
		edge.SourceID = sourceID
	}
	if targetID, ok := record["target_id"].(string); ok {
		edge.TargetID = targetID
	}

	// Extract timestamps
	if createdAt, ok := record["created_at"].(time.Time); ok {
		edge.CreatedAt = createdAt
	}
	if updatedAt, ok := record["updated_at"].(time.Time); ok {
		edge.UpdatedAt = updatedAt
	}
	if validFrom, ok := record["valid_from"].(time.Time); ok {
		edge.ValidFrom = validFrom
	}
	if validTo, ok := record["valid_to"].(time.Time); ok {
		edge.ValidTo = &validTo
	}

	// Set edge type - assume EntityEdge for relationships from RelatesToNode_
	edge.Type = types.EntityEdgeType

	// Extract source IDs if present
	if sourceIDs, ok := record["source_ids"].([]interface{}); ok {
		strSourceIDs := make([]string, len(sourceIDs))
		for i, id := range sourceIDs {
			if strID, ok := id.(string); ok {
				strSourceIDs[i] = strID
			}
		}
		edge.SourceIDs = strSourceIDs
	}

	return edge, nil
}

// ResolveExtractedEdges resolves newly extracted edges with existing ones in the graph
func (eo *EdgeOperations) ResolveExtractedEdges(ctx context.Context, extractedEdges []*types.Edge, episode *types.Node, entities []*types.Node) ([]*types.Edge, []*types.Edge, error) {
	if len(extractedEdges) == 0 {
		return []*types.Edge{}, []*types.Edge{}, nil
	}

	// Create entity UUID to node mapping for quick lookup
	entityMap := make(map[string]*types.Node)
	for _, entity := range entities {
		entityMap[entity.ID] = entity
	}

	resolvedEdges := make([]*types.Edge, 0, len(extractedEdges))
	invalidatedEdges := make([]*types.Edge, 0)

	// Process each extracted edge
	for _, extractedEdge := range extractedEdges {
		// Create embeddings for the edge
		if err := eo.createEdgeEmbedding(ctx, extractedEdge); err != nil {
			log.Printf("Warning: failed to create embedding for edge: %v", err)
		}

		// Get existing edges between the same nodes
		existingEdges, err := eo.GetBetweenNodes(ctx, extractedEdge.SourceID, extractedEdge.TargetID)
		if err != nil {
			log.Printf("Warning: failed to get existing edges: %v", err)
			existingEdges = []*types.Edge{}
		}

		// Search for related edges using semantic search
		relatedEdges, err := eo.searchRelatedEdges(ctx, extractedEdge, existingEdges)
		if err != nil {
			log.Printf("Warning: failed to search related edges: %v", err)
			relatedEdges = []*types.Edge{}
		}

		// Resolve the edge against existing ones
		resolvedEdge, newlyInvalidated, err := eo.resolveExtractedEdge(ctx, extractedEdge, relatedEdges, existingEdges, episode)
		if err != nil {
			log.Printf("Warning: failed to resolve edge: %v", err)
			// Use the original edge if resolution fails
			resolvedEdge = extractedEdge
		}

		// If the edge is a duplicate, add episode to existing edge
		if resolvedEdge != extractedEdge && episode != nil {
			// Add episode to source IDs if not already present
			found := false
			for _, sourceID := range resolvedEdge.SourceIDs {
				if sourceID == episode.ID {
					found = true
					break
				}
			}
			if !found {
				resolvedEdge.SourceIDs = append(resolvedEdge.SourceIDs, episode.ID)
				resolvedEdge.UpdatedAt = time.Now().UTC()
			}
		}

		resolvedEdges = append(resolvedEdges, resolvedEdge)
		invalidatedEdges = append(invalidatedEdges, newlyInvalidated...)
	}

	// Create embeddings for all resolved and invalidated edges
	allEdges := append(resolvedEdges, invalidatedEdges...)
	for _, edge := range allEdges {
		if err := eo.createEdgeEmbedding(ctx, edge); err != nil {
			log.Printf("Warning: failed to create embedding for edge: %v", err)
		}
	}

	log.Printf("Resolved %d edges, invalidated %d edges", len(resolvedEdges), len(invalidatedEdges))
	return resolvedEdges, invalidatedEdges, nil
}

// createEdgeEmbedding creates an embedding for an edge based on its summary
func (eo *EdgeOperations) createEdgeEmbedding(ctx context.Context, edge *types.Edge) error {
	if edge.Summary == "" {
		return nil
	}

	embedding, err := eo.embedder.EmbedSingle(ctx, edge.Summary)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	edge.Embedding = embedding
	return nil
}

// searchRelatedEdges searches for semantically related edges using hybrid search with UUID filtering
func (eo *EdgeOperations) searchRelatedEdges(ctx context.Context, extractedEdge *types.Edge, existingEdges []*types.Edge) ([]*types.Edge, error) {
	if extractedEdge.Summary == "" {
		return []*types.Edge{}, nil
	}

	// Create UUID filter for existing edges (equivalent to Python's SearchFilters(edge_uuids=...))
	edgeUUIDs := make([]string, len(existingEdges))
	for i, edge := range existingEdges {
		edgeUUIDs[i] = edge.ID
	}

	// Create a map for quick UUID lookup
	validUUIDs := make(map[string]bool)
	for _, uuid := range edgeUUIDs {
		validUUIDs[uuid] = true
	}

	// Use hybrid search with proper filtering
	// This is equivalent to Python's EDGE_HYBRID_SEARCH_RRF config
	searchOptions := &driver.SearchOptions{
		Limit:     50,
		EdgeTypes: []types.EdgeType{types.EntityEdgeType},
		// Note: GroupIDs filtering would need to be added to SearchOptions
	}

	// Search for edges using semantic similarity (fact content)
	edges, err := eo.driver.SearchEdges(ctx, extractedEdge.Summary, extractedEdge.GroupID, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to search related edges: %w", err)
	}

	// Filter results to only include edges in the UUID filter
	var relatedEdges []*types.Edge
	for _, edge := range edges {
		// Only include edges that are in the valid UUID set
		if len(edgeUUIDs) == 0 || validUUIDs[edge.ID] {
			// Exclude the extracted edge itself
			if edge.ID != extractedEdge.ID {
				relatedEdges = append(relatedEdges, edge)
			}
		}
	}

	log.Printf("Found %d related edges for edge fact: %s", len(relatedEdges), extractedEdge.Summary)
	return relatedEdges, nil
}

// resolveExtractedEdge resolves a single extracted edge against existing edges
func (eo *EdgeOperations) resolveExtractedEdge(ctx context.Context, extractedEdge *types.Edge, relatedEdges []*types.Edge, existingEdges []*types.Edge, episode *types.Node) (*types.Edge, []*types.Edge, error) {
	if len(relatedEdges) == 0 && len(existingEdges) == 0 {
		return extractedEdge, []*types.Edge{}, nil
	}

	start := time.Now()

	// Prepare context for LLM deduplication
	relatedEdgesContext := make([]map[string]interface{}, len(relatedEdges))
	for i, edge := range relatedEdges {
		relatedEdgesContext[i] = map[string]interface{}{
			"id":   edge.ID,
			"fact": edge.Summary,
		}
	}

	invalidationCandidatesContext := make([]map[string]interface{}, len(existingEdges))
	for i, edge := range existingEdges {
		invalidationCandidatesContext[i] = map[string]interface{}{
			"id":   i,
			"fact": edge.Summary,
		}
	}

	promptContext := map[string]interface{}{
		"existing_edges":               relatedEdgesContext,
		"new_edge":                     extractedEdge.Summary,
		"edge_invalidation_candidates": invalidationCandidatesContext,
		"edge_types":                   []map[string]interface{}{}, // Simplified
		"ensure_ascii":                 true,
	}

	// Use LLM to resolve duplicates and contradictions
	messages, err := eo.prompts.DedupeEdges().ResolveEdge().Call(promptContext)
	if err != nil {
		log.Printf("Warning: failed to create dedupe prompt: %v", err)
		return extractedEdge, []*types.Edge{}, nil
	}

	response, err := eo.llm.ChatWithStructuredOutput(ctx, messages, &prompts.EdgeDuplicate{})
	if err != nil {
		log.Printf("Warning: LLM edge resolution failed: %v", err)
		return extractedEdge, []*types.Edge{}, nil
	}

	var edgeDuplicate prompts.EdgeDuplicate
	if err := json.Unmarshal(response, &edgeDuplicate); err != nil {
		log.Printf("Warning: failed to unmarshal dedupe response: %v", err)
		return extractedEdge, []*types.Edge{}, nil
	}

	// Process duplicate facts
	resolvedEdge := extractedEdge
	for _, duplicateFactID := range edgeDuplicate.DuplicateFacts {
		if duplicateFactID >= 0 && duplicateFactID < len(relatedEdges) {
			resolvedEdge = relatedEdges[duplicateFactID]
			break
		}
	}

	// Process contradicted facts (invalidation candidates)
	var invalidatedEdges []*types.Edge
	for _, contradictedFactID := range edgeDuplicate.ContradictedFacts {
		if contradictedFactID >= 0 && contradictedFactID < len(existingEdges) {
			candidateEdge := existingEdges[contradictedFactID]

			// Apply temporal logic for invalidation
			invalidatedEdge := eo.resolveEdgeContradictions(resolvedEdge, []*types.Edge{candidateEdge})
			invalidatedEdges = append(invalidatedEdges, invalidatedEdge...)
		}
	}

	// Update fact type if specified
	if edgeDuplicate.FactType != "" && strings.ToUpper(edgeDuplicate.FactType) != "DEFAULT" {
		resolvedEdge.Name = edgeDuplicate.FactType
	}

	// Handle temporal invalidation logic
	now := time.Now().UTC()
	if resolvedEdge.ValidTo != nil && resolvedEdge.ValidTo.Before(now) {
		// Edge is already expired, don't modify expiration
	}

	log.Printf("Resolved edge %s in %v", extractedEdge.Name, time.Since(start))
	return resolvedEdge, invalidatedEdges, nil
}

// resolveEdgeContradictions handles temporal contradictions between edges
func (eo *EdgeOperations) resolveEdgeContradictions(resolvedEdge *types.Edge, invalidationCandidates []*types.Edge) []*types.Edge {
	if len(invalidationCandidates) == 0 {
		return []*types.Edge{}
	}

	now := time.Now().UTC()
	var invalidatedEdges []*types.Edge

	for _, edge := range invalidationCandidates {
		// Skip edges that are already invalid before the new edge becomes valid
		if edge.ValidTo != nil && resolvedEdge.ValidFrom.After(*edge.ValidTo) {
			continue
		}

		// Skip if new edge is invalid before the candidate becomes valid
		if resolvedEdge.ValidTo != nil && edge.ValidFrom.After(*resolvedEdge.ValidTo) {
			continue
		}

		// Invalidate edge if the new edge becomes valid after this one
		if edge.ValidFrom.Before(resolvedEdge.ValidFrom) {
			edgeCopy := *edge
			validTo := resolvedEdge.ValidFrom
			edgeCopy.ValidTo = &validTo
			edgeCopy.UpdatedAt = now
			invalidatedEdges = append(invalidatedEdges, &edgeCopy)
		}
	}

	return invalidatedEdges
}

// FilterExistingDuplicateOfEdges filters out duplicate node pairs that already have IS_DUPLICATE_OF edges using proper Kuzu query
func (eo *EdgeOperations) FilterExistingDuplicateOfEdges(ctx context.Context, duplicateNodePairs []NodePair) ([]NodePair, error) {
	if len(duplicateNodePairs) == 0 {
		return []NodePair{}, nil
	}

	// Prepare parameters exactly like Python implementation
	duplicateNodeUUIDs := make([]map[string]interface{}, len(duplicateNodePairs))
	for i, pair := range duplicateNodePairs {
		duplicateNodeUUIDs[i] = map[string]interface{}{
			"src": pair.Source.ID,
			"dst": pair.Target.ID,
		}
	}

	query := `
		UNWIND $duplicate_node_uuids AS duplicate
		MATCH (n:Entity {uuid: duplicate.src})-[:RELATES_TO]->(e:RelatesToNode_ {name: 'IS_DUPLICATE_OF'})-[:RELATES_TO]->(m:Entity {uuid: duplicate.dst})
		RETURN DISTINCT
			n.uuid AS source_uuid,
			m.uuid AS target_uuid
	`

	params := map[string]interface{}{
		"duplicate_node_uuids": duplicateNodeUUIDs,
	}

	result, _, _, err := eo.driver.ExecuteQuery(query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute FilterExistingDuplicateOfEdges query: %w", err)
	}

	// Create a set of existing duplicate pairs
	existingPairs := make(map[string]bool)
	if result != nil {
		switch records := result.(type) {
		case []map[string]interface{}:
			for _, record := range records {
				if sourceUUID, ok := record["source_uuid"].(string); ok {
					if targetUUID, ok := record["target_uuid"].(string); ok {
						key := fmt.Sprintf("%s-%s", sourceUUID, targetUUID)
						existingPairs[key] = true
					}
				}
			}
		default:
			log.Printf("Warning: unexpected result type from FilterExistingDuplicateOfEdges query: %T", result)
		}
	}

	// Filter out pairs that already exist
	var filteredPairs []NodePair
	for _, pair := range duplicateNodePairs {
		key := fmt.Sprintf("%s-%s", pair.Source.ID, pair.Target.ID)
		if !existingPairs[key] {
			filteredPairs = append(filteredPairs, pair)
		}
	}

	log.Printf("Filtered %d duplicate node pairs, %d remain after filtering existing IS_DUPLICATE_OF edges",
		len(duplicateNodePairs)-len(filteredPairs), len(filteredPairs))

	return filteredPairs, nil
}
