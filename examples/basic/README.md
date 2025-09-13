# Basic Go-Graphiti Example

This example demonstrates the basic usage of go-graphiti with OpenAI LLM and Neo4j database.

## Features

This example shows how to:
- Create and configure a Graphiti client with Neo4j and OpenAI
- Add episodes (data) to the knowledge graph
- Search the knowledge graph for relevant information

## Prerequisites

Before running this example, you need:

1. **Neo4j Database**: A running Neo4j instance
   - Default connection: `bolt://localhost:7687`
   - You can run Neo4j locally using Docker:
     ```bash
     docker run --name neo4j -p 7687:7687 -p 7474:7474 -e NEO4J_AUTH=neo4j/password neo4j:latest
     ```

2. **OpenAI API Key**: An active OpenAI API key for LLM and embedding services

## Environment Variables

### Required
- `OPENAI_API_KEY`: Your OpenAI API key
- `NEO4J_PASSWORD`: Your Neo4j database password

### Optional
- `NEO4J_URI`: Neo4j connection URI (default: `bolt://localhost:7687`)
- `NEO4J_USER`: Neo4j username (default: `neo4j`)

## Usage

1. Set up your environment variables:
   ```bash
   export OPENAI_API_KEY=your_openai_api_key_here
   export NEO4J_PASSWORD=your_neo4j_password_here
   ```

2. Run the example:
   ```bash
   go run .
   ```

3. Or build and run:
   ```bash
   go build -o basic_example .
   ./basic_example
   ```

## Example Output

When run successfully, you'll see output similar to:
```
🚀 Starting go-graphiti basic example
   Neo4j URI: bolt://localhost:7687
   Neo4j User: neo4j

📊 Creating Neo4j driver...
   ✅ Neo4j driver created successfully

🧠 Creating OpenAI LLM client...
   ✅ OpenAI LLM client created (model: gpt-4o-mini)

🔤 Creating OpenAI embedder client...
   ✅ OpenAI embedder client created (model: text-embedding-3-small)

🌐 Creating Graphiti client...
   ✅ Graphiti client created (group: example-group)

📝 Preparing sample episodes...
Adding episodes to the knowledge graph...
✅ Episodes successfully added to the knowledge graph!

Searching the knowledge graph...
✅ Found 2 nodes and 1 edges

Sample nodes found:
  - Meeting with Alice (episode)
  - Project Research (episode)

Example completed successfully!
```

## Testing

Run the tests to verify everything works correctly:
```bash
go test -v ./...
```

## Troubleshooting

### Missing Environment Variables
If you see error messages about missing environment variables, the example will provide helpful instructions on how to set them up.

### Neo4j Connection Issues
- Ensure Neo4j is running and accessible at the configured URI
- Verify your username and password are correct
- Check that the Neo4j ports (7687, 7474) are not blocked by a firewall

### OpenAI API Issues
- Verify your API key is valid and has sufficient credits
- Check your OpenAI API usage limits

## What This Example Demonstrates

1. **Client Setup**: How to create and configure all the necessary clients (Neo4j, OpenAI LLM, OpenAI Embedder, Graphiti)
2. **Data Ingestion**: Adding structured episodes to the knowledge graph
3. **Information Retrieval**: Searching the knowledge graph with natural language queries
4. **Error Handling**: Graceful handling of missing dependencies or configuration issues