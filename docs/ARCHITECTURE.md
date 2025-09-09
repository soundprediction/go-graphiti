# Architecture

This document describes the architecture and design principles of go-graphiti.

## Overview

go-graphiti follows a modular, interface-driven architecture that allows for easy extension and testing. The library is designed around the principle of temporal knowledge graphs with support for real-time updates and hybrid search capabilities.

## Core Components

### Main Client (`graphiti.go`)

The main `Graphiti` interface provides the primary API for users:

```go
type Graphiti interface {
    Add(ctx context.Context, episodes []types.Episode) error
    Search(ctx context.Context, query string, config *types.SearchConfig) (*types.SearchResults, error)
    GetNode(ctx context.Context, nodeID string) (*types.Node, error)
    GetEdge(ctx context.Context, edgeID string) (*types.Edge, error)
    Close(ctx context.Context) error
}
```

The `Client` struct orchestrates all components and maintains configuration.

### Type System (`pkg/types/`)

Central type definitions that avoid circular dependencies:

- **Node**: Represents entities, episodes, or communities in the graph
- **Edge**: Represents relationships between nodes
- **Episode**: Input data units for processing
- **SearchConfig/SearchResults**: Search operation types

### Graph Database Layer (`pkg/driver/`)

Abstract interface for graph database operations:

```go
type GraphDriver interface {
    // Node operations
    GetNode(ctx context.Context, nodeID, groupID string) (*types.Node, error)
    UpsertNode(ctx context.Context, node *types.Node) error
    // ... more operations
}
```

**Implementations:**
- `Neo4jDriver`: Production-ready Neo4j integration
- Extensible for other graph databases (ArangoDB, Neptune, etc.)

### Language Model Layer (`pkg/llm/`)

Interface for language model interactions:

```go
type Client interface {
    Chat(ctx context.Context, messages []Message) (*Response, error)
    ChatWithStructuredOutput(ctx context.Context, messages []Message, schema any) (json.RawMessage, error)
    Close() error
}
```

**Implementations:**
- `OpenAIClient`: OpenAI GPT models
- Extensible for Anthropic, Google, local models, etc.

### Embedding Layer (`pkg/embedder/`)

Interface for text embedding generation:

```go
type Client interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedSingle(ctx context.Context, text string) ([]float32, error)
    Dimensions() int
    Close() error
}
```

**Implementations:**
- `OpenAIEmbedder`: OpenAI embedding models
- Extensible for other providers

## Design Principles

### 1. Interface-Driven Design

All major components are defined as interfaces, enabling:
- Easy mocking for testing
- Pluggable implementations
- Future extensibility

### 2. Context-Aware Operations

All operations accept `context.Context` for:
- Timeout handling
- Cancellation support
- Request tracing

### 3. Temporal Awareness

All data includes temporal information:
- Creation and update timestamps
- Validity periods (ValidFrom/ValidTo)
- Reference times for episodes

### 4. Multi-tenancy Support

GroupID concept enables:
- Data isolation between users/contexts
- Secure multi-tenant deployments
- Organizational data separation

### 5. Error Handling

Typed errors for predictable error handling:
- `ErrNodeNotFound`
- `ErrEdgeNotFound`
- `ErrInvalidEpisode`

## Data Flow

### Episode Processing Pipeline

1. **Input**: Episodes with raw content
2. **Entity Extraction**: LLM extracts entities and relationships
3. **Deduplication**: Merge similar nodes and edges
4. **Embedding Generation**: Create vector representations
5. **Storage**: Persist to graph database

### Search Pipeline

1. **Query Processing**: Analyze search query
2. **Embedding Generation**: Create query vector
3. **Semantic Search**: Vector similarity search
4. **Keyword Search**: Traditional text search
5. **Graph Traversal**: Explore connected nodes
6. **Result Fusion**: Combine and rank results

## Configuration

### Client Configuration

```go
type Config struct {
    GroupID      string                // Multi-tenancy identifier
    TimeZone     *time.Location        // Temporal operations timezone
    SearchConfig *types.SearchConfig   // Default search settings
}
```

### Search Configuration

```go
type SearchConfig struct {
    Limit              int            // Maximum results
    CenterNodeDistance int            // Graph traversal depth
    MinScore           float64        // Minimum relevance score
    IncludeEdges       bool           // Include edges in results
    Rerank             bool           // Apply reranking
    Filters            *SearchFilters // Additional constraints
}
```

## Testing Strategy

### Mock Implementations

Each interface has corresponding mock implementations:
- `MockGraphDriver`
- `MockLLMClient` 
- `MockEmbedderClient`

### Test Coverage

- Unit tests for all components
- Integration tests for complete workflows
- Mock-based testing for external dependencies

## Extension Points

### Adding New Database Drivers

1. Implement the `GraphDriver` interface
2. Handle temporal operations
3. Support multi-tenancy via GroupID

### Adding New LLM Providers

1. Implement the `llm.Client` interface
2. Handle structured output if supported
3. Manage API-specific configurations

### Adding New Embedding Providers

1. Implement the `embedder.Client` interface
2. Handle batch processing efficiently
3. Provide dimension information

## Performance Considerations

### Batch Operations

- Embedding generation supports batching
- Graph operations can be batched for efficiency
- Connection pooling for database operations

### Memory Management

- Streaming for large result sets
- Resource cleanup via `Close()` methods
- Context-based cancellation

### Concurrency

- Thread-safe operations
- Concurrent processing where appropriate
- Resource sharing through interfaces

## Security Considerations

### API Key Management

- No API keys stored in code
- Environment variable configuration
- Secure credential handling

### Data Isolation

- GroupID-based multi-tenancy
- Query-level access control
- No cross-tenant data leakage

### Input Validation

- Episode content validation
- Query parameter sanitization
- Type safety throughout

## Future Extensions

### Planned Features

- Additional graph database backends
- More LLM and embedding providers
- Advanced search algorithms
- Community detection algorithms
- Temporal query capabilities

### Plugin Architecture

The interface-driven design enables plugin-style extensions for:
- Custom entity extractors
- Specialized search algorithms
- Alternative storage backends
- Custom preprocessing pipelines