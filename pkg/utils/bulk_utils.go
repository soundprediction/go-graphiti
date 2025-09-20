package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/prompts"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// Clients represents the set of clients needed for bulk operations
type Clients struct {
	Driver   driver.GraphDriver
	LLM      llm.Client
	Embedder embedder.Client
	Prompts  prompts.Library
}

// ExtractNodesAndEdgesResult represents the result of bulk node and edge extraction
type ExtractNodesAndEdgesResult struct {
	ExtractedNodes []*types.Node
	ExtractedEdges []*types.Edge
}

// AddNodesAndEdgesResult represents the result of bulk add operations
type AddNodesAndEdgesResult struct {
	EpisodicNodes []*types.Node
	EpisodicEdges []*types.Edge
	EntityNodes   []*types.Node
	EntityEdges   []*types.Edge
	Errors        []error
}

// RetrievePreviousEpisodesBulk retrieves previous episodes for a list of episodes
// This matches the Python function signature: retrieve_previous_episodes_bulk(driver, episodes)
func RetrievePreviousEpisodesBulk(ctx context.Context, driver driver.GraphDriver, episodes []*types.Episode) ([]EpisodeTuple, error) {
	var episodeTuples []EpisodeTuple

	for _, episode := range episodes {
		// Get previous episodes using temporal search
		// Get nodes in the time range before this episode
		previousNodes, err := driver.GetNodesInTimeRange(ctx, episode.CreatedAt.Add(-24*time.Hour), episode.CreatedAt, episode.GroupID)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous episodes for group %s: %w", episode.GroupID, err)
		}

		// Convert Node results to Episodes and filter for episodic nodes
		var prevEpisodes []*types.Episode
		for _, node := range previousNodes {
			if node.Type == types.EpisodicNodeType && node.ID != episode.ID {
				prevEpisodes = append(prevEpisodes, &types.Episode{
					ID:        node.ID,
					Name:      node.Name,
					Content:   node.Content,
					Reference: node.Reference,
					CreatedAt: node.CreatedAt,
					GroupID:   node.GroupID,
					Metadata:  node.Metadata,
				})
			}
		}

		episodeTuples = append(episodeTuples, EpisodeTuple{
			Episode:          episode,
			PreviousEpisodes: prevEpisodes,
		})
	}

	return episodeTuples, nil
}

// AddNodesAndEdgesBulk adds nodes and edges to the graph database in bulk
// This matches the Python function signature: add_nodes_and_edges_bulk(driver, episodic_nodes, episodic_edges, entity_nodes, entity_edges, embedder)
func AddNodesAndEdgesBulk(
	ctx context.Context,
	driver driver.GraphDriver,
	episodicNodes []*types.Node,
	episodicEdges []*types.Edge,
	entityNodes []*types.Node,
	entityEdges []*types.Edge,
	embedder embedder.Client,
) (*AddNodesAndEdgesResult, error) {
	result := &AddNodesAndEdgesResult{}

	// Add episodic nodes
	if len(episodicNodes) > 0 {
		for _, node := range episodicNodes {
			if err := driver.UpsertNode(ctx, node); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to upsert episodic node %s: %w", node.ID, err))
			} else {
				result.EpisodicNodes = append(result.EpisodicNodes, node)
			}
		}
	}

	// Add entity nodes with embeddings
	if len(entityNodes) > 0 {
		// Generate embeddings for entity nodes if needed
		var textsToEmbed []string
		var nodeIndices []int
		for i, node := range entityNodes {
			if len(node.Embedding) == 0 && node.Name != "" {
				textsToEmbed = append(textsToEmbed, node.Name)
				nodeIndices = append(nodeIndices, i)
			}
		}

		if len(textsToEmbed) > 0 && embedder != nil {
			embeddings, err := embedder.Embed(ctx, textsToEmbed)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to generate embeddings: %w", err))
			} else {
				for i, embedding := range embeddings {
					if i < len(nodeIndices) {
						entityNodes[nodeIndices[i]].Embedding = embedding
					}
				}
			}
		}

		// Upsert entity nodes
		for _, node := range entityNodes {
			if err := driver.UpsertNode(ctx, node); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to upsert entity node %s: %w", node.ID, err))
			} else {
				result.EntityNodes = append(result.EntityNodes, node)
			}
		}
	}

	// Add episodic edges
	if len(episodicEdges) > 0 {
		for _, edge := range episodicEdges {
			if err := driver.UpsertEdge(ctx, edge); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to upsert episodic edge %s: %w", edge.ID, err))
			} else {
				result.EpisodicEdges = append(result.EpisodicEdges, edge)
			}
		}
	}

	// Add entity edges with embeddings
	if len(entityEdges) > 0 {
		// Generate embeddings for entity edges if needed
		var textsToEmbed []string
		var edgeIndices []int
		for i, edge := range entityEdges {
			if len(edge.Embedding) == 0 && edge.Summary != "" {
				textsToEmbed = append(textsToEmbed, edge.Summary)
				edgeIndices = append(edgeIndices, i)
			}
		}

		if len(textsToEmbed) > 0 && embedder != nil {
			embeddings, err := embedder.Embed(ctx, textsToEmbed)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to generate edge embeddings: %w", err))
			} else {
				for i, embedding := range embeddings {
					if i < len(edgeIndices) {
						entityEdges[edgeIndices[i]].Embedding = embedding
					}
				}
			}
		}

		// Upsert entity edges
		for _, edge := range entityEdges {
			if err := driver.UpsertEdge(ctx, edge); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to upsert entity edge %s: %w", edge.ID, err))
			} else {
				result.EntityEdges = append(result.EntityEdges, edge)
			}
		}
	}

	return result, nil
}

// ExtractNodesAndEdgesBulk extracts nodes and edges from episodes in bulk
// This matches the Python function signature: extract_nodes_and_edges_bulk(clients, episode_tuples, edge_type_map, ...)
func ExtractNodesAndEdgesBulk(
	ctx context.Context,
	clients *Clients,
	episodeTuples []EpisodeTuple,
	edgeTypeMap map[string]string,
	entityTypes []string,
	excludedEntityTypes []string,
	edgeTypes []string,
) (*ExtractNodesAndEdgesResult, error) {
	var allExtractedNodes []*types.Node
	var allExtractedEdges []*types.Edge

	// Process episodes in batches for better performance
	batchProcessor := NewBatchProcessor(
		GetSemaphoreLimit(),
		GetSemaphoreLimit(),
		func(ctx context.Context, batch []EpisodeTuple) ([]*ExtractNodesAndEdgesResult, error) {
			var results []*ExtractNodesAndEdgesResult
			for _, episodeTuple := range batch {
				result, err := extractFromSingleEpisode(ctx, clients, episodeTuple, edgeTypeMap, entityTypes, excludedEntityTypes, edgeTypes)
				if err != nil {
					return nil, err
				}
				results = append(results, result)
			}
			return results, nil
		},
	)

	batchResults, err := batchProcessor.Process(ctx, episodeTuples)
	if err != nil {
		return nil, fmt.Errorf("failed to process episode batches: %w", err)
	}

	// Aggregate results
	for _, result := range batchResults {
		allExtractedNodes = append(allExtractedNodes, result.ExtractedNodes...)
		allExtractedEdges = append(allExtractedEdges, result.ExtractedEdges...)
	}

	return &ExtractNodesAndEdgesResult{
		ExtractedNodes: allExtractedNodes,
		ExtractedEdges: allExtractedEdges,
	}, nil
}

// extractFromSingleEpisode extracts nodes and edges from a single episode
func extractFromSingleEpisode(
	ctx context.Context,
	clients *Clients,
	episodeTuple EpisodeTuple,
	edgeTypeMap map[string]string,
	entityTypes []string,
	excludedEntityTypes []string,
	edgeTypes []string,
) (*ExtractNodesAndEdgesResult, error) {
	// This is a simplified implementation - in practice this would use
	// the LLM client and prompts library to extract entities and relationships

	// Create context from episode and previous episodes
	content := episodeTuple.Episode.Content
	for _, prevEpisode := range episodeTuple.PreviousEpisodes {
		content += "\nPrevious: " + prevEpisode.Content
	}

	// Use LLM to extract entities (simplified)
	// Build context for the prompt
	promptContext := map[string]interface{}{
		"episode_content":    content,
		"entity_types":       entityTypes,
		"excluded_entity_types": excludedEntityTypes,
		"previous_episodes":  episodeTuple.PreviousEpisodes,
	}

	entityMessages, err := clients.Prompts.ExtractNodes().ExtractMessage().Call(promptContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity extraction prompt: %w", err)
	}

	entityResponse, err := clients.LLM.Chat(ctx, entityMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract entities: %w", err)
	}

	// Parse entities from response (this would need proper JSON parsing)
	var extractedNodes []*types.Node
	// Simplified parsing - in practice this would parse JSON response
	entities := strings.Split(entityResponse.Content, ",")
	for i, entity := range entities {
		entity = strings.TrimSpace(entity)
		if entity != "" {
			node := &types.Node{
				ID:        fmt.Sprintf("entity_%d_%s", i, episodeTuple.Episode.ID),
				Name:      entity,
				Type:      types.EntityNodeType,
				GroupID:   episodeTuple.Episode.GroupID,
				CreatedAt: episodeTuple.Episode.CreatedAt,
				Summary:   entity,
			}
			extractedNodes = append(extractedNodes, node)
		}
	}

	// Use LLM to extract relationships (simplified)
	edgeContext := map[string]interface{}{
		"episode_content": content,
		"extracted_nodes": extractedNodes,
		"edge_types":      edgeTypes,
	}

	edgeMessages, err := clients.Prompts.ExtractEdges().Edge().Call(edgeContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create edge extraction prompt: %w", err)
	}

	edgeResponse, err := clients.LLM.Chat(ctx, edgeMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract edges: %w", err)
	}

	// Parse edges from response (simplified)
	var extractedEdges []*types.Edge
	// This would need proper JSON parsing in practice
	relationships := strings.Split(edgeResponse.Content, ";")
	for i, rel := range relationships {
		rel = strings.TrimSpace(rel)
		if rel != "" && len(extractedNodes) >= 2 {
			edge := &types.Edge{
				ID:        fmt.Sprintf("edge_%d_%s", i, episodeTuple.Episode.ID),
				SourceID:  extractedNodes[0].ID,
				TargetID:  extractedNodes[min(1, len(extractedNodes)-1)].ID,
				Type:      types.EntityEdgeType,
				GroupID:   episodeTuple.Episode.GroupID,
				CreatedAt: episodeTuple.Episode.CreatedAt,
				Summary:   rel,
				Name:      rel,
			}
			extractedEdges = append(extractedEdges, edge)
		}
	}

	return &ExtractNodesAndEdgesResult{
		ExtractedNodes: extractedNodes,
		ExtractedEdges: extractedEdges,
	}, nil
}

// DedupeNodesBulk deduplicates extracted nodes across episodes
// This matches the Python function signature: dedupe_nodes_bulk(clients, extracted_nodes, episode_tuples, ...)
func DedupeNodesBulk(
	ctx context.Context,
	clients *Clients,
	extractedNodes []*types.Node,
	episodeTuples []EpisodeTuple,
	embedder embedder.Client,
) (*DedupeNodesResult, error) {
	if len(extractedNodes) == 0 {
		return &DedupeNodesResult{
			NodesByEpisode: make(map[string][]*types.Node),
			UUIDMap:        make(map[string]string),
		}, nil
	}

	// Generate embeddings for nodes if not present
	var nodesToEmbed []*types.Node
	var textsToEmbed []string
	for _, node := range extractedNodes {
		if len(node.Embedding) == 0 && node.Name != "" {
			nodesToEmbed = append(nodesToEmbed, node)
			textsToEmbed = append(textsToEmbed, node.Name)
		}
	}

	if len(textsToEmbed) > 0 && embedder != nil {
		embeddings, err := embedder.Embed(ctx, textsToEmbed)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings for deduplication: %w", err)
		}
		for i, embedding := range embeddings {
			if i < len(nodesToEmbed) {
				nodesToEmbed[i].Embedding = embedding
			}
		}
	}

	// Find duplicates using similarity comparison
	var duplicatePairs [][]string
	processed := make(map[string]bool)

	for i, node1 := range extractedNodes {
		if processed[node1.ID] {
			continue
		}

		similar := FindSimilarNodes(node1, extractedNodes[i+1:], MinScoreNodes)
		if len(similar) > 0 {
			for _, node2 := range similar {
				if !processed[node2.ID] {
					duplicatePairs = append(duplicatePairs, []string{node1.ID, node2.ID})
					processed[node2.ID] = true
				}
			}
		}
		processed[node1.ID] = true
	}

	// Use LLM to confirm duplicates (simplified)
	if len(duplicatePairs) > 0 && clients != nil && clients.LLM != nil {
		confirmedPairs := make([][]string, 0, len(duplicatePairs))

		for _, pair := range duplicatePairs {
			// Find the actual nodes
			var node1, node2 *types.Node
			for _, node := range extractedNodes {
				if node.ID == pair[0] {
					node1 = node
				} else if node.ID == pair[1] {
					node2 = node
				}
			}

			if node1 != nil && node2 != nil {
				// Use LLM to confirm if they are duplicates
				dedupeContext := map[string]interface{}{
					"nodes": []*types.Node{node1, node2},
				}

				dedupeMessages, err := clients.Prompts.DedupeNodes().Node().Call(dedupeContext)
				if err == nil {
					response, err := clients.LLM.Chat(ctx, dedupeMessages)
					if err == nil && strings.Contains(strings.ToLower(response.Content), "duplicate") {
						confirmedPairs = append(confirmedPairs, pair)
					}
				}
			}
		}
		duplicatePairs = confirmedPairs
	}

	// Create UUID mapping using UnionFind
	uuidMap := CompressUUIDMap(duplicatePairs)

	// Group nodes by episode
	nodesByEpisode := make(map[string][]*types.Node)
	nodeMap := make(map[string]*types.Node)

	// Create node map and apply UUID mappings
	for _, node := range extractedNodes {
		canonicalID := uuidMap[node.ID]
		if canonicalID == "" {
			canonicalID = node.ID
		}

		// Use the canonical node (lexicographically smallest ID)
		if existingNode, exists := nodeMap[canonicalID]; exists {
			// Merge properties if needed (simplified)
			if existingNode.Name == "" && node.Name != "" {
				existingNode.Name = node.Name
			}
			if existingNode.Summary == "" && node.Summary != "" {
				existingNode.Summary = node.Summary
			}
		} else {
			// Create a copy with canonical ID
			canonicalNode := *node
			canonicalNode.ID = canonicalID
			nodeMap[canonicalID] = &canonicalNode
		}
	}

	// Group nodes by their source episodes
	for _, episodeTuple := range episodeTuples {
		var episodeNodes []*types.Node
		seen := make(map[string]bool)

		// Find nodes that came from this episode
		for _, node := range extractedNodes {
			// This is simplified - in practice you'd track which episode each node came from
			if strings.Contains(node.ID, episodeTuple.Episode.ID) {
				canonicalID := uuidMap[node.ID]
				if canonicalID == "" {
					canonicalID = node.ID
				}

				if !seen[canonicalID] && nodeMap[canonicalID] != nil {
					episodeNodes = append(episodeNodes, nodeMap[canonicalID])
					seen[canonicalID] = true
				}
			}
		}

		nodesByEpisode[episodeTuple.Episode.ID] = episodeNodes
	}

	return &DedupeNodesResult{
		NodesByEpisode: nodesByEpisode,
		UUIDMap:        uuidMap,
	}, nil
}

// DedupeEdgesBulk deduplicates extracted edges across episodes
// This matches the Python function signature: dedupe_edges_bulk(clients, extracted_edges, episode_tuples, ...)
func DedupeEdgesBulk(
	ctx context.Context,
	clients *Clients,
	extractedEdges []*types.Edge,
	episodeTuples []EpisodeTuple,
	embedder embedder.Client,
) (*DedupeEdgesResult, error) {
	if len(extractedEdges) == 0 {
		return &DedupeEdgesResult{
			EdgesByEpisode: make(map[string][]*types.Edge),
			UUIDMap:        make(map[string]string),
		}, nil
	}

	// Generate embeddings for edges if not present
	var edgesToEmbed []*types.Edge
	var textsToEmbed []string
	for _, edge := range extractedEdges {
		if len(edge.Embedding) == 0 && edge.Summary != "" {
			edgesToEmbed = append(edgesToEmbed, edge)
			textsToEmbed = append(textsToEmbed, edge.Summary)
		}
	}

	if len(textsToEmbed) > 0 && embedder != nil {
		embeddings, err := embedder.Embed(ctx, textsToEmbed)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings for edge deduplication: %w", err)
		}
		for i, embedding := range embeddings {
			if i < len(edgesToEmbed) {
				edgesToEmbed[i].Embedding = embedding
			}
		}
	}

	// Find duplicates using similarity comparison
	var duplicatePairs [][]string
	processed := make(map[string]bool)

	for i, edge1 := range extractedEdges {
		if processed[edge1.ID] {
			continue
		}

		similar := FindSimilarEdges(edge1, extractedEdges[i+1:], MinScoreEdges)
		if len(similar) > 0 {
			for _, edge2 := range similar {
				if !processed[edge2.ID] {
					duplicatePairs = append(duplicatePairs, []string{edge1.ID, edge2.ID})
					processed[edge2.ID] = true
				}
			}
		}
		processed[edge1.ID] = true
	}

	// Use LLM to confirm duplicates (simplified)
	if len(duplicatePairs) > 0 && clients != nil && clients.LLM != nil {
		confirmedPairs := make([][]string, 0, len(duplicatePairs))

		for _, pair := range duplicatePairs {
			// Find the actual edges
			var edge1, edge2 *types.Edge
			for _, edge := range extractedEdges {
				if edge.ID == pair[0] {
					edge1 = edge
				} else if edge.ID == pair[1] {
					edge2 = edge
				}
			}

			if edge1 != nil && edge2 != nil {
				// Use LLM to confirm if they are duplicates
				dedupeContext := map[string]interface{}{
					"edges": []*types.Edge{edge1, edge2},
				}

				dedupeMessages, err := clients.Prompts.DedupeEdges().Edge().Call(dedupeContext)
				if err == nil {
					response, err := clients.LLM.Chat(ctx, dedupeMessages)
					if err == nil && strings.Contains(strings.ToLower(response.Content), "duplicate") {
						confirmedPairs = append(confirmedPairs, pair)
					}
				}
			}
		}
		duplicatePairs = confirmedPairs
	}

	// Create UUID mapping using UnionFind
	uuidMap := CompressUUIDMap(duplicatePairs)

	// Group edges by episode
	edgesByEpisode := make(map[string][]*types.Edge)
	edgeMap := make(map[string]*types.Edge)

	// Create edge map and apply UUID mappings
	for _, edge := range extractedEdges {
		canonicalID := uuidMap[edge.ID]
		if canonicalID == "" {
			canonicalID = edge.ID
		}

		// Use the canonical edge (lexicographically smallest ID)
		if existingEdge, exists := edgeMap[canonicalID]; exists {
			// Merge properties if needed (simplified)
			if existingEdge.Summary == "" && edge.Summary != "" {
				existingEdge.Summary = edge.Summary
			}
			if existingEdge.Name == "" && edge.Name != "" {
				existingEdge.Name = edge.Name
			}
		} else {
			// Create a copy with canonical ID
			canonicalEdge := *edge
			canonicalEdge.ID = canonicalID
			edgeMap[canonicalID] = &canonicalEdge
		}
	}

	// Group edges by their source episodes
	for _, episodeTuple := range episodeTuples {
		var episodeEdges []*types.Edge
		seen := make(map[string]bool)

		// Find edges that came from this episode
		for _, edge := range extractedEdges {
			// This is simplified - in practice you'd track which episode each edge came from
			if strings.Contains(edge.ID, episodeTuple.Episode.ID) {
				canonicalID := uuidMap[edge.ID]
				if canonicalID == "" {
					canonicalID = edge.ID
				}

				if !seen[canonicalID] && edgeMap[canonicalID] != nil {
					episodeEdges = append(episodeEdges, edgeMap[canonicalID])
					seen[canonicalID] = true
				}
			}
		}

		edgesByEpisode[episodeTuple.Episode.ID] = episodeEdges
	}

	return &DedupeEdgesResult{
		EdgesByEpisode: edgesByEpisode,
		UUIDMap:        uuidMap,
	}, nil
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}