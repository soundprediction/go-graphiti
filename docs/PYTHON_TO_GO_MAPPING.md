# Python to Go Method Mapping

This document tracks the mapping between the original Python Graphiti methods and their corresponding Go implementations in go-graphiti.

## Table of Contents
- [Core Graphiti Class](#core-graphiti-class)
- [Core Graph Queries](#core-graph-queries)
- [Search Functionality](#search-functionality)
- [Driver Interface](#driver-interface)
- [Node and Edge Types](#node-and-edge-types)
- [LLM Client Interface](#llm-client-interface)
- [Embedder Client Interface](#embedder-client-interface)
- [Cross Encoder Interface](#cross-encoder-interface)
- [Search Filters](#search-filters)
- [Prompts and Models](#prompts-and-models)
- [Utilities and Helpers](#utilities-and-helpers)
- [Telemetry](#telemetry)

## Core Graphiti Class

### graphiti.py - Main Graphiti Class

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|--------|
| `Graphiti.__init__()` | `NewClient()` | `graphiti.go` | ✅ Implemented | Go uses functional construction pattern |
| `Graphiti.close()` | `Client.Close()` | `graphiti.go` | ✅ Implemented | |
| `Graphiti.build_indices_and_constraints()` | `Client.CreateIndices()` | `graphiti.go` | ✅ Implemented | |
| `Graphiti.retrieve_episodes()` | `Client.GetEpisodes()` | `graphiti.go` | ✅ Implemented | |
| `Graphiti.add_episode()` | `Client.Add()` | `graphiti.go` | ✅ Implemented | Go method accepts multiple episodes |
| `Graphiti.add_episode_bulk()` | `Client.Add()` | `graphiti.go` | ✅ Implemented | Same as single episode in Go |
| `Graphiti.build_communities()` | `Builder.BuildCommunities()` | `pkg/community/community.go` | ✅ Implemented | Community building with label propagation |
| `Graphiti.search()` | `Client.Search()` | `graphiti.go` | ✅ Implemented | |
| `Graphiti._search()` | `Client.Search()` internal | `graphiti.go` | ✅ Implemented | Merged into main Search method |
| `Graphiti.search_()` | `searcher.HybridSearch()` | `pkg/search/search.go` | ✅ Implemented | Direct searcher access |
| `Graphiti.get_nodes_and_edges_by_episode()` | `Client.GetNode()` / `Client.GetEdge()` | `graphiti.go` | ⚠️ Partial | No bulk episode-based retrieval |
| `Graphiti.add_triplet()` | `Client.addTriplet()` | `graphiti.go` | ❌ Missing | Direct triplet addition not implemented |
| `Graphiti.remove_episode()` | `Client.removeEpisode()` | `graphiti.go` | ❌ Missing | Episode removal not implemented |

### Result Types

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| `AddEpisodeResults` | `AddEpisodeResults` | `pkg/types/types.go` | ✅ Implemented |
| `AddBulkEpisodeResults` | `AddBulkEpisodeResults` | `pkg/types/types.go` | ✅ Implemented |
| `AddTripletResults` | `AddTripletResults` | `pkg/types/types.go` | ✅ Implemented |

### Additional Go Result Types

| Go Type | Description | File Location |
|---------|-------------|---------------|
| `EpisodeProcessingResult` | Internal episode processing result | `pkg/types/types.go` |
| `BulkEpisodeResults` | Bulk episode processing statistics | `pkg/types/types.go` |
| `TripletResults` | Enhanced triplet operation result | `pkg/types/types.go` |

## Core Graph Queries

### graph_queries.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| `get_range_indices(provider)` | `GetRangeIndices(provider GraphProvider)` | `pkg/driver/graph_queries.go` | ✅ Implemented |
| `get_fulltext_indices(provider)` | `GetFulltextIndices(provider GraphProvider)` | `pkg/driver/graph_queries.go` | ✅ Implemented |
| `get_nodes_query(name, query, limit, provider)` | `GetNodesQuery(indexName, query string, limit int, provider GraphProvider)` | `pkg/driver/graph_queries.go` | ✅ Implemented |
| `get_relationships_query(name, limit, provider)` | `GetRelationshipsQuery(indexName string, limit int, provider GraphProvider)` | `pkg/driver/graph_queries.go` | ✅ Implemented |
| `get_vector_cosine_func_query(vec1, vec2, provider)` | `GetVectorCosineFuncQuery(vec1, vec2 string, provider GraphProvider)` | `pkg/driver/graph_queries.go` | ✅ Implemented |
| `GraphProvider` enum | `GraphProvider` type | `pkg/driver/graph_queries.go` | ✅ Implemented |
| `NEO4J_TO_FALKORDB_MAPPING` | `neo4jToFalkorDBMapping` | `pkg/driver/graph_queries.go` | ✅ Implemented |
| `INDEX_TO_LABEL_KUZU_MAPPING` | `indexToLabelKuzuMapping` | `pkg/driver/graph_queries.go` | ✅ Implemented |

### Additional Go Utilities (not in Python)

| Go Method | Description | File Location |
|-----------|-------------|---------------|
| `NewQueryBuilder(provider GraphProvider)` | Creates database-agnostic query builder | `pkg/driver/graph_queries.go` |
| `QueryBuilder.BuildFulltextNodeQuery()` | Builds fulltext node search queries | `pkg/driver/graph_queries.go` |
| `QueryBuilder.BuildFulltextRelationshipQuery()` | Builds fulltext relationship queries | `pkg/driver/graph_queries.go` |
| `QueryBuilder.BuildCosineSimilarityQuery()` | Builds cosine similarity queries | `pkg/driver/graph_queries.go` |
| `EscapeQueryString(query string)` | Escapes special characters in queries | `pkg/driver/graph_queries.go` |
| `BuildParameterizedQuery()` | Builds parameterized queries | `pkg/driver/graph_queries.go` |

## Search Functionality

### search/search.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| `Searcher` class | `Searcher` struct | `pkg/search/search.go` | ✅ Implemented |
| `HybridSearch()` | `HybridSearch()` | `pkg/search/search.go` | ✅ Implemented |
| Search methods (cosine_similarity, bm25, bfs) | `SearchMethod` constants | `pkg/search/search.go` | ✅ Implemented |
| Reranker types | `RerankerType` constants | `pkg/search/search.go` | ✅ Implemented |

### search/search_config.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| `SearchConfig` class | `SearchConfig` struct | `pkg/search/search.go` | ✅ Implemented |
| `NodeSearchConfig` | `NodeSearchConfig` struct | `pkg/search/search.go` | ✅ Implemented |
| `EdgeSearchConfig` | `EdgeSearchConfig` struct | `pkg/search/search.go` | ✅ Implemented |
| `EpisodeSearchConfig` | `EpisodeSearchConfig` struct | `pkg/search/search.go` | ✅ Implemented |
| `CommunitySearchConfig` | `CommunitySearchConfig` struct | `pkg/search/search.go` | ✅ Implemented |
| `SearchResults` | `HybridSearchResult` struct | `pkg/search/search.go` | ✅ Implemented |

### search/search_config_recipes.py

| Python Configuration | Go Configuration | File Location | Status |
|----------------------|------------------|---------------|--------|
| `COMBINED_HYBRID_SEARCH_RRF` | `CombinedHybridSearchRRF` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `COMBINED_HYBRID_SEARCH_MMR` | `CombinedHybridSearchMMR` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `COMBINED_HYBRID_SEARCH_CROSS_ENCODER` | `CombinedHybridSearchCrossEncoder` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `EDGE_HYBRID_SEARCH_RRF` | `EdgeHybridSearchRRF` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `EDGE_HYBRID_SEARCH_MMR` | `EdgeHybridSearchMMR` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `EDGE_HYBRID_SEARCH_NODE_DISTANCE` | `EdgeHybridSearchNodeDistance` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `EDGE_HYBRID_SEARCH_EPISODE_MENTIONS` | `EdgeHybridSearchEpisodeMentions` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `EDGE_HYBRID_SEARCH_CROSS_ENCODER` | `EdgeHybridSearchCrossEncoder` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `NODE_HYBRID_SEARCH_RRF` | `NodeHybridSearchRRF` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `NODE_HYBRID_SEARCH_MMR` | `NodeHybridSearchMMR` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `NODE_HYBRID_SEARCH_NODE_DISTANCE` | `NodeHybridSearchNodeDistance` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `NODE_HYBRID_SEARCH_EPISODE_MENTIONS` | `NodeHybridSearchEpisodeMentions` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `NODE_HYBRID_SEARCH_CROSS_ENCODER` | `NodeHybridSearchCrossEncoder` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `COMMUNITY_HYBRID_SEARCH_RRF` | `CommunityHybridSearchRRF` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `COMMUNITY_HYBRID_SEARCH_MMR` | `CommunityHybridSearchMMR` | `pkg/search/config_recipes.go` | ✅ Implemented |
| `COMMUNITY_HYBRID_SEARCH_CROSS_ENCODER` | `CommunityHybridSearchCrossEncoder` | `pkg/search/config_recipes.go` | ✅ Implemented |

### search/search_filters.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| `SearchFilters` class | `SearchFilters` struct | `pkg/search/search.go` | ✅ Implemented |
| `ComparisonOperator` enum | `ComparisonOperator` type | `pkg/search/filters.go` | ✅ Implemented |
| `DateFilter` class | `DateFilter` struct | `pkg/search/filters.go` | ✅ Implemented |
| `node_search_filter_query_constructor()` | `NodeSearchFilterQueryConstructor()` | `pkg/search/filters.go` | ✅ Implemented |
| `edge_search_filter_query_constructor()` | `EdgeSearchFilterQueryConstructor()` | `pkg/search/filters.go` | ✅ Implemented |
| `date_filter_query_constructor()` | `constructDateFilterQuery()` | `pkg/search/filters.go` | ✅ Implemented |

### search/search_helpers.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| `format_edge_date_range(edge)` | `FormatEdgeDateRange(edge *types.Edge)` | `pkg/search/helpers.go` | ✅ Implemented |
| `search_results_to_context_string()` | `SearchResultsToContextString()` | `pkg/search/helpers.go` | ✅ Implemented |

### search/search_utils.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| Various search utility functions | `SearchUtilities` struct methods | `pkg/search/search_utils.go` | ✅ Implemented |
| Cosine similarity calculation | `CalculateCosineSimilarity()` | `pkg/search/search_utils.go` | ✅ Implemented |
| RRF reranking | `RRF()` | `pkg/search/rerankers.go` | ✅ Implemented |
| MMR reranking | `MMR()` | `pkg/search/rerankers.go` | ✅ Implemented |

## Driver Interface

### driver/driver.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| `GraphDriver` interface | `GraphDriver` interface | `pkg/driver/driver.go` | ✅ Implemented |
| Database operations (GetNode, UpsertNode, etc.) | Same method names | `pkg/driver/driver.go` | ✅ Implemented |

### driver/neo4j.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| `Neo4jDriver` class | `Neo4jDriver` struct | `pkg/driver/neo4j.go` | ✅ Implemented |
| All GraphDriver interface methods | Same method names | `pkg/driver/neo4j.go` | ✅ Implemented |

### driver/kuzu.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| `KuzuDriver` class | `KuzuDriver` struct | `pkg/driver/kuzu.go` | ✅ Implemented |
| All GraphDriver interface methods | Same method names | `pkg/driver/kuzu.go` | ✅ Implemented |
| Schema setup | `setupSchema()` method | `pkg/driver/kuzu.go` | ✅ Implemented |

## Node and Edge Types

### nodes.py / edges.py

| Python Type | Go Type | File Location | Status | Notes |
|-------------|---------|---------------|--------|--------|
| `Node` base class | `types.Node` struct | `pkg/types/types.go` | ✅ Implemented | Single struct for all node types |
| `EntityNode` | `types.Node` with `Type: EntityNodeType` | `pkg/types/types.go` | ✅ Implemented | |
| `EpisodicNode` | `types.Node` with `Type: EpisodicNodeType` | `pkg/types/types.go` | ✅ Implemented | |
| `CommunityNode` | `types.Node` with `Type: CommunityNodeType` | `pkg/types/types.go` | ✅ Implemented | |
| `Edge` base class | `types.Edge` struct | `pkg/types/types.go` | ✅ Implemented | Single struct for all edge types |
| `EntityEdge` | `types.Edge` with `Type: EntityEdgeType` | `pkg/types/types.go` | ✅ Implemented | |
| `EpisodicEdge` | `types.Edge` with `Type: EpisodicEdgeType` | `pkg/types/types.go` | ✅ Implemented | |
| `CommunityEdge` | `types.Edge` with `Type: CommunityEdgeType` | `pkg/types/types.go` | ✅ Implemented | |

### Node and Edge Functions

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| `create_entity_node_embeddings()` | `EmbedNodeContent()` | `pkg/embedder/` | ⚠️ Partial |
| `create_entity_edge_embeddings()` | `EmbedEdgeContent()` | `pkg/embedder/` | ⚠️ Partial |
| `get_entity_node_from_record()` | `NodeFromDBRecord()` | `pkg/driver/` | ✅ Implemented |
| `get_entity_edge_from_record()` | `EdgeFromDBRecord()` | `pkg/driver/` | ✅ Implemented |

## LLM Client Interface

### llm_client/client.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|--------|
| `LLMClient` abstract class | `llm.Client` interface | `pkg/llm/client.go` | ✅ Implemented | |
| `LLMClient.generate()` | `Client.Chat()` | `pkg/llm/client.go` | ✅ Implemented | |
| `LLMClient.generate_batch()` | `Client.ChatBatch()` | `pkg/llm/` | ❌ Missing | Batch operations not implemented |
| `LLMClient.generate_with_schema()` | `Client.ChatWithStructuredOutput()` | `pkg/llm/client.go` | ✅ Implemented | |

### LLM Client Implementations

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `OpenAIClient` | `openai.Client` | `pkg/llm/openai/` | ✅ Implemented |
| `AnthropicClient` | `anthropic.Client` | `pkg/llm/anthropic/` | ❌ Missing |
| `GeminiClient` | `gemini.Client` | `pkg/llm/gemini/` | ❌ Missing |
| `GroqClient` | `groq.Client` | `pkg/llm/groq/` | ❌ Missing |
| `AzureOpenAIClient` | `azure.Client` | `pkg/llm/azure/` | ❌ Missing |

### LLM Configuration

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| `LLMConfig` | `llm.Config` | `pkg/llm/config.go` | ✅ Implemented |
| `ModelSize` enum | `ModelSize` constants | `pkg/llm/config.go` | ✅ Implemented |

## Embedder Client Interface

### embedder/client.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|--------|
| `EmbedderClient` abstract class | `embedder.Client` interface | `pkg/embedder/client.go` | ✅ Implemented | |
| `EmbedderClient.create()` | `Client.EmbedSingle()` | `pkg/embedder/client.go` | ✅ Implemented | |
| `EmbedderClient.create_batch()` | `Client.Embed()` | `pkg/embedder/client.go` | ✅ Implemented | |

### Embedder Implementations

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `OpenAIEmbedder` | `openai.EmbedderClient` | `pkg/embedder/openai/` | ✅ Implemented |
| `VoyageEmbedder` | `voyage.Client` | `pkg/embedder/voyage/` | ❌ Missing |
| `GeminiEmbedder` | `gemini.Client` | `pkg/embedder/gemini/` | ❌ Missing |
| `AzureOpenAIEmbedder` | `azure.Client` | `pkg/embedder/azure/` | ❌ Missing |

### Embedder Configuration

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| `EmbedderConfig` | `embedder.Config` | `pkg/embedder/client.go` | ✅ Implemented |
| `EMBEDDING_DIM` constant | `DefaultDimensions` | `pkg/embedder/client.go` | ✅ Implemented |

## Cross Encoder Interface

### cross_encoder/client.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|--------|
| `CrossEncoderClient` abstract class | `crossencoder.Client` interface | `pkg/crossencoder/` | ❌ Missing | Cross encoder not implemented |
| `CrossEncoderClient.rerank()` | `Client.Rerank()` | `pkg/crossencoder/` | ❌ Missing | |

### Cross Encoder Implementations

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `OpenAIRerankerClient` | N/A | N/A | ❌ Missing |
| `BGERerankerClient` | N/A | N/A | ❌ Missing |
| `GeminiRerankerClient` | N/A | N/A | ❌ Missing |

## Community Operations

### utils/maintenance/community_operations.py

| Python Function | Go Method | File Location | Status | Notes |
|-----------------|-----------|---------------|--------|--------|
| `get_community_clusters()` | `Builder.GetCommunityClusters()` | `pkg/community/community.go` | ✅ Implemented | |
| `label_propagation()` | `Builder.labelPropagation()` | `pkg/community/label_propagation.go` | ✅ Implemented | |
| `build_community()` | `Builder.buildCommunity()` | `pkg/community/community.go` | ✅ Implemented | |
| `build_communities()` | `Builder.BuildCommunities()` | `pkg/community/community.go` | ✅ Implemented | |
| `remove_communities()` | `Builder.RemoveCommunities()` | `pkg/community/community.go` | ✅ Implemented | |
| `determine_entity_community()` | `Builder.DetermineEntityCommunity()` | `pkg/community/update.go` | ✅ Implemented | |
| `update_community()` | `Builder.UpdateCommunity()` | `pkg/community/update.go` | ✅ Implemented | |
| `summarize_pair()` | `Builder.summarizePair()` | `pkg/community/community.go` | ✅ Implemented | |
| `generate_summary_description()` | `Builder.generateCommunityName()` | `pkg/community/community.go` | ✅ Implemented | |

### Community Types and Models

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| `Neighbor` class | `Neighbor` struct | `pkg/community/community.go` | ✅ Implemented |
| `BuildCommunitiesResult` | `BuildCommunitiesResult` struct | `pkg/community/community.go` | ✅ Implemented |
| `DetermineEntityCommunityResult` | `DetermineEntityCommunityResult` struct | `pkg/community/update.go` | ✅ Implemented |
| `UpdateCommunityResult` | `UpdateCommunityResult` struct | `pkg/community/update.go` | ✅ Implemented |

### Additional Go Community Functions

| Go Method | Description | File Location |
|-----------|-------------|---------------|
| `NewBuilder()` | Creates new community builder | `pkg/community/community.go` |
| `buildProjection()` | Builds neighbor projection for clustering | `pkg/community/label_propagation.go` |
| `getNodeNeighbors()` | Gets node neighbors with edge counts | `pkg/community/label_propagation.go` |
| `getAllGroupIDs()` | Gets all distinct group IDs | `pkg/community/label_propagation.go` |
| `getEntityNodesByGroup()` | Gets entity nodes by group | `pkg/community/label_propagation.go` |
| `hierarchicalSummarize()` | Performs hierarchical summarization | `pkg/community/community.go` |
| `generateCommunityEmbedding()` | Generates embeddings for communities | `pkg/community/community.go` |
| `buildCommunityEdges()` | Creates HAS_MEMBER edges | `pkg/community/community.go` |

## Prompts and Models

### prompts/models.py

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| `Message` class | `Message` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `PromptFunction` type | `PromptFunction` type | `pkg/prompts/types.go` | ✅ Implemented |
| `ExtractedEntity` | `ExtractedEntity` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `ExtractedEntities` | `ExtractedEntities` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `EntityClassificationTriple` | `EntityClassificationTriple` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `EntitySummary` | `EntitySummary` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `Edge` | `Edge` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `ExtractedEdges` | `ExtractedEdges` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `NodeDuplicate` | `NodeDuplicate` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `EdgeDuplicate` | `EdgeDuplicate` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `InvalidatedEdges` | `InvalidatedEdges` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `EdgeDates` | `EdgeDates` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `Summary` | `Summary` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `SummaryDescription` | `SummaryDescription` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `QueryExpansion` | `QueryExpansion` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `QAResponse` | `QAResponse` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `EvalResponse` | `EvalResponse` struct | `pkg/prompts/models.go` | ✅ Implemented |

### Prompt Templates

| Python Module | Go Implementation | File Location | Status | Notes |
|---------------|-------------------|---------------|--------|--------|
| `prompts/extract_nodes.py` | `ExtractNodesPrompt` interface | `pkg/prompts/extract_nodes.go` | ✅ Implemented | All 7 functions implemented |
| `prompts/extract_edges.py` | `ExtractEdgesPrompt` interface | `pkg/prompts/extract_edges.go` | ✅ Implemented | All 3 functions implemented |
| `prompts/dedupe_nodes.py` | `DedupeNodesPrompt` interface | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented | All 3 functions implemented |
| `prompts/dedupe_edges.py` | `DedupeEdgesPrompt` interface | `pkg/prompts/dedupe_edges.go` | ✅ Implemented | All 3 functions implemented |
| `prompts/summarize_nodes.py` | `SummarizeNodesPrompt` interface | `pkg/prompts/summarize_nodes.go` | ✅ Implemented | All 3 functions implemented |
| `prompts/invalidate_edges.py` | `InvalidateEdgesPrompt` interface | `pkg/prompts/invalidate_edges.go` | ✅ Implemented | Both v1 and v2 functions |
| `prompts/extract_edge_dates.py` | `ExtractEdgeDatesPrompt` interface | `pkg/prompts/extract_edge_dates.go` | ✅ Implemented | v1 function implemented |
| `prompts/eval.py` | `EvalPrompt` interface | `pkg/prompts/eval.go` | ✅ Implemented | All 4 functions implemented |

### Extract Nodes Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| `extract_message()` | `ExtractMessage()` | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| `extract_json()` | `ExtractJSON()` | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| `extract_text()` | `ExtractText()` | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| `reflexion()` | `Reflexion()` | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| `classify_nodes()` | `ClassifyNodes()` | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| `extract_attributes()` | `ExtractAttributes()` | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| `extract_summary()` | `ExtractSummary()` | `pkg/prompts/extract_nodes.go` | ✅ Implemented |

### Extract Edges Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| `edge()` | `Edge()` | `pkg/prompts/extract_edges.go` | ✅ Implemented |
| `reflexion()` | `Reflexion()` | `pkg/prompts/extract_edges.go` | ✅ Implemented |
| `extract_attributes()` | `ExtractAttributes()` | `pkg/prompts/extract_edges.go` | ✅ Implemented |

### Dedupe Nodes Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| `node()` | `Node()` | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented |
| `node_list()` | `NodeList()` | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented |
| `nodes()` | `Nodes()` | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented |

### Dedupe Edges Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| `edge()` | `Edge()` | `pkg/prompts/dedupe_edges.go` | ✅ Implemented |
| `edge_list()` | `EdgeList()` | `pkg/prompts/dedupe_edges.go` | ✅ Implemented |
| `resolve_edge()` | `ResolveEdge()` | `pkg/prompts/dedupe_edges.go` | ✅ Implemented |

### Summarize Nodes Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| `summarize_pair()` | `SummarizePair()` | `pkg/prompts/summarize_nodes.go` | ✅ Implemented |
| `summarize_context()` | `SummarizeContext()` | `pkg/prompts/summarize_nodes.go` | ✅ Implemented |
| `summary_description()` | `SummaryDescription()` | `pkg/prompts/summarize_nodes.go` | ✅ Implemented |

### Invalidate Edges Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| `v1()` | `V1()` | `pkg/prompts/invalidate_edges.go` | ✅ Implemented |
| `v2()` | `V2()` | `pkg/prompts/invalidate_edges.go` | ✅ Implemented |

### Extract Edge Dates Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| `v1()` | `V1()` | `pkg/prompts/extract_edge_dates.go` | ✅ Implemented |

### Eval Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| `qa_prompt()` | `QAPrompt()` | `pkg/prompts/eval.go` | ✅ Implemented |
| `eval_prompt()` | `EvalPrompt()` | `pkg/prompts/eval.go` | ✅ Implemented |
| `query_expansion()` | `QueryExpansion()` | `pkg/prompts/eval.go` | ✅ Implemented |
| `eval_add_episode_results()` | `EvalAddEpisodeResults()` | `pkg/prompts/eval.go` | ✅ Implemented |

### Prompt Library

| Python Component | Go Component | File Location | Status |
|------------------|--------------|---------------|--------|
| `PromptLibrary` interface | `Library` interface | `pkg/prompts/library.go` | ✅ Implemented |
| `PromptLibraryImpl` | `LibraryImpl` struct | `pkg/prompts/library.go` | ✅ Implemented |
| `prompt_library` instance | `NewLibrary()` function | `pkg/prompts/library.go` | ✅ Implemented |

### Prompt Helpers

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| `to_prompt_json()` | `ToPromptJSON()` | `pkg/prompts/types.go` | ✅ Implemented |
| `DO_NOT_ESCAPE_UNICODE` | `DoNotEscapeUnicode` const | `pkg/prompts/models.go` | ✅ Implemented |

## Utilities and Helpers

### helpers.py (graphiti_core/)

| Python Function | Go Function | File Location | Status | Notes |
|-----------------|-------------|---------------|--------|--------|
| `get_default_group_id()` | `GetDefaultGroupID()` | `pkg/utils/helpers.go` | ✅ Implemented | |
| `semaphore_gather()` | `SemaphoreGather()` | `pkg/utils/concurrent.go` | ✅ Implemented | |
| `validate_excluded_entity_types()` | `ValidateExcludedEntityTypes()` | `pkg/utils/helpers.go` | ✅ Implemented | |
| `validate_group_id()` | `ValidateGroupID()` | `pkg/utils/helpers.go` | ✅ Implemented | |
| `lucene_sanitize()` | `LuceneSanitize()` | `pkg/utils/helpers.go` | ✅ Implemented | |
| `normalize_l2()` | `NormalizeL2()` / `NormalizeL2Float32()` | `pkg/utils/helpers.go` | ✅ Implemented | |
| `parse_db_date()` | `ParseDBDate()` | `pkg/utils/helpers.go` | ✅ Implemented | |

### utils/bulk_utils.py

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| `add_nodes_and_edges_bulk()` | `Client.Add()` | `graphiti.go` | ⚠️ Partial |
| `dedupe_edges_bulk()` | Helper functions | `pkg/utils/bulk.go` | ✅ Implemented |
| `dedupe_nodes_bulk()` | Helper functions | `pkg/utils/bulk.go` | ✅ Implemented |
| `extract_nodes_and_edges_bulk()` | Embedded in `Client.Add()` | `graphiti.go` | ⚠️ Partial |
| `resolve_edge_pointers()` | `ResolveEdgePointers()` | `pkg/utils/bulk.go` | ✅ Implemented |
| `retrieve_previous_episodes_bulk()` | `GetEpisodes()` | `graphiti.go` | ⚠️ Partial |
| `compress_uuid_map()` | `CompressUUIDMap()` | `pkg/utils/bulk.go` | ✅ Implemented |
| `UnionFind` class | `UnionFind` struct | `pkg/utils/bulk.go` | ✅ Implemented |

### utils/datetime_utils.py

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| `utc_now()` | `UTCNow()` | `pkg/utils/datetime.go` | ✅ Implemented |
| `ensure_utc()` | `EnsureUTC()` | `pkg/utils/datetime.go` | ✅ Implemented |
| `convert_datetimes_to_strings()` | `ConvertDatetimesToStrings()` | `pkg/utils/datetime.go` | ✅ Implemented |

### utils/ontology_utils/entity_types_utils.py

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| `validate_entity_types()` | `ValidateEntityTypes()` | `pkg/utils/validation.go` | ✅ Implemented |

### Additional Go Utility Functions

| Go Function | Description | File Location |
|-------------|-------------|---------------|
| `GetUseParallelRuntime()` | Gets parallel runtime setting from env | `pkg/utils/helpers.go` |
| `GetSemaphoreLimit()` | Gets semaphore limit from env | `pkg/utils/helpers.go` |
| `GetMaxReflexionIterations()` | Gets max reflexion iterations from env | `pkg/utils/helpers.go` |
| `NewConcurrentExecutor()` | Creates concurrent executor with semaphore | `pkg/utils/concurrent.go` |
| `ExecuteWithResults()` | Concurrent execution with results | `pkg/utils/concurrent.go` |
| `NewWorkerPool()` | Creates worker pool for processing | `pkg/utils/concurrent.go` |
| `NewBatchProcessor()` | Creates batch processor | `pkg/utils/bulk.go` |
| `HasWordOverlap()` | Checks word overlap for deduplication | `pkg/utils/bulk.go` |
| `CalculateCosineSimilarity()` | Computes cosine similarity | `pkg/utils/bulk.go` |
| `FindSimilarNodes()` / `FindSimilarEdges()` | Find duplicate candidates | `pkg/utils/bulk.go` |
| `ChunkSlice()` | Splits slices into chunks | `pkg/utils/bulk.go` |
| `RemoveDuplicateStrings()` | Removes duplicates from string slice | `pkg/utils/bulk.go` |
| `ValidateUUID()` | Validates UUID format | `pkg/utils/validation.go` |
| `ValidateRequired()` | Validates required fields | `pkg/utils/validation.go` |
| `ValidateRange()` | Validates numeric ranges | `pkg/utils/validation.go` |
| `ValidateEmbeddingDimensions()` | Validates embedding consistency | `pkg/utils/validation.go` |
| `FormatTimeForDB()` / `ParseTimeFromDB()` | Database time formatting | `pkg/utils/datetime.go` |
| `TimeToMilliseconds()` / `MillisecondsToTime()` | Time conversion utilities | `pkg/utils/datetime.go` |

### utils/maintenance/

| Python Module | Go Implementation | File Location | Status |
|---------------|-------------------|---------------|--------|
| `community_operations.py` | `pkg/community/` | Multiple files | ✅ Implemented |
| `edge_operations.py` | Embedded in main client | `graphiti.go` | ⚠️ Partial |
| `node_operations.py` | Embedded in main client | `graphiti.go` | ⚠️ Partial |
| `temporal_operations.py` | `pkg/temporal/` | Not implemented | ❌ Missing |
| `graph_data_operations.py` | Various locations | Multiple files | ⚠️ Partial |

### Additional Go Helper Functions

| Go Method | Description | File Location |
|-----------|-------------|---------------|
| `GetDefaultSearchConfig()` | Returns default search configuration | `pkg/search/helpers.go` |
| `GetSearchConfigByName()` | Gets predefined config by name | `pkg/search/helpers.go` |
| `ListAvailableSearchConfigs()` | Lists all available configurations | `pkg/search/helpers.go` |

## Telemetry

### telemetry/telemetry.py

| Python Function | Go Function | File Location | Status | Notes |
|-----------------|-------------|---------------|--------|--------|
| `capture_event()` | `CaptureEvent()` | `pkg/telemetry/` | ❌ Missing | Telemetry not implemented |
| `TelemetryEvent` class | `TelemetryEvent` struct | `pkg/telemetry/` | ❌ Missing | |

## Architecture Differences

### Python vs Go Design Patterns

| Aspect | Python Approach | Go Approach |
|--------|----------------|-------------|
| Configuration | Class-based with Pydantic models | Struct-based with JSON tags |
| Error Handling | Exceptions | Explicit error returns |
| Interface Definition | Duck typing + Abstract base classes | Explicit interfaces |
| Database Abstraction | Runtime polymorphism | Compile-time interfaces |
| Search Configuration | Enum-based method selection | Constant-based method selection |

## Migration Notes

1. **Error Handling**: Go methods return explicit errors instead of raising exceptions
2. **Configuration**: Go uses struct pointers and nil checks instead of Python's Optional types
3. **Database Drivers**: Go uses explicit interface implementation rather than inheritance
4. **Type Safety**: Go provides compile-time type checking vs Python's runtime checks
5. **Performance**: Go implementations are optimized for concurrent operations

## Status Legend

- ✅ **Implemented**: Fully ported and functional
- ⚠️ **Partial**: Basic implementation exists but may lack features
- ❌ **Missing**: Not yet implemented
- 🔄 **In Progress**: Currently being worked on

## Contributing

When adding new Python-to-Go mappings:

1. Add the mapping to the appropriate section above
2. Include file location and implementation status
3. Note any significant API differences
4. Update the migration notes if architectural patterns differ

## Implementation Status Summary

### Overall Porting Progress

| Category | Total Methods | ✅ Implemented | ⚠️ Partial | ❌ Missing | Coverage |
|----------|---------------|----------------|------------|------------|----------|
| Core Graphiti Class | 12 | 7 | 2 | 3 | 75% |
| Graph Queries | 8 | 8 | 0 | 0 | 100% |
| Search Functionality | 25+ | 20+ | 3 | 2 | 85% |
| Driver Interface | 25+ | 25+ | 0 | 0 | 100% |
| Node/Edge Types | 12 | 8 | 4 | 0 | 100% |
| LLM Clients | 8 | 4 | 0 | 4 | 50% |
| Embedder Clients | 6 | 3 | 0 | 3 | 50% |
| Cross Encoder | 3 | 0 | 0 | 3 | 0% |
| Prompts | 29 | 29 | 0 | 0 | 100% |
| Utilities | 25+ | 20+ | 3 | 2 | 85% |
| Telemetry | 2 | 0 | 0 | 2 | 0% |

### Key Missing Components

1. ~~**Community Operations** - Community building and management not implemented~~ ✅ **Completed**
2. **Cross Encoder Support** - Reranking with cross encoders missing
3. ~~**Advanced Prompt Templates** - Most prompt templates need implementation~~ ✅ **Completed**
4. **Bulk Utilities** - Deduplication and bulk operations partially implemented
5. **Telemetry** - Event tracking and metrics collection missing
6. **Multiple LLM Providers** - Only OpenAI client implemented
7. **Advanced Temporal Operations** - Time-based graph operations limited

### Well-Implemented Areas

1. **Core Search** - Hybrid search with multiple methods working
2. **Database Drivers** - Neo4j and Kuzu drivers fully functional
3. **Basic Graph Operations** - Node/edge CRUD operations complete
4. **Query Building** - Database-agnostic query construction implemented
5. **Search Configuration** - Comprehensive search configs and recipes
6. **Community Operations** - Label propagation clustering and community building
7. **Prompt Templates** - Complete prompt library with all Python functions ported
8. **Utility Functions** - Comprehensive helper functions, datetime utils, validation, and bulk operations

## Last Updated

This document was last updated: 2024-12-19

*Note: This mapping reflects the current state of the go-graphiti implementation. Status may change as development continues.*