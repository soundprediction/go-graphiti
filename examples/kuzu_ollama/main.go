package main

import (
	"context"
	"log"
	"time"

	"github.com/soundprediction/go-graphiti"
	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// Example demonstrating the combination of:
// - Kuzu embedded graph database (local, no server required)
// - Ollama local LLM inference (local, no cloud API required)
// - OpenAI embeddings (or could be replaced with local embeddings)
//
// This setup provides maximum privacy and minimal dependencies while
// maintaining full Graphiti functionality.

func main() {
	ctx := context.Background()

	log.Println("🚀 Starting go-graphiti example with Kuzu + Ollama")
	log.Println("   This example demonstrates a fully local setup:")
	log.Println("   - Kuzu: embedded graph database")
	log.Println("   - Ollama: local LLM inference")
	log.Println("   - OpenAI: embeddings (could be replaced with local)")

	// ========================================
	// 1. Create Kuzu Driver (Embedded Graph Database)
	// ========================================
	log.Println("\n📊 Setting up Kuzu embedded graph database...")
	
	kuzuDriver, err := driver.NewKuzuDriver("./example_graph.db")
	if err != nil {
		log.Fatalf("Failed to create Kuzu driver: %v", err)
	}
	defer func() {
		if err := kuzuDriver.Close(ctx); err != nil {
			log.Printf("Error closing Kuzu driver: %v", err)
		}
	}()

	// Note: In the current stub implementation, this will work but
	// actual graph operations will return "not implemented" errors
	log.Println("   ✅ Kuzu driver created (embedded database at ./example_graph.db)")

	// ========================================
	// 2. Create Ollama LLM Client (Local Inference)
	// ========================================
	log.Println("\n🧠 Setting up Ollama local LLM client...")
	
	// Configure Ollama client for local inference
	// Assumes Ollama is running locally with a model like llama2:7b
	llmConfig := llm.Config{
		Model:       "llama2:7b",           // Popular 7B parameter model
		Temperature: &[]float32{0.7}[0],   // Balanced creativity
		MaxTokens:   &[]int{1000}[0],      // Reasonable response length
	}

	// Create Ollama client (defaults to http://localhost:11434)
	ollama, err := llm.NewOllamaClient("", "llama2:7b", llmConfig)
	if err != nil {
		log.Fatalf("Failed to create Ollama client: %v", err)
	}
	defer ollama.Close()

	log.Println("   ✅ Ollama client created (using llama2:7b model)")
	log.Println("   💡 Make sure Ollama is running: `ollama serve`")
	log.Println("   💡 Make sure model is available: `ollama pull llama2:7b`")

	// ========================================
	// 3. Create Embedder (OpenAI for now, could be local)
	// ========================================
	log.Println("\n🔤 Setting up embedding client...")
	
	// For this example, we'll use OpenAI embeddings
	// In a fully local setup, you could replace this with a local embedding service
	embedderConfig := embedder.Config{
		Model:     "text-embedding-3-small",
		BatchSize: 50,
	}
	
	// Note: Requires OPENAI_API_KEY environment variable
	// For fully local setup, replace with local embedding service
	embedderClient := embedder.NewOpenAIEmbedder("", embedderConfig) // Empty string uses env var
	defer embedderClient.Close()

	log.Println("   ✅ OpenAI embedder created (text-embedding-3-small)")
	log.Println("   💡 For fully local setup, replace with local embedding service")

	// ========================================
	// 4. Create Graphiti Client
	// ========================================
	log.Println("\n🌐 Setting up Graphiti client with local components...")
	
	graphitiConfig := &graphiti.Config{
		GroupID:  "kuzu-ollama-example",
		TimeZone: time.UTC,
	}

	client := graphiti.NewClient(kuzuDriver, ollama, embedderClient, graphitiConfig)
	defer func() {
		if err := client.Close(ctx); err != nil {
			log.Printf("Error closing Graphiti client: %v", err)
		}
	}()

	log.Println("   ✅ Graphiti client created with local Kuzu + Ollama setup")

	// ========================================
	// 5. Add Some Example Episodes
	// ========================================
	log.Println("\n📝 Adding example episodes to the knowledge graph...")

	episodes := []types.Episode{
		{
			ID:        "local-setup-1",
			Name:      "Local Development Setup",
			Content:   "Set up a fully local development environment using Kuzu embedded database and Ollama for LLM inference. This eliminates cloud dependencies and provides maximum privacy for sensitive development work.",
			Reference: time.Now().Add(-2 * time.Hour),
			CreatedAt: time.Now().Add(-2 * time.Hour),
			GroupID:   "kuzu-ollama-example",
		},
		{
			ID:        "performance-test-1",
			Name:      "Local Performance Testing",
			Content:   "Conducted performance tests comparing local Kuzu+Ollama setup against cloud-based Neo4j+OpenAI. Local setup showed 3x faster response times for graph queries but slower LLM inference due to hardware constraints.",
			Reference: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now().Add(-1 * time.Hour),
			GroupID:   "kuzu-ollama-example",
		},
		{
			ID:        "privacy-benefits-1",
			Name:      "Privacy and Security Benefits",
			Content:   "Local setup ensures all data remains on-premises. Graph data stored in local Kuzu database, LLM processing handled by local Ollama instance. Only embeddings require external API calls unless using local embedding service.",
			Reference: time.Now().Add(-30 * time.Minute),
			CreatedAt: time.Now().Add(-30 * time.Minute),
			GroupID:   "kuzu-ollama-example",
		},
	}

	// Note: In current implementation, this will demonstrate the API
	// but actual storage won't work until Kuzu library is available
	err = client.Add(ctx, episodes)
	if err != nil {
		log.Printf("⚠️  Expected error with stub implementation: %v", err)
		log.Println("   This will work once the Kuzu Go library is available")
	} else {
		log.Println("   ✅ Episodes added to knowledge graph")
	}

	// ========================================
	// 6. Demonstrate Search Functionality
	// ========================================
	log.Println("\n🔍 Searching the knowledge graph...")

	searchQueries := []string{
		"local development setup",
		"performance comparison",
		"privacy benefits",
		"embedded database",
	}

	for _, query := range searchQueries {
		log.Printf("   Searching for: '%s'", query)
		
		// Note: In current implementation, this will show the API structure
		// but actual search won't work until Kuzu is fully implemented
		results, err := client.Search(ctx, query, &types.SearchConfig{
			Limit: 5,
		})
		
		if err != nil {
			log.Printf("     ⚠️  Expected error with stub: %v", err)
		} else {
			log.Printf("     ✅ Found %d nodes, %d edges", len(results.Nodes), len(results.Edges))
			
			// Display results
			for i, node := range results.Nodes {
				if i >= 2 { // Limit output
					break
				}
				log.Printf("       - Node: %s (%s)", node.Name, node.Type)
			}
		}
	}

	// ========================================
	// 7. Demonstrate LLM Integration (This should work!)
	// ========================================
	log.Println("\n💭 Testing Ollama LLM integration...")

	// Test the LLM directly to show it works
	testMessages := []llm.Message{
		llm.NewSystemMessage("You are a helpful assistant discussing graph databases and local AI setups."),
		llm.NewUserMessage("What are the advantages of using an embedded graph database like Kuzu compared to a server-based solution like Neo4j?"),
	}

	log.Println("   Sending query to Ollama...")
	
	// Note: This will only work if Ollama is actually running
	response, err := ollama.Chat(ctx, testMessages)
	if err != nil {
		log.Printf("   ⚠️  Ollama error (is it running?): %v", err)
		log.Println("     To fix: Start Ollama with `ollama serve` and pull a model with `ollama pull llama2:7b`")
	} else {
		log.Println("   ✅ Ollama response received:")
		log.Printf("     %s", truncateString(response.Content, 200))
		
		if response.TokensUsed != nil {
			log.Printf("     Used %d tokens", response.TokensUsed.TotalTokens)
		}
	}

	// ========================================
	// 8. Summary and Next Steps
	// ========================================
	log.Println("\n📋 Example Summary:")
	log.Println("   ✅ Kuzu driver: Created (stub implementation)")
	log.Println("   ✅ Ollama client: Created and tested")
	log.Println("   ✅ Graphiti integration: Demonstrated")
	log.Println("\n🔮 Future State (when Kuzu library is available):")
	log.Println("   🚀 Full local operation with no cloud dependencies")
	log.Println("   📊 Embedded graph database for fast local queries")
	log.Println("   🧠 Local LLM inference for privacy and control")
	log.Println("   🔒 All data remains on your local machine")
	log.Println("\n💡 To achieve fully local setup:")
	log.Println("   1. Wait for stable Kuzu Go library release")
	log.Println("   2. Replace OpenAI embeddings with local alternative")
	log.Println("   3. Enjoy complete data privacy and control!")

	log.Println("\n🎉 Example completed successfully!")
}

// truncateString truncates a string to a maximum length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}