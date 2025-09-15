# Kuzu Driver Setup Guide

This guide explains how to use the Kuzu graph database driver with go-graphiti.

## What is Kuzu?

[Kuzu](https://kuzudb.com/) is an embedded graph database management system built for speed and scalability. Unlike Neo4j which runs as a separate server, Kuzu is embedded directly into your application, similar to SQLite for relational databases.

## Why Kuzu is the Default

Kuzu is the **default and recommended** database driver for go-graphiti because:

- **Zero Setup**: No external database server required
- **Embedded**: Database files stored locally alongside your application
- **Fast**: Optimized for graph queries and traversal
- **No Dependencies**: Works out-of-the-box with no configuration
- **Portable**: Database files can be easily backed up and moved

## Prerequisites

- Go 1.24+
- **That's it!** No external database installation required

## Installation

No installation required! Kuzu is embedded and works immediately when you import go-graphiti.

```bash
go get github.com/soundprediction/go-graphiti
```

## Usage

### Basic Setup

The simplest way to create a Kuzu-based knowledge graph:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/soundprediction/go-graphiti"
    "github.com/soundprediction/go-graphiti/pkg/driver"
    "github.com/soundprediction/go-graphiti/pkg/types"
)

func main() {
    ctx := context.Background()

    // Create Kuzu driver - creates database files in the specified directory
    kuzuDriver, err := driver.NewKuzuDriver("./kuzu_db")
    if err != nil {
        log.Fatal("Failed to create Kuzu driver:", err)
    }
    defer kuzuDriver.Close(ctx)

    // Create Graphiti client
    config := &graphiti.Config{
        GroupID:  "my-app",
        TimeZone: time.UTC,
    }
    client := graphiti.NewClient(kuzuDriver, nil, nil, config)
    defer client.Close(ctx)

    log.Println("Kuzu-based knowledge graph ready!")
}

### Basic Usage

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
    kuzuDriver, err := driver.NewKuzuDriver("./my_graph_db")
    if err != nil {
        log.Fatal(err)
    }
    defer kuzuDriver.Close(ctx)

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

    // Create Graphiti client with Kuzu
    config := &graphiti.Config{
        GroupID:  "my-group",
        TimeZone: time.UTC,
    }
    client := graphiti.NewClient(kuzuDriver, llmClient, embedderClient, config)
    defer client.Close(ctx)

    // Use normally - Kuzu handles all graph operations locally
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

### Configuration Options

```go
// Create with custom database path
driver, err := driver.NewKuzuDriver("/path/to/my/graph.db")

// Create with default path (./kuzu_graphiti_db)
driver, err := driver.NewKuzuDriver("")
```

## Advantages of Kuzu

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

| Feature | Kuzu (Embedded) | Neo4j (Server) |
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

### Current State (Stub Implementation)

```go
// This will return an error
driver, err := driver.NewKuzuDriver("./test.db")
if err != nil {
    log.Fatal(err)
}

// All operations return "not implemented" errors
node, err := driver.GetNode(ctx, "node-id", "group-id")
// err: "KuzuDriver not implemented - requires github.com/kuzudb/go-kuzu dependency"
```

### Future State (Full Implementation)

```go
// This will work once the library is available
driver, err := driver.NewKuzuDriver("./test.db")
if err != nil {
    log.Fatal(err)
}

// All operations will work with actual Kuzu database
node, err := driver.GetNode(ctx, "node-id", "group-id")
if err != nil {
    log.Fatal(err)
}
```

## Testing

The Kuzu driver includes comprehensive tests that verify:

1. **Interface compliance**: Ensures KuzuDriver implements GraphDriver interface
2. **Stub behavior**: Verifies all methods return appropriate "not implemented" errors
3. **Configuration**: Tests driver creation with various parameters
4. **Future usage patterns**: Includes skipped tests showing expected usage

Run tests with:

```bash
go test ./pkg/driver -v
```

## Migration from Neo4j

If you're currently using Neo4j and want to switch to Kuzu:

### Data Migration

```go
// Example migration script
func migrateFromNeo4j(neo4jDriver *driver.Neo4jDriver, kuzuDriver *driver.KuzuDriver) error {
    ctx := context.Background()
    
    // 1. Export all nodes from Neo4j
    // 2. Import nodes to Kuzu
    // 3. Export all edges from Neo4j  
    // 4. Import edges to Kuzu
    
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

// After (Kuzu)
kuzuDriver, err := driver.NewKuzuDriver("./graph.db")
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
   Error: github.com/kuzudb/go-kuzu not found
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
# See Kuzu documentation for Windows setup
```

## Contributing

To contribute to the Kuzu driver implementation:

1. **Monitor the go-kuzu repository**: https://github.com/kuzudb/go-kuzu
2. **Implement missing functionality**: Replace stub implementations with actual Kuzu API calls
3. **Add comprehensive tests**: Test all driver operations
4. **Update documentation**: Keep this guide current with implementation status

## Resources

- **Kuzu Documentation**: https://docs.kuzudb.com/
- **Kuzu GitHub**: https://github.com/kuzudb/kuzu
- **Go Binding**: https://github.com/kuzudb/go-kuzu
- **Community**: https://github.com/kuzudb/kuzu/discussions

## Roadmap

- [ ] Monitor go-kuzu library stability
- [ ] Replace stub implementations with actual Kuzu API calls
- [ ] Add comprehensive integration tests
- [ ] Performance benchmarks vs Neo4j
- [ ] Migration tools from Neo4j to Kuzu
- [ ] Advanced features (streaming, backup/restore)
