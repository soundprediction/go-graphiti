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
| [`Graphiti.__init__()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L169) | [`NewClient()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L88) | `graphiti.go` | ✅ Implemented | Go uses functional construction pattern |
| [`Graphiti.close()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L368) | [`Client.Close()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1300) | `graphiti.go` | ✅ Implemented | |
| [`Graphiti.build_indices_and_constraints()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L401) | [`Client.CreateIndices()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1296) | `graphiti.go` | ✅ Implemented | |
| [`Graphiti.retrieve_episodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L440) | [`Client.GetEpisodes()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1253) | `graphiti.go` | ✅ Implemented | |
| [`Graphiti.add_episode()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L477) | [`Client.Add()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L115) | `graphiti.go` | ✅ Implemented | Go method accepts multiple episodes |
| [`Graphiti.add_episode_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L650) | [`Client.Add()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L115) | `graphiti.go` | ✅ Implemented | Same as single episode in Go |
| [`Graphiti.build_communities()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L886) | [`Builder.BuildCommunities()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L121) | `pkg/community/community.go` | ✅ Implemented | Community building with label propagation |
| [`Graphiti.search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L920) | [`Client.Search()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1181) | `graphiti.go` | ✅ Implemented | |
| [`Graphiti._search()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L971) | `Client.Search()` internal | `graphiti.go` | ✅ Implemented | Merged into main Search method |
| [`Graphiti.search_()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L980) | [`searcher.Search()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L142) | `pkg/search/search.go` | ✅ Implemented | Direct searcher access |
| [`Graphiti.get_nodes_and_edges_by_episode()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L1000) | [`Client.GetNode()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1248) / [`Client.GetEdge()`](https://github.com/soundprediction/go-graphiti/blob/main/graphiti.go#L1252) | `graphiti.go` | ⚠️ Partial | No bulk episode-based retrieval |
| [`Graphiti.add_triplet()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L1012) | `Client.addTriplet()` | `graphiti.go` | ❌ Missing | Direct triplet addition not implemented |
| [`Graphiti.remove_episode()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L1061) | `Client.removeEpisode()` | `graphiti.go` | ❌ Missing | Episode removal not implemented |

### Result Types

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| [`AddEpisodeResults`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L133) | [`AddEpisodeResults`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L273) | `pkg/types/types.go` | ✅ Implemented |
| [`AddBulkEpisodeResults`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L141) | [`AddBulkEpisodeResults`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L286) | `pkg/types/types.go` | ✅ Implemented |
| [`AddTripletResults`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graphiti.py#L149) | [`AddTripletResults`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L299) | `pkg/types/types.go` | ✅ Implemented |

### Additional Go Result Types

| Go Type | Description | File Location |
|---------|-------------|---------------|
| [`EpisodeProcessingResult`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L305) | Internal episode processing result | `pkg/types/types.go` |
| [`BulkEpisodeResults`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L318) | Bulk episode processing statistics | `pkg/types/types.go` |
| [`TripletResults`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L327) | Enhanced triplet operation result | `pkg/types/types.go` |

## Core Graph Queries

### graph_queries.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`get_range_indices(provider)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L30) | [`GetRangeIndices(provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L40) | `pkg/driver/graph_queries.go` | ✅ Implemented |
| [`get_fulltext_indices(provider)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L80) | [`GetFulltextIndices(provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L100) | `pkg/driver/graph_queries.go` | ✅ Implemented |
| [`get_nodes_query(name, query, limit, provider)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L108) | [`GetNodesQuery(indexName, query string, limit int, provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L136) | `pkg/driver/graph_queries.go` | ✅ Implemented |
| [`get_relationships_query(name, limit, provider)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L128) | [`GetRelationshipsQuery(indexName string, limit int, provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L152) | `pkg/driver/graph_queries.go` | ✅ Implemented |
| [`get_vector_cosine_func_query(vec1, vec2, provider)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L119) | [`GetVectorCosineFuncQuery(vec1, vec2 string, provider GraphProvider)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L168) | `pkg/driver/graph_queries.go` | ✅ Implemented |
| `GraphProvider` enum | `GraphProvider` type | `pkg/driver/graph_queries.go` | ✅ Implemented |
| [`NEO4J_TO_FALKORDB_MAPPING`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L10) | [`neo4jToFalkorDBMapping`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L21) | `pkg/driver/graph_queries.go` | ✅ Implemented |
| [`INDEX_TO_LABEL_KUZU_MAPPING`](https://github.com/getzep/graphiti/blob/main/graphiti_core/graph_queries.py#L17) | [`indexToLabelKuzuMapping`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/graph_queries.go#L29) | `pkg/driver/graph_queries.go` | ✅ Implemented |

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
| `Searcher` class | [`Searcher`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L118) struct | `pkg/search/search.go` | ✅ Implemented |
| [`HybridSearch()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search.py#L70) | [`HybridSearch()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L142) | `pkg/search/search.go` | ✅ Implemented |
| Search methods (cosine_similarity, bm25, bfs) | [`SearchMethod`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L15) constants | `pkg/search/search.go` | ✅ Implemented |
| Reranker types | [`RerankerType`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L22) constants | `pkg/search/search.go` | ✅ Implemented |

### search/search_config.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`SearchConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L130) class | [`SearchConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L35) struct | `pkg/search/search.go` | ✅ Implemented |
| [`NodeSearchConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L108) | [`NodeSearchConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L42) struct | `pkg/search/search.go` | ✅ Implemented |
| [`EdgeSearchConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L99) | [`EdgeSearchConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L50) struct | `pkg/search/search.go` | ✅ Implemented |
| [`EpisodeSearchConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L117) | [`EpisodeSearchConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L58) struct | `pkg/search/search.go` | ✅ Implemented |
| [`CommunitySearchConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L124) | [`CommunitySearchConfig`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L64) struct | `pkg/search/search.go` | ✅ Implemented |
| [`SearchResults`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config.py#L137) | [`HybridSearchResult`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L95) struct | `pkg/search/search.go` | ✅ Implemented |

### search/search_config_recipes.py

| Python Configuration | Go Configuration | File Location | Status |
|----------------------|------------------|---------------|--------|
| [`COMBINED_HYBRID_SEARCH_RRF`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L27) | [`CombinedHybridSearchRRF`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L3) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`COMBINED_HYBRID_SEARCH_MMR`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L50) | [`CombinedHybridSearchMMR`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L25) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`COMBINED_HYBRID_SEARCH_CROSS_ENCODER`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L74) | [`CombinedHybridSearchCrossEncoder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L50) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`EDGE_HYBRID_SEARCH_RRF`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L102) | [`EdgeHybridSearchRRF`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L72) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`EDGE_HYBRID_SEARCH_MMR`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L109) | [`EdgeHybridSearchMMR`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L79) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`EDGE_HYBRID_SEARCH_NODE_DISTANCE`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L116) | [`EdgeHybridSearchNodeDistance`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L86) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`EDGE_HYBRID_SEARCH_EPISODE_MENTIONS`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L123) | [`EdgeHybridSearchEpisodeMentions`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L93) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`EDGE_HYBRID_SEARCH_CROSS_ENCODER`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L130) | [`EdgeHybridSearchCrossEncoder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L100) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`NODE_HYBRID_SEARCH_RRF`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L141) | [`NodeHybridSearchRRF`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L108) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`NODE_HYBRID_SEARCH_MMR`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L148) | [`NodeHybridSearchMMR`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L115) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`NODE_HYBRID_SEARCH_NODE_DISTANCE`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L155) | [`NodeHybridSearchNodeDistance`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L122) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`NODE_HYBRID_SEARCH_EPISODE_MENTIONS`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L162) | [`NodeHybridSearchEpisodeMentions`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L129) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`NODE_HYBRID_SEARCH_CROSS_ENCODER`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L169) | [`NodeHybridSearchCrossEncoder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L136) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`COMMUNITY_HYBRID_SEARCH_RRF`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L180) | [`CommunityHybridSearchRRF`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L144) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`COMMUNITY_HYBRID_SEARCH_MMR`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L187) | [`CommunityHybridSearchMMR`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L151) | `pkg/search/config_recipes.go` | ✅ Implemented |
| [`COMMUNITY_HYBRID_SEARCH_CROSS_ENCODER`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_config_recipes.py#L194) | [`CommunityHybridSearchCrossEncoder`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/config_recipes.go#L158) | `pkg/search/config_recipes.go` | ✅ Implemented |

### search/search_filters.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`SearchFilters`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L40) class | [`SearchFilters`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search.go#L70) struct | `pkg/search/search.go` | ✅ Implemented |
| [`ComparisonOperator`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L19) enum | [`ComparisonOperator`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/filters.go#L11) type | `pkg/search/filters.go` | ✅ Implemented |
| [`DateFilter`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L33) class | [`DateFilter`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/filters.go#L25) struct | `pkg/search/filters.go` | ✅ Implemented |
| [`node_search_filter_query_constructor()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L54) | [`NodeSearchFilterQueryConstructor()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/filters.go#L64) | `pkg/search/filters.go` | ✅ Implemented |
| [`edge_search_filter_query_constructor()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L95) | [`EdgeSearchFilterQueryConstructor()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/filters.go#L110) | `pkg/search/filters.go` | ✅ Implemented |
| [`date_filter_query_constructor()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_filters.py#L75) | [`constructDateFilterQuery()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/filters.go#L211) | `pkg/search/filters.go` | ✅ Implemented |

### search/search_helpers.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`format_edge_date_range(edge)`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_helpers.py#L21) | [`FormatEdgeDateRange(edge *types.Edge)`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/helpers.go#L12) | `pkg/search/helpers.go` | ✅ Implemented |
| [`search_results_to_context_string()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_helpers.py#L26) | [`SearchResultsToContextString()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/helpers.go#L27) | `pkg/search/helpers.go` | ✅ Implemented |

### search/search_utils.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| Various search utility functions | `SearchUtilities` struct methods | `pkg/search/search_utils.go` | ✅ Implemented |
| [`Cosine similarity calculation`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L61) | [`CalculateCosineSimilarity()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/search_utils.go#L40) | `pkg/search/search_utils.go` | ✅ Implemented |
| [`RRF reranking`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L1433) | [`RRF()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/rerankers.go#L25) | `pkg/search/rerankers.go` | ✅ Implemented |
| [`MMR reranking`](https://github.com/getzep/graphiti/blob/main/graphiti_core/search/search_utils.py#L1501) | [`MMR()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/search/rerankers.go#L231) | `pkg/search/rerankers.go` | ✅ Implemented |

## Driver Interface

### driver/driver.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`GraphDriver`](https://github.com/getzep/graphiti/blob/main/graphiti_core/driver/driver.py#L63) interface | [`GraphDriver`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/driver.go#L12) interface | `pkg/driver/driver.go` | ✅ Implemented |
| Database operations (GetNode, UpsertNode, etc.) | Same method names | `pkg/driver/driver.go` | ✅ Implemented |

### driver/neo4j.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`Neo4jDriver`](https://github.com/getzep/graphiti/blob/main/graphiti_core/driver/neo4j_driver.py#L28) class | [`Neo4jDriver`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/neo4j.go#L19) struct | `pkg/driver/neo4j.go` | ✅ Implemented |
| All GraphDriver interface methods | Same method names | `pkg/driver/neo4j.go` | ✅ Implemented |

### driver/kuzu.py

| Python Method | Go Method | File Location | Status |
|---------------|-----------|---------------|--------|
| [`KuzuDriver`](https://github.com/getzep/graphiti/blob/main/graphiti_core/driver/kuzu_driver.py#L91) class | [`KuzuDriver`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/kuzu.go#L16) struct | `pkg/driver/kuzu.go` | ✅ Implemented |
| All GraphDriver interface methods | Same method names | `pkg/driver/kuzu.go` | ✅ Implemented |
| [`setupSchema()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/driver/kuzu_driver.py#L140) | [`setupSchema()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/driver/kuzu.go#L158) method | `pkg/driver/kuzu.go` | ✅ Implemented |

## Node and Edge Types

### nodes.py / edges.py

| Python Type | Go Type | File Location | Status | Notes |
|-------------|---------|---------------|--------|--------|
| [`Node`](https://github.com/getzep/graphiti/blob/main/graphiti_core/nodes.py#L88) base class | [`types.Node`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L8) struct | `pkg/types/types.go` | ✅ Implemented | Single struct for all node types |
| [`EntityNode`](https://github.com/getzep/graphiti/blob/main/graphiti_core/nodes.py#L513) | `types.Node` with `Type: EntityNodeType` | `pkg/types/types.go` | ✅ Implemented | |
| [`EpisodicNode`](https://github.com/getzep/graphiti/blob/main/graphiti_core/nodes.py#L321) | `types.Node` with `Type: EpisodicNodeType` | `pkg/types/types.go` | ✅ Implemented | |
| [`CommunityNode`](https://github.com/getzep/graphiti/blob/main/graphiti_core/nodes.py#L653) | `types.Node` with `Type: CommunityNodeType` | `pkg/types/types.go` | ✅ Implemented | |
| [`Edge`](https://github.com/getzep/graphiti/blob/main/graphiti_core/edges.py#L40) base class | [`types.Edge`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/types/types.go#L41) struct | `pkg/types/types.go` | ✅ Implemented | Single struct for all edge types |
| [`EntityEdge`](https://github.com/getzep/graphiti/blob/main/graphiti_core/edges.py#L241) | `types.Edge` with `Type: EntityEdgeType` | `pkg/types/types.go` | ✅ Implemented | |
| [`EpisodicEdge`](https://github.com/getzep/graphiti/blob/main/graphiti_core/edges.py#L141) | `types.Edge` with `Type: EpisodicEdgeType` | `pkg/types/types.go` | ✅ Implemented | |
| [`CommunityEdge`](https://github.com/getzep/graphiti/blob/main/graphiti_core/edges.py#L517) | `types.Edge` with `Type: CommunityEdgeType` | `pkg/types/types.go` | ✅ Implemented | |

### Node and Edge Functions

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| [`create_entity_node_embeddings()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/nodes.py#L853) | `EmbedNodeContent()` | `pkg/embedder/` | ⚠️ Partial |
| [`create_entity_edge_embeddings()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/edges.py#L663) | `EmbedEdgeContent()` | `pkg/embedder/` | ⚠️ Partial |
| [`get_entity_node_from_record()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/nodes.py#L808) | `NodeFromDBRecord()` | `pkg/driver/` | ✅ Implemented |
| [`get_entity_edge_from_record()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/edges.py#L608) | `EdgeFromDBRecord()` | `pkg/driver/` | ✅ Implemented |

## LLM Client Interface

### llm_client/client.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|--------|
| [`LLMClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/client.py#L53) abstract class | [`llm.Client`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/client.go#L8) interface | `pkg/llm/client.go` | ✅ Implemented | |
| [`LLMClient.generate()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/client.py#L165) | [`Client.Chat()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/client.go#L10) | `pkg/llm/client.go` | ✅ Implemented | |
| `LLMClient.generate_batch()` | `Client.ChatBatch()` | `pkg/llm/` | ❌ Missing | Batch operations not implemented |
| [`LLMClient.generate_with_schema()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/client.py#L165) | [`Client.ChatWithStructuredOutput()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/client.go#L13) | `pkg/llm/client.go` | ✅ Implemented | |

### LLM Client Implementations

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`OpenAIClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/openai_client.py#L25) | [`openai.Client`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/openai.go#L12) | `pkg/llm/openai/` | ✅ Implemented |
| `AnthropicClient` | `anthropic.Client` | `pkg/llm/anthropic/` | ❌ Missing |
| `GeminiClient` | `gemini.Client` | `pkg/llm/gemini/` | ❌ Missing |
| `GroqClient` | `groq.Client` | `pkg/llm/groq/` | ❌ Missing |
| `AzureOpenAIClient` | `azure.Client` | `pkg/llm/azure/` | ❌ Missing |

### LLM Configuration

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| [`LLMConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/config.py#L21) | [`llm.Config`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/llm/client.go#L65) | `pkg/llm/config.go` | ✅ Implemented |
| [`ModelSize`](https://github.com/getzep/graphiti/blob/main/graphiti_core/llm_client/config.py#L15) enum | `ModelSize` constants | `pkg/llm/config.go` | ✅ Implemented |

## Embedder Client Interface

### embedder/client.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|--------|
| [`EmbedderClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/embedder/client.py#L22) abstract class | [`embedder.Client`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/client.go#L8) interface | `pkg/embedder/client.go` | ✅ Implemented | |
| [`EmbedderClient.create()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/embedder/client.py#L24) | [`Client.EmbedSingle()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/client.go#L13) | `pkg/embedder/client.go` | ✅ Implemented | |
| [`EmbedderClient.create_batch()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/embedder/client.py#L29) | [`Client.Embed()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/client.go#L10) | `pkg/embedder/client.go` | ✅ Implemented | |

### Embedder Implementations

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`OpenAIEmbedder`](https://github.com/getzep/graphiti/blob/main/graphiti_core/embedder/openai.py#L31) | [`openai.EmbedderClient`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/openai.go#L12) | `pkg/embedder/openai/` | ✅ Implemented |
| `VoyageEmbedder` | `voyage.Client` | `pkg/embedder/voyage/` | ❌ Missing |
| `GeminiEmbedder` | `gemini.Client` | `pkg/embedder/gemini/` | ❌ Missing |
| `AzureOpenAIEmbedder` | `azure.Client` | `pkg/embedder/azure/` | ❌ Missing |

### Embedder Configuration

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| [`EmbedderConfig`](https://github.com/getzep/graphiti/blob/main/graphiti_core/embedder/client.py#L17) | [`embedder.Config`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/embedder/client.go#L22) | `pkg/embedder/client.go` | ✅ Implemented |
| `EMBEDDING_DIM` constant | `DefaultDimensions` | `pkg/embedder/client.go` | ✅ Implemented |

## Cross Encoder Interface

### cross_encoder/client.py

| Python Method | Go Method | File Location | Status | Notes |
|---------------|-----------|---------------|--------|--------|
| [`CrossEncoderClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/cross_encoder/client.py#L18) abstract class | [`crossencoder.Client`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/crossencoder/client.go#L13) interface | `pkg/crossencoder/` | ❌ Missing | Cross encoder not implemented |
| [`CrossEncoderClient.rerank()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/cross_encoder/client.py#L25) | [`Client.Rerank()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/crossencoder/reranker.go#L203) | `pkg/crossencoder/` | ❌ Missing | |

### Cross Encoder Implementations

| Python Class | Go Implementation | File Location | Status |
|--------------|-------------------|---------------|--------|
| [`OpenAIRerankerClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/cross_encoder/openai_reranker_client.py#L21) | N/A | N/A | ❌ Missing |
| [`BGERerankerClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/cross_encoder/bge_reranker_client.py#L21) | N/A | N/A | ❌ Missing |
| [`GeminiRerankerClient`](https://github.com/getzep/graphiti/blob/main/graphiti_core/cross_encoder/gemini_reranker_client.py#L29) | N/A | N/A | ❌ Missing |

## Community Operations

### utils/maintenance/community_operations.py

| Python Function | Go Method | File Location | Status | Notes |
|-----------------|-----------|---------------|--------|--------|
| [`get_community_clusters()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L33) | [`Builder.GetCommunityClusters()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L49) | `pkg/community/community.go` | ✅ Implemented | |
| [`label_propagation()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L100) | [`Builder.labelPropagation()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/label_propagation.go#L11) | `pkg/community/label_propagation.go` | ✅ Implemented | |
| [`build_community()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L201) | [`Builder.buildCommunity()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L181) | `pkg/community/community.go` | ✅ Implemented | |
| [`build_communities()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L233) | [`Builder.BuildCommunities()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L121) | `pkg/community/community.go` | ✅ Implemented | |
| [`remove_communities()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L256) | [`Builder.RemoveCommunities()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L461) | `pkg/community/community.go` | ✅ Implemented | |
| [`determine_entity_community()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L264) | [`Builder.DetermineEntityCommunity()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/update.go#L23) | `pkg/community/update.go` | ✅ Implemented | |
| [`update_community()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L311) | [`Builder.UpdateCommunity()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/update.go#L60) | `pkg/community/update.go` | ✅ Implemented | |
| [`summarize_pair()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L158) | [`Builder.summarizePair()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L351) | `pkg/community/community.go` | ✅ Implemented | |
| [`generate_summary_description()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L179) | [`Builder.generateCommunityName()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L380) | `pkg/community/community.go` | ✅ Implemented | |

### Community Types and Models

| Python Type | Go Type | File Location | Status |
|-------------|---------|---------------|--------|
| [`Neighbor`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/maintenance/community_operations.py#L27) class | [`Neighbor`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L38) struct | `pkg/community/community.go` | ✅ Implemented |
| `BuildCommunitiesResult` | [`BuildCommunitiesResult`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/community.go#L44) struct | `pkg/community/community.go` | ✅ Implemented |
| `DetermineEntityCommunityResult` | [`DetermineEntityCommunityResult`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/update.go#L11) struct | `pkg/community/update.go` | ✅ Implemented |
| `UpdateCommunityResult` | [`UpdateCommunityResult`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/community/update.go#L17) struct | `pkg/community/update.go` | ✅ Implemented |

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
| [`Message`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/models.py#L18) class | `Message` struct | `pkg/prompts/models.go` | ✅ Implemented |
| `PromptFunction` type | `PromptFunction` type | `pkg/prompts/types.go` | ✅ Implemented |
| [`ExtractedEntity`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L25) | [`ExtractedEntity`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L5) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`ExtractedEntities`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L30) | [`ExtractedEntities`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L11) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`EntityClassificationTriple`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L45) | [`EntityClassificationTriple`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L22) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`EntitySummary`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L84) | [`EntitySummary`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L34) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`Edge`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py#L20) | [`Edge`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L39) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`ExtractedEdges`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py#L28) | [`ExtractedEdges`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L49) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`NodeDuplicate`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_nodes.py#L19) | [`NodeDuplicate`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L60) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`EdgeDuplicate`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_edges.py#L19) | [`EdgeDuplicate`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L73) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`InvalidatedEdges`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/invalidate_edges.py#L19) | [`InvalidatedEdges`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L85) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`EdgeDates`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edge_dates.py#L19) | [`EdgeDates`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L90) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`Summary`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py#L19) | [`Summary`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L96) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`SummaryDescription`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py#L23) | [`SummaryDescription`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L101) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`QueryExpansion`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L24) | [`QueryExpansion`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L106) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`QAResponse`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L28) | [`QAResponse`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L111) struct | `pkg/prompts/models.go` | ✅ Implemented |
| [`EvalResponse`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L32) | [`EvalResponse`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/models.go#L116) struct | `pkg/prompts/models.go` | ✅ Implemented |

### Prompt Templates

| Python Module | Go Implementation | File Location | Status | Notes |
|---------------|-------------------|---------------|--------|--------|
| [`prompts/extract_nodes.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py) | `ExtractNodesPrompt` interface | `pkg/prompts/extract_nodes.go` | ✅ Implemented | All 7 functions implemented |
| [`prompts/extract_edges.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py) | `ExtractEdgesPrompt` interface | `pkg/prompts/extract_edges.go` | ✅ Implemented | All 3 functions implemented |
| [`prompts/dedupe_nodes.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_nodes.py) | `DedupeNodesPrompt` interface | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented | All 3 functions implemented |
| [`prompts/dedupe_edges.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_edges.py) | `DedupeEdgesPrompt` interface | `pkg/prompts/dedupe_edges.go` | ✅ Implemented | All 3 functions implemented |
| [`prompts/summarize_nodes.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py) | `SummarizeNodesPrompt` interface | `pkg/prompts/summarize_nodes.go` | ✅ Implemented | All 3 functions implemented |
| [`prompts/invalidate_edges.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/invalidate_edges.py) | `InvalidateEdgesPrompt` interface | `pkg/prompts/invalidate_edges.go` | ✅ Implemented | Both v1 and v2 functions |
| [`prompts/extract_edge_dates.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edge_dates.py) | `ExtractEdgeDatesPrompt` interface | `pkg/prompts/extract_edge_dates.go` | ✅ Implemented | v1 function implemented |
| [`prompts/eval.py`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py) | `EvalPrompt` interface | `pkg/prompts/eval.go` | ✅ Implemented | All 4 functions implemented |

### Extract Nodes Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`extract_message()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L91) | [`ExtractMessage()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L41) | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| [`extract_json()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L142) | [`ExtractJSON()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L111) | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| [`extract_text()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L171) | [`ExtractText()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L141) | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| [`reflexion()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L198) | [`Reflexion()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L171) | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| [`classify_nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L221) | [`ClassifyNodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L204) | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| [`extract_attributes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L252) | [`ExtractAttributes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L240) | `pkg/prompts/extract_nodes.go` | ✅ Implemented |
| [`extract_summary()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_nodes.py#L281) | [`ExtractSummary()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_nodes.go#L281) | `pkg/prompts/extract_nodes.go` | ✅ Implemented |

### Extract Edges Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`edge()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py#L57) | [`Edge()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_edges.go#L27) | `pkg/prompts/extract_edges.go` | ✅ Implemented |
| [`reflexion()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py#L130) | [`Reflexion()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_edges.go#L100) | `pkg/prompts/extract_edges.go` | ✅ Implemented |
| [`extract_attributes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edges.py#L156) | [`ExtractAttributes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_edges.go#L133) | `pkg/prompts/extract_edges.go` | ✅ Implemented |

### Dedupe Nodes Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`node()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_nodes.py#L51) | [`Node()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_nodes.go#L27) | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented |
| [`node_list()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_nodes.py#L180) | [`NodeList()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_nodes.go#L190) | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented |
| [`nodes()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_nodes.py#L109) | [`Nodes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_nodes.go#L103) | `pkg/prompts/dedupe_nodes.go` | ✅ Implemented |

### Dedupe Edges Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`edge()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_edges.py#L53) | [`Edge()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_edges.go#L27) | `pkg/prompts/dedupe_edges.go` | ✅ Implemented |
| [`edge_list()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_edges.py#L94) | [`EdgeList()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_edges.go#L90) | `pkg/prompts/dedupe_edges.go` | ✅ Implemented |
| [`resolve_edge()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/dedupe_edges.py#L125) | [`ResolveEdge()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/dedupe_edges.go#L123) | `pkg/prompts/dedupe_edges.go` | ✅ Implemented |

### Summarize Nodes Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`summarize_pair()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py#L43) | [`SummarizePair()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/summarize_nodes.go#L27) | `pkg/prompts/summarize_nodes.go` | ✅ Implemented |
| [`summarize_context()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py#L64) | [`SummarizeContext()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/summarize_nodes.go#L60) | `pkg/prompts/summarize_nodes.go` | ✅ Implemented |
| [`summary_description()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/summarize_nodes.py#L109) | [`SummaryDescription()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/summarize_nodes.go#L115) | `pkg/prompts/summarize_nodes.go` | ✅ Implemented |

### Invalidate Edges Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`v1()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/invalidate_edges.py#L35) | [`V1()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/invalidate_edges.go#L20) | `pkg/prompts/invalidate_edges.go` | ✅ Implemented |
| [`v2()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/invalidate_edges.py#L68) | `V2()` | `pkg/prompts/invalidate_edges.go` | ✅ Implemented |

### Extract Edge Dates Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`v1()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/extract_edge_dates.py#L33) | [`V1()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/extract_edge_dates.go#L19) | `pkg/prompts/extract_edge_dates.go` | ✅ Implemented |

### Eval Functions

| Python Function | Go Method | File Location | Status |
|-----------------|-----------|---------------|--------|
| [`qa_prompt()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L88) | [`QAPrompt()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/eval.go#L65) | `pkg/prompts/eval.go` | ✅ Implemented |
| [`eval_prompt()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L111) | [`EvalPrompt()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/eval.go#L106) | `pkg/prompts/eval.go` | ✅ Implemented |
| [`query_expansion()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L65) | [`QueryExpansion()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/eval.go#L29) | `pkg/prompts/eval.go` | ✅ Implemented |
| [`eval_add_episode_results()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/eval.py#L132) | [`EvalAddEpisodeResults()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/eval.go#L130) | `pkg/prompts/eval.go` | ✅ Implemented |

### Prompt Library

| Python Component | Go Component | File Location | Status |
|------------------|--------------|---------------|--------|
| [`PromptLibrary`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/lib.py#L48) interface | [`Library`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/library.go#L4) interface | `pkg/prompts/library.go` | ✅ Implemented |
| [`PromptLibraryImpl`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/lib.py#L59) | [`LibraryImpl`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/library.go#L16) struct | `pkg/prompts/library.go` | ✅ Implemented |
| `prompt_library` instance | [`NewLibrary()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/library.go#L49) function | `pkg/prompts/library.go` | ✅ Implemented |

### Prompt Helpers

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| [`to_prompt_json()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/prompts/prompt_helpers.py#L7) | [`ToPromptJSON()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/prompts/types.go#L47) | `pkg/prompts/types.go` | ✅ Implemented |
| `DO_NOT_ESCAPE_UNICODE` | `DoNotEscapeUnicode` const | `pkg/prompts/models.go` | ✅ Implemented |

## Utilities and Helpers

### helpers.py (graphiti_core/)

| Python Function | Go Function | File Location | Status | Notes |
|-----------------|-------------|---------------|--------|--------|
| [`get_default_group_id()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L50) | [`GetDefaultGroupID()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L111) | `pkg/utils/helpers.go` | ✅ Implemented | |
| [`semaphore_gather()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L118) | [`SemaphoreGather()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/concurrent.go#L102) | `pkg/utils/concurrent.go` | ✅ Implemented | |
| [`validate_excluded_entity_types()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L158) | [`ValidateExcludedEntityTypes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L248) | `pkg/utils/helpers.go` | ✅ Implemented | |
| [`validate_group_id()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L132) | [`ValidateGroupID()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L228) | `pkg/utils/helpers.go` | ✅ Implemented | |
| [`lucene_sanitize()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L61) | [`LuceneSanitize()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L118) | `pkg/utils/helpers.go` | ✅ Implemented | |
| [`normalize_l2()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L109) | [`NormalizeL2()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L154) / `NormalizeL2Float32()` | `pkg/utils/helpers.go` | ✅ Implemented | |
| [`parse_db_date()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/helpers.py#L38) | [`ParseDBDate()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/helpers.go#L78) | `pkg/utils/helpers.go` | ✅ Implemented | |

### utils/bulk_utils.py

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| [`add_nodes_and_edges_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L78) | `Client.Add()` | `graphiti.go` | ⚠️ Partial |
| [`dedupe_edges_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L383) | Helper functions | `pkg/utils/bulk.go` | ✅ Implemented |
| [`dedupe_nodes_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L281) | Helper functions | `pkg/utils/bulk.go` | ✅ Implemented |
| [`extract_nodes_and_edges_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L219) | Embedded in `Client.Add()` | `graphiti.go` | ⚠️ Partial |
| [`resolve_edge_pointers()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L548) | [`ResolveEdgePointers()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L114) | `pkg/utils/bulk.go` | ✅ Implemented |
| [`retrieve_previous_episodes_bulk()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L45) | `GetEpisodes()` | `graphiti.go` | ⚠️ Partial |
| [`compress_uuid_map()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L520) | [`CompressUUIDMap()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L61) | `pkg/utils/bulk.go` | ✅ Implemented |
| [`UnionFind`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/bulk_utils.py#L500) class | [`UnionFind`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/bulk.go#L15) struct | `pkg/utils/bulk.go` | ✅ Implemented |

### utils/datetime_utils.py

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| [`utc_now()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/datetime_utils.py#L18) | [`UTCNow()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/datetime.go#L9) | `pkg/utils/datetime.go` | ✅ Implemented |
| [`ensure_utc()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/datetime_utils.py#L22) | [`EnsureUTC()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/datetime.go#L14) | `pkg/utils/datetime.go` | ✅ Implemented |
| [`convert_datetimes_to_strings()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/datetime_utils.py#L37) | [`ConvertDatetimesToStrings()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/datetime.go#L29) | `pkg/utils/datetime.go` | ✅ Implemented |

### utils/ontology_utils/entity_types_utils.py

| Python Function | Go Function | File Location | Status |
|-----------------|-------------|---------------|--------|
| [`validate_entity_types()`](https://github.com/getzep/graphiti/blob/main/graphiti_core/utils/ontology_utils/entity_types_utils.py#L16) | [`ValidateEntityTypes()`](https://github.com/soundprediction/go-graphiti/blob/main/pkg/utils/validation.go#L16) | `pkg/utils/validation.go` | ✅ Implemented |

### Additional Go Utility Functions

| Go Function | Description | File Location |
|-------------|-------------|---------------|
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