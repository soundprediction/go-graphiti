package deferred

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/soundprediction/go-graphiti/pkg/utils"
)

func TestDeferredProcessorStats(t *testing.T) {
	// Create a temporary DuckDB file with test data
	tmpFile := "./test_deferred_processor.duckdb"
	defer os.Remove(tmpFile)

	// Write some test data
	writer, err := utils.NewDuckDBWriter(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create DuckDB writer: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Create test episodes
	episode1 := &types.Node{
		ID:        "episode-1",
		Name:      "Episode 1",
		Type:      types.EpisodicNodeType,
		GroupID:   "test-group",
		Content:   "Test content 1",
		CreatedAt: now,
		UpdatedAt: now,
		ValidFrom: now,
		Metadata:  map[string]interface{}{},
	}

	episode2 := &types.Node{
		ID:        "episode-2",
		Name:      "Episode 2",
		Type:      types.EpisodicNodeType,
		GroupID:   "test-group",
		Content:   "Test content 2",
		CreatedAt: now.Add(1 * time.Hour),
		UpdatedAt: now.Add(1 * time.Hour),
		ValidFrom: now.Add(1 * time.Hour),
		Metadata:  map[string]interface{}{},
	}

	// Write episodes
	if err := writer.WriteEpisode(ctx, episode1); err != nil {
		t.Fatalf("Failed to write episode 1: %v", err)
	}
	if err := writer.WriteEpisode(ctx, episode2); err != nil {
		t.Fatalf("Failed to write episode 2: %v", err)
	}

	// Create test entity nodes
	nodes := []*types.Node{
		{
			ID:         "node-1",
			Name:       "Entity 1",
			Type:       types.EntityNodeType,
			EntityType: "Person",
			GroupID:    "test-group",
			CreatedAt:  now,
			UpdatedAt:  now,
			ValidFrom:  now,
			Summary:    "A person",
			Metadata:   map[string]interface{}{},
		},
		{
			ID:         "node-2",
			Name:       "Entity 2",
			Type:       types.EntityNodeType,
			EntityType: "Person",
			GroupID:    "test-group",
			CreatedAt:  now,
			UpdatedAt:  now,
			ValidFrom:  now,
			Summary:    "Another person",
			Metadata:   map[string]interface{}{},
		},
	}

	// Write nodes for both episodes
	if err := writer.WriteEntityNodes(ctx, nodes[:1], episode1.ID); err != nil {
		t.Fatalf("Failed to write entity nodes for episode 1: %v", err)
	}
	if err := writer.WriteEntityNodes(ctx, nodes[1:], episode2.ID); err != nil {
		t.Fatalf("Failed to write entity nodes for episode 2: %v", err)
	}

	// Create test entity edges
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
			Fact:      "Entity 1 knows Entity 2",
			Episodes:  []string{episode1.ID},
		},
	}

	if err := writer.WriteEntityEdges(ctx, edges, episode1.ID); err != nil {
		t.Fatalf("Failed to write entity edges: %v", err)
	}

	writer.Close()

	// Now test the processor stats
	processor := NewDeferredProcessor(nil, nil, nil, nil)

	stats, err := processor.GetDeferredStats(ctx, tmpFile)
	if err != nil {
		t.Fatalf("Failed to get deferred stats: %v", err)
	}

	// Verify stats
	if stats["episodes"] != 2 {
		t.Errorf("Expected 2 episodes, got %d", stats["episodes"])
	}
	if stats["entity_nodes"] != 2 {
		t.Errorf("Expected 2 entity nodes, got %d", stats["entity_nodes"])
	}
	if stats["entity_edges"] != 1 {
		t.Errorf("Expected 1 entity edge, got %d", stats["entity_edges"])
	}

	t.Logf("Deferred stats: %+v", stats)
}
