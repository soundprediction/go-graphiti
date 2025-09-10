# Examples

This document provides practical examples of using go-graphiti for various use cases.

## Basic Examples

### 1. Simple Knowledge Building

Build a knowledge graph from basic episodes:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/soundprediction/go-graphiti"
    "github.com/soundprediction/go-graphiti/pkg/types"
)

func buildKnowledge(client graphiti.Graphiti) {
    ctx := context.Background()

    episodes := []types.Episode{
        {
            ID:        "intro-1",
            Name:      "Alice Introduction",
            Content:   "Alice is a software engineer who works on backend systems. She specializes in database optimization and has 5 years of experience.",
            Reference: time.Now().Add(-1 * time.Hour),
            CreatedAt: time.Now(),
            GroupID:   "company",
        },
        {
            ID:        "intro-2", 
            Name:      "Bob Introduction",
            Content:   "Bob is a frontend developer who loves React and TypeScript. He's been working on the user interface for our main product.",
            Reference: time.Now().Add(-45 * time.Minute),
            CreatedAt: time.Now(),
            GroupID:   "company",
        },
        {
            ID:        "meeting-1",
            Name:      "Project Sync",
            Content:   "Alice and Bob met to discuss the integration between the backend API and frontend components. They decided to use REST endpoints for now.",
            Reference: time.Now().Add(-30 * time.Minute),
            CreatedAt: time.Now(),
            GroupID:   "company",
        },
    }

    // Add episodes to build knowledge graph
    if err := client.Add(ctx, episodes); err != nil {
        log.Printf("Error adding episodes: %v", err)
        return
    }

    fmt.Println("Knowledge graph built successfully!")

    // Search for information
    results, err := client.Search(ctx, "Who works on backend systems?", nil)
    if err != nil {
        log.Printf("Search error: %v", err)
        return
    }

    fmt.Printf("Found %d relevant nodes:\n", len(results.Nodes))
    for _, node := range results.Nodes {
        fmt.Printf("- %s: %s\n", node.Name, node.Summary)
    }
}
```

### 2. Meeting Minutes Processing

Process meeting minutes into structured knowledge:

```go
func processMeetingMinutes(client graphiti.Graphiti) {
    ctx := context.Background()

    meetingMinutes := `
    Meeting: Q4 Planning Session
    Date: 2024-12-15
    Attendees: Sarah (Product Manager), Mike (Tech Lead), Lisa (Designer)
    
    Agenda Items:
    1. Review Q3 performance metrics
    2. Plan Q4 feature roadmap
    3. Resource allocation discussion
    
    Key Decisions:
    - Prioritize mobile app development
    - Hire 2 additional engineers
    - Launch beta program in January
    
    Action Items:
    - Sarah: Create user research plan by Friday
    - Mike: Technical architecture proposal by next week  
    - Lisa: Mobile UI mockups by Thursday
    `

    episode := types.Episode{
        ID:        "meeting-q4-planning",
        Name:      "Q4 Planning Session",
        Content:   meetingMinutes,
        Reference: time.Date(2024, 12, 15, 14, 0, 0, 0, time.UTC),
        CreatedAt: time.Now(),
        GroupID:   "company",
        Metadata: map[string]interface{}{
            "meeting_type": "planning",
            "quarter":      "Q4",
            "attendees":    []string{"Sarah", "Mike", "Lisa"},
        },
    }

    if err := client.Add(ctx, []types.Episode{episode}); err != nil {
        log.Fatal(err)
    }

    // Query for action items
    results, err := client.Search(ctx, "action items assignments", nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Action items found:")
    for _, node := range results.Nodes {
        if node.Type == types.EntityNodeType {
            fmt.Printf("- %s\n", node.Summary)
        }
    }
}
```

## Advanced Examples

### 3. Multi-Tenant Knowledge Management

Handle multiple organizations or users:

```go
func multiTenantExample() {
    // Client for Organization A
    configA := &graphiti.Config{
        GroupID: "org-a",
        TimeZone: time.UTC,
    }
    clientA := createClient(configA)
    defer clientA.Close(context.Background())

    // Client for Organization B
    configB := &graphiti.Config{
        GroupID: "org-b", 
        TimeZone: time.UTC,
    }
    clientB := createClient(configB)
    defer clientB.Close(context.Background())

    // Add data for each organization
    ctx := context.Background()

    // Organization A data
    episodesA := []types.Episode{
        {
            ID:        "org-a-proj-1",
            Content:   "Project Alpha is making good progress. Team lead John reports 80% completion.",
            GroupID:   "org-a",
            Reference: time.Now(),
            CreatedAt: time.Now(),
        },
    }

    // Organization B data  
    episodesB := []types.Episode{
        {
            ID:        "org-b-proj-1", 
            Content:   "Project Beta launched successfully. Sarah leads a team of 5 developers.",
            GroupID:   "org-b",
            Reference: time.Now(),
            CreatedAt: time.Now(),
        },
    }

    // Each client only sees its own organization's data
    clientA.Add(ctx, episodesA)
    clientB.Add(ctx, episodesB)

    // Search results are isolated
    resultsA, _ := clientA.Search(ctx, "project progress", nil)
    resultsB, _ := clientB.Search(ctx, "project progress", nil)

    fmt.Printf("Org A found %d results\n", len(resultsA.Nodes))
    fmt.Printf("Org B found %d results\n", len(resultsB.Nodes))
}
```

### 4. Customer Support Knowledge Base

Build a customer support knowledge base:

```go
func customerSupportKB(client graphiti.Graphiti) {
    ctx := context.Background()

    supportEpisodes := []types.Episode{
        {
            ID:        "ticket-1001",
            Name:      "Login Issue Resolution",
            Content:   "Customer Jane Smith reported unable to login. Issue was caused by browser cache. Resolved by clearing cache and cookies. Ticket closed.",
            Reference: time.Now().Add(-2 * time.Hour),
            GroupID:   "support",
            Metadata: map[string]interface{}{
                "ticket_id":   "1001",
                "customer":    "Jane Smith",
                "category":    "authentication",
                "resolution":  "cache_clear",
                "severity":    "medium",
            },
        },
        {
            ID:        "ticket-1002",
            Name:      "Payment Processing Error", 
            Content:   "Customer Bob Wilson experienced payment failure during checkout. Error was due to expired credit card. Customer updated payment method and transaction succeeded.",
            Reference: time.Now().Add(-1 * time.Hour),
            GroupID:   "support",
            Metadata: map[string]interface{}{
                "ticket_id":  "1002",
                "customer":   "Bob Wilson", 
                "category":   "payment",
                "resolution": "payment_method_update",
                "severity":   "high",
            },
        },
        {
            ID:        "faq-1",
            Name:      "Password Reset Process",
            Content:   "To reset your password: 1) Go to login page 2) Click 'Forgot Password' 3) Enter your email 4) Check email for reset link 5) Follow link and create new password",
            Reference: time.Now().Add(-1 * time.Week),
            GroupID:   "support",
            Metadata: map[string]interface{}{
                "type":     "faq",
                "category": "authentication",
            },
        },
    }

    if err := client.Add(ctx, supportEpisodes); err != nil {
        log.Fatal(err)
    }

    // Advanced search with filters
    filters := &types.SearchFilters{
        TimeRange: &types.TimeRange{
            Start: time.Now().Add(-24 * time.Hour),
            End:   time.Now(),
        },
    }

    config := &types.SearchConfig{
        Limit:    10,
        Filters:  filters,
        Rerank:   true,
    }

    // Search for authentication issues
    results, err := client.Search(ctx, "login authentication problems", config)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Recent authentication issues:")
    for _, node := range results.Nodes {
        fmt.Printf("- %s (%.2f relevance)\n", 
            node.Name, node.Metadata["score"])
    }
}
```

### 5. Research Paper Analysis

Analyze and connect research papers:

```go
func researchPaperAnalysis(client graphiti.Graphiti) {
    ctx := context.Background()

    papers := []types.Episode{
        {
            ID:      "paper-transformer",
            Name:    "Attention Is All You Need",
            Content: "The Transformer architecture revolutionized natural language processing by introducing self-attention mechanisms. Authors: Vaswani et al. Key contributions: eliminated recurrence, improved parallelization, achieved state-of-the-art results on translation tasks.",
            Reference: time.Date(2017, 6, 12, 0, 0, 0, 0, time.UTC),
            GroupID: "research",
            Metadata: map[string]interface{}{
                "authors":     []string{"Vaswani", "Shazeer", "Parmar", "Uszkoreit"},
                "venue":       "NIPS 2017",
                "arxiv_id":    "1706.03762",
                "citations":   75000,
                "field":       "NLP",
                "keywords":    []string{"attention", "transformer", "neural machine translation"},
            },
        },
        {
            ID:      "paper-bert",
            Name:    "BERT: Pre-training of Deep Bidirectional Transformers",
            Content: "BERT introduced bidirectional pre-training for language representations. Built on Transformer architecture. Authors: Devlin et al. Achieved new state-of-the-art on 11 NLP tasks including GLUE benchmark.",
            Reference: time.Date(2018, 10, 11, 0, 0, 0, 0, time.UTC),
            GroupID: "research",
            Metadata: map[string]interface{}{
                "authors":     []string{"Devlin", "Chang", "Lee", "Toutanova"},
                "venue":       "NAACL 2019",
                "arxiv_id":    "1810.04805", 
                "citations":   45000,
                "field":       "NLP",
                "keywords":    []string{"bert", "pre-training", "bidirectional", "transformer"},
                "builds_on":   []string{"paper-transformer"},
            },
        },
        {
            ID:      "paper-gpt",
            Name:    "Language Models are Unsupervised Multitask Learners",
            Content: "GPT-2 demonstrated that language models can perform many NLP tasks without task-specific training. Authors: Radford et al. Showed emergent abilities with scale. Used transformer decoder architecture.",
            Reference: time.Date(2019, 2, 14, 0, 0, 0, 0, time.UTC),
            GroupID: "research", 
            Metadata: map[string]interface{}{
                "authors":     []string{"Radford", "Wu", "Child", "Luan"},
                "venue":       "OpenAI Blog",
                "citations":   25000,
                "field":       "NLP",
                "keywords":    []string{"gpt", "language model", "unsupervised", "multitask"},
                "builds_on":   []string{"paper-transformer"},
            },
        },
    }

    if err := client.Add(ctx, papers); err != nil {
        log.Fatal(err)
    }

    // Search for papers about attention mechanisms
    results, err := client.Search(ctx, "attention mechanisms transformers", nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Papers about attention mechanisms:")
    for _, node := range results.Nodes {
        if node.Type == types.EpisodicNodeType {
            fmt.Printf("- %s (%s)\n", node.Name, 
                node.Metadata["venue"])
        }
    }

    // Find influential authors
    authorResults, err := client.Search(ctx, "Vaswani Devlin Radford authors", nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nInfluential authors found:")
    for _, node := range authorResults.Nodes {
        if node.Type == types.EntityNodeType && node.EntityType == "Person" {
            fmt.Printf("- %s: %s\n", node.Name, node.Summary)
        }
    }
}
```

### 6. Personal Knowledge Management

Build a personal knowledge base:

```go
func personalKnowledgeManagement(client graphiti.Graphiti) {
    ctx := context.Background()

    personalNotes := []types.Episode{
        {
            ID:      "book-summary-1",
            Name:    "Clean Code Summary",
            Content: "Key takeaways from Clean Code by Robert Martin: Write code for humans, not computers. Use meaningful names. Keep functions small. Comments should explain why, not what. Practice refactoring regularly.",
            Reference: time.Now().Add(-3 * 24 * time.Hour),
            GroupID: "personal",
            Metadata: map[string]interface{}{
                "type":     "book_summary",
                "author":   "Robert Martin",
                "topic":    "software_engineering",
                "rating":   9,
                "finished": true,
            },
        },
        {
            ID:      "learning-note-1", 
            Name:    "Go Concurrency Patterns",
            Content: "Learned about Go concurrency patterns today. Key concepts: goroutines are lightweight threads, channels for communication, select for multiplexing. Fan-out/fan-in pattern useful for parallel processing.",
            Reference: time.Now().Add(-1 * 24 * time.Hour),
            GroupID: "personal",
            Metadata: map[string]interface{}{
                "type":         "learning_note",
                "technology":   "golang",
                "topic":        "concurrency",
                "difficulty":   "intermediate",
                "time_spent":   "2 hours",
            },
        },
        {
            ID:      "project-idea-1",
            Name:    "Personal Finance Tracker",
            Content: "Idea for personal project: Build a finance tracker using Go backend and React frontend. Features: expense tracking, budget management, investment portfolio tracking. Could use this to practice full-stack development.",
            Reference: time.Now().Add(-12 * time.Hour),
            GroupID: "personal", 
            Metadata: map[string]interface{}{
                "type":        "project_idea",
                "status":      "idea",
                "technologies": []string{"go", "react", "postgres"},
                "priority":    "medium",
            },
        },
    }

    if err := client.Add(ctx, personalNotes); err != nil {
        log.Fatal(err)
    }

    // Search for learning materials about Go
    results, err := client.Search(ctx, "golang programming learning", nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Go learning resources:")
    for _, node := range results.Nodes {
        fmt.Printf("- %s\n", node.Name)
        if len(node.Summary) > 0 {
            fmt.Printf("  %s\n", node.Summary)
        }
    }

    // Find project ideas
    projectConfig := &types.SearchConfig{
        Limit: 5,
        Filters: &types.SearchFilters{
            // Add metadata filters if needed
        },
    }

    projectResults, err := client.Search(ctx, "project ideas programming", projectConfig)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nProject ideas:")
    for _, node := range projectResults.Nodes {
        if node.Metadata["type"] == "project_idea" {
            fmt.Printf("- %s (Priority: %s)\n", 
                node.Name, node.Metadata["priority"])
        }
    }
}
```

## Local Setup Examples

### Complete Local Setup with Kuzu + Ollama

For maximum privacy and control, you can run go-graphiti entirely locally using:
- **Kuzu**: Embedded graph database (no server required)
- **Ollama**: Local LLM inference (no cloud API required)  
- **Local embeddings**: Optional local embedding service

**Complete example**: See [`examples/kuzu_ollama/`](../examples/kuzu_ollama/) for a full working example.

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/soundprediction/go-graphiti"
    "github.com/soundprediction/go-graphiti/pkg/driver"
    "github.com/soundprediction/go-graphiti/pkg/embedder"
    "github.com/soundprediction/go-graphiti/pkg/llm"
)

func main() {
    ctx := context.Background()

    // 1. Embedded graph database (local file)
    kuzuDriver, err := driver.NewKuzuDriver("./my_graph.db")
    if err != nil {
        log.Fatal(err)
    }
    defer kuzuDriver.Close(ctx)

    // 2. Local LLM inference with Ollama
    ollama, err := llm.NewOllamaClient("", "llama2:7b", llm.Config{
        Temperature: &[]float32{0.7}[0],
        MaxTokens:   &[]int{1000}[0],
    })
    if err != nil {
        log.Fatal(err)
    }
    defer ollama.Close()

    // 3. Embeddings (could be local too)
    embedder := embedder.NewOpenAIEmbedder("", embedder.Config{
        Model: "text-embedding-3-small",
    })
    defer embedder.Close()

    // 4. Create fully local Graphiti client
    client := graphiti.NewClient(kuzuDriver, ollama, embedder, &graphiti.Config{
        GroupID: "local-setup",
    })
    defer client.Close(ctx)

    // Use normally - everything runs locally!
    // (Current implementation uses stub drivers)
    log.Println("Local Graphiti client ready!")
}
```

**Benefits of Local Setup**:
- 🔒 **Privacy**: All data stays on your machine
- ⚡ **Performance**: No network latency for graph queries
- 💰 **Cost**: No cloud hosting or API charges
- 🛠️ **Development**: Easy to version control and test

**Requirements**:
- Ollama installed and running (`ollama serve`)
- Model downloaded (`ollama pull llama2:7b`)
- Sufficient RAM for local model (4-8GB recommended)

**Current Status**: Kuzu driver is implemented as a stub. Full functionality will be available when the Kuzu Go library is released.

### Alternative Local LLM Services

You can also use other local LLM services:

```go
// LocalAI
localAI, err := llm.NewLocalAIClient("http://localhost:8080", "gpt-3.5-turbo", llm.Config{})

// vLLM server  
vllm, err := llm.NewVLLMClient("http://localhost:8000", "microsoft/DialoGPT-medium", llm.Config{})

// Any OpenAI-compatible service
custom, err := llm.NewOpenAICompatibleClient("http://localhost:1234", "", "my-model", llm.Config{})
```

## Utility Functions

### Helper Functions for Common Tasks

```go
// Create a standard client configuration
func createStandardClient() graphiti.Graphiti {
    // Neo4j driver
    driver, err := driver.NewNeo4jDriver(
        os.Getenv("NEO4J_URI"),
        os.Getenv("NEO4J_USER"),
        os.Getenv("NEO4J_PASSWORD"),
        os.Getenv("NEO4J_DATABASE"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // LLM client
    llmConfig := llm.Config{
        Model:       "gpt-4o-mini",
        Temperature: &[]float32{0.7}[0],
        MaxTokens:   &[]int{2000}[0],
    }
    llmClient := llm.NewOpenAIClient(os.Getenv("OPENAI_API_KEY"), llmConfig)

    // Embedder client
    embedConfig := embedder.Config{
        Model:     "text-embedding-3-small",
        BatchSize: 100,
    }
    embedderClient := embedder.NewOpenAIEmbedder(os.Getenv("OPENAI_API_KEY"), embedConfig)

    // Graphiti configuration
    config := &graphiti.Config{
        GroupID:  "default",
        TimeZone: time.UTC,
    }

    return graphiti.NewClient(driver, llmClient, embedderClient, config)
}

// Batch process episodes
func batchProcessEpisodes(client graphiti.Graphiti, episodes []types.Episode, batchSize int) error {
    ctx := context.Background()
    
    for i := 0; i < len(episodes); i += batchSize {
        end := i + batchSize
        if end > len(episodes) {
            end = len(episodes)
        }
        
        batch := episodes[i:end]
        if err := client.Add(ctx, batch); err != nil {
            return fmt.Errorf("batch %d failed: %w", i/batchSize, err)
        }
        
        fmt.Printf("Processed batch %d/%d\n", 
            i/batchSize+1, (len(episodes)+batchSize-1)/batchSize)
    }
    
    return nil
}

// Search with retry logic
func searchWithRetry(client graphiti.Graphiti, query string, maxRetries int) (*types.SearchResults, error) {
    var results *types.SearchResults
    var err error
    
    for i := 0; i < maxRetries; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        results, err = client.Search(ctx, query, nil)
        cancel()
        
        if err == nil {
            return results, nil
        }
        
        if i < maxRetries-1 {
            time.Sleep(time.Duration(i+1) * time.Second)
        }
    }
    
    return nil, fmt.Errorf("search failed after %d retries: %w", maxRetries, err)
}

// Print search results in a formatted way
func printSearchResults(results *types.SearchResults) {
    fmt.Printf("Search Results for '%s':\n", results.Query)
    fmt.Printf("Found %d nodes, %d edges (total: %d)\n\n", 
        len(results.Nodes), len(results.Edges), results.Total)

    // Group nodes by type
    nodesByType := make(map[types.NodeType][]*types.Node)
    for _, node := range results.Nodes {
        nodesByType[node.Type] = append(nodesByType[node.Type], node)
    }

    // Print each type
    for nodeType, nodes := range nodesByType {
        fmt.Printf("%s Nodes (%d):\n", strings.Title(string(nodeType)), len(nodes))
        for _, node := range nodes {
            fmt.Printf("  - %s", node.Name)
            if len(node.Summary) > 0 {
                fmt.Printf(": %s", node.Summary)
            }
            fmt.Printf(" (Created: %s)\n", node.CreatedAt.Format("2006-01-02"))
        }
        fmt.Println()
    }

    if len(results.Edges) > 0 {
        fmt.Printf("Relationships (%d):\n", len(results.Edges))
        for _, edge := range results.Edges {
            fmt.Printf("  - %s -> %s", edge.SourceID, edge.TargetID)
            if len(edge.Name) > 0 {
                fmt.Printf(" (%s)", edge.Name)
            }
            fmt.Println()
        }
    }
}
```

## Error Handling Examples

```go
func handleErrors(client graphiti.Graphiti) {
    ctx := context.Background()

    // Handle node not found
    node, err := client.GetNode(ctx, "nonexistent-id")
    if err != nil {
        if errors.Is(err, graphiti.ErrNodeNotFound) {
            fmt.Println("Node not found - this is expected")
        } else {
            log.Printf("Unexpected error: %v", err)
        }
    }

    // Handle search errors
    results, err := client.Search(ctx, "test query", nil)
    if err != nil {
        log.Printf("Search failed: %v", err)
        return
    }

    if len(results.Nodes) == 0 {
        fmt.Println("No results found for query")
    }

    // Handle episode processing errors
    episodes := []types.Episode{
        {
            ID: "test-episode",
            // Missing required fields to trigger validation
        },
    }

    if err := client.Add(ctx, episodes); err != nil {
        if errors.Is(err, graphiti.ErrInvalidEpisode) {
            fmt.Println("Episode validation failed")
        } else {
            log.Printf("Processing error: %v", err)
        }
    }
}
```

These examples demonstrate various use cases and patterns for working with go-graphiti. The library's flexibility allows it to be adapted for many different knowledge management scenarios while maintaining temporal awareness and multi-tenancy support.
