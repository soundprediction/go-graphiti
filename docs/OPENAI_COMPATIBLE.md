# OpenAI-Compatible Services

go-graphiti includes support for any service that implements the OpenAI API specification. This allows you to use local models, alternative providers, or self-hosted services while maintaining the same interface.

## Supported Services

### Local Inference Servers

- **[Ollama](https://ollama.ai/)**: Easy local LLM serving
- **[LocalAI](https://localai.io/)**: Self-hosted OpenAI alternative
- **[vLLM](https://github.com/vllm-project/vllm)**: High-throughput LLM serving
- **[Text Generation Inference](https://github.com/huggingface/text-generation-inference)**: Hugging Face's serving solution
- **[FastChat](https://github.com/lm-sys/FastChat)**: Multi-model serving platform

### Cloud Alternatives

- **[Together AI](https://www.together.ai/)**: Cloud-hosted open models
- **[Anyscale](https://www.anyscale.com/)**: Serverless LLM inference
- **[Replicate](https://replicate.com/)**: Cloud API for open models
- **[Hugging Face Inference Endpoints](https://huggingface.co/inference-endpoints)**: Dedicated model endpoints

## Basic Usage

### Generic OpenAI-Compatible Client

```go
import "github.com/getzep/go-graphiti/pkg/llm"

// Create a client for any OpenAI-compatible service
client, err := llm.NewOpenAICompatibleClient(
    "http://your-service.com:8080",  // Base URL
    "your-api-key",                  // API key (use "" if not required)
    "your-model-name",               // Model identifier
    llm.Config{
        Temperature: &[]float32{0.7}[0],
        MaxTokens:   &[]int{1000}[0],
    },
)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Use the client
messages := []llm.Message{
    llm.NewUserMessage("Hello, how are you?"),
}

response, err := client.Chat(context.Background(), messages)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### Convenience Functions

The library provides convenience functions for popular services:

#### Ollama

```go
// Ollama (local inference)
client, err := llm.NewOllamaClient(
    "http://localhost:11434",  // Default Ollama URL
    "llama2:7b",               // Model name
    llm.Config{
        Temperature: &[]float32{0.7}[0],
    },
)

// Or use default URL
client, err := llm.NewOllamaClient("", "llama2:7b", llm.Config{})
```

#### LocalAI

```go
// LocalAI (self-hosted)
client, err := llm.NewLocalAIClient(
    "http://localhost:8080",   // Default LocalAI URL
    "gpt-3.5-turbo",          // Model name configured in LocalAI
    llm.Config{},
)
```

#### vLLM

```go
// vLLM (high-performance serving)
client, err := llm.NewVLLMClient(
    "http://vllm-server:8000",        // vLLM server URL
    "microsoft/DialoGPT-medium",      // Model name
    llm.Config{
        MaxTokens: &[]int{500}[0],
    },
)
```

#### Text Generation Inference

```go
// Hugging Face TGI
client, err := llm.NewTextGenerationInferenceClient(
    "http://tgi-server:3000",    // TGI server URL
    "bigscience/bloom",          // Model name
    llm.Config{},
)
```

## Service-Specific Setup Guides

### Ollama Setup

1. **Install Ollama:**
   ```bash
   # macOS
   brew install ollama
   
   # Linux
   curl -fsSL https://ollama.ai/install.sh | sh
   ```

2. **Start Ollama server:**
   ```bash
   ollama serve
   ```

3. **Pull a model:**
   ```bash
   ollama pull llama2:7b
   # or
   ollama pull codellama:7b
   ollama pull mistral:7b
   ```

4. **Use with go-graphiti:**
   ```go
   client, err := llm.NewOllamaClient("", "llama2:7b", llm.Config{})
   ```

### LocalAI Setup

1. **Run with Docker:**
   ```bash
   docker run -p 8080:8080 --name localai -ti localai/localai:latest
   ```

2. **Configure models** (create `models.yaml`):
   ```yaml
   - name: gpt-3.5-turbo
     backend: llama-cpp
     parameters:
       model: /models/ggml-model.bin
   ```

3. **Use with go-graphiti:**
   ```go
   client, err := llm.NewLocalAIClient("", "gpt-3.5-turbo", llm.Config{})
   ```

### vLLM Setup

1. **Install vLLM:**
   ```bash
   pip install vllm
   ```

2. **Start server:**
   ```bash
   python -m vllm.entrypoints.openai.api_server \
     --model microsoft/DialoGPT-medium \
     --port 8000
   ```

3. **Use with go-graphiti:**
   ```go
   client, err := llm.NewVLLMClient("http://localhost:8000", "microsoft/DialoGPT-medium", llm.Config{})
   ```

## Integration with Graphiti

You can use any OpenAI-compatible service as the LLM component in a full Graphiti setup:

```go
// Create your preferred LLM client
llmClient, err := llm.NewOllamaClient("", "llama2:7b", llm.Config{
    Temperature: &[]float32{0.7}[0],
})
if err != nil {
    log.Fatal(err)
}
defer llmClient.Close()

// Create other required components
neo4jDriver, err := driver.NewNeo4jDriver("bolt://localhost:7687", "neo4j", "password", "neo4j")
if err != nil {
    log.Fatal(err)
}
defer neo4jDriver.Close(context.Background())

embedderClient := embedder.NewOpenAIEmbedder("your-openai-key", embedder.Config{
    Model: "text-embedding-3-small",
})
defer embedderClient.Close()

// Create Graphiti client with local LLM
config := &graphiti.Config{
    GroupID: "local-llm-demo",
}

graphitiClient := graphiti.NewClient(neo4jDriver, llmClient, embedderClient, config)
defer graphitiClient.Close(context.Background())

// Use normally
episodes := []types.Episode{
    {
        ID:      "episode-1",
        Content: "Your content here...",
        // ... other fields
    },
}

err = graphitiClient.Add(context.Background(), episodes)
if err != nil {
    log.Fatal(err)
}
```

## Configuration Options

All OpenAI-compatible clients support the standard LLM configuration:

```go
config := llm.Config{
    Model:            "your-model-name",           // Model identifier
    Temperature:      &[]float32{0.7}[0],         // Creativity (0.0-1.0)
    MaxTokens:        &[]int{1000}[0],            // Response length limit
    TopP:             &[]float32{0.9}[0],         // Nucleus sampling
    Stop:             []string{"</s>", "\n\n"},   // Stop sequences
}
```

## Structured Output Support

Some OpenAI-compatible services support structured output (JSON mode):

```go
messages := []llm.Message{
    llm.NewSystemMessage("Respond with valid JSON only."),
    llm.NewUserMessage("Describe a person with name and age."),
}

schema := map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "name": map[string]string{"type": "string"},
        "age":  map[string]string{"type": "integer"},
    },
}

response, err := client.ChatWithStructuredOutput(context.Background(), messages, schema)
if err != nil {
    // Fallback to regular chat if structured output isn't supported
    regularResponse, err := client.Chat(context.Background(), messages)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(regularResponse.Content)
} else {
    fmt.Println(string(response)) // JSON response
}
```

## Error Handling

OpenAI-compatible services may have different error responses. The client handles common cases:

```go
response, err := client.Chat(ctx, messages)
if err != nil {
    if strings.Contains(err.Error(), "connection refused") {
        log.Println("Service is not running or unreachable")
    } else if strings.Contains(err.Error(), "unauthorized") {
        log.Println("Invalid API key or authentication failed")
    } else if strings.Contains(err.Error(), "model not found") {
        log.Println("Model not available on this service")
    } else {
        log.Printf("Other error: %v", err)
    }
    return
}
```

## Performance Considerations

### Local vs Cloud Services

**Local Services (Ollama, LocalAI):**
- ✅ No API costs
- ✅ Data privacy
- ✅ No rate limits
- ❌ Slower inference
- ❌ Limited model selection
- ❌ Hardware requirements

**Cloud Services (OpenAI, Together AI):**
- ✅ Fast inference
- ✅ Latest models
- ✅ No hardware requirements
- ❌ API costs
- ❌ Rate limits
- ❌ Data leaves your infrastructure

### Model Selection Guidelines

**For Development/Testing:**
- Ollama with `llama2:7b` or `codellama:7b`
- Fast startup, good for iteration

**For Production (Local):**
- vLLM with larger models (`llama2:13b`, `codellama:13b`)
- Better performance and quality

**For Production (Cloud):**
- OpenAI `gpt-4o-mini` for cost-effectiveness
- OpenAI `gpt-4o` for best quality

## Troubleshooting

### Common Issues

1. **Connection Refused**
   ```
   Error: connection refused
   ```
   - Check if the service is running
   - Verify the URL and port
   - Test with `curl http://localhost:11434/api/tags` (for Ollama)

2. **Model Not Found**
   ```
   Error: model not found
   ```
   - Verify the model name is correct
   - For Ollama: `ollama list` to see available models
   - For LocalAI: check your `models.yaml` configuration

3. **Authentication Failed**
   ```
   Error: unauthorized
   ```
   - Check API key is correct
   - Some services don't require keys (use `""` or `"dummy"`)

4. **Slow Responses**
   - Use smaller models for faster inference
   - Consider GPU acceleration for local services
   - Adjust `MaxTokens` to limit response length

### Service-Specific Debugging

**Ollama:**
```bash
# Check available models
ollama list

# Test model directly
ollama run llama2:7b "Hello world"

# Check server status
curl http://localhost:11434/api/tags
```

**LocalAI:**
```bash
# Check available models
curl http://localhost:8080/v1/models

# Test completion
curl http://localhost:8080/v1/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-3.5-turbo", "prompt": "Hello", "max_tokens": 10}'
```

## Best Practices

1. **Use appropriate timeouts:**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
   defer cancel()
   ```

2. **Handle service unavailability gracefully:**
   ```go
   response, err := client.Chat(ctx, messages)
   if err != nil {
       // Log error and maybe fallback to another service
       log.Printf("LLM service unavailable: %v", err)
       return handleFallback(messages)
   }
   ```

3. **Monitor token usage:**
   ```go
   if response.TokensUsed != nil {
       log.Printf("Used %d tokens", response.TokensUsed.TotalTokens)
   }
   ```

4. **Use connection pooling for high-throughput scenarios:**
   - The underlying HTTP client handles connection pooling automatically
   - Consider service-specific optimizations (e.g., vLLM batching)

## Examples

See [`examples/openai_compatible/`](../examples/openai_compatible/) for complete working examples with different services.