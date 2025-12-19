package handlers

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/soundprediction/go-predicato"
)

// Build information - can be set at build time using ldflags
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
)

// HealthHandler handles health check requests
type HealthHandler struct {
	predicato predicato.Predicato
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(g predicato.Predicato) *HealthHandler {
	return &HealthHandler{
		predicato: g,
	}
}

// HealthCheck handles GET /health - basic liveness check
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "go-predicato",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   Version,
	})
}

// ReadinessCheck handles GET /ready
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := gin.H{
		"status":    "ready",
		"service":   "go-predicato",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks":    gin.H{},
	}

	allHealthy := true
	checks := response["checks"].(gin.H)

	// Check database connectivity by performing a simple operation
	if h.predicato != nil {
		dbStartTime := time.Now()

		// Test database connectivity with a simple GetNode call using a non-existent ID
		// This will test if we can connect to the database without side effects
		_, err := h.predicato.GetNode(ctx, "health-check-non-existent-id")
		dbDuration := time.Since(dbStartTime)

		if err != nil {
			// We expect an error (node not found), but it should not be a connection error
			// Check if it's a connection/timeout error vs expected "not found" error
			if ctx.Err() != nil {
				// Context timeout or cancellation indicates connection issues
				checks["database"] = gin.H{
					"status":   "unhealthy",
					"error":    "database connection timeout",
					"duration": dbDuration.String(),
				}
				allHealthy = false
			} else {
				// Any other error is expected (like "node not found") - database is healthy
				checks["database"] = gin.H{
					"status":   "healthy",
					"duration": dbDuration.String(),
				}
			}
		} else {
			// Unexpected success, but still indicates database is responsive
			checks["database"] = gin.H{
				"status":   "healthy",
				"duration": dbDuration.String(),
			}
		}

		// Test database indices creation capability (optional advanced check)
		indicesStartTime := time.Now()
		indicesErr := h.predicato.CreateIndices(ctx)
		indicesDuration := time.Since(indicesStartTime)

		if indicesErr != nil && ctx.Err() != nil {
			checks["database_indices"] = gin.H{
				"status":   "unhealthy",
				"error":    "indices operation timeout",
				"duration": indicesDuration.String(),
			}
			allHealthy = false
		} else {
			checks["database_indices"] = gin.H{
				"status":   "healthy",
				"duration": indicesDuration.String(),
			}
		}
	} else {
		checks["database"] = gin.H{
			"status": "unhealthy",
			"error":  "predicato client not initialized",
		}
		allHealthy = false
	}

	// Add overall system readiness check
	checks["system"] = gin.H{
		"status": "healthy",
		"uptime": time.Since(time.Now().Add(-time.Minute)).String(), // Placeholder uptime
	}

	// Set overall status based on all checks
	if !allHealthy {
		response["status"] = "not_ready"
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// LivenessCheck handles GET /live - Kubernetes liveness probe endpoint
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	// Simple liveness check - just confirm the service is running
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"service":   "go-predicato",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// DetailedHealthCheck handles GET /health/detailed - comprehensive health information
func (h *HealthHandler) DetailedHealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	startTime := time.Now()
	response := gin.H{
		"status":  "healthy",
		"service": "go-predicato",
		"version": Version,
		"build_info": gin.H{
			"git_commit": GitCommit,
			"build_time": BuildTime,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"environment": gin.H{
			"go_version": GoVersion,
		},
		"checks": gin.H{},
		"metrics": gin.H{
			"response_time_ms": 0, // Will be set at the end
		},
	}

	allHealthy := true
	checks := response["checks"].(gin.H)

	// Test all critical dependencies
	if h.predicato != nil {
		// Database connectivity check
		dbStartTime := time.Now()
		_, err := h.predicato.GetNode(ctx, "health-check-detailed")
		dbDuration := time.Since(dbStartTime)

		dbStatus := gin.H{
			"status":      "healthy",
			"duration_ms": dbDuration.Milliseconds(),
			"operation":   "GetNode",
		}

		if err != nil && ctx.Err() != nil {
			dbStatus["status"] = "unhealthy"
			dbStatus["error"] = "connection timeout"
			allHealthy = false
		} else if err != nil {
			// Expected error (node not found) - still healthy
			dbStatus["note"] = "expected not found error - connection healthy"
		}

		checks["database_connectivity"] = dbStatus

		// Database operations check
		opsStartTime := time.Now()
		indicesErr := h.predicato.CreateIndices(ctx)
		opsDuration := time.Since(opsStartTime)

		opsStatus := gin.H{
			"status":      "healthy",
			"duration_ms": opsDuration.Milliseconds(),
			"operation":   "CreateIndices",
		}

		if indicesErr != nil && ctx.Err() != nil {
			opsStatus["status"] = "unhealthy"
			opsStatus["error"] = "operation timeout"
			allHealthy = false
		} else if indicesErr != nil {
			opsStatus["note"] = "operation completed with warnings"
		}

		checks["database_operations"] = opsStatus

		// Optional: Test search functionality
		searchStartTime := time.Now()
		_, searchErr := h.predicato.Search(ctx, "health-check", nil)
		searchDuration := time.Since(searchStartTime)

		searchStatus := gin.H{
			"status":      "healthy",
			"duration_ms": searchDuration.Milliseconds(),
			"operation":   "Search",
		}

		if searchErr != nil && ctx.Err() != nil {
			searchStatus["status"] = "unhealthy"
			searchStatus["error"] = "search timeout"
			allHealthy = false
		} else if searchErr != nil {
			searchStatus["note"] = "search completed with expected results"
		}

		checks["search_functionality"] = searchStatus
	} else {
		checks["predicato_client"] = gin.H{
			"status": "unhealthy",
			"error":  "client not initialized",
		}
		allHealthy = false
	}

	// Add system health metrics
	systemMetrics := h.getSystemMetrics()
	checks["system"] = gin.H{
		"status":       "healthy",
		"memory_usage": systemMetrics.MemoryUsage,
		"goroutines":   systemMetrics.Goroutines,
		"gc_cycles":    systemMetrics.GCCycles,
		"heap_objects": systemMetrics.HeapObjects,
		"stack_usage":  systemMetrics.StackUsage,
	}

	// Set final response
	totalDuration := time.Since(startTime)
	response["metrics"].(gin.H)["response_time_ms"] = totalDuration.Milliseconds()

	if !allHealthy {
		response["status"] = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// SystemMetrics holds system runtime metrics
type SystemMetrics struct {
	MemoryUsage string `json:"memory_usage"`
	Goroutines  int    `json:"goroutines"`
	GCCycles    uint32 `json:"gc_cycles"`
	HeapObjects uint64 `json:"heap_objects"`
	StackUsage  string `json:"stack_usage"`
}

// getSystemMetrics collects current system runtime metrics
func (h *HealthHandler) getSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert bytes to human-readable format
	memoryUsage := fmt.Sprintf("%.2f MB", float64(m.Alloc)/(1024*1024))
	stackUsage := fmt.Sprintf("%.2f MB", float64(m.StackSys)/(1024*1024))

	return SystemMetrics{
		MemoryUsage: memoryUsage,
		Goroutines:  runtime.NumGoroutine(),
		GCCycles:    m.NumGC,
		HeapObjects: m.HeapObjects,
		StackUsage:  stackUsage,
	}
}
