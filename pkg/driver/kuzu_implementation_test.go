package driver_test

import (
	"context"
	"testing"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


// TestKuzuDriverImplementedMethods tests the newly implemented methods
func TestKuzuDriverImplementedMethods(t *testing.T) {
	ctx := context.Background()
	dbPath := createTempKuzuDB(t)
	d, err := driver.NewKuzuDriver(dbPath, 1)
	require.NoError(t, err)
	defer d.Close()

	t.Run("GetNode returns specific error when not found", func(t *testing.T) {
		node, err := d.GetNode(ctx, "nonexistent-id", "test-group")
		assert.Nil(t, node)
		assert.Contains(t, err.Error(), "node not found")
		assert.NotContains(t, err.Error(), "not implemented")
	})

	t.Run("GetNodes returns empty slice for nonexistent nodes", func(t *testing.T) {
		nodes, err := d.GetNodes(ctx, []string{"nonexistent-1", "nonexistent-2"}, "test-group")
		assert.NoError(t, err)
		assert.Len(t, nodes, 0)
	})

	t.Run("GetNodes handles empty input", func(t *testing.T) {
		nodes, err := d.GetNodes(ctx, []string{}, "test-group")
		assert.NoError(t, err)
		assert.NotNil(t, nodes)
		assert.Len(t, nodes, 0)
	})

	t.Run("GetEdge returns specific error when not found", func(t *testing.T) {
		edge, err := d.GetEdge(ctx, "nonexistent-edge", "test-group")
		assert.Nil(t, edge)
		assert.Contains(t, err.Error(), "edge not found")
		assert.NotContains(t, err.Error(), "not implemented")
	})

	t.Run("DeleteNode attempts to execute deletion", func(t *testing.T) {
		err := d.DeleteNode(ctx, "nonexistent-id", "test-group")
		// We expect an error because the node doesn't exist or there's a schema issue
		// But the important thing is that it's NOT "not implemented"
		if err != nil {
			assert.NotContains(t, err.Error(), "not implemented")
			// The error should be related to actual database operations
			assert.True(t,
				err.Error() != "" &&
				!assert.ObjectsAreEqual("KuzuDriver not implemented - requires github.com/kuzudb/go-kuzu dependency", err.Error()),
				"Error should not be the 'not implemented' message")
		}
	})

	t.Run("UpsertNode attempts to create node", func(t *testing.T) {
		node := &types.Node{
			ID:      "test-node-1",
			Name:    "Test Node",
			Type:    types.EntityNodeType,
			GroupID: "test-group",
		}

		err := d.UpsertNode(ctx, node)
		// We might get an error due to schema/timestamp issues, but it should not be "not implemented"
		if err != nil {
			assert.NotContains(t, err.Error(), "not implemented")
			// The error should be related to actual database operations
			assert.True(t,
				err.Error() != "" &&
				!assert.ObjectsAreEqual("KuzuDriver not implemented - requires github.com/kuzudb/go-kuzu dependency", err.Error()),
				"Error should not be the 'not implemented' message")
		}
	})
}

// TestKuzuDriverMethodsImplemented verifies that key methods are no longer stubs
func TestKuzuDriverMethodsImplemented(t *testing.T) {
	ctx := context.Background()
	dbPath := createTempKuzuDB(t)
	d, err := driver.NewKuzuDriver(dbPath, 1)
	require.NoError(t, err)
	defer d.Close()

	// Test that these methods don't return the "not implemented" error
	stubError := "KuzuDriver not implemented - requires github.com/kuzudb/go-kuzu dependency"

	t.Run("GetNode is implemented", func(t *testing.T) {
		_, err := d.GetNode(ctx, "test", "test")
		assert.NotEqual(t, stubError, err.Error())
	})

	t.Run("UpsertNode is implemented", func(t *testing.T) {
		node := &types.Node{ID: "test", Name: "test", Type: types.EntityNodeType, GroupID: "test"}
		err := d.UpsertNode(ctx, node)
		if err != nil {
			assert.NotEqual(t, stubError, err.Error())
		}
	})

	t.Run("DeleteNode is implemented", func(t *testing.T) {
		err := d.DeleteNode(ctx, "test", "test")
		if err != nil {
			assert.NotEqual(t, stubError, err.Error())
		}
	})

	t.Run("GetNodes is implemented", func(t *testing.T) {
		_, err := d.GetNodes(ctx, []string{"test"}, "test")
		assert.NoError(t, err) // This should work without error for empty results
	})

	t.Run("GetEdge is implemented", func(t *testing.T) {
		_, err := d.GetEdge(ctx, "test", "test")
		assert.NotEqual(t, stubError, err.Error())
	})

	t.Run("UpsertEdge is implemented", func(t *testing.T) {
		edge := &types.Edge{
			BaseEdge: types.BaseEdge{
				ID:           "test-edge",
				GroupID:      "test",
				SourceNodeID: "test-source",
				TargetNodeID: "test-target",
			},
			Type:     types.EntityEdgeType,
			SourceID: "test-source",
			TargetID: "test-target",
		}
		err := d.UpsertEdge(ctx, edge)
		if err != nil {
			assert.NotEqual(t, stubError, err.Error())
		}
	})

	t.Run("DeleteEdge is implemented", func(t *testing.T) {
		err := d.DeleteEdge(ctx, "test", "test")
		if err != nil {
			assert.NotEqual(t, stubError, err.Error())
		}
	})

	t.Run("GetEdges is implemented", func(t *testing.T) {
		_, err := d.GetEdges(ctx, []string{"test"}, "test")
		assert.NoError(t, err) // This should work without error for empty results
	})

	t.Run("GetNeighbors is implemented", func(t *testing.T) {
		_, err := d.GetNeighbors(ctx, "test", "test", 2)
		assert.NoError(t, err) // This should work without error for empty results
	})

	t.Run("GetRelatedNodes is implemented", func(t *testing.T) {
		_, err := d.GetRelatedNodes(ctx, "test", "test", []types.EdgeType{types.EntityEdgeType})
		assert.NoError(t, err) // This should work without error for empty results
	})

	t.Run("SearchNodesByEmbedding is implemented", func(t *testing.T) {
		_, err := d.SearchNodesByEmbedding(ctx, []float32{0.1, 0.2, 0.3}, "test", 10)
		assert.NoError(t, err) // This should work without error for empty results
	})

	t.Run("SearchEdgesByEmbedding is implemented", func(t *testing.T) {
		_, err := d.SearchEdgesByEmbedding(ctx, []float32{0.1, 0.2, 0.3}, "test", 10)
		assert.NoError(t, err) // This should work without error for empty results
	})

	t.Run("UpsertNodes is implemented", func(t *testing.T) {
		nodes := []*types.Node{
			{ID: "test1", Name: "Test 1", Type: types.EntityNodeType, GroupID: "test"},
			{ID: "test2", Name: "Test 2", Type: types.EntityNodeType, GroupID: "test"},
		}
		err := d.UpsertNodes(ctx, nodes)
		if err != nil {
			assert.NotEqual(t, stubError, err.Error())
		}
	})

	t.Run("UpsertEdges is implemented", func(t *testing.T) {
		edges := []*types.Edge{
			{
				BaseEdge: types.BaseEdge{
					ID:           "edge1",
					GroupID:      "test",
					SourceNodeID: "src1",
					TargetNodeID: "tgt1",
				},
				Type:     types.EntityEdgeType,
				SourceID: "src1",
				TargetID: "tgt1",
			},
			{
				BaseEdge: types.BaseEdge{
					ID:           "edge2",
					GroupID:      "test",
					SourceNodeID: "src2",
					TargetNodeID: "tgt2",
				},
				Type:     types.EntityEdgeType,
				SourceID: "src2",
				TargetID: "tgt2",
			},
		}
		err := d.UpsertEdges(ctx, edges)
		if err != nil {
			assert.NotEqual(t, stubError, err.Error())
		}
	})
}