package main

import (
	"context"
	"log"
	"time"

	"github.com/soundprediction/go-predicato"
	"github.com/soundprediction/go-predicato/pkg/driver"
	"github.com/soundprediction/go-predicato/pkg/embedder"
	"github.com/soundprediction/go-predicato/pkg/llm"
	"github.com/soundprediction/go-predicato/pkg/types"
)

// Example demonstrating the combination of:
// - Ladybug embedded graph database (local, no server required)
// - Ollama local LLM inference via OpenAI-compatible API (local, no cloud API required)
// - OpenAI embeddings (or could be replaced with local embeddings)
//
// This setup provides maximum privacy and minimal dependencies while
// maintaining full Predicato functionality. Ollama's OpenAI-compatible API
// allows seamless integration with existing OpenAI client code.

func main() {
	ctx := context.Background()

	log.Println("🚀 Starting go-predicato example with Ladybug + Ollama (OpenAI-compatible)")
	log.Println("   This example demonstrates a fully local setup:")
	log.Println("   - Ladybug: embedded graph database")
	log.Println("   - Ollama: local LLM inference via OpenAI-compatible API")
	log.Println("   - OpenAI: embeddings (could be replaced with local)")

	// ========================================
	// 1. Create Ladybug Driver (Embedded Graph Database)
	// ========================================
	log.Println("\n📊 Setting up Ladybug embedded graph database...")

	ladybugDriver, err := driver.NewLadybugDriver("./example_graph.db", 1)
	if err != nil {
		log.Fatalf("Failed to create Ladybug driver: %v", err)
	}
	defer func() {
		if err := ladybugDriver.Close(); err != nil {
			log.Printf("Error closing Ladybug driver: %v", err)
		}
	}()

	// Note: In the current stub implementation, this will work but
	// actual graph operations will return "not implemented" errors
	log.Println("   ✅ Ladybug driver created (embedded database at ./example_graph.db)")

	// ========================================
	// 2. Create Ollama LLM Client (Local Inference)
	// ========================================
	log.Println("\n🧠 Setting up Ollama local LLM client...")

	// Create Ollama client using OpenAI-compatible API
	// Assumes Ollama is running locally with a model like llama2:7b
	baseOllama, err := llm.NewOpenAIClient("", llm.Config{
		BaseURL:     "http://localhost:11434", // Ollama's OpenAI-compatible endpoint
		Model:       "llama2:7b",              // Popular 7B parameter model
		Temperature: &[]float32{0.7}[0],       // Balanced creativity
		MaxTokens:   &[]int{1000}[0],          // Reasonable response length
	})
	if err != nil {
		log.Fatalf("Failed to create Ollama client: %v", err)
	}
	// Wrap with retry client for automatic retry on errors
	ollama := llm.NewRetryClient(baseOllama, llm.DefaultRetryConfig())
	defer ollama.Close()

	log.Println("   ✅ Ollama client created with retry support (using OpenAI-compatible API with llama2:7b)")
	log.Println("   💡 Make sure Ollama is running: `ollama serve`")
	log.Println("   💡 Make sure model is available: `ollama pull llama2:7b`")
	log.Println("   💡 Ollama exposes OpenAI-compatible API at /v1/chat/completions")

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
	// 4. Create Predicato Client
	// ========================================
	log.Println("\n🌐 Setting up Predicato client with local components...")

	predicatoConfig := &predicato.Config{
		GroupID:  "ladybug-ollama-example",
		TimeZone: time.UTC,
	}

	client := predicato.NewClient(ladybugDriver, ollama, embedderClient, predicatoConfig, nil)
	defer func() {
		if err := client.Close(ctx); err != nil {
			log.Printf("Error closing Predicato client: %v", err)
		}
	}()

	log.Println("   ✅ Predicato client created with local Ladybug + Ollama setup")

	// ========================================
	// 5. Add Some Example Episodes
	// ========================================
	log.Println("\n📝 Adding example episodes to the knowledge graph...")

	episodes := []types.Episode{
		{
			ID:        "local-setup-1",
			Name:      "Local Development Setup",
			Content:   "Set up a fully local development environment using Ladybug embedded database and Ollama for LLM inference. This eliminates cloud dependencies and provides maximum privacy for sensitive development work.",
			Reference: time.Now().Add(-2 * time.Hour),
			CreatedAt: time.Now().Add(-2 * time.Hour),
			GroupID:   "ladybug-ollama-example",
		},
		{
			ID:        "performance-test-1",
			Name:      "Local Performance Testing",
			Content:   "Conducted performance tests comparing local Ladybug+Ollama setup against cloud-based Neo4j+OpenAI. Local setup showed 3x faster response times for graph queries but slower LLM inference due to hardware constraints.",
			Reference: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now().Add(-1 * time.Hour),
			GroupID:   "ladybug-ollama-example",
		},
		{
			ID:        "privacy-benefits-1",
			Name:      "Privacy and Security Benefits",
			Content:   "Local setup ensures all data remains on-premises. Graph data stored in local Ladybug database, LLM processing handled by local Ollama instance. Only embeddings require external API calls unless using local embedding service.",
			Reference: time.Now().Add(-30 * time.Minute),
			CreatedAt: time.Now().Add(-30 * time.Minute),
			GroupID:   "ladybug-ollama-example",
		},
	}

	// Note: In current implementation, this will demonstrate the API
	// but actual storage won't work until Ladybug library is available
	_, err = client.Add(ctx, episodes, nil)
	if err != nil {
		log.Printf("⚠️  Expected error with stub implementation: %v", err)
		log.Println("   This will work once the Ladybug Go library is available")
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
		// but actual search won't work until Ladybug is fully implemented
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
	testMessages := []types.Message{
		llm.NewSystemMessage("You are a helpful assistant discussing graph databases and local AI setups."),
		llm.NewUserMessage("What are the advantages of using an embedded graph database like Ladybug compared to a server-based solution like Neo4j?"),
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
	log.Println("   ✅ Ladybug driver: Created (stub implementation)")
	log.Println("   ✅ Ollama client: Created using OpenAI-compatible API and tested")
	log.Println("   ✅ Predicato integration: Demonstrated with modern API approach")
	log.Println("\n🔮 Future State (when Ladybug library is available):")
	log.Println("   🚀 Full local operation with no cloud dependencies")
	log.Println("   📊 Embedded graph database for fast local queries")
	log.Println("   🧠 Local LLM inference via standardized OpenAI-compatible API")
	log.Println("   🔒 All data remains on your local machine")
	log.Println("\n💡 To achieve fully local setup:")
	log.Println("   1. Wait for stable Ladybug Go library release")
	log.Println("   2. Replace OpenAI embeddings with local alternative")
	log.Println("   3. Enjoy complete data privacy and control!")
	log.Println("\n🔧 OpenAI-Compatible API Benefits:")
	log.Println("   ✅ Standardized interface across different LLM providers")
	log.Println("   ✅ Easy switching between local and cloud LLM services")
	log.Println("   ✅ Leverages existing OpenAI tooling and libraries")

	log.Println("\n🎉 Example completed successfully!")
}

// truncateString truncates a string to a maximum length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
