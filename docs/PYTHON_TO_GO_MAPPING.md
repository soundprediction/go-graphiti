# Python to Go Method Mapping

This document tracks the mapping between the original Python Graphiti methods and their corresponding Go implementations in go-graphiti.

## Port Version Tracking

**Last Major Update:** 2025-09-20
**Python Graphiti Reference:** As of 2025-09-20 (active development)
**Port Status:** Core methods implemented with comprehensive date tracking

### Porting Progress Summary

**Total Methods Tracked:** 246 methods/components
- ✅ **Implemented:** 234 methods (95.1%)
- ❌ **Missing:** 8 methods (3.3%)
- ⚠️ **Partial:** 4 methods (1.6%)

**Recently Completed (2025-10-03):**
- Kuzu driver embedding search methods (SearchNodesByEmbedding, SearchEdgesByEmbedding)

**Recently Completed (2025-09-20):**
- All missing LLM clients (Anthropic, Gemini, Groq, Azure OpenAI)
- All missing embedder clients (Azure OpenAI, Gemini, Voyage)
- Cross encoder functionality (BGE and Gemini rerankers)
- Comprehensive Python method linking and date tracking

### Implementation Date Legend
- **2025-10-03:** Kuzu driver embedding search - implemented SearchNodesByEmbedding and SearchEdgesByEmbedding based on Python's node_similarity_search and edge_similarity_search functions
- **2025-09-20:** Major implementation push - added missing LLM clients (Anthropic, Gemini, Groq, Azure), embedder clients (Azure, Gemini, Voyage), cross encoder functionality (BGE, Gemini rerankers), and get_token_count utility
- Methods marked as "implemented" without specific dates were completed prior to version tracking implementation

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
| [`Graphiti.__init__()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L169) | [`NewClient()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L88) | `graphiti.go` | ✅ Implemented (2025-09-20) | Go uses functional construction pattern. Go is missing `cross_encoder`, `store_raw_episode_content`, `max_coroutines`, `ensure_ascii`. |
| [`Graphiti.close()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L368) | [`Client.Close()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1300) | `graphiti.go` | ✅ Implemented (2025-09-20) | Go version takes a `context` argument. |
| [`Graphiti.build_indices_and_constraints()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L401) | [`Client.CreateIndices()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1296) | `graphiti.go` | ✅ Implemented (2025-09-20) | Python version takes `delete_existing` argument. |
| [`Graphiti.retrieve_episodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L440) | [`Client.GetEpisodes()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1253) | `graphiti.go` | ✅ Implemented (2025-09-20) | Go version is missing `reference_time` and `source` arguments. |
| [`Graphiti.add_episode()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L477) | [`Client.Add()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L115) | `graphiti.go` | ✅ Implemented (2025-09-20) | Go method accepts multiple episodes. Python version has more arguments like `update_communities`, `entity_types`, `excluded_entity_types`, `previous_episode_uuids`, `edge_types`, `edge_type_map`. |
| [`Graphiti.add_episode_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L650) | [`Client.Add()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L115) | `graphiti.go` | ✅ Implemented (2025-09-20) | Same as single episode in Go. Python version has more arguments like `entity_types`, `excluded_entity_types`, `edge_types`, `edge_type_map`. |
| [`Graphiti.build_communities()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L886) | [`Builder.BuildCommunities()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L121) | `pkg/community/community.go` | ✅ Implemented (2025-09-20) | Community building with label propagation |
| [`Graphiti.search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L920) | [`Client.Search()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1181) | `graphiti.go` | ✅ Implemented (2025-09-20) | Go version is missing `center_node_uuid` argument. |
| [`Graphiti._search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L971) | `Client.Search()` internal | `graphiti.go` | ✅ Implemented (2025-09-20) | Merged into main Search method |
| [`Graphiti.search_()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L980) | [`searcher.Search()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L142) | `pkg/search/search.go` | ✅ Implemented (2025-09-20) | Go version is missing `center_node_uuid` and `bfs_origin_node_uuids` arguments. |
| [`Graphiti.get_nodes_and_edges_by_episode()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L1000) | [`Client.GetNodesAndEdgesByEpisode()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1105) | `graphiti.go` | ✅ Implemented (2025-09-20) | Retrieves all nodes and edges associated with a specific episode |
| [`Graphiti.add_triplet()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L1012) | [`Client.AddTriplet()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1137) | `graphiti.go` | ✅ Implemented (2025-09-20) | Direct triplet addition |
| [`Graphiti.remove_episode()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L1087) | [`Client.RemoveEpisode()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1014) | `graphiti.go` | ✅ Implemented (2025-09-20) | Exact translation of Python remove_episode logic |

### Result Types

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| [`AddEpisodeResults`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L133) | [`AddEpisodeResults`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L273) | `pkg/types/types.go` | ✅ Implemented (2025-09-20) |
| [`AddBulkEpisodeResults`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L141) | [`AddBulkEpisodeResults`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L286) | `pkg/types/types.go` | ✅ Implemented (2025-09-20) |
| [`AddTripletResults`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L149) | [`AddTripletResults`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L299) | `pkg/types/types.go` | ✅ Implemented (2025-09-20) |

### Additional Go Result Types

| Go Type | Description | File Location |
|---------|-------------|---------------|
| [`EpisodeProcessingResult`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L305) | Internal episode processing result | `pkg/types/types.go` |
| [`BulkEpisodeResults`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L318) | Bulk episode processing statistics | `pkg/types/types.go` |
| [`TripletResults`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L327) | Enhanced triplet operation result | `pkg/types/types.go` |

## Core Graph Queries

### graph_queries.py (Go Implementation: `pkg/driver/graph_queries.go`)

| Python Method | Go Method | Status |
|---------------|-----------|--------|
| [`get_range_indices(provider)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L30) | [`GetRangeIndices(provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L40) | ✅ Implemented (2025-09-20) |
| [`get_fulltext_indices(provider)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L80) | [`GetFulltextIndices(provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L100) | ✅ Implemented (2025-09-20) |
| [`get_nodes_query(name, query, limit, provider)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L108) | [`GetNodesQuery(indexName, query string, limit int, provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L136) | ✅ Implemented (2025-09-20) |
| [`get_relationships_query(name, limit, provider)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L128) | [`GetRelationshipsQuery(indexName string, limit int, provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L152) | ✅ Implemented (2025-09-20) |
| [`get_vector_cosine_func_query(vec1, vec2, provider)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L119) | [`GetVectorCosineFuncQuery(vec1, vec2 string, provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L168) | ✅ Implemented (2025-09-20) |
| `GraphProvider` enum | `GraphProvider` type | ✅ Implemented (2025-09-20) |
| [`NEO4J_TO_FALKORDB_MAPPING`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L10) | [`neo4jToFalkorDBMapping`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L21) | ✅ Implemented (2025-09-20) |
| [`INDEX_TO_LABEL_KUZU_MAPPING`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L17) | [`indexToLabelKuzuMapping`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L29) | ✅ Implemented (2025-09-20) |

### Additional Go Utilities (not in Python)

| Go Method | Description | File Location |
|-----------|-------------|---------------|
| [`NewQueryBuilder(provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L184) | Creates database-agnostic query builder | `pkg/driver/graph_queries.go` |
| [`QueryBuilder.BuildFulltextNodeQuery()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L191) | Builds fulltext node search queries | `pkg/driver/graph_queries.go` |
| [`QueryBuilder.BuildFulltextRelationshipQuery()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L196) | Builds fulltext relationship queries | `pkg/driver/graph_queries.go` |
| [`QueryBuilder.BuildCosineSimilarityQuery()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L201) | Builds cosine similarity queries | `pkg/driver/graph_queries.go` |
| [`EscapeQueryString(query string)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L226) | Escapes special characters in queries | `pkg/driver/graph_queries.go` |
| [`BuildParameterizedQuery()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L255) | Builds parameterized queries | `pkg/driver/graph_queries.go` |

## Search Functionality

### search/search.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| `Searcher` class | [`Searcher`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L118) struct | `pkg/search/search.go` | ✅ Implemented (2025-09-20) | Python version is a collection of methods, while the go version is a struct. |
| [`HybridSearch()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search.py#L70) | [`HybridSearch()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L142) | `pkg/search/search.go` | ✅ Implemented (2025-09-20) | Go version is missing `center_node_uuid`, `bfs_origin_node_uuids`, and `query_vector` arguments. |
| Search methods (cosine_similarity, bm25, bfs) | [`SearchMethod`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L15) constants | `pkg/search/search.go` | ✅ Implemented (2025-09-20) | |
| Reranker types | [`RerankerType`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L22) constants | `pkg/search/search.go` | ✅ Implemented (2025-09-20) |

### search/search_config.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`SearchConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L130) class | [`SearchConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L35) struct | `pkg/search/search.go` | ✅ Implemented (2025-09-20) | Go version is missing `edge_config`, `node_config`, `episode_config`, `community_config`, and `reranker_min_score`. |
| [`NodeSearchConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L108) | [`NodeSearchConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L42) struct | `pkg/search/search.go` | ✅ Implemented (2025-09-20) | Go version is missing `mmr_lambda` and `bfs_max_depth`. |
| [`EdgeSearchConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L99) | [`EdgeSearchConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L50) struct | `pkg/search/search.go` | ✅ Implemented (2025-09-20) | Go version is missing `mmr_lambda` and `bfs_max_depth`. |
| [`EpisodeSearchConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L117) | [`EpisodeSearchConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L58) struct | `pkg/search/search.go` | ✅ Implemented (2025-09-20) | Go version is missing `sim_min_score`, `mmr_lambda`, and `bfs_max_depth`. |
| [`CommunitySearchConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L124) | [`CommunitySearchConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L64) struct | `pkg/search/search.go` | ✅ Implemented (2025-09-20) | Go version is missing `sim_min_score`, `mmr_lambda`, and `bfs_max_depth`. |
| [`SearchResults`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L137) | [`HybridSearchResult`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L95) struct | `pkg/search/search.go` | ✅ Implemented (2025-09-20) | Go version is missing `episodes`, `episode_reranker_scores`, `communities`, and `community_reranker_scores`. |

### search/search_config_recipes.py

| Python Configuration | Go Configuration | File Location | Status |
|----------------------|------------------|---------------|--------|
| [`COMBINED_HYBRID_SEARCH_RRF`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L27) | [`CombinedHybridSearchRRF`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L3) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`COMBINED_HYBRID_SEARCH_MMR`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L50) | [`CombinedHybridSearchMMR`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L25) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`COMBINED_HYBRID_SEARCH_CROSS_ENCODER`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L74) | [`CombinedHybridSearchCrossEncoder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L50) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`EDGE_HYBRID_SEARCH_RRF`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L102) | [`EdgeHybridSearchRRF`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L72) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`EDGE_HYBRID_SEARCH_MMR`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L109) | [`EdgeHybridSearchMMR`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L79) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`EDGE_HYBRID_SEARCH_NODE_DISTANCE`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L116) | [`EdgeHybridSearchNodeDistance`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L86) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`EDGE_HYBRID_SEARCH_EPISODE_MENTIONS`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L123) | [`EdgeHybridSearchEpisodeMentions`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L93) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`EDGE_HYBRID_SEARCH_CROSS_ENCODER`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L130) | [`EdgeHybridSearchCrossEncoder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L100) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`NODE_HYBRID_SEARCH_RRF`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L141) | [`NodeHybridSearchRRF`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L108) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`NODE_HYBRID_SEARCH_MMR`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L148) | [`NodeHybridSearchMMR`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L115) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`NODE_HYBRID_SEARCH_NODE_DISTANCE`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L155) | [`NodeHybridSearchNodeDistance`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L122) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`NODE_HYBRID_SEARCH_EPISODE_MENTIONS`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L162) | [`NodeHybridSearchEpisodeMentions`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L129) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`NODE_HYBRID_SEARCH_CROSS_ENCODER`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L169) | [`NodeHybridSearchCrossEncoder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L136) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`COMMUNITY_HYBRID_SEARCH_RRF`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L180) | [`CommunityHybridSearchRRF`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L144) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`COMMUNITY_HYBRID_SEARCH_MMR`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L187) | [`CommunityHybridSearchMMR`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L151) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |
| [`COMMUNITY_HYBRID_SEARCH_CROSS_ENCODER`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L194) | [`CommunityHybridSearchCrossEncoder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L158) | `pkg/search/config_recipes.go` | ✅ Implemented (2025-09-20) |

### search/search_filters.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`SearchFilters`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L40) class | [`SearchFilters`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L70) struct | `pkg/search/search.go` | ✅ Implemented (2025-09-20) |
| [`ComparisonOperator`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L19) enum | [`ComparisonOperator`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/filters.go#L11) type | `pkg/search/filters.go` | ✅ Implemented (2025-09-20) |
| [`DateFilter`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L33) class | [`DateFilter`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/filters.go#L25) struct | `pkg/search/filters.go` | ✅ Implemented (2025-09-20) |
| [`node_search_filter_query_constructor()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L54) | [`NodeSearchFilterQueryConstructor()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/filters.go#L64) | `pkg/search/filters.go` | ✅ Implemented (2025-09-20) |
| [`edge_search_filter_query_constructor()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L95) | [`EdgeSearchFilterQueryConstructor()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/filters.go#L110) | `pkg/search/filters.go` | ✅ Implemented (2025-09-20) |
| [`date_filter_query_constructor()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L75) | [`constructDateFilterQuery()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/filters.go#L211) | `pkg/search/filters.go` | ✅ Implemented (2025-09-20) |

### search/search_helpers.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`format_edge_date_range(edge)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_helpers.py#L21) | [`FormatEdgeDateRange(edge *types.Edge)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/helpers.go#L12) | `pkg/search/helpers.go` | ✅ Implemented (2025-09-20) |
| [`search_results_to_context_string()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_helpers.py#L26) | [`SearchResultsToContextString()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/helpers.go#L27) | `pkg/search/helpers.go` | ✅ Implemented (2025-09-20) |

### search/search_utils.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|-------|
| [`calculate_cosine_similarity()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L61) | [`CalculateCosineSimilarity()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L38) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`fulltext_query()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L75) | [`FulltextQuery()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L63) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`get_episodes_by_mentions()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L93) | [`GetEpisodesByMentions()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L297) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`get_mentioned_nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L104) | [`GetMentionedNodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L326) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`get_communities_by_nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L121) | [`GetCommunitiesByNodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L333) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`edge_fulltext_search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L131) | [`EdgeFulltextSearch()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L178) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`edge_similarity_search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L242) | [`EdgeSimilaritySearch()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L208) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`edge_bfs_search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L371) | [`EdgeBFSSearch()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/graph_traversal.go#L91) | `pkg/search/graph_traversal.go` | ✅ Implemented (2025-09-20) | |
| [`node_fulltext_search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L460) | [`NodeFulltextSearch()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L119) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`node_similarity_search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L548) | [`NodeSimilaritySearch()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L150) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`node_bfs_search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L623) | [`NodeBFSSearch()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/graph_traversal.go#L20) | `pkg/search/graph_traversal.go` | ✅ Implemented (2025-09-20) | |
| [`hybrid_node_search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L670) | [`HybridNodeSearch()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L236) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`get_relevant_nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L722) | [`GetRelevantNodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/specialized_search.go#L126) | `pkg/search/specialized_search.go` | ✅ Implemented (2025-09-20) | Different signature than Python |
| [`get_relevant_edges()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L784) | [`GetRelevantEdges()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/specialized_search.go#L188) | `pkg/search/specialized_search.go` | ✅ Implemented (2025-09-20) | Different signature than Python |
| [`get_relevant_schema()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L858) | Not implemented | - | ❌ Missing | Complex unified function |
| [`mmr_rerank()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L1501) | [`MMRRerank()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L424) | `pkg/search/search_utils.go` | ✅ Implemented (2025-09-20) | |
| [`rrf_fuse()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L1433) | [`RRF()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/rerankers.go#L25) | `pkg/search/rerankers.go` | ✅ Implemented (2025-09-20) | |

## Driver Interface

### driver/driver.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`GraphDriver`](https://github.com/getzep/graphiti/blob/main/graphiti_core/driver/driver.py#L63) interface | [`GraphDriver`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/driver.go#L12) interface | `pkg/driver/driver.go` | ✅ Implemented (2025-09-20) | Go version has many more methods. | Go version has many more methods. | Go version has many more methods. | Go version has many more methods. | Go version has many more methods. | Go version has many more methods. | Go version has many more methods. | Go version has many more methods. | Go version has many more methods. | Go version has many more methods. | Go version has many more methods. | Go version has many more methods. |
| Database operations (GetNode, UpsertNode, etc.) | Same method names | `pkg/driver/driver.go` | ✅ Implemented (2025-09-20) | |

### driver/neo4j.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`Neo4jDriver`](https://github.com/getzep/graphiti/blob/main/graphiti_core/driver/neo4j_driver.py#L28) class | [`Neo4jDriver`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/neo4j.go#L19) struct | `pkg/driver/neo4j.go` | ✅ Implemented (2025-09-20) |
| All GraphDriver interface methods | Same method names | `pkg/driver/neo4j.go` | ✅ Implemented (2025-09-20) |

### driver/falkordb_driver.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `FalkorDBDriver` | N/A | N/A | ❌ Missing |

### driver/neptune_driver.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `NeptuneDriver` | N/A | N/A | ❌ Missing |

### driver/kuzu_driver.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|-------|
| [`KuzuDriver`](https://github.com/getzep/graphiti/blob/main/graphiti_core/driver/kuzu_driver.py#L93) class | [`KuzuDriver`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/kuzu.go#L84) struct | `pkg/driver/kuzu.go` | ✅ Implemented | |
| All GraphDriver interface methods | Same method names | `pkg/driver/kuzu.go` | ✅ Implemented | |

### Additional Kuzu Driver Methods (Go-specific)

These methods implement vector similarity search based on Python's `search_utils.py` functions but are exposed directly on the Kuzu driver for efficiency:

| Python Function (search_utils.py) | Go Method (KuzuDriver) | File Location | Status | Implementation Date | Notes |
|---------------|-----------|---------------|--------|---------------------|-------|
| [`node_similarity_search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L728-869) (Kuzu provider) | [`SearchNodesByEmbedding()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/kuzu.go#L495-601) | `pkg/driver/kuzu.go` | ✅ Implemented | 2025-10-03 | Performs cosine similarity search on Entity.name_embedding using Kuzu's array_cosine_similarity function |
| [`edge_similarity_search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L312-477) (Kuzu provider) | [`SearchEdgesByEmbedding()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/kuzu.go#L603-725) | `pkg/driver/kuzu.go` | ✅ Implemented | 2025-10-03 | Performs cosine similarity search on RelatesToNode_.fact_embedding using Kuzu's array_cosine_similarity function |
| [`get_nodes_in_time_range()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/driver/neo4j_driver.py#L395-424) (Neo4j reference) | [`GetNodesInTimeRange()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/kuzu.go#L837-902) | `pkg/driver/kuzu.go` | ✅ Implemented | 2025-10-03 | Retrieves Entity nodes filtered by created_at within time range and group_id |
| [`get_edges_in_time_range()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/driver/neo4j_driver.py#L426-459) (Neo4j reference) | [`GetEdgesInTimeRange()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/kuzu.go#L904-997) | `pkg/driver/kuzu.go` | ✅ Implemented | 2025-10-03 | Retrieves RelatesToNode_ edges filtered by created_at within time range and group_id |


## Node and Edge Types

### nodes.py / edges.py

| Python Type | Go Type | File Location | Status | Notes |
|-------------|---------|---------------|--------|--------|
| [`Node`](https://github.com/getzep/graphiti/blob/main/graphiti_core/nodes.py#L88) base class | [`types.Node`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L8) struct | `pkg/types/types.go` | ⚠️ Partial | Go version has many more fields. |
| [`EntityNode`](https://github.com/getzep/graphiti/blob/main/graphiti_core/nodes.py#L513) | `types.Node` with `Type: EntityNodeType` | `pkg/types/types.go` | ✅ Implemented (2025-09-20) | Go version has these fields in the `Node` struct. |
| [`EpisodicNode`](https://github.com/getzep/graphiti/blob/main/graphiti_core/nodes.py#L321) | `types.Node` with `Type: EpisodicNodeType` | `pkg/types/types.go` | ✅ Implemented (2025-09-20) | Go version has these fields in the `Node` struct. |
| [`CommunityNode`](https://github.com/getzep/graphiti/blob/main/graphiti_core/nodes.py#L653) | `types.Node` with `Type: CommunityNodeType` | `pkg/types/types.go` | ✅ Implemented (2025-09-20) | Go version has these fields in the `Node` struct. |
| [`Edge`](https://github.com/getzep/graphiti/blob/main/graphiti_core/edges.py#L40) base class | [`types.Edge`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L41) struct | `pkg/types/types.go` | ⚠️ Partial | Go version has many more fields. |
| [`EntityEdge`](https://github.com/getzep/graphiti/blob/main/graphiti_core/edges.py#L241) | `types.Edge` with `Type: EntityEdgeType` | `pkg/types/types.go` | ✅ Implemented (2025-09-20) | Go version has these fields in the `Edge` struct. |
| [`EpisodicEdge`](https://github.com/getzep/graphiti/blob/main/graphiti_core/edges.py#L141) | `types.Edge` with `Type: EpisodicEdgeType` | `pkg/types/types.go` | ✅ Implemented (2025-09-20) | Go version has the fields from the `Edge` struct. |
| [`CommunityEdge`](https://github.com/getzep/graphiti/blob/main/graphiti_core/edges.py#L517) | `types.Edge` with `Type: CommunityEdgeType` | `pkg/types/types.go` | ✅ Implemented (2025-09-20) | Go version has the fields from the `Edge` struct. |

### models/edges/edge_db_queries.py

| Python Function | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`get_episodic_edge_save_bulk_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/edges/edge_db_queries.py#L30) | [`GetEpisodicEdgeSaveBulkQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/edges/edge_db_queries.go#L19) | `pkg/models/edges/edge_db_queries.go` | ✅ Implemented (2025-09-20) |
| [`get_entity_edge_save_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/edges/edge_db_queries.py#L63) | [`GetEntityEdgeSaveQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/edges/edge_db_queries.go#L54) | `pkg/models/edges/edge_db_queries.go` | ✅ Implemented (2025-09-20) |
| [`get_entity_edge_save_bulk_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/edges/edge_db_queries.py#L123) | [`GetEntityEdgeSaveBulkQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/edges/edge_db_queries.go#L110) | `pkg/models/edges/edge_db_queries.go` | ✅ Implemented (2025-09-20) |
| [`get_entity_edge_return_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/edges/edge_db_queries.py#L186) | [`GetEntityEdgeReturnQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/edges/edge_db_queries.go#L172) | `pkg/models/edges/edge_db_queries.go` | ✅ Implemented (2025-09-20) |
| [`get_community_edge_save_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/edges/edge_db_queries.py#L224) | [`GetCommunityEdgeSaveQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/edges/edge_db_queries.go#L211) | `pkg/models/edges/edge_db_queries.go` | ✅ Implemented (2025-09-20) |
| `EPISODIC_EDGE_SAVE` | [`EPISODIC_EDGE_SAVE`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/edges/edge_db_queries.go#L8) | `pkg/models/edges/edge_db_queries.go` | ✅ Implemented (2025-09-20) |
| `EPISODIC_EDGE_RETURN` | [`EPISODIC_EDGE_RETURN`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/edges/edge_db_queries.go#L45) | `pkg/models/edges/edge_db_queries.go` | ✅ Implemented (2025-09-20) |
| `COMMUNITY_EDGE_RETURN` | [`COMMUNITY_EDGE_RETURN`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/edges/edge_db_queries.go#L262) | `pkg/models/edges/edge_db_queries.go` | ✅ Implemented (2025-09-20) |

### models/nodes/node_db_queries.py

| Python Function | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`get_episode_node_save_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/nodes/node_db_queries.py#L22) | [`GetEpisodeNodeSaveQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/nodes/node_db_queries.go#L11) | `pkg/models/nodes/node_db_queries.go` | ✅ Implemented (2025-09-20) |
| [`get_episode_node_save_bulk_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/nodes/node_db_queries.py#L61) | [`GetEpisodeNodeSaveBulkQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/nodes/node_db_queries.go#L52) | `pkg/models/nodes/node_db_queries.go` | ✅ Implemented (2025-09-20) |
| [`get_entity_node_save_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/nodes/node_db_queries.py#L129) | [`GetEntityNodeSaveQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/nodes/node_db_queries.go#L123) | `pkg/models/nodes/node_db_queries.go` | ✅ Implemented (2025-09-20) |
| [`get_entity_node_save_bulk_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/nodes/node_db_queries.py#L182) | [`GetEntityNodeSaveBulkQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/nodes/node_db_queries.go#L184) | `pkg/models/nodes/node_db_queries.go` | ✅ Implemented (2025-09-20) |
| [`get_entity_node_return_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/nodes/node_db_queries.py#L255) | [`GetEntityNodeReturnQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/nodes/node_db_queries.go#L271) | `pkg/models/nodes/node_db_queries.go` | ✅ Implemented (2025-09-20) |
| [`get_community_node_save_query`](https://github.com/getzep/graphiti/blob/main/graphiti_core/models/nodes/node_db_queries.py#L279) | [`GetCommunityNodeSaveQuery`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/nodes/node_db_queries.go#L296) | `pkg/models/nodes/node_db_queries.go` | ✅ Implemented (2025-09-20) |
| `EPISODIC_NODE_RETURN` | [`EPISODIC_NODE_RETURN`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/nodes/node_db_queries.go#L97) | `pkg/models/nodes/node_db_queries.go` | ✅ Implemented (2025-09-20) |
| `EPISODIC_NODE_RETURN_NEPTUNE` | [`EPISODIC_NODE_RETURN_NEPTUNE`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/nodes/node_db_queries.go#L110) | `pkg/models/nodes/node_db_queries.go` | ✅ Implemented (2025-09-20) |
| `COMMUNITY_NODE_RETURN` | [`COMMUNITY_NODE_RETURN`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/nodes/node_db_queries.go#L333) | `pkg/models/nodes/node_db_queries.go` | ✅ Implemented (2025-09-20) |
| `COMMUNITY_NODE_RETURN_NEPTUNE` | [`COMMUNITY_NODE_RETURN_NEPTUNE`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/models/nodes/node_db_queries.go#L343) | `pkg/models/nodes/node_db_queries.go` | ✅ Implemented (2025-09-20) |


## LLM Client Interface

### llm_client/client.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|--------|
| [`LLMClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/client.py#L53) abstract class | [`llm.Client`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/client.go#L8) interface | `pkg/llm/client.go` | ✅ Implemented (2025-09-20) | |
| [`LLMClient.generate()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/client.py#L165) | [`Client.Chat()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/client.go#L10) | `pkg/llm/client.go` | ✅ Implemented (2025-09-20) | |
| `LLMClient.generate_batch()` | `Client.ChatBatch()` | `pkg/llm/` | ❌ Missing | Batch operations not implemented |
| [`LLMClient.generate_with_schema()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/client.py#L165) | [`Client.ChatWithStructuredOutput()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/client.go#L13) | `pkg/llm/client.go` | ✅ Implemented (2025-09-20) | |

### llm_client/anthropic_client.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `AnthropicClient` | [`AnthropicClient`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/anthropic.go#L13) | `pkg/llm/anthropic.go` | ✅ Implemented (2025-09-20) |

### llm_client/azure_openai_client.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `AzureOpenAIClient` | [`AzureOpenAIClient`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/azure_openai.go#L14) | `pkg/llm/azure_openai.go` | ✅ Implemented (2025-09-20) |

### llm_client/gemini_client.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `GeminiClient` | [`GeminiClient`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/gemini.go#L14) | `pkg/llm/gemini.go` | ✅ Implemented (2025-09-20) |

### llm_client/groq_client.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `GroqClient` | [`GroqClient`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/groq.go#L15) | `pkg/llm/groq.go` | ✅ Implemented (2025-09-20) |

### llm_client/openai_base_client.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`BaseOpenAIClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/openai_base_client.py#L40) | [`BaseOpenAIClient`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/openai_base.go#L26) | `pkg/llm/openai_base.go` | ✅ Implemented (2025-09-20) | Provides common functionality for OpenAI-compatible clients with retry logic, message conversion, and error handling |

### llm_client/openai_generic_client.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`OpenAIGenericClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/openai_generic_client.py#L37) | [`OpenAIGenericClient`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/openai_generic.go#L16) | `pkg/llm/openai_generic.go` | ✅ Implemented (2025-09-20) | OpenAI client with structured output, retry logic, and enhanced error feedback |

### llm_client/utils.py

| Python Function | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `get_token_count` | N/A | N/A | ❌ Missing |


### LLM Configuration

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| [`LLMConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/config.py#L21) | [`LLMConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/config.go#L20) | `pkg/llm/config.go` | ✅ Implemented (2025-09-20) | Go version matches Python structure with APIKey, Model, BaseURL, Temperature, MaxTokens, and SmallModel |
| [`ModelSize`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/config.py#L15) enum | [`ModelSize`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/config.go#L5) | `pkg/llm/config.go` | ✅ Implemented (2025-09-20) | Go constants for ModelSizeSmall and ModelSizeMedium |

## Embedder Client Interface

### embedder/client.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|--------|
| [`EmbedderClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/embedder/client.py#L22) abstract class | [`embedder.Client`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/client.go#L8) interface | `pkg/embedder/client.go` | ✅ Implemented (2025-09-20) | |
| [`EmbedderClient.create()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/embedder/client.py#L24) | [`Client.EmbedSingle()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/client.go#L13) | `pkg/embedder/client.go` | ✅ Implemented (2025-09-20) | |
| [`EmbedderClient.create_batch()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/embedder/client.py#L29) | [`Client.Embed()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/client.go#L10) | `pkg/embedder/client.go` | ✅ Implemented (2025-09-20) | |

### embedder/azure_openai.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `AzureOpenAIEmbedder` | [`AzureOpenAIEmbedder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/azure_openai.go#L14) | `pkg/embedder/azure_openai.go` | ✅ Implemented (2025-09-20) |

### embedder/gemini.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `GeminiEmbedder` | [`GeminiEmbedder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/gemini.go#L14) | `pkg/embedder/gemini.go` | ✅ Implemented (2025-09-20) |

### embedder/voyage.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `VoyageEmbedder` | [`VoyageEmbedder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/voyage.go#L12) | `pkg/embedder/voyage.go` | ✅ Implemented (2025-09-20) |


### Embedder Configuration

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| [`EmbedderConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/embedder/client.py#L17) | [`embedder.Config`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/client.go#L22) | `pkg/embedder/client.go` | ⚠️ Partial | Go version is missing `embedding_dim`. It has `Model`, `BatchSize`, `BaseURL`, `Headers` which are not in the python version. |
| `EMBEDDING_DIM` constant | `Dimensions` | `pkg/embedder/client.go` | ✅ Implemented (2025-09-20) | In go, this is a field in the `Config` struct. |

## Cross Encoder Interface

### cross_encoder/client.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|--------|
| [`CrossEncoderClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/cross_encoder/client.py#L18) abstract class | [`crossencoder.Client`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/crossencoder/client.go#L14) interface | `pkg/crossencoder/client.go` | ✅ Implemented (2025-09-20) | Go interface with Rank() and Close() methods |
| [`CrossEncoderClient.rerank()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/cross_encoder/client.py#L25) | [`Client.Rank()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/crossencoder/client.go#L17) | `pkg/crossencoder/client.go` | ✅ Implemented (2025-09-20) | Method renamed to Rank for Go conventions |

### cross_encoder/bge_reranker_client.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `BGERerankerClient` | [`BGERerankerClient`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/crossencoder/bge.go#L12) | `pkg/crossencoder/bge.go` | ✅ Implemented (2025-09-20) |

### cross_encoder/gemini_reranker_client.py

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `GeminiRerankerClient` | [`GeminiRerankerClient`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/crossencoder/gemini.go#L15) | `pkg/crossencoder/gemini.go` | ✅ Implemented (2025-09-20) |

### Additional Cross Encoder Implementations

| Go Implementation | Description | File Location |
|-------------------|-------------|---------------|
| [`OpenAIRerankerClient`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/crossencoder/openai.go#L15) | OpenAI-based cross encoder using boolean classification | `pkg/crossencoder/openai.go` |


## Community Operations

### utils/maintenance/community_operations.py

| Python Function | Go Method | File Location | Status | Notes |
|-----------------|-----------|---------------|--------|--------|
| [`get_community_clusters()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L33) | [`Builder.GetCommunityClusters()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L49) | `pkg/community/community.go` | ✅ Implemented (2025-09-20) | |
| [`label_propagation()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L100) | [`Builder.labelPropagation()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/label_propagation.go#L11) | `pkg/community/label_propagation.go` | ✅ Implemented (2025-09-20) | |
| [`build_community()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L201) | [`Builder.buildCommunity()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L181) | `pkg/community/community.go` | ✅ Implemented (2025-09-20) | |
| [`build_communities()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L233) | [`Builder.BuildCommunities()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L121) | `pkg/community/community.go` | ✅ Implemented (2025-09-20) | |
| [`remove_communities()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L256) | [`Builder.RemoveCommunities()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L461) | `pkg/community/community.go` | ✅ Implemented (2025-09-20) | |
| [`determine_entity_community()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L264) | [`Builder.DetermineEntityCommunity()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/update.go#L23) | `pkg/community/update.go` | ✅ Implemented (2025-09-20) | |
| [`update_community()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L311) | [`Builder.UpdateCommunity()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/update.go#L60) | `pkg/community/update.go` | ✅ Implemented (2025-09-20) | |
| [`summarize_pair()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L158) | [`Builder.summarizePair()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L351) | `pkg/community/community.go` | ✅ Implemented (2025-09-20) | |
| [`generate_summary_description()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L179) | [`Builder.generateCommunityName()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L380) | `pkg/community/community.go` | ✅ Implemented (2025-09-20) | |

### Community Types and Models

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| [`Neighbor`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L27) class | [`Neighbor`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L38) struct | `pkg/community/community.go` | ✅ Implemented (2025-09-20) |
| `BuildCommunitiesResult` | [`BuildCommunitiesResult`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L44) struct | `pkg/community/community.go` | ✅ Implemented (2025-09-20) |
| `DetermineEntityCommunityResult` | [`DetermineEntityCommunityResult`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/update.go#L11) struct | `pkg/community/update.go` | ✅ Implemented (2025-09-20) |
| `UpdateCommunityResult` | [`UpdateCommunityResult`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/update.go#L17) struct | `pkg/community/update.go` | ✅ Implemented (2025-09-20) |

### Additional Go Community Functions

| Go Method | Description | File Location |
|-----------|-------------|---------------|
| [`NewBuilder()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L28) | Creates new community builder | `pkg/community/community.go` |
| [`buildProjection()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/label_propagation.go#L138) | Builds neighbor projection for clustering | `pkg/community/label_propagation.go` |
| [`getNodeNeighbors()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/label_propagation.go#L154) | Gets node neighbors with edge counts | `pkg/community/label_propagation.go` |
| [`getAllGroupIDs()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/label_propagation.go#L215) | Gets all distinct group IDs | `pkg/community/label_propagation.go` |
| [`getEntityNodesByGroup()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/label_propagation.go#L248) | Gets entity nodes by group | `pkg/community/label_propagation.go` |
| [`hierarchicalSummarize()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L283) | Performs hierarchical summarization | `pkg/community/community.go` |
| [`generateCommunityEmbedding()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L410) | Generates embeddings for communities | `pkg/community/community.go` |
| [`buildCommunityEdges()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L420) | Creates HAS_MEMBER edges | `pkg/community/community.go` |

## Prompts and Models

### prompts/models.py

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| [`Message`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/models.py#L18) class | `Message` struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| `PromptFunction` type | `PromptFunction` type | `pkg/prompts/types.go` | ✅ Implemented (2025-09-20) |
| [`ExtractedEntity`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L25) | [`ExtractedEntity`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L5) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`ExtractedEntities`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L30) | [`ExtractedEntities`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L11) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`EntityClassificationTriple`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L45) | [`EntityClassificationTriple`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L22) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`EntitySummary`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L84) | [`EntitySummary`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L34) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`Edge`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py#L20) | [`Edge`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L39) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`ExtractedEdges`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py#L28) | [`ExtractedEdges`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L49) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`NodeDuplicate`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_nodes.py#L19) | [`NodeDuplicate`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L60) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`EdgeDuplicate`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_edges.py#L19) | [`EdgeDuplicate`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L73) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`InvalidatedEdges`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/invalidate_edges.py#L19) | [`InvalidatedEdges`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L85) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`EdgeDates`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edge_dates.py#L19) | [`EdgeDates`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L90) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`Summary`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py#L19) | [`Summary`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L96) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`SummaryDescription`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py#L23) | [`SummaryDescription`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L101) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`QueryExpansion`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L24) | [`QueryExpansion`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L106) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`QAResponse`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L28) | [`QAResponse`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L111) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |
| [`EvalResponse`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L32) | [`EvalResponse`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L116) struct | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |

### Prompt Templates

| Python Module | Go Implementation | File Location | Status | Notes |
|---------------|-------------------|---------------|--------|--------|
| [`prompts/extract_nodes.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py) | `ExtractNodesPrompt` interface | `pkg/prompts/extract_nodes.go` | ✅ Implemented (2025-09-20) | All 7 functions implemented |
| [`prompts/extract_edges.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py) | `ExtractEdgesPrompt` interface | `pkg/prompts/extract_edges.go` | ✅ Implemented (2025-09-20) | All 3 functions implemented |
| [`prompts/dedupe_nodes.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_nodes.py) | `DedupeNodesPrompt` interface | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented (2025-09-20) | All 3 functions implemented |
| [`prompts/dedupe_edges.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_edges.py) | `DedupeEdgesPrompt` interface | `pkg/prompts/dedupe_edges.go` | ✅ Implemented (2025-09-20) | All 3 functions implemented |
| [`prompts/summarize_nodes.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py) | `SummarizeNodesPrompt` interface | `pkg/prompts/summarize_nodes.go` | ✅ Implemented (2025-09-20) | All 3 functions implemented |
| [`prompts/invalidate_edges.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/invalidate_edges.py) | `InvalidateEdgesPrompt` interface | `pkg/prompts/invalidate_edges.go` | ✅ Implemented (2025-09-20) | All functions implemented including v2 |
| [`prompts/extract_edge_dates.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edge_dates.py) | `ExtractEdgeDatesPrompt` interface | `pkg/prompts/extract_edge_dates.go` | ✅ Implemented (2025-09-20) | v1 function implemented |
| [`prompts/eval.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py) | `EvalPrompt` interface | `pkg/prompts/eval.go` | ✅ Implemented (2025-09-20) | All 4 functions implemented |

### Extract Nodes Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`extract_message()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L91) | [`ExtractMessage()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L41) | `pkg/prompts/extract_nodes.go` | ✅ Implemented (2025-09-20) |
| [`extract_json()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L142) | [`ExtractJSON()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L111) | `pkg/prompts/extract_nodes.go` | ✅ Implemented (2025-09-20) |
| [`extract_text()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L171) | [`ExtractText()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L141) | `pkg/prompts/extract_nodes.go` | ✅ Implemented (2025-09-20) |
| [`reflexion()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L198) | [`Reflexion()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L171) | `pkg/prompts/extract_nodes.go` | ✅ Implemented (2025-09-20) |
| [`classify_nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L221) | [`ClassifyNodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L204) | `pkg/prompts/extract_nodes.go` | ✅ Implemented (2025-09-20) |
| [`extract_attributes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L252) | [`ExtractAttributes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L240) | `pkg/prompts/extract_nodes.go` | ✅ Implemented (2025-09-20) |
| [`extract_summary()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L281) | [`ExtractSummary()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L281) | `pkg/prompts/extract_nodes.go` | ✅ Implemented (2025-09-20) |

### Extract Edges Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`edge()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py#L57) | [`Edge()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_edges.go#L27) | `pkg/prompts/extract_edges.go` | ✅ Implemented (2025-09-20) |
| [`reflexion()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py#L130) | [`Reflexion()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_edges.go#L100) | `pkg/prompts/extract_edges.go` | ✅ Implemented (2025-09-20) |
| [`extract_attributes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py#L156) | [`ExtractAttributes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_edges.go#L133) | `pkg/prompts/extract_edges.go` | ✅ Implemented (2025-09-20) |

### Dedupe Nodes Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`node()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_nodes.py#L51) | [`Node()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_nodes.go#L27) | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented (2025-09-20) |
| [`node_list()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_nodes.py#L180) | [`NodeList()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_nodes.go#L190) | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented (2025-09-20) |
| [`nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_nodes.py#L109) | [`Nodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_nodes.go#L103) | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented (2025-09-20) |

### Dedupe Edges Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`edge()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_edges.py#L53) | [`Edge()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_edges.go#L27) | `pkg/prompts/dedupe_edges.go` | ✅ Implemented (2025-09-20) |
| [`edge_list()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_edges.py#L94) | [`EdgeList()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_edges.go#L90) | `pkg/prompts/dedupe_edges.go` | ✅ Implemented (2025-09-20) |
| [`resolve_edge()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_edges.py#L125) | [`ResolveEdge()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_edges.go#L123) | `pkg/prompts/dedupe_edges.go` | ✅ Implemented (2025-09-20) |

### Summarize Nodes Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`summarize_pair()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py#L43) | [`SummarizePair()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/summarize_nodes.go#L27) | `pkg/prompts/summarize_nodes.go` | ✅ Implemented (2025-09-20) |
| [`summarize_context()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py#L64) | [`SummarizeContext()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/summarize_nodes.go#L60) | `pkg/prompts/summarize_nodes.go` | ✅ Implemented (2025-09-20) |
| [`summary_description()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py#L109) | [`SummaryDescription()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/summarize_nodes.go#L115) | `pkg/prompts/summarize_nodes.go` | ✅ Implemented (2025-09-20) |

### Invalidate Edges Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`v1()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/invalidate_edges.py#L35) | [`V1()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/invalidate_edges.go#L20) | `pkg/prompts/invalidate_edges.go` | ✅ Implemented (2025-09-20) |
| [`v2()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/invalidate_edges.py#L68) | [`V2()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/invalidate_edges.go#L57) | `pkg/prompts/invalidate_edges.go` | ✅ Implemented (2025-09-20) |

### Extract Edge Dates Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`v1()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edge_dates.py#L33) | [`V1()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_edge_dates.go#L19) | `pkg/prompts/extract_edge_dates.go` | ✅ Implemented (2025-09-20) |

### Eval Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`qa_prompt()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L88) | [`QAPrompt()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/eval.go#L65) | `pkg/prompts/eval.go` | ✅ Implemented (2025-09-20) |
| [`eval_prompt()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L111) | [`EvalPrompt()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/eval.go#L106) | `pkg/prompts/eval.go` | ✅ Implemented (2025-09-20) |
| [`query_expansion()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L65) | [`QueryExpansion()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/eval.go#L29) | `pkg/prompts/eval.go` | ✅ Implemented (2025-09-20) |
| [`eval_add_episode_results()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L132) | [`EvalAddEpisodeResults()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/eval.go#L130) | `pkg/prompts/eval.go` | ✅ Implemented (2025-09-20) |

### Prompt Library

| Python Component | Go Component | File Location | Status |
|------------------|--------------|---------------|--------|
| [`PromptLibrary`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/lib.py#L48) interface | [`Library`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/library.go#L4) interface | `pkg/prompts/library.go` | ✅ Implemented (2025-09-20) |
| [`PromptLibraryImpl`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/lib.py#L59) | [`LibraryImpl`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/library.go#L16) struct | `pkg/prompts/library.go` | ✅ Implemented (2025-09-20) |
| `prompt_library` instance | [`NewLibrary()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/library.go#L49) function | `pkg/prompts/library.go` | ✅ Implemented (2025-09-20) |

### Prompt Helpers

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| [`to_prompt_json()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/prompt_helpers.py#L7) | [`ToPromptJSON()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/types.go#L47) | `pkg/prompts/types.go` | ✅ Implemented (2025-09-20) |
| `DO_NOT_ESCAPE_UNICODE` | `DoNotEscapeUnicode` const | `pkg/prompts/models.go` | ✅ Implemented (2025-09-20) |

## Utilities and Helpers

### helpers.py (graphiti_core/)

| Python Function | Go Function | File Location | Status | Notes |
|-----------------|-------------|---------------|--------|--------|
| [`get_default_group_id()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L50) | [`GetDefaultGroupID()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L111) | `pkg/utils/helpers.go` | ✅ Implemented (2025-09-20) | |
| [`semaphore_gather()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L118) | [`SemaphoreGather()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/concurrent.go#L102) | `pkg/utils/concurrent.go` | ✅ Implemented (2025-09-20) | |
| [`validate_excluded_entity_types()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L158) | [`ValidateExcludedEntityTypes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L248) | `pkg/utils/helpers.go` | ✅ Implemented (2025-09-20) | |
| [`validate_group_id()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L132) | [`ValidateGroupID()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L228) | `pkg/utils/helpers.go` | ✅ Implemented (2025-09-20) | |
| [`lucene_sanitize()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L61) | [`LuceneSanitize()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L118) | `pkg/utils/helpers.go` | ✅ Implemented (2025-09-20) | |
| [`normalize_l2()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L109) | [`NormalizeL2()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L154) / `NormalizeL2Float32()` | `pkg/utils/helpers.go` | ✅ Implemented (2025-09-20) | |
| [`parse_db_date()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L38) | [`ParseDBDate()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L78) | `pkg/utils/helpers.go` | ✅ Implemented (2025-09-20) |

### utils/bulk_utils.py

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| [`add_nodes_and_edges_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L78) | [`AddNodesAndEdgesBulk()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk_utils.go#L77) | `pkg/utils/bulk_utils.go` | ✅ Implemented (2025-09-20) | Complete bulk operation with embeddings |
| [`dedupe_edges_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L383) | [`DedupeEdgesBulk()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk_utils.go#L491) | `pkg/utils/bulk_utils.go` | ✅ Implemented (2025-09-20) | Full deduplication with LLM confirmation |
| [`dedupe_nodes_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L281) | [`DedupeNodesBulk()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk_utils.go#L335) | `pkg/utils/bulk_utils.go` | ✅ Implemented (2025-09-20) | Full deduplication with LLM confirmation |
| [`extract_nodes_and_edges_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L219) | [`ExtractNodesAndEdgesBulk()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk_utils.go#L161) | `pkg/utils/bulk_utils.go` | ✅ Implemented (2025-09-20) | Complete extraction with batch processing |
| [`resolve_edge_pointers()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L548) | [`ResolveEdgePointers()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L94) | `pkg/utils/bulk.go` | ✅ Implemented (2025-09-20) | |
| [`retrieve_previous_episodes_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L45) | [`RetrievePreviousEpisodesBulk()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk_utils.go#L40) | `pkg/utils/bulk_utils.go` | ✅ Implemented (2025-09-20) | Temporal episode retrieval |
| [`compress_uuid_map()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L520) | [`CompressUUIDMap()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L55) | `pkg/utils/bulk.go` | ✅ Implemented (2025-09-20) | |
| [`UnionFind`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L500) class | [`UnionFind`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L18) struct | `pkg/utils/bulk.go` | ✅ Implemented (2025-09-20) | |

### utils/datetime_utils.py

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| [`utc_now()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/datetime_utils.py#L18) | [`UTCNow()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/datetime.go#L9) | `pkg/utils/datetime.go` | ✅ Implemented (2025-09-20) |
| [`ensure_utc()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/datetime_utils.py#L22) | [`EnsureUTC()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/datetime.go#L14) | `pkg/utils/datetime.go` | ✅ Implemented (2025-09-20) |
| [`convert_datetimes_to_strings()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/datetime_utils.py#L37) | [`ConvertDatetimesToStrings()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/datetime.go#L29) | `pkg/utils/datetime.go` | ✅ Implemented (2025-09-20) |

### utils/ontology_utils/entity_types_utils.py

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| [`validate_entity_types()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/ontology_utils/entity_types_utils.py#L16) | [`ValidateEntityTypes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/validation.go#L16) | `pkg/utils/validation.go` | ✅ Implemented (2025-09-20) |

### Additional Go Utility Functions

| Go Method | Description | File Location |
|-----------|-------------|---------------|
| [`GetUseParallelRuntime()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L33) | Gets parallel runtime setting from env | `pkg/utils/helpers.go` |
| [`GetSemaphoreLimit()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L43) | Gets semaphore limit from env | `pkg/utils/helpers.go` |
| [`GetMaxReflexionIterations()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L54) | Gets max reflexion iterations from env | `pkg/utils/helpers.go` |
| [`NewConcurrentExecutor()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/concurrent.go#L11) | Creates concurrent executor with semaphore | `pkg/utils/concurrent.go` |
| [`ExecuteWithResults()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/concurrent.go#L58) | Concurrent execution with results | `pkg/utils/concurrent.go` |
| [`NewWorkerPool()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/concurrent.go#L118) | Creates worker pool for processing | `pkg/utils/concurrent.go` |
| [`NewBatchProcessor()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L328) | Creates batch processor | `pkg/utils/bulk.go` |
| [`HasWordOverlap()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L160) | Checks word overlap for deduplication | `pkg/utils/bulk.go` |
| [`CalculateCosineSimilarity()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L179) | Computes cosine similarity | `pkg/utils/bulk.go` |
| [`FindSimilarNodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L201) / [`FindSimilarEdges()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L224) | Find duplicate candidates | `pkg/utils/bulk.go` |
| [`ChunkSlice()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L411) | Splits slices into chunks | `pkg/utils/bulk.go` |
| [`RemoveDuplicateStrings()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L396) | Removes duplicates from string slice | `pkg/utils/bulk.go` |
| [`ValidateUUID()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/validation.go#L150) | Validates UUID format | `pkg/utils/validation.go` |
| [`ValidateRequired()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/validation.go#L162) | Validates required fields | `pkg/utils/validation.go` |
| [`ValidateRange()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/validation.go#L171) | Validates numeric ranges | `pkg/utils/validation.go` |
| [`ValidateEmbeddingDimensions()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/validation.go#L184) | Validates embedding consistency | `pkg/utils/validation.go` |
| [`FormatTimeForDB()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/datetime.go#L90) / [`ParseTimeFromDB()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/datetime.go#L95) | Database time formatting | `pkg/utils/datetime.go` |
| [`TimeToMilliseconds()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/datetime.go#L100) / [`MillisecondsToTime()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/datetime.go#L105) | Time conversion utilities | `pkg/utils/datetime.go` |

### utils/maintenance/edge_operations.py

| Python Function | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`extract_edges()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/edge_operations.py#L112) | [`EdgeOperations.ExtractEdges()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/edge_operations.go#L96) | `pkg/utils/maintenance/edge_operations.go` | ✅ Implemented (2025-09-20) |
| [`resolve_extracted_edge()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/edge_operations.py#L365) | [`EdgeOperations.resolveExtractedEdge()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/edge_operations.go#L340) | `pkg/utils/maintenance/edge_operations.go` | ✅ Implemented (2025-09-20) |
| [`resolve_extracted_edges()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/edge_operations.py#L248) | [`EdgeOperations.ResolveExtractedEdges()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/edge_operations.go#L232) | `pkg/utils/maintenance/edge_operations.go` | ✅ Implemented (2025-09-20) |
| [`build_episodic_edges()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/edge_operations.py#L48) | [`EdgeOperations.BuildEpisodicEdges()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/edge_operations.go#L33) | `pkg/utils/maintenance/edge_operations.go` | ✅ Implemented (2025-09-20) |
| [`build_duplicate_of_edges()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/edge_operations.py#L68) | [`EdgeOperations.BuildDuplicateOfEdges()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/edge_operations.go#L57) | `pkg/utils/maintenance/edge_operations.go` | ✅ Implemented (2025-09-20) |
| [`get_between_nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/edge_operations.py#L218) | [`EdgeOperations.GetBetweenNodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/edge_operations.go#L204) | `pkg/utils/maintenance/edge_operations.go` | ✅ Implemented (2025-09-20) |
| [`filter_existing_duplicate_of_edges()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/edge_operations.py#L578) | [`EdgeOperations.FilterExistingDuplicateOfEdges()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/edge_operations.go#L555) | `pkg/utils/maintenance/edge_operations.go` | ✅ Implemented (2025-09-20) |
| [`resolve_edge_contradictions()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/edge_operations.py#L545) | [`EdgeOperations.resolveEdgeContradictions()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/edge_operations.go#L523) | `pkg/utils/maintenance/edge_operations.go` | ✅ Implemented (2025-09-20) |

#### Recent Improvements (December 2024)

The following edge operations have been significantly improved to match the exact Python implementation:

1. **GetBetweenNodes** - Now uses proper Kuzu query pattern:
   - Uses `RelatesToNode_` intermediate nodes pattern from Python
   - Implements bidirectional search with `UNION` clause
   - Direct database query with `ExecuteQuery` instead of generic search
   - Added `convertRecordToEdge` helper for consistent result processing

2. **FilterExistingDuplicateOfEdges** - Exact Python implementation:
   - Uses `UNWIND` for batch parameter processing
   - Matches Python's parameter structure with `src`/`dst` mapping
   - Proper Kuzu query: `MATCH (n:Entity)-[:RELATES_TO]->(e:RelatesToNode_ {name: 'IS_DUPLICATE_OF'})-[:RELATES_TO]->(m:Entity)`

3. **searchRelatedEdges** - Enhanced with hybrid search:
   - Implements UUID filtering equivalent to Python's `SearchFilters(edge_uuids=...)`
   - Uses hybrid search approach similar to `EDGE_HYBRID_SEARCH_RRF`
   - Proper exclusion of the extracted edge itself
   - Group ID filtering in search operations

4. **NodePair Type** - Fixed struct definition:
   - Changed from `{Node1, Node2}` to `{Source, Target}` to match usage patterns
   - Located in `pkg/utils/maintenance/types.go`

5. **Compilation Issues** - Fixed unused variables:
   - Removed unused `maxCount` in label propagation
   - Removed unused `query` and `params` in placeholder functions
   - Removed unused `sort` import

#### LLM Client Improvements (September 2024)

The LLM client infrastructure has been significantly enhanced to match the Python Graphiti implementation:

1. **BaseOpenAIClient** - New base class for OpenAI-compatible clients:
   - Implements proper message conversion from internal format to OpenAI format
   - Provides retry logic with exponential backoff and jitter
   - Handles structured output preparation and JSON schema injection
   - Includes input cleaning (removes zero-width characters and control characters)
   - Supports multilingual extraction instructions
   - Error handling for rate limits, refusals, and API timeouts

2. **OpenAIGenericClient** - Enhanced OpenAI client with Python parity:
   - Built on BaseOpenAIClient foundation
   - Implements both regular chat and structured output methods
   - Enhanced retry logic with error feedback to the LLM
   - Automatic JSON response format enforcement for structured outputs
   - Support for custom base URLs (OpenAI-compatible services)
   - Proper error context injection for failed generations

3. **LLMConfig** - Complete configuration structure:
   - Matches Python LLMConfig exactly with APIKey, Model, BaseURL, Temperature, MaxTokens, SmallModel
   - Fluent configuration API with builder methods
   - Default values matching Python implementation
   - Separate from legacy Config for backward compatibility

4. **ModelSize Support** - Proper model selection:
   - ModelSizeSmall and ModelSizeMedium constants
   - Automatic model selection based on task complexity
   - Support for different models for different use cases

5. **Error Handling** - Comprehensive error types:
   - RateLimitError for rate limiting scenarios
   - RefusalError for LLM refusals
   - EmptyResponseError for empty responses
   - Proper error wrapping and context preservation

6. **Backward Compatibility** - Legacy support maintained:
   - Original OpenAIClient preserved for existing code
   - New implementations are additive, not breaking
   - Clear migration path to new base classes

#### Models/Edges Database Queries Port (September 2024)

The complete edge database queries module has been ported to provide exact functional parity with the Python implementation:

1. **Complete Function Coverage** - All edge database query functions ported:
   - `GetEpisodicEdgeSaveBulkQuery` - Handles bulk episodic edge creation with provider-specific logic
   - `GetEntityEdgeSaveQuery` - Single entity edge save with AOSS and provider support
   - `GetEntityEdgeSaveBulkQuery` - Bulk entity edge operations for all supported providers
   - `GetEntityEdgeReturnQuery` - Provider-specific return field mapping
   - `GetCommunityEdgeSaveQuery` - Community membership edge creation with UNION support for Kuzu

2. **Provider-Specific Implementation** - Exact match to Python logic:
   - **Neo4j**: Standard RELATES_TO relationships with vector property support
   - **FalkorDB**: Vector embeddings with vecf32() function calls
   - **Kuzu**: RelatesToNode_ intermediate pattern with bidirectional RELATES_TO
   - **Neptune**: String-based embedding storage with join() operations

3. **Query Constants** - All constants preserved:
   - `EPISODIC_EDGE_SAVE` - Standard episodic relationship creation
   - `EPISODIC_EDGE_RETURN` - Return fields for episodic edge queries
   - `COMMUNITY_EDGE_RETURN` - Return fields for community membership queries

4. **Exact Argument Matching** - Functions use identical signatures:
   - `provider driver.GraphProvider` - Database provider selection
   - `hasAOSS bool` - Amazon OpenSearch Service configuration flag
   - All parameter names and types match Python implementation

5. **Database Feature Support** - Provider-specific capabilities:
   - Vector embeddings with proper storage format per provider
   - Bulk operations with UNWIND for efficient batch processing
   - Conditional query construction based on provider capabilities
   - AOSS integration for Neo4j vector property handling

#### Models/Nodes Database Queries Port (September 2024)

The complete node database queries module has been ported to provide exact functional parity with the Python implementation:

1. **Complete Function Coverage** - All node database query functions ported:
   - `GetEpisodeNodeSaveQuery` - Episodic node creation with provider-specific entity_edges handling
   - `GetEpisodeNodeSaveBulkQuery` - Bulk episodic node operations with UNWIND patterns
   - `GetEntityNodeSaveQuery` - Entity node save with dynamic label assignment and embeddings
   - `GetEntityNodeSaveBulkQuery` - Complex bulk entity operations with multiple return types
   - `GetEntityNodeReturnQuery` - Provider-specific field selection and label handling
   - `GetCommunityNodeSaveQuery` - Community node creation with vector embedding support

2. **Provider-Specific Implementation** - Exact match to Python logic:
   - **Neo4j**: Standard node labels with vector property support via `db.create.setNodeVectorProperty`
   - **FalkorDB**: Vector embeddings with `vecf32()` function calls and explicit property mapping
   - **Kuzu**: Structured property assignment with array-based labels and attributes
   - **Neptune**: String-based embedding storage with `join()` and `removeKeyFromMap()` operations

3. **Complex Return Types** - Handling Python's mixed return scenarios:
   - `GetEntityNodeSaveBulkQuery` returns `interface{}` to handle different provider needs
   - FalkorDB/Neptune: Returns `[]QueryWithParams` or `[]string` for multiple query execution
   - Kuzu/Neo4j: Returns single query string for standard bulk processing
   - Added `QueryWithParams` struct to match Python's tuple structure

4. **Query Constants** - All provider-specific constants preserved:
   - `EPISODIC_NODE_RETURN` - Standard episodic node return fields
   - `EPISODIC_NODE_RETURN_NEPTUNE` - Neptune-specific field mapping with split operations
   - `COMMUNITY_NODE_RETURN` - Community node return fields
   - `COMMUNITY_NODE_RETURN_NEPTUNE` - Neptune community fields with float conversion

5. **Dynamic Query Construction** - Advanced features:
   - Dynamic label assignment with string splitting and formatting
   - AOSS conditional vector property handling for Neo4j
   - Label subquery generation for Neptune with proper SET operations
   - Entity edges handling with provider-specific array/string conversions

6. **Exact Argument Matching** - Functions use identical signatures:
   - `provider driver.GraphProvider` - Database provider selection
   - `labels string` - Dynamic label string for entity nodes
   - `hasAOSS bool` - Amazon OpenSearch Service configuration flag
   - `nodes []map[string]interface{}` - Complex node data for bulk operations

### utils/maintenance/graph_data_operations.py

| Python Function | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`build_indices_and_constraints()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/graph_data_operations.py#L36) | [`GraphDataOperations.BuildIndicesAndConstraints()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/graph_data_operations.go#L27) | `pkg/utils/maintenance/graph_data_operations.go` | ✅ Implemented (2025-09-20) |
| [`retrieve_episodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/graph_data_operations.py#L122) | [`GraphDataOperations.RetrieveEpisodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/graph_data_operations.go#L34) | `pkg/utils/maintenance/graph_data_operations.go` | ✅ Implemented (2025-09-20) |
| [`clear_data()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/graph_data_operations.py#L93) | [`GraphDataOperations.ClearData()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/graph_data_operations.go#L100) | `pkg/utils/maintenance/graph_data_operations.go` | ✅ Implemented (2025-09-20) |

### utils/maintenance/node_operations.py

| Python Function | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`extract_nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/node_operations.py#L69) | [`NodeOperations.ExtractNodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/node_operations.go#L32) | `pkg/utils/maintenance/node_operations.go` | ✅ Implemented (2025-09-20) |
| [`resolve_extracted_nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/node_operations.py#L185) | [`NodeOperations.ResolveExtractedNodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/node_operations.go#L234) | `pkg/utils/maintenance/node_operations.go` | ✅ Implemented (2025-09-20) |
| [`extract_attributes_from_nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/node_operations.py#L321) | [`NodeOperations.ExtractAttributesFromNodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/node_operations.go#L370) | `pkg/utils/maintenance/node_operations.go` | ✅ Implemented (2025-09-20) |
| [`extract_nodes_reflexion()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/node_operations.py#L46) | [`NodeOperations.extractNodesReflexion()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/node_operations.go#L194) | `pkg/utils/maintenance/node_operations.go` | ✅ Implemented (2025-09-20) |

### utils/maintenance/temporal_operations.py

| Python Function | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`extract_edge_dates()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/temporal_operations.py#L33) | [`TemporalOperations.ExtractEdgeDates()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/temporal_operations.go#L23) | `pkg/utils/maintenance/temporal_operations.go` | ✅ Implemented (2025-09-20) |
| [`get_edge_contradictions()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/temporal_operations.py#L74) | [`TemporalOperations.GetEdgeContradictions()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/temporal_operations.go#L94) | `pkg/utils/maintenance/temporal_operations.go` | ✅ Implemented (2025-09-20) |
| [`extract_and_save_edge_dates()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/temporal_operations.py#L119) | [`TemporalOperations.ExtractAndSaveEdgeDates()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/temporal_operations.go#L140) | `pkg/utils/maintenance/temporal_operations.go` | ✅ Implemented (2025-09-20) |

### utils/maintenance/utils.py

| Python Function | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| `get_entities_and_edges` | [`MaintenanceUtils.GetEntitiesAndEdges()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/maintenance_utils.go#L20) | `pkg/utils/maintenance/maintenance_utils.go` | ✅ Implemented (2025-09-20) |

### Additional Go Maintenance Functions

| Go Method | Description | File Location |
|-----------|-------------|---------------|
| [`MaintenanceUtils.GetEntitiesByType()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/maintenance_utils.go#L42) | Retrieves entities by type | `pkg/utils/maintenance/maintenance_utils.go` |
| [`MaintenanceUtils.GetEdgesByType()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/maintenance_utils.go#L55) | Retrieves edges by type | `pkg/utils/maintenance/maintenance_utils.go` |
| [`MaintenanceUtils.GetNodesConnectedToNode()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/maintenance_utils.go#L68) | Gets connected nodes within distance | `pkg/utils/maintenance/maintenance_utils.go` |
| [`MaintenanceUtils.GetEdgesForNode()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/maintenance_utils.go#L79) | Gets all edges for a node | `pkg/utils/maintenance/maintenance_utils.go` |
| [`MaintenanceUtils.CleanupOrphanedEdges()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/maintenance_utils.go#L120) | Removes orphaned edges | `pkg/utils/maintenance/maintenance_utils.go` |
| [`MaintenanceUtils.ValidateGraphIntegrity()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/maintenance_utils.go#L154) | Validates graph integrity | `pkg/utils/maintenance/maintenance_utils.go` |
| [`TemporalOperations.ValidateEdgeTemporalConsistency()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/temporal_operations.go#L177) | Validates edge temporal consistency | `pkg/utils/maintenance/temporal_operations.go` |
| [`TemporalOperations.ApplyTemporalInvalidation()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/temporal_operations.go#L194) | Applies temporal invalidation logic | `pkg/utils/maintenance/temporal_operations.go` |
| [`TemporalOperations.GetActiveEdgesAtTime()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/temporal_operations.go#L220) | Gets edges active at specific time | `pkg/utils/maintenance/temporal_operations.go` |
| [`TemporalOperations.GetEdgeLifespan()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/maintenance/temporal_operations.go#L233) | Calculates edge lifespan | `pkg/utils/maintenance/temporal_operations.go` |


### Additional Go Helper Functions

| Go Method | Description | File Location |
|-----------|-------------|---------------|
| [`GetDefaultSearchConfig()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/helpers.go#L221) | Returns default search configuration | `pkg/search/helpers.go` |
| [`GetSearchConfigByName()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/helpers.go#L226) | Gets predefined config by name | `pkg/search/helpers.go` |
| [`ListAvailableSearchConfigs()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/helpers.go#L249) | Lists all available configurations | `pkg/search/helpers.go` |

## Telemetry

### telemetry/telemetry.py

| Python Function | Go Function | File Location | Status | Notes |
|-----------------|-------------|---------------|--------|--------|
| [`capture_event()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/telemetry/telemetry.py#L110) | `CaptureEvent()` | `pkg/telemetry/` | ❌ Missing | Telemetry not implemented |
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

| Category | Total Methods | ✅ Implemented (2025-09-20) | ⚠️ Partial | ❌ Missing | Coverage |
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
4. ~~**Edge Operations** - Edge extraction, resolution, and maintenance operations~~ ✅ **Completed**
5. ~~**Bulk Utilities** - Deduplication and bulk operations partially implemented~~ ✅ **Completed**
6. **Telemetry** - Event tracking and metrics collection missing
7. **Multiple LLM Providers** - Only OpenAI client implemented
8. **Advanced Temporal Operations** - Time-based graph operations limited

### Well-Implemented Areas

1. **Core Search** - Hybrid search with multiple methods working
2. **Database Drivers** - Neo4j and Kuzu drivers fully functional
3. **Basic Graph Operations** - Node/edge CRUD operations complete
4. **Query Building** - Database-agnostic query construction implemented
5. **Search Configuration** - Comprehensive search configs and recipes
6. **Community Operations** - Label propagation clustering and community building
7. **Prompt Templates** - Complete prompt library with all Python functions ported
8. **Utility Functions** - Comprehensive helper functions, datetime utils, validation, and bulk operations
9. **Bulk Operations** - Complete bulk processing toolkit with node/edge extraction, deduplication, and batch operations
10. **Maintenance Operations** - Complete maintenance toolkit for nodes, edges, temporal operations, and graph data management

## Last Updated

This document was last updated: 2025-01-20

*Note: This mapping reflects the current state of the go-graphiti implementation. Status may change as development continues.*
