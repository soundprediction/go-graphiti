package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/soundprediction/go-graphiti/pkg/config"
	"github.com/soundprediction/go-graphiti/pkg/server/handlers"
	"github.com/soundprediction/go-graphiti"
)

// Server represents the HTTP server
type Server struct {
	config   *config.Config
	router   *gin.Engine
	graphiti graphiti.Graphiti
	server   *http.Server
}

// New creates a new server instance
func New(cfg *config.Config, graphitiClient graphiti.Graphiti) *Server {
	return &Server{
		config:   cfg,
		graphiti: graphitiClient,
	}
}

// Setup sets up the server routes and middleware
func (s *Server) Setup() {
	// Set gin mode
	gin.SetMode(s.config.Server.Mode)
	
	// Create router
	s.router = gin.New()
	
	// Add middleware
	s.router.Use(gin.Logger())
	s.router.Use(gin.Recovery())
	s.router.Use(corsMiddleware())
	
	// Setup routes
	s.setupRoutes()
	
	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
}

// setupRoutes sets up all the routes
func (s *Server) setupRoutes() {
	// Create handlers
	healthHandler := handlers.NewHealthHandler(s.graphiti)
	ingestHandler := handlers.NewIngestHandler(s.graphiti)
	retrieveHandler := handlers.NewRetrieveHandler(s.graphiti)
	
	// Health endpoints
	s.router.GET("/health", healthHandler.HealthCheck)
	s.router.GET("/healthcheck", healthHandler.HealthCheck) // Legacy endpoint
	s.router.GET("/ready", healthHandler.ReadinessCheck)
	s.router.GET("/live", healthHandler.LivenessCheck)       // Kubernetes liveness probe
	s.router.GET("/health/detailed", healthHandler.DetailedHealthCheck)
	
	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Ingest routes
		ingest := v1.Group("/ingest")
		{
			ingest.POST("/messages", ingestHandler.AddMessages)
			ingest.POST("/entity", ingestHandler.AddEntityNode)
			ingest.DELETE("/clear", ingestHandler.ClearData)
		}
		
		// Retrieve routes
		v1.POST("/search", retrieveHandler.Search)
		v1.GET("/entity-edge/:uuid", retrieveHandler.GetEntityEdge)
		v1.GET("/episodes/:group_id", retrieveHandler.GetEpisodes)
		v1.POST("/get-memory", retrieveHandler.GetMemory)
	}
	
	// Legacy routes for compatibility with Python server
	s.router.POST("/search", retrieveHandler.Search)
	s.router.GET("/entity-edge/:uuid", retrieveHandler.GetEntityEdge)
	s.router.GET("/episodes/:group_id", retrieveHandler.GetEpisodes)
	s.router.POST("/get-memory", retrieveHandler.GetMemory)
}

// Start starts the server
func (s *Server) Start() error {
	fmt.Printf("Starting server on %s\n", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop stops the server gracefully
func (s *Server) Stop(ctx context.Context) error {
	fmt.Println("Stopping server...")
	return s.server.Shutdown(ctx)
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}