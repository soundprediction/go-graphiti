package graphiti

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soundprediction/go-graphiti"
	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Default configuration values for MCP server
const (
	DefaultMCPLLMModel      = "gpt-4o-mini"
	DefaultMCPSmallModel    = "gpt-4o-mini"
	DefaultMCPEmbedderModel = "text-embedding-3-small"
	DefaultMCPSemaphoreLimit = 10
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Model Context Protocol (MCP) server",
	Long: `Start the Model Context Protocol (MCP) server to provide MCP tool access to the knowledge graph.

The MCP server provides tools for:
- Adding episodes/memories to the knowledge graph
- Searching nodes and facts in the graph
- Managing entities and episodes
- Clearing graph data

The server can communicate over stdio or HTTP/SSE transport protocols and is designed
to work with MCP clients like Claude Desktop or other compatible applications.`,
	RunE: runMCPServer,
}

var (
	mcpGroupID           string
	mcpTransport         string
	mcpHost             string
	mcpPort             int
	mcpModel            string
	mcpSmallModel       string
	mcpTemperature      float64
	mcpUseCustomEntities bool
	mcpDestroyGraph     bool
	mcpSemaphoreLimit   int
)

func init() {
	rootCmd.AddCommand(mcpCmd)

	// Configure viper to automatically check for environment variables
	viper.AutomaticEnv()

	// Set up specific environment variable bindings to maintain compatibility
	// with existing environment variable names
	viper.BindEnv("llm.api_key", "OPENAI_API_KEY")
	viper.BindEnv("llm.base_url", "LLM_BASE_URL")
	viper.BindEnv("embedder.api_key", "EMBEDDING_API_KEY", "OPENAI_API_KEY") // Fallback to OpenAI key
	viper.BindEnv("embedder.base_url", "EMBEDDING_BASE_URL")
	viper.BindEnv("embedder.model", "EMBEDDER_MODEL_NAME")
	viper.BindEnv("database.uri", "NEO4J_URI")
	viper.BindEnv("database.username", "NEO4J_USER")
	viper.BindEnv("database.password", "NEO4J_PASSWORD")
	viper.BindEnv("database.database", "NEO4J_DATABASE")
	viper.BindEnv("mcp.group_id", "GROUP_ID")
	viper.BindEnv("mcp.transport", "MCP_TRANSPORT")
	viper.BindEnv("mcp.host", "MCP_HOST")
	viper.BindEnv("mcp.port", "MCP_PORT")
	viper.BindEnv("mcp.model", "MODEL_NAME")
	viper.BindEnv("mcp.small_model", "SMALL_MODEL_NAME")
	viper.BindEnv("mcp.temperature", "LLM_TEMPERATURE")
	viper.BindEnv("mcp.use_custom_entities", "USE_CUSTOM_ENTITIES")
	viper.BindEnv("mcp.destroy_graph", "DESTROY_GRAPH")
	viper.BindEnv("mcp.semaphore_limit", "SEMAPHORE_LIMIT")

	// MCP Server specific flags
	mcpCmd.Flags().StringVar(&mcpGroupID, "group-id", "default", "Namespace for the graph")
	mcpCmd.Flags().StringVar(&mcpTransport, "transport", "stdio", "Transport to use (stdio or sse)")
	mcpCmd.Flags().StringVar(&mcpHost, "host", "localhost", "Host to bind the MCP server to")
	mcpCmd.Flags().IntVar(&mcpPort, "port", 3000, "Port to bind the MCP server to")
	mcpCmd.Flags().StringVar(&mcpModel, "model", DefaultMCPLLMModel, "LLM model name")
	mcpCmd.Flags().StringVar(&mcpSmallModel, "small-model", DefaultMCPSmallModel, "Small LLM model name")
	mcpCmd.Flags().Float64Var(&mcpTemperature, "temperature", 0.0, "Temperature setting for the LLM (0.0-2.0)")
	mcpCmd.Flags().BoolVar(&mcpUseCustomEntities, "use-custom-entities", false, "Enable entity extraction using predefined entity types")
	mcpCmd.Flags().BoolVar(&mcpDestroyGraph, "destroy-graph", false, "Destroy all Graphiti graphs on startup")
	mcpCmd.Flags().IntVar(&mcpSemaphoreLimit, "semaphore-limit", DefaultMCPSemaphoreLimit, "Concurrency limit for operations")

	// Database flags
	mcpCmd.Flags().String("db-uri", "bolt://localhost:7687", "Database URI")
	mcpCmd.Flags().String("db-username", "neo4j", "Database username")
	mcpCmd.Flags().String("db-password", "password", "Database password")
	mcpCmd.Flags().String("db-database", "neo4j", "Database name")

	// LLM flags
	mcpCmd.Flags().String("llm-api-key", "", "OpenAI API key")
	mcpCmd.Flags().String("llm-base-url", "", "LLM base URL (for OpenAI-compatible services)")

	// Embedding flags
	mcpCmd.Flags().String("embedder-model", DefaultMCPEmbedderModel, "Embedding model name")
	mcpCmd.Flags().String("embedding-api-key", "", "Embedding API key")
	mcpCmd.Flags().String("embedding-base-url", "", "Embedding base URL")

	// Bind flags to viper for configuration
	viper.BindPFlag("mcp.group_id", mcpCmd.Flags().Lookup("group-id"))
	viper.BindPFlag("mcp.transport", mcpCmd.Flags().Lookup("transport"))
	viper.BindPFlag("mcp.host", mcpCmd.Flags().Lookup("host"))
	viper.BindPFlag("mcp.port", mcpCmd.Flags().Lookup("port"))
	viper.BindPFlag("mcp.model", mcpCmd.Flags().Lookup("model"))
	viper.BindPFlag("mcp.small_model", mcpCmd.Flags().Lookup("small-model"))
	viper.BindPFlag("mcp.temperature", mcpCmd.Flags().Lookup("temperature"))
	viper.BindPFlag("mcp.use_custom_entities", mcpCmd.Flags().Lookup("use-custom-entities"))
	viper.BindPFlag("mcp.destroy_graph", mcpCmd.Flags().Lookup("destroy-graph"))
	viper.BindPFlag("mcp.semaphore_limit", mcpCmd.Flags().Lookup("semaphore-limit"))

	// Database configuration
	viper.BindPFlag("database.uri", mcpCmd.Flags().Lookup("db-uri"))
	viper.BindPFlag("database.username", mcpCmd.Flags().Lookup("db-username"))
	viper.BindPFlag("database.password", mcpCmd.Flags().Lookup("db-password"))
	viper.BindPFlag("database.database", mcpCmd.Flags().Lookup("db-database"))

	// LLM configuration
	viper.BindPFlag("llm.api_key", mcpCmd.Flags().Lookup("llm-api-key"))
	viper.BindPFlag("llm.base_url", mcpCmd.Flags().Lookup("llm-base-url"))

	// Embedder configuration
	viper.BindPFlag("embedder.model", mcpCmd.Flags().Lookup("embedder-model"))
	viper.BindPFlag("embedder.api_key", mcpCmd.Flags().Lookup("embedding-api-key"))
	viper.BindPFlag("embedder.base_url", mcpCmd.Flags().Lookup("embedding-base-url"))
}

// MCPConfig holds all configuration for the MCP server
type MCPConfig struct {
	// LLM Configuration
	LLMModel         string
	SmallLLMModel    string
	LLMTemperature   float64
	OpenAIAPIKey     string
	LLMBaseURL      string

	// Embedder Configuration
	EmbedderModel    string
	EmbeddingAPIKey  string
	EmbeddingBaseURL string

	// Database Configuration
	Neo4jURI         string
	Neo4jUser        string
	Neo4jPassword    string
	Neo4jDatabase    string

	// MCP Server Configuration
	GroupID          string
	UseCustomEntities bool
	DestroyGraph     bool
	Transport        string
	Host             string
	Port             int

	// Concurrency limits
	SemaphoreLimit   int
}

// MCPServer wraps the Graphiti client for MCP operations
type MCPServer struct {
	config  *MCPConfig
	client  *graphiti.Client
	logger  *slog.Logger
}

// EntityTypes represents custom entity types for extraction
var EntityTypes = map[string]interface{}{
	"Requirement": struct {
		ProjectName string `json:"project_name" description:"The name of the project to which the requirement belongs."`
		Description string `json:"description" description:"Description of the requirement. Only use information mentioned in the context to write this description."`
	}{},
	"Preference": struct {
		Category    string `json:"category" description:"The category of the preference. (e.g., 'Brands', 'Food', 'Music')"`
		Description string `json:"description" description:"Brief description of the preference. Only use information mentioned in the context to write this description."`
	}{},
	"Procedure": struct {
		Description string `json:"description" description:"Brief description of the procedure. Only use information mentioned in the context to write this description."`
	}{},
}

func runMCPServer(cmd *cobra.Command, args []string) error {
	// Create configuration using viper (supports config files, env vars, and flags)
	config := &MCPConfig{
		// MCP Server configuration
		GroupID:          getViperStringWithFallback("mcp.group_id", mcpGroupID),
		Transport:        getViperStringWithFallback("mcp.transport", mcpTransport),
		Host:             getViperStringWithFallback("mcp.host", mcpHost),
		Port:             getViperIntWithFallback("mcp.port", mcpPort),
		LLMModel:         getViperStringWithFallback("mcp.model", mcpModel),
		SmallLLMModel:    getViperStringWithFallback("mcp.small_model", mcpSmallModel),
		LLMTemperature:   getViperFloat64WithFallback("mcp.temperature", mcpTemperature),
		UseCustomEntities: getViperBoolWithFallback("mcp.use_custom_entities", mcpUseCustomEntities),
		DestroyGraph:     getViperBoolWithFallback("mcp.destroy_graph", mcpDestroyGraph),
		SemaphoreLimit:   getViperIntWithFallback("mcp.semaphore_limit", mcpSemaphoreLimit),

		// Database configuration - viper handles env vars automatically
		Neo4jURI:      getViperStringWithFallback("database.uri", "bolt://localhost:7687"),
		Neo4jUser:     getViperStringWithFallback("database.username", "neo4j"),
		Neo4jPassword: getViperStringWithFallback("database.password", "password"),
		Neo4jDatabase: getViperStringWithFallback("database.database", "neo4j"),

		// LLM configuration - now optional
		OpenAIAPIKey:  viper.GetString("llm.api_key"), // No fallback - truly optional
		LLMBaseURL:    viper.GetString("llm.base_url"),

		// Embedder configuration
		EmbedderModel:    getViperStringWithFallback("embedder.model", DefaultMCPEmbedderModel),
		EmbeddingAPIKey:  viper.GetString("embedder.api_key"),    // No fallback - truly optional
		EmbeddingBaseURL: viper.GetString("embedder.base_url"),
	}

	// Use LLM API key for embeddings if embedding API key not provided
	if config.EmbeddingAPIKey == "" {
		config.EmbeddingAPIKey = config.OpenAIAPIKey
	}

	// Validate required configuration
	if err := validateMCPConfig(config); err != nil {
		return fmt.Errorf("invalid MCP configuration: %w", err)
	}

	// Create MCP server
	server, err := NewMCPServer(config)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Initialize server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		serverErrChan <- server.Run(ctx)
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrChan:
		if err != nil && err != context.Canceled {
			return fmt.Errorf("MCP server error: %w", err)
		}
		return nil
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal: %v\n", sig)
		cancel()

		// Give server time to shutdown gracefully
		select {
		case <-time.After(10 * time.Second):
			return fmt.Errorf("server shutdown timeout")
		case <-serverErrChan:
			fmt.Println("MCP server stopped gracefully")
			return nil
		}
	}
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(config *MCPConfig) (*MCPServer, error) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create Neo4j driver
	neo4jDriver, err := driver.NewNeo4jDriver(
		config.Neo4jURI,
		config.Neo4jUser,
		config.Neo4jPassword,
		config.Neo4jDatabase,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Create LLM client - only if we have an API key or base URL
	var llmClient llm.Client
	if config.OpenAIAPIKey != "" || config.LLMBaseURL != "" {
		llmConfig := llm.Config{
			Model:       config.LLMModel,
			Temperature: &[]float32{float32(config.LLMTemperature)}[0],
			BaseURL:     config.LLMBaseURL,
		}
		// Use empty string as API key if only base URL is provided (for services that don't require auth)
		apiKey := config.OpenAIAPIKey
		if apiKey == "" && config.LLMBaseURL != "" {
			apiKey = "dummy" // Some OpenAI-compatible services require a non-empty key
		}
		llmClient, err = llm.NewOpenAIClient(apiKey, llmConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create LLM client: %w", err)
		}
	} else {
		logger.Warn("No LLM configuration provided - LLM functionality will be disabled")
	}

	// Create embedder client - only if we have an API key or base URL
	var embedderClient embedder.Client
	if config.EmbeddingAPIKey != "" || config.EmbeddingBaseURL != "" {
		embedderConfig := embedder.Config{
			Model:   config.EmbedderModel,
			BaseURL: config.EmbeddingBaseURL,
		}
		// Use empty string as API key if only base URL is provided
		apiKey := config.EmbeddingAPIKey
		if apiKey == "" && config.EmbeddingBaseURL != "" {
			apiKey = "dummy"
		}
		embedderClient = embedder.NewOpenAIEmbedder(apiKey, embedderConfig)
	} else {
		logger.Warn("No embedder configuration provided - embedding functionality will be disabled")
	}

	// Create Graphiti client
	graphitiConfig := &graphiti.Config{
		GroupID:  config.GroupID,
		TimeZone: time.UTC,
	}

	client := graphiti.NewClient(neo4jDriver, llmClient, embedderClient, graphitiConfig)

	return &MCPServer{
		config: config,
		client: client,
		logger: logger,
	}, nil
}

// Initialize sets up the MCP server and Graphiti client
func (s *MCPServer) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing Graphiti MCP server...")

	// Verify the client is ready
	if s.client == nil {
		return fmt.Errorf("graphiti client not initialized")
	}

	// Clear graph if requested
	if s.config.DestroyGraph {
		s.logger.Info("Graph destruction requested - this would clear all data")
		// TODO: Implement graph clearing functionality when available
		// For now, just log the intent
	}

	s.logger.Info("Graphiti client initialized successfully")
	s.logger.Info("MCP server configuration",
		"llm_model", s.config.LLMModel,
		"temperature", s.config.LLMTemperature,
		"group_id", s.config.GroupID,
		"transport", s.config.Transport,
		"custom_entities", s.config.UseCustomEntities,
		"semaphore_limit", s.config.SemaphoreLimit,
	)

	return nil
}

// RegisterTools registers all MCP tools
func (s *MCPServer) RegisterTools() error {
	// TODO: Implement actual MCP tool registrations
	// For now, we'll log the available tools that would be registered

	tools := []string{
		"add_memory",
		"search_memory_nodes",
		"search_memory_facts",
		"get_episodes",
		"clear_graph",
	}

	s.logger.Info("Registering MCP tools", "tools", tools)
	return nil
}

// Run starts the MCP server
func (s *MCPServer) Run(ctx context.Context) error {
	s.logger.Info("Starting MCP server", "transport", s.config.Transport)

	// Register all tools
	if err := s.RegisterTools(); err != nil {
		return fmt.Errorf("failed to register MCP tools: %w", err)
	}

	s.logger.Info("MCP server is ready to accept requests")

	// TODO: Implement actual MCP protocol handling based on transport
	if s.config.Transport == "stdio" {
		s.logger.Info("MCP server would handle stdio transport here")
	} else if s.config.Transport == "sse" {
		s.logger.Info("MCP server would start HTTP server", "host", s.config.Host, "port", s.config.Port)
	}

	// Keep the server running
	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Tool implementations would be added here when actual MCP protocol is implemented
// For now, the server provides a framework for adding MCP functionality

// Helper functions for configuration
func validateMCPConfig(config *MCPConfig) error {
	if config.GroupID == "" {
		return fmt.Errorf("group ID is required")
	}

	if config.Neo4jURI == "" {
		return fmt.Errorf("Neo4j URI is required")
	}

	// Only require API key if custom entities are enabled AND no base URL is provided
	// This allows for OpenAI-compatible services that might not need API keys or use different auth
	if config.UseCustomEntities && config.OpenAIAPIKey == "" && config.LLMBaseURL == "" {
		return fmt.Errorf("LLM API key is required when custom entities are enabled, unless using a custom base URL")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}

	return nil
}

// Viper helper functions with fallback support
func getViperStringWithFallback(key, fallback string) string {
	if viper.IsSet(key) {
		return viper.GetString(key)
	}
	return fallback
}

func getViperIntWithFallback(key string, fallback int) int {
	if viper.IsSet(key) {
		return viper.GetInt(key)
	}
	return fallback
}

func getViperFloat64WithFallback(key string, fallback float64) float64 {
	if viper.IsSet(key) {
		return viper.GetFloat64(key)
	}
	return fallback
}

func getViperBoolWithFallback(key string, fallback bool) bool {
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return fallback
}

func getStringFlagOrEnv(cmd *cobra.Command, flagName, envName, defaultValue string) string {
	if cmd.Flags().Changed(flagName) {
		value, _ := cmd.Flags().GetString(flagName)
		return value
	}
	if value := os.Getenv(envName); value != "" {
		return value
	}
	return defaultValue
}

func getConfigString(key, defaultValue string) string {
	if viper.IsSet(key) {
		return viper.GetString(key)
	}
	return defaultValue
}

func getConfigInt(key string, defaultValue int) int {
	if viper.IsSet(key) {
		return viper.GetInt(key)
	}
	return defaultValue
}

func getConfigFloat64(key string, defaultValue float64) float64 {
	if viper.IsSet(key) {
		return viper.GetFloat64(key)
	}
	return defaultValue
}

func getConfigBool(key string, defaultValue bool) bool {
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return defaultValue
}