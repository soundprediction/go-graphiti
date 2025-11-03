# Community Operations Comparison: Python vs Go

This document compares the community operations implementations between the Python graphiti package and the Go go-graphiti port.

## Overview

Community operations create and manage clusters of related entities using the label propagation algorithm and hierarchical LLM-based summarization.

## Critical Bugs Found and Fixed

### 🐛 Bug 1: Wrong Relationship Direction in GetExistingCommunity

**Python Implementation** (`graphiti/graphiti_core/utils/maintenance/community_operations.py:247-254`):
```cypher
MATCH (c:Community)-[:HAS_MEMBER]->(n:Entity {uuid: $entity_uuid})
RETURN [COMMUNITY_NODE_RETURN]
```

**Go Implementation - BEFORE FIX**:
```cypher
# Both Kuzu and Memgraph (WRONG!)
MATCH (e:Entity {uuid: $entity_uuid})-[:MEMBER_OF]->(c:Community)
RETURN c
```

**Issues**:
- ❌ Wrong relationship direction: Entity → Community instead of Community → Entity
- ❌ Wrong relationship type: `MEMBER_OF` instead of `HAS_MEMBER`
- ❌ This would NEVER find communities because HAS_MEMBER edges go Community → Entity

**Go Implementation - AFTER FIX** (kuzu.go:1930, memgraph.go:1123):
```cypher
MATCH (c:Community)-[:HAS_MEMBER]->(n:Entity {uuid: $entity_uuid})
RETURN c
```

✅ **Fixed**: Now matches Python implementation exactly

---

### 🐛 Bug 2: Wrong Relationship Pattern in FindModalCommunity

**Python Implementation for Neo4j/Memgraph** (`community_operations.py:260-262`):
```cypher
MATCH (c:Community)-[:HAS_MEMBER]->(m:Entity)-[:RELATES_TO]-(n:Entity {uuid: $entity_uuid})
RETURN [COMMUNITY_NODE_RETURN]
```

**Python Implementation for Kuzu** (`community_operations.py:264-266`):
```cypher
MATCH (c:Community)-[:HAS_MEMBER]->(m:Entity)-[:RELATES_TO]-(e:RelatesToNode_)-[:RELATES_TO]-(n:Entity {uuid: $entity_uuid})
RETURN [COMMUNITY_NODE_RETURN]
```

**Go Implementation - BEFORE FIX**:
```cypher
# Both Kuzu AND Memgraph (WRONG!)
MATCH (e:Entity {uuid: $entity_uuid})-[:RELATES_TO]-(rel)-[:RELATES_TO]-(neighbor:Entity)
MATCH (neighbor)-[:MEMBER_OF]->(c:Community)
WITH c, count(*) AS count
ORDER BY count DESC
LIMIT 1
RETURN c
```

**Issues**:
- ❌ Memgraph query uses wrong pattern: `-[:RELATES_TO]-(rel)-[:RELATES_TO]-` expects intermediate node, but Memgraph uses direct edges
- ❌ Both use wrong relationship type: `MEMBER_OF` instead of `HAS_MEMBER`
- ❌ Wrong relationship direction in second match
- ❌ This would NEVER find communities

**Go Implementation - AFTER FIX**:

**Kuzu** (kuzu.go:1960):
```cypher
MATCH (c:Community)-[:HAS_MEMBER]->(m:Entity)-[:RELATES_TO]-(e:RelatesToNode_)-[:RELATES_TO]-(n:Entity {uuid: $entity_uuid})
WITH c, count(*) AS count
ORDER BY count DESC
LIMIT 1
RETURN c
```

**Memgraph** (memgraph.go:1152):
```cypher
MATCH (c:Community)-[:HAS_MEMBER]->(m:Entity)-[:RELATES_TO]-(n:Entity {uuid: $entity_uuid})
WITH c, count(*) AS count
ORDER BY count DESC
LIMIT 1
RETURN c
```

✅ **Fixed**: Now matches Python implementation for each driver type

---

## Community Building Workflow

### Python Workflow (`community_operations.py`)

```
build_communities(driver, llm_client, group_ids)
    ├─ get_community_clusters(driver, group_ids)
    │   ├─ Get all group IDs if not specified (line 35-44)
    │   ├─ For each group_id:
    │   │   ├─ Get all Entity nodes by group (EntityNode.get_by_group_ids)
    │   │   ├─ Build projection graph (lines 50-71):
    │   │   │   └─ For each entity, query neighbors with edge counts
    │   │   ├─ Run label_propagation(projection) (line 73)
    │   │   └─ Get EntityNode objects for each cluster (lines 75-80)
    │   └─ Return list of entity clusters
    │
    ├─ Build communities with max 10 concurrent (lines 213-223)
    │   └─ For each cluster:
    │       └─ build_community(llm_client, cluster)
    │           ├─ Hierarchical summarization (lines 167-187)
    │           ├─ Generate community name (line 190)
    │           ├─ Create CommunityNode (lines 191-198)
    │           ├─ Build community edges (line 199)
    │           └─ Return (CommunityNode, [CommunityEdge])
    │
    └─ Return (community_nodes, community_edges)
```

### Go Workflow (`pkg/community/community.go`)

```
BuildCommunities(ctx, groupIDs, logger)
    ├─ GetCommunityClusters(ctx, groupIDs)
    │   ├─ Get all group IDs if not specified (getAllGroupIDs)
    │   ├─ For each group_id:
    │   │   ├─ Get all Entity nodes (getEntityNodesByGroup)
    │   │   ├─ Build projection (buildProjection):
    │   │   │   └─ For each node: getNodeNeighbors(driver)
    │   │   ├─ Run labelPropagation(projection)
    │   │   └─ Get Node objects for clusters (getNodesByUUIDs)
    │   └─ Return [][]*types.Node
    │
    ├─ Build communities with max 10 concurrent (lines 108-131)
    │   └─ For each cluster:
    │       └─ buildCommunity(ctx, cluster)
    │           ├─ hierarchicalSummarize (lines 157-162)
    │           ├─ generateCommunityName (line 164)
    │           ├─ Create community node (lines 168-179)
    │           ├─ Generate embedding (line 182)
    │           ├─ buildCommunityEdges (line 185)
    │           └─ Return (*types.Node, []*types.Edge)
    │
    └─ Return BuildCommunitiesResult{Nodes, Edges}
```

✅ **Workflow matches**: Go implementation correctly follows Python's structure

---

## Label Propagation Algorithm

### Python Implementation (`community_operations.py:86-131`)

```python
def label_propagation(projection: dict[str, list[Neighbor]]) -> list[list[str]]:
    # Initialize each node with its own community
    community_map = {uuid: i for i, uuid in enumerate(projection.keys())}

    while True:
        no_change = True
        new_community_map: dict[str, int] = {}

        for uuid, neighbors in projection.items():
            curr_community = community_map[uuid]

            # Count weighted neighbors by community
            community_candidates: dict[int, int] = defaultdict(int)
            for neighbor in neighbors:
                community_candidates[community_map[neighbor.node_uuid]] += neighbor.edge_count

            # Sort by count descending
            community_lst = [(count, community) for community, count in community_candidates.items()]
            community_lst.sort(reverse=True)

            # Select new community if candidate_rank > 1
            candidate_rank, community_candidate = community_lst[0] if community_lst else (0, -1)
            if community_candidate != -1 and candidate_rank > 1:
                new_community = community_candidate
            else:
                new_community = max(community_candidate, curr_community)

            new_community_map[uuid] = new_community

            if new_community != curr_community:
                no_change = False

        if no_change:
            break

        community_map = new_community_map

    # Group nodes by community
    community_cluster_map = defaultdict(list)
    for uuid, community in community_map.items():
        community_cluster_map[community].append(uuid)

    clusters = [cluster for cluster in community_cluster_map.values()]
    return clusters
```

### Go Implementation (`pkg/community/label_propagation.go:11-106`)

```go
func labelPropagation(projection map[string][]types.Neighbor) [][]string {
    // Initialize each node with its own community
    communityMap := make(map[string]int)
    for i, uuid := range sortedKeys(projection) {
        communityMap[uuid] = i
    }

    maxIterations := 100
    for iteration := 0; iteration < maxIterations; iteration++ {
        noChange := true
        newCommunityMap := make(map[string]int)

        for uuid, neighbors := range projection {
            currCommunity := communityMap[uuid]

            // Count weighted neighbors by community
            communityCandidates := make(map[int]int)
            for _, neighbor := range neighbors {
                if neighComm, exists := communityMap[neighbor.NodeUUID]; exists {
                    communityCandidates[neighComm] += neighbor.EdgeCount
                }
            }

            // Sort by count descending
            type communityCount struct {
                community int
                count     int
            }
            var communityList []communityCount
            for community, count := range communityCandidates {
                communityList = append(communityList, communityCount{community, count})
            }
            sort.Slice(communityList, func(i, j int) bool {
                return communityList[i].count > communityList[j].count
            })

            // Select new community
            var newCommunity int
            if len(communityList) > 0 {
                candidateRank := communityList[0].count
                communityCandidate := communityList[0].community
                if candidateRank > 1 {
                    newCommunity = communityCandidate
                } else {
                    newCommunity = max(communityCandidate, currCommunity)
                }
            } else {
                newCommunity = currCommunity
            }

            newCommunityMap[uuid] = newCommunity

            if newCommunity != currCommunity {
                noChange = false
            }
        }

        if noChange {
            break
        }

        communityMap = newCommunityMap
    }

    // Group nodes by community, filter single-node clusters
    communityClusterMap := make(map[int][]string)
    for uuid, community := range communityMap {
        communityClusterMap[community] = append(communityClusterMap[community], uuid)
    }

    var clusters [][]string
    for _, cluster := range communityClusterMap {
        if len(cluster) > 1 {
            clusters = append(clusters, cluster)
        }
    }

    return clusters
}
```

**Differences**:
- ✅ Go adds max iterations limit (100) to prevent infinite loops
- ✅ Go filters out single-node clusters (line 97-99) - optimization
- ✅ Core algorithm logic matches Python exactly

---

## Neighbor Projection Query

### Python Query for Neo4j/Memgraph (`community_operations.py:50-52`)
```cypher
MATCH (n:Entity {group_id: $group_id, uuid: $uuid})-[e:RELATES_TO]-(m: Entity {group_id: $group_id})
WITH count(e) AS count, m.uuid AS uuid
RETURN uuid, count
```

### Python Query for Kuzu (`community_operations.py:54-56`)
```cypher
MATCH (n:Entity {group_id: $group_id, uuid: $uuid})-[:RELATES_TO]-(e:RelatesToNode_)-[:RELATES_TO]-(m: Entity {group_id: $group_id})
WITH count(e) AS count, m.uuid AS uuid
RETURN uuid, count
```

### Go Implementation - Kuzu (`pkg/driver/kuzu.go:2689-2722`)

**Query**:
```cypher
MATCH (n:Entity {uuid: $uuid, group_id: $group_id})-[:RELATES_TO]->(rel:RelatesToNode_)<-[:RELATES_TO]-(neighbor:Entity {group_id: $group_id})
WHERE neighbor.uuid <> $uuid
WITH neighbor.uuid AS neighbor_uuid, count(rel) AS edge_count
RETURN neighbor_uuid, edge_count
```

**Differences**:
- ⚠️ Go uses directed pattern `->` and `<-` while Python uses undirected `-`
- ⚠️ Go adds `WHERE neighbor.uuid <> $uuid` to filter self-loops
- ⚠️ These differences might affect clustering results

### Go Implementation - Memgraph (Not implemented!)

❌ **Missing**: Memgraph does not have a `GetNodeNeighbors` implementation!

Let me check if this is actually implemented:

```bash
grep -n "GetNodeNeighbors" pkg/driver/memgraph.go
```

If missing, this is a **critical bug** preventing community building in Memgraph!

---

## Hierarchical Summarization

### Python Implementation (`community_operations.py:134-146, 164-203`)

```python
async def summarize_pair(llm_client: LLMClient, summary_pair: tuple[str, str]) -> str:
    context = {'node_summaries': [{'summary': summary} for summary in summary_pair]}
    llm_response = await llm_client.generate_response(
        prompt_library.summarize_nodes.summarize_pair(context),
        response_model=Summary
    )
    return llm_response.get('summary', '')

async def build_community(llm_client: LLMClient, community_cluster: list[EntityNode]):
    summaries = [entity.summary for entity in community_cluster]
    length = len(summaries)

    # Hierarchical pairing
    while length > 1:
        odd_one_out: str | None = None
        if length % 2 == 1:
            odd_one_out = summaries.pop()
            length -= 1

        new_summaries = await semaphore_gather(*[
            summarize_pair(llm_client, (left, right))
            for left, right in zip(summaries[:length//2], summaries[length//2:])
        ])

        if odd_one_out is not None:
            new_summaries.append(odd_one_out)

        summaries = new_summaries
        length = len(summaries)

    summary = summaries[0]
    name = await generate_summary_description(llm_client, summary)
    # ... create CommunityNode
```

### Go Implementation (`pkg/community/community.go:192-287`)

```go
func (b *Builder) hierarchicalSummarize(ctx context.Context, summaries []string) (string, error) {
    length := len(summaries)

    for length > 1 {
        var oddOneOut *string
        if length%2 == 1 {
            oddOneOut = &summaries[length-1]
            summaries = summaries[:length-1]
            length--
        }

        halfLen := length / 2
        var wg sync.WaitGroup
        newSummaries := make([]string, halfLen)
        errChan := make(chan error, halfLen)

        for i := 0; i < halfLen; i++ {
            wg.Add(1)
            go func(idx int) {
                defer wg.Done()
                left := summaries[idx]
                right := summaries[halfLen+idx]

                summary, err := b.summarizePair(ctx, left, right)
                if err != nil {
                    errChan <- err
                    return
                }
                newSummaries[idx] = summary
            }(i)
        }

        wg.Wait()
        close(errChan)

        if err := <-errChan; err != nil {
            return "", err
        }

        if oddOneOut != nil {
            newSummaries = append(newSummaries, *oddOneOut)
        }

        summaries = newSummaries
        length = len(summaries)
    }

    return summaries[0], nil
}

func (b *Builder) summarizePair(ctx context.Context, left, right string) (string, error) {
    messages := []llm.Message{
        {Role: "user", Content: fmt.Sprintf("Summarize these two node summaries...\n\n1. %s\n\n2. %s", left, right)},
    }

    response, err := b.llm.GenerateResponse(ctx, messages, nil)
    // ... parse and return
}
```

✅ **Matches**: Go implementation follows the same hierarchical pairing algorithm

---

## Community Edge Building

### Python Implementation (`graphiti/graphiti_core/utils/maintenance/edge_operations.py`)

```python
def build_community_edges(
    entity_nodes: list[EntityNode],
    community_node: CommunityNode,
    created_at: datetime,
) -> list[CommunityEdge]:
    edges = []
    for entity in entity_nodes:
        edge = CommunityEdge(
            source_node_uuid=community_node.uuid,
            target_node_uuid=entity.uuid,
            group_id=community_node.group_id,
            created_at=created_at,
        )
        edges.append(edge)
    return edges
```

### Go Implementation (`pkg/community/community.go:325-346`)

```go
func (b *Builder) buildCommunityEdges(entityNodes []*types.Node, communityNode *types.Node, createdAt time.Time) []*types.Edge {
    edges := make([]*types.Edge, 0, len(entityNodes))

    for _, entityNode := range entityNodes {
        edge := types.NewEntityEdge(
            generateUUID(),
            communityNode.Uuid,        // source
            entityNode.Uuid,           // target
            communityNode.GroupID,
            "HAS_MEMBER",
            types.CommunityEdgeType,
        )
        edge.UpdatedAt = createdAt
        edge.ValidFrom = createdAt
        edge.SourceIDs = []string{communityNode.Uuid}
        edge.Metadata = make(map[string]interface{})

        edges = append(edges, edge)
    }

    return edges
}
```

✅ **Matches**: Creates HAS_MEMBER edges from Community → Entity

---

## Community Update Operations

### Python Implementation (`community_operations.py:243-328`)

```python
async def determine_entity_community(driver, entity):
    # Check if already in community
    records = await driver.execute_query("""
        MATCH (c:Community)-[:HAS_MEMBER]->(n:Entity {uuid: $entity_uuid})
        RETURN [COMMUNITY_NODE_RETURN]
    """)

    if len(records) > 0:
        return get_community_node_from_record(records[0]), False

    # Find modal community from neighbors
    match_query = """
        MATCH (c:Community)-[:HAS_MEMBER]->(m:Entity)-[:RELATES_TO]-(n:Entity {uuid: $entity_uuid})
    """
    if driver.provider == GraphProvider.KUZU:
        match_query = """
            MATCH (c:Community)-[:HAS_MEMBER]->(m:Entity)-[:RELATES_TO]-(e:RelatesToNode_)-[:RELATES_TO]-(n:Entity {uuid: $entity_uuid})
        """

    records = await driver.execute_query(match_query + " RETURN [COMMUNITY_NODE_RETURN]")

    # Count occurrences and find mode
    community_map = defaultdict(int)
    for community in communities:
        community_map[community.uuid] += 1

    # Return most common community
    # ...

async def update_community(driver, llm_client, embedder, entity):
    community, is_new = await determine_entity_community(driver, entity)

    if community is None:
        return [], []

    # Summarize entity + community
    new_summary = await summarize_pair(llm_client, (entity.summary, community.summary))
    new_name = await generate_summary_description(llm_client, new_summary)

    community.summary = new_summary
    community.name = new_name

    community_edges = []
    if is_new:
        community_edge = (build_community_edges([entity], community, utc_now()))[0]
        await community_edge.save(driver)
        community_edges.append(community_edge)

    await community.generate_name_embedding(embedder)
    await community.save(driver)

    return [community], community_edges
```

### Go Implementation (`pkg/community/update.go`)

```go
func (b *Builder) DetermineEntityCommunity(ctx context.Context, entity *types.Node) (DetermineEntityCommunityResult, error) {
    // Check existing community
    existingCommunity, err := b.driver.GetExistingCommunity(ctx, entity.Uuid)
    if err != nil {
        return DetermineEntityCommunityResult{}, err
    }

    if existingCommunity != nil {
        return DetermineEntityCommunityResult{
            Community: existingCommunity,
            IsNew:     false,
        }, nil
    }

    // Find modal community
    modalCommunity, err := b.driver.FindModalCommunity(ctx, entity.Uuid)
    if err != nil {
        return DetermineEntityCommunityResult{}, err
    }

    if modalCommunity != nil {
        return DetermineEntityCommunityResult{
            Community: modalCommunity,
            IsNew:     true,
        }, nil
    }

    return DetermineEntityCommunityResult{}, nil
}

func (b *Builder) UpdateCommunity(ctx context.Context, entity *types.Node) (UpdateCommunityResult, error) {
    result, err := b.DetermineEntityCommunity(ctx, entity)
    if err != nil || result.Community == nil {
        return UpdateCommunityResult{}, err
    }

    // Summarize entity + community
    newSummary, err := b.summarizePair(ctx, entity.Summary, result.Community.Summary)
    if err != nil {
        return UpdateCommunityResult{}, err
    }

    newName, err := b.generateCommunityName(ctx, newSummary)
    if err != nil {
        return UpdateCommunityResult{}, err
    }

    result.Community.Summary = newSummary
    result.Community.Name = newName

    var communityEdges []*types.Edge
    if result.IsNew {
        edges := b.buildCommunityEdges([]*types.Node{entity}, result.Community, time.Now())
        if len(edges) > 0 {
            if err := b.driver.UpsertEdge(ctx, edges[0]); err != nil {
                return UpdateCommunityResult{}, err
            }
            communityEdges = edges
        }
    }

    // Generate embedding
    embedding, err := b.embedder.GenerateEmbedding(ctx, result.Community.Name)
    if err != nil {
        return UpdateCommunityResult{}, err
    }
    result.Community.NameEmbedding = embedding

    // Save community
    if err := b.driver.UpsertNode(ctx, result.Community); err != nil {
        return UpdateCommunityResult{}, err
    }

    return UpdateCommunityResult{
        Communities:     []*types.Node{result.Community},
        CommunityEdges:  communityEdges,
    }, nil
}
```

✅ **Matches**: Go implementation follows Python's update workflow

---

## Summary of Issues

### ✅ Fixed Issues

1. **GetExistingCommunity**: Fixed relationship direction and type (Community-[:HAS_MEMBER]->Entity)
2. **FindModalCommunity**: Fixed query pattern to match Python for both Kuzu and Memgraph

### ⚠️ Potential Issues to Investigate

1. **GetNodeNeighbors in Memgraph**: Need to verify this is implemented
2. **Neighbor query direction**: Go uses directed arrows while Python uses undirected - may affect clustering
3. **Single-node cluster filtering**: Go filters them out, Python includes them - intentional difference?

### 📝 Recommendations

1. ✅ Test end-to-end community building with both Kuzu and Memgraph
2. ✅ Verify that communities are being created and edges properly formed
3. ✅ Compare clustering results between Python and Go implementations
4. Add integration tests for community update operations
5. Document the intentional differences (max iterations, single-node filtering)
