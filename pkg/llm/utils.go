package llm

import (
	"strings"
	"unicode"
)

// TokenCounter provides token counting functionality.
type TokenCounter interface {
	CountTokens(text string) int
}

// SimpleTokenCounter provides a basic token counting implementation.
// This is a simplified version - for production use, you should use
// model-specific tokenizers like tiktoken for OpenAI models.
type SimpleTokenCounter struct{}

// NewSimpleTokenCounter creates a new simple token counter.
func NewSimpleTokenCounter() *SimpleTokenCounter {
	return &SimpleTokenCounter{}
}

// CountTokens estimates token count using a simple word-based approach.
// This is a rough approximation - real tokenizers are more complex.
func (s *SimpleTokenCounter) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// Basic tokenization: split on whitespace and punctuation
	words := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})

	// Remove empty strings
	tokenCount := 0
	for _, word := range words {
		if strings.TrimSpace(word) != "" {
			tokenCount++
		}
	}

	// Rough estimation: tokens are often ~0.75 of words for English
	// Add some overhead for special tokens
	return int(float64(tokenCount) * 1.3)
}

// GetTokenCount is a convenience function that uses the simple token counter.
// This matches the Python get_token_count function signature.
func GetTokenCount(text string) int {
	counter := NewSimpleTokenCounter()
	return counter.CountTokens(text)
}

// EstimateTokensFromMessages estimates tokens for a slice of messages.
func EstimateTokensFromMessages(messages []Message) int {
	totalTokens := 0
	for _, msg := range messages {
		totalTokens += GetTokenCount(msg.Content)
		totalTokens += 4 // Overhead for role and formatting
	}
	return totalTokens
}
