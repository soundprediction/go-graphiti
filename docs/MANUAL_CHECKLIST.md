# Manual Verification Checklist

This checklist is designed for a human to track the manual verification of each method in the `go-graphiti` library, based on the Python to Go mapping.

## Core Graphiti Class
- [ ] `Graphiti.__init__()`
- [ ] `Graphiti.close()`
- [ ] `Graphiti.build_indices_and_constraints()`
- [ ] `Graphiti.retrieve_episodes()`
- [ ] `Graphiti.add_episode()`
- [ ] `Graphiti.add_episode_bulk()`
- [ ] `Graphiti.build_communities()`
- [ ] `Graphiti.search()`
- [ ] `Graphiti._search()`
- [ ] `Graphiti.search_()`
- [ ] `Graphiti.get_nodes_and_edges_by_episode()`
- [ ] `Graphiti.add_triplet()`
- [ ] `Graphiti.remove_episode()`

## Core Graph Queries
- [ ] `get_range_indices(provider)`
- [ ] `get_fulltext_indices(provider)`
- [ ] `get_nodes_query(name, query, limit, provider)`
- [ ] `get_relationships_query(name, limit, provider)`
- [ ] `get_vector_cosine_func_query(vec1, vec2, provider)`
- [ ] `GraphProvider`
- [ ] `NEO4J_TO_FALKORDB_MAPPING`
- [ ] `INDEX_TO_LABEL_KUZU_MAPPING`

## Search Functionality
- [ ] `Searcher` class
- [ ] `HybridSearch()`
- [ ] Search methods (cosine_similarity, bm25, bfs)
- [ ] Reranker types
- [ ] `SearchConfig` class
- [ ] `NodeSearchConfig`
- [ ] `EdgeSearchConfig`
- [ ] `EpisodeSearchConfig`
- [ ] `CommunitySearchConfig`
- [ ] `SearchResults`
- [ ] `COMBINED_HYBRID_SEARCH_RRF`
- [ ] `COMBINED_HYBRID_SEARCH_MMR`
- [ ] `COMBINED_HYBRID_SEARCH_CROSS_ENCODER`
- [ ] `EDGE_HYBRID_SEARCH_RRF`
- [ ] `EDGE_HYBRID_SEARCH_MMR`
- [ ] `EDGE_HYBRID_SEARCH_NODE_DISTANCE`
- [ ] `EDGE_HYBRID_SEARCH_EPISODE_MENTIONS`
- [ ] `EDGE_HYBRID_SEARCH_CROSS_ENCODER`
- [ ] `NODE_HYBRID_SEARCH_RRF`
- [ ] `NODE_HYBRID_SEARCH_MMR`
- [ ] `NODE_HYBRID_SEARCH_NODE_DISTANCE`
- [ ] `NODE_HYBRID_SEARCH_EPISODE_MENTIONS`
- [ ] `NODE_HYBRID_SEARCH_CROSS_ENCODER`
- [ ] `COMMUNITY_HYBRID_SEARCH_RRF`
- [ ] `COMMUNITY_HYBRID_SEARCH_MMR`
- [ ] `COMMUNITY_HYBRID_SEARCH_CROSS_ENCODER`
- [ ] `SearchFilters` class
- [ ] `ComparisonOperator` enum
- [ ] `DateFilter` class
- [ ] `node_search_filter_query_constructor()`
- [ ] `edge_search_filter_query_constructor()`
- [ ] `date_filter_query_constructor()`
- [ ] `format_edge_date_range(edge)`
- [ ] `search_results_to_context_string()`
- [ ] `calculate_cosine_similarity()`
- [ ] `fulltext_query()`
- [ ] `get_episodes_by_mentions()`
- [ ] `get_mentioned_nodes()`
- [ ] `get_communities_by_nodes()`
- [ ] `edge_fulltext_search()`
- [ ] `edge_similarity_search()`
- [ ] `edge_bfs_search()`
- [ ] `node_fulltext_search()`
- [ ] `node_similarity_search()`
- [ ] `node_bfs_search()`
- [ ] `hybrid_node_search()`
- [ ] `get_relevant_nodes()`
- [ ] `get_relevant_edges()`
- [ ] `get_relevant_schema()`
- [ ] `mmr_rerank()`
- [ ] `rrf_fuse()`

## Driver Interface
- [ ] `GraphDriver` interface
- [ ] `Neo4jDriver` class
- [ ] `FalkorDBDriver`
- [ ] `NeptuneDriver`

## Node and Edge Types
- [ ] `Node` base class
- [ ] `EntityNode`
- [ ] `EpisodicNode`
- [ ] `CommunityNode`
- [ ] `Edge` base class
- [ ] `EntityEdge`
- [ ] `EpisodicEdge`
- [ ] `CommunityEdge`
- [ ] `get_episodic_edge_save_bulk_query`
- [ ] `get_entity_edge_save_query`
- [ ] `get_entity_edge_save_bulk_query`
- [ ] `get_entity_edge_return_query`
- [ ] `get_community_edge_save_query`
- [ ] `EPISODIC_EDGE_SAVE`
- [ ] `EPISODIC_EDGE_RETURN`
- [ ] `COMMUNITY_EDGE_RETURN`
- [ ] `get_episode_node_save_query`
- [ ] `get_episode_node_save_bulk_query`
- [ ] `get_entity_node_save_query`
- [ ] `get_entity_node_save_bulk_query`
- [ ] `get_entity_node_return_query`
- [ ] `get_community_node_save_query`
- [ ] `EPISODIC_NODE_RETURN`
- [ ] `EPISODIC_NODE_RETURN_NEPTUNE`
- [ ] `COMMUNITY_NODE_RETURN`
- [ ] `COMMUNITY_NODE_RETURN_NEPTUNE`

## LLM Client Interface
- [ ] `LLMClient` abstract class
- [ ] `LLMClient.generate()`
- [ ] `LLMClient.generate_batch()`
- [ ] `LLMClient.generate_with_schema()`
- [ ] `AnthropicClient`
- [ ] `AzureOpenAIClient`
- [ ] `GeminiClient`
- [ ] `GroqClient`
- [ ] `BaseOpenAIClient`
- [ ] `OpenAIGenericClient`
- [ ] `get_token_count`
- [ ] `LLMConfig`
- [ ] `ModelSize` enum

## Embedder Client Interface
- [ ] `EmbedderClient` abstract class
- [ ] `EmbedderClient.create()`
- [ ] `EmbedderClient.create_batch()`
- [ ] `AzureOpenAIEmbedder`
- [ ] `GeminiEmbedder`
- [ ] `VoyageEmbedder`
- [ ] `EmbedderConfig`
- [ ] `EMBEDDING_DIM` constant

## Cross Encoder Interface
- [ ] `CrossEncoderClient` abstract class
- [ ] `CrossEncoderClient.rerank()`
- [ ] `BGERerankerClient`
- [ ] `GeminiRerankerClient`

## Community Operations
- [ ] `get_community_clusters()`
- [ ] `label_propagation()`
- [ ] `build_community()`
- [ ] `build_communities()`
- [ ] `remove_communities()`
- [ ] `determine_entity_community()`
- [ ] `update_community()`
- [ ] `summarize_pair()`
- [ ] `generate_summary_description()`

## Prompts and Models
- [ ] `Message` class
- [ ] `PromptFunction` type
- [ ] `ExtractedEntity`
- [ ] `ExtractedEntities`
- [ ] `EntityClassificationTriple`
- [ ] `EntitySummary`
- [ ] `Edge`
- [ ] `ExtractedEdges`
- [ ] `NodeDuplicate`
- [ ] `EdgeDuplicate`
- [ ] `InvalidatedEdges`
- [ ] `EdgeDates`
- [ ] `Summary`
- [ ] `SummaryDescription`
- [ ] `QueryExpansion`
- [ ] `QAResponse`
- [ ] `EvalResponse`
- [ ] `prompts/extract_nodes.py`
- [ ] `prompts/extract_edges.py`
- [ ] `prompts/dedupe_nodes.py`
- [ ] `prompts/dedupe_edges.py`
- [ ] `prompts/summarize_nodes.py`
- [ ] `prompts/invalidate_edges.py`
- [ ] `prompts/extract_edge_dates.py`
- [ ] `prompts/eval.py`
- [ ] `extract_message()`
- [ ] `extract_json()`
- [ ] `extract_text()`
- [ ] `reflexion()`
- [ ] `classify_nodes()`
- [ ] `extract_attributes()`
- [ ] `extract_summary()`
- [ ] `edge()`
- [ ] `reflexion()`
- [ ] `extract_attributes()`
- [ ] `node()`
- [ ] `node_list()`
- [ ] `nodes()`
- [ ] `edge()`
- [ ] `edge_list()`
- [ ] `resolve_edge()`
- [ ] `summarize_pair()`
- [ ] `summarize_context()`
- [ ] `summary_description()`
- [ ] `v1()`
- [ ] `v2()`
- [ ] `v1()`
- [ ] `qa_prompt()`
- [ ] `eval_prompt()`
- [ ] `query_expansion()`
- [ ] `eval_add_episode_results()`
- [ ] `PromptLibrary` interface
- [ ] `PromptLibraryImpl`
- [ ] `prompt_library` instance
- [ ] `to_prompt_json()`
- [ ] `DO_NOT_ESCAPE_UNICODE`

## Utilities and Helpers
- [ ] `get_default_group_id()`
- [ ] `semaphore_gather()`
- [ ] `validate_excluded_entity_types()`
- [ ] `validate_group_id()`
- [ ] `lucene_sanitize()`
- [ ] `normalize_l2()`
- [ ] `parse_db_date()`
- [ ] `add_nodes_and_edges_bulk()`
- [ ] `dedupe_edges_bulk()`
- [ ] `dedupe_nodes_bulk()`
- [ ] `extract_nodes_and_edges_bulk()`
- [ ] `resolve_edge_pointers()`
- [ ] `retrieve_previous_episodes_bulk()`
- [ ] `compress_uuid_map()`
- [ ] `UnionFind` class
- [ ] `utc_now()`
- [ ] `ensure_utc()`
- [ ] `convert_datetimes_to_strings()`
- [ ] `validate_entity_types()`
- [ ] `build_indices_and_constraints`
- [ ] `retrieve_episodes`
- [ ] `clear_data`
- [ ] `extract_nodes`
- [ ] `resolve_extracted_nodes`
- [ ] `extract_attributes_from_nodes`
- [ ] `extract_nodes_reflexion`
- [ ] `extract_edge_dates`
- [ ] `get_edge_contradictions`
- [ ] `extract_and_save_edge_dates`
- [ ] `get_entities_and_edges`

## Telemetry
- [ ] `capture_event()`
- [ ] `TelemetryEvent` class
