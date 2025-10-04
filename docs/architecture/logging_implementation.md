# Logging Implementation

**Date:** 2025-10-04
**Status:** ✅ COMPLETED

## Overview

Added structured logging to the `Client` using Go's standard `log/slog` package. The logger provides visibility into each phase of the `AddEpisode` pipeline.

## Changes Made

### 1. Updated Client Constructor

**Modified:** `NewClient()` function signature

```go
// Before
func NewClient(driver driver.GraphDriver, llmClient llm.Client,
    embedderClient embedder.Client, config *Config) *Client

// After
func NewClient(driver driver.GraphDriver, llmClient llm.Client,
    embedderClient embedder.Client, config *Config, logger *slog.Logger) *Client
```

**Behavior:**
- If `logger` is `nil`, uses `slog.Default()`
- Logger is stored in the `Client` struct for use across all methods

### 2. Added Logging to AddEpisode Phases

#### Phase 3: Entity Extraction
```go
c.logger.Info("Starting entity extraction",
    "episode_id", episodeNode.ID,
    "group_id", episode.GroupID,
    "previous_episodes", len(previousEpisodes))

// ... extraction logic ...

c.logger.Info("Entity extraction completed",
    "episode_id", episodeNode.ID,
    "entities_extracted", len(extractedNodes))
```

#### Phase 4: Entity Resolution
```go
c.logger.Info("Starting entity resolution and deduplication",
    "episode_id", episodeNode.ID,
    "entities_to_resolve", len(extractedNodes))

// ... resolution logic ...

c.logger.Info("Entity resolution completed",
    "episode_id", episodeNode.ID,
    "resolved_entities", len(resolvedNodes),
    "duplicates_found", len(duplicatePairs))
```

#### Phase 5: Relationship Extraction
```go
c.logger.Info("Starting relationship extraction",
    "episode_id", episodeNode.ID,
    "entity_count", len(resolvedNodes))

// ... extraction logic ...

c.logger.Info("Relationship extraction completed",
    "episode_id", episodeNode.ID,
    "relationships_extracted", len(extractedEdges))
```

#### Phase 6: Relationship Resolution
```go
c.logger.Info("Starting relationship resolution",
    "episode_id", episodeNode.ID,
    "relationships_to_resolve", len(extractedEdges))

// ... resolution logic ...

c.logger.Info("Relationship resolution completed",
    "episode_id", episodeNode.ID,
    "resolved_relationships", len(resolvedEdges),
    "invalidated_relationships", len(invalidatedEdges))
```

#### Phase 7: Attribute Extraction
```go
c.logger.Info("Starting attribute extraction",
    "episode_id", episodeNode.ID,
    "entities_to_hydrate", len(resolvedNodes))

// ... extraction logic ...

c.logger.Info("Attribute extraction completed",
    "episode_id", episodeNode.ID,
    "hydrated_entities", len(hydratedNodes))
```

#### Phase 10: Community Update
```go
c.logger.Info("Starting community update",
    "episode_id", episodeNode.ID,
    "group_id", episode.GroupID)

// ... community logic ...

c.logger.Info("Community update completed",
    "episode_id", episodeNode.ID,
    "communities", len(result.Communities),
    "community_edges", len(result.CommunityEdges))
```

#### Final Summary
```go
c.logger.Info("Episode processing completed",
    "episode_id", episodeNode.ID,
    "group_id", episode.GroupID,
    "total_entities", len(result.Nodes),
    "total_relationships", len(result.Edges),
    "episodic_edges", len(result.EpisodicEdges),
    "communities", len(result.Communities))
```

## Log Format

All logs use structured logging with key-value pairs:

```
2025/10/04 18:52:46 INFO Starting entity extraction episode_id=test-episode group_id=test-group previous_episodes=0
2025/10/04 18:52:46 INFO Entity extraction completed episode_id=test-episode entities_extracted=0
2025/10/04 18:52:46 INFO Episode processing completed episode_id=test-episode group_id=test-group total_entities=0 total_relationships=0 episodic_edges=0 communities=0
```

## Usage Examples

### Basic Usage (Default Logger)
```go
client := graphiti.NewClient(driver, llmClient, embedderClient, config, nil)
```

### Custom Logger
```go
import "log/slog"

// JSON logger
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

client := graphiti.NewClient(driver, llmClient, embedderClient, config, logger)
```

### Custom Logger with Level Control
```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug, // Show debug logs
}))

client := graphiti.NewClient(driver, llmClient, embedderClient, config, logger)
```

### Disable Logging
```go
import "io"

// Discard all logs
logger := slog.New(slog.NewTextHandler(io.Discard, nil))

client := graphiti.NewClient(driver, llmClient, embedderClient, config, logger)
```

## Logged Metrics

### Per Episode
- **episode_id** - Unique identifier for the episode
- **group_id** - Group/tenant identifier
- **previous_episodes** - Number of previous episodes used for context

### Entity Extraction
- **entities_extracted** - Total entities extracted from episode

### Entity Resolution
- **entities_to_resolve** - Number of entities to deduplicate
- **resolved_entities** - Final number of entities after deduplication
- **duplicates_found** - Number of duplicate pairs identified

### Relationship Extraction
- **entity_count** - Number of entities available for relationship extraction
- **relationships_extracted** - Total relationships extracted

### Relationship Resolution
- **relationships_to_resolve** - Number of relationships to deduplicate
- **resolved_relationships** - Final number of relationships after deduplication
- **invalidated_relationships** - Number of relationships invalidated due to temporal conflicts

### Attribute Extraction
- **entities_to_hydrate** - Number of entities to enrich with attributes
- **hydrated_entities** - Number of entities successfully hydrated

### Community Update
- **communities** - Number of community nodes created/updated
- **community_edges** - Number of community edges created

### Final Summary
- **total_entities** - Total entities in final result
- **total_relationships** - Total relationships in final result
- **episodic_edges** - Total episodic edges created
- **communities** - Total communities in final result

## Benefits

1. **Observability** - Track episode processing in production
2. **Debugging** - Identify which phase is failing or slow
3. **Metrics** - Extract metrics for monitoring dashboards
4. **Auditing** - Trace entity/relationship extraction over time
5. **Performance** - Identify bottlenecks in the pipeline

## Future Enhancements

1. Add timing metrics for each phase
2. Add error logging with context
3. Add debug-level logs for detailed tracing
4. Add OpenTelemetry tracing support
5. Add metrics export (Prometheus, StatsD, etc.)

## Breaking Changes

**Yes** - The `NewClient()` function signature changed to include a `logger` parameter.

### Migration Guide

**Old Code:**
```go
client := graphiti.NewClient(driver, llmClient, embedderClient, config)
```

**New Code:**
```go
// Use default logger
client := graphiti.NewClient(driver, llmClient, embedderClient, config, nil)

// Or use custom logger
logger := slog.Default()
client := graphiti.NewClient(driver, llmClient, embedderClient, config, logger)
```

## Testing

All tests updated to pass `nil` for the logger parameter, which uses `slog.Default()`.

```bash
$ go test -v .
=== RUN   TestClient_Add
2025/10/04 18:52:46 INFO Starting entity extraction episode_id=test-episode group_id=test-group previous_episodes=0
2025/10/04 18:52:46 INFO Entity extraction completed episode_id=test-episode entities_extracted=0
2025/10/04 18:52:46 INFO Episode processing completed episode_id=test-episode group_id=test-group total_entities=0 total_relationships=0 episodic_edges=0 communities=0
--- PASS: TestClient_Add (0.00s)
PASS
```

## References

- Go slog documentation: https://pkg.go.dev/log/slog
- Structured logging best practices: https://go.dev/blog/slog
