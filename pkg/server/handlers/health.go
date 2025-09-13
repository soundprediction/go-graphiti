package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthCheck handles GET /health
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "go-graphiti",
	})
}

// ReadinessCheck handles GET /ready
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// TODO: Add actual readiness checks (database connectivity, etc.)
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"service": "go-graphiti",
	})
}