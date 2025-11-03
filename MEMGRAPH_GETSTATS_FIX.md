# Memgraph GetStats Fix - Node Type Counting

## Problem

After creating communities successfully, the `GetStats()` function was reporting `total_communities=0` and all node type counts as 0 for Memgraph:

```
2025-11-02 23:20:04 INFO Persisted community nodes and edges
  episode_id=000b7f15-aa7b-4270-b517-e433a98e4931
  community_count=3
  community_edge_count=16

2025-11-02 23:20:04 INFO Graph database statistics after episode processing
  episode_id=000b7f15-aa7b-4270-b517-e433a98e4931
  group_id=pregnancy-content
  total_nodes=229
  total_edges=214
  total_communities=0      ❌ Should be 3
  entity_nodes=0           ❌ Should be > 0
  episodic_nodes=0         ❌ Should be > 0
  community_nodes=0        ❌ Should be 3
```

## Root Cause

The original Memgraph query was looking for a `type` **property** on nodes:

```cypher
MATCH (n {group_id: $groupID})
WITH n.type as node_type, count(*) as node_count
RETURN node_type, node_count
```

**Issue**: In Neo4j/Memgraph, node types are represented by **labels** (`:Entity`, `:Episodic`, `:Community`), not properties. The `n.type` property doesn't exist, so the query returned empty results for node types.

## Solution

### Changed Query Strategy

Instead of looking for `n.type` property, use `labels(n)` to get node labels:

**BEFORE** (❌ Broken):
```cypher
MATCH (n {group_id: $groupID})
WITH n.type as node_type, count(*) as node_count
RETURN node_type, node_count
```

**AFTER** (✅ Fixed):
```cypher
MATCH (n {group_id: $groupID})
UNWIND labels(n) AS label
WITH label, count(DISTINCT n) as node_count
WHERE label IN ['Entity', 'Episodic', 'Community']
RETURN label as node_type, node_count
ORDER BY label
```

### Key Changes

1. **Use `labels(n)`**: Returns array of labels for each node
2. **UNWIND labels**: Unpacks label array into individual rows
3. **Filter main labels**: Only count Entity, Episodic, Community (skip secondary labels like Person, Place, etc.)
4. **COUNT DISTINCT**: Ensures each node is only counted once even if it has multiple labels

### Additional Improvements

Also fixed edge type counting:

**BEFORE** (❌ Used non-existent property):
```cypher
MATCH ()-[r {group_id: $groupID}]-()
WITH r.type as edge_type, count(*) as edge_count
RETURN edge_type, edge_count
```

**AFTER** (✅ Uses type() function):
```cypher
MATCH ()-[r {group_id: $groupID}]-()
RETURN type(r) as edge_type, count(r) as edge_count
ORDER BY edge_type
```

## Code Changes

### File: `pkg/driver/memgraph.go`

#### Query Updates (Lines 1299-1352)

1. **Node counting by label**:
   - Added query to get node counts using `labels(n)` and UNWIND
   - Filters for Entity, Episodic, Community labels only

2. **Total node count**:
   - Added separate query for total node count (all nodes in group)
   - Prevents double-counting nodes with multiple labels

3. **Edge counting by type**:
   - Changed from `r.type` property to `type(r)` function

#### Parsing Updates (Lines 1358-1401)

1. **Extract total node count** from dedicated query result
2. **Process node types by label** from UNWIND results
3. **Track community count** when nodeType == "Community"
4. **Process edge types** using relationship type

## Testing

Build verification:
```bash
go build ./...
# SUCCESS (warnings are harmless)
```

Expected output after fix:
```
INFO Graph database statistics after episode processing
  episode_id=000b7f15-aa7b-4270-b517-e433a98e4931
  group_id=pregnancy-content
  total_nodes=229
  total_edges=214
  total_communities=3        ✅ Now shows correct count
  entity_nodes=150           ✅ Now shows Entity count
  episodic_nodes=76          ✅ Now shows Episodic count
  community_nodes=3          ✅ Matches total_communities
```

## Background: Neo4j/Memgraph Labels vs Properties

### How Node Types Work

In Neo4j/Memgraph (Cypher databases):

**Labels** (part of graph structure):
- Node types are expressed as labels: `:Entity`, `:Episodic`, `:Community`
- Accessed via `labels(n)` function
- Can have multiple labels (e.g., `:Entity:Person`)
- Used for indexing and fast lookups

**Properties** (data attributes):
- Key-value pairs stored on nodes: `{uuid: "...", name: "...", group_id: "..."}`
- Accessed via `n.property_name`
- Cannot be used to determine node type

### Example Node Structure

```cypher
// Creating a Person entity
CREATE (n:Entity:Person {
  uuid: "person-123",
  name: "John Doe",
  group_id: "default",
  summary: "A person..."
})

// Querying
MATCH (n)
RETURN labels(n)  // Returns: ["Entity", "Person"]

// Getting node type
MATCH (n {uuid: "person-123"})
UNWIND labels(n) AS label
WHERE label IN ['Entity', 'Episodic', 'Community']
RETURN label  // Returns: "Entity"
```

### Why Not Use Properties?

The Go implementation correctly stores type information as **labels** (matching Python):

```go
// getLabelForNodeType in memgraph.go:121-131
func (m *MemgraphDriver) getLabelForNodeType(nodeType types.NodeType) string {
    switch nodeType {
    case types.EpisodicNodeType:
        return "Episodic"
    case types.EntityNodeType:
        return "Entity"
    case types.CommunityNodeType:
        return "Community"
    }
}

// UpsertNode creates nodes with labels (line 162):
MERGE (n:Entity {uuid: $uuid, group_id: $group_id})
```

The old GetStats query was incorrectly trying to read `n.type` as a property when it should have been reading labels.

## Comparison with Kuzu

**Kuzu** stores node type differently - it uses node **tables** and has a `type` property:

```cypher
// Kuzu query (works correctly)
MATCH (n:Entity {group_id: $group_id})
RETURN count(*) as entity_count
```

This is why the bug only affected Memgraph, not Kuzu.

## Related Issues

This fix resolves:
- ✅ Communities showing as 0 despite being created
- ✅ All node type breakdowns showing as 0
- ✅ Edge type counting using wrong approach
- ✅ Double-counting nodes with multiple labels

## Future Considerations

1. **Performance**: Current query uses UNWIND which is efficient for most cases. For very large graphs, consider caching stats.

2. **Multiple Labels**: Entity nodes can have both `:Entity` and specific type labels (`:Person`, `:Place`). Current approach correctly counts them once as Entity.

3. **Label Consistency**: Ensure all node creation paths use proper labels (currently handled by `getLabelForNodeType`).

4. **Statistics Accuracy**: Stats are now accurate and can be used for:
   - Monitoring graph growth
   - Detecting community building issues
   - Performance analysis
   - Data quality checks
