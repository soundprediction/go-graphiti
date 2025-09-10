package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/sashabaranov/go-openai"
)

// OpenAIClient implements the Client interface for OpenAI's language models.
type OpenAIClient struct {
	client *openai.Client
	config Config
}

// NewOpenAIClient creates a new OpenAI client.
// Supports OpenAI-compatible services through custom BaseURL configuration.
func NewOpenAIClient(apiKey string, config Config) (*OpenAIClient, error) {
	var client *openai.Client

	if config.BaseURL != "" {
		// Validate and configure custom base URL for OpenAI-compatible services
		if err := validateBaseURL(config.BaseURL); err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}

		// Use dummy API key if none provided (some services don't require authentication)
		if apiKey == "" {
			apiKey = "dummy-key"
		}

		// Create OpenAI client configuration with custom base URL
		clientConfig := openai.DefaultConfig(apiKey)
		clientConfig.BaseURL = config.BaseURL

		// Handle common base URL patterns
		// Many services expect "/v1" to be appended to the base URL
		if !hasAPIPath(config.BaseURL) {
			clientConfig.BaseURL = config.BaseURL + "/v1"
		}

		client = openai.NewClientWithConfig(clientConfig)
	} else {
		// Use default OpenAI client
		client = openai.NewClient(apiKey)
	}

	if config.Model == "" {
		if config.BaseURL != "" {
			config.Model = "gpt-3.5-turbo" // Default fallback for OpenAI-compatible services
		} else {
			config.Model = openai.GPT4o // Default for OpenAI
		}
	}

	return &OpenAIClient{
		client: client,
		config: config,
	}, nil
}

// Chat sends a chat completion request to OpenAI or OpenAI-compatible service.
func (c *OpenAIClient) Chat(ctx context.Context, messages []Message) (*Response, error) {
	req := c.buildChatRequest(messages, false, nil)

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		if c.config.BaseURL != "" {
			return nil, fmt.Errorf("openai-compatible chat completion failed: %w", err)
		}
		return nil, fmt.Errorf("openai chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		if c.config.BaseURL != "" {
			return nil, fmt.Errorf("no choices returned from openai-compatible service")
		}
		return nil, fmt.Errorf("no choices returned from openai")
	}

	choice := resp.Choices[0]
	response := &Response{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
	}

	// Include token usage if available (some OpenAI-compatible services might not provide this)
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
func (c *OpenAIClient) ChatWithStructuredOutput(ctx context.Context, messages []Message, schema any) (json.RawMessage, error) {
	req := c.buildChatRequest(messages, true, schema)

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai structured output failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from openai")
	}

	choice := resp.Choices[0]
	return json.RawMessage(choice.Message.Content), nil
}

// Close cleans up resources (no-op for OpenAI client).
func (c *OpenAIClient) Close() error {
	return nil
}

func (c *OpenAIClient) buildChatRequest(messages []Message, structuredOutput bool, schema any) openai.ChatCompletionRequest {
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

	if structuredOutput {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}

		// Add instruction for JSON output if not already present for OpenAI-compatible services
		if c.config.BaseURL != "" && len(openaiMessages) > 0 {
			lastMessage := &req.Messages[len(req.Messages)-1]
			if lastMessage.Role == string(RoleUser) {
				lastMessage.Content += "\n\nPlease respond with valid JSON only."
			}
		}
	}

	return req
}

// validateBaseURL validates the base URL format.
func validateBaseURL(baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("baseURL cannot be empty")
	}

	// Validate URL format
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid baseURL format: %w", err)
	}

	// Ensure URL has a valid scheme
	if parsedURL.Scheme == "" {
		return fmt.Errorf("baseURL must include scheme (http:// or https://)")
	}

	// Ensure scheme is http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("baseURL must use http:// or https:// scheme")
	}

	return nil
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
