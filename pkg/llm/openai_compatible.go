package llm

// This file provides convenience functions for common OpenAI-compatible services.
// These functions use the enhanced OpenAI client with BaseURL configuration.

// NewOpenAICompatibleClient creates a client for any OpenAI-compatible service.
// This allows you to use local models, alternative providers, or self-hosted services
// that implement the OpenAI API specification.
//
// Parameters:
//   - baseURL: The base URL of the OpenAI-compatible service (e.g., "http://localhost:11434" for Ollama)
//   - apiKey: API key for authentication (use "" if not required)
//   - model: Model name to use (e.g., "llama2", "codellama", "mistral")
//   - config: Additional configuration options
//
// Example usage:
//
//	// Ollama local instance
//	client, err := llm.NewOpenAICompatibleClient(
//		"http://localhost:11434",
//		"",  // Ollama doesn't require API key
//		"llama2:7b",
//		llm.Config{Temperature: &[]float32{0.7}[0]},
//	)
//
//	// LocalAI instance
//	client, err := llm.NewOpenAICompatibleClient(
//		"http://localhost:8080",
//		"your-api-key",
//		"gpt-3.5-turbo",  // LocalAI model name
//		llm.Config{},
//	)
//
//	// vLLM server
//	client, err := llm.NewOpenAICompatibleClient(
//		"http://vllm-server:8000",
//		"",
//		"microsoft/DialoGPT-medium",
//		llm.Config{MaxTokens: &[]int{1000}[0]},
//	)
func NewOpenAICompatibleClient(baseURL, apiKey, model string, config Config) (*OpenAIClient, error) {
	config.BaseURL = baseURL
	config.Model = model
	return NewOpenAIClient(apiKey, config)
}

// NewOllamaClient creates a client for Ollama local inference.
// Ollama runs locally and doesn't require authentication.
//
// Example:
//
//	client, err := llm.NewOllamaClient("http://localhost:11434", "llama2:7b", llm.Config{
//		Temperature: &[]float32{0.7}[0],
//	})
func NewOllamaClient(baseURL, model string, config Config) (*OpenAIClient, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	config.BaseURL = baseURL
	config.Model = model
	return NewOpenAIClient("", config)
}

// NewLocalAIClient creates a client for LocalAI.
// LocalAI is a self-hosted OpenAI alternative.
//
// Example:
//
//	client, err := llm.NewLocalAIClient("http://localhost:8080", "gpt-3.5-turbo", llm.Config{})
func NewLocalAIClient(baseURL, model string, config Config) (*OpenAIClient, error) {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	config.BaseURL = baseURL
	config.Model = model
	return NewOpenAIClient("", config)
}

// NewVLLMClient creates a client for vLLM server.
// vLLM is a high-throughput serving engine for LLMs.
//
// Example:
//
//	client, err := llm.NewVLLMClient("http://vllm-server:8000", "microsoft/DialoGPT-medium", llm.Config{})
func NewVLLMClient(baseURL, model string, config Config) (*OpenAIClient, error) {
	config.BaseURL = baseURL
	config.Model = model
	return NewOpenAIClient("", config)
}

// NewTextGenerationInferenceClient creates a client for Hugging Face Text Generation Inference.
// TGI is Hugging Face's solution for deploying large language models.
//
// Example:
//
//	client, err := llm.NewTextGenerationInferenceClient("http://tgi-server:3000", "bigscience/bloom", llm.Config{})
func NewTextGenerationInferenceClient(baseURL, model string, config Config) (*OpenAIClient, error) {
	config.BaseURL = baseURL
	config.Model = model
	return NewOpenAIClient("", config)
}