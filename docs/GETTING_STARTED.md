# Getting Started with go-graphiti

This guide will help you get started with go-graphiti, a temporal knowledge graph library for Go.

## Prerequisites

- Go 1.24 or later
- Neo4j database (local or cloud)
- OpenAI API key (for LLM and embeddings)

## Installation

```bash
go get github.com/getzep/go-graphiti
```

## Basic Setup

### 1. Environment Configuration

Create a `.env` file based on the provided example:

```bash
# Copy the example environment file
cp .env.example .env

# Edit with your values
OPENAI_API_KEY=sk-your-openai-api-key
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-password
NEO4J_DATABASE=neo4j
```

### 2. Neo4j Setup

#### Option A: Local Neo4j with Docker

```bash
# Start Neo4j with Docker
docker run \
    --name neo4j \
    -p 7474:7474 -p 7687:7687 \
    -e NEO4J_AUTH=neo4j/password \
    neo4j:latest
```

#### Option B: Neo4j Desktop

1. Download and install [Neo4j Desktop](https://neo4j.com/download/)
2. Create a new project and database
3. Set the password and note the connection details

#### Option C: Neo4j Aura (Cloud)

1. Sign up for [Neo4j Aura](https://neo4j.com/cloud/aura/)
2. Create a new database instance
3. Note the connection URI and credentials

### 3. Basic Usage Example

Create `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/getzep/go-graphiti"
    "github.com/getzep/go-graphiti/pkg/driver"
    "github.com/getzep/go-graphiti/pkg/embedder"
    "github.com/getzep/go-graphiti/pkg/llm"
    "github.com/getzep/go-graphiti/pkg/types"
)

func main() {
    ctx := context.Background()

    // Create Neo4j driver
    neo4jDriver, err := driver.NewNeo4jDriver(
        os.Getenv("NEO4J_URI"),
        os.Getenv("NEO4J_USER"),
        os.Getenv("NEO4J_PASSWORD"),
        os.Getenv("NEO4J_DATABASE"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer neo4jDriver.Close(ctx)

    // Create LLM client
    llmConfig := llm.Config{
        Model:       "gpt-4o-mini",
        Temperature: &[]float32{0.7}[0],
    }
    llmClient := llm.NewOpenAIClient(os.Getenv("OPENAI_API_KEY"), llmConfig)

    // Create embedder
    embedderConfig := embedder.Config{
        Model:     "text-embedding-3-small",
        BatchSize: 100,
    }
    embedderClient := embedder.NewOpenAIEmbedder(os.Getenv("OPENAI_API_KEY"), embedderConfig)

    // Create Graphiti client
    config := &graphiti.Config{
        GroupID:  "getting-started",
        TimeZone: time.UTC,
    }
    client := graphiti.NewClient(neo4jDriver, llmClient, embedderClient, config)
    defer client.Close(ctx)

    fmt.Println("Graphiti client created successfully!")
}
```

Run the example:

```bash
go run main.go
```

## Core Concepts

### Episodes

Episodes are temporal data units that you add to the knowledge graph. They represent events, conversations, documents, or any time-bound information.

```go
episodes := []types.Episode{
    {
        ID:        "meeting-001",
        Name:      "Weekly Team Standup",
        Content:   "Alice reported progress on the API integration. Bob mentioned issues with the database connection. Carol suggested using connection pooling.",
        Reference: time.Now().Add(-2 * time.Hour), // 2 hours ago
        CreatedAt: time.Now(),
        GroupID:   "team-alpha",
        Metadata: map[string]interface{}{
            "meeting_type": "standup",
            "duration":     "30min",
        },
    },
}

// Add to knowledge graph
err := client.Add(ctx, episodes)
if err != nil {
    log.Fatal(err)
}
```

### Nodes

Nodes represent entities in your knowledge graph:

- **EntityNode**: People, places, concepts, objects
- **EpisodicNode**: Events, meetings, conversations
- **CommunityNode**: Groups of related entities

### Edges

Edges represent relationships between nodes:

- **EntityEdge**: "Alice works with Bob"
- **EpisodicEdge**: "Meeting occurred in Conference Room A"  
- **CommunityEdge**: "Engineering Team includes Alice and Bob"

### Search

Perform hybrid search combining semantic similarity, keywords, and graph traversal:

```go
// Basic search
results, err := client.Search(ctx, "database connection issues", nil)
if err != nil {
    log.Fatal(err)
}

// Advanced search with configuration
searchConfig := &types.SearchConfig{
    Limit:              10,
    CenterNodeDistance: 3,
    MinScore:           0.1,
    IncludeEdges:       true,
    Rerank:             true,
}

results, err = client.Search(ctx, "API integration progress", searchConfig)
if err != nil {
    log.Fatal(err)
}

// Process results
for _, node := range results.Nodes {
    fmt.Printf("Found: %s (%s)\n", node.Name, node.Type)
}
```

## Configuration Options

### LLM Configuration

```go
llmConfig := llm.Config{
    Model:       "gpt-4o",           // Model name
    Temperature: &[]float32{0.7}[0], // Creativity (0.0-1.0)
    MaxTokens:   &[]int{2000}[0],    // Response length limit
    TopP:        &[]float32{0.9}[0], // Nucleus sampling
}
```

### Embedder Configuration

```go
embedderConfig := embedder.Config{
    Model:      "text-embedding-3-large", // Embedding model
    BatchSize:  50,                       // Batch processing size
    Dimensions: 3072,                     // Embedding dimensions
}
```

### Search Configuration

```go
searchConfig := &types.SearchConfig{
    Limit:              20,    // Max results
    CenterNodeDistance: 2,     // Graph traversal depth
    MinScore:           0.0,   // Minimum relevance
    IncludeEdges:       true,  // Include relationships
    Rerank:             false, // Apply reranking
    Filters: &types.SearchFilters{
        NodeTypes:   []types.NodeType{types.EntityNodeType},
        EntityTypes: []string{"Person", "Project"},
        TimeRange: &types.TimeRange{
            Start: time.Now().Add(-30 * 24 * time.Hour),
            End:   time.Now(),
        },
    },
}
```

## Error Handling

The library provides typed errors for common scenarios:

```go
node, err := client.GetNode(ctx, "nonexistent-id")
if err != nil {
    if errors.Is(err, graphiti.ErrNodeNotFound) {
        fmt.Println("Node not found")
    } else {
        log.Printf("Error: %v", err)
    }
}
```

## Multi-tenancy

Use GroupID to isolate data:

```go
// User-specific client
userConfig := &graphiti.Config{
    GroupID: fmt.Sprintf("user-%s", userID),
}

// Organization-specific client  
orgConfig := &graphiti.Config{
    GroupID: fmt.Sprintf("org-%s", orgID),
}
```

## Best Practices

### 1. Resource Management

Always close clients and drivers:

```go
defer client.Close(ctx)
defer neo4jDriver.Close(ctx)
```

### 2. Context Usage

Use context for timeouts and cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := client.Search(ctx, query, nil)
```

### 3. Error Handling

Handle specific error types:

```go
if err != nil {
    switch {
    case errors.Is(err, graphiti.ErrNodeNotFound):
        // Handle missing node
    case errors.Is(err, graphiti.ErrInvalidEpisode):
        // Handle invalid input
    default:
        // Handle other errors
    }
}
```

### 4. Batch Processing

Process episodes in batches for efficiency:

```go
const batchSize = 100

for i := 0; i < len(allEpisodes); i += batchSize {
    end := i + batchSize
    if end > len(allEpisodes) {
        end = len(allEpisodes)
    }
    
    batch := allEpisodes[i:end]
    if err := client.Add(ctx, batch); err != nil {
        log.Printf("Batch %d failed: %v", i/batchSize, err)
    }
}
```

## Next Steps

- Read the [Architecture Guide](ARCHITECTURE.md) for deeper understanding
- Check out the [API Reference](API_REFERENCE.md) for detailed documentation
- Explore the [examples/](../examples/) directory for more use cases
- Join our community for support and discussions

## Troubleshooting

### Common Issues

1. **Connection Failed**: Check Neo4j is running and credentials are correct
2. **API Key Error**: Verify OpenAI API key is valid and has sufficient credits
3. **Import Errors**: Ensure you're using Go 1.24+ and all dependencies are downloaded

### Getting Help

- Check the [FAQ](FAQ.md)
- Open an issue on GitHub
- Join our community discussions