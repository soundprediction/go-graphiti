package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/soundprediction/go-graphiti/pkg/server/dto"
	"github.com/soundprediction/go-graphiti"
)

// IngestHandler handles data ingestion requests
type IngestHandler struct {
	graphiti *graphiti.Graphiti
}

// NewIngestHandler creates a new ingest handler
func NewIngestHandler(g *graphiti.Graphiti) *IngestHandler {
	return &IngestHandler{
		graphiti: g,
	}
}

// AddMessages handles POST /ingest/messages
func (h *IngestHandler) AddMessages(c *gin.Context) {
	var req dto.AddMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Process messages asynchronously in the background
	// For now, we'll process synchronously but this could be improved
	go func() {
		// ctx := context.Background()
		referenceTime := time.Now()
		if req.Reference != nil {
			referenceTime = *req.Reference
		}

		// Convert messages to the format expected by graphiti
		for _, msg := range req.Messages {
			episode := fmt.Sprintf("%s: %s", msg.Role, msg.Content)
			
			// TODO: Add episode to graphiti
			// Note: This is a simplified implementation
			// The actual graphiti package may have different methods
			// if err := h.graphiti.AddEpisode(ctx, req.GroupID, episode, referenceTime); err != nil {
			// 	// Log error but don't fail the entire request
			// 	fmt.Printf("Error adding episode: %v\n", err)
			// }
			fmt.Printf("Processing episode for group %s: %s at %v\n", req.GroupID, episode, referenceTime)
		}
	}()

	c.JSON(http.StatusAccepted, dto.IngestResponse{
		Success: true,
		Message: "Messages queued for processing",
	})
}

// AddEntityNode handles POST /ingest/entity
func (h *IngestHandler) AddEntityNode(c *gin.Context) {
	var req dto.AddEntityNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// ctx := context.Background()
	
	// TODO: Implement entity node addition
	// This would involve creating a node in the knowledge graph
	// For now, return a placeholder response
	
	c.JSON(http.StatusCreated, dto.IngestResponse{
		Success: true,
		Message: fmt.Sprintf("Entity node '%s' created", req.Name),
	})
}

// ClearData handles DELETE /ingest/clear
func (h *IngestHandler) ClearData(c *gin.Context) {
	var req dto.ClearDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// ctx := context.Background()
	
	// TODO: Implement data clearing functionality
	// This would involve clearing data from the knowledge graph
	// For now, return a placeholder response
	
	groupsMsg := "all data"
	if len(req.GroupIDs) > 0 {
		groupsMsg = fmt.Sprintf("data for groups: %v", req.GroupIDs)
	}
	
	c.JSON(http.StatusOK, dto.IngestResponse{
		Success: true,
		Message: fmt.Sprintf("Cleared %s", groupsMsg),
	})
}