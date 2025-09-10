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
	// Get environment variables
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}
	
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
		log.Fatal("NEO4J_PASSWORD environment variable is required")
	}

	ctx := context.Background()

	// Create Neo4j driver
	neo4jDriver, err := driver.NewNeo4jDriver(neo4jURI, neo4jUser, neo4jPassword, "neo4j")
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	defer neo4jDriver.Close(ctx)

	// Create LLM client
	llmConfig := llm.Config{
		Model:       "gpt-4o-mini",
		Temperature: floatPtr(0.7),
		MaxTokens:   intPtr(1000),
	}
	llmClient := llm.NewOpenAIClient(openaiAPIKey, llmConfig)
	defer llmClient.Close()

	// Create embedder client
	embedderConfig := embedder.Config{
		Model:     "text-embedding-3-small",
		BatchSize: 100,
	}
	embedderClient := embedder.NewOpenAIEmbedder(openaiAPIKey, embedderConfig)
	defer embedderClient.Close()

	// Create Graphiti client
	config := &graphiti.Config{
		GroupID:  "example-group",
		TimeZone: time.UTC,
	}
	
	client := graphiti.NewClient(neo4jDriver, llmClient, embedderClient, config)
	defer client.Close(ctx)

	// Example: Add some episodes
	episodes := []types.Episode{
		{
			ID:        "episode-1",
			Name:      "Meeting with Alice",
			Content:   "Had a productive meeting with Alice about the new project. She mentioned that the deadline is next month and we need to focus on the API design.",
			Reference: time.Now().Add(-24 * time.Hour), // Yesterday
			CreatedAt: time.Now(),
			GroupID:   "example-group",
			Metadata: map[string]interface{}{
				"type": "meeting",
			},
		},
		{
			ID:        "episode-2",
			Name:      "Project Research",
			Content:   "Researched various approaches for implementing the API. Found that GraphQL might be a good fit for our use case due to its flexibility.",
			Reference: time.Now().Add(-12 * time.Hour), // 12 hours ago
			CreatedAt: time.Now(),
			GroupID:   "example-group",
			Metadata: map[string]interface{}{
				"type": "research",
			},
		},
	}

	fmt.Println("Adding episodes to the knowledge graph...")
	if err := client.Add(ctx, episodes); err != nil {
		log.Printf("Warning: Add operation not yet implemented: %v", err)
	}

	// Example: Search the knowledge graph
	fmt.Println("Searching the knowledge graph...")
	searchConfig := &types.SearchConfig{
		Limit:              10,
		CenterNodeDistance: 2,
		MinScore:           0.0,
		IncludeEdges:       true,
		Rerank:             false,
	}

	results, err := client.Search(ctx, "API design and deadlines", searchConfig)
	if err != nil {
		log.Printf("Warning: Search operation not yet implemented: %v", err)
	} else if results != nil {
		fmt.Printf("Found %d nodes and %d edges\n", len(results.Nodes), len(results.Edges))
	}

	fmt.Println("Example completed successfully!")
}

func floatPtr(f float32) *float32 {
	return &f
}

func intPtr(i int) *int {
	return &i
}