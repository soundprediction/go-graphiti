package graphiti

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soundprediction/go-graphiti"
	"github.com/soundprediction/go-graphiti/pkg/config"
	"github.com/soundprediction/go-graphiti/pkg/server"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the Go-Graphiti HTTP server",
	Long: `Start the Go-Graphiti HTTP server to provide REST API access to the knowledge graph.

The server provides endpoints for:
- Ingesting data (messages, entities)
- Searching the knowledge graph
- Retrieving episodes and memory
- Health checks

Configuration can be provided through config files, environment variables, or command-line flags.`,
	RunE: runServer,
}

var (
	serverHost string
	serverPort int
	serverMode string
)

func init() {
	rootCmd.AddCommand(serverCmd)

	// Server-specific flags
	serverCmd.Flags().StringVar(&serverHost, "host", "localhost", "Server host")
	serverCmd.Flags().IntVar(&serverPort, "port", 8080, "Server port")
	serverCmd.Flags().StringVar(&serverMode, "mode", "debug", "Server mode (debug, release, test)")

	// Database flags
	serverCmd.Flags().String("db-driver", "neo4j", "Database driver (neo4j, falkordb)")
	serverCmd.Flags().String("db-uri", "bolt://localhost:7687", "Database URI")
	serverCmd.Flags().String("db-username", "neo4j", "Database username")
	serverCmd.Flags().String("db-password", "password", "Database password")
	serverCmd.Flags().String("db-database", "neo4j", "Database name")

	// LLM flags
	serverCmd.Flags().String("llm-provider", "openai", "LLM provider")
	serverCmd.Flags().String("llm-model", "gpt-4", "LLM model")
	serverCmd.Flags().String("llm-api-key", "", "LLM API key")
	serverCmd.Flags().String("llm-base-url", "", "LLM base URL")
	serverCmd.Flags().Float32("llm-temperature", 0.1, "LLM temperature")
	serverCmd.Flags().Int("llm-max-tokens", 2048, "LLM max tokens")

	// Embedding flags
	serverCmd.Flags().String("embedding-provider", "openai", "Embedding provider")
	serverCmd.Flags().String("embedding-model", "text-embedding-3-small", "Embedding model")
	serverCmd.Flags().String("embedding-api-key", "", "Embedding API key")
	serverCmd.Flags().String("embedding-base-url", "", "Embedding base URL")
}

func runServer(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override config with command-line flags
	overrideConfigWithFlags(cmd, cfg)

	// Validate configuration
	if err := validateServerConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize Graphiti
	fmt.Println("Initializing Graphiti...")
	graphitiInstance, err := initializeGraphiti(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize Graphiti: %w", err)
	}

	// Create and setup server
	srv := server.New(cfg, graphitiInstance)
	srv.Setup()

	// Setup graceful shutdown
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			serverErrChan <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrChan:
		return fmt.Errorf("server error: %w", err)
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal: %v\n", sig)

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// Shutdown server
		if err := srv.Stop(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}

		fmt.Println("Server stopped gracefully")
		return nil
	}
}

func overrideConfigWithFlags(cmd *cobra.Command, cfg *config.Config) {
	// Server flags
	if cmd.Flags().Changed("host") {
		cfg.Server.Host = serverHost
	}
	if cmd.Flags().Changed("port") {
		cfg.Server.Port = serverPort
	}
	if cmd.Flags().Changed("mode") {
		cfg.Server.Mode = serverMode
	}

	// Database flags
	if cmd.Flags().Changed("db-driver") {
		cfg.Database.Driver, _ = cmd.Flags().GetString("db-driver")
	}
	if cmd.Flags().Changed("db-uri") {
		cfg.Database.URI, _ = cmd.Flags().GetString("db-uri")
	}
	if cmd.Flags().Changed("db-username") {
		cfg.Database.Username, _ = cmd.Flags().GetString("db-username")
	}
	if cmd.Flags().Changed("db-password") {
		cfg.Database.Password, _ = cmd.Flags().GetString("db-password")
	}
	if cmd.Flags().Changed("db-database") {
		cfg.Database.Database, _ = cmd.Flags().GetString("db-database")
	}

	// LLM flags
	if cmd.Flags().Changed("llm-provider") {
		cfg.LLM.Provider, _ = cmd.Flags().GetString("llm-provider")
	}
	if cmd.Flags().Changed("llm-model") {
		cfg.LLM.Model, _ = cmd.Flags().GetString("llm-model")
	}
	if cmd.Flags().Changed("llm-api-key") {
		cfg.LLM.APIKey, _ = cmd.Flags().GetString("llm-api-key")
	}
	if cmd.Flags().Changed("llm-base-url") {
		cfg.LLM.BaseURL, _ = cmd.Flags().GetString("llm-base-url")
	}
	if cmd.Flags().Changed("llm-temperature") {
		cfg.LLM.Temperature, _ = cmd.Flags().GetFloat32("llm-temperature")
	}
	if cmd.Flags().Changed("llm-max-tokens") {
		cfg.LLM.MaxTokens, _ = cmd.Flags().GetInt("llm-max-tokens")
	}

	// Embedding flags
	if cmd.Flags().Changed("embedding-provider") {
		cfg.Embedding.Provider, _ = cmd.Flags().GetString("embedding-provider")
	}
	if cmd.Flags().Changed("embedding-model") {
		cfg.Embedding.Model, _ = cmd.Flags().GetString("embedding-model")
	}
	if cmd.Flags().Changed("embedding-api-key") {
		cfg.Embedding.APIKey, _ = cmd.Flags().GetString("embedding-api-key")
	}
	if cmd.Flags().Changed("embedding-base-url") {
		cfg.Embedding.BaseURL, _ = cmd.Flags().GetString("embedding-base-url")
	}
}

func validateServerConfig(cfg *config.Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Server.Port)
	}

	if cfg.Database.URI == "" {
		return fmt.Errorf("database URI is required")
	}

	if cfg.LLM.APIKey == "" {
		return fmt.Errorf("LLM API key is required")
	}

	return nil
}

func initializeGraphiti(cfg *config.Config) (*graphiti.Graphiti, error) {
	// TODO: Initialize Graphiti with the provided configuration
	// This is a placeholder implementation
	// The actual implementation would depend on the Graphiti package API

	// For now, return a nil instance with a note
	fmt.Println("Note: Graphiti initialization is a placeholder - needs actual implementation")
	return nil, nil
}
