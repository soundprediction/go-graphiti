# Add Episode Implementation Gap Analysis

## Overview

This document identifies gaps between the Python `Graphiti.add_episode()` implementation and the Go `Client.Add()` / `Client.AddEpisode()` implementation, based on the flow diagram in `add_episode_flow.md`.

**Date:** 2025-10-04
**Python Reference:** [graphiti/graphiti_core/graphiti.py:376-573](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L376)
**Go Implementation:** `/Users/josh/workspace/go-graphiti/graphiti.go:156-308`

## Executive Summary

**Status: CRITICAL GAPS FOUND**

The Go implementation has the necessary infrastructure (NodeOperations, EdgeOperations) but the main `Add()` and `AddEpisode()` functions are NOT using them properly. The current implementation uses simplified extraction logic that bypasses critical deduplication, resolution, and validation steps.

### Critical Missing Features

1. ❌ **Input Validation** - No validation of entity types, excluded types, or group IDs
2. ❌ **Node Resolution & Deduplication** - Using simple name-based dedup instead of hybrid search + LLM
3. ❌ **Attribute Extraction** - Not extracting structured attributes or summaries for nodes
4. ❌ **Edge Resolution & Deduplication** - Not deduplicating edges properly
5. ❌ **Temporal Edge Invalidation** - Not handling edge contradictions or temporal validity
6. ❌ **Reflexion Pattern** - Not using reflexion to validate extractions
7. ❌ **Previous Episode Context** - Not retrieving previous episodes for context

## Detailed Gap Analysis

### Phase 1: Validation (MISSING)

#### Python Implementation
```python
validate_entity_types(entity_types)
validate_excluded_entity_types(excluded_entity_types, entity_types)
validate_group_id(group_id)
group_id = group_id or get_default_group_id(self.driver.provider)
```

#### Go Implementation
```go
// NO VALIDATION AT ALL!
// Just proceeds directly to episode creation
```

#### Required Changes
- [ ] Add validation calls at the start of `AddEpisode()`:
  ```go
  if err := utils.ValidateEntityTypes(options.EntityTypes); err != nil {
      return nil, fmt.Errorf("invalid entity types: %w", err)
  }

  entityTypeNames := make([]string, 0, len(options.EntityTypes))
  for name := range options.EntityTypes {
      entityTypeNames = append(entityTypeNames, name)
  }

  if err := utils.ValidateExcludedEntityTypes(options.ExcludedEntityTypes, entityTypeNames); err != nil {
      return nil, fmt.Errorf("invalid excluded entity types: %w", err)
  }

  if err := utils.ValidateGroupID(episode.GroupID); err != nil {
      return nil, fmt.Errorf("invalid group ID: %w", err)
  }

  if episode.GroupID == "" {
      episode.GroupID = utils.GetDefaultGroupID(c.driver.GetProvider())
  }
  ```

**Impact:** HIGH - Invalid inputs can corrupt the graph

---

### Phase 2: Context Retrieval (PARTIALLY MISSING)

#### Python Implementation
```python
previous_episodes = (
    await self.retrieve_episodes(
        reference_time,
        last_n=RELEVANT_SCHEMA_LIMIT,
        group_ids=[group_id],
        source=source,
    )
    if previous_episode_uuids is None
    else await EpisodicNode.get_by_uuids(self.driver, previous_episode_uuids)
)
```

#### Go Implementation
```go
// HARDCODED TO EMPTY!
previousEpisodes := []string{} // Would be populated with recent episode content
```

#### Required Changes
- [ ] Implement `Client.GetEpisodes()` function (already exists at line 1253)
- [ ] Call it in `extractEntities()` and `extractRelationships()`:
  ```go
  var previousEpisodes []*types.Node
  var err error

  if options != nil && len(options.PreviousEpisodeUUIDs) > 0 {
      // Get specific episodes by UUIDs
      previousEpisodes, err = c.getEpisodesByUUIDs(ctx, options.PreviousEpisodeUUIDs)
  } else {
      // Get recent episodes for context
      previousEpisodes, err = c.GetEpisodes(ctx, episode.Reference,
          maintenance.RelevantSchemaLimit, []string{episode.GroupID}, episode.EpisodeType)
  }
  if err != nil {
      return nil, fmt.Errorf("failed to retrieve previous episodes: %w", err)
  }
  ```

**Impact:** HIGH - Lack of context degrades extraction quality

---

### Phase 3: Entity Extraction (PARTIALLY IMPLEMENTED)

#### Python Implementation
```python
extracted_nodes = await extract_nodes(
    self.clients, episode, previous_episodes, entity_types, excluded_entity_types
)
```
- Uses **reflexion** to validate missed entities
- Supports custom entity types
- Supports excluded entity types

#### Go Implementation
```go
extractedNodes, err = c.extractEntities(ctx, episode)
```
- ✅ Has reflexion implemented in `NodeOperations.extractNodesReflexion()`
- ❌ NOT using `NodeOperations.ExtractNodes()` which has the full implementation
- ❌ NOT passing entity types or excluded types
- ❌ NOT using previous episodes context

#### Required Changes
- [ ] Replace `c.extractEntities()` with `NodeOperations.ExtractNodes()`:
  ```go
  nodeOps := maintenance.NewNodeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())

  extractedNodes, err := nodeOps.ExtractNodes(ctx, episodeNode, previousEpisodes,
      options.EntityTypes, options.ExcludedEntityTypes)
  if err != nil {
      return nil, fmt.Errorf("failed to extract nodes: %w", err)
  }
  ```

**Impact:** MEDIUM - Missing reflexion and entity type support

---

### Phase 4: Entity Resolution (CRITICAL GAP)

#### Python Implementation
```python
(nodes, uuid_map, _), extracted_edges = await semaphore_gather(
    resolve_extracted_nodes(
        self.clients,
        extracted_nodes,
        episode,
        previous_episodes,
        entity_types,
    ),
    # ... edge extraction in parallel
)
```

**Python `resolve_extracted_nodes()` does:**
1. **Hybrid search** for similar existing nodes
2. **Deterministic similarity matching** using MinHash
3. **LLM-based deduplication** for ambiguous cases
4. **Filters existing DUPLICATE_OF edges**
5. Returns `(resolved_nodes, uuid_map, duplicate_pairs)`

#### Go Implementation
```go
finalNodes, err := c.deduplicateAndStoreNodes(ctx, extractedNodes, episode.GroupID, options)
```

**Go `deduplicateAndStoreNodes()` does:**
1. ❌ **Simple name-based search** only
2. ❌ No hybrid search (semantic + BM25)
3. ❌ No similarity matching
4. ❌ No LLM-based deduplication
5. ❌ No duplicate edge filtering

#### Required Changes
- [ ] Replace `c.deduplicateAndStoreNodes()` with `NodeOperations.ResolveExtractedNodes()`:
  ```go
  nodeOps := maintenance.NewNodeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())

  resolvedNodes, uuidMap, duplicatePairs, err := nodeOps.ResolveExtractedNodes(ctx,
      extractedNodes, episodeNode, previousEpisodes, options.EntityTypes)
  if err != nil {
      return nil, fmt.Errorf("failed to resolve nodes: %w", err)
  }

  // Store duplicate edges
  edgeOps := maintenance.NewEdgeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())
  duplicateEdges, err := edgeOps.BuildDuplicateOfEdges(ctx, episodeNode, time.Now(), duplicatePairs)
  if err != nil {
      return nil, fmt.Errorf("failed to build duplicate edges: %w", err)
  }

  for _, edge := range duplicateEdges {
      if err := c.driver.UpsertEdge(ctx, edge); err != nil {
          return nil, fmt.Errorf("failed to store duplicate edge: %w", err)
      }
  }
  ```

**Impact:** CRITICAL - Poor deduplication causes node proliferation and graph pollution

---

### Phase 5: Relationship Extraction (PARTIALLY IMPLEMENTED)

#### Python Implementation
```python
extracted_edges = await extract_edges(
    self.clients,
    episode,
    extracted_nodes,
    previous_episodes,
    edge_type_map or edge_type_map_default,
    group_id,
    edge_types,
)
```
- Uses reflexion for missed facts
- Supports custom edge types and edge type maps
- Uses previous episodes for context

#### Go Implementation
```go
extractedEdges, err = c.extractRelationships(ctx, episode, finalNodes)
```
- ✅ Has reflexion implemented in `EdgeOperations.extractEdgesReflexion()`
- ❌ NOT using `EdgeOperations.ExtractEdges()`
- ❌ NOT passing edge types or edge type map
- ❌ NOT using previous episodes context

#### Required Changes
- [ ] Replace `c.extractRelationships()` with `EdgeOperations.ExtractEdges()`:
  ```go
  edgeOps := maintenance.NewEdgeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())

  edgeTypeMap := options.EdgeTypeMap
  if edgeTypeMap == nil && options.EdgeTypes != nil {
      // Create default edge type map
      edgeTypeMap = map[string][]string{
          "Entity_Entity": make([]string, 0, len(options.EdgeTypes)),
      }
      for edgeType := range options.EdgeTypes {
          edgeTypeMap["Entity_Entity"] = append(edgeTypeMap["Entity_Entity"], edgeType)
      }
  }

  extractedEdges, err := edgeOps.ExtractEdges(ctx, episodeNode, resolvedNodes,
      previousEpisodes, edgeTypeMap, episode.GroupID)
  if err != nil {
      return nil, fmt.Errorf("failed to extract edges: %w", err)
  }
  ```

**Impact:** MEDIUM - Missing custom edge types and reflexion

---

### Phase 6: Relationship Resolution (CRITICAL GAP)

#### Python Implementation
```python
edges = resolve_edge_pointers(extracted_edges, uuid_map)

(resolved_edges, invalidated_edges), hydrated_nodes = await semaphore_gather(
    resolve_extracted_edges(
        self.clients,
        edges,
        episode,
        nodes,
        edge_types or {},
        edge_type_map or edge_type_map_default,
    ),
    extract_attributes_from_nodes(
        self.clients, nodes, episode, previous_episodes, entity_types
    ),
)
```

**Python `resolve_extracted_edges()` does:**
1. **Hybrid search** for duplicate/contradictory edges
2. **LLM-based deduplication**
3. **Temporal contradiction handling** (invalidate old edges)
4. **Structured attribute extraction** for custom edge types
5. Returns `(resolved_edges, invalidated_edges)`

#### Go Implementation
```go
// NO RESOLUTION AT ALL!
for _, edge := range extractedEdges {
    // Just generate embedding and store
    if err := c.driver.UpsertEdge(ctx, edge); err != nil {
        return nil, fmt.Errorf("failed to store edge %s: %w", edge.BaseEdge.ID, err)
    }
}
```

#### Required Changes
- [ ] Implement proper edge resolution:
  ```go
  // 1. Resolve edge pointers using uuid map from node resolution
  edgeOps := maintenance.NewEdgeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())
  resolvedPointerEdges := maintenance.ResolveEdgePointers(extractedEdges, uuidMap)

  // 2. Resolve extracted edges (dedupe + invalidation)
  resolvedEdges, invalidatedEdges, err := edgeOps.ResolveExtractedEdges(ctx,
      resolvedPointerEdges, episodeNode, resolvedNodes)
  if err != nil {
      return nil, fmt.Errorf("failed to resolve edges: %w", err)
  }

  // 3. Store all edges (resolved + invalidated)
  allEdges := append(resolvedEdges, invalidatedEdges...)
  for _, edge := range allEdges {
      if err := c.driver.UpsertEdge(ctx, edge); err != nil {
          return nil, fmt.Errorf("failed to store edge: %w", err)
      }
  }

  result.Edges = allEdges
  ```

**Impact:** CRITICAL - No deduplication or temporal invalidation of edges

---

### Phase 7: Attribute Extraction (MISSING)

#### Python Implementation
```python
hydrated_nodes = await extract_attributes_from_nodes(
    self.clients, nodes, episode, previous_episodes, entity_types
)
```

**Python `extract_attributes_from_nodes()` does:**
1. **Extracts structured attributes** for custom entity types
2. **Generates summaries** for each entity
3. **Creates name embeddings** for each entity
4. Runs in parallel for all nodes

#### Go Implementation
```go
// COMPLETELY MISSING!
// Just stores nodes without attributes or summaries
```

#### Required Changes
- [ ] Add attribute extraction after node resolution:
  ```go
  nodeOps := maintenance.NewNodeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())

  hydratedNodes, err := nodeOps.ExtractAttributesFromNodes(ctx,
      resolvedNodes, episodeNode, previousEpisodes, options.EntityTypes)
  if err != nil {
      return nil, fmt.Errorf("failed to extract attributes: %w", err)
  }

  // Store hydrated nodes
  for _, node := range hydratedNodes {
      if err := c.driver.UpsertNode(ctx, node); err != nil {
          return nil, fmt.Errorf("failed to store hydrated node: %w", err)
      }
  }

  result.Nodes = hydratedNodes
  ```

**Impact:** HIGH - Nodes lack structured attributes and summaries

---

### Phase 8: Build Episodic Edges (PARTIALLY CORRECT)

#### Python Implementation
```python
episodic_edges = build_episodic_edges(nodes, episode.uuid, now)
episode.entity_edges = [edge.uuid for edge in entity_edges]
```

#### Go Implementation
```go
for _, node := range finalNodes {
    episodeEdge := types.NewEntityEdge(
        generateID(),
        episodeNode.ID,
        node.ID,
        episode.GroupID,
        "MENTIONED_IN",
        types.EpisodicEdgeType,
    )
    // ...
}
```

#### Required Changes
- [ ] Use `EdgeOperations.BuildEpisodicEdges()` for consistency:
  ```go
  edgeOps := maintenance.NewEdgeOperations(c.driver, c.llm, c.embedder, prompts.NewLibrary())

  episodicEdges, err := edgeOps.BuildEpisodicEdges(ctx, hydratedNodes, episodeNode.ID, time.Now())
  if err != nil {
      return nil, fmt.Errorf("failed to build episodic edges: %w", err)
  }

  for _, edge := range episodicEdges {
      if err := c.driver.UpsertEdge(ctx, edge); err != nil {
          return nil, fmt.Errorf("failed to store episodic edge: %w", err)
      }
  }

  result.EpisodicEdges = episodicEdges
  ```
- [ ] Store entity edge UUIDs on episode node

**Impact:** LOW - Current implementation mostly works but not using shared utilities

---

### Phase 9: Persistence (MISSING BULK OPERATIONS)

#### Python Implementation
```python
await add_nodes_and_edges_bulk(
    self.driver, [episode], episodic_edges, hydrated_nodes, entity_edges, self.embedder
)
```
- Uses **bulk operations** for efficiency
- Generates embeddings in bulk
- Single transaction for consistency

#### Go Implementation
```go
// Individual UpsertNode/UpsertEdge calls throughout the function
for _, node := range finalNodes {
    if err := c.driver.UpsertNode(ctx, node); err != nil {
        // ...
    }
}
```

#### Required Changes
- [ ] Implement bulk persistence:
  ```go
  // Use bulk operations from pkg/utils/bulk.go
  bulkOps := bulk.NewBulkOperations(c.driver, c.embedder)

  err := bulkOps.AddNodesAndEdgesBulk(ctx,
      []*types.Node{episodeNode},
      episodicEdges,
      hydratedNodes,
      allEdges)
  if err != nil {
      return nil, fmt.Errorf("failed to bulk persist data: %w", err)
  }
  ```

**Impact:** MEDIUM - Performance and consistency issues

---

### Phase 10: Community Update (CORRECT)

#### Python Implementation
```python
if update_communities:
    communities, community_edges = await semaphore_gather(
        *[
            update_community(self.driver, self.llm_client, self.embedder, node)
            for node in nodes
        ],
    )
```

#### Go Implementation
```go
if options.UpdateCommunities {
    communityResult, err := c.community.BuildCommunities(ctx, []string{episode.GroupID})
    // ...
}
```

✅ **Already implemented correctly!**

---

## Implementation Priority

### P0 - Critical (Breaks Core Functionality)
1. **Input Validation** - Add validation calls
2. **Node Resolution** - Use `NodeOperations.ResolveExtractedNodes()`
3. **Edge Resolution** - Use `EdgeOperations.ResolveExtractedEdges()`
4. **Attribute Extraction** - Use `NodeOperations.ExtractAttributesFromNodes()`

### P1 - High (Degrades Quality)
5. **Previous Episode Context** - Retrieve and use previous episodes
6. **Entity Extraction** - Use `NodeOperations.ExtractNodes()` with reflexion
7. **Edge Extraction** - Use `EdgeOperations.ExtractEdges()` with reflexion

### P2 - Medium (Performance & Consistency)
8. **Bulk Operations** - Use bulk persistence
9. **Episodic Edges** - Use `EdgeOperations.BuildEpisodicEdges()`

## Summary Statistics

| Phase | Python Functions | Go Equivalent | Status |
|-------|-----------------|---------------|--------|
| 1. Validation | `validate_entity_types()`, `validate_excluded_entity_types()`, `validate_group_id()` | ✅ Exists but ❌ Not called | ⚠️ Missing |
| 2. Context Retrieval | `retrieve_episodes()`, `EpisodicNode.get_by_uuids()` | ✅ Exists but ❌ Not used | ⚠️ Missing |
| 3. Entity Extraction | `extract_nodes()` with reflexion | ✅ Exists but ❌ Simplified version used | ⚠️ Partial |
| 4. Entity Resolution | `resolve_extracted_nodes()` | ✅ Exists but ❌ Not used | ❌ Critical |
| 5. Edge Extraction | `extract_edges()` with reflexion | ✅ Exists but ❌ Simplified version used | ⚠️ Partial |
| 6. Edge Resolution | `resolve_extracted_edges()` | ✅ Exists but ❌ Not used | ❌ Critical |
| 7. Attribute Extraction | `extract_attributes_from_nodes()` | ✅ Exists but ❌ Not used | ❌ Critical |
| 8. Episodic Edges | `build_episodic_edges()` | ✅ Exists but ⚠️ Custom impl | ⚠️ Partial |
| 9. Persistence | `add_nodes_and_edges_bulk()` | ✅ Exists but ❌ Not used | ⚠️ Missing |
| 10. Communities | `update_community()` | ✅ Correctly implemented | ✅ Complete |

**Overall Status:** 🔴 **1/10 phases fully complete, 7/10 critical or high priority gaps**

## Recommended Fix

Create a new `Client.AddEpisodeV2()` function that properly uses all the maintenance operations:

```go
func (c *Client) AddEpisodeV2(ctx context.Context, episode types.Episode, options *AddEpisodeOptions) (*types.AddEpisodeResults, error) {
    // Phase 1: Validation
    // Phase 2: Context Retrieval
    // Phase 3: Entity Extraction (use NodeOperations)
    // Phase 4: Entity Resolution (use NodeOperations)
    // Phase 5: Edge Extraction (use EdgeOperations)
    // Phase 6: Edge Resolution (use EdgeOperations)
    // Phase 7: Attribute Extraction (use NodeOperations)
    // Phase 8: Episodic Edges (use EdgeOperations)
    // Phase 9: Bulk Persistence (use BulkOperations)
    // Phase 10: Communities (already correct)
}
```

This would match the Python implementation's flow exactly while leveraging all the existing Go infrastructure.
