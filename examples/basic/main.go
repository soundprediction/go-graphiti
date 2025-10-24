// Package main demonstrates basic usage of go-graphiti with OpenAI LLM and Neo4j database.
//
// This example shows how to:
// - Create and configure a Graphiti client with Neo4j and OpenAI
// - Add episodes (data) to the knowledge graph
// - Search the knowledge graph for relevant information
//
// Prerequisites:
// - Neo4j database running (default: bolt://localhost:7687)
// - OpenAI API key
//
// Environment Variables:
// - OPENAI_API_KEY (required): Your OpenAI API key
// - NEO4J_PASSWORD (required): Your Neo4j database password
// - NEO4J_URI (optional): Neo4j connection URI (default: bolt://localhost:7687)
// - NEO4J_USER (optional): Neo4j username (default: neo4j)
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
		fmt.Println("OPENAI_API_KEY environment variable is not set.")
		fmt.Println("Please set it to run the full example:")
		fmt.Println("  export OPENAI_API_KEY=your_api_key_here")
		fmt.Println()
		fmt.Println("Exiting...")
		return
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
		fmt.Println("NEO4J_PASSWORD environment variable is not set.")
		fmt.Println("Please set it to run the full example:")
		fmt.Println("  export NEO4J_PASSWORD=your_password_here")
		fmt.Println()
		fmt.Println("You can also set these optional variables:")
		fmt.Printf("  export NEO4J_URI=%s\n", neo4jURI)
		fmt.Printf("  export NEO4J_USER=%s\n", neo4jUser)
		fmt.Println()
		fmt.Println("Exiting...")
		return
	}

	ctx := context.Background()

	fmt.Println("🚀 Starting go-graphiti basic example")
	fmt.Printf("   Neo4j URI: %s\n", neo4jURI)
	fmt.Printf("   Neo4j User: %s\n", neo4jUser)
	fmt.Println()

	// Create Neo4j driver
	fmt.Println("📊 Creating Neo4j driver...")
	neo4jDriver, err := driver.NewNeo4jDriver(neo4jURI, neo4jUser, neo4jPassword, "neo4j")
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	defer neo4jDriver.Close(ctx)
	fmt.Println("   ✅ Neo4j driver created successfully")

	// Create LLM client
	fmt.Println("\n🧠 Creating OpenAI LLM client...")
	llmConfig := llm.Config{
		Model:       "gpt-4o-mini",
		Temperature: floatPtr(0.7),
		MaxTokens:   intPtr(1000),
	}
	llmClient, err := llm.NewOpenAIClient(openaiAPIKey, llmConfig)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}
	defer llmClient.Close()
	fmt.Printf("   ✅ OpenAI LLM client created (model: %s)\n", llmConfig.Model)

	// Create embedder client
	fmt.Println("\n🔤 Creating OpenAI embedder client...")
	embedderConfig := embedder.Config{
		Model:     "text-embedding-3-small",
		BatchSize: 100,
	}
	embedderClient := embedder.NewOpenAIEmbedder(openaiAPIKey, embedderConfig)
	defer embedderClient.Close()
	fmt.Printf("   ✅ OpenAI embedder client created (model: %s)\n", embedderConfig.Model)

	// Create Graphiti client
	fmt.Println("\n🌐 Creating Graphiti client...")
	config := &graphiti.Config{
		GroupID:  "example-group",
		TimeZone: time.UTC,
	}

	client := graphiti.NewClient(neo4jDriver, llmClient, embedderClient, config)
	defer client.Close(ctx)
	fmt.Printf("   ✅ Graphiti client created (group: %s)\n", config.GroupID)

	// Example: Add some episodes
	fmt.Println("\n📝 Preparing sample episodes...")
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
	if err := client.Add(ctx, episodes, nil); err != nil {
		fmt.Printf("Warning: Add operation failed: %v\n", err)
		fmt.Println("This is expected if the implementation is still in development.")
	} else {
		fmt.Println("✅ Episodes successfully added to the knowledge graph!")
	}

	// Example: Search the knowledge graph
	fmt.Println("\nSearching the knowledge graph...")
	searchConfig := &types.SearchConfig{
		Limit:              10,
		CenterNodeDistance: 2,
		MinScore:           0.0,
		IncludeEdges:       true,
		Rerank:             false,
	}

	results, err := client.Search(ctx, "API design and deadlines", searchConfig)
	if err != nil {
		fmt.Printf("Warning: Search operation failed: %v\n", err)
		fmt.Println("This is expected if the implementation is still in development.")
	} else if results != nil {
		fmt.Printf("✅ Found %d nodes and %d edges\n", len(results.Nodes), len(results.Edges))

		// Display some results if available
		if len(results.Nodes) > 0 {
			fmt.Println("\nSample nodes found:")
			for i, node := range results.Nodes {
				if i >= 3 { // Limit to first 3 nodes
					break
				}
				fmt.Printf("  - %s (%s)\n", node.Name, node.Type)
			}
		}
	}

	fmt.Println("Example completed successfully!")
}

func floatPtr(f float32) *float32 {
	return &f
}

func intPtr(i int) *int {
	return &i
}
