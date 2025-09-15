# API Reference

Complete API reference for go-graphiti.

## Core Interface

### Graphiti

The main interface for interacting with temporal knowledge graphs.

```go
type Graphiti interface {
    Add(ctx context.Context, episodes []types.Episode) error
    Search(ctx context.Context, query string, config *types.SearchConfig) (*types.SearchResults, error)
    GetNode(ctx context.Context, nodeID string) (*types.Node, error)
    GetEdge(ctx context.Context, edgeID string) (*types.Edge, error)
    Close(ctx context.Context) error
}
```

#### Methods

##### Add

```go
Add(ctx context.Context, episodes []types.Episode) error
```

Processes and adds new episodes to the knowledge graph.

**Parameters:**
- `ctx`: Context for timeout and cancellation
- `episodes`: Slice of episodes to process

**Returns:**
- `error`: Processing error, if any

**Example:**
```go
episodes := []types.Episode{
    {
        ID:        "meeting-1",
        Content:   "Team discussed project timeline",
        Reference: time.Now(),
    },
}
err := client.Add(ctx, episodes)
```

##### Search

```go
Search(ctx context.Context, query string, config *types.SearchConfig) (*types.SearchResults, error)
```

Performs hybrid search across the knowledge graph.

**Parameters:**
- `ctx`: Context for timeout and cancellation
- `query`: Search query string
- `config`: Search configuration (optional, uses default if nil)

**Returns:**
- `*types.SearchResults`: Search results
- `error`: Search error, if any

**Example:**
```go
config := &types.SearchConfig{
    Limit: 10,
    IncludeEdges: true,
}
results, err := client.Search(ctx, "project timeline", config)
```

##### GetNode

```go
GetNode(ctx context.Context, nodeID string) (*types.Node, error)
```

Retrieves a specific node from the knowledge graph.

**Parameters:**
- `ctx`: Context for timeout and cancellation
- `nodeID`: Unique identifier of the node

**Returns:**
- `*types.Node`: The requested node
- `error`: `ErrNodeNotFound` if node doesn't exist, other errors for failures

##### GetEdge

```go
GetEdge(ctx context.Context, edgeID string) (*types.Edge, error)
```

Retrieves a specific edge from the knowledge graph.

**Parameters:**
- `ctx`: Context for timeout and cancellation
- `edgeID`: Unique identifier of the edge

**Returns:**
- `*types.Edge`: The requested edge
- `error`: `ErrEdgeNotFound` if edge doesn't exist, other errors for failures

##### Close

```go
Close(ctx context.Context) error
```

Closes all connections and cleans up resources.

**Parameters:**
- `ctx`: Context for timeout and cancellation

**Returns:**
- `error`: Cleanup error, if any

## Client

### NewClient

```go
func NewClient(driver driver.GraphDriver, llmClient llm.Client, embedderClient embedder.Client, config *Config) *Client
```

Creates a new Graphiti client.

**Parameters:**
- `driver`: Graph database driver
- `llmClient`: Language model client
- `embedderClient`: Embedding client
- `config`: Client configuration (uses defaults if nil)

**Returns:**
- `*Client`: Configured Graphiti client

## Configuration

### Config

```go
type Config struct {
    GroupID      string                // Multi-tenancy identifier
    TimeZone     *time.Location        // Timezone for temporal operations
    SearchConfig *types.SearchConfig   // Default search configuration
}
```

### NewDefaultSearchConfig

```go
func NewDefaultSearchConfig() *types.SearchConfig
```

Creates a default search configuration.

**Returns:**
- `*types.SearchConfig`: Default search settings

## Types

### Node

```go
type Node struct {
    ID           string                 `json:"id"`
    Name         string                 `json:"name"`
    Type         NodeType               `json:"type"`
    GroupID      string                 `json:"group_id"`
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
    
    // Entity-specific fields
    EntityType   string                 `json:"entity_type,omitempty"`
    Summary      string                 `json:"summary,omitempty"`
    
    // Episode-specific fields
    EpisodeType  EpisodeType            `json:"episode_type,omitempty"`
    Content      string                 `json:"content,omitempty"`
    Reference    time.Time              `json:"reference,omitempty"`
    
    // Community-specific fields
    Level        int                    `json:"level,omitempty"`
    
    // Common fields
    Embedding    []float32              `json:"embedding,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
    
    // Temporal fields
    ValidFrom    time.Time              `json:"valid_from"`
    ValidTo      *time.Time             `json:"valid_to,omitempty"`
    
    // Source tracking
    SourceIDs    []string               `json:"source_ids,omitempty"`
}
```

### Edge

```go
type Edge struct {
    ID           string                 `json:"id"`
    Type         EdgeType               `json:"type"`
    SourceID     string                 `json:"source_id"`
    TargetID     string                 `json:"target_id"`
    GroupID      string                 `json:"group_id"`
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
    
    // Relationship details
    Name         string                 `json:"name,omitempty"`
    Summary      string                 `json:"summary,omitempty"`
    Strength     float64                `json:"strength,omitempty"`
    
    // Embedding for semantic search
    Embedding    []float32              `json:"embedding,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
    
    // Temporal fields
    ValidFrom    time.Time              `json:"valid_from"`
    ValidTo      *time.Time             `json:"valid_to,omitempty"`
    
    // Source tracking
    SourceIDs    []string               `json:"source_ids,omitempty"`
}
```

### Episode

```go
type Episode struct {
    ID        string                 // Unique identifier
    Name      string                 // Display name
    Content   string                 // Text content to process
    Reference time.Time              // When the episode occurred
    CreatedAt time.Time              // When added to system
    GroupID   string                 // Multi-tenancy identifier
    Metadata  map[string]interface{} // Additional metadata
}
```

### SearchConfig

```go
type SearchConfig struct {
    Limit              int            // Maximum number of results
    CenterNodeDistance int            // Maximum distance from center nodes
    MinScore           float64        // Minimum relevance score
    IncludeEdges       bool           // Include edges in results
    Rerank             bool           // Apply reranking
    Filters            *SearchFilters // Additional constraints
}
```

### SearchFilters

```go
type SearchFilters struct {
    GroupIDs    []string      // Group IDs to include
    NodeTypes   []NodeType    // Node types to include
    EdgeTypes   []EdgeType    // Edge types to include
    EntityTypes []string      // Entity types to include
    TimeRange   *TimeRange    // Temporal filtering
}
```

### SearchResults

```go
type SearchResults struct {
    Nodes []*Node  // Found nodes
    Edges []*Edge  // Found edges
    Query string   // Original query
    Total int      // Total results before limit
}
```

### TimeRange

```go
type TimeRange struct {
    Start time.Time // Start of time range
    End   time.Time // End of time range
}
```

## Type Constants

### NodeType

```go
type NodeType string

const (
    EntityNodeType    NodeType = "entity"    // Entities from content
    EpisodicNodeType  NodeType = "episodic"  // Episodes and events
    CommunityNodeType NodeType = "community" // Communities of entities
)
```

### EdgeType

```go
type EdgeType string

const (
    EntityEdgeType    EdgeType = "entity"    // Entity relationships
    EpisodicEdgeType  EdgeType = "episodic"  // Episode relationships
    CommunityEdgeType EdgeType = "community" // Community relationships
)
```

### EpisodeType

```go
type EpisodeType string

const (
    ConversationEpisodeType EpisodeType = "conversation" // Conversations
    DocumentEpisodeType     EpisodeType = "document"     // Documents
    EventEpisodeType        EpisodeType = "event"        // Events/actions
)
```

## Drivers

### GraphDriver Interface

```go
type GraphDriver interface {
    // Node operations
    GetNode(ctx context.Context, nodeID, groupID string) (*types.Node, error)
    UpsertNode(ctx context.Context, node *types.Node) error
    DeleteNode(ctx context.Context, nodeID, groupID string) error
    GetNodes(ctx context.Context, nodeIDs []string, groupID string) ([]*types.Node, error)

    // Edge operations  
    GetEdge(ctx context.Context, edgeID, groupID string) (*types.Edge, error)
    UpsertEdge(ctx context.Context, edge *types.Edge) error
    DeleteEdge(ctx context.Context, edgeID, groupID string) error
    GetEdges(ctx context.Context, edgeIDs []string, groupID string) ([]*types.Edge, error)

    // Graph traversal operations
    GetNeighbors(ctx context.Context, nodeID, groupID string, maxDistance int) ([]*types.Node, error)
    GetRelatedNodes(ctx context.Context, nodeID, groupID string, edgeTypes []types.EdgeType) ([]*types.Node, error)

    // Search operations
    SearchNodesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Node, error)
    SearchEdgesByEmbedding(ctx context.Context, embedding []float32, groupID string, limit int) ([]*types.Edge, error)

    // Bulk operations
    UpsertNodes(ctx context.Context, nodes []*types.Node) error
    UpsertEdges(ctx context.Context, edges []*types.Edge) error

    // Temporal operations
    GetNodesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Node, error)
    GetEdgesInTimeRange(ctx context.Context, start, end time.Time, groupID string) ([]*types.Edge, error)

    // Community operations
    GetCommunities(ctx context.Context, groupID string, level int) ([]*types.Node, error)
    BuildCommunities(ctx context.Context, groupID string) error

    // Database maintenance
    CreateIndices(ctx context.Context) error
    GetStats(ctx context.Context, groupID string) (*GraphStats, error)

    // Connection management
    Close(ctx context.Context) error
}
```

### NewKuzuDriver

```go
func NewKuzuDriver(dbPath string) (*KuzuDriver, error)
```

Creates a new Kuzu embedded database driver instance (recommended default).

**Parameters:**
- `dbPath`: Path to database directory (defaults to "./kuzu_db" if empty)

**Returns:**
- `*KuzuDriver`: Kuzu driver instance
- `error`: Creation error, if any

### NewNeo4jDriver

```go
func NewNeo4jDriver(uri, username, password, database string) (*Neo4jDriver, error)
```

Creates a new Neo4j driver instance for external database setups.

**Parameters:**
- `uri`: Neo4j connection URI (e.g., "bolt://localhost:7687")
- `username`: Database username
- `password`: Database password
- `database`: Database name (defaults to "neo4j" if empty)

**Returns:**
- `*Neo4jDriver`: Neo4j driver instance
- `error`: Connection error, if any

## LLM Clients

### Client Interface

```go
type Client interface {
    Chat(ctx context.Context, messages []Message) (*Response, error)
    ChatWithStructuredOutput(ctx context.Context, messages []Message, schema any) (json.RawMessage, error)
    Close() error
}
```

### NewOpenAIClient

```go
func NewOpenAIClient(apiKey string, config Config) *OpenAIClient
```

Creates a new OpenAI-compatible LLM client. Works with any service implementing the OpenAI API specification.

**Parameters:**
- `apiKey`: API key (use "dummy" for services that don't require keys like Ollama)
- `config`: LLM configuration (includes BaseURL for local services)

**Returns:**
- `*OpenAIClient`: OpenAI-compatible client instance

### Convenience Functions

#### NewOllamaClient

```go
func NewOllamaClient(baseURL, model string, config Config) (*OpenAIClient, error)
```

Creates a client for Ollama local LLM service.

#### NewLocalAIClient

```go
func NewLocalAIClient(baseURL, model string, config Config) (*OpenAIClient, error)
```

Creates a client for LocalAI self-hosted service.

#### NewVLLMClient

```go
func NewVLLMClient(baseURL, model string, config Config) (*OpenAIClient, error)
```

Creates a client for vLLM high-performance serving.

### LLM Config

```go
type Config struct {
    Model       string   `json:"model"`                  // Model name
    Temperature *float32 `json:"temperature,omitempty"`  // Randomness (0.0-1.0)
    MaxTokens   *int     `json:"max_tokens,omitempty"`   // Response length
    TopP        *float32 `json:"top_p,omitempty"`        // Nucleus sampling
    Stop        []string `json:"stop,omitempty"`         // Stop sequences
}
```

### Message Types

```go
type Message struct {
    Role    Role   `json:"role"`    // Message role
    Content string `json:"content"` // Message content
}

type Role string

const (
    RoleSystem    Role = "system"    // System message
    RoleUser      Role = "user"      // User message
    RoleAssistant Role = "assistant" // Assistant message
)
```

Helper functions:
```go
func NewMessage(role Role, content string) Message
func NewSystemMessage(content string) Message
func NewUserMessage(content string) Message
func NewAssistantMessage(content string) Message
```

### Response

```go
type Response struct {
    Content      string                 `json:"content"`              // Response content
    TokensUsed   *TokenUsage            `json:"tokens_used,omitempty"` // Usage statistics
    FinishReason string                 `json:"finish_reason,omitempty"` // Why generation stopped
    Metadata     map[string]interface{} `json:"metadata,omitempty"`   // Additional metadata
}

type TokenUsage struct {
    PromptTokens     int `json:"prompt_tokens"`     // Input tokens
    CompletionTokens int `json:"completion_tokens"` // Output tokens
    TotalTokens      int `json:"total_tokens"`      // Total tokens
}
```

## Embedding Clients

### Client Interface

```go
type Client interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedSingle(ctx context.Context, text string) ([]float32, error)
    Dimensions() int
    Close() error
}
```

### NewOpenAIEmbedder

```go
func NewOpenAIEmbedder(apiKey string, config Config) *OpenAIEmbedder
```

Creates a new OpenAI-compatible embedder client. Works with any service implementing the OpenAI embeddings API.

**Parameters:**
- `apiKey`: API key (use "dummy" for services that don't require keys)
- `config`: Embedder configuration (includes BaseURL for local services)

**Returns:**
- `*OpenAIEmbedder`: OpenAI-compatible embedder instance

### Embedder Config

```go
type Config struct {
    Model      string `json:"model"`       // Embedding model name
    BatchSize  int    `json:"batch_size"`  // Batch processing size
    Dimensions int    `json:"dimensions"`  // Embedding dimensions
}
```

## Error Types

### Predefined Errors

```go
var (
    ErrNodeNotFound     = errors.New("node not found")
    ErrEdgeNotFound     = errors.New("edge not found")
    ErrInvalidEpisode   = errors.New("invalid episode")
)
```

### Error Handling

```go
node, err := client.GetNode(ctx, nodeID)
if err != nil {
    if errors.Is(err, graphiti.ErrNodeNotFound) {
        // Handle missing node
    } else {
        // Handle other errors
    }
}
```

## Usage Patterns

### Basic Client Setup

```go
// Create all required clients

// Option 1: Embedded database (recommended)
driver, _ := driver.NewKuzuDriver("./my_graph_db")

// Option 2: External database
// driver, _ := driver.NewNeo4jDriver(uri, user, pass, db)

// LLM client (works with any OpenAI-compatible service)
llmClient := llm.NewOpenAIClient(apiKey, llmConfig)  // or NewOllamaClient, etc.

// Embedder client (works with any compatible service)
embedder := embedder.NewOpenAIEmbedder(apiKey, embedConfig)

// Create Graphiti client
client := graphiti.NewClient(driver, llmClient, embedder, config)
defer client.Close(ctx)
```

### Episode Processing

```go
episodes := []types.Episode{
    {
        ID:        "episode-1",
        Name:      "Team Meeting",
        Content:   "Meeting content...",
        Reference: time.Now(),
        CreatedAt: time.Now(),
        GroupID:   "team-alpha",
    },
}

if err := client.Add(ctx, episodes); err != nil {
    log.Fatal(err)
}
```

### Advanced Search

```go
filters := &types.SearchFilters{
    NodeTypes:   []types.NodeType{types.EntityNodeType},
    EntityTypes: []string{"Person", "Project"},
    TimeRange: &types.TimeRange{
        Start: time.Now().Add(-7 * 24 * time.Hour),
        End:   time.Now(),
    },
}

config := &types.SearchConfig{
    Limit:              20,
    CenterNodeDistance: 3,
    MinScore:           0.1,
    IncludeEdges:       true,
    Rerank:             true,
    Filters:            filters,
}

results, err := client.Search(ctx, "project updates", config)
if err != nil {
    log.Fatal(err)
}

for _, node := range results.Nodes {
    fmt.Printf("Found: %s (type: %s, score: %.2f)\n", 
        node.Name, node.Type, node.Metadata["score"])
}
```