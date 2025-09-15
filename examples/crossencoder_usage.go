package main

import (
	"context"
	"fmt"
	"log"

	"github.com/soundprediction/go-graphiti/pkg/crossencoder"
)

// Example demonstrating how to use the generic Jina-compatible reranker
// with different services (vLLM, LocalAI, Jina AI, etc.)
func main() {
	ctx := context.Background()

	// Example passages to rank
	query := "What is machine learning?"
	passages := []string{
		"Machine learning is a subset of artificial intelligence that uses algorithms to learn patterns from data.",
		"Cooking involves combining ingredients and applying heat to create delicious meals.",
		"Deep learning uses neural networks with multiple layers to solve complex problems.",
		"Weather patterns are influenced by atmospheric pressure, temperature, and humidity.",
		"Supervised learning trains models on labeled data to make predictions on new data.",
	}

	fmt.Println("🔄 Cross-Encoder Reranking Example")
	fmt.Printf("Query: %s\n", query)
	fmt.Printf("Passages to rank: %d\n\n", len(passages))

	// Example 1: Using vLLM with a cross-encoder model
	fmt.Println("1. Using vLLM Reranker (assuming vLLM server is running)")
	vllmReranker := crossencoder.NewVLLMRerankerClient(
		"http://localhost:8000/v1", // vLLM server URL
		"BAAI/bge-reranker-large",  // Cross-encoder model
	)
	defer vllmReranker.Close()

	// Note: This will fail if vLLM server is not running, but shows the API
	results, err := vllmReranker.Rank(ctx, query, passages)
	if err != nil {
		fmt.Printf("   ⚠️  vLLM reranking failed (server may not be running): %v\n", err)
	} else {
		fmt.Println("   ✅ vLLM reranking successful:")
		for i, result := range results {
			fmt.Printf("   %d. (Score: %.3f) %s\n", i+1, result.Score, result.Passage[:50]+"...")
		}
	}
	fmt.Println()

	// Example 2: Using Jina AI reranking service
	fmt.Println("2. Using Jina AI Reranker (requires API key)")
	jinaReranker := crossencoder.NewJinaRerankerClient(
		"your-jina-api-key",       // API key from Jina AI
		"jina-reranker-v1-base-en", // Jina reranker model
	)
	defer jinaReranker.Close()

	// Note: This will fail without a valid API key, but shows the API
	results, err = jinaReranker.Rank(ctx, query, passages)
	if err != nil {
		fmt.Printf("   ⚠️  Jina reranking failed (API key may be invalid): %v\n", err)
	} else {
		fmt.Println("   ✅ Jina reranking successful:")
		for i, result := range results {
			fmt.Printf("   %d. (Score: %.3f) %s\n", i+1, result.Score, result.Passage[:50]+"...")
		}
	}
	fmt.Println()

	// Example 3: Using LocalAI reranking service
	fmt.Println("3. Using LocalAI Reranker (assuming LocalAI server is running)")
	localAIReranker := crossencoder.NewLocalAIRerankerClient(
		"http://localhost:8080/v1", // LocalAI server URL
		"reranker",                 // Model name configured in LocalAI
		"",                         // API key (empty if not required)
	)
	defer localAIReranker.Close()

	// Note: This will fail if LocalAI server is not running, but shows the API
	results, err = localAIReranker.Rank(ctx, query, passages)
	if err != nil {
		fmt.Printf("   ⚠️  LocalAI reranking failed (server may not be running): %v\n", err)
	} else {
		fmt.Println("   ✅ LocalAI reranking successful:")
		for i, result := range results {
			fmt.Printf("   %d. (Score: %.3f) %s\n", i+1, result.Score, result.Passage[:50]+"...")
		}
	}
	fmt.Println()

	// Example 4: Using generic reranker client for any Jina-compatible service
	fmt.Println("4. Using Generic Reranker Client")
	config := crossencoder.RerankerConfig{
		Config: crossencoder.Config{
			Model: "my-custom-reranker-model",
		},
		BaseURL: "http://my-custom-service:8080/v1",
		APIKey:  "my-api-key", // If required by your service
	}
	genericReranker := crossencoder.NewRerankerClient(config)
	defer genericReranker.Close()

	// Note: This will fail without a real service, but shows the API
	results, err = genericReranker.Rank(ctx, query, passages)
	if err != nil {
		fmt.Printf("   ⚠️  Generic reranking failed (custom service not available): %v\n", err)
	} else {
		fmt.Println("   ✅ Generic reranking successful:")
		for i, result := range results {
			fmt.Printf("   %d. (Score: %.3f) %s\n", i+1, result.Score, result.Passage[:50]+"...")
		}
	}

	fmt.Println()
	fmt.Println("💡 Key Benefits of Generic Jina-Compatible Approach:")
	fmt.Println("   - Works with any service implementing the Jina reranking API")
	fmt.Println("   - No vendor lock-in - switch between services easily")
	fmt.Println("   - Supports local services (vLLM, LocalAI) and cloud services (Jina AI)")
	fmt.Println("   - Consistent API regardless of underlying service")
	fmt.Println()
	fmt.Println("🚀 To use with a real service:")
	fmt.Println("   1. Start your preferred reranking service (vLLM, LocalAI, etc.)")
	fmt.Println("   2. Update the BaseURL and model name")
	fmt.Println("   3. Provide API key if required")
	fmt.Println("   4. The client will work with any Jina-compatible endpoint!")
}

// Helper function to demonstrate how the reranker would be integrated
// into a larger search system
func demonstrateSearchIntegration() {
	ctx := context.Background()

	// This shows how you might use the reranker in a real search system
	query := "artificial intelligence applications"
	initialResults := []string{
		"AI is used in healthcare for medical diagnosis and drug discovery.",
		"The weather today is sunny with a chance of rain.",
		"Machine learning algorithms power recommendation systems.",
		"Cooking recipes often include step-by-step instructions.",
		"Natural language processing enables chatbots and translation.",
	}

	// Create reranker based on your deployment
	var reranker crossencoder.Client

	// Option 1: Use vLLM for local deployment
	reranker = crossencoder.NewVLLMRerankerClient("http://localhost:8000/v1", "BAAI/bge-reranker-large")

	// Option 2: Use Jina AI for cloud deployment
	// reranker = crossencoder.NewJinaRerankerClient("your-api-key", "jina-reranker-v1-base-en")

	// Option 3: Use any other Jina-compatible service
	// config := crossencoder.RerankerConfig{...}
	// reranker = crossencoder.NewRerankerClient(config)

	defer reranker.Close()

	// Rerank the initial results
	rerankedResults, err := reranker.Rank(ctx, query, initialResults)
	if err != nil {
		log.Printf("Reranking failed, using original order: %v", err)
		return
	}

	fmt.Println("Reranked results (most relevant first):")
	for i, result := range rerankedResults {
		fmt.Printf("%d. (Score: %.3f) %s\n", i+1, result.Score, result.Passage)
	}
}