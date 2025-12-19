# Checkpoint Package

This package provides checkpoint/resume functionality for the `addEpisodeChunked` pipeline in go-predicato. It allows you to save the state of a partially processed episode and resume from where it left off if processing fails due to temporary issues (like LLM timeouts, rate limits, or network errors).

## Features

- **Automatic state persistence** - Save episode processing state at each pipeline step
- **Resume from failure** - Continue processing from the last successful step
- **Retry management** - Track attempt counts and prevent infinite retries
- **Error tracking** - Record errors and stack traces for debugging
- **Cleanup utilities** - Remove old or completed checkpoints
- **Statistics** - Monitor checkpoint health and identify stalled processes

## Architecture

### Processing Pipeline Steps

The `addEpisodeChunked` pipeline is broken down into 13 steps:

1. **StepInitial** - Starting point
2. **StepPrepared** - Episode validated and chunks prepared
3. **StepGotPreviousEpisodes** - Previous episodes retrieved for context
4. **StepCreatedChunks** - Chunk episode structures created
5. **StepExtractedEntities** - Entities extracted from chunks
6. **StepDeduplicatedEntities** - Entities deduplicated across chunks
7. **StepExtractedEdges** - Relationships extracted
8. **StepResolvedEdges** - Relationships resolved and persisted
9. **StepExtractedAttributes** - Entity attributes extracted
10. **StepBuiltEpisodicEdges** - Episodic edges built
11. **StepPerformedGraphUpdate** - Final graph updates performed
12. **StepUpdatedCommunities** - Communities updated
13. **StepCompleted** - Processing complete

### Checkpoint Data Structure

The `EpisodeCheckpoint` struct stores:
- Episode identification (ID, group ID)
- Current processing step
- Timestamp tracking (created, last updated)
- Retry tracking (attempt count, errors)
- Original episode and options
- Intermediate results from each step (chunks, nodes, edges, etc.)

## Usage

### Basic Usage

```go
import (
    "context"
    "github.com/soundprediction/go-predicato/pkg/checkpoint"
)

// Create checkpoint manager
manager, err := checkpoint.NewCheckpointManager("")
if err != nil {
    return err
}

// Create new checkpoint for episode
cp := checkpoint.NewCheckpoint(episode, options, maxCharacters)

// Save checkpoint
if err := manager.Save(ctx, cp); err != nil {
    return err
}

// Update step as processing progresses
cp.Step = checkpoint.StepExtractedEntities
cp.ExtractedNodesByChunk = extractedNodes
if err := manager.Save(ctx, cp); err != nil {
    return err
}

// Load checkpoint to resume
cp, err = manager.Load(ctx, episodeID)
if err != nil {
    return err
}

// Delete checkpoint when complete
if err := manager.Delete(ctx, episodeID); err != nil {
    return err
}
```

### Resume from Checkpoint

```go
// Try to load existing checkpoint
cp, err := manager.Load(ctx, episode.ID)
if err != nil {
    return err
}

if cp != nil {
    // Checkpoint exists - resume from last step
    log.Printf("Resuming episode %s from step %s", cp.EpisodeID, cp.Step)

    // Check if retryable
    if !cp.CanRetry(maxAttempts, maxAge) {
        return fmt.Errorf("episode %s exceeded retry limits", cp.EpisodeID)
    }

    // Resume processing from current step
    return resumeProcessing(ctx, cp)
} else {
    // No checkpoint - start fresh
    cp = checkpoint.NewCheckpoint(episode, options, maxCharacters)
    if err := manager.Save(ctx, cp); err != nil {
        return err
    }
    return startProcessing(ctx, cp)
}
```

### Error Handling

```go
// Record error in checkpoint
if err := processStep(ctx); err != nil {
    // Save error to checkpoint
    if saveErr := manager.SaveWithError(ctx, cp, err); saveErr != nil {
        log.Printf("Failed to save error: %v", saveErr)
    }
    return err
}

// Check if error is recoverable
if cp.IsRecoverable() {
    // Can retry this step
    log.Printf("Step %s failed but is recoverable, will retry", cp.Step)
} else {
    // Non-recoverable error
    log.Printf("Step %s failed with non-recoverable error", cp.Step)
}
```

### Helper Methods

```go
// Load or create checkpoint
cp, existed, err := manager.LoadOrCreate(ctx, episode, options, maxCharacters)
if err != nil {
    return err
}
if existed {
    log.Printf("Resuming from existing checkpoint at step %s", cp.Step)
}

// Update step and save
if err := manager.SaveWithStep(ctx, cp, checkpoint.StepExtractedEntities); err != nil {
    return err
}

// Get progress percentage
progress := cp.GetProgress() // Returns "38% (extracted_entities)"

// Get human-readable summary
summary := cp.Summary()
fmt.Println(summary)
```

### Monitoring and Cleanup

```go
// Get statistics about all checkpoints
stats, err := manager.GetStatistics(ctx, maxAttempts, stalledDuration)
if err != nil {
    return err
}
fmt.Printf("Total: %d, Completed: %d, In Progress: %d, Failed: %d, Stalled: %d\n",
    stats.Total, stats.Completed, stats.InProgress, stats.Failed, stats.Stalled)

// Find stalled checkpoints (not updated in 1 hour)
stalled, err := manager.FindStalled(ctx, 1*time.Hour)
if err != nil {
    return err
}
for _, cp := range stalled {
    log.Printf("Stalled episode: %s at step %s", cp.EpisodeID, cp.Step)
}

// Find failed checkpoints (exceeded max attempts)
failed, err := manager.FindFailed(ctx, 3)
if err != nil {
    return err
}

// Clean old checkpoints (older than 7 days)
removed, err := manager.CleanOld(ctx, 7*24*time.Hour)
if err != nil {
    return err
}
log.Printf("Removed %d old checkpoints", removed)

// List all checkpoints
checkpoints, err := manager.List(ctx)
if err != nil {
    return err
}
for _, cp := range checkpoints {
    fmt.Printf("%s: %s\n", cp.EpisodeID, cp.GetProgress())
}
```

## Configuration

### Checkpoint Directory

By default, checkpoints are stored in `$TMPDIR/predicato-checkpoints`. You can customize this:

```go
// Use custom directory
manager, err := checkpoint.NewCheckpointManager("/path/to/checkpoints")

// Use default temp directory
manager, err := checkpoint.NewCheckpointManager("")
```

### Retry Limits

Configure retry behavior using the `CanRetry` method:

```go
const (
    maxAttempts = 3                  // Maximum retry attempts
    maxAge      = 24 * time.Hour     // Maximum checkpoint age
)

if !cp.CanRetry(maxAttempts, maxAge) {
    // Checkpoint exceeded limits
    return fmt.Errorf("episode processing failed after %d attempts", cp.AttemptCount)
}
```

## Integration with addEpisodeChunked

To integrate checkpoint support into `addEpisodeChunked`:

```go
func (c *Client) addEpisodeChunked(ctx context.Context, episode types.Episode, options *AddEpisodeOptions, maxCharacters int) (*types.AddEpisodeResults, error) {
    manager, err := checkpoint.NewCheckpointManager("")
    if err != nil {
        return nil, err
    }

    // Load or create checkpoint
    cp, existed, err := manager.LoadOrCreate(ctx, episode, options, maxCharacters)
    if err != nil {
        return nil, err
    }

    // Resume from checkpoint or start fresh
    defer func() {
        if r := recover(); r != nil {
            // Save panic to checkpoint
            manager.SaveWithError(ctx, cp, fmt.Errorf("panic: %v", r))
            panic(r)
        }
    }()

    // STEP 1: Prepare and validate (skip if already done)
    if cp.Step == checkpoint.StepInitial {
        chunks, err := c.prepareAndValidateEpisode(&episode, options, maxCharacters)
        if err != nil {
            manager.SaveWithError(ctx, cp, err)
            return nil, err
        }
        cp.Chunks = chunks
        manager.SaveWithStep(ctx, cp, checkpoint.StepPrepared)
    }

    // STEP 2: Get previous episodes (skip if already done)
    if cp.Step == checkpoint.StepPrepared {
        previousEpisodes, err := c.getPreviousEpisodesForContext(ctx, episode, options)
        if err != nil {
            manager.SaveWithError(ctx, cp, err)
            return nil, err
        }
        cp.PreviousEpisodes = previousEpisodes
        manager.SaveWithStep(ctx, cp, checkpoint.StepGotPreviousEpisodes)
    }

    // ... continue for each step ...

    // Delete checkpoint when complete
    if err := manager.Delete(ctx, episode.ID); err != nil {
        c.logger.Warn("Failed to delete checkpoint", "episode_id", episode.ID, "error", err)
    }

    return result, nil
}
```

## Best Practices

1. **Save after expensive operations** - Save checkpoints after LLM calls, embeddings, or large data processing
2. **Atomic saves** - The checkpoint manager uses atomic writes (temp file + rename) to prevent corruption
3. **Clean up completed** - Delete checkpoints for successfully completed episodes
4. **Monitor stalled** - Periodically check for stalled checkpoints and retry or clean them up
5. **Set retry limits** - Use reasonable retry limits (3-5 attempts) and age limits (24-48 hours)
6. **Log checkpoint usage** - Log when resuming from checkpoints for visibility

## Troubleshooting

### Checkpoint file corruption

If a checkpoint file is corrupted, it will be skipped during `List()`. Delete the file manually:

```bash
rm $TMPDIR/predicato-checkpoints/checkpoint_<episode-id>.json
```

### Disk space

Checkpoints can be large if episodes contain many entities. Monitor disk usage and clean old checkpoints:

```go
// Clean checkpoints older than 24 hours
removed, err := manager.CleanOld(ctx, 24*time.Hour)
```

### Stalled checkpoints

If a checkpoint hasn't been updated in a while, it may be stalled:

```go
stalled, err := manager.FindStalled(ctx, 1*time.Hour)
// Investigate or retry
```

## Performance Considerations

- Checkpoint saves are O(n) where n is the size of intermediate data
- Use JSON serialization (human-readable but slower than binary)
- Large episodes (1000+ entities) may produce checkpoints of 1-10 MB
- Consider compressing checkpoint files for long-term storage

## Future Enhancements

Potential improvements:
- Binary serialization for faster saves/loads
- Compression for large checkpoints
- Remote storage backends (S3, database)
- Distributed locking for multi-process safety
- Checkpoint versioning for schema evolution
