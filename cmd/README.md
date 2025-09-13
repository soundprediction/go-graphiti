# Go-Graphiti CLI

The Go-Graphiti CLI provides command-line access to the Graphiti knowledge graph framework.

## Installation

```bash
go build -o graphiti ./cmd/main.go
```

## Configuration

Configuration can be provided through:
1. Configuration files (YAML format)
2. Environment variables
3. Command-line flags

### Configuration File

Copy `.graphiti.example.yaml` to `.graphiti.yaml` and customize:

```yaml
# Server configuration
server:
  host: localhost
  port: 8080
  mode: debug

# Database configuration  
database:
  driver: neo4j
  uri: bolt://localhost:7687
  username: neo4j
  password: password
  database: neo4j

# LLM configuration
llm:
  provider: openai
  model: gpt-4
  api_key: "" # Set via OPENAI_API_KEY environment variable
  temperature: 0.1
  max_tokens: 2048

# Embedding configuration
embedding:
  provider: openai
  model: text-embedding-3-small
  api_key: "" # Set via OPENAI_API_KEY environment variable
```

### Environment Variables

Key environment variables:
- `OPENAI_API_KEY` - OpenAI API key for LLM and embeddings
- `NEO4J_URI` - Neo4j database URI
- `NEO4J_USER` - Neo4j username
- `NEO4J_PASSWORD` - Neo4j password
- `SERVER_HOST` - Server host
- `SERVER_PORT` - Server port

## Commands

### Server

Start the HTTP server:

```bash
./graphiti server
```

With custom configuration:

```bash
./graphiti server --port 9090 --llm-api-key your-key-here
```

The server provides REST API endpoints:

- `GET /health` - Health check
- `POST /api/v1/ingest/messages` - Add messages to knowledge graph
- `POST /api/v1/search` - Search the knowledge graph
- `GET /api/v1/episodes/:group_id` - Get episodes for a group
- `POST /api/v1/get-memory` - Get memory based on messages

### Version

Show version information:

```bash
./graphiti version
```

### Help

Get help for any command:

```bash
./graphiti --help
./graphiti server --help
```

## API Examples

### Add Messages

```bash
curl -X POST http://localhost:8080/api/v1/ingest/messages \
  -H "Content-Type: application/json" \
  -d '{
    "group_id": "user123",
    "messages": [
      {
        "role": "user",
        "content": "Hello, I work at Acme Corp"
      }
    ]
  }'
```

### Search

```bash
curl -X POST http://localhost:8080/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Acme Corp",
    "group_ids": ["user123"],
    "max_facts": 10
  }'
```

### Health Check

```bash
curl http://localhost:8080/health
```

## Development

### Building

```bash
go build ./cmd/...
```

### Testing

```bash
go test ./...
```

### Running with Development Config

```bash
export OPENAI_API_KEY=your-key-here
go run ./cmd/main.go server --mode debug
```