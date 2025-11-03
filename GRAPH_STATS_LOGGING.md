# Graph Database Statistics Logging

## Overview

After Step 14 in the episode ingestion workflow, the system now reports comprehensive graph database statistics to provide visibility into the overall state of the knowledge graph.

## Implementation

### Location

**File**: `ingestion.go`
**Function**: `addEpisodeChunked`
**Line**: 395-411 (Step 15)

### Code Added

```go
// STEP 15: Report overall graph database statistics
stats, err := c.driver.GetStats(ctx, episode.GroupID)
if err == nil {
    c.logger.Info("Graph database statistics after episode processing",
        "episode_id", episode.ID,
        "group_id", episode.GroupID,
        "total_nodes", stats.NodeCount,
        "total_edges", stats.EdgeCount,
        "total_communities", stats.CommunityCount,
        "entity_nodes", stats.NodesByType["Entity"],
        "episodic_nodes", stats.NodesByType["Episodic"],
        "community_nodes", stats.NodesByType["Community"])
} else {
    c.logger.Warn("Failed to retrieve graph database statistics",
        "episode_id", episode.ID,
        "error", err)
}
```

## Statistics Reported

After each episode is processed, the following statistics are logged:

### Overall Metrics
- **total_nodes**: Total number of nodes in the graph for this group
- **total_edges**: Total number of edges in the graph for this group
- **total_communities**: Total number of community nodes

### Breakdown by Node Type
- **entity_nodes**: Number of Entity nodes (people, places, concepts, etc.)
- **episodic_nodes**: Number of Episodic nodes (conversation chunks, events)
- **community_nodes**: Number of Community nodes (clusters of related entities)

## Log Output Example

### Step 14: Episode Processing Summary
```
time=2025-11-02T23:14:42.879-05:00 level=INFO
msg="Chunked episode processing completed with bulk deduplication"
episode_id=episode-12345
total_chunks=3
total_entities=15
total_relationships=8
total_episodic_edges=15
total_communities=2
```

### Step 15: Graph Database Statistics (NEW)
```
time=2025-11-02T23:14:42.879-05:00 level=INFO
msg="Graph database statistics after episode processing"
episode_id=episode-12345
group_id=default-group
total_nodes=127
total_edges=83
total_communities=5
entity_nodes=112
episodic_nodes=10
community_nodes=5
```

## Use Cases

### 1. Monitoring Graph Growth
Track how the knowledge graph evolves over time:
- Monitor total node/edge counts across episodes
- Identify when the graph reaches certain size thresholds
- Detect anomalous growth patterns

### 2. Debugging
Quickly diagnose issues:
- Verify entities are being created (check `entity_nodes`)
- Confirm communities are being built (check `community_nodes`)
- Detect if episodes are processing correctly (check `episodic_nodes`)

### 3. Performance Analysis
Understand the relationship between graph size and performance:
- Correlate processing times with graph size
- Identify when to optimize queries or add indexing
- Plan for scaling as the graph grows

### 4. Data Quality Assurance
Validate data integrity:
- Ensure node counts align with expectations
- Verify community building is working (communities should appear after sufficient entities)
- Check for data loss or duplication

## Error Handling

If statistics retrieval fails, a warning is logged instead of failing the entire operation:

```
time=2025-11-02T23:14:42.879-05:00 level=WARN
msg="Failed to retrieve graph database statistics"
episode_id=episode-12345
error="database connection timeout"
```

This ensures that episode processing completes successfully even if stats collection has issues.

## Driver Support

Both database drivers support statistics collection:

### Kuzu Driver (`pkg/driver/kuzu.go`)
- Queries node and edge counts by type
- Efficiently aggregates statistics using Cypher `count()` functions

### Memgraph Driver (`pkg/driver/memgraph.go`)
- Uses Neo4j-compatible Cypher queries
- Provides same statistics structure as Kuzu

## GraphStats Structure

```go
type GraphStats struct {
    NodeCount      int64            // Total nodes in group
    EdgeCount      int64            // Total edges in group
    NodesByType    map[string]int64 // Breakdown by node type
    EdgesByType    map[string]int64 // Breakdown by edge type
    CommunityCount int64            // Total communities
    LastUpdated    time.Time        // When stats were collected
}
```

## Integration with Existing Logging

The new statistics logging integrates seamlessly with existing workflow:

```
Step 1-13: Episode processing steps...
Step 14: Log episode-specific results (entities, relationships, communities created)
Step 15: Log overall graph statistics (cumulative state)
```

This provides both:
- **Incremental view**: What was added in this episode (Step 14)
- **Cumulative view**: Total state of the graph (Step 15)

## Future Enhancements

Potential improvements to consider:

1. **Historical Tracking**: Store stats over time to visualize growth trends
2. **Group Comparison**: Compare statistics across different groups
3. **Alerts**: Trigger notifications when stats cross thresholds
4. **Detailed Breakdown**: Add edge type statistics (RELATES_TO, HAS_MEMBER, etc.)
5. **Performance Metrics**: Include query execution times in stats
6. **Export**: Provide API to export statistics for external monitoring tools

## Notes

- Statistics are scoped to the episode's `group_id`
- Stats collection is fast (single query) and doesn't significantly impact performance
- The feature gracefully degrades if stats collection fails (warning logged, processing continues)
- Logs use structured logging format compatible with log aggregation tools
