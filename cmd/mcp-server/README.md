# Go-Predicato MCP Server

A Model Context Protocol (MCP) server implementation for go-predicato using Google's Genkit framework.

## Overview

This MCP server exposes Predicato's temporal knowledge graph functionality through the Model Context Protocol, allowing AI assistants to store, search, and manage dynamic memory using a graph-based approach.

## Features

- **Memory Storage**: Add episodes (text, JSON, conversations) to the knowledge graph
- **Hybrid Search**: Search nodes and facts using semantic similarity and keyword search
- **Entity Management**: Retrieve, delete, and manage entities and relationships
- **Temporal Tracking**: Built-in support for temporal data with validity periods
- **Custom Entity Types**: Support for Requirements, Preferences, and Procedures
- **Multi-tenant**: Group ID support for data isolation

## Installation

1. Ensure you have Go 1.21+ installed
2. Build the server:
   ```bash
   go build ./cmd/mcp-server
   ```

## Configuration

The server can be configured via environment variables or command-line flags:

### Environment Variables

- `OPENAI_API_KEY`: Required for LLM and embedding operations
- `MODEL_NAME`: LLM model to use (default: gpt-4o-mini)
- `EMBEDDER_MODEL_NAME`: Embedding model (default: text-embedding-3-small)
- `DB_DRIVER`: Database driver to use (default: ladybug)
- `DB_URI`: Database connection URI/path (default: ./ladybug_db for ladybug, bolt://localhost:7687 for neo4j)
- `ladybug_DB_PATH`: Path to ladybug database directory (default: ./ladybug_db)
- `NEO4J_USER`: Neo4j username (required when using neo4j driver)
- `NEO4J_PASSWORD`: Neo4j password (required when using neo4j driver)
- `GROUP_ID`: Default group ID for data isolation (default: default)
- `LLM_TEMPERATURE`: Temperature for LLM operations (default: 0.0)
- `SEMAPHORE_LIMIT`: Concurrency limit (default: 10)

### Command Line Flags

```bash
./mcp-server --help
```

Available flags:
- `--group-id`: Namespace for the graph
- `--transport`: Communication transport (stdio or sse)
- `--model`: LLM model name
- `--small-model`: Small LLM model name
- `--temperature`: LLM temperature (0.0-2.0)
- `--destroy-graph`: Destroy all graphs on startup
- `--use-custom-entities`: Enable custom entity extraction
- `--host`: Host to bind to
- `--port`: Port to bind to

## Usage

### Basic Usage

```bash
# Start with default settings (uses ladybug database)
./mcp-server

# Start with custom group ID and destroy existing graph
./mcp-server --group-id my-project --destroy-graph

# Start with custom entities enabled
./mcp-server --use-custom-entities

# Use Neo4j instead of ladybug (requires NEO4J_USER and NEO4J_PASSWORD)
DB_DRIVER=neo4j DB_URI=bolt://localhost:7687 NEO4J_USER=neo4j NEO4J_PASSWORD=password ./mcp-server

# Use custom ladybug database path
ladybug_DB_PATH=/path/to/my/ladybug_db ./mcp-server
```

### Available Tools

The MCP server exposes the following tools:

#### `add_memory`
Add an episode to memory.

Parameters:
- `name` (string): Name of the episode
- `episode_body` (string): Content to store
- `group_id` (string, optional): Group identifier
- `source` (string, optional): Source type (text, json, message)
- `source_description` (string, optional): Description of the source
- `uuid` (string, optional): Custom UUID

#### `search_memory_nodes`
Search for relevant nodes in the graph.

Parameters:
- `query` (string): Search query
- `limit` (int, optional): Maximum results (default: 10)

#### `search_memory_facts`
Search for relevant facts (relationships) in the graph.

Parameters:
- `query` (string): Search query
- `limit` (int, optional): Maximum results (default: 10)

#### `get_entity_edge`
Retrieve a specific entity edge by UUID.

Parameters:
- `uuid` (string): Edge UUID

#### `delete_entity_edge`
Delete an entity edge.

Parameters:
- `uuid` (string): Edge UUID to delete

#### `delete_episode`
Delete an episode.

Parameters:
- `uuid` (string): Episode UUID to delete

#### `get_episodes`
Get recent episodes (placeholder - not yet implemented).

#### `clear_graph`
Clear all data from the graph (placeholder - not yet implemented).

## Examples

### Adding Memory

```json
{
  "name": "Project Requirements",
  "episode_body": "The new web application must support user authentication, real-time notifications, and mobile responsiveness.",
  "source": "text",
  "source_description": "Product requirements document"
}
```

### Searching Nodes

```json
{
  "query": "user authentication requirements",
  "limit": 5
}
```

### Searching Facts

```json
{
  "query": "authentication relationships",
  "limit": 10
}
```

## Architecture

The MCP server is built on:

- **Genkit**: Google's framework for AI applications, handling MCP protocol
- **go-predicato**: Temporal knowledge graph implementation
- **ladybug**: Default graph database backend (high-performance embedded graph database)
- **Neo4j**: Alternative graph database backend (requires separate installation)
- **OpenAI API**: LLM and embedding services

## Custom Entity Types

When `--use-custom-entities` is enabled, the server recognizes:

- **Requirements**: Project or system requirements
- **Preferences**: User preferences and choices  
- **Procedures**: Step-by-step instructions or processes

## Development

The implementation consists of:

- `main.go`: Server initialization and configuration
- `tools.go`: MCP tool implementations
- Integration with go-predicato's search and storage capabilities

## Limitations

Current limitations (TODOs):

- Some advanced search features from Python version not ported
- Limited error handling and validation

## License

This implementation follows the same license as the go-predicato project.