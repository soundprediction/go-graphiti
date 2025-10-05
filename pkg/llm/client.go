package llm

import (
	"context"
	"encoding/json"
)

// Client defines the interface for language model operations.
type Client interface {
	// Chat sends a chat completion request and returns the response.
	Chat(ctx context.Context, messages []Message) (*Response, error)
	
	// ChatWithStructuredOutput sends a chat completion request with structured output.
	ChatWithStructuredOutput(ctx context.Context, messages []Message, schema any) (json.RawMessage, error)
	
	// Close cleans up any resources.
	Close() error
}

// Message represents a chat message.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Role represents the role of a message sender.
type Role string

const (
	// RoleSystem represents a system message.
	RoleSystem Role = "system"
	// RoleUser represents a user message.
	RoleUser Role = "user"
	// RoleAssistant represents an assistant message.
	RoleAssistant Role = "assistant"
)

// Response represents a chat completion response.
type Response struct {
	Content      string                 `json:"content"`
	TokensUsed   *TokenUsage            `json:"tokens_used,omitempty"`
	FinishReason string                 `json:"finish_reason,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TokenUsage represents token usage statistics.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Config holds legacy configuration for LLM clients (deprecated, use LLMConfig)
// Kept for backward compatibility
type Config struct {
	Model       string   `json:"model"`
	Temperature *float32 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	TopP        *float32 `json:"top_p,omitempty"`
	TopK        *int     `json:"top_k,omitempty"`
	MinP        *float32 `json:"min_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
	BaseURL     string   `json:"base_url,omitempty"`    // Custom base URL for OpenAI-compatible services
}

// NewMessage creates a new message with the specified role and content.
func NewMessage(role Role, content string) Message {
	return Message{
		Role:    role,
		Content: content,
	}
}

// NewSystemMessage creates a new system message.
func NewSystemMessage(content string) Message {
	return NewMessage(RoleSystem, content)
}

// NewUserMessage creates a new user message.
func NewUserMessage(content string) Message {
	return NewMessage(RoleUser, content)
}

// NewAssistantMessage creates a new assistant message.
func NewAssistantMessage(content string) Message {
	return NewMessage(RoleAssistant, content)
}