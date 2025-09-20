package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Constants matching Python defaults
const (
	DefaultModel           = "gpt-4o-mini"
	DefaultSmallModel      = "gpt-4o-mini"
	DefaultReasoning       = "minimal"
	DefaultVerbosity       = "low"
	MaxRetries             = 2
	MultilingualExtraction = "\n\nAny extracted information should be returned in the same language as it was written in."
)

// BaseOpenAIClient provides common functionality for OpenAI-compatible clients
// This is equivalent to Python's BaseOpenAIClient class
type BaseOpenAIClient struct {
	config     *LLMConfig
	model      string
	smallModel string
	reasoning  string
	verbosity  string
	maxRetries int
}

// NewBaseOpenAIClient creates a new base OpenAI client
func NewBaseOpenAIClient(config *LLMConfig, reasoning, verbosity string) *BaseOpenAIClient {
	if config == nil {
		config = NewLLMConfig()
	}

	model := config.Model
	if model == "" {
		model = DefaultModel
	}

	smallModel := config.SmallModel
	if smallModel == "" {
		smallModel = DefaultSmallModel
	}

	return &BaseOpenAIClient{
		config:     config,
		model:      model,
		smallModel: smallModel,
		reasoning:  reasoning,
		verbosity:  verbosity,
		maxRetries: MaxRetries,
	}
}

// ConvertMessagesToOpenAIFormat converts internal Message format to OpenAI format
func (b *BaseOpenAIClient) ConvertMessagesToOpenAIFormat(messages []Message) []openai.ChatCompletionMessage {
	openaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages))

	for _, m := range messages {
		content := b.cleanInput(m.Content)

		switch m.Role {
		case RoleUser:
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: content,
			})
		case RoleSystem:
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: content,
			})
		case RoleAssistant:
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: content,
			})
		}
	}

	return openaiMessages
}

// GetModelForSize returns the appropriate model based on the requested size
func (b *BaseOpenAIClient) GetModelForSize(modelSize ModelSize) string {
	if modelSize == ModelSizeSmall {
		return b.smallModel
	}
	return b.model
}

// HandleJSONResponse parses a JSON response from the LLM
func (b *BaseOpenAIClient) HandleJSONResponse(response openai.ChatCompletionResponse) (map[string]interface{}, error) {
	if len(response.Choices) == 0 {
		return nil, NewEmptyResponseError("no choices returned from API")
	}

	content := response.Choices[0].Message.Content
	if content == "" {
		content = "{}"
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return result, nil
}

// PrepareMessages prepares messages for sending to the LLM
func (b *BaseOpenAIClient) PrepareMessages(messages []Message, responseModel interface{}) ([]Message, error) {
	// Make a copy to avoid modifying the original
	preparedMessages := make([]Message, len(messages))
	copy(preparedMessages, messages)

	// Add structured output instructions if response model is provided
	if responseModel != nil {
		schemaBytes, err := json.Marshal(responseModel)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize response model: %w", err)
		}

		lastIdx := len(preparedMessages) - 1
		preparedMessages[lastIdx].Content += fmt.Sprintf(
			"\n\nRespond with a JSON object in the following format:\n\n%s",
			string(schemaBytes),
		)
	}

	// Add multilingual extraction instructions to the first message
	if len(preparedMessages) > 0 {
		preparedMessages[0].Content += MultilingualExtraction
	}

	return preparedMessages, nil
}

// cleanInput cleans input string of invalid unicode and control characters
func (b *BaseOpenAIClient) cleanInput(input string) string {
	// Remove zero-width characters and other invisible unicode
	zeroWidthChars := []string{"\u200b", "\u200c", "\u200d", "\ufeff", "\u2060"}
	cleaned := input

	for _, char := range zeroWidthChars {
		cleaned = strings.ReplaceAll(cleaned, char, "")
	}

	// Remove control characters except newlines, returns, and tabs
	var builder strings.Builder
	for _, r := range cleaned {
		if r >= 32 || r == '\n' || r == '\r' || r == '\t' {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// BuildChatRequest builds a chat completion request with common parameters
func (b *BaseOpenAIClient) BuildChatRequest(messages []openai.ChatCompletionMessage, model string, maxTokens int) openai.ChatCompletionRequest {
	req := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		Temperature: b.config.Temperature,
	}

	if maxTokens > 0 {
		req.MaxTokens = maxTokens
	} else if b.config.MaxTokens > 0 {
		req.MaxTokens = b.config.MaxTokens
	}

	return req
}

// GenerateResponseWithRetry implements retry logic similar to Python implementation
func (b *BaseOpenAIClient) GenerateResponseWithRetry(
	ctx context.Context,
	client *openai.Client,
	messages []Message,
	responseModel interface{},
	maxTokens int,
	modelSize ModelSize,
) (map[string]interface{}, error) {
	var lastError error
	model := b.GetModelForSize(modelSize)

	preparedMessages, err := b.PrepareMessages(messages, responseModel)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare messages: %w", err)
	}

	openaiMessages := b.ConvertMessagesToOpenAIFormat(preparedMessages)

	for attempt := 0; attempt <= b.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			backoff := time.Duration(attempt*attempt) * time.Second
			log.Printf("Retrying LLM request after %v (attempt %d/%d)", backoff, attempt+1, b.maxRetries+1)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req := b.BuildChatRequest(openaiMessages, model, maxTokens)

		resp, err := client.CreateChatCompletion(ctx, req)
		if err != nil {
			lastError = err

			// Check if this is a rate limit error
			if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "rate_limit") {
				if attempt == b.maxRetries {
					return nil, NewRateLimitError(err.Error())
				}
				continue
			}

			// Check for retriable errors
			if isRetriableError(err) && attempt < b.maxRetries {
				continue
			}

			// Non-retriable error, return immediately
			return nil, fmt.Errorf("openai completion failed: %w", err)
		}

		// Success, parse and return response
		return b.HandleJSONResponse(resp)
	}

	// All retries exhausted
	return nil, fmt.Errorf("all retries exhausted, last error: %w", lastError)
}

// isRetriableError determines if an error should trigger a retry
func isRetriableError(err error) bool {
	errStr := strings.ToLower(err.Error())
	retriableErrors := []string{
		"timeout",
		"connection",
		"internal server error",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
	}

	for _, retriable := range retriableErrors {
		if strings.Contains(errStr, retriable) {
			return true
		}
	}

	return false
}

// Note: validateBaseURL and hasAPIPath are defined in openai.go to avoid duplication