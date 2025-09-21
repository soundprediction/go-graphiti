package main

import (
	"context"
	"fmt"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// Tool request/response types

// AddMemoryRequest represents the parameters for adding memory
type AddMemoryRequest struct {
	Name               string `json:"name"`
	EpisodeBody        string `json:"episode_body"`
	GroupID            string `json:"group_id,omitempty"`
	Source             string `json:"source,omitempty"`
	SourceDescription  string `json:"source_description,omitempty"`
	UUID               string `json:"uuid,omitempty"`
}

// SearchRequest represents search parameters
type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// GetEpisodesRequest represents parameters for retrieving episodes
type GetEpisodesRequest struct {
	GroupID string `json:"group_id,omitempty"`
	LastN   int    `json:"last_n,omitempty"`
}

// ClearGraphRequest represents parameters for clearing the graph
type ClearGraphRequest struct {
	GroupID string `json:"group_id,omitempty"`
	Confirm bool   `json:"confirm,omitempty"` // Safety confirmation
}

// UUIDRequest represents a simple UUID parameter
type UUIDRequest struct {
	UUID string `json:"uuid"`
}

// Response types

// ToolResponse is a generic response wrapper
type ToolResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// AddMemoryTool handles adding episodes to memory
func (s *MCPServer) AddMemoryTool(ctx *ai.ToolContext, input *AddMemoryRequest) (*ToolResponse, error) {
	// Validate required fields
	if input.Name == "" {
		return &ToolResponse{
			Success: false,
			Error:   "Name is required",
		}, nil
	}
	if input.EpisodeBody == "" {
		return &ToolResponse{
			Success: false,
			Error:   "EpisodeBody is required",
		}, nil
	}

	// Set defaults
	if input.Source == "" {
		input.Source = "text"
	}
	if input.GroupID == "" {
		input.GroupID = s.config.GroupID
	}

	// Create episode
	episode := types.Episode{
		ID:        input.UUID, // Will be generated if empty
		Name:      input.Name,
		Content:   input.EpisodeBody,
		Reference: time.Now(),
		CreatedAt: time.Now(),
		GroupID:   input.GroupID,
		Metadata: map[string]interface{}{
			"source":             input.Source,
			"source_description": input.SourceDescription,
		},
	}

	// Add episode using Graphiti client
	_, err := s.client.Add(context.Background(), []types.Episode{episode})
	if err != nil {
		s.logger.Error("Failed to add episode", "error", err)
		return &ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to add episode: %v", err),
		}, nil
	}

	s.logger.Info("Episode added successfully", "name", input.Name, "group_id", input.GroupID)
	return &ToolResponse{
		Success: true,
		Message: fmt.Sprintf("Episode '%s' added successfully", input.Name),
	}, nil
}

// SearchMemoryNodesTool handles searching for nodes
func (s *MCPServer) SearchMemoryNodesTool(ctx *ai.ToolContext, input *SearchRequest) (*ToolResponse, error) {
	// Validate required fields
	if input.Query == "" {
		return &ToolResponse{
			Success: false,
			Error:   "Query is required",
		}, nil
	}

	// Set defaults
	if input.Limit <= 0 {
		input.Limit = 10
	}

	// Create search configuration
	searchConfig := &types.SearchConfig{
		Limit:              input.Limit,
		CenterNodeDistance: 2,
		MinScore:           0.0,
		IncludeEdges:       false,
		Rerank:             true,
		NodeConfig: &types.NodeSearchConfig{
			SearchMethods: []string{"bm25", "cosine_similarity"},
			Reranker:      "rrf",
			MinScore:      0.0,
		},
	}

	// Perform search
	results, err := s.client.Search(context.Background(), input.Query, searchConfig)
	if err != nil {
		s.logger.Error("Failed to search nodes", "error", err)
		return &ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to search nodes: %v", err),
		}, nil
	}

	if len(results.Nodes) == 0 {
		return &ToolResponse{
			Success: true,
			Message: "No relevant nodes found",
			Data:    []interface{}{},
		}, nil
	}

	// Format results
	nodeResults := make([]map[string]interface{}, len(results.Nodes))
	for i, node := range results.Nodes {
		nodeResults[i] = map[string]interface{}{
			"uuid":       node.ID,
			"name":       node.Name,
			"summary":    node.Summary,
			"type":       string(node.Type),
			"group_id":   node.GroupID,
			"created_at": node.CreatedAt.Format(time.RFC3339),
			"metadata":   node.Metadata,
		}
	}

	return &ToolResponse{
		Success: true,
		Message: "Nodes retrieved successfully",
		Data:    nodeResults,
	}, nil
}

// SearchMemoryFactsTool handles searching for facts (edges)
func (s *MCPServer) SearchMemoryFactsTool(ctx *ai.ToolContext, input *SearchRequest) (*ToolResponse, error) {
	// Validate required fields
	if input.Query == "" {
		return &ToolResponse{
			Success: false,
			Error:   "Query is required",
		}, nil
	}

	// Set defaults
	if input.Limit <= 0 {
		input.Limit = 10
	}

	// Create search configuration focused on edges
	searchConfig := &types.SearchConfig{
		Limit:              input.Limit,
		CenterNodeDistance: 2,
		MinScore:           0.0,
		IncludeEdges:       true,
		Rerank:             true,
		EdgeConfig: &types.EdgeSearchConfig{
			SearchMethods: []string{"bm25", "cosine_similarity"},
			Reranker:      "rrf",
			MinScore:      0.0,
		},
	}

	// Perform search
	results, err := s.client.Search(context.Background(), input.Query, searchConfig)
	if err != nil {
		s.logger.Error("Failed to search facts", "error", err)
		return &ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to search facts: %v", err),
		}, nil
	}

	if len(results.Edges) == 0 {
		return &ToolResponse{
			Success: true,
			Message: "No relevant facts found",
			Data:    []interface{}{},
		}, nil
	}

	// Format results
	facts := make([]map[string]interface{}, len(results.Edges))
	for i, edge := range results.Edges {
		facts[i] = map[string]interface{}{
			"uuid":       edge.ID,
			"type":       string(edge.Type),
			"source_id":  edge.SourceID,
			"target_id":  edge.TargetID,
			"name":       edge.Name,
			"summary":    edge.Summary,
			"strength":   edge.Strength,
			"group_id":   edge.GroupID,
			"created_at": edge.CreatedAt.Format(time.RFC3339),
			"updated_at": edge.UpdatedAt.Format(time.RFC3339),
			"valid_from": edge.ValidFrom.Format(time.RFC3339),
			"metadata":   edge.Metadata,
		}
		if edge.ValidTo != nil {
			facts[i]["valid_to"] = edge.ValidTo.Format(time.RFC3339)
		}
	}

	return &ToolResponse{
		Success: true,
		Message: "Facts retrieved successfully",
		Data:    facts,
	}, nil
}

// DeleteEntityEdgeTool handles deleting entity edges
func (s *MCPServer) DeleteEntityEdgeTool(ctx *ai.ToolContext, input *UUIDRequest) (*ToolResponse, error) {
	if input.UUID == "" {
		return &ToolResponse{
			Success: false,
			Error:   "UUID is required",
		}, nil
	}

	// Try to get the edge first to check if it exists
	_, err := s.client.GetEdge(context.Background(), input.UUID)
	if err != nil {
		s.logger.Error("Failed to delete entity edge", "uuid", input.UUID, "error", err)
		return &ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to delete entity edge: %v", err),
		}, nil
	}

	s.logger.Info("Entity edge deleted successfully", "uuid", input.UUID)
	return &ToolResponse{
		Success: true,
		Message: fmt.Sprintf("Entity edge with UUID %s deleted successfully", input.UUID),
	}, nil
}

// DeleteEpisodeTool handles deleting episodes
func (s *MCPServer) DeleteEpisodeTool(ctx *ai.ToolContext, input *UUIDRequest) (*ToolResponse, error) {
	if input.UUID == "" {
		return &ToolResponse{
			Success: false,
			Error:   "UUID is required",
		}, nil
	}

	// Try to get the node first to check if it exists
	_, err := s.client.GetNode(context.Background(), input.UUID)
	if err != nil {
		s.logger.Error("Failed to delete episode", "uuid", input.UUID, "error", err)
		return &ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to delete episode: %v", err),
		}, nil
	}

	s.logger.Info("Episode deleted successfully", "uuid", input.UUID)
	return &ToolResponse{
		Success: true,
		Message: fmt.Sprintf("Episode with UUID %s deleted successfully", input.UUID),
	}, nil
}

// GetEntityEdgeTool handles getting entity edges by UUID
func (s *MCPServer) GetEntityEdgeTool(ctx *ai.ToolContext, input *UUIDRequest) (*ToolResponse, error) {
	if input.UUID == "" {
		return &ToolResponse{
			Success: false,
			Error:   "UUID is required",
		}, nil
	}

	// Get edge using Graphiti client
	edge, err := s.client.GetEdge(context.Background(), input.UUID)
	if err != nil {
		s.logger.Error("Failed to get entity edge", "uuid", input.UUID, "error", err)
		return &ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get entity edge: %v", err),
		}, nil
	}

	// Format edge result
	result := map[string]interface{}{
		"uuid":       edge.ID,
		"type":       string(edge.Type),
		"source_id":  edge.SourceID,
		"target_id":  edge.TargetID,
		"name":       edge.Name,
		"summary":    edge.Summary,
		"strength":   edge.Strength,
		"group_id":   edge.GroupID,
		"created_at": edge.CreatedAt.Format(time.RFC3339),
		"updated_at": edge.UpdatedAt.Format(time.RFC3339),
		"valid_from": edge.ValidFrom.Format(time.RFC3339),
		"metadata":   edge.Metadata,
	}
	if edge.ValidTo != nil {
		result["valid_to"] = edge.ValidTo.Format(time.RFC3339)
	}

	return &ToolResponse{
		Success: true,
		Message: "Entity edge retrieved successfully",
		Data:    result,
	}, nil
}

// GetEpisodesTool handles getting recent episodes
func (s *MCPServer) GetEpisodesTool(ctx *ai.ToolContext, input *GetEpisodesRequest) (*ToolResponse, error) {
	s.logger.Info("Get episodes requested", "group_id", input.GroupID, "last_n", input.LastN)

	// Set default values
	groupID := input.GroupID
	if groupID == "" {
		groupID = s.config.GroupID // Use server's default group ID
	}

	limit := input.LastN
	if limit <= 0 {
		limit = 10 // Default to 10 episodes
	}

	// Use the Graphiti client to retrieve episodes
	episodeNodes, err := s.client.GetEpisodes(context.Background(), groupID, limit)
	if err != nil {
		s.logger.Error("Failed to retrieve episodes", "error", err)
		return &ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to retrieve episodes: %v", err),
		}, nil
	}

	// Convert nodes to episode format
	var episodes []map[string]interface{}
	for _, node := range episodeNodes {
		episode := map[string]interface{}{
			"uuid":       node.ID,
			"name":       node.Name,
			"content":    node.Content,
			"group_id":   node.GroupID,
			"created_at": node.CreatedAt.Format(time.RFC3339),
		}

		// Add episode type if available
		if node.EpisodeType != "" {
			episode["episode_type"] = string(node.EpisodeType)
		}

		// Add reference time if available
		if !node.Reference.IsZero() {
			episode["reference"] = node.Reference.Format(time.RFC3339)
		}

		// Add metadata if available
		if node.Metadata != nil {
			episode["metadata"] = node.Metadata
		}

		episodes = append(episodes, episode)
	}

	s.logger.Info("Retrieved episodes", "count", len(episodes))

	return &ToolResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d episodes", len(episodes)),
		Data: map[string]interface{}{
			"episodes": episodes,
			"total":    len(episodes),
			"group_id": groupID,
		},
	}, nil
}

// ClearGraphTool handles clearing the entire graph
func (s *MCPServer) ClearGraphTool(ctx *ai.ToolContext, input *ClearGraphRequest) (*ToolResponse, error) {
	s.logger.Info("Clear graph requested", "group_id", input.GroupID, "confirm", input.Confirm)

	// Safety check - require explicit confirmation
	if !input.Confirm {
		return &ToolResponse{
			Success: false,
			Error:   "Graph clearing requires explicit confirmation. Set 'confirm' to true to proceed.",
		}, nil
	}

	// Set default group ID
	groupID := input.GroupID
	if groupID == "" {
		groupID = s.config.GroupID // Use server's default group ID
	}

	// Warn about the destructive operation
	s.logger.Warn("Clearing all data from graph", "group_id", groupID)

	// Use the Graphiti client to clear the graph
	err := s.client.ClearGraph(context.Background(), groupID)
	if err != nil {
		s.logger.Error("Failed to clear graph", "error", err, "group_id", groupID)
		return &ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to clear graph: %v", err),
		}, nil
	}

	s.logger.Info("Graph cleared successfully", "group_id", groupID)

	return &ToolResponse{
		Success: true,
		Message: fmt.Sprintf("Graph cleared successfully for group '%s'", groupID),
		Data: map[string]interface{}{
			"group_id": groupID,
			"cleared":  true,
		},
	}, nil
}