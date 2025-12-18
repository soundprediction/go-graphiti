# Frequently Asked Questions

## General Questions

### What is go-graphiti?

go-graphiti is a Go port of the Python Graphiti library, designed for building temporally-aware knowledge graphs for AI agents. It enables real-time incremental updates without batch recomputation and provides hybrid search capabilities.

### How does it differ from the Python version?

The Go version maintains the same core concepts but follows Go idioms and patterns:
- Interface-driven architecture for better testability
- Context-aware operations for timeout/cancellation support
- Strong typing throughout the API
- Go-style error handling
- Resource management with Close() methods

### What are the main use cases?

- **Personal Knowledge Management**: Build searchable knowledge bases from notes, documents, and experiences
- **Customer Support**: Create intelligent support systems that learn from past interactions
- **Research Analysis**: Connect and analyze relationships between papers, authors, and concepts
- **Meeting Intelligence**: Extract insights and track action items from meeting transcripts
- **Multi-tenant Applications**: Provide isolated knowledge graphs for different users/organizations

## Technical Questions

### What databases are supported?

Currently supported:
- **ladybug**: Embedded graph database (default, recommended) - no external setup required
- **Neo4j**: External graph database for advanced production deployments
- **Planned**: FalkorDB, ArangoDB, Amazon Neptune, other graph databases

The modular driver architecture makes it easy to add new database backends. ladybug is recommended for most use cases due to its simplicity and zero-setup requirements.

### What LLM providers are supported?

go-graphiti works with **any OpenAI-compatible API**, including:
- **OpenAI**: GPT-3.5, GPT-4, and all variants
- **Local services**: Ollama, LocalAI, vLLM, Text Generation Inference
- **Cloud alternatives**: Together AI, Anyscale, Replicate, Hugging Face
- **Self-hosted**: Any service implementing OpenAI's API specification

The library provides convenience functions for popular services, but the standard OpenAI client works with any compatible API.

### What embedding providers are supported?

go-graphiti works with **any OpenAI-compatible embedding API**, including:
- **OpenAI**: text-embedding-ada-002, text-embedding-3-small, text-embedding-3-large
- **Local services**: Ollama with embedding models, LocalAI, vLLM
- **Cloud alternatives**: Together AI, Voyage AI, Cohere (via compatibility layers)
- **Self-hosted**: Any service implementing OpenAI's embeddings API

### How does temporal awareness work?

Every node and edge includes temporal information:
- `CreatedAt`: When the data was first added
- `UpdatedAt`: When it was last modified  
- `ValidFrom`: When the information becomes valid
- `ValidTo`: When it expires (optional)
- `Reference`: When the original event occurred (for episodes)

This enables:
- Time-based queries
- Historical analysis
- Data freshness tracking
- Temporal relationship reasoning

### How does multi-tenancy work?

Multi-tenancy is implemented via `GroupID`:
- Each client is configured with a specific GroupID
- All data operations are scoped to that GroupID
- Cross-tenant data access is prevented at the database level
- Search results are automatically filtered by GroupID

```go
// User-specific client
config := &graphiti.Config{
    GroupID: fmt.Sprintf("user-%s", userID),
}
```

## Setup and Configuration

### What are the minimum requirements?

**Minimal Setup (Recommended):**
- **Go**: Version 1.24 or later
- **Database**: ladybug embedded (no external setup required)
- **LLM/Embeddings**: Optional - can work without LLM features
- **Memory**: 256MB+ for basic usage
- **Storage**: Minimal - database files stored locally

**With External Services:**
- **Database**: Neo4j 5.0+ (if not using ladybug)
- **API Keys**: For external LLM/embedding services (OpenAI, etc.)
- **Memory**: 512MB+ recommended
- **Storage**: Depends on database choice and data volume

### How do I set up the graph database?

**Option 1 - ladybug Embedded (Recommended):**
```go
// No setup required! Just specify a directory path
driver, err := driver.NewLadybugDriver("./my_graph_db")
```
ladybug creates database files locally and requires no external services.

**Option 2 - Neo4j (For External Database):**

**Docker (Easiest):**
```bash
docker run \
    --name neo4j \
    -p 7474:7474 -p 7687:7687 \
    -e NEO4J_AUTH=neo4j/password \
    neo4j:latest
```

**Neo4j Desktop:**
1. Download from https://neo4j.com/download/
2. Create new project and database
3. Set password and note connection details

**Neo4j Aura (Cloud):**
1. Sign up at https://neo4j.com/cloud/aura/
2. Create database instance
3. Use provided connection URI

### How do I get API keys for LLM services?

**For OpenAI:**
1. Sign up at https://platform.openai.com/
2. Go to API Keys section
3. Create new secret key
4. Set environment variable: `OPENAI_API_KEY=sk-...`

**For Local Services (No API key needed):**
- **Ollama**: Just install and run `ollama serve`
- **LocalAI**: Run with Docker, no key required
- **vLLM**: Self-hosted, no key required

**For Alternative Cloud Services:**
- Check the specific provider's documentation
- Most use similar API key mechanisms

### What environment variables do I need?

**Minimal Setup (No external dependencies):**
```bash
# No environment variables required!
# ladybug database and basic functionality work out-of-the-box
```

**With LLM Services:**
```bash
# For OpenAI
OPENAI_API_KEY=sk-your-key-here

# For local LLMs (Ollama, LocalAI, etc.)
LLM_BASE_URL=http://localhost:11434  # Optional: defaults vary by service
```

**With External Database (Neo4j):**
```bash
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-password
NEO4J_DATABASE=neo4j  # Optional: defaults to "neo4j"
```

**With ladybug (Optional):**
```bash
ladybug_DB_PATH=./my_graph_db  # Optional: defaults to "./ladybug_db"
```

## Usage Questions

### How do I add data to the knowledge graph?

Create episodes with your content:

```go
episodes := []types.Episode{
    {
        ID:        "unique-id",
        Name:      "Display Name",
        Content:   "Your text content here",
        Reference: time.Now(),  // When it happened
        CreatedAt: time.Now(),  // When you're adding it
        GroupID:   "your-group",
    },
}

err := client.Add(ctx, episodes)
```

The library automatically:
- Extracts entities and relationships
- Creates embeddings for semantic search
- Stores everything in the graph database

### How does the search work?

The hybrid search combines:
1. **Semantic similarity**: Vector embeddings match meaning
2. **Keyword search**: Traditional text matching
3. **Graph traversal**: Explores connected concepts
4. **Ranking**: Combines and ranks all results

```go
results, err := client.Search(ctx, "your query", nil)
```

### Can I customize the search behavior?

Yes, use `SearchConfig`:

```go
config := &types.SearchConfig{
    Limit:              20,    // Max results
    CenterNodeDistance: 3,     // Graph traversal depth  
    MinScore:           0.1,   // Minimum relevance
    IncludeEdges:       true,  // Include relationships
    Rerank:             true,  // Apply reranking
    Filters: &types.SearchFilters{
        NodeTypes:   []types.NodeType{types.EntityNodeType},
        EntityTypes: []string{"Person", "Project"},
        TimeRange: &types.TimeRange{
            Start: time.Now().Add(-7 * 24 * time.Hour),
            End:   time.Now(),
        },
    },
}
```

### How do I handle errors?

The library provides typed errors:

```go
node, err := client.GetNode(ctx, nodeID)
if err != nil {
    if errors.Is(err, graphiti.ErrNodeNotFound) {
        // Handle missing node
        fmt.Println("Node not found")
    } else {
        // Handle other errors
        log.Printf("Error: %v", err)
    }
}
```

### How do I manage resources properly?

Always close clients and drivers:

```go
client := graphiti.NewClient(driver, llm, embedder, config)
defer client.Close(ctx)

driver, err := driver.NewNeo4jDriver(uri, user, pass, db)
if err != nil {
    log.Fatal(err)
}
defer driver.Close(ctx)
```

Use contexts for timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := client.Search(ctx, query, nil)
```

## Performance Questions

### How much does it cost to run?

**OpenAI API costs (approximate):**
- Text processing: $0.002 per 1K tokens (GPT-4o-mini)
- Embeddings: $0.00002 per 1K tokens (text-embedding-3-small)

**Typical usage:**
- Small episodes (500 words): ~$0.002 to process
- Search queries: ~$0.0001 per query
- Monthly cost for active usage: $10-50 for most users

**Neo4j costs:**
- Local/Docker: Free
- Neo4j Aura: Starting at $65/month
- Self-hosted: Infrastructure costs only

### How does it scale?

**Data volume:**
- Tested with 100K+ episodes
- Neo4j can handle millions of nodes/edges
- Search performance remains good with proper indexing

**Query performance:**
- Typical search: 100-500ms
- Complex queries: 1-3 seconds
- Caching can reduce response times significantly

**Concurrent users:**
- Single instance can handle 100+ concurrent requests
- Database is typically the bottleneck
- Horizontal scaling possible with load balancing

### How can I optimize performance?

1. **Batch processing:**
```go
// Process episodes in batches
const batchSize = 100
for i := 0; i < len(episodes); i += batchSize {
    batch := episodes[i:min(i+batchSize, len(episodes))]
    client.Add(ctx, batch)
}
```

2. **Use appropriate search limits:**
```go
config := &types.SearchConfig{
    Limit: 10,  // Don't request more than you need
}
```

3. **Connection pooling:**
   - Neo4j driver handles connection pooling automatically
   - Configure pool size based on load

4. **Database indexing:**
   - Call `driver.CreateIndices(ctx)` after setup
   - Neo4j will create appropriate indexes

## Troubleshooting

### "Connection refused" error

**Cause**: Neo4j is not running or wrong connection details

**Solutions:**
1. Check if Neo4j is running: `docker ps` or Neo4j Desktop status
2. Verify connection URI (bolt:// vs neo4j:// vs https://)
3. Check credentials are correct
4. Ensure ports 7687 (bolt) and 7474 (http) are accessible

### "API key invalid" error

**Cause**: OpenAI API key issues

**Solutions:**
1. Verify API key is correct and properly set
2. Check API key has sufficient credits
3. Ensure no extra spaces or newlines in environment variable
4. Try regenerating the API key

### "Out of memory" error

**Cause**: Processing too much data at once

**Solutions:**
1. Reduce batch size when adding episodes
2. Use search limits to control result size
3. Increase available memory for your application
4. Process data in smaller chunks

### Search returns no results

**Possible causes:**
1. No data has been added yet
2. Search query doesn't match indexed content  
3. GroupID mismatch between client and data
4. MinScore threshold too high

**Debug steps:**
```go
// Check if any data exists
stats, err := driver.GetStats(ctx, groupID)
fmt.Printf("Node count: %d\n", stats.NodeCount)

// Try broader search
results, err := client.Search(ctx, "test", &types.SearchConfig{
    MinScore: 0.0,  // Accept any relevance
    Limit:    1,    // Just need one result
})
```

### Performance is slow

**Common causes:**
1. Missing database indices
2. Large result sets without limits
3. Complex graph traversal
4. Network latency to database/APIs

**Solutions:**
1. Call `driver.CreateIndices(ctx)` 
2. Add search limits and filters
3. Reduce `CenterNodeDistance` in search config
4. Use local Neo4j instance if possible
5. Profile queries in Neo4j browser

### Memory usage keeps growing

**Cause**: Not closing resources properly

**Solution:**
```go
// Always close clients
defer client.Close(ctx)
defer driver.Close(ctx)

// Use context timeouts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

## Contributing and Support

### How can I contribute?

1. **Report issues**: Use GitHub issues for bugs and feature requests
2. **Submit PRs**: Follow Go conventions and include tests
3. **Add drivers**: Implement new database backends
4. **Add providers**: Support new LLM/embedding services
5. **Improve docs**: Help make documentation clearer

### Where can I get help?

1. **Documentation**: Start with [Getting Started](GETTING_STARTED.md)
2. **Examples**: Check [examples/](../examples/) directory
3. **GitHub Issues**: Search existing issues or create new one
4. **Community**: Join discussions in GitHub Discussions

### How do I report a bug?

Include:
1. Go version (`go version`)
2. Library version
3. Database version (Neo4j, etc.)
4. Minimal code example that reproduces the issue
5. Full error message and stack trace
6. Environment details (OS, Docker, cloud provider)

### What's on the roadmap?

**Near term:**
- Additional database drivers (ArangoDB, Neptune)
- More LLM providers (Anthropic, Google)
- Performance optimizations
- Advanced search algorithms

**Medium term:**
- Community detection algorithms
- Temporal query language
- GraphQL API layer
- Distributed deployment support

**Long term:**
- Real-time streaming updates
- Advanced visualization tools
- Plugin architecture for custom processing
- Multi-modal support (images, audio)