package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// DuckDBWriter handles writing nodes and edges to DuckDB tables
type DuckDBWriter struct {
	db *sql.DB
}

// NewDuckDBWriter creates a new DuckDB writer
// dbPath should be the path to the DuckDB database file
func NewDuckDBWriter(dbPath string) (*DuckDBWriter, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		// Directory creation would need os package
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}

	writer := &DuckDBWriter{db: db}

	// Create tables if they don't exist
	if err := writer.createTables(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return writer, nil
}

// createTables creates the necessary DuckDB tables for deferred ingestion
func (w *DuckDBWriter) createTables(ctx context.Context) error {
	// Create episodes table
	_, err := w.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS episodes (
			id VARCHAR PRIMARY KEY,
			name VARCHAR,
			content VARCHAR,
			reference TIMESTAMP,
			group_id VARCHAR,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			valid_from TIMESTAMP,
			embedding FLOAT[],
			metadata JSON
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create episodes table: %w", err)
	}

	// Create entity nodes table
	_, err = w.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS entity_nodes (
			id VARCHAR PRIMARY KEY,
			name VARCHAR,
			entity_type VARCHAR,
			group_id VARCHAR,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			valid_from TIMESTAMP,
			valid_to TIMESTAMP,
			summary VARCHAR,
			embedding FLOAT[],
			name_embedding FLOAT[],
			metadata JSON,
			episode_id VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create entity_nodes table: %w", err)
	}

	// Create entity edges table
	_, err = w.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS entity_edges (
			id VARCHAR PRIMARY KEY,
			source_id VARCHAR,
			target_id VARCHAR,
			name VARCHAR,
			fact VARCHAR,
			summary VARCHAR,
			edge_type VARCHAR,
			group_id VARCHAR,
			created_at TIMESTAMP,
			valid_from TIMESTAMP,
			invalid_at TIMESTAMP,
			expired_at TIMESTAMP,
			embedding FLOAT[],
			fact_embedding FLOAT[],
			episodes JSON,
			metadata JSON,
			episode_id VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create entity_edges table: %w", err)
	}

	// Create episodic edges table
	_, err = w.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS episodic_edges (
			id VARCHAR PRIMARY KEY,
			source_id VARCHAR,
			target_id VARCHAR,
			name VARCHAR,
			edge_type VARCHAR,
			group_id VARCHAR,
			created_at TIMESTAMP,
			valid_from TIMESTAMP,
			episode_id VARCHAR
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create episodic_edges table: %w", err)
	}

	return nil
}

// WriteEpisode writes an episode node to DuckDB
func (w *DuckDBWriter) WriteEpisode(ctx context.Context, episode *types.Node) error {
	metadataJSON, err := json.Marshal(episode.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Convert timestamps to sql.NullTime to handle zero values
	reference := sql.NullTime{}
	if !episode.Reference.IsZero() {
		reference = sql.NullTime{Time: episode.Reference, Valid: true}
	}

	createdAt := sql.NullTime{}
	if !episode.CreatedAt.IsZero() {
		createdAt = sql.NullTime{Time: episode.CreatedAt, Valid: true}
	}

	updatedAt := sql.NullTime{}
	if !episode.UpdatedAt.IsZero() {
		updatedAt = sql.NullTime{Time: episode.UpdatedAt, Valid: true}
	}

	validFrom := sql.NullTime{}
	if !episode.ValidFrom.IsZero() {
		validFrom = sql.NullTime{Time: episode.ValidFrom, Valid: true}
	}

	// Handle empty GroupID
	groupID := episode.GroupID
	if groupID == "" {
		groupID = "default"
	}

	_, err = w.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO episodes (
			id, name, content, reference, group_id,
			created_at, updated_at, valid_from, embedding, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		episode.ID,
		episode.Name,
		episode.Content,
		reference,
		groupID,
		createdAt,
		updatedAt,
		validFrom,
		episode.Embedding,
		string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to write episode: %w", err)
	}

	return nil
}

// WriteEntityNodes writes entity nodes to DuckDB
func (w *DuckDBWriter) WriteEntityNodes(ctx context.Context, nodes []*types.Node, episodeID string) error {
	if len(nodes) == 0 {
		return nil
	}

	// Prepare batch insert
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO entity_nodes (
			id, name, entity_type, group_id, created_at, updated_at,
			valid_from, valid_to, summary, embedding, name_embedding, metadata, episode_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, node := range nodes {
		metadataJSON, err := json.Marshal(node.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		// Convert timestamps to sql.NullTime
		createdAt := sql.NullTime{}
		if !node.CreatedAt.IsZero() {
			createdAt = sql.NullTime{Time: node.CreatedAt, Valid: true}
		}

		updatedAt := sql.NullTime{}
		if !node.UpdatedAt.IsZero() {
			updatedAt = sql.NullTime{Time: node.UpdatedAt, Valid: true}
		}

		validFrom := sql.NullTime{}
		if !node.ValidFrom.IsZero() {
			validFrom = sql.NullTime{Time: node.ValidFrom, Valid: true}
		}

		validTo := sql.NullTime{}
		if node.ValidTo != nil && !node.ValidTo.IsZero() {
			validTo = sql.NullTime{Time: *node.ValidTo, Valid: true}
		}

		_, err = stmt.ExecContext(ctx,
			node.ID,
			node.Name,
			node.EntityType,
			node.GroupID,
			createdAt,
			updatedAt,
			validFrom,
			validTo,
			node.Summary,
			node.Embedding,
			node.NameEmbedding,
			string(metadataJSON),
			episodeID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert node %s: %w", node.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WriteEntityEdges writes entity edges to DuckDB
func (w *DuckDBWriter) WriteEntityEdges(ctx context.Context, edges []*types.Edge, episodeID string) error {
	if len(edges) == 0 {
		return nil
	}

	// Prepare batch insert
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO entity_edges (
			id, source_id, target_id, name, fact, summary, edge_type, group_id,
			created_at, valid_from, invalid_at, expired_at, embedding, fact_embedding,
			episodes, metadata, episode_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, edge := range edges {
		metadataJSON, err := json.Marshal(edge.BaseEdge.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		episodesJSON, err := json.Marshal(edge.Episodes)
		if err != nil {
			return fmt.Errorf("failed to marshal episodes: %w", err)
		}

		// Convert timestamps to sql.NullTime
		createdAt := sql.NullTime{}
		if !edge.CreatedAt.IsZero() {
			createdAt = sql.NullTime{Time: edge.CreatedAt, Valid: true}
		}

		validFrom := sql.NullTime{}
		if !edge.ValidFrom.IsZero() {
			validFrom = sql.NullTime{Time: edge.ValidFrom, Valid: true}
		}

		invalidAt := sql.NullTime{}
		if edge.InvalidAt != nil && !edge.InvalidAt.IsZero() {
			invalidAt = sql.NullTime{Time: *edge.InvalidAt, Valid: true}
		}

		expiredAt := sql.NullTime{}
		if edge.ExpiredAt != nil && !edge.ExpiredAt.IsZero() {
			expiredAt = sql.NullTime{Time: *edge.ExpiredAt, Valid: true}
		}

		_, err = stmt.ExecContext(ctx,
			edge.ID,
			edge.SourceID,
			edge.TargetID,
			edge.Name,
			edge.Fact,
			edge.Summary,
			string(edge.Type),
			edge.GroupID,
			createdAt,
			validFrom,
			invalidAt,
			expiredAt,
			edge.Embedding,
			edge.FactEmbedding,
			string(episodesJSON),
			string(metadataJSON),
			episodeID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert edge %s: %w", edge.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WriteEpisodicEdges writes episodic edges to DuckDB
func (w *DuckDBWriter) WriteEpisodicEdges(ctx context.Context, edges []*types.Edge, episodeID string) error {
	if len(edges) == 0 {
		return nil
	}

	// Prepare batch insert
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO episodic_edges (
			id, source_id, target_id, name, edge_type, group_id,
			created_at, valid_from, episode_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, edge := range edges {
		// Convert timestamps to sql.NullTime
		createdAt := sql.NullTime{}
		if !edge.CreatedAt.IsZero() {
			createdAt = sql.NullTime{Time: edge.CreatedAt, Valid: true}
		}

		validFrom := sql.NullTime{}
		if !edge.ValidFrom.IsZero() {
			validFrom = sql.NullTime{Time: edge.ValidFrom, Valid: true}
		}

		_, err = stmt.ExecContext(ctx,
			edge.ID,
			edge.SourceID,
			edge.TargetID,
			edge.Name,
			string(edge.Type),
			edge.GroupID,
			createdAt,
			validFrom,
			episodeID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert episodic edge %s: %w", edge.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close closes the DuckDB connection
func (w *DuckDBWriter) Close() error {
	if w.db != nil {
		return w.db.Close()
	}
	return nil
}
