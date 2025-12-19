# Ladybug Driver Setup Guide

This guide explains how to use the Ladybug graph database driver with go-predicato.

## What is Ladybug?

[Ladybug](https://ladybugdb.com/) is an embedded graph database management system built for speed and scalability. Unlike Neo4j which runs as a separate server, Ladybug is embedded directly into your application, similar to SQLite for relational databases.

## Why Ladybug is the Default

Ladybug is the **default and recommended** database driver for go-predicato because:

- **Zero Setup**: No external database server required
- **Embedded**: Database files stored locally alongside your application
- **Fast**: Optimized for graph queries and traversal
- **No Dependencies**: Works out-of-the-box with no configuration
- **Portable**: Database files can be easily backed up and moved

## Prerequisites

- Go 1.24+
- **That's it!** No external database installation required

## Installation

No installation required! Ladybug is embedded and works immediately when you import go-predicato.

```bash
go get github.com/soundprediction/go-predicato
```

## Usage

### Basic Setup

The simplest way to create a Ladybug-based knowledge graph:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/soundprediction/go-predicato"
    "github.com/soundprediction/go-predicato/pkg/driver"
    "github.com/soundprediction/go-predicato/pkg/types"
)

func main() {
    ctx := context.Background()

    // Create Ladybug driver - creates database files in the specified directory
    ladybugDriver, err := driver.NewLadybugDriver("./ladybug_db")
    if err != nil {
        log.Fatal("Failed to create Ladybug driver:", err)
    }
    defer ladybugDriver.Close(ctx)

    // Create Predicato client
    config := &predicato.Config{
        GroupID:  "my-app",
        TimeZone: time.UTC,
    }
    client := predicato.NewClient(ladybugDriver, nil, nil, config)
    defer client.Close(ctx)

    log.Println("Ladybug-based knowledge graph ready!")
}

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/soundprediction/go-predicato"
    "github.com/soundprediction/go-predicato/pkg/driver"
    "github.com/soundprediction/go-predicato/pkg/embedder"
    "github.com/soundprediction/go-predicato/pkg/llm"
)

func main() {
    ctx := context.Background()

    // Create Ladybug driver (embedded database)
    ladybugDriver, err := driver.NewLadybugDriver("./my_graph_db")
    if err != nil {
        log.Fatal(err)
    }
    defer ladybugDriver.Close(ctx)

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

    // Create Predicato client with Ladybug
    config := &predicato.Config{
        GroupID:  "my-group",
        TimeZone: time.UTC,
    }
    client := predicato.NewClient(ladybugDriver, llmClient, embedderClient, config)
    defer client.Close(ctx)

    // Use normally - Ladybug handles all graph operations locally
    episodes := []predicato.Episode{
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

### Configuration Options

```go
// Create with custom database path
driver, err := driver.NewLadybugDriver("/path/to/my/graph.db")

// Create with default path (./ladybug_predicato_db)
driver, err := driver.NewLadybugDriver("")
```

## Advantages of Ladybug

### ✅ Benefits

- **Embedded**: No separate server to manage
- **Fast**: Optimized for high-performance graph queries
- **Lightweight**: Minimal resource overhead
- **ACID**: Full transaction support
- **Cypher**: Supports Cypher query language
- **Schema-flexible**: Property graph model

### 📋 Use Cases

**Ideal for:**
- Desktop applications
- Edge computing
- Development and testing
- Single-node deployments
- Applications requiring fast local graph access

**Consider alternatives for:**
- Multi-user concurrent access
- Distributed graph processing
- Web applications with high concurrency
- Applications requiring remote graph access

## Performance Characteristics

### Local vs Server Databases

| Feature | Ladybug (Embedded) | Neo4j (Server) |
|---------|----------------|----------------|
| Setup complexity | Low | High |
| Performance | Very fast (local) | Fast (network overhead) |
| Concurrency | Single process | Multi-user |
| Resource usage | Low | Higher |
| Backup/replication | File-based | Built-in tools |
| Scaling | Vertical only | Horizontal + vertical |

### Performance Tips

1. **Use transactions**: Group related operations in transactions for better performance
2. **Index key properties**: Create indexes for frequently queried node properties
3. **Optimize embeddings**: Use appropriate embedding dimensions for your use case
4. **Batch operations**: Use bulk operations for inserting many nodes/edges

## Development Workflow

### Driver Usage

```go
// Create driver instance
driver, err := driver.NewLadybugDriver("./test.db", 1)
if err != nil {
    log.Fatal(err)
}

// All operations will work with actual Ladybug database
node, err := driver.GetNode(ctx, "node-id", "group-id")
if err != nil {
    // Handle error
}
```

## Testing

The Ladybug driver includes comprehensive tests that verify:

1. **Interface compliance**: Ensures LadybugDriver implements GraphDriver interface
2. **Stub behavior**: Verifies all methods return appropriate "not implemented" errors
3. **Configuration**: Tests driver creation with various parameters
4. **Future usage patterns**: Includes skipped tests showing expected usage

Run tests with:

```bash
go test ./pkg/driver -v
```

## Migration from Neo4j

If you're currently using Neo4j and want to switch to Ladybug:

### Data Migration

```go
// Example migration script
func migrateFromNeo4j(neo4jDriver *driver.Neo4jDriver, ladybugDriver *driver.LadybugDriver) error {
    ctx := context.Background()
    
    // 1. Export all nodes from Neo4j
    // 2. Import nodes to Ladybug
    // 3. Export all edges from Neo4j  
    // 4. Import edges to Ladybug
    
    // This is conceptual - actual implementation depends on your data structure
    return nil
}
```

### Configuration Changes

```go
// Before (Neo4j)
neo4jDriver, err := driver.NewNeo4jDriver(
    "bolt://localhost:7687",
    "neo4j",
    "password", 
    "neo4j",
)

// After (Ladybug)
ladybugDriver, err := driver.NewLadybugDriver("./graph.db")
```

## Troubleshooting

### Common Issues

1. **Build errors with CGO**
   ```
   Error: CGO_ENABLED required
   ```
   - Solution: Ensure CGO is enabled and C/C++ build tools are installed

2. **Library not found**
   ```
   Error: github.com/ladybugdb/go-ladybug not found
   ```
   - Solution: Wait for stable release or build from source

3. **File permissions**
   ```
   Error: failed to create database directory
   ```
   - Solution: Ensure write permissions for database directory

### Platform-Specific Notes

**macOS:**
```bash
# May need Xcode command line tools
xcode-select --install
```

**Linux:**
```bash
# May need build-essential
sudo apt-get install build-essential
```

**Windows:**
```bash
# May need MSYS2 with UCRT64 environment
# See Ladybug documentation for Windows setup
```

## Contributing

To contribute to the Ladybug driver implementation:

1. **Monitor the go-ladybug repository**: https://github.com/ladybugdb/go-ladybug
2. **Implement missing functionality**: Replace stub implementations with actual Ladybug API calls
3. **Add comprehensive tests**: Test all driver operations
4. **Update documentation**: Keep this guide current with implementation status

## Resources

- **Ladybug Documentation**: https://docs.ladybugdb.com/
- **Ladybug GitHub**: https://github.com/ladybugdb/ladybug
- **Go Binding**: https://github.com/ladybugdb/go-ladybug
- **Community**: https://github.com/ladybugdb/ladybug/discussions

## Roadmap

- [ ] Monitor go-ladybug library stability
- [ ] Replace stub implementations with actual Ladybug API calls
- [ ] Add comprehensive integration tests
- [ ] Performance benchmarks vs Neo4j
- [ ] Migration tools from Neo4j to Ladybug
- [ ] Advanced features (streaming, backup/restore)
