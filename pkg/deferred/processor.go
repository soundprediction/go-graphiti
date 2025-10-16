package deferred

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"
	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/prompts"
	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/soundprediction/go-graphiti/pkg/utils"
	"github.com/soundprediction/go-graphiti/pkg/utils/maintenance"
)

// DeferredProcessor handles batch processing of deferred graph data
type DeferredProcessor struct {
	driver   driver.GraphDriver
	llm      llm.Client
	embedder embedder.Client
	logger   *slog.Logger
}

// NewDeferredProcessor creates a new processor for deferred graph data
func NewDeferredProcessor(driver driver.GraphDriver, llmClient llm.Client, embedderClient embedder.Client, logger *slog.Logger) *DeferredProcessor {
	if logger == nil {
		logger = slog.Default()
	}

	return &DeferredProcessor{
		driver:   driver,
		llm:      llmClient,
		embedder: embedderClient,
		logger:   logger,
	}
}

// ProcessDeferredOptions holds options for processing deferred data
type ProcessDeferredOptions struct {
	// BatchSize is the number of episodes to process in a single batch
	BatchSize int
	// EntityTypes custom entity type definitions for resolution
	EntityTypes map[string]interface{}
	// GenerateEmbeddings whether to generate missing embeddings
	GenerateEmbeddings bool
	// DeleteAfterProcessing whether to delete processed data from DuckDB
	DeleteAfterProcessing bool
	// EpisodeIDs specific episodes to process (if empty, processes all)
	EpisodeIDs []string
	// GroupID to filter episodes by group (if empty, processes all groups)
	GroupID string
}

// ProcessDeferredResult represents the result of processing deferred data
type ProcessDeferredResult struct {
	EpisodesProcessed int
	EntitiesIngested  int
	EdgesIngested     int
	DuplicatesFound   int
	EdgesInvalidated  int
	Errors            []error
}

// ProcessDeferred reads deferred data from DuckDB and ingests it into the graph database
// with full deduplication and relationship resolution
func (p *DeferredProcessor) ProcessDeferred(ctx context.Context, duckDBPath string, options *ProcessDeferredOptions) (*ProcessDeferredResult, error) {
	if options == nil {
		options = &ProcessDeferredOptions{
			BatchSize:             10,
			GenerateEmbeddings:    true,
			DeleteAfterProcessing: false,
		}
	}

	if options.BatchSize <= 0 {
		options.BatchSize = 10
	}

	p.logger.Info("Starting deferred data processing",
		"duckdb_path", duckDBPath,
		"batch_size", options.BatchSize)

	// Open DuckDB connection
	db, err := sql.Open("duckdb", duckDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	// Load episodes to process
	episodes, err := p.loadEpisodes(ctx, db, options)
	if err != nil {
		return nil, fmt.Errorf("failed to load episodes: %w", err)
	}

	p.logger.Info("Loaded episodes for processing",
		"episode_count", len(episodes))

	result := &ProcessDeferredResult{
		Errors: []error{},
	}

	// Process episodes in batches
	batches := utils.ChunkSlice(episodes, options.BatchSize)
	for batchIdx, batch := range batches {
		p.logger.Info("Processing batch",
			"batch", batchIdx+1,
			"total_batches", len(batches),
			"batch_size", len(batch))

		batchResult, err := p.processBatch(ctx, db, batch, options)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("batch %d failed: %w", batchIdx+1, err))
			continue
		}

		// Aggregate results
		result.EpisodesProcessed += batchResult.EpisodesProcessed
		result.EntitiesIngested += batchResult.EntitiesIngested
		result.EdgesIngested += batchResult.EdgesIngested
		result.DuplicatesFound += batchResult.DuplicatesFound
		result.EdgesInvalidated += batchResult.EdgesInvalidated
		result.Errors = append(result.Errors, batchResult.Errors...)
	}

	// Delete processed data if requested
	if options.DeleteAfterProcessing {
		p.logger.Info("Cleaning up processed data from DuckDB")
		if err := p.deleteProcessedData(ctx, db, episodes); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to clean up DuckDB: %w", err))
		}
	}

	p.logger.Info("Deferred data processing completed",
		"episodes_processed", result.EpisodesProcessed,
		"entities_ingested", result.EntitiesIngested,
		"edges_ingested", result.EdgesIngested,
		"duplicates_found", result.DuplicatesFound,
		"edges_invalidated", result.EdgesInvalidated,
		"errors", len(result.Errors))

	return result, nil
}

// loadEpisodes loads episode data from DuckDB
func (p *DeferredProcessor) loadEpisodes(ctx context.Context, db *sql.DB, options *ProcessDeferredOptions) ([]*types.Node, error) {
	query := `SELECT id, name, content, reference, group_id, created_at, updated_at, valid_from, embedding, metadata FROM episodes`
	var conditions []string
	var args []interface{}

	if options.GroupID != "" {
		conditions = append(conditions, "group_id = ?")
		args = append(args, options.GroupID)
	}

	if len(options.EpisodeIDs) > 0 {
		placeholders := ""
		for i, id := range options.EpisodeIDs {
			if i > 0 {
				placeholders += ", "
			}
			placeholders += "?"
			args = append(args, id)
		}
		conditions = append(conditions, fmt.Sprintf("id IN (%s)", placeholders))
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY created_at ASC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query episodes: %w", err)
	}
	defer rows.Close()

	var episodes []*types.Node
	for rows.Next() {
		var (
			id, name, content, groupID      string
			reference, createdAt, updatedAt time.Time
			validFrom                       time.Time
			embedding                       []float32
			metadataStr                     string
		)

		err := rows.Scan(&id, &name, &content, &reference, &groupID, &createdAt, &updatedAt, &validFrom, &embedding, &metadataStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan episode row: %w", err)
		}

		// Parse metadata
		var metadata map[string]interface{}
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
				p.logger.Warn("Failed to parse episode metadata", "episode_id", id, "error", err)
				metadata = make(map[string]interface{})
			}
		}

		episode := &types.Node{
			ID:        id,
			Name:      name,
			Type:      types.EpisodicNodeType,
			GroupID:   groupID,
			Content:   content,
			Reference: reference,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			ValidFrom: validFrom,
			Embedding: embedding,
			Metadata:  metadata,
		}

		episodes = append(episodes, episode)
	}

	return episodes, rows.Err()
}

// processBatch processes a batch of episodes with full deduplication
func (p *DeferredProcessor) processBatch(ctx context.Context, db *sql.DB, episodes []*types.Node, options *ProcessDeferredOptions) (*ProcessDeferredResult, error) {
	result := &ProcessDeferredResult{}

	// Load all nodes and edges for this batch from DuckDB
	var allExtractedNodes []*types.Node
	var allExtractedEdges []*types.Edge
	episodeIDToNodes := make(map[string][]*types.Node)
	episodeIDToEdges := make(map[string][]*types.Edge)

	for _, episode := range episodes {
		// Load entity nodes for this episode
		nodes, err := p.loadEntityNodes(ctx, db, episode.ID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to load nodes for episode %s: %w", episode.ID, err))
			continue
		}

		// Load entity edges for this episode
		edges, err := p.loadEntityEdges(ctx, db, episode.ID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to load edges for episode %s: %w", episode.ID, err))
			continue
		}

		allExtractedNodes = append(allExtractedNodes, nodes...)
		allExtractedEdges = append(allExtractedEdges, edges...)
		episodeIDToNodes[episode.ID] = nodes
		episodeIDToEdges[episode.ID] = edges
	}

	p.logger.Info("Loaded extracted data for batch",
		"episodes", len(episodes),
		"nodes", len(allExtractedNodes),
		"edges", len(allExtractedEdges))

	// PHASE 1: CROSS-EPISODE NODE DEDUPLICATION
	nodeOps := maintenance.NewNodeOperations(p.driver, p.llm, p.embedder, prompts.NewLibrary())

	p.logger.Info("Starting cross-episode node deduplication")

	// Collect previous episodes for context (get episodes from graph that occurred before this batch)
	var previousEpisodes []*types.Node
	if len(episodes) > 0 {
		firstEpisodeTime := episodes[0].CreatedAt
		groupID := episodes[0].GroupID
		prevNodes, err := p.driver.GetNodesInTimeRange(ctx, firstEpisodeTime.Add(-7*24*time.Hour), firstEpisodeTime, groupID)
		if err != nil {
			p.logger.Warn("Failed to get previous episodes for context", "error", err)
		} else {
			for _, node := range prevNodes {
				if node.Type == types.EpisodicNodeType {
					previousEpisodes = append(previousEpisodes, node)
				}
			}
		}
	}

	resolvedNodes, uuidMap, duplicatePairs, err := nodeOps.ResolveExtractedNodes(ctx,
		allExtractedNodes, episodes[0], previousEpisodes, options.EntityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve nodes: %w", err)
	}

	result.DuplicatesFound = len(duplicatePairs)

	p.logger.Info("Node deduplication completed",
		"original_nodes", len(allExtractedNodes),
		"resolved_nodes", len(resolvedNodes),
		"duplicates_found", len(duplicatePairs))

	// Build and store duplicate edges
	if len(duplicatePairs) > 0 {
		edgeOps := maintenance.NewEdgeOperations(p.driver, p.llm, p.embedder, prompts.NewLibrary())
		duplicateEdges, err := edgeOps.BuildDuplicateOfEdges(ctx, episodes[0], time.Now(), duplicatePairs)
		if err != nil {
			return nil, fmt.Errorf("failed to build duplicate edges: %w", err)
		}
		for _, edge := range duplicateEdges {
			if err := p.driver.UpsertEdge(ctx, edge); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to store duplicate edge: %w", err))
			}
		}
	}

	// PHASE 2: EDGE RESOLUTION & TEMPORAL INVALIDATION
	p.logger.Info("Starting relationship resolution")

	// Update edge pointers to use resolved node UUIDs
	utils.ResolveEdgePointers(allExtractedEdges, uuidMap)

	edgeOps := maintenance.NewEdgeOperations(p.driver, p.llm, p.embedder, prompts.NewLibrary())
	resolvedEdges, invalidatedEdges, err := edgeOps.ResolveExtractedEdges(ctx,
		allExtractedEdges, episodes[0], resolvedNodes, options.GenerateEmbeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve edges: %w", err)
	}

	result.EdgesInvalidated = len(invalidatedEdges)

	p.logger.Info("Relationship resolution completed",
		"original_edges", len(allExtractedEdges),
		"resolved_edges", len(resolvedEdges),
		"invalidated_edges", len(invalidatedEdges))

	// PHASE 3: ATTRIBUTE EXTRACTION
	p.logger.Info("Starting attribute extraction")

	hydratedNodes, err := nodeOps.ExtractAttributesFromNodes(ctx,
		resolvedNodes, episodes[0], previousEpisodes, options.EntityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to extract attributes: %w", err)
	}

	p.logger.Info("Attribute extraction completed",
		"hydrated_nodes", len(hydratedNodes))

	// PHASE 4: BULK PERSISTENCE TO KUZU
	p.logger.Info("Writing to Kuzu graph database")

	// Process each episode separately for episodic edges
	for _, episode := range episodes {
		// Build episodic edges for this episode's nodes
		episodeNodes := episodeIDToNodes[episode.ID]

		// Filter hydrated nodes to only those from this episode
		var episodeHydratedNodes []*types.Node
		nodeIDSet := make(map[string]bool)
		for _, node := range episodeNodes {
			nodeIDSet[node.ID] = true
			// Also check for resolved node ID
			if resolvedID, ok := uuidMap[node.ID]; ok {
				nodeIDSet[resolvedID] = true
			}
		}

		for _, hydratedNode := range hydratedNodes {
			if nodeIDSet[hydratedNode.ID] {
				episodeHydratedNodes = append(episodeHydratedNodes, hydratedNode)
			}
		}

		// Build episodic edges
		episodicEdges, err := edgeOps.BuildEpisodicEdges(ctx, episodeHydratedNodes, episode.ID, time.Now())
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to build episodic edges for episode %s: %w", episode.ID, err))
			continue
		}

		// Update episode metadata with entity edge UUIDs
		entityEdgeUUIDs := make([]string, 0)
		for _, edge := range resolvedEdges {
			// Check if this edge belongs to this episode
			for _, episodeID := range edge.Episodes {
				if episodeID == episode.ID {
					entityEdgeUUIDs = append(entityEdgeUUIDs, edge.ID)
					break
				}
			}
		}
		for _, edge := range invalidatedEdges {
			for _, episodeID := range edge.Episodes {
				if episodeID == episode.ID {
					entityEdgeUUIDs = append(entityEdgeUUIDs, edge.ID)
					break
				}
			}
		}

		if episode.Metadata == nil {
			episode.Metadata = make(map[string]interface{})
		}
		episode.Metadata["entity_edges"] = entityEdgeUUIDs

		// Write episode, episodic edges, and this episode's portion of entity data
		_, err = utils.AddNodesAndEdgesBulk(ctx, p.driver,
			[]*types.Node{episode},
			episodicEdges,
			episodeHydratedNodes,
			[]*types.Edge{}, // Entity edges written separately below
			p.embedder)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to persist episode %s: %w", episode.ID, err))
			continue
		}

		result.EpisodesProcessed++
	}

	// Write all resolved and invalidated edges (these are deduplicated across episodes)
	allEdges := append(resolvedEdges, invalidatedEdges...)
	for _, edge := range allEdges {
		if err := p.driver.UpsertEdge(ctx, edge); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to upsert edge %s: %w", edge.ID, err))
		}
	}

	result.EntitiesIngested = len(hydratedNodes)
	result.EdgesIngested = len(allEdges)

	p.logger.Info("Batch processing completed",
		"episodes_processed", result.EpisodesProcessed,
		"entities_ingested", result.EntitiesIngested,
		"edges_ingested", result.EdgesIngested)

	return result, nil
}

// loadEntityNodes loads entity nodes for an episode from DuckDB
func (p *DeferredProcessor) loadEntityNodes(ctx context.Context, db *sql.DB, episodeID string) ([]*types.Node, error) {
	query := `
		SELECT id, name, entity_type, group_id, created_at, updated_at,
		       valid_from, valid_to, summary, embedding, name_embedding, metadata
		FROM entity_nodes
		WHERE episode_id = ?
	`

	rows, err := db.QueryContext(ctx, query, episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*types.Node
	for rows.Next() {
		var (
			id, name, entityType, groupID      string
			createdAt, updatedAt, validFrom    time.Time
			validTo                            sql.NullTime
			summary                            string
			embeddingBytes, nameEmbeddingBytes []byte
			metadataStr                        string
		)

		err := rows.Scan(&id, &name, &entityType, &groupID, &createdAt, &updatedAt,
			&validFrom, &validTo, &summary, &embeddingBytes, &nameEmbeddingBytes, &metadataStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity node row: %w", err)
		}

		var metadata map[string]interface{}
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
				p.logger.Warn("Failed to parse node metadata", "node_id", id, "error", err)
				metadata = make(map[string]interface{})
			}
		}

		node := &types.Node{
			ID:         id,
			Name:       name,
			Type:       types.EntityNodeType,
			EntityType: entityType,
			GroupID:    groupID,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
			ValidFrom:  validFrom,
			Summary:    summary,
			Metadata:   metadata,
		}

		if validTo.Valid {
			node.ValidTo = &validTo.Time
		}

		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// loadEntityEdges loads entity edges for an episode from DuckDB
func (p *DeferredProcessor) loadEntityEdges(ctx context.Context, db *sql.DB, episodeID string) ([]*types.Edge, error) {
	query := `
		SELECT id, source_id, target_id, name, fact, summary, edge_type, group_id,
		       created_at, valid_from, invalid_at, expired_at, embedding, fact_embedding,
		       episodes, metadata
		FROM entity_edges
		WHERE episode_id = ?
	`

	rows, err := db.QueryContext(ctx, query, episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity edges: %w", err)
	}
	defer rows.Close()

	var edges []*types.Edge
	for rows.Next() {
		var (
			id, sourceID, targetID, name       string
			fact, summary, edgeType, groupID   string
			createdAt, validFrom               time.Time
			invalidAt, expiredAt               sql.NullTime
			embeddingBytes, factEmbeddingBytes []byte
			episodesStr, metadataStr           string
		)

		err := rows.Scan(&id, &sourceID, &targetID, &name, &fact, &summary, &edgeType, &groupID,
			&createdAt, &validFrom, &invalidAt, &expiredAt, &embeddingBytes, &factEmbeddingBytes,
			&episodesStr, &metadataStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity edge row: %w", err)
		}

		var metadata map[string]interface{}
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
				p.logger.Warn("Failed to parse edge metadata", "edge_id", id, "error", err)
				metadata = make(map[string]interface{})
			}
		}

		var episodes []string
		if episodesStr != "" {
			if err := json.Unmarshal([]byte(episodesStr), &episodes); err != nil {
				p.logger.Warn("Failed to parse episodes", "edge_id", id, "error", err)
				episodes = []string{episodeID}
			}
		}

		edge := &types.Edge{
			BaseEdge: types.BaseEdge{
				ID:           id,
				GroupID:      groupID,
				SourceNodeID: sourceID,
				TargetNodeID: targetID,
				CreatedAt:    createdAt,
				Metadata:     metadata,
			},
			SourceID:  sourceID,
			TargetID:  targetID,
			Name:      name,
			Fact:      fact,
			Summary:   summary,
			Type:      types.EdgeType(edgeType),
			ValidFrom: validFrom,
			Episodes:  episodes,
		}

		if invalidAt.Valid {
			edge.InvalidAt = &invalidAt.Time
		}
		if expiredAt.Valid {
			edge.ExpiredAt = &expiredAt.Time
		}

		edges = append(edges, edge)
	}

	return edges, rows.Err()
}

// deleteProcessedData removes processed episodes and their data from DuckDB
func (p *DeferredProcessor) deleteProcessedData(ctx context.Context, db *sql.DB, episodes []*types.Node) error {
	if len(episodes) == 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Build episode ID list
	episodeIDs := make([]string, len(episodes))
	for i, ep := range episodes {
		episodeIDs[i] = ep.ID
	}

	// Create placeholders for IN clause
	placeholders := ""
	args := make([]interface{}, len(episodeIDs))
	for i, id := range episodeIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args[i] = id
	}

	// Delete from all tables
	tables := []string{"episodic_edges", "entity_edges", "entity_nodes", "episodes"}
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE episode_id IN (%s)", table, placeholders)
		if table == "episodes" {
			query = fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", table, placeholders)
		}

		_, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to delete from %s: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit deletion: %w", err)
	}

	return nil
}

// GetDeferredStats returns statistics about deferred data in DuckDB
func (p *DeferredProcessor) GetDeferredStats(ctx context.Context, duckDBPath string) (map[string]int, error) {
	db, err := sql.Open("duckdb", duckDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer db.Close()

	stats := make(map[string]int)

	tables := []string{"episodes", "entity_nodes", "entity_edges", "episodic_edges"}
	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		err := db.QueryRowContext(ctx, query).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to count %s: %w", table, err)
		}
		stats[table] = count
	}

	return stats, nil
}
