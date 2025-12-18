# go-graphiti

A production-ready Temporal Knowledge Graph library for Go, designed for building robust, context-aware AI applications.

## Core Advancements

* **Universal LLM Support**: Native support for any OpenAI-compatible provider (OpenAI, Anthropic, Together AI, Ollama, vLLM).
* **Cost & Budgeting**: Built-in real-time cost tracking with serverless pricing models (Together AI, OpenAI) and granular token usage analytics.
* **Resiliency**: Advanced routing capabilities with provider fallback, circuit breaking, and usage-based routing rules (e.g., routing HIPAA compliant requests to specific providers).
* **High Performance**: 
    * **Embedded Database**: Native KuzuDB support for zero-setup, high-performance graph storage.
    * **Caching**: BadgerDB-based caching layer for embeddings and LLM responses to reduce latency and costs.
    * **Protocol Buffers**: TSV-based prompting reduces token usage by 30-50% compared to JSON.
* **Observability**: Comprehensive error tracking and telemetry with DuckDB persistence.



## Features

- **Temporal Knowledge Graphs**: Bi-temporal data model with explicit tracking of event occurrence times
- **Hybrid Search**: Combines semantic embeddings, keyword search (BM25), and graph traversal
- **Multiple Graph Backends**: Primary support for embedded Kuzu database, also supports Memgraph and Neo4j
- **Flexible LLM Integration**: Works with any OpenAI-compatible API (OpenAI, Ollama, LocalAI, vLLM, etc.)
- **No Vendor Lock-in**: No required dependencies on specific services - use local or cloud providers
- **CLI Tool**: Command-line interface for running servers and managing the knowledge graph
- **HTTP Server**: REST API server for web applications and services
- **MCP Protocol**: Model Context Protocol support for integration with Claude Desktop and other MCP clients
- **Cross-Encoder Reranking**: Advanced reranking with multiple backends (Jina API, embedding similarity, LLM-based)


## Installation

```bash
go get github.com/soundprediction/go-graphiti
```

## Quick Start

### Prerequisites

- Go 1.24+
- **Optional**: Graph database (Kuzu embedded by default, or external Memgraph/Neo4j)
- **Optional**: LLM API access (OpenAI, Ollama, vLLM, or any OpenAI-compatible service)

### Environment Variables

**Basic Setup (Local/Embedded):**
```bash
# No environment variables required for basic usage with Kuzu embedded database
# and without LLM features
```

**With OpenAI-compatible LLM (optional):**
```bash
export OPENAI_API_KEY="your-api-key"           # For OpenAI
export LLM_BASE_URL="http://localhost:11434"   # For local LLMs like Ollama
```

**With External Graph Database (optional):**
```bash
# For Memgraph
export MEMGRAPH_URI="bolt://localhost:7687"
export MEMGRAPH_USER="memgraph"
export MEMGRAPH_PASSWORD="your-password"

# For Neo4j
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USER="neo4j"
export NEO4J_PASSWORD="your-neo4j-password"

# Or for embedded Kuzu (default)
export KUZU_DB_PATH="./kuzu_db"  # Optional: defaults to "./kuzu_db"
```

### Basic Usage

**Basic Example (Kuzu + No LLM):**

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/soundprediction/go-graphiti"
    "github.com/soundprediction/go-graphiti/pkg/driver"
)

func main() {
    ctx := context.Background()

    // Create Kuzu driver (embedded database)
    kuzuDriver, err := driver.NewKuzuDriver("./kuzu_db")
    if err != nil {
        log.Fatal(err)
    }
    defer kuzuDriver.Close(ctx)

    // Create Graphiti client (LLM and embedder are optional)
    config := &graphiti.Config{
        GroupID:  "my-group",
        TimeZone: time.UTC,
    }
    client := graphiti.NewClient(kuzuDriver, nil, nil, config)
    defer client.Close(ctx)

    // Add episodes
    episodes := []graphiti.Episode{
        {
            ID:        "meeting-1",
            Name:      "Team Meeting",
            Content:   "Discussed project timeline and resource allocation",
            Reference: time.Now(),
            CreatedAt: time.Now(),
            GroupID:   "my-group",
        },
    }

    err = client.Add(ctx, episodes)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Episode added to knowledge graph")
}
```

**With OpenAI-Compatible LLM:**

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

    // Create Kuzu driver (embedded database)
    kuzuDriver, err := driver.NewKuzuDriver("./kuzu_db")
    if err != nil {
        log.Fatal(err)
    }
    defer kuzuDriver.Close(ctx)

    // Create LLM client (works with any OpenAI-compatible API)
    llmConfig := llm.Config{
        Model:       "gpt-4o-mini",  // Or "llama3", "mistral", etc.
        Temperature: &[]float32{0.7}[0],
        BaseURL:     "http://localhost:11434",  // Optional: for local LLMs
    }
    llmClient, err := llm.NewOpenAIClient("your-api-key", llmConfig)
    if err != nil {
        log.Fatal(err)
    }

    // Create embedder (optional, but recommended for semantic search)
    embedderConfig := embedder.Config{
        Model:     "text-embedding-3-small",  // Or local embedding model
        BaseURL:   "http://localhost:11434",  // Optional: for local embeddings
    }
    embedderClient := embedder.NewOpenAIEmbedder("your-api-key", embedderConfig)

    // Create Graphiti client
    config := &graphiti.Config{
        GroupID:  "my-group",
        TimeZone: time.UTC,
    }
    client := graphiti.NewClient(kuzuDriver, llmClient, embedderClient, config)
    defer client.Close(ctx)

    // Add episodes
    episodes := []graphiti.Episode{
        {
            ID:        "meeting-1",
            Name:      "Team Meeting",
            Content:   "Discussed project timeline and resource allocation",
            Reference: time.Now(),
            CreatedAt: time.Now(),
            GroupID:   "my-group",
        },
    }

    err = client.Add(ctx, episodes)
    if err != nil {
        log.Fatal(err)
    }

    // Search the knowledge graph (requires embedder for semantic search)
    results, err := client.Search(ctx, "project timeline", nil)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d nodes", len(results.Nodes))
}
```

## CLI Tool

Go-Graphiti includes a command-line interface for managing the knowledge graph and running servers.

### Installation

```bash
# Build from source
make build-cli

# Or build directly
go build -o bin/graphiti ./cmd/main.go
```

### Server Command

Start the HTTP server:

```bash
./bin/graphiti server
```

With custom configuration:

```bash
./bin/graphiti server --port 9090 --llm-api-key your-key-here
```

### Configuration

Create a configuration file:

```bash
cp .graphiti.example.yaml .graphiti.yaml
# Edit the configuration as needed
```

The server provides REST API endpoints:

- `GET /health` - Health check
- `POST /api/v1/ingest/messages` - Add messages to knowledge graph
- `POST /api/v1/search` - Search the knowledge graph
- `GET /api/v1/episodes/:group_id` - Get episodes for a group
- `POST /api/v1/get-memory` - Get memory based on messages

### API Examples

Add messages:
```bash
curl -X POST http://localhost:8080/api/v1/ingest/messages \
  -H "Content-Type: application/json" \
  -d '{
    "group_id": "user123",
    "messages": [{"role": "user", "content": "Hello, I work at Acme Corp"}]
  }'
```

Search:
```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Acme Corp",
    "group_ids": ["user123"],
    "max_facts": 10
  }'
```

See [cmd/README.md](cmd/README.md) for detailed CLI documentation.

## Architecture

The library is structured into several key packages:

- **`graphiti.go`**: Main client interface and configuration
- **`pkg/driver/`**: Graph database drivers (Kuzu, Memgraph, Neo4j)
- **`pkg/llm/`**: Language model clients (OpenAI-compatible APIs)
- **`pkg/embedder/`**: Embedding model clients (OpenAI, Gemini, Voyage)
- **`pkg/search/`**: Hybrid search functionality
- **`pkg/types/`**: Core types for nodes, edges, and data structures
- **`pkg/models/`**: Database models and query operations for nodes and edges
- **`pkg/prompts/`**: LLM prompts for extraction and processing
- **`pkg/crossencoder/`**: Cross-encoder reranking for improved relevance
- **`pkg/community/`**: Community detection and management
- **`pkg/utils/`**: Utility functions for maintenance and operations

## Node Types

- **EntityNode**: Represents entities extracted from content
- **EpisodicNode**: Represents episodic memories or events  
- **CommunityNode**: Represents communities of related entities

## Edge Types

- **EntityEdge**: Relationships between entities
- **EpisodicEdge**: Episodic relationships
- **CommunityEdge**: Community relationships

## Current Status

🚧 **Work in Progress**: Key features implementation status:

- [x] Entity and relationship extraction
- [x] Node and edge deduplication  
- [x] Embedding generation and storage
- [x] Hybrid search implementation
- [ ] Community detection
- [ ] Temporal operations
- [ ] Bulk operations
- [x] Error Tracking & Telemetry
- [x] Cost Calculation Service
- [x] Advanced Router & Provider Fallback
- [x] Caching Layer (BadgerDB)
- [x] Circuit Breaker & Email Alerts

## Documentation

📚 **Documentation**:
- **[Getting Started](docs/GETTING_STARTED.md)**: Setup guide and first steps
- **[Examples](docs/EXAMPLES.md)**: Practical usage examples
- **[Kuzu Setup Guide](docs/KUZU_SETUP.md)**: Using the embedded Kuzu graph database
- **[FAQ](docs/FAQ.md)**: Common questions and troubleshooting
- **[Python to Go Mapping](docs/PYTHON_TO_GO_MAPPING.md)**: Port status tracking

## Examples

See the `examples/` directory for complete usage examples:

- `examples/basic/`: Minimal setup with Kuzu embedded database
- `examples/kuzu_ollama/`: Local setup with Kuzu + Ollama (maximum privacy)
- `examples/openai_compatible/`: Using various OpenAI-compatible services
- `examples/chat/`: Chat interface example
- `examples/prompts/`: Prompt engineering examples
- More examples in [docs/EXAMPLES.md](docs/EXAMPLES.md)

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build ./...
```

### Running Examples

```bash
# Basic example (no external dependencies)
cd examples/basic
go run main.go

# Or with local LLM
cd examples/kuzu_ollama
go run main.go

# Chat interface example
cd examples/chat
go run main.go
```

## Contributing

This project follows the same patterns as [go-light-rag](https://github.com/soundprediction/go-light-rag) for consistency. Contributions are welcome!

## License

Apache 2.0 License - see the original [Graphiti license](https://github.com/getzep/graphiti/blob/main/LICENSE)

## Acknowledgments

- Original [Graphiti](https://github.com/getzep/graphiti) Python library by Zep
- [go-light-rag](https://github.com/soundprediction/go-light-rag) for Go patterns and inspiration
