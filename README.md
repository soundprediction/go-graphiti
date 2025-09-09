# go-graphiti

Go port of the [Graphiti](https://github.com/getzep/graphiti) temporal knowledge graph library. Graphiti is designed for building temporally-aware knowledge graphs for AI agents, enabling real-time incremental updates without batch recomputation.

## Features

- **Temporal Knowledge Graphs**: Bi-temporal data model with explicit tracking of event occurrence times
- **Hybrid Search**: Combines semantic embeddings, keyword search (BM25), and graph traversal
- **Multiple Backends**: Support for Neo4j and other graph databases
- **LLM Integration**: Built-in support for OpenAI and other language models
- **Go Idioms**: Follows Go conventions and coding patterns similar to [go-light-rag](https://github.com/MegaGrindStone/go-light-rag)

## Installation

```bash
go get github.com/getzep/go-graphiti
```

## Quick Start

### Prerequisites

- Go 1.24+
- Neo4j database (local or cloud)
- OpenAI API key

### Environment Variables

```bash
export OPENAI_API_KEY="your-openai-api-key"
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USER="neo4j"
export NEO4J_PASSWORD="your-neo4j-password"
```

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/getzep/go-graphiti"
    "github.com/getzep/go-graphiti/pkg/driver"
    "github.com/getzep/go-graphiti/pkg/embedder"
    "github.com/getzep/go-graphiti/pkg/llm"
)

func main() {
    ctx := context.Background()

    // Create Neo4j driver
    neo4jDriver, err := driver.NewNeo4jDriver(
        "bolt://localhost:7687", 
        "neo4j", 
        "password", 
        "neo4j",
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
    llmClient := llm.NewOpenAIClient("your-api-key", llmConfig)

    // Create embedder
    embedderConfig := embedder.Config{
        Model:     "text-embedding-3-small",
        BatchSize: 100,
    }
    embedderClient := embedder.NewOpenAIEmbedder("your-api-key", embedderConfig)

    // Create Graphiti client
    config := &graphiti.Config{
        GroupID:  "my-group",
        TimeZone: time.UTC,
    }
    client := graphiti.NewClient(neo4jDriver, llmClient, embedderClient, config)
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

    // Search the knowledge graph
    results, err := client.Search(ctx, "project timeline", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Found %d nodes", len(results.Nodes))
}
```

## Architecture

The library is structured into several key packages:

- **`graphiti.go`**: Main client interface and configuration
- **`pkg/driver/`**: Graph database drivers (Neo4j, etc.)
- **`pkg/llm/`**: Language model clients (OpenAI, etc.)
- **`pkg/embedder/`**: Embedding model clients (OpenAI, etc.)
- **`pkg/search/`**: Hybrid search functionality
- **`pkg/nodes/`**: Node types and operations
- **`pkg/edges/`**: Edge types and operations
- **`pkg/prompts/`**: LLM prompts for extraction and processing

## Node Types

- **EntityNode**: Represents entities extracted from content
- **EpisodicNode**: Represents episodic memories or events  
- **CommunityNode**: Represents communities of related entities

## Edge Types

- **EntityEdge**: Relationships between entities
- **EpisodicEdge**: Episodic relationships
- **CommunityEdge**: Community relationships

## Current Status

🚧 **Work in Progress**: This is an initial port with basic structure in place. Key features still being implemented:

- [ ] Entity and relationship extraction
- [ ] Node and edge deduplication  
- [ ] Embedding generation and storage
- [ ] Hybrid search implementation
- [ ] Community detection
- [ ] Temporal operations
- [ ] Bulk operations
- [ ] Additional graph drivers

## Documentation

📚 **Complete Documentation**:
- **[Getting Started](docs/GETTING_STARTED.md)**: Setup guide and first steps
- **[API Reference](docs/API_REFERENCE.md)**: Complete API documentation
- **[Architecture](docs/ARCHITECTURE.md)**: Design principles and components
- **[Examples](docs/EXAMPLES.md)**: Practical usage examples
- **[FAQ](docs/FAQ.md)**: Common questions and troubleshooting

## Examples

See the `examples/` directory for complete usage examples:

- `examples/basic/`: Basic usage with Neo4j
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
cd examples/basic
go run main.go
```

## Contributing

This project follows the same patterns as [go-light-rag](https://github.com/MegaGrindStone/go-light-rag) for consistency. Contributions are welcome!

## License

Apache 2.0 License - see the original [Graphiti license](https://github.com/getzep/graphiti/blob/main/LICENSE)

## Acknowledgments

- Original [Graphiti](https://github.com/getzep/graphiti) Python library by Zep
- [go-light-rag](https://github.com/MegaGrindStone/go-light-rag) for Go patterns and inspiration