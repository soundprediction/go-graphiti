package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/sashabaranov/go-openai"
)

// OpenAICompatibleClient implements the Client interface for any OpenAI-compatible API.
// This includes services like Ollama, LocalAI, vLLM, Text Generation Inference, and others.
type OpenAICompatibleClient struct {
	client *openai.Client
	config Config
}

// NewOpenAICompatibleClient creates a new OpenAI-compatible client for any service.
// This allows you to use local models, alternative providers, or self-hosted services
// that implement the OpenAI API specification.
//
// Parameters:
//   - baseURL: The base URL of the OpenAI-compatible service (e.g., "http://localhost:11434" for Ollama)
//   - apiKey: API key for authentication (use "dummy" or empty string if not required)
//   - model: Model name to use (e.g., "llama2", "codellama", "mistral")
//   - config: Additional configuration options
//
// Example usage:
//
//	// Ollama local instance
//	client := llm.NewOpenAICompatibleClient(
//		"http://localhost:11434",
//		"",  // Ollama doesn't require API key
//		"llama2:7b",
//		llm.Config{Temperature: &[]float32{0.7}[0]},
//	)
//
//	// LocalAI instance
//	client := llm.NewOpenAICompatibleClient(
//		"http://localhost:8080",
//		"your-api-key",
//		"gpt-3.5-turbo",  // LocalAI model name
//		llm.Config{},
//	)
//
//	// vLLM server
//	client := llm.NewOpenAICompatibleClient(
//		"http://vllm-server:8000",
//		"",
//		"microsoft/DialoGPT-medium",
//		llm.Config{MaxTokens: &[]int{1000}[0]},
//	)
func NewOpenAICompatibleClient(baseURL, apiKey, model string, config Config) (*OpenAICompatibleClient, error) {
	// Validate base URL
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	
	// Validate URL format
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid baseURL format: %w", err)
	}
	
	// Ensure URL has a valid scheme
	if parsedURL.Scheme == "" {
		return nil, fmt.Errorf("baseURL must include scheme (http:// or https://)")
	}
	
	// Ensure scheme is http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("baseURL must use http:// or https:// scheme")
	}
	
	// Set default model if not provided
	if model == "" {
		model = "gpt-3.5-turbo" // Default fallback
	}
	config.Model = model
	
	// Use dummy API key if none provided (some services don't require authentication)
	if apiKey == "" {
		apiKey = "dummy-key"
	}
	
	// Create OpenAI client configuration with custom base URL
	clientConfig := openai.DefaultConfig(apiKey)
	clientConfig.BaseURL = baseURL
	
	// Handle common base URL patterns
	// Many services expect "/v1" to be appended to the base URL
	if !hasAPIPath(baseURL) {
		clientConfig.BaseURL = baseURL + "/v1"
	}
	
	client := openai.NewClientWithConfig(clientConfig)
	
	return &OpenAICompatibleClient{
		client: client,
		config: config,
	}, nil
}

// Chat sends a chat completion request to the OpenAI-compatible service.
func (c *OpenAICompatibleClient) Chat(ctx context.Context, messages []Message) (*Response, error) {
	req := c.buildChatRequest(messages, false, nil)
	
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai-compatible chat completion failed: %w", err)
	}
	
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from openai-compatible service")
	}
	
	choice := resp.Choices[0]
	response := &Response{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
	}
	
	// Include token usage if available
	if resp.Usage.TotalTokens > 0 {
		response.TokensUsed = &TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	
	return response, nil
}

// ChatWithStructuredOutput sends a chat completion request with structured output.
// Note: Not all OpenAI-compatible services support structured output.
// This method will attempt to use JSON mode if available, otherwise it falls back to regular chat.
func (c *OpenAICompatibleClient) ChatWithStructuredOutput(ctx context.Context, messages []Message, schema any) (json.RawMessage, error) {
	req := c.buildChatRequest(messages, true, schema)
	
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai-compatible structured output failed: %w", err)
	}
	
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from openai-compatible service")
	}
	
	choice := resp.Choices[0]
	return json.RawMessage(choice.Message.Content), nil
}

// Close cleans up resources (no-op for OpenAI-compatible client).
func (c *OpenAICompatibleClient) Close() error {
	return nil
}

// buildChatRequest constructs the chat completion request.
func (c *OpenAICompatibleClient) buildChatRequest(messages []Message, structuredOutput bool, schema any) openai.ChatCompletionRequest {
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}
	
	req := openai.ChatCompletionRequest{
		Model:    c.config.Model,
		Messages: openaiMessages,
	}
	
	// Apply configuration parameters
	if c.config.Temperature != nil {
		req.Temperature = *c.config.Temperature
	}
	if c.config.MaxTokens != nil {
		req.MaxTokens = *c.config.MaxTokens
	}
	if c.config.TopP != nil {
		req.TopP = *c.config.TopP
	}
	if len(c.config.Stop) > 0 {
		req.Stop = c.config.Stop
	}
	
	// Handle structured output if requested
	// Note: This may not work with all OpenAI-compatible services
	if structuredOutput {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
		
		// Add instruction for JSON output if not already present
		if len(openaiMessages) > 0 {
			lastMessage := &req.Messages[len(req.Messages)-1]
			if lastMessage.Role == string(RoleUser) {
				lastMessage.Content += "\n\nPlease respond with valid JSON only."
			}
		}
	}
	
	return req
}

// hasAPIPath checks if the base URL already includes an API path component.
func hasAPIPath(baseURL string) bool {
	commonPaths := []string{"/v1", "/api", "/v1/", "/api/"}
	for _, path := range commonPaths {
		if len(baseURL) >= len(path) && baseURL[len(baseURL)-len(path):] == path {
			return true
		}
	}
	return false
}

// Convenience functions for common OpenAI-compatible services

// NewOllamaClient creates a client for Ollama local inference.
// Ollama runs locally and doesn't require authentication.
//
// Example:
//
//	client, err := llm.NewOllamaClient("http://localhost:11434", "llama2:7b", llm.Config{
//		Temperature: &[]float32{0.7}[0],
//	})
func NewOllamaClient(baseURL, model string, config Config) (*OpenAICompatibleClient, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return NewOpenAICompatibleClient(baseURL, "", model, config)
}

// NewLocalAIClient creates a client for LocalAI.
// LocalAI is a self-hosted OpenAI alternative.
//
// Example:
//
//	client, err := llm.NewLocalAIClient("http://localhost:8080", "gpt-3.5-turbo", llm.Config{})
func NewLocalAIClient(baseURL, model string, config Config) (*OpenAICompatibleClient, error) {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return NewOpenAICompatibleClient(baseURL, "", model, config)
}

// NewVLLMClient creates a client for vLLM server.
// vLLM is a high-throughput serving engine for LLMs.
//
// Example:
//
//	client, err := llm.NewVLLMClient("http://vllm-server:8000", "microsoft/DialoGPT-medium", llm.Config{})
func NewVLLMClient(baseURL, model string, config Config) (*OpenAICompatibleClient, error) {
	return NewOpenAICompatibleClient(baseURL, "", model, config)
}

// NewTextGenerationInferenceClient creates a client for Hugging Face Text Generation Inference.
// TGI is Hugging Face's solution for deploying large language models.
//
// Example:
//
//	client, err := llm.NewTextGenerationInferenceClient("http://tgi-server:3000", "bigscience/bloom", llm.Config{})
func NewTextGenerationInferenceClient(baseURL, model string, config Config) (*OpenAICompatibleClient, error) {
	return NewOpenAICompatibleClient(baseURL, "", model, config)
}