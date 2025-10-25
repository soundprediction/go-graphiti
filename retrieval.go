package graphiti

import (
	"context"
	"fmt"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/search"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

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

// GetStats retrieves statistics about the knowledge graph.
func (c *Client) GetStats(ctx context.Context) (*driver.GraphStats, error) {
	return c.driver.GetStats(ctx, c.config.GroupID)
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

// GetNodesAndEdgesByEpisode retrieves all nodes and edges mentioned in a specific episode.
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

// convertSearchMethods converts string search methods to search.SearchMethod enum.
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

// convertReranker converts string reranker to search.RerankerType enum.
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
