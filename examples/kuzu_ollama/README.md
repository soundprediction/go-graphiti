# Kuzu + Ollama Example

This example demonstrates using go-graphiti with a fully local setup combining:

- **Kuzu**: Embedded graph database (no server required)
- **Ollama**: Local LLM inference (no cloud API required)
- **OpenAI Embeddings**: Could be replaced with local embeddings for complete locality

## Benefits of This Setup

### 🔒 **Maximum Privacy**
- All graph data stays local in embedded Kuzu database
- All LLM processing happens locally with Ollama
- Only embeddings use external API (replaceable with local service)

### ⚡ **High Performance**
- Embedded database eliminates network latency
- Local LLM avoids API rate limits
- No internet dependency for core operations

### 💰 **Cost Effective**
- No cloud database hosting costs
- No per-token LLM API charges (except embeddings)
- Run on your own hardware

### 🛠️ **Development Friendly**
- No complex server setup
- Easy to version control database with your code
- Perfect for development and testing

## Prerequisites

### Required
- Go 1.24+
- [Ollama](https://ollama.ai/) installed and running
- OpenAI API key (for embeddings)

### Optional (for fully local setup)
- Local embedding service to replace OpenAI

## Setup Instructions

### 1. Install Ollama

**macOS:**
```bash
brew install ollama
```

**Linux:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

**Windows:**
Download from https://ollama.ai/

### 2. Start Ollama and Pull Model

```bash
# Start Ollama server
ollama serve

# In another terminal, pull a model
ollama pull llama2:7b

# Verify it works
ollama run llama2:7b "Hello world"
```

### 3. Set Environment Variables

```bash
# Required for embeddings (until replaced with local service)
export OPENAI_API_KEY="your-openai-api-key"
```

### 4. Run the Example

```bash
cd examples/kuzu_ollama
go run main.go
```

## Expected Output

```
🚀 Starting go-graphiti example with Kuzu + Ollama
   This example demonstrates a fully local setup:
   - Kuzu: embedded graph database
   - Ollama: local LLM inference
   - OpenAI: embeddings (could be replaced with local)

📊 Setting up Kuzu embedded graph database...
   ✅ Kuzu driver created (embedded database at ./example_graph.db)

🧠 Setting up Ollama local LLM client...
   ✅ Ollama client created (using llama2:7b model)
   💡 Make sure Ollama is running: `ollama serve`
   💡 Make sure model is available: `ollama pull llama2:7b`

🔤 Setting up embedding client...
   ✅ OpenAI embedder created (text-embedding-3-small)
   💡 For fully local setup, replace with local embedding service

🌐 Setting up Graphiti client with local components...
   ✅ Graphiti client created with local Kuzu + Ollama setup

📝 Adding example episodes to the knowledge graph...
   ⚠️  Expected error with stub implementation: KuzuDriver not implemented
   This will work once the Kuzu Go library is available

🔍 Searching the knowledge graph...
   ⚠️  Expected errors with stub implementation
   This will work once the Kuzu Go library is available

💭 Testing Ollama LLM integration...
   Sending query to Ollama...
   ✅ Ollama response received:
     Embedded graph databases like Kuzu offer several advantages over server-based solutions...
     Used 245 tokens

📋 Example Summary:
   ✅ Kuzu driver: Created (stub implementation)
   ✅ Ollama client: Created and tested
   ✅ Graphiti integration: Demonstrated

🎉 Example completed successfully!
```

## Current Status

### What Works Now ✅
- Kuzu driver creation (stub implementation)
- Ollama LLM client integration
- OpenAI embeddings
- Complete API demonstration

### What Will Work Later 🔮
- Actual graph database operations (when Kuzu Go library is available)
- Full knowledge graph storage and retrieval
- Hybrid search with local graph traversal

## Configuration Options

### Different Ollama Models

```go
// Larger model for better quality
llmConfig := llm.Config{
    Model: "llama2:13b",  // or "codellama:7b", "mistral:7b", etc.
    Temperature: &[]float32{0.5}[0],  // Lower for more focused responses
    MaxTokens: &[]int{2000}[0],       // Longer responses
}
```

### Custom Ollama URL

```go
// If Ollama is running on different host/port
ollama, err := llm.NewOllamaClient("http://192.168.1.100:11434", "llama2:7b", llmConfig)
```

### Different Kuzu Database Path

```go
// Custom database location
kuzuDriver, err := driver.NewKuzuDriver("/path/to/my/graph.db")
```

## Troubleshooting

### Ollama Issues

**Problem**: `connection refused`
```bash
# Make sure Ollama is running
ollama serve

# Check if it's responding
curl http://localhost:11434/api/tags
```

**Problem**: `model not found`
```bash
# List available models
ollama list

# Pull the required model
ollama pull llama2:7b
```

**Problem**: Slow responses
- Try smaller model: `llama2:7b` instead of `llama2:13b`
- Reduce `MaxTokens` in config
- Ensure sufficient RAM (8GB+ recommended)

### Kuzu Issues

**Current**: All Kuzu operations return "not implemented" errors - this is expected until the Kuzu Go library is available.

**Future**: Once available, potential issues might include:
- Database file permissions
- Disk space for database files
- CGO compilation requirements

### Memory Usage

This setup can be memory-intensive:
- **Ollama models**: 4-8GB RAM (depending on model size)
- **Embeddings**: Temporary memory for batch processing
- **Kuzu database**: Memory-mapped files

**Recommendations**:
- Start with `llama2:7b` model (smaller)
- Monitor system resources
- Consider using swap if RAM is limited

## Performance Comparison

| Component | Local (This Setup) | Cloud Alternative | Notes |
|-----------|-------------------|-------------------|--------|
| Graph DB | Kuzu (embedded) | Neo4j (server) | Local: faster queries, no network |
| LLM | Ollama (local) | OpenAI API | Local: no rate limits, slower inference |
| Embeddings | OpenAI API | OpenAI API | Could be local in future |
| **Overall** | **Privacy + Control** | **Speed + Convenience** | Trade-offs depend on use case |

## Future Enhancements

### Complete Local Setup
```go
// Replace OpenAI embeddings with local service
localEmbedder := embedder.NewLocalEmbedder("http://localhost:8080", embedderConfig)
```

### Advanced Ollama Configuration
```go
// Custom system prompts for graph-specific tasks
systemPrompt := `You are an AI assistant specialized in analyzing temporal knowledge graphs. 
Focus on relationships between entities and temporal patterns in the data.`

llmConfig := llm.Config{
    Model: "llama2:7b",
    Temperature: &[]float32{0.3}[0],  // More focused for graph analysis
    Stop: []string{"</analysis>", "\n\n"},  // Custom stop sequences
}
```

### Production Considerations
- Database backup strategies for Kuzu files
- Model version management for Ollama
- Resource monitoring and scaling
- Error recovery and fallback mechanisms

## Related Examples

- **[Basic Example](../basic/)**: Neo4j + OpenAI setup
- **[OpenAI Compatible](../openai_compatible/)**: Various local LLM services

## Resources

- **Kuzu Documentation**: https://docs.kuzudb.com/
- **Ollama Models**: https://ollama.ai/library
- **Local Embeddings**: Consider sentence-transformers, BGE, or similar