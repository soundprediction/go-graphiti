package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/soundprediction/go-graphiti"
	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

func main() {
	// Example 1: Using Ollama (local LLM server)
	fmt.Println("=== Ollama Example ===")
	if err := runOllamaExample(); err != nil {
		log.Printf("Ollama example failed: %v", err)
	}

	// Example 2: Using LocalAI
	fmt.Println("\n=== LocalAI Example ===")
	if err := runLocalAIExample(); err != nil {
		log.Printf("LocalAI example failed: %v", err)
	}

	// Example 3: Using vLLM
	fmt.Println("\n=== vLLM Example ===")
	if err := runVLLMExample(); err != nil {
		log.Printf("vLLM example failed: %v", err)
	}

	// Example 4: Custom OpenAI-compatible service
	fmt.Println("\n=== Custom Service Example ===")
	if err := runCustomServiceExample(); err != nil {
		log.Printf("Custom service example failed: %v", err)
	}

	// Example 5: Using with full Graphiti client
	fmt.Println("\n=== Full Graphiti Integration Example ===")
	if err := runGraphitiIntegrationExample(); err != nil {
		log.Printf("Graphiti integration example failed: %v", err)
	}
}

func runOllamaExample() error {
	fmt.Println("Creating Ollama client...")
	
	// Create Ollama client (assumes Ollama is running on localhost:11434)
	client, err := llm.NewOpenAIClient(
		"", // No API key needed for Ollama
		llm.Config{
			BaseURL:     "http://localhost:11434", // Ollama default URL
			Model:       "llama2:7b",              // Model name
			Temperature: &[]float32{0.7}[0],
			MaxTokens:   &[]int{100}[0],
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create Ollama client: %w", err)
	}
	defer client.Close()

	// Test basic chat functionality
	messages := []llm.Message{
		llm.NewUserMessage("Explain what a knowledge graph is in one sentence."),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("chat failed: %w", err)
	}

	fmt.Printf("Ollama Response: %s\n", response.Content)
	if response.TokensUsed != nil {
		fmt.Printf("Tokens used: %d\n", response.TokensUsed.TotalTokens)
	}

	return nil
}

func runLocalAIExample() error {
	fmt.Println("Creating LocalAI client...")
	
	// Create LocalAI client
	client, err := llm.NewOpenAIClient(
		"", // No API key needed for LocalAI
		llm.Config{
			BaseURL:     "http://localhost:8080", // LocalAI default URL
			Model:       "gpt-3.5-turbo",         // Model name configured in LocalAI
			Temperature: &[]float32{0.8}[0],
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create LocalAI client: %w", err)
	}
	defer client.Close()

	messages := []llm.Message{
		llm.NewSystemMessage("You are a helpful assistant specialized in graph databases."),
		llm.NewUserMessage("What are the benefits of using Neo4j for knowledge graphs?"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("chat failed: %w", err)
	}

	fmt.Printf("LocalAI Response: %s\n", response.Content)
	return nil
}

func runVLLMExample() error {
	fmt.Println("Creating vLLM client...")
	
	// Create vLLM client
	client, err := llm.NewOpenAIClient(
		"", // No API key needed for vLLM
		llm.Config{
			BaseURL:   "http://vllm-server:8000", // vLLM server URL
			Model:     "microsoft/DialoGPT-medium", // Model name
			MaxTokens: &[]int{150}[0],
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create vLLM client: %w", err)
	}
	defer client.Close()

	messages := []llm.Message{
		llm.NewUserMessage("How do you implement efficient graph traversal algorithms?"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("chat failed: %w", err)
	}

	fmt.Printf("vLLM Response: %s\n", response.Content)
	return nil
}

func runCustomServiceExample() error {
	fmt.Println("Creating custom OpenAI-compatible client...")
	
	// Create client for a custom OpenAI-compatible service
	client, err := llm.NewOpenAIClient(
		"your-api-key",                     // API key
		llm.Config{
			BaseURL:     "https://api.your-service.com",     // Your service URL
			Model:       "your-model-name",                  // Model identifier
			Temperature: &[]float32{0.5}[0],
			MaxTokens:   &[]int{200}[0],
			Stop:        []string{"</s>", "\n\n"},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create custom client: %w", err)
	}
	defer client.Close()

	// Test structured output (if your service supports it)
	messages := []llm.Message{
		llm.NewSystemMessage("You are an expert in data structures. Respond with valid JSON."),
		llm.NewUserMessage("Describe a graph data structure with its properties."),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try structured output
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":        map[string]string{"type": "string"},
			"description": map[string]string{"type": "string"},
			"properties":  map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
		},
	}

	structuredResponse, err := client.ChatWithStructuredOutput(ctx, messages, schema)
	if err != nil {
		// Fallback to regular chat if structured output fails
		fmt.Printf("Structured output not supported, falling back to regular chat: %v\n", err)
		
		response, err := client.Chat(ctx, messages)
		if err != nil {
			return fmt.Errorf("chat failed: %w", err)
		}
		fmt.Printf("Custom Service Response: %s\n", response.Content)
	} else {
		fmt.Printf("Custom Service Structured Response: %s\n", string(structuredResponse))
	}

	return nil
}

func runGraphitiIntegrationExample() error {
	fmt.Println("Creating Graphiti client with Ollama LLM...")

	// This example shows how to integrate the OpenAI-compatible client
	// with the full Graphiti system

	// Create Neo4j driver (you'll need Neo4j running)
	neo4jURI := os.Getenv("NEO4J_URI")
	if neo4jURI == "" {
		neo4jURI = "bolt://localhost:7687"
	}
	
	neo4jUser := os.Getenv("NEO4J_USER")
	if neo4jUser == "" {
		neo4jUser = "neo4j"
	}
	
	neo4jPassword := os.Getenv("NEO4J_PASSWORD")
	if neo4jPassword == "" {
		fmt.Println("Warning: NEO4J_PASSWORD not set, using 'password'")
		neo4jPassword = "password"
	}

	neo4jDriver, err := driver.NewNeo4jDriver(neo4jURI, neo4jUser, neo4jPassword, "neo4j")
	if err != nil {
		return fmt.Errorf("failed to create Neo4j driver: %w", err)
	}
	defer neo4jDriver.Close()

	// Create Ollama LLM client
	llmClient, err := llm.NewOpenAIClient(
		"", // No API key needed for Ollama
		llm.Config{
			BaseURL:     "http://localhost:11434",
			Model:       "llama2:7b",
			Temperature: &[]float32{0.7}[0],
			MaxTokens:   &[]int{1000}[0],
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create Ollama client: %w", err)
	}
	defer llmClient.Close()

	// For embeddings, we'll still use OpenAI since most local solutions
	// don't have great embedding models yet, but you could also use
	// a local embedding service
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		fmt.Println("Warning: OPENAI_API_KEY not set, skipping Graphiti integration")
		return nil
	}

	embedderClient := embedder.NewOpenAIEmbedder(openaiAPIKey, embedder.Config{
		Model:     "text-embedding-3-small",
		BatchSize: 100,
	})
	defer embedderClient.Close()

	// Create Graphiti client with local LLM and cloud embeddings
	config := &graphiti.Config{
		GroupID:  "ollama-example",
		TimeZone: time.UTC,
	}

	graphitiClient := graphiti.NewClient(neo4jDriver, llmClient, embedderClient, config)
	defer graphitiClient.Close(context.Background())

	// Add some sample data
	episodes := []types.Episode{
		{
			ID:        "local-llm-test",
			Name:      "Local LLM Testing",
			Content:   "We successfully integrated Ollama (local LLM) with Graphiti for knowledge graph processing. This allows us to run entirely locally except for embeddings.",
			Reference: time.Now(),
			CreatedAt: time.Now(),
			GroupID:   "ollama-example",
			Metadata: map[string]interface{}{
				"llm_provider": "ollama",
				"model":        "llama2:7b",
			},
		},
	}

	ctx := context.Background()
	fmt.Println("Adding episodes to knowledge graph...")
	if _, err := graphitiClient.Add(ctx, episodes, nil); err != nil {
		// Note: This might fail if the LLM processing pipeline isn't fully implemented yet
		fmt.Printf("Warning: Episode processing not yet implemented: %v\n", err)
	} else {
		fmt.Println("Successfully processed episodes with local LLM!")

		// Search the knowledge graph
		results, err := graphitiClient.Search(ctx, "local LLM integration", nil)
		if err != nil {
			fmt.Printf("Warning: Search not yet implemented: %v\n", err)
		} else {
			fmt.Printf("Found %d relevant nodes in knowledge graph\n", len(results.Nodes))
		}
	}

	return nil
}

// Helper function to check if a service is available
func checkServiceAvailable(url string) bool {
	// In a real implementation, you might want to make a health check request
	// For now, we'll assume services are available
	fmt.Printf("Note: Assuming service at %s is available\n", url)
	return true
}

func init() {
	// Print usage instructions
	fmt.Println("OpenAI-Compatible Client Examples")
	fmt.Println("=================================")
	fmt.Println()
	fmt.Println("This example demonstrates how to use go-graphiti with various")
	fmt.Println("OpenAI-compatible services. Make sure you have the following")
	fmt.Println("services running:")
	fmt.Println()
	fmt.Println("1. Ollama: Install and run 'ollama serve', then 'ollama pull llama2:7b'")
	fmt.Println("2. LocalAI: Run LocalAI server on http://localhost:8080")
	fmt.Println("3. vLLM: Run vLLM server on the specified URL")
	fmt.Println("4. Neo4j: Required for full Graphiti integration")
	fmt.Println()
	fmt.Println("Set these environment variables:")
	fmt.Println("- NEO4J_URI (default: bolt://localhost:7687)")
	fmt.Println("- NEO4J_USER (default: neo4j)")  
	fmt.Println("- NEO4J_PASSWORD (required for Graphiti integration)")
	fmt.Println("- OPENAI_API_KEY (optional, for embeddings)")
	fmt.Println()
}