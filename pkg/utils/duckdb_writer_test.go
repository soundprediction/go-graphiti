package utils

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
)

func TestDuckDBWriter(t *testing.T) {
	// Create a temporary DuckDB file
	tmpFile := "./test_deferred.duckdb"
	defer os.Remove(tmpFile)

	// Create writer
	writer, err := NewDuckDBWriter(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create DuckDB writer: %v", err)
	}
	defer writer.Close()

	ctx := context.Background()

	// Test episode writing
	now := time.Now()
	episode := &types.Node{
		ID:        "episode-1",
		Name:      "Test Episode",
		Type:      types.EpisodicNodeType,
		GroupID:   "test-group",
		Content:   "This is test content",
		CreatedAt: now,
		UpdatedAt: now,
		ValidFrom: now,
		Metadata:  map[string]interface{}{"test": "value"},
	}

	err = writer.WriteEpisode(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to write episode: %v", err)
	}

	// Test entity node writing
	nodes := []*types.Node{
		{
			ID:         "node-1",
			Name:       "Test Entity",
			Type:       types.EntityNodeType,
			EntityType: "Person",
			GroupID:    "test-group",
			CreatedAt:  now,
			UpdatedAt:  now,
			ValidFrom:  now,
			Summary:    "A test person",
			Metadata:   map[string]interface{}{},
		},
	}

	err = writer.WriteEntityNodes(ctx, nodes, episode.ID)
	if err != nil {
		t.Fatalf("Failed to write entity nodes: %v", err)
	}

	// Test entity edge writing
	edges := []*types.Edge{
		{
			BaseEdge: types.BaseEdge{
				ID:           "edge-1",
				GroupID:      "test-group",
				SourceNodeID: "node-1",
				TargetNodeID: "node-2",
				CreatedAt:    now,
				Metadata:     map[string]interface{}{},
			},
			SourceID:  "node-1",
			TargetID:  "node-2",
			Name:      "knows",
			Type:      types.EntityEdgeType,
			ValidFrom: now,
			Summary:   "Person knows another person",
			Fact:      "Person A knows Person B",
			Episodes:  []string{episode.ID},
		},
	}

	err = writer.WriteEntityEdges(ctx, edges, episode.ID)
	if err != nil {
		t.Fatalf("Failed to write entity edges: %v", err)
	}

	// Test episodic edge writing
	episodicEdges := []*types.Edge{
		{
			BaseEdge: types.BaseEdge{
				ID:           "episodic-edge-1",
				GroupID:      "test-group",
				SourceNodeID: episode.ID,
				TargetNodeID: "node-1",
				CreatedAt:    now,
			},
			SourceID:  episode.ID,
			TargetID:  "node-1",
			Name:      "mentions",
			Type:      types.EpisodicEdgeType,
			ValidFrom: now,
		},
	}

	err = writer.WriteEpisodicEdges(ctx, episodicEdges, episode.ID)
	if err != nil {
		t.Fatalf("Failed to write episodic edges: %v", err)
	}

	// Verify data was written by querying
	rows, err := writer.db.QueryContext(ctx, "SELECT COUNT(*) FROM episodes")
	if err != nil {
		t.Fatalf("Failed to query episodes: %v", err)
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			t.Fatalf("Failed to scan count: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 episode, got %d", count)
		}
	}

	t.Logf("Successfully wrote and verified data in DuckDB: %s", tmpFile)
}
