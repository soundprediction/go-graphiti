# LadybugDriver Write Queue - Usage Guide

The LadybugDriver now includes transparent write queue handling that automatically manages concurrent writes to the Ladybug database, which cannot safely handle concurrent write operations.

## Overview

The write queue system provides:
- **Automatic write queuing** - No code changes needed
- **Thread-safe writes** - All write operations execute sequentially
- **Concurrent reads** - Read operations don't block on the write queue
- **Configurable buffering** - Adjust queue size for your workload

## Quick Start

### Basic Usage (Same as Before)

```go
package main

import (
    "context"
    "log"
    "github.com/soundprediction/go-graphiti/pkg/driver"
)

func main() {
    // Create driver with default settings (1000-operation write queue)
    ladybugDriver, err := driver.NewLadybugDriver("./my_db", 4)
    if err != nil {
        log.Fatal(err)
    }
    defer ladybugDriver.Close()

    ctx := context.Background()

    // Write operations are automatically queued
    _, _, _, err = ladybugDriver.ExecuteQuery(
        "CREATE (n:Entity {uuid: $uuid, name: $name})",
        map[string]interface{}{
            "uuid": "id1",
            "name": "Alice",
        },
    )
    if err != nil {
        log.Fatal(err)
    }

    // Multiple concurrent writes work seamlessly
    for i := 0; i < 100; i++ {
        go ladybugDriver.ExecuteQuery(
            "CREATE (n:Entity {uuid: $uuid, name: $name})",
            map[string]interface{}{
                "uuid": fmt.Sprintf("id%d", i),
                "name": fmt.Sprintf("Person%d", i),
            },
        )
    }

    // Read operations execute immediately (don't wait in queue)
    result, _, _, err := ladybugDriver.ExecuteQuery(
        "MATCH (n:Entity) RETURN count(n) as count",
        nil,
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Result: %v", result)
}
```

### Advanced Configuration

For fine-tuned control, use `NewLadybugDriverWithConfig`:

```go
package main

import (
    "log"
    "github.com/soundprediction/go-graphiti/pkg/driver"
)

func main() {
    // Create custom configuration
    config := driver.DefaultLadybugDriverConfig().
        WithDBPath("./my_db").
        WithMaxConcurrentQueries(8).
        WithWriteQueueSize(5000).                    // Larger queue for high write throughput
        WithBufferPoolSize(2 * 1024 * 1024 * 1024). // 2GB buffer pool
        WithCompression(true).
        WithMaxDbSize(1 << 44)                       // 16TB max DB size

    ladybugDriver, err := driver.NewLadybugDriverWithConfig(config)
    if err != nil {
        log.Fatal(err)
    }
    defer ladybugDriver.Close()

    // Use driver as normal...
}
```

## Configuration Options

### LadybugDriverConfig Fields

| Field | Default | Description |
|-------|---------|-------------|
| `DBPath` | `:memory:` | Database file path or `:memory:` for in-memory |
| `MaxConcurrentQueries` | `1` | Maximum concurrent read operations |
| `WriteQueueSize` | `1000` | Write operation buffer size |
| `BufferPoolSize` | `1GB` | Memory buffer for database operations |
| `EnableCompression` | `true` | Enable data compression |
| `MaxDbSize` | `8TB` | Maximum database size |

### Builder Methods

All builder methods return `*LadybugDriverConfig` for chaining:

```go
config := driver.DefaultLadybugDriverConfig().
    WithDBPath("./my_db").
    WithMaxConcurrentQueries(8).
    WithWriteQueueSize(5000).
    WithBufferPoolSize(2 * 1024 * 1024 * 1024).
    WithCompression(true).
    WithMaxDbSize(1 << 44)
```

## Write Queue Behavior

### How It Works

1. **Write Detection**: Queries are analyzed for write keywords (CREATE, MERGE, SET, DELETE, etc.)
2. **Queue Routing**:
   - **Write queries** → Sent to write queue → Executed sequentially by worker goroutine
   - **Read queries** → Execute immediately with mutex protection
3. **Result Delivery**: Write operations block until their result is available (appears synchronous)

### Write Keywords Detected

The system detects these write operations:
- `CREATE` - Create nodes or relationships
- `MERGE` - Merge nodes or relationships
- `SET` - Set properties
- `DELETE` - Delete nodes or relationships
- `DETACH DELETE` - Delete with relationship cleanup
- `REMOVE` - Remove properties or labels
- `DROP` - Drop indexes or constraints
- `INSERT` - Insert data
- `UPDATE` - Update data

### Performance Characteristics

#### Write Operations
- **Latency**: Slightly increased (queuing overhead ~1-5ms)
- **Throughput**: Same as sequential (limited by Ladybug's thread safety)
- **Concurrency**: Transparent - callers can issue concurrent writes safely

#### Read Operations
- **Latency**: Unchanged (direct execution with mutex)
- **Throughput**: Improved (not blocked by queued writes)
- **Concurrency**: Multiple reads can execute while writes are queued

## Tuning the Write Queue

### Small Queue (100-500)
**Use when:**
- Low write volume
- Memory constrained
- Need predictable latency

**Pros:**
- Lower memory usage
- Faster timeout detection
- Predictable behavior

**Cons:**
- May block on bursts

### Default Queue (1000)
**Use when:**
- General purpose applications
- Moderate write volume
- Balanced memory/performance

**Pros:**
- Good burst handling
- Reasonable memory usage
- Works for most cases

**Cons:**
- May still block on heavy bursts

### Large Queue (5000+)
**Use when:**
- High write throughput
- Bursty write patterns
- Memory available

**Pros:**
- Excellent burst absorption
- Smooth performance under load
- Reduces blocking

**Cons:**
- Higher memory usage
- Longer shutdown time (draining queue)

### Memory Usage Calculation

Each queued operation uses approximately:
```
~200 bytes (writeOperation struct + string buffers + result channel)
```

For a 5000-operation queue:
```
5000 * 200 bytes = ~1 MB
```

Memory usage is minimal compared to Ladybug's buffer pool.

## Examples by Use Case

### High-Volume Indexing

```go
config := driver.DefaultLadybugDriverConfig().
    WithDBPath("./large_db").
    WithWriteQueueSize(10000).              // Large queue for bursts
    WithBufferPoolSize(4 * 1024 * 1024 * 1024) // 4GB buffer

driver, _ := driver.NewLadybugDriverWithConfig(config)
defer driver.Close()

// Process 100K documents concurrently
for i := 0; i < 100000; i++ {
    go func(id int) {
        driver.ExecuteQuery(
            "CREATE (n:Document {uuid: $uuid, content: $content})",
            map[string]interface{}{
                "uuid":    fmt.Sprintf("doc%d", id),
                "content": generateContent(id),
            },
        )
    }(i)
}
```

### Real-Time Updates with Reads

```go
config := driver.DefaultLadybugDriverConfig().
    WithDBPath("./realtime_db").
    WithMaxConcurrentQueries(8).  // More concurrent reads
    WithWriteQueueSize(2000)       // Moderate write buffer

driver, _ := driver.NewLadybugDriverWithConfig(config)
defer driver.Close()

// Concurrent writes don't block reads
go func() {
    for update := range updatesChan {
        driver.ExecuteQuery(
            "MERGE (n:Entity {uuid: $uuid}) SET n.value = $value",
            map[string]interface{}{
                "uuid":  update.ID,
                "value": update.Value,
            },
        )
    }
}()

// Reads execute immediately
go func() {
    for query := range queriesChan {
        result, _, _, _ := driver.ExecuteQuery(
            "MATCH (n:Entity {uuid: $uuid}) RETURN n.value",
            map[string]interface{}{"uuid": query.ID},
        )
        resultsChan <- result
    }
}()
```

### Memory-Constrained Environment

```go
config := driver.DefaultLadybugDriverConfig().
    WithDBPath("./small_db").
    WithWriteQueueSize(100).                       // Small queue
    WithBufferPoolSize(256 * 1024 * 1024).        // 256MB buffer
    WithCompression(true)                          // Enable compression

driver, _ := driver.NewLadybugDriverWithConfig(config)
defer driver.Close()

// Still handles concurrent writes, just with smaller buffer
// May block if more than 100 writes are pending
```

## Error Handling

### Write Queue Timeout

If the write queue is full for more than 30 seconds:

```go
_, _, _, err := driver.ExecuteQuery("CREATE (n:Entity {uuid: $uuid})", params)
if err != nil {
    if strings.Contains(err.Error(), "write queue timeout") {
        log.Printf("Write queue is full - system may be overwhelmed")
        // Consider:
        // 1. Increasing WriteQueueSize
        // 2. Reducing write rate
        // 3. Checking for slow queries
    }
}
```

### Driver Closed

Attempting to use a closed driver:

```go
driver.Close()

_, _, _, err := driver.ExecuteQuery("MATCH (n) RETURN n", nil)
if err != nil {
    if strings.Contains(err.Error(), "driver is closed") {
        log.Printf("Cannot execute query on closed driver")
    }
}
```

## Shutdown Behavior

The driver ensures graceful shutdown:

```go
driver, _ := driver.NewLadybugDriver("./my_db", 4)

// Queue 1000 writes
for i := 0; i < 1000; i++ {
    go driver.ExecuteQuery(
        "CREATE (n:Entity {uuid: $uuid})",
        map[string]interface{}{"uuid": fmt.Sprintf("id%d", i)},
    )
}

// Close waits for all queued writes to complete
driver.Close() // Blocks until write queue is drained

// All 1000 writes are guaranteed to be processed
```

### Shutdown Process

1. Mark driver as closed (reject new operations)
2. Signal write worker to finish
3. Drain remaining operations from queue
4. Clean up resources (temp files, connections)
5. Return

**Timeout**: No timeout - ensures all data is persisted

## Best Practices

1. **Use Default Queue Size** for most applications (1000 operations)
2. **Increase Queue Size** if you see timeout errors or have bursty writes
3. **Monitor Memory** if using very large queues (10000+)
4. **Profile Your Workload** to find optimal settings
5. **Always defer Close()** to ensure graceful shutdown
6. **Handle Timeouts** if your application has extreme write bursts

## Migration from Old Code

No code changes required! The write queue is completely transparent:

```go
// Old code (works exactly the same)
driver, _ := driver.NewLadybugDriver("./db", 4)
defer driver.Close()

driver.ExecuteQuery("CREATE (n:Entity {uuid: 'id1'})", nil)
```

If you want to optimize:

```go
// New code (same behavior, optimized config)
config := driver.DefaultLadybugDriverConfig().
    WithDBPath("./db").
    WithWriteQueueSize(5000)  // Tune for your workload

driver, _ := driver.NewLadybugDriverWithConfig(config)
defer driver.Close()

driver.ExecuteQuery("CREATE (n:Entity {uuid: 'id1'})", nil)
```

## Troubleshooting

### Writes are slow

**Symptom**: Write operations take a long time

**Possible causes**:
1. Write queue is full (check for timeout errors)
2. Ladybug database is slow (check disk I/O, buffer pool size)
3. Complex queries (optimize Cypher queries)

**Solutions**:
- Increase `WriteQueueSize`
- Increase `BufferPoolSize`
- Optimize queries
- Use batch operations when possible

### Memory usage is high

**Symptom**: Application uses more memory than expected

**Possible causes**:
1. Large write queue buffer
2. Many queued operations with large parameters
3. Large buffer pool

**Solutions**:
- Decrease `WriteQueueSize` if not needed
- Decrease `BufferPoolSize` if workload allows
- Monitor queue depth

### Reads are blocked

**Symptom**: Read operations are slow despite write queue

**Possible causes**:
1. Mutex contention (many reads competing)
2. Long-running read queries
3. Database lock contention

**Solutions**:
- Increase `MaxConcurrentQueries`
- Optimize read queries
- Use indices for common queries
- Consider read replicas for heavy read workloads

## FAQ

**Q: Do I need to change my code to use the write queue?**
A: No, it's completely transparent. Existing code works without modification.

**Q: Can I disable the write queue?**
A: No, it's always active for thread safety. You can set `WriteQueueSize` to 1 for minimal buffering.

**Q: What happens if the queue fills up?**
A: New write operations will block until space is available, or timeout after 30 seconds.

**Q: Are reads affected by the write queue?**
A: No, reads execute directly with mutex protection and don't enter the queue.

**Q: How do I know if my queue size is right?**
A: Monitor for timeout errors. If you see them, increase the queue size. Otherwise, the default (1000) works well.

**Q: Does this work with transactions?**
A: Yes, but each ExecuteQuery call is independent. For true transactions, use the Session API.

**Q: What's the performance impact?**
A: Write latency increases by 1-5ms (queuing overhead). Read performance improves (not blocked by queue).
