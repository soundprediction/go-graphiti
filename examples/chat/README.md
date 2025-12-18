# Interactive Chat Example

This example demonstrates how to build an interactive chat application using go-graphiti with both global knowledge base and user-specific episodic memory.

## Features

- **Dual Knowledge Stores**: Separates global knowledge (shared facts) from user-specific episodic memory
- **Conversation Continuity**: Uses `AddToEpisode` to maintain a single episode per chat session
- **UUID v7 Episode IDs**: Leverages time-sortable UUIDs for natural episode ordering
- **Hybrid Search**: Combines global knowledge base search with conversation history
- **Interactive Commands**: Supports history viewing and direct knowledge base queries

## Architecture

The example creates two separate Graphiti clients:

1. **Global Graphiti Client** (optional):
   - Shared knowledge base across all users
   - Read-only for chat purposes
   - Used for contextual information retrieval

2. **User Graphiti Client**:
   - User-specific episodic memory
   - Stores conversation history
   - One episode per chat session

## Prerequisites

- OpenAI API key for LLM and embeddings
- Go 1.21 or later

## Setup

1. Set your OpenAI API key:
```bash
export OPENAI_API_KEY=your_api_key_here
```

2. (Optional) Populate a global knowledge base:
```bash
# Use the basic example or another method to create a knowledge graph
# at ./knowledge_db/content_graph.ladybugdb
```

## Usage

### Basic Usage

Run with default settings (creates user database in `./user_dbs/`):

```bash
go run main.go
```

### With Custom User ID

```bash
go run main.go --user-id bob
```

### With Custom Database Paths

```bash
go run main.go \
  --user-id alice \
  --global-db /path/to/global/knowledge.ladybugdb \
  --user-db-dir /path/to/user/databases
```

### Without Global Knowledge Base

```bash
go run main.go --skip-global
```

## Interactive Commands

Once the chat is running, you can use these commands:

- `<your question>` - Ask the assistant a question
- `history` - View conversation history
- `search <query>` - Search the global knowledge base directly
- `exit` or `quit` - End the chat session

## Example Session

```
🚀 Starting Graphiti Chat Example
   User ID: alice

🔧 Initializing clients...
   ✅ LLM client created (model: gpt-4o-mini)
   ✅ Embedder client created (model: text-embedding-3-small)
   ✅ Global knowledge base loaded from ./knowledge_db/content_graph.ladybugdb
   ✅ User database initialized at ./user_dbs/user_alice.ladybugdb

======================================================================
💬 Graphiti Interactive Chat
======================================================================

Commands:
  Type your question and press Enter
  Type 'exit' or 'quit' to end the session
  Type 'history' to view conversation history
  Type 'search <query>' to search the global knowledge base
======================================================================

💬 You: What is GraphQL?
✨ Created episode: 01930e1c-3a4f-7b2a-8c5d-1e2f3a4b5c6d
🔍 Searching global knowledge base...
📚 Found 2 relevant nodes
  1. GraphQL: A query language for APIs developed by Facebook...
  2. API Design: Best practices for designing application programming interfaces...

🤖 Assistant:
----------------------------------------------------------------------
GraphQL is a query language for APIs developed by Facebook. It provides
a more flexible approach to API development compared to traditional REST...
----------------------------------------------------------------------

💬 You: history
📝 Conversation History:
----------------------------------------------------------------------
1. You: What is GraphQL?
   Assistant: GraphQL is a query language for APIs developed by Facebook...
----------------------------------------------------------------------

💬 You: exit
👋 Goodbye!
```

## Key Concepts Demonstrated

### 1. Episode Management with UUID v7

```go
episodeID, err := uuid.NewV7()
if err != nil {
    episodeID = uuid.New() // Fallback to v4
}

episode := types.Episode{
    ID:        episodeID.String(),
    Name:      fmt.Sprintf("Chat with %s", userID),
    Content:   conversationTurn,
    GroupID:   fmt.Sprintf("user-%s-chat", userID),
    Metadata:  map[string]interface{}{"session_id": session.SessionID},
    Reference: time.Now(),
}
```

### 2. Conversation Continuity with AddToEpisode

First message creates the episode:
```go
result, err := clients.UserGraphiti.Add(ctx, []types.Episode{episode}, nil)
```

Subsequent messages append to it:
```go
_, err := clients.UserGraphiti.AddToEpisode(ctx, session.EpisodeID, conversationTurn, nil)
```

### 3. Hybrid Search Pattern

```go
// Search global knowledge base
results, err := clients.GlobalGraphiti.Search(ctx, input, searchConfig)

// Build prompt with both:
// - Context from knowledge base search results
// - Recent conversation history
prompt := buildPrompt(input, session.Messages, contextNodes)
```

### 4. Separate Client Management

```go
type ChatClients struct {
    GlobalGraphiti *graphiti.Client // Optional, shared knowledge
    UserGraphiti   *graphiti.Client // Required, user-specific
    LLM            llm.Client
    Context        context.Context
}
```

## File Structure

After running:

```
./
├── knowledge_db/              # Global knowledge base (optional)
│   └── content_graph.ladybugdb/
└── user_dbs/                  # User-specific databases
    ├── user_alice.ladybugdb/
    ├── user_bob.ladybugdb/
    └── ...
```

## Extending This Example

### Adding Knowledge to Global Database

You can populate the global knowledge base using the `Add` method before starting the chat:

```go
globalClient.Add(ctx, []types.Episode{
    {
        ID:        "fact-001",
        Name:      "GraphQL Introduction",
        Content:   "GraphQL is a query language for APIs...",
        Reference: time.Now(),
        GroupID:   "global-knowledge",
    },
}, nil)
```

### Persisting Chat Sessions

The current example keeps sessions in memory. To persist across runs, you could:

1. Store session metadata in a simple JSON file
2. Load previous episode IDs on startup
3. Use `AddToEpisode` to continue previous conversations

### Multi-User Support

The example already supports multiple users through the `--user-id` flag. Each user gets their own ladybug database file, ensuring data isolation.

## Notes

- User databases are created automatically on first use
- Each user's conversation history is stored in their own ladybug database
- The global knowledge base is optional and read-only during chat
- Episode IDs use UUID v7 for temporal ordering
- All conversation turns are appended to a single episode per session
