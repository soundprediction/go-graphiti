# Maintenance Package

The maintenance package provides comprehensive utilities for maintaining and operating on graph nodes, edges, and temporal relationships in go-predicato. This package is a complete Go implementation of the Python predicato maintenance utilities found in `utils/maintenance/`.

## Overview

This package implements four main areas of functionality:

1. **Edge Operations** - Complete edge lifecycle management
2. **Node Operations** - Entity extraction, resolution, and attribute management
3. **Temporal Operations** - Time-based edge operations and contradiction resolution
4. **Graph Data Operations** - Database indices, episode retrieval, and data management
5. **Maintenance Utils** - General graph integrity and utility functions

## Components

### EdgeOperations (`edge_operations.go`)

Handles all edge-related maintenance operations:

- **BuildEpisodicEdges**: Creates MENTIONED_IN edges between episodes and entities
- **BuildDuplicateOfEdges**: Creates IS_DUPLICATE_OF edges for entity deduplication
- **ExtractEdges**: Uses LLM to extract relationship triples from episode content
- **ResolveExtractedEdges**: Resolves new edges against existing ones, handling duplicates and contradictions
- **GetBetweenNodes**: Retrieves edges between two specific nodes
- **FilterExistingDuplicateOfEdges**: Filters duplicate pairs that already have IS_DUPLICATE_OF edges

**Usage:**
```go
edgeOps := maintenance.NewEdgeOperations(driver, llm, embedder, prompts)
edges, err := edgeOps.ExtractEdges(ctx, episode, nodes, previousEpisodes, edgeTypeMap, groupID)
resolvedEdges, invalidated, err := edgeOps.ResolveExtractedEdges(ctx, edges, episode, entities)
```

### NodeOperations (`node_operations.go`)

Manages entity node extraction, resolution, and enhancement:

- **ExtractNodes**: Extracts entity nodes from episode content using LLM with reflexion
- **ResolveExtractedNodes**: Resolves newly extracted nodes against existing ones in the graph
- **ExtractAttributesFromNodes**: Extracts and updates attributes for nodes using LLM

**Usage:**
```go
nodeOps := maintenance.NewNodeOperations(driver, llm, embedder, prompts)
nodes, err := nodeOps.ExtractNodes(ctx, episode, previousEpisodes, entityTypes, excludedTypes)
resolved, uuidMap, duplicates, err := nodeOps.ResolveExtractedNodes(ctx, nodes, episode, previousEpisodes, entityTypes)
enhanced, err := nodeOps.ExtractAttributesFromNodes(ctx, nodes, episode, previousEpisodes, entityTypes)
```

### TemporalOperations (`temporal_operations.go`)

Provides temporal analysis and edge dating operations:

- **ExtractEdgeDates**: Extracts temporal information for edges from episode context
- **GetEdgeContradictions**: Identifies edges that contradict a new edge
- **ExtractAndSaveEdgeDates**: Batch extracts temporal information for multiple edges
- **ValidateEdgeTemporalConsistency**: Validates edge temporal information
- **ApplyTemporalInvalidation**: Applies temporal invalidation logic
- **GetActiveEdgesAtTime**: Returns edges active at a specific time
- **GetEdgeLifespan**: Calculates edge lifespan

**Usage:**
```go
temporalOps := maintenance.NewTemporalOperations(llm, prompts)
validAt, invalidAt, err := temporalOps.ExtractEdgeDates(ctx, edge, currentEpisode, previousEpisodes)
contradicted, err := temporalOps.GetEdgeContradictions(ctx, newEdge, existingEdges)
updated, err := temporalOps.ExtractAndSaveEdgeDates(ctx, edges, episode, previousEpisodes)
```

### GraphDataOperations (`graph_data_operations.go`)

Handles graph database operations and episode management:

- **BuildIndicesAndConstraints**: Creates necessary indices and constraints
- **RetrieveEpisodes**: Retrieves episodic nodes with filtering and time constraints
- **ClearData**: Removes data from the graph (all data or specific group IDs)
- **GetStats**: Returns basic graph statistics

**Usage:**
```go
graphOps := maintenance.NewGraphDataOperations(driver)
err := graphOps.BuildIndicesAndConstraints(ctx, false)
episodes, err := graphOps.RetrieveEpisodes(ctx, referenceTime, 10, groupIDs, source)
stats, err := graphOps.GetStats(ctx, groupID)
```

### MaintenanceUtils (`maintenance_utils.go`)

General utility functions for graph maintenance:

- **GetEntitiesAndEdges**: Retrieves all entities and edges for a group
- **GetEntitiesByType/GetEdgesByType**: Retrieves nodes/edges by specific types
- **GetNodesConnectedToNode**: Gets nodes connected within a distance
- **GetEdgesForNode**: Gets all edges connected to a specific node
- **CleanupOrphanedEdges**: Removes edges referencing non-existent nodes
- **ValidateGraphIntegrity**: Performs comprehensive integrity checks

**Usage:**
```go
utils := maintenance.NewMaintenanceUtils(driver)
entities, edges, err := utils.GetEntitiesAndEdges(ctx, groupID)
orphanedCount, err := utils.CleanupOrphanedEdges(ctx, groupID)
issues, err := utils.ValidateGraphIntegrity(ctx, groupID)
```

## Key Features

### UUID7 Generation
All maintenance operations use UUID7 for generating new UUIDs, ensuring time-ordered identifiers for better database performance.

### Temporal Logic
The package implements sophisticated temporal logic for edge invalidation and contradiction resolution, maintaining temporal consistency across the graph.

### LLM Integration
Seamless integration with the prompt library and LLM clients for:
- Entity extraction with reflexion to catch missed entities
- Edge extraction with fact type classification
- Node and edge deduplication
- Attribute extraction and summarization
- Temporal information extraction

### Embeddings Support
Automatic embedding generation for:
- Extracted nodes based on name and summary
- Extracted edges based on fact summaries
- Semantic similarity for deduplication

### Error Handling
Comprehensive error handling with detailed logging for debugging and monitoring.

### Concurrent Operations
Support for concurrent processing where applicable, with proper synchronization and error aggregation.

## Dependencies

The maintenance package requires:
- `driver.GraphDriver` - For database operations
- `llm.Client` - For LLM-based extraction and reasoning
- `embedder.Client` - For generating semantic embeddings
- `prompts.Library` - For prompt template management

## Integration

This package is designed to be used by the main predicato client and can be integrated into processing pipelines for:
- Episode ingestion and processing
- Graph maintenance and cleanup
- Data integrity validation
- Temporal consistency management

## Relationship to Python Implementation

This Go implementation maintains functional parity with the Python predicato maintenance utilities while leveraging Go's type safety, performance characteristics, and concurrent programming model. All major functions from the Python implementation are represented, with additional utility functions for enhanced graph management.