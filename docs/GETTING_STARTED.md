# Getting Started with go-predicato

This guide will help you get started with go-predicato, a temporal knowledge graph library for Go.

## Prerequisites

- Go 1.24 or later
- **Optional**: External graph database (Neo4j or FalkorDB)
- **Optional**: LLM API access (OpenAI, Ollama, vLLM, or any OpenAI-compatible service)

> **Note**: go-predicato works out-of-the-box with embedded ladybug database and no external dependencies!

## Installation

```bash
go get github.com/soundprediction/go-predicato
```

## Quick Start Options

### Option 1: Minimal Setup (No External Dependencies)

The simplest way to get started is with embedded ladybug database and no LLM:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/soundprediction/go-predicato"
    "github.com/soundprediction/go-predicato/pkg/driver"
)

func main() {
    ctx := context.Background()

    // Create ladybug driver (embedded database)
    ladybugDriver, err := driver.NewLadybugDriver("./ladybug_db")
    if err != nil {
        log.Fatal(err)
    }
    defer ladybugDriver.Close(ctx)

    // Create Predicato client
    config := &predicato.Config{
        GroupID:  "my-group",
        TimeZone: time.UTC,
    }
    client := predicato.NewClient(ladybugDriver, nil, nil, config)
    defer client.Close(ctx)

    // Your code here...
}
```

### Option 2: With Local LLM (Ollama)

For enhanced features with a local LLM:

**1. Install Ollama:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull llama3.2
```

**2. Set environment variables:**
```bash
export LLM_BASE_URL="http://localhost:11434"
export EMBEDDER_MODEL_NAME="llama3.2"  # Or any embedding model
```

**3. Use in code:**
```go
// Create LLM client for Ollama
llmConfig := llm.Config{
    Model:   "llama3.2",
    BaseURL: "http://localhost:11434",
}
llmClient, err := llm.NewOpenAIClient("dummy", llmConfig)  // API key not needed for Ollama

// Create Predicato client with LLM
client := predicato.NewClient(ladybugDriver, llmClient, nil, config)
```

### Option 3: Full Setup with External Services

For production use with OpenAI and external databases:

**1. Environment Configuration:**

Create a `.env` file:

```bash
# OpenAI (or compatible service)
OPENAI_API_KEY=sk-your-openai-api-key
LLM_BASE_URL=https://api.openai.com/v1  # Optional: for custom endpoints

# External Graph Database (optional)
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-password
NEO4J_DATABASE=neo4j
```

**2. Database Setup (Optional - only for Neo4j):**

> **Note**: This is only needed if you want to use Neo4j instead of the default ladybug embedded database.

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

## Complete Examples

### Example 1: Minimal Setup (Recommended for Getting Started)

Create `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/soundprediction/go-predicato"
    "github.com/soundprediction/go-predicato/pkg/driver"
    "github.com/soundprediction/go-predicato/pkg/types"
)

func main() {
    ctx := context.Background()

    // Create ladybug driver (embedded database - no setup required)
    ladybugDriver, err := driver.NewLadybugDriver("./ladybug_db")
    if err != nil {
        log.Fatal(err)
    }
    defer ladybugDriver.Close(ctx)

    // Create Predicato client (no LLM or embedder required for basic usage)
    config := &predicato.Config{
        GroupID:  "getting-started",
        TimeZone: time.UTC,
    }
    client := predicato.NewClient(ladybugDriver, nil, nil, config)
    defer client.Close(ctx)

    fmt.Println("Predicato client created successfully with embedded database!")

    // Add some sample data
    episodes := []types.Episode{
        {
            ID:        "episode-1",
            Name:      "First Episode",
            Content:   "This is my first episode in the knowledge graph",
            Reference: time.Now(),
            CreatedAt: time.Now(),
            GroupID:   "getting-started",
        },
    }

    err = client.Add(ctx, episodes)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Episode added successfully!")
}
```

### Example 2: With OpenAI-Compatible LLM

For enhanced features with semantic understanding:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/soundprediction/go-predicato"
    "github.com/soundprediction/go-predicato/pkg/driver"
    "github.com/soundprediction/go-predicato/pkg/embedder"
    "github.com/soundprediction/go-predicato/pkg/llm"
    "github.com/soundprediction/go-predicato/pkg/types"
)

func main() {
    ctx := context.Background()

    // Create ladybug driver (still using embedded database)
    ladybugDriver, err := driver.NewLadybugDriver("./ladybug_db")
    if err != nil {
        log.Fatal(err)
    }
    defer ladybugDriver.Close(ctx)

    // Create LLM client (works with OpenAI, Ollama, vLLM, etc.)
    llmConfig := llm.Config{
        Model:       "gpt-4o-mini",  // or "llama3.2", "mistral", etc.
        Temperature: &[]float32{0.7}[0],
        BaseURL:     os.Getenv("LLM_BASE_URL"), // Optional: for local LLMs
    }
    llmClient, err := llm.NewOpenAIClient(os.Getenv("OPENAI_API_KEY"), llmConfig)
    if err != nil {
        log.Fatal(err)
    }

    // Create embedder (optional but recommended for semantic search)
    embedderConfig := embedder.Config{
        Model:     "text-embedding-3-small",  // or local embedding model
        BaseURL:   os.Getenv("EMBEDDING_BASE_URL"), // Optional: for local embeddings
    }
    embedderClient := embedder.NewOpenAIEmbedder(os.Getenv("OPENAI_API_KEY"), embedderConfig)

    // Create Predicato client
    config := &predicato.Config{
        GroupID:  "getting-started",
        TimeZone: time.UTC,
    }
    client := predicato.NewClient(ladybugDriver, llmClient, embedderClient, config)
    defer client.Close(ctx)

    fmt.Println("Predicato client created with LLM integration!")
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
    if errors.Is(err, predicato.ErrNodeNotFound) {
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
userConfig := &predicato.Config{
    GroupID: fmt.Sprintf("user-%s", userID),
}

// Organization-specific client  
orgConfig := &predicato.Config{
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
    case errors.Is(err, predicato.ErrNodeNotFound):
        // Handle missing node
    case errors.Is(err, predicato.ErrInvalidEpisode):
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