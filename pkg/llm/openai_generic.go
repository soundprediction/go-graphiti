package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// OpenAIGenericClient implements the Client interface for OpenAI's language models
// This is equivalent to Python's OpenAIGenericClient class
type OpenAIGenericClient struct {
	*BaseOpenAIClient
	client *openai.Client
}

// NewOpenAIGenericClient creates a new OpenAI generic client
func NewOpenAIGenericClient(config *LLMConfig) (*OpenAIGenericClient, error) {
	if config == nil {
		config = NewLLMConfig()
	}

	if config.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}

	baseClient := NewBaseOpenAIClient(config, DefaultReasoning, DefaultVerbosity)

	var client *openai.Client
	if config.BaseURL != "" {
		// Validate and configure custom base URL for OpenAI-compatible services
		if err := validateBaseURL(config.BaseURL); err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}

		// Create OpenAI client configuration with custom base URL
		clientConfig := openai.DefaultConfig(config.APIKey)
		clientConfig.BaseURL = config.BaseURL

		// Handle common base URL patterns
		if !hasAPIPath(config.BaseURL) {
			clientConfig.BaseURL = config.BaseURL + "/v1"
		}

		client = openai.NewClientWithConfig(clientConfig)
	} else {
		// Use default OpenAI client
		client = openai.NewClient(config.APIKey)
	}

	return &OpenAIGenericClient{
		BaseOpenAIClient: baseClient,
		client:           client,
	}, nil
}

// Chat implements the Client interface
func (c *OpenAIGenericClient) Chat(ctx context.Context, messages []Message) (*Response, error) {
	// Use the base client's retry mechanism for regular chat
	responseMap, err := c.GenerateResponseWithRetry(ctx, c.client, messages, nil, 0, ModelSizeMedium)
	if err != nil {
		return nil, err
	}

	// Convert map response to Response struct
	response := &Response{}

	// Try to extract content from various possible keys
	if content, ok := responseMap["content"].(string); ok {
		response.Content = content
	} else if text, ok := responseMap["text"].(string); ok {
		response.Content = text
	} else {
		// If no standard content field, serialize the entire response
		contentBytes, err := json.Marshal(responseMap)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize response: %w", err)
		}
		response.Content = string(contentBytes)
	}

	return response, nil
}

// ChatWithStructuredOutput implements the Client interface
func (c *OpenAIGenericClient) ChatWithStructuredOutput(ctx context.Context, messages []Message, schema interface{}) (json.RawMessage, error) {
	responseMap, err := c.GenerateResponseWithRetry(ctx, c.client, messages, schema, 0, ModelSizeMedium)
	if err != nil {
		return nil, err
	}

	// Convert response map to JSON
	responseBytes, err := json.Marshal(responseMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal structured response: %w", err)
	}

	return json.RawMessage(responseBytes), nil
}

// generateResponseWithEnhancedRetry implements the Python-style retry logic with error feedback
func (c *OpenAIGenericClient) generateResponseWithEnhancedRetry(
	ctx context.Context,
	messages []Message,
	responseModel interface{},
	maxTokens int,
	modelSize ModelSize,
) (map[string]interface{}, error) {
	var lastError error
	retryCount := 0
	workingMessages := make([]Message, len(messages))
	copy(workingMessages, messages)

	// Prepare messages with schema if needed
	preparedMessages, err := c.PrepareMessages(workingMessages, responseModel)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare messages: %w", err)
	}

	for retryCount <= c.maxRetries {
		// Convert to OpenAI format
		openaiMessages := c.ConvertMessagesToOpenAIFormat(preparedMessages)
		model := c.GetModelForSize(modelSize)

		// Build request
		req := c.BuildChatRequest(openaiMessages, model, maxTokens)

		// Force JSON response format
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}

		// Make the API call
		resp, err := c.client.CreateChatCompletion(ctx, req)
		if err != nil {
			lastError = err

			// Check for rate limit errors (don't retry)
			if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "rate_limit") {
				return nil, NewRateLimitError(err.Error())
			}

			// Check for OpenAI-specific errors that shouldn't be retried
			if strings.Contains(err.Error(), "timeout") ||
			   strings.Contains(err.Error(), "connection") ||
			   strings.Contains(err.Error(), "internal server error") {
				return nil, fmt.Errorf("openai API error: %w", err)
			}

			// For other errors, don't retry if we've hit max retries
			if retryCount >= c.maxRetries {
				log.Printf("Max retries (%d) exceeded. Last error: %v", c.maxRetries, err)
				return nil, fmt.Errorf("max retries exceeded: %w", err)
			}

			retryCount++

			// Add error context to messages for next retry
			errorContext := fmt.Sprintf(
				"The previous response attempt was invalid. "+
					"Error type: %T. "+
					"Error details: %s. "+
					"Please try again with a valid response, ensuring the output matches "+
					"the expected format and constraints.",
				err, err.Error(),
			)

			errorMessage := NewUserMessage(errorContext)
			preparedMessages = append(preparedMessages, errorMessage)

			log.Printf("Retrying after application error (attempt %d/%d): %v", retryCount, c.maxRetries, err)
			continue
		}

		// Parse response
		result, err := c.HandleJSONResponse(resp)
		if err != nil {
			lastError = err

			// Don't retry if we've hit max retries
			if retryCount >= c.maxRetries {
				log.Printf("Max retries (%d) exceeded. Last error: %v", c.maxRetries, err)
				return nil, fmt.Errorf("max retries exceeded: %w", err)
			}

			retryCount++

			// Add parsing error context to messages
			errorContext := fmt.Sprintf(
				"The previous response could not be parsed as valid JSON. "+
					"Error: %s. "+
					"Please ensure your response is valid JSON that matches the expected format.",
				err.Error(),
			)

			errorMessage := NewUserMessage(errorContext)
			preparedMessages = append(preparedMessages, errorMessage)

			log.Printf("Retrying after parsing error (attempt %d/%d): %v", retryCount, c.maxRetries, err)
			continue
		}

		// Success!
		return result, nil
	}

	// If we get here, we've exhausted retries
	if lastError != nil {
		return nil, fmt.Errorf("max retries exceeded with last error: %w", lastError)
	}
	return nil, fmt.Errorf("max retries exceeded with no specific error")
}

// Close implements the Client interface
func (c *OpenAIGenericClient) Close() error {
	// OpenAI client doesn't require explicit cleanup
	return nil
}

// GetClient returns the underlying OpenAI client for advanced usage
func (c *OpenAIGenericClient) GetClient() *openai.Client {
	return c.client
}

// GetConfig returns the client configuration
func (c *OpenAIGenericClient) GetConfig() *LLMConfig {
	return c.config
}