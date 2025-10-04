package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/soundprediction/go-graphiti/pkg/server/dto"
	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/soundprediction/go-graphiti"
)

// IngestHandler handles data ingestion requests
type IngestHandler struct {
	graphiti graphiti.Graphiti
}

// NewIngestHandler creates a new ingest handler
func NewIngestHandler(g graphiti.Graphiti) *IngestHandler {
	return &IngestHandler{
		graphiti: g,
	}
}

// generateProcessID generates a unique process ID for tracking async operations
func generateProcessID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random generation fails
		return fmt.Sprintf("proc_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("proc_%s", hex.EncodeToString(bytes))
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

	// Validate required fields
	if req.GroupID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "group_id is required",
		})
		return
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "messages array cannot be empty",
		})
		return
	}

	// Generate a process ID for tracking this async operation
	processID := generateProcessID()

	// Process messages asynchronously in the background
	go func() {
		ctx := context.Background()
		referenceTime := time.Now()
		if req.Reference != nil {
			referenceTime = *req.Reference
		}

		fmt.Printf("[%s] Starting processing of %d messages for group %s\n", processID, len(req.Messages), req.GroupID)

		// Convert messages to episodes and add them to graphiti
		var episodes []types.Episode
		for i, msg := range req.Messages {
			// Generate a unique ID for each episode
			episodeID := fmt.Sprintf("%s-msg-%d-%d", req.GroupID, referenceTime.Unix(), i)

			// Create episode name from role and timestamp
			episodeName := fmt.Sprintf("%s message at %s", msg.Role, referenceTime.Format("2006-01-02 15:04:05"))

			// Create episode content
			episodeContent := fmt.Sprintf("%s: %s", msg.Role, msg.Content)

			// Use message timestamp if provided, otherwise use reference time
			episodeTime := referenceTime
			if msg.Timestamp != nil {
				episodeTime = *msg.Timestamp
			}

			episode := types.Episode{
				ID:        episodeID,
				Name:      episodeName,
				Content:   episodeContent,
				Reference: episodeTime,
				CreatedAt: time.Now(),
				GroupID:   req.GroupID,
				Metadata: map[string]interface{}{
					"role":             msg.Role,
					"original_content": msg.Content,
					"source":           "api_ingest",
					"process_id":       processID,
				},
			}

			episodes = append(episodes, episode)
		}

		// Add episodes to graphiti
		if _, err := h.graphiti.Add(ctx, episodes, nil); err != nil {
			// Log error but don't fail the entire request since it's async
			fmt.Printf("[%s] Error adding episodes to graphiti for group %s: %v\n", processID, req.GroupID, err)
		} else {
			fmt.Printf("[%s] Successfully processed %d episodes for group %s\n", processID, len(episodes), req.GroupID)
		}
	}()

	c.JSON(http.StatusAccepted, dto.IngestResponse{
		Success:   true,
		Message:   fmt.Sprintf("Queued %d messages for processing", len(req.Messages)),
		ProcessID: processID,
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

	// Validate required fields
	if req.GroupID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "group_id is required",
		})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "name is required",
		})
		return
	}

	ctx := context.Background()

	// Create an episode that mentions this entity to add it to the knowledge graph
	// This leverages graphiti's entity extraction capabilities
	now := time.Now()
	episodeID := fmt.Sprintf("%s-entity-%d", req.GroupID, now.Unix())

	// Create episode content that mentions the entity
	entityType := req.EntityType
	if entityType == "" {
		entityType = "entity"
	}

	episodeContent := fmt.Sprintf("New %s entity: %s", entityType, req.Name)
	if len(req.Attributes) > 0 {
		episodeContent += fmt.Sprintf(" with attributes: %v", req.Attributes)
	}

	// Create metadata that includes the entity information
	metadata := map[string]interface{}{
		"source":      "api_entity_ingest",
		"entity_name": req.Name,
		"entity_type": entityType,
	}

	// Add request attributes to metadata
	if req.Attributes != nil {
		for key, value := range req.Attributes {
			metadata["attr_"+key] = value
		}
	}

	episode := types.Episode{
		ID:        episodeID,
		Name:      fmt.Sprintf("Entity creation: %s", req.Name),
		Content:   episodeContent,
		Reference: now,
		CreatedAt: now,
		GroupID:   req.GroupID,
		Metadata:  metadata,
	}

	// Add the episode to graphiti which will extract and create the entity
	if _, err := h.graphiti.Add(ctx, []types.Episode{episode}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "creation_failed",
			Message: fmt.Sprintf("Failed to create entity node: %v", err),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.IngestResponse{
		Success: true,
		Message: fmt.Sprintf("Entity node '%s' created via episode processing", req.Name),
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

	ctx := context.Background()

	// If no specific group IDs provided, this is a dangerous operation
	if len(req.GroupIDs) == 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "group_ids must be specified for data clearing. Clearing all data is not supported via API for safety.",
		})
		return
	}

	// Process clearing for each specified group
	var successGroups []string
	var failedGroups []string

	for _, groupID := range req.GroupIDs {
		if groupID == "" {
			failedGroups = append(failedGroups, "(empty group ID)")
			continue
		}

		// Use graphiti's ClearGraph method to clear data for this group
		if err := h.graphiti.ClearGraph(ctx, groupID); err != nil {
			fmt.Printf("Error clearing data for group %s: %v\n", groupID, err)
			failedGroups = append(failedGroups, groupID)
		} else {
			fmt.Printf("Successfully cleared data for group %s\n", groupID)
			successGroups = append(successGroups, groupID)
		}
	}

	// Prepare response message
	var message string
	var success bool
	var statusCode int

	if len(failedGroups) == 0 {
		// All groups cleared successfully
		message = fmt.Sprintf("Successfully cleared data for groups: %v", successGroups)
		success = true
		statusCode = http.StatusOK
	} else if len(successGroups) == 0 {
		// All groups failed to clear
		message = fmt.Sprintf("Failed to clear data for all groups: %v", failedGroups)
		success = false
		statusCode = http.StatusInternalServerError
	} else {
		// Partial success
		message = fmt.Sprintf("Partially cleared data. Success: %v, Failed: %v", successGroups, failedGroups)
		success = true // Consider partial success as success
		statusCode = http.StatusOK
	}

	if !success {
		c.JSON(statusCode, dto.ErrorResponse{
			Error:   "clear_failed",
			Message: message,
		})
		return
	}

	c.JSON(statusCode, dto.IngestResponse{
		Success: success,
		Message: message,
	})
}