// Package graphiti provides a temporal knowledge graph library for Go.
//
// Graphiti is designed for building temporally-aware knowledge graphs for AI agents,
// enabling real-time incremental updates without batch recomputation. It supports
// hybrid search combining semantic embeddings, keyword search, and graph traversal.
//
// # Basic Usage
//
// Create a new Graphiti client with the required components:
//
//	// Create Neo4j driver
//	driver, err := driver.NewNeo4jDriver("bolt://localhost:7687", "neo4j", "password", "neo4j")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer driver.Close(ctx)
//
//	// Create LLM client
//	llmConfig := llm.Config{Model: "gpt-4o-mini"}
//	llmClient := llm.NewOpenAIClient("your-api-key", llmConfig)
//
//	// Create embedder
//	embConfig := embedder.Config{Model: "text-embedding-3-small"}
//	embedderClient := embedder.NewOpenAIEmbedder("your-api-key", embConfig)
//
//	// Create Graphiti client
//	config := &graphiti.Config{GroupID: "my-group"}
//	client := graphiti.NewClient(driver, llmClient, embedderClient, config)
//
// # Adding Episodes
//
// Episodes are temporal data units that get processed into the knowledge graph:
//
//	episodes := []types.Episode{
//		{
//			ID:        "meeting-1",
//			Name:      "Team Meeting",
//			Content:   "Discussed project timeline with Alice and Bob",
//			Reference: time.Now(),
//			CreatedAt: time.Now(),
//			GroupID:   "my-group",
//		},
//	}
//
//	err = client.Add(ctx, episodes, nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Searching
//
// Perform hybrid search across the knowledge graph:
//
//	results, err := client.Search(ctx, "project timeline", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for _, node := range results.Nodes {
//		fmt.Printf("Found node: %s\n", node.Name)
//	}
//
// # Node Types
//
// Graphiti supports three types of nodes:
//
//   - EntityNode: Represents entities extracted from content (people, places, concepts)
//   - EpisodicNode: Represents episodic memories or events
//   - CommunityNode: Represents communities of related entities
//
// # Edge Types
//
// Relationships between nodes are represented by edges:
//
//   - EntityEdge: Relationships between entities
//   - EpisodicEdge: Episodic relationships
//   - CommunityEdge: Community relationships
//
// # Temporal Awareness
//
// All nodes and edges include temporal information:
//
//   - CreatedAt: When the node/edge was first created
//   - UpdatedAt: When it was last modified
//   - ValidFrom: When the information becomes valid
//   - ValidTo: When the information expires (optional)
//
// # Multi-tenancy
//
// Use GroupID to isolate data for different users or contexts:
//
//	config := &graphiti.Config{
//		GroupID: "user-123",
//		TimeZone: time.UTC,
//	}
//
// # Error Handling
//
// The library provides typed errors for common scenarios:
//
//   - ErrNodeNotFound: Returned when a requested node doesn't exist
//   - ErrEdgeNotFound: Returned when a requested edge doesn't exist
//   - ErrInvalidEpisode: Returned when an episode is malformed
//
// # Architecture
//
// The library follows a modular architecture:
//
//   - pkg/driver: Graph database abstraction layer
//   - pkg/llm: Language model client interfaces
//   - pkg/embedder: Embedding model client interfaces
//   - pkg/types: Core type definitions
//
// This design allows easy extension with additional database backends,
// LLM providers, and embedding services.
package graphiti
