# Deferred Graph Ingestion with DuckDB

## Overview

This feature allows you to skip the expensive deduplication and graph ingestion steps when adding episodes to the knowledge graph, and instead write the extracted entities and relationships directly to DuckDB tables for later batch processing.

## Use Cases

- **Bulk Content Ingestion**: When you need to ingest large amounts of content quickly without waiting for deduplication
- **Offline Processing**: Extract entities and relationships first, then deduplicate and ingest into the graph database later
- **Performance Optimization**: Skip expensive LLM calls for deduplication during initial content processing
- **Data Pipeline Separation**: Separate extraction from deduplication/ingestion for better pipeline control

## How It Works

When `AddEpisodeOptions.DeferGraphIngestion` is set to `true`, the following changes occur:

### Phases Skipped

1. **Phase 4 - Entity Resolution & Deduplication**: Extracted entities are used as-is without deduplication
2. **Phase 6 - Relationship Resolution & Temporal Invalidation**: Extracted relationships are used without checking for duplicates or temporal invalidation
3. **Phase 7 - Attribute Extraction**: Nodes are used without additional LLM-based attribute hydration

### Data Storage

Instead of writing to the Kuzu graph database, all data is written to DuckDB tables:

- **episodes**: Episode nodes
- **entity_nodes**: Extracted entity nodes with embeddings
- **entity_edges**: Extracted relationships between entities
- **episodic_edges**: Edges connecting episodes to mentioned entities

## Usage

```go
import (
    "context"
    "github.com/soundprediction/go-graphiti"
    "github.com/soundprediction/go-graphiti/pkg/types"
)

// Create Graphiti client
client := graphiti.NewClient(driver, llmClient, embedderClient, config, logger)

// Define episode
episode := types.Episode{
    ID:        "episode-123",
    Name:      "Patient Consultation",
    Content:   "Patient reports...",
    CreatedAt: time.Now(),
    GroupID:   "patient-001",
}

// Add episode with deferred ingestion
options := &graphiti.AddEpisodeOptions{
    DeferGraphIngestion: true,
    DuckDBPath:         "./data/extracted_content.duckdb",
    EntityTypes:        myEntityTypes,
    EdgeTypes:          myEdgeTypes,
    GenerateEmbeddings: true,
}

result, err := client.AddEpisode(ctx, episode, options)
if err != nil {
    log.Fatal(err)
}
```

## Configuration Options

### `DeferGraphIngestion` (bool)
- **Default**: `false`
- **Description**: When `true`, skips deduplication and writes to DuckDB instead of Kuzu

### `DuckDBPath` (string)
- **Default**: `"./graphiti_deferred.duckdb"`
- **Description**: Path to the DuckDB file for storing extracted data
- **Note**: If empty and `DeferGraphIngestion` is `true`, uses the default path

## DuckDB Schema

### episodes Table
```sql
CREATE TABLE episodes (
    id VARCHAR PRIMARY KEY,
    name VARCHAR,
    content VARCHAR,
    reference TIMESTAMP,
    group_id VARCHAR,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    valid_from TIMESTAMP,
    embedding FLOAT[],
    metadata JSON
)
```

### entity_nodes Table
```sql
CREATE TABLE entity_nodes (
    id VARCHAR PRIMARY KEY,
    name VARCHAR,
    entity_type VARCHAR,
    group_id VARCHAR,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    valid_from TIMESTAMP,
    valid_to TIMESTAMP,
    summary VARCHAR,
    embedding FLOAT[],
    name_embedding FLOAT[],
    metadata JSON,
    episode_id VARCHAR
)
```

### entity_edges Table
```sql
CREATE TABLE entity_edges (
    id VARCHAR PRIMARY KEY,
    source_id VARCHAR,
    target_id VARCHAR,
    name VARCHAR,
    fact VARCHAR,
    summary VARCHAR,
    edge_type VARCHAR,
    group_id VARCHAR,
    created_at TIMESTAMP,
    valid_from TIMESTAMP,
    invalid_at TIMESTAMP,
    expired_at TIMESTAMP,
    embedding FLOAT[],
    fact_embedding FLOAT[],
    episodes JSON,
    metadata JSON,
    episode_id VARCHAR
)
```

### episodic_edges Table
```sql
CREATE TABLE episodic_edges (
    id VARCHAR PRIMARY KEY,
    source_id VARCHAR,
    target_id VARCHAR,
    name VARCHAR,
    edge_type VARCHAR,
    group_id VARCHAR,
    created_at TIMESTAMP,
    valid_from TIMESTAMP,
    episode_id VARCHAR
)
```

## Performance Benefits

### With Deferred Ingestion
- ✅ **Fast extraction**: Only runs entity/relationship extraction LLM calls
- ✅ **No deduplication**: Skips expensive similarity comparisons and LLM deduplication calls
- ✅ **No attribute hydration**: Skips additional LLM calls for entity attributes
- ✅ **Simple writes**: Direct writes to DuckDB (no graph traversal)

### Traditional Ingestion
- ❌ **Full pipeline**: Runs all phases including deduplication
- ❌ **Multiple LLM calls**: Extraction + deduplication + attribute hydration
- ❌ **Graph operations**: Requires graph database queries for deduplication
- ❌ **Slower per-episode**: Each episode processes through full pipeline

## Processing Deferred Data

To process the deferred data and ingest it into the graph database later, use the `DeferredProcessor`:

```go
import (
    "context"
    "github.com/soundprediction/go-graphiti/pkg/deferred"
)

// Create the deferred processor
processor := deferred.NewDeferredProcessor(driver, llmClient, embedderClient, logger)

// Configure processing options
options := &deferred.ProcessDeferredOptions{
    BatchSize:             10,   // Process 10 episodes at a time
    GenerateEmbeddings:    true, // Generate missing embeddings
    DeleteAfterProcessing: true, // Clean up DuckDB after successful ingestion
    EntityTypes:           myEntityTypes,
    GroupID:               "patient-001", // Optional: filter by group
    // EpisodeIDs:         []string{"ep1", "ep2"}, // Optional: process specific episodes
}

// Process all deferred data
result, err := processor.ProcessDeferred(ctx, "./data/extracted_content.duckdb", options)
if err != nil {
    log.Fatal(err)
}

log.Printf("Processed: %d episodes, %d entities, %d edges",
    result.EpisodesProcessed,
    result.EntitiesIngested,
    result.EdgesIngested)
log.Printf("Found %d duplicates, invalidated %d edges",
    result.DuplicatesFound,
    result.EdgesInvalidated)

if len(result.Errors) > 0 {
    log.Printf("Encountered %d errors during processing", len(result.Errors))
}
```

### Check Stats Before Processing

```go
// Get statistics about deferred data without processing
stats, err := processor.GetDeferredStats(ctx, "./data/extracted_content.duckdb")
if err != nil {
    log.Fatal(err)
}

log.Printf("Deferred data: %d episodes, %d nodes, %d edges",
    stats["episodes"],
    stats["entity_nodes"],
    stats["entity_edges"])
```

## Implementation Details

### Modified Phases

**Phase 4 - Entity Resolution & Deduplication**
```go
if options.DeferGraphIngestion {
    // Skip deduplication - use extracted nodes as-is
    resolvedNodes = extractedNodes
    uuidMap = make(map[string]string)
} else {
    // Normal deduplication flow
    resolvedNodes, uuidMap, duplicatePairs, err = nodeOps.ResolveExtractedNodes(...)
}
```

**Phase 6 - Relationship Resolution**
```go
if options.DeferGraphIngestion {
    // Skip edge deduplication and temporal invalidation
    resolvedEdges = extractedEdges
    invalidatedEdges = []*types.Edge{}
} else {
    // Normal edge resolution flow
    resolvedEdges, invalidatedEdges, err = edgeOps.ResolveExtractedEdges(...)
}
```

**Phase 7 - Attribute Extraction**
```go
if options.DeferGraphIngestion {
    // Skip attribute extraction - use resolved nodes as-is
    hydratedNodes = resolvedNodes
} else {
    // Normal attribute extraction
    hydratedNodes, err = nodeOps.ExtractAttributesFromNodes(...)
}
```

**Phase 9 - Persistence**
```go
if !options.DeferGraphIngestion {
    // Write to Kuzu graph database
    utils.AddNodesAndEdgesBulk(ctx, c.driver, ...)
} else {
    // Write to DuckDB tables
    duckDBWriter, err := utils.NewDuckDBWriter(duckDBPath)
    duckDBWriter.WriteEpisode(ctx, episodeNode)
    duckDBWriter.WriteEntityNodes(ctx, hydratedNodes, episodeNode.ID)
    duckDBWriter.WriteEntityEdges(ctx, allEdges, episodeNode.ID)
    duckDBWriter.WriteEpisodicEdges(ctx, episodicEdges, episodeNode.ID)
}
```

## Files Modified/Created

### Modified
- `/Users/josh/workspace/go-graphiti/graphiti.go`
  - Added `DuckDBPath` field to `AddEpisodeOptions`
  - Modified `addEpisodeSingle` to skip deduplication phases when `DeferGraphIngestion=true`
  - Modified Phase 9 persistence to write to DuckDB when enabled

### Created
- `/Users/josh/workspace/go-graphiti/pkg/utils/duckdb_writer.go`
  - `DuckDBWriter` struct for managing DuckDB connections
  - `NewDuckDBWriter()` to create and initialize DuckDB tables
  - `WriteEpisode()` to write episode nodes
  - `WriteEntityNodes()` to batch write entity nodes
  - `WriteEntityEdges()` to batch write entity edges
  - `WriteEpisodicEdges()` to batch write episodic edges

- `/Users/josh/workspace/go-graphiti/pkg/utils/duckdb_writer_test.go`
  - Comprehensive tests for DuckDB writer functionality

- `/Users/josh/workspace/go-graphiti/pkg/deferred/processor.go`
  - `DeferredProcessor` struct for batch processing deferred data
  - `NewDeferredProcessor()` to create a processor instance
  - `ProcessDeferred()` to process all deferred data with full deduplication
  - `GetDeferredStats()` to get statistics about deferred data
  - Implements all missing phases: deduplication, temporal invalidation, attribute extraction

- `/Users/josh/workspace/go-graphiti/pkg/deferred/processor_test.go`
  - Tests for deferred processor functionality

- `/Users/josh/workspace/go-graphiti/DEFERRED_INGESTION.md` (this file)
  - Documentation for the deferred ingestion feature

## Deferred Processor Implementation

The `DeferredProcessor` performs all the steps that were skipped during deferred ingestion:

### Phase 1: Cross-Episode Node Deduplication
- Loads all extracted nodes from DuckDB
- Groups nodes across multiple episodes
- Uses LLM-based deduplication to find duplicate entities
- Creates a UUID mapping to canonical node IDs
- Builds "DUPLICATE_OF" edges to track relationships

### Phase 2: Relationship Resolution & Temporal Invalidation
- Loads all extracted edges from DuckDB
- Updates edge pointers to use canonical node IDs (from deduplication)
- Deduplicates relationships across episodes
- Performs temporal invalidation (marks old edges as invalid when new contradictory facts appear)

### Phase 3: Attribute Extraction
- Uses LLM to extract detailed attributes for each entity
- Hydrates nodes with summaries and metadata

### Phase 4: Bulk Persistence to Kuzu
- Writes all episodes to graph database
- Writes deduplicated entity nodes
- Writes resolved and invalidated edges
- Creates episodic edges (episode → entity mentions)
- Maintains episode metadata with edge references

### Batch Processing
- Processes episodes in configurable batches (default: 10)
- Tracks errors per episode without failing entire batch
- Provides detailed statistics about processing results

## Future Enhancements

1. ~~**Batch Deduplication Tool**: Create a utility to read from DuckDB, deduplicate across all episodes, and ingest into Kuzu~~ ✅ **IMPLEMENTED** (`pkg/deferred/processor.go`)
2. ~~**Statistics**: Add summary statistics about deferred data (entity counts, relationship counts, etc.)~~ ✅ **IMPLEMENTED** (`GetDeferredStats()`)
3. **Incremental Processing**: Support processing subsets of deferred data (partially implemented via `EpisodeIDs` filter)
4. **Validation**: Add validation tools to check deferred data quality before ingestion
5. **Migration**: Create migration utilities to move data between DuckDB and Kuzu
6. **Parallel Processing**: Add support for concurrent batch processing for large datasets
7. **Resume Capability**: Add checkpointing to resume interrupted processing
8. **Dry Run Mode**: Preview what would be processed without making changes

## Example Workflow

### 1. Fast Extraction Phase
```go
// Process 1000 documents quickly
for _, doc := range documents {
    episode := types.Episode{
        ID:      doc.ID,
        Content: doc.Content,
        GroupID: "corpus-001",
    }

    result, err := client.AddEpisode(ctx, episode, &graphiti.AddEpisodeOptions{
        DeferGraphIngestion: true,
        DuckDBPath:         "./data/corpus.duckdb",
    })
    // Fast: Only extraction, no deduplication
}
```

### 2. Batch Deduplication & Ingestion Phase
```go
// Later: deduplicate and ingest in batches
processor := deferred.NewDeferredProcessor(driver, llmClient, embedderClient, logger)

// Check what's in the deferred storage
stats, _ := processor.GetDeferredStats(ctx, "./data/corpus.duckdb")
log.Printf("Ready to process: %d episodes, %d entities, %d relationships",
    stats["episodes"], stats["entity_nodes"], stats["entity_edges"])

// Process with full deduplication
result, err := processor.ProcessDeferred(ctx, "./data/corpus.duckdb", &deferred.ProcessDeferredOptions{
    BatchSize:             50,  // Process 50 episodes at a time
    GenerateEmbeddings:    true,
    DeleteAfterProcessing: true, // Clean up DuckDB after success
    EntityTypes:           myEntityTypes,
    GroupID:               "corpus-001",
})

log.Printf("Processing complete:")
log.Printf("  Episodes: %d", result.EpisodesProcessed)
log.Printf("  Entities: %d (found %d duplicates)", result.EntitiesIngested, result.DuplicatesFound)
log.Printf("  Edges: %d (invalidated %d)", result.EdgesIngested, result.EdgesInvalidated)
```

## Testing

Run the tests:
```bash
# Test DuckDB writer
cd pkg/utils
go test -run TestDuckDBWriter -v

# Test deferred processor
cd pkg/deferred
go test -run TestDeferredProcessorStats -v
```

## Dependencies

- `github.com/marcboeker/go-duckdb/v2` - DuckDB driver for Go
