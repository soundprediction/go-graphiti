// Package main demonstrates how to build an interactive chat application using go-graphiti
// with both global knowledge base and user-specific episodic memory.
//
// This example shows how to:
// - Create separate Graphiti clients for global knowledge and user-specific data
// - Use episodes to track conversation history with AddToEpisode
// - Search the global knowledge graph for context
// - Maintain conversation continuity with UUID v7 episode IDs
// - Build an interactive chat loop with conversation history
//
// Prerequisites:
// - OpenAI API key for LLM and embeddings
//
// Environment Variables:
// - OPENAI_API_KEY (required): Your OpenAI API key
//
// Usage:
//
//	go run main.go --user-id alice
//	go run main.go --user-id alice --global-db ./knowledge_db --user-db-dir ./user_dbs
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/soundprediction/go-graphiti"
	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// ChatSession holds the conversation state
type ChatSession struct {
	SessionID string
	EpisodeID string
	Messages  []Message
}

// Message represents a single conversation turn
type Message struct {
	UserQuery   string
	LLMResponse string
	Timestamp   time.Time
}

// ChatClients holds all the clients needed for the chat application
type ChatClients struct {
	GlobalGraphiti *graphiti.Client // Global knowledge base (can be nil)
	UserGraphiti   *graphiti.Client // User-specific episodic memory
	LLM            llm.Client
	Context        context.Context
}

func main() {
	// Command-line flags
	userID := flag.String("user-id", "alice", "User ID for the chat session")
	globalDBPath := flag.String("global-db", "./knowledge_db/content_graph.ladybugdb", "Path to global knowledge base (ladybug database)")
	userDBDir := flag.String("user-db-dir", "./user_dbs", "Directory for user-specific databases")
	skipGlobal := flag.Bool("skip-global", false, "Skip loading global knowledge base")
	flag.Parse()

	// Get OpenAI API key
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		fmt.Println("❌ OPENAI_API_KEY environment variable is not set.")
		fmt.Println("Please set it to run this example:")
		fmt.Println("  export OPENAI_API_KEY=your_api_key_here")
		os.Exit(1)
	}

	fmt.Println("🚀 Starting Graphiti Chat Example")
	fmt.Printf("   User ID: %s\n", *userID)
	fmt.Println()

	// Initialize clients
	clients, err := initializeClients(openaiAPIKey, *userID, *globalDBPath, *userDBDir, *skipGlobal)
	if err != nil {
		log.Fatalf("❌ Failed to initialize clients: %v", err)
	}
	defer clients.Close()

	// Run the chat loop
	runChatLoop(clients, *userID)
}

func initializeClients(apiKey, userID, globalDBPath, userDBDir string, skipGlobal bool) (*ChatClients, error) {
	ctx := context.Background()

	fmt.Println("🔧 Initializing clients...")

	// Create LLM client
	llmConfig := llm.Config{
		Model:       "gpt-4o-mini",
		Temperature: floatPtr(0.7),
		MaxTokens:   intPtr(2000),
	}
	baseLLMClient, err := llm.NewOpenAIClient(apiKey, llmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Wrap with retry logic
	llmClient := llm.NewRetryClient(baseLLMClient, llm.DefaultRetryConfig())
	fmt.Printf("   ✅ LLM client created (model: %s)\n", llmConfig.Model)

	// Create embedder client
	embedderConfig := embedder.Config{
		Model:      "text-embedding-3-small",
		BatchSize:  100,
		Dimensions: 1536,
	}
	embedderClient := embedder.NewOpenAIEmbedder(apiKey, embedderConfig)
	fmt.Printf("   ✅ Embedder client created (model: %s)\n", embedderConfig.Model)

	// Create global Graphiti client (if enabled)
	var globalGraphitiClient *graphiti.Client
	if !skipGlobal {
		// Check if global database exists
		if _, err := os.Stat(globalDBPath); err == nil {
			ladybugDriver, err := driver.NewLadybugDriver(globalDBPath, 1)
			if err != nil {
				fmt.Printf("   ⚠️  Failed to load global ladybug database: %v\n", err)
				fmt.Println("   Continuing without global knowledge base...")
			} else {
				globalConfig := &graphiti.Config{
					GroupID:  "global-knowledge",
					TimeZone: time.UTC,
				}
				globalGraphitiClient = graphiti.NewClient(ladybugDriver, llmClient, embedderClient, globalConfig, nil)
				fmt.Printf("   ✅ Global knowledge base loaded from %s\n", globalDBPath)
			}
		} else {
			fmt.Printf("   ℹ️  Global knowledge base not found at %s\n", globalDBPath)
			fmt.Println("   Continuing without global knowledge base...")
		}
	}

	// Create user-specific Graphiti client
	userDBPath := filepath.Join(userDBDir, fmt.Sprintf("user_%s.ladybugdb", userID))

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(userDBPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create user database directory: %w", err)
	}

	userLadybugDriver, err := driver.NewLadybugDriver(userDBPath, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to create user ladybug driver: %w", err)
	}

	userConfig := &graphiti.Config{
		GroupID:  fmt.Sprintf("user-%s-chat", userID),
		TimeZone: time.UTC,
	}
	userGraphitiClient := graphiti.NewClient(userLadybugDriver, llmClient, embedderClient, userConfig, nil)
	fmt.Printf("   ✅ User database initialized at %s\n", userDBPath)

	return &ChatClients{
		GlobalGraphiti: globalGraphitiClient,
		UserGraphiti:   userGraphitiClient,
		LLM:            llmClient,
		Context:        ctx,
	}, nil
}

func (c *ChatClients) Close() {
	if c.GlobalGraphiti != nil {
		c.GlobalGraphiti.Close(c.Context)
	}
	if c.UserGraphiti != nil {
		c.UserGraphiti.Close(c.Context)
	}
	if c.LLM != nil {
		c.LLM.Close()
	}
}

func runChatLoop(clients *ChatClients, userID string) {
	// Initialize chat session
	sessionID, err := uuid.NewV7()
	if err != nil {
		sessionID = uuid.New()
	}

	session := &ChatSession{
		SessionID: sessionID.String(),
		Messages:  []Message{},
	}

	// Print welcome banner
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("💬 Graphiti Interactive Chat")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("\nCommands:")
	fmt.Println("  Type your question and press Enter")
	fmt.Println("  Type 'exit' or 'quit' to end the session")
	fmt.Println("  Type 'history' to view conversation history")
	fmt.Println("  Type 'search <query>' to search the global knowledge base")
	fmt.Println(strings.Repeat("=", 70) + "\n")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("💬 You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// Handle commands
		switch {
		case input == "exit" || input == "quit":
			fmt.Println("\n👋 Goodbye!")
			return
		case input == "history":
			showHistory(session)
			continue
		case strings.HasPrefix(input, "search "):
			query := strings.TrimPrefix(input, "search ")
			searchGlobalKnowledge(clients, query)
			continue
		case input == "":
			continue
		}

		// Process the query
		processQuery(clients, session, userID, input)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("\n❌ Error reading input: %v\n", err)
	}
}

func processQuery(clients *ChatClients, session *ChatSession, userID, input string) {
	ctx := clients.Context

	// Format conversation turn
	conversationTurn := fmt.Sprintf("User: %s\n", input)

	// Add to episode (create or append)
	// Run in background to avoid freezing the chat
	if session.EpisodeID == "" {
		// First message - create initial episode with UUID v7
		episodeID, err := uuid.NewV7()
		if err != nil {
			episodeID = uuid.New()
		}

		episode := types.Episode{
			ID:        episodeID.String(),
			Name:      fmt.Sprintf("Chat with %s", userID),
			Content:   conversationTurn,
			GroupID:   fmt.Sprintf("user-%s-chat", userID),
			Metadata:  map[string]interface{}{"session_id": session.SessionID, "type": "chat"},
			Reference: time.Now(),
		}

		// Set the episode ID immediately so subsequent messages can reference it
		session.EpisodeID = episode.ID

		// Create episode in background
		go func(ep types.Episode) {
			result, err := clients.UserGraphiti.Add(ctx, []types.Episode{ep}, nil)
			if err != nil {
				fmt.Printf("⚠️  Background: Failed to create episode: %v\n", err)
			} else if result != nil && len(result.Episodes) > 0 {
				fmt.Printf("✨ Background: Created episode %s\n", result.Episodes[0].Uuid)
			}
		}(episode)
	} else {
		// Subsequent messages - append to existing episode in background
		go func(episodeID, content string) {
			_, err := clients.UserGraphiti.AddToEpisode(ctx, episodeID, content, nil)
			if err != nil {
				fmt.Printf("⚠️  Background: Failed to append to episode: %v\n", err)
			}
		}(session.EpisodeID, conversationTurn)
	}

	// Search global knowledge base if available
	var contextNodes []*types.Node
	if clients.GlobalGraphiti != nil {
		fmt.Println("🔍 Searching global knowledge base...")
		searchConfig := &types.SearchConfig{
			Limit:              5,
			CenterNodeDistance: 2,
			MinScore:           0.0,
			IncludeEdges:       true,
		}

		results, err := clients.GlobalGraphiti.Search(ctx, input, searchConfig)
		if err != nil {
			fmt.Printf("⚠️  Search failed: %v\n", err)
		} else if results != nil && len(results.Nodes) > 0 {
			contextNodes = results.Nodes
			fmt.Printf("📚 Found %d relevant nodes\n", len(contextNodes))
			for i, node := range contextNodes {
				if i >= 3 {
					break
				}
				summary := truncate(node.Summary, 80)
				fmt.Printf("  %d. %s: %s\n", i+1, node.Name, summary)
			}
		} else {
			fmt.Println("📚 No relevant context found")
		}
	}

	// Build prompt with context
	prompt := buildPrompt(input, session.Messages, contextNodes)

	// Generate response
	fmt.Println("\n🤖 Assistant:")
	fmt.Println(strings.Repeat("-", 70))

	messages := []types.Message{
		{Role: llm.RoleUser, Content: prompt},
	}

	llmResponse, err := clients.LLM.Chat(ctx, messages)
	if err != nil {
		fmt.Printf("❌ Failed to generate response: %v\n", err)
		return
	}

	response := llmResponse.Content
	fmt.Println(response)
	fmt.Println(strings.Repeat("-", 70))

	// Save message to session
	message := Message{
		UserQuery:   input,
		LLMResponse: response,
		Timestamp:   time.Now(),
	}
	session.Messages = append(session.Messages, message)

	// Append assistant response to episode in background
	if session.EpisodeID != "" {
		go func(episodeID, resp string) {
			assistantTurn := fmt.Sprintf("Assistant: %s\n", resp)
			_, err := clients.UserGraphiti.AddToEpisode(ctx, episodeID, assistantTurn, nil)
			if err != nil {
				fmt.Printf("⚠️  Background: Failed to append assistant response: %v\n", err)
			}
		}(session.EpisodeID, response)
	}
}

func buildPrompt(query string, history []Message, contextNodes []*types.Node) string {
	var prompt strings.Builder

	prompt.WriteString("You are a helpful AI assistant. ")

	// Add context from knowledge base
	if len(contextNodes) > 0 {
		prompt.WriteString("Use the following context from the knowledge base to help answer the question:\n\n")
		for i, node := range contextNodes {
			if i >= 3 {
				break
			}
			prompt.WriteString(fmt.Sprintf("Context %d: %s\n%s\n\n", i+1, node.Name, node.Summary))
		}
	}

	// Add conversation history (last 5 messages)
	if len(history) > 0 {
		prompt.WriteString("Previous conversation:\n")
		start := 0
		if len(history) > 5 {
			start = len(history) - 5
		}
		for _, msg := range history[start:] {
			prompt.WriteString(fmt.Sprintf("User: %s\nAssistant: %s\n", msg.UserQuery, msg.LLMResponse))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString(fmt.Sprintf("Current question: %s\n\nPlease provide a helpful and accurate response:", query))
	return prompt.String()
}

func showHistory(session *ChatSession) {
	if len(session.Messages) == 0 {
		fmt.Println("📝 No conversation history yet")
		return
	}

	fmt.Println("\n📝 Conversation History:")
	fmt.Println(strings.Repeat("-", 70))
	for i, msg := range session.Messages {
		fmt.Printf("%d. You: %s\n", i+1, msg.UserQuery)
		fmt.Printf("   Assistant: %s\n", truncate(msg.LLMResponse, 100))
		fmt.Println(strings.Repeat("-", 70))
	}
}

func searchGlobalKnowledge(clients *ChatClients, query string) {
	if clients.GlobalGraphiti == nil {
		fmt.Println("⚠️  Global knowledge base not available")
		return
	}

	fmt.Printf("🔍 Searching for: %s\n", query)

	searchConfig := &types.SearchConfig{
		Limit:              10,
		CenterNodeDistance: 2,
		MinScore:           0.0,
		IncludeEdges:       true,
	}

	results, err := clients.GlobalGraphiti.Search(clients.Context, query, searchConfig)
	if err != nil {
		fmt.Printf("❌ Search failed: %v\n", err)
		return
	}

	if results == nil || len(results.Nodes) == 0 {
		fmt.Println("📚 No results found")
		return
	}

	fmt.Printf("📚 Found %d nodes:\n", len(results.Nodes))
	for i, node := range results.Nodes {
		fmt.Printf("%d. %s (%s)\n", i+1, node.Name, node.Type)
		if node.Summary != "" {
			fmt.Printf("   %s\n", truncate(node.Summary, 100))
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func floatPtr(f float32) *float32 {
	return &f
}

func intPtr(i int) *int {
	return &i
}
