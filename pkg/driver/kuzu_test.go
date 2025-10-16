package driver_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTempKuzuDB creates a temporary directory for Kuzu database testing
func createTempKuzuDB(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	return filepath.Join(tempDir, "kuzu_test.db")
}

func TestNewKuzuDriver(t *testing.T) {
	t.Run("default path", func(t *testing.T) {
		d, err := driver.NewKuzuDriver("", 1)
		require.NoError(t, err)
		assert.NotNil(t, d)

		// Test that Close works
		err = d.Close()
		assert.NoError(t, err)
	})

	t.Run("custom path", func(t *testing.T) {
		dbPath := createTempKuzuDB(t)
		d, err := driver.NewKuzuDriver(dbPath, 1)
		require.NoError(t, err)
		assert.NotNil(t, d)

		// Test that Close works
		err = d.Close()
		assert.NoError(t, err)
	})
}

// TestKuzuDriverStubImplementation is now deprecated since KuzuDriver is fully implemented
// Kept as a placeholder to maintain test compatibility, but skipped
func TestKuzuDriverStubImplementation(t *testing.T) {
	t.Skip("KuzuDriver is now fully implemented - this stub test is no longer needed")
}

// TestKuzuDriverInterface verifies that KuzuDriver implements GraphDriver interface
func TestKuzuDriverInterface(t *testing.T) {
	var _ driver.GraphDriver = (*driver.KuzuDriver)(nil)
}

// Example test showing expected usage once the full implementation is available
func TestKuzuDriverUsageExample(t *testing.T) {
	t.Skip("Skip until Kuzu library is available")
	
	// This test demonstrates expected usage patterns but is skipped
	// until the actual Kuzu library dependency is available
	d, err := driver.NewKuzuDriver("./test_kuzu_db", 1)
	require.NoError(t, err)
	defer d.Close()
	
	// In a real scenario, you would:
	// 1. Create nodes
	// node := &types.Node{
	//     ID: "test-node",
	//     Name: "Test Node", 
	//     Type: types.NodeTypeEntity,
	//     GroupID: "test-group",
	// }
	// err = d.UpsertNode(ctx, node)
	// require.NoError(t, err)
	//
	// 2. Create edges
	// edge := &types.Edge{
	//     ID: "test-edge",
	//     Type: types.EdgeTypeEntity,
	//     GroupID: "test-group", 
	//     SourceID: "source-node",
	//     TargetID: "target-node",
	// }
	// err = d.UpsertEdge(ctx, edge)
	// require.NoError(t, err)
	//
	// 3. Query neighbors
	// neighbors, err := d.GetNeighbors(ctx, "test-node", "test-group", 2)
	// require.NoError(t, err)
	// assert.NotEmpty(t, neighbors)
}

func TestKuzuDriver_UpsertNode(t *testing.T) {
	dbPath := createTempKuzuDB(t)
	d, err := driver.NewKuzuDriver(dbPath, 1)
	require.NoError(t, err)
	defer d.Close()

	ctx := context.Background()

	// Create indices for the database
	err = d.CreateIndices(ctx)
	require.NoError(t, err)

	// Create a test node
	now := time.Now()
	testNode := &types.Node{
		ID:         "test-node-123",
		Name:       "Test Entity",
		Type:       types.EntityNodeType,
		GroupID:    "test-group",
		EntityType: "Person",
		Summary:    "A test entity for UpsertNode",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Upsert the node
	err = d.UpsertNode(ctx, testNode)
	require.NoError(t, err, "UpsertNode should succeed")

	// Read the node back from the database
	retrievedNode, err := d.GetNode(ctx, testNode.ID, testNode.GroupID)
	require.NoError(t, err, "GetNode should succeed")
	require.NotNil(t, retrievedNode, "Retrieved node should not be nil")

	// Verify the node data matches
	assert.Equal(t, testNode.ID, retrievedNode.ID, "Node ID should match")
	assert.Equal(t, testNode.Name, retrievedNode.Name, "Node name should match")
	assert.Equal(t, testNode.Type, retrievedNode.Type, "Node type should match")
	assert.Equal(t, testNode.GroupID, retrievedNode.GroupID, "Node GroupID should match")
	assert.Equal(t, testNode.EntityType, retrievedNode.EntityType, "Node EntityType should match")
	assert.Equal(t, testNode.Summary, retrievedNode.Summary, "Node summary should match")

	// Test updating the same node (upsert should update existing)
	testNode.Summary = "Updated summary for test entity"
	testNode.UpdatedAt = time.Now()

	err = d.UpsertNode(ctx, testNode)
	require.NoError(t, err, "Second UpsertNode (update) should succeed")

	// Read the updated node back
	updatedNode, err := d.GetNode(ctx, testNode.ID, testNode.GroupID)
	require.NoError(t, err, "GetNode after update should succeed")
	require.NotNil(t, updatedNode, "Updated node should not be nil")

	// Verify the update was applied
	assert.Equal(t, "Updated summary for test entity", updatedNode.Summary, "Node summary should be updated")
	assert.Equal(t, testNode.ID, updatedNode.ID, "Node ID should remain the same")
	assert.Equal(t, testNode.Name, updatedNode.Name, "Node name should remain the same")
}