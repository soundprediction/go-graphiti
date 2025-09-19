// Package maintenance provides utilities for maintaining and operating on graph edges.
//
// This package implements the missing functionality from the Python graphiti edge_operations.py module,
// providing Go implementations for:
//
// - Building episodic edges between episodes and entities
// - Creating duplicate-of edges for entity deduplication
// - Extracting edges from text using LLM
// - Resolving extracted edges against existing edges in the graph
// - Temporal edge contradiction resolution
// - Edge deduplication and semantic similarity matching
//
// The EdgeOperations struct serves as the main interface for all edge-related operations,
// requiring a graph driver, LLM client, embedder client, and prompts library.
//
// Key functions:
// - BuildEpisodicEdges: Creates MENTIONED_IN edges from episodes to entities
// - BuildDuplicateOfEdges: Creates IS_DUPLICATE_OF edges between duplicate entities
// - ExtractEdges: Uses LLM to extract relationship triples from episode content
// - ResolveExtractedEdges: Resolves new edges against existing ones, handling duplicates and contradictions
// - GetBetweenNodes: Retrieves edges between two specific nodes
//
// This implementation uses UUID7 for all generated UUIDs and follows the temporal logic
// established in the Python codebase for edge invalidation and contradiction resolution.
package maintenance
