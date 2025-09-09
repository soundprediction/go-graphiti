package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAIClient implements the Client interface for OpenAI's language models.
type OpenAIClient struct {
	client *openai.Client
	config Config
}

// NewOpenAIClient creates a new OpenAI client.
func NewOpenAIClient(apiKey string, config Config) *OpenAIClient {
	client := openai.NewClient(apiKey)
	
	if config.Model == "" {
		config.Model = openai.GPT4o
	}
	
	return &OpenAIClient{
		client: client,
		config: config,
	}
}

// Chat sends a chat completion request to OpenAI.
func (c *OpenAIClient) Chat(ctx context.Context, messages []Message) (*Response, error) {
	req := c.buildChatRequest(messages, false, nil)
	
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion failed: %w", err)
	}
	
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from openai")
	}
	
	choice := resp.Choices[0]
	response := &Response{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
		TokensUsed: &TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
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
	
	if structuredOutput && schema != nil {
		// Convert schema to JSON Schema format
		schemaBytes, _ := json.Marshal(schema)
		var jsonSchema map[string]interface{}
		json.Unmarshal(schemaBytes, &jsonSchema)
		
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}
	
	return req
}