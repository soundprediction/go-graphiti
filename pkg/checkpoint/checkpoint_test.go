package checkpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpointManager(t *testing.T) {
	// Create temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "graphiti-checkpoint-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	t.Run("Create manager with custom directory", func(t *testing.T) {
		manager, err := NewCheckpointManager(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, tmpDir, manager.GetCheckpointDir())
	})

	t.Run("Create manager with default directory", func(t *testing.T) {
		manager, err := NewCheckpointManager("")
		require.NoError(t, err)
		expectedDir := filepath.Join(os.TempDir(), "graphiti-checkpoints")
		assert.Equal(t, expectedDir, manager.GetCheckpointDir())
	})

	t.Run("Save and load checkpoint", func(t *testing.T) {
		manager, err := NewCheckpointManager(tmpDir)
		require.NoError(t, err)

		checkpoint := &EpisodeCheckpoint{
			EpisodeID:     "episode-123",
			GroupID:       "group-456",
			Step:          StepPrepared,
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
			Episode: types.Episode{
				ID:      "episode-123",
				Name:    "Test Episode",
				Content: "Test content",
				GroupID: "group-456",
			},
			Chunks: []string{"chunk1", "chunk2"},
		}

		// Save checkpoint
		err = manager.Save(ctx, checkpoint)
		require.NoError(t, err)

		// Load checkpoint
		loaded, err := manager.Load(ctx, "episode-123")
		require.NoError(t, err)
		require.NotNil(t, loaded)

		assert.Equal(t, checkpoint.EpisodeID, loaded.EpisodeID)
		assert.Equal(t, checkpoint.GroupID, loaded.GroupID)
		assert.Equal(t, checkpoint.Step, loaded.Step)
		assert.Equal(t, checkpoint.Episode.ID, loaded.Episode.ID)
		assert.Equal(t, len(checkpoint.Chunks), len(loaded.Chunks))
	})

	t.Run("Load non-existent checkpoint", func(t *testing.T) {
		manager, err := NewCheckpointManager(tmpDir)
		require.NoError(t, err)

		loaded, err := manager.Load(ctx, "non-existent")
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("Delete checkpoint", func(t *testing.T) {
		manager, err := NewCheckpointManager(tmpDir)
		require.NoError(t, err)

		checkpoint := &EpisodeCheckpoint{
			EpisodeID: "episode-delete",
			GroupID:   "group-456",
			Step:      StepPrepared,
			CreatedAt: time.Now(),
		}

		// Save and verify exists
		err = manager.Save(ctx, checkpoint)
		require.NoError(t, err)

		exists, err := manager.Exists(ctx, "episode-delete")
		require.NoError(t, err)
		assert.True(t, exists)

		// Delete and verify doesn't exist
		err = manager.Delete(ctx, "episode-delete")
		require.NoError(t, err)

		exists, err = manager.Exists(ctx, "episode-delete")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Update step", func(t *testing.T) {
		manager, err := NewCheckpointManager(tmpDir)
		require.NoError(t, err)

		checkpoint := &EpisodeCheckpoint{
			EpisodeID: "episode-step",
			GroupID:   "group-456",
			Step:      StepPrepared,
			CreatedAt: time.Now(),
		}

		err = manager.Save(ctx, checkpoint)
		require.NoError(t, err)

		// Update step
		err = manager.UpdateStep(ctx, "episode-step", StepExtractedEntities)
		require.NoError(t, err)

		// Verify updated
		loaded, err := manager.Load(ctx, "episode-step")
		require.NoError(t, err)
		assert.Equal(t, StepExtractedEntities, loaded.Step)
	})

	t.Run("Record error", func(t *testing.T) {
		manager, err := NewCheckpointManager(tmpDir)
		require.NoError(t, err)

		checkpoint := &EpisodeCheckpoint{
			EpisodeID:    "episode-error",
			GroupID:      "group-456",
			Step:         StepExtractedEntities,
			CreatedAt:    time.Now(),
			AttemptCount: 0,
		}

		err = manager.Save(ctx, checkpoint)
		require.NoError(t, err)

		// Record error
		testErr := assert.AnError
		err = manager.RecordError(ctx, "episode-error", testErr, "stack trace here")
		require.NoError(t, err)

		// Verify error recorded
		loaded, err := manager.Load(ctx, "episode-error")
		require.NoError(t, err)
		assert.Equal(t, 1, loaded.AttemptCount)
		assert.Contains(t, loaded.LastError, "assert.AnError")
		assert.Equal(t, "stack trace here", loaded.LastErrorStack)
	})

	t.Run("List checkpoints", func(t *testing.T) {
		manager, err := NewCheckpointManager(tmpDir)
		require.NoError(t, err)

		// Create multiple checkpoints
		for i := 0; i < 3; i++ {
			checkpoint := &EpisodeCheckpoint{
				EpisodeID: fmt.Sprintf("episode-list-%d", i),
				GroupID:   "group-456",
				Step:      StepPrepared,
				CreatedAt: time.Now(),
			}
			err = manager.Save(ctx, checkpoint)
			require.NoError(t, err)
		}

		// List all
		checkpoints, err := manager.List(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(checkpoints), 3)
	})

	t.Run("Clean old checkpoints", func(t *testing.T) {
		manager, err := NewCheckpointManager(tmpDir)
		require.NoError(t, err)

		// Create old checkpoint - manually write with old timestamp
		oldTime := time.Now().Add(-48 * time.Hour)
		oldCheckpoint := &EpisodeCheckpoint{
			EpisodeID:     "episode-old",
			GroupID:       "group-456",
			Step:          StepPrepared,
			CreatedAt:     oldTime,
			LastUpdatedAt: oldTime,
		}
		// Manually write to preserve old timestamp
		data, err := json.MarshalIndent(oldCheckpoint, "", "  ")
		require.NoError(t, err)
		oldPath := manager.GetCheckpointPath("episode-old")
		err = os.WriteFile(oldPath, data, 0644)
		require.NoError(t, err)

		// Create recent checkpoint
		recentCheckpoint := &EpisodeCheckpoint{
			EpisodeID:     "episode-recent",
			GroupID:       "group-456",
			Step:          StepPrepared,
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
		}
		err = manager.Save(ctx, recentCheckpoint)
		require.NoError(t, err)

		// Clean old (older than 24 hours)
		removed, err := manager.CleanOld(ctx, 24*time.Hour)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, removed, 1)

		// Verify old checkpoint is gone but recent remains
		exists, err := manager.Exists(ctx, "episode-old")
		require.NoError(t, err)
		assert.False(t, exists)

		exists, err = manager.Exists(ctx, "episode-recent")
		require.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestProcessingSteps(t *testing.T) {
	steps := []ProcessingStep{
		StepInitial,
		StepPrepared,
		StepGotPreviousEpisodes,
		StepCreatedChunks,
		StepExtractedEntities,
		StepDeduplicatedEntities,
		StepExtractedEdges,
		StepResolvedEdges,
		StepExtractedAttributes,
		StepBuiltEpisodicEdges,
		StepPerformedGraphUpdate,
		StepUpdatedCommunities,
		StepCompleted,
	}

	// Verify all steps are unique
	stepMap := make(map[ProcessingStep]bool)
	for _, step := range steps {
		assert.False(t, stepMap[step], "Duplicate step: %s", step)
		stepMap[step] = true
	}
}
