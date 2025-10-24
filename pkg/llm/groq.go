package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GroqClient implements the Client interface for Groq models.
// Groq uses OpenAI-compatible API format.
type GroqClient struct {
	config     *LLMConfig
	httpClient *http.Client
}

// NewGroqClient creates a new Groq client.
func NewGroqClient(config *LLMConfig) *GroqClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.groq.com"
	}

	return &GroqClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// groqRequest represents the request structure for Groq API (OpenAI-compatible).
type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

// groqMessage represents a message in Groq format.
type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// groqResponse represents the response from Groq API.
type groqResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []groqChoice `json:"choices"`
	Error   *groqError   `json:"error,omitempty"`
}

// groqChoice represents a choice in the response.
type groqChoice struct {
	Index        int         `json:"index"`
	Message      groqMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// groqError represents an error response.
type groqError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Chat implements the Client interface for Groq.
func (g *GroqClient) Chat(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages provided")
	}

	// Convert messages to Groq format
	groqMessages := make([]groqMessage, 0, len(messages))
	for _, msg := range messages {
		groqMessages = append(groqMessages, groqMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	req := groqRequest{
		Model:       g.config.Model,
		Messages:    groqMessages,
		MaxTokens:   g.config.MaxTokens,
		Temperature: float64(g.config.Temperature),
		Stream:      false,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.config.BaseURL+"/openai/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.config.APIKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var groqResp groqResponse
	if err := json.Unmarshal(body, &groqResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if groqResp.Error != nil {
		return "", fmt.Errorf("API error: %s", groqResp.Error.Message)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return groqResp.Choices[0].Message.Content, nil
}

// ChatWithStructuredOutput implements structured output for Groq.
// Since Groq uses OpenAI-compatible format, we use the same approach as OpenAI.
func (g *GroqClient) ChatWithStructuredOutput(ctx context.Context, messages []Message, schema interface{}) (string, error) {
	// For now, use prompt engineering like other providers
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	modifiedMessages := append(messages, Message{
		Role:    RoleUser,
		Content: fmt.Sprintf("Please respond with valid JSON that matches this schema: %s", string(schemaBytes)),
	})

	return g.Chat(ctx, modifiedMessages)
}
