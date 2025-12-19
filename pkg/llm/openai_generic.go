package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/soundprediction/go-predicato/pkg/types"
)

// OpenAIGenericClient implements the Client interface for OpenAI's language models
// This is equivalent to Python's OpenAIGenericClient class
type OpenAIGenericClient struct {
	*BaseOpenAIClient
	client *openai.Client
}

// NewOpenAIGenericClient creates a new OpenAI generic client
// Supports both signatures:
//   - NewOpenAIGenericClient(apiKey string, config Config) - legacy
//   - NewOpenAIGenericClient(config *LLMConfig) - new
func NewOpenAIGenericClient(args ...interface{}) (*OpenAIGenericClient, error) {
	var llmConfig *LLMConfig

	// Handle different signatures
	switch len(args) {
	case 1:
		// New signature: NewOpenAIGenericClient(config *LLMConfig)
		var ok bool
		llmConfig, ok = args[0].(*LLMConfig)
		if !ok {
			return nil, fmt.Errorf("expected *LLMConfig, got %T", args[0])
		}
	case 2:
		// Legacy signature: NewOpenAIGenericClient(apiKey string, config Config)
		apiKey, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("expected string for apiKey, got %T", args[0])
		}
		config, ok := args[1].(Config)
		if !ok {
			return nil, fmt.Errorf("expected Config, got %T", args[1])
		}

		// Convert legacy Config to LLMConfig
		llmConfig = &LLMConfig{
			APIKey:  apiKey,
			Model:   config.Model,
			BaseURL: config.BaseURL,
		}
		if config.Temperature != nil {
			llmConfig.Temperature = *config.Temperature
		}
		if config.MaxTokens != nil {
			llmConfig.MaxTokens = *config.MaxTokens
		}
		if config.TopP != nil {
			llmConfig.TopP = *config.TopP
		}
		if config.TopK != nil {
			llmConfig.TopK = *config.TopK
		}
		if config.MinP != nil {
			llmConfig.MinP = *config.MinP
		}
		// Note: Stop sequences are not supported in LLMConfig yet
	default:
		return nil, fmt.Errorf("invalid number of arguments: expected 1 or 2, got %d", len(args))
	}

	if llmConfig == nil {
		llmConfig = NewLLMConfig()
	}

	baseClient := NewBaseOpenAIClient(llmConfig, DefaultReasoning, DefaultVerbosity)
	baseClient.maxRetries = llmConfig.MaxRetries
	var client *openai.Client
	if llmConfig.BaseURL != "" {
		// Validate and configure custom base URL for OpenAI-compatible services
		if err := validateBaseURL(llmConfig.BaseURL); err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}

		// Create OpenAI client configuration with custom base URL
		clientConfig := openai.DefaultConfig(llmConfig.APIKey)
		clientConfig.BaseURL = llmConfig.BaseURL

		// Handle common base URL patterns
		if !hasAPIPath(llmConfig.BaseURL) {
			clientConfig.BaseURL = llmConfig.BaseURL + "/v1"
		}

		client = openai.NewClientWithConfig(clientConfig)
	} else {
		// Use default OpenAI client
		client = openai.NewClient(llmConfig.APIKey)
	}

	return &OpenAIGenericClient{
		BaseOpenAIClient: baseClient,
		client:           client,
	}, nil
}

// Chat implements the Client interface
func (c *OpenAIGenericClient) Chat(ctx context.Context, messages []types.Message) (*types.Response, error) {
	// Debug logging for prompts if DEBUG_LLM_PROMPTS environment variable is set
	// if os.Getenv("DEBUG_LLM_PROMPTS") == "true" {
	// 	fmt.Printf("\n========== LLM CHAT REQUEST ==========\n")
	// 	for i, msg := range messages {
	// 		fmt.Printf("Message %d [%s]:\n%s\n\n", i+1, msg.Role, msg.Content)
	// 	}
	// 	fmt.Printf("======================================\n\n")
	// }

	// Use the base client's retry mechanism for regular chat
	response, err := c.GenerateResponseWithRetry(ctx, c.client, messages, nil, 0, ModelSizeMedium)
	if err != nil {
		return nil, err
	}

	// Try to extract content from various possible keys if the response is JSON
	// This maintains compatibility with the previous map-based approach
	var responseMap map[string]interface{}
	if err := json.Unmarshal([]byte(response.Content), &responseMap); err == nil {
		if content, ok := responseMap["content"].(string); ok {
			response.Content = content
		} else if text, ok := responseMap["text"].(string); ok {
			response.Content = text
		}
	}

	return response, nil
}

// ChatWithStructuredOutput implements the Client interface
func (c *OpenAIGenericClient) ChatWithStructuredOutput(ctx context.Context, messages []types.Message, schema interface{}) (*types.Response, error) {
	// Use continuation-based JSON generation for more robust handling
	jsonStr, err := GenerateJSONResponseWithContinuationMessages(ctx, c, messages, schema, c.maxRetries)
	if err != nil {
		return nil, err
	}

	// OpenAIGenericClient currently returns raw string from the helper
	// We extract usage from the underlying calls inside the helper if possible,
	// but the helper signature only returns string.
	// For now, we wrap the string result.
	// TODO: Refactor GenerateJSONResponseWithContinuationMessages to return usage stats
	return &types.Response{
		Content: jsonStr,
	}, nil
}

// generateResponseWithEnhancedRetry implements the Python-style retry logic with error feedback
func (c *OpenAIGenericClient) generateResponseWithEnhancedRetry(
	ctx context.Context,
	messages []types.Message,
	responseModel interface{},
	maxTokens int,
	modelSize ModelSize,
) (map[string]interface{}, error) {
	var lastError error
	retryCount := 0
	workingMessages := make([]types.Message, len(messages))
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
