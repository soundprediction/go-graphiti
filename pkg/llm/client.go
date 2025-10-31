package llm

import (
	"context"
	"encoding/json"

	"github.com/soundprediction/go-graphiti/pkg/types"
)

// Client defines the interface for language model operations.
type Client interface {
	// Chat sends a chat completion request and returns the response.
	Chat(ctx context.Context, messages []types.Message) (*types.Response, error)

	// ChatWithStructuredOutput sends a chat completion request with structured output.
	ChatWithStructuredOutput(ctx context.Context, messages []types.Message, schema any) (json.RawMessage, error)

	// Close cleans up any resources.
	Close() error
}

const (
	// RoleSystem represents a system message.
	RoleSystem types.Role = "system"
	// RoleUser represents a user message.
	RoleUser types.Role = "user"
	// RoleAssistant represents an assistant message.
	RoleAssistant types.Role = "assistant"
)

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
	BaseURL     string   `json:"base_url,omitempty"` // Custom base URL for OpenAI-compatible services
}

// NewMessage creates a new message with the specified role and content.
func NewMessage(role types.Role, content string) types.Message {
	return types.Message{
		Role:    role,
		Content: content,
	}
}

// NewSystemMessage creates a new system message.
func NewSystemMessage(content string) types.Message {
	return NewMessage(RoleSystem, content)
}

// NewUserMessage creates a new user message.
func NewUserMessage(content string) types.Message {
	return NewMessage(RoleUser, content)
}

// NewAssistantMessage creates a new assistant message.
func NewAssistantMessage(content string) types.Message {
	return NewMessage(RoleAssistant, content)
}
