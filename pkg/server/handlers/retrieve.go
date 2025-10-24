package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/soundprediction/go-graphiti"
	"github.com/soundprediction/go-graphiti/pkg/server/dto"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// RetrieveHandler handles data retrieval requests
type RetrieveHandler struct {
	graphiti graphiti.Graphiti
}

// NewRetrieveHandler creates a new retrieve handler
func NewRetrieveHandler(g graphiti.Graphiti) *RetrieveHandler {
	return &RetrieveHandler{
		graphiti: g,
	}
}

// Search handles POST /search
func (h *RetrieveHandler) Search(c *gin.Context) {
	var req dto.SearchQuery
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Query) == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "query field is required and cannot be empty",
		})
		return
	}

	ctx := context.Background()

	// Set default max facts if not provided
	if req.MaxFacts <= 0 {
		req.MaxFacts = 10
	}

	// Create search configuration
	searchConfig := &types.SearchConfig{
		Limit:        req.MaxFacts,
		MinScore:     0.0,
		IncludeEdges: true,
		Rerank:       true,
	}

	// Perform the search using graphiti
	// Note: Group ID filtering would be implemented in the search configuration
	// For now, we use the global search
	searchResults, err := h.graphiti.Search(ctx, req.Query, searchConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "search_failed",
			Message: err.Error(),
		})
		return
	}

	// Convert graphiti search results to DTO format
	var facts []dto.FactResult

	// Process nodes as facts
	for _, node := range searchResults.Nodes {
		fact := dto.FactResult{
			UUID:         node.ID,
			Fact:         h.nodeToFactDescription(node),
			SourceName:   node.Name,
			TargetName:   "", // Nodes don't have targets
			RelationType: string(node.Type),
			CreatedAt:    node.CreatedAt,
			ValidAt:      &node.ValidFrom,
		}

		if node.ValidTo != nil {
			fact.InvalidAt = node.ValidTo
		}

		facts = append(facts, fact)
	}

	// Process edges as facts
	for _, edge := range searchResults.Edges {
		fact := dto.FactResult{
			UUID:         edge.ID,
			Fact:         h.edgeToFactDescription(edge),
			SourceName:   edge.SourceID, // Could be enhanced to resolve actual names
			TargetName:   edge.TargetID,
			RelationType: string(edge.Type),
			CreatedAt:    edge.CreatedAt,
			ValidAt:      &edge.ValidFrom,
		}

		if edge.ValidTo != nil {
			fact.InvalidAt = edge.ValidTo
		}

		facts = append(facts, fact)
	}

	// Create response
	results := dto.SearchResults{
		Facts: facts,
		Total: len(facts),
	}

	c.JSON(http.StatusOK, results)
}

// GetEntityEdge handles GET /entity-edge/:uuid
func (h *RetrieveHandler) GetEntityEdge(c *gin.Context) {
	uuid := c.Param("uuid")
	if uuid == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "UUID parameter is required",
		})
		return
	}

	ctx := context.Background()

	// Try to retrieve the edge from graphiti
	edge, err := h.graphiti.GetEdge(ctx, uuid)
	if err != nil {
		// If edge not found, try as a node
		node, nodeErr := h.graphiti.GetNode(ctx, uuid)
		if nodeErr != nil {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "entity_not_found",
				Message: "Entity with the specified UUID was not found",
			})
			return
		}

		// Convert node to fact format
		fact := dto.FactResult{
			UUID:         node.ID,
			Fact:         h.nodeToFactDescription(node),
			SourceName:   node.Name,
			TargetName:   "",
			RelationType: string(node.Type),
			CreatedAt:    node.CreatedAt,
			ValidAt:      &node.ValidFrom,
		}

		if node.ValidTo != nil {
			fact.InvalidAt = node.ValidTo
		}

		c.JSON(http.StatusOK, fact)
		return
	}

	// Convert edge to fact format
	fact := dto.FactResult{
		UUID:         edge.ID,
		Fact:         h.edgeToFactDescription(edge),
		SourceName:   edge.SourceID, // Could be enhanced to resolve actual names
		TargetName:   edge.TargetID,
		RelationType: string(edge.Type),
		CreatedAt:    edge.CreatedAt,
		ValidAt:      &edge.ValidFrom,
	}

	if edge.ValidTo != nil {
		fact.InvalidAt = edge.ValidTo
	}

	c.JSON(http.StatusOK, fact)
}

// GetEpisodes handles GET /episodes/:group_id
func (h *RetrieveHandler) GetEpisodes(c *gin.Context) {
	groupID := c.Param("group_id")
	if groupID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "Group ID parameter is required",
		})
		return
	}

	// Parse query parameters
	lastNStr := c.DefaultQuery("last_n", "10")
	lastN, err := strconv.Atoi(lastNStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "last_n must be a valid integer",
		})
		return
	}

	// Ensure reasonable limits
	if lastN <= 0 {
		lastN = 10
	}
	if lastN > 100 {
		lastN = 100 // Cap at 100 for performance
	}

	ctx := context.Background()

	// Retrieve episodes from graphiti
	episodeNodes, err := h.graphiti.GetEpisodes(ctx, groupID, lastN)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	// Convert nodes to episode DTOs
	var episodes []dto.Episode
	for _, node := range episodeNodes {
		episode := dto.Episode{
			UUID:      node.ID,
			GroupID:   node.GroupID,
			Content:   node.Content,
			CreatedAt: node.CreatedAt,
		}

		// Add source information if available in metadata
		if node.Metadata != nil {
			if source, ok := node.Metadata["source"].(string); ok {
				episode.Source = source
			}
		}

		episodes = append(episodes, episode)
	}

	response := dto.GetEpisodesResponse{
		Episodes: episodes,
		Total:    len(episodes),
	}

	c.JSON(http.StatusOK, response)
}

// GetMemory handles POST /get-memory
func (h *RetrieveHandler) GetMemory(c *gin.Context) {
	var req dto.GetMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Validate required fields
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "messages array is required and cannot be empty",
		})
		return
	}

	ctx := context.Background()

	// Set default max facts if not provided
	if req.MaxFacts <= 0 {
		req.MaxFacts = 10
	}

	// Compose query from messages
	var queryParts []string
	for _, msg := range req.Messages {
		if strings.TrimSpace(msg.Content) != "" {
			queryParts = append(queryParts, msg.Content)
		}
	}

	if len(queryParts) == 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "at least one message must have non-empty content",
		})
		return
	}

	combinedQuery := strings.Join(queryParts, " ")

	// Create search configuration for memory retrieval
	searchConfig := &types.SearchConfig{
		Limit:        req.MaxFacts,
		MinScore:     0.1, // Slightly higher threshold for memory relevance
		IncludeEdges: true,
		Rerank:       true,
	}

	// Perform search using the combined query
	searchResults, err := h.graphiti.Search(ctx, combinedQuery, searchConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "memory_retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	// Convert search results to memory facts
	var facts []dto.FactResult

	// Process nodes as memory facts
	for _, node := range searchResults.Nodes {
		// Prioritize episodic nodes for memory retrieval
		fact := dto.FactResult{
			UUID:         node.ID,
			Fact:         h.nodeToFactDescription(node),
			SourceName:   node.Name,
			TargetName:   "",
			RelationType: string(node.Type),
			CreatedAt:    node.CreatedAt,
			ValidAt:      &node.ValidFrom,
		}

		if node.ValidTo != nil {
			fact.InvalidAt = node.ValidTo
		}

		facts = append(facts, fact)
	}

	// Process edges as relationship facts
	for _, edge := range searchResults.Edges {
		fact := dto.FactResult{
			UUID:         edge.ID,
			Fact:         h.edgeToFactDescription(edge),
			SourceName:   edge.SourceID,
			TargetName:   edge.TargetID,
			RelationType: string(edge.Type),
			CreatedAt:    edge.CreatedAt,
			ValidAt:      &edge.ValidFrom,
		}

		if edge.ValidTo != nil {
			fact.InvalidAt = edge.ValidTo
		}

		facts = append(facts, fact)
	}

	// Create response
	results := dto.GetMemoryResponse{
		Facts: facts,
		Total: len(facts),
	}

	c.JSON(http.StatusOK, results)
}

// Helper methods for converting graph entities to fact descriptions

// nodeToFactDescription converts a node to a human-readable fact description
func (h *RetrieveHandler) nodeToFactDescription(node *types.Node) string {
	if node.Summary != "" {
		return node.Summary
	}
	if node.Content != "" {
		return node.Content
	}
	return node.Name + " is a " + string(node.Type)
}

// edgeToFactDescription converts an edge to a human-readable fact description
func (h *RetrieveHandler) edgeToFactDescription(edge *types.Edge) string {
	if edge.Summary != "" {
		return edge.Summary
	}
	if edge.Name != "" {
		return edge.Name
	}
	return edge.SourceID + " " + string(edge.Type) + " " + edge.TargetID
}
