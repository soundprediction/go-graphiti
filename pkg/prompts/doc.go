/*
Package prompts provides a comprehensive collection of prompts for knowledge graph extraction
and processing operations.

This package is a Go port of the prompts from the Python graphiti project. It includes
prompts for:

  - Extracting nodes (entities) from text, conversations, and JSON
  - Deduplicating nodes to avoid redundant entities
  - Extracting edges (relationships) between entities
  - Deduplicating edges and resolving conflicts
  - Invalidating outdated edges
  - Extracting temporal information for edges
  - Summarizing node information
  - Evaluating extraction quality

Usage:

	library := prompts.NewLibrary()
	extractNodes := library.ExtractNodes()
	
	// Use the prompt with context
	context := map[string]interface{}{
		"entity_types": entityTypes,
		"episode_content": message,
		"custom_prompt": customInstructions,
	}
	
	messages, err := extractNodes.ExtractMessage().Call(context)
	if err != nil {
		// handle error
	}

The prompts are organized into different categories with versioned implementations
to support different use cases and backwards compatibility.
*/
package prompts