package llm

import "errors"

// Common LLM client errors
var (
	// ErrRateLimit indicates the rate limit has been exceeded
	ErrRateLimit = errors.New("rate limit exceeded. Please try again later")
	
	// ErrRefusal indicates the LLM refused to respond to the prompt
	ErrRefusal = errors.New("the LLM refused to respond to this prompt")
	
	// ErrEmptyResponse indicates the LLM returned an empty response
	ErrEmptyResponse = errors.New("the LLM returned an empty response")
	
	// ErrInvalidModel indicates an invalid model was specified
	ErrInvalidModel = errors.New("invalid model specified")
	
	// ErrAPIKeyMissing indicates the API key is missing
	ErrAPIKeyMissing = errors.New("API key is required")
)

// RateLimitError represents a rate limit error with optional custom message
type RateLimitError struct {
	Message string
}

func (e *RateLimitError) Error() string {
	if e.Message == "" {
		return "rate limit exceeded. Please try again later"
	}
	return e.Message
}

// NewRateLimitError creates a new rate limit error with optional custom message
func NewRateLimitError(message ...string) *RateLimitError {
	err := &RateLimitError{}
	if len(message) > 0 {
		err.Message = message[0]
	}
	return err
}

// RefusalError represents an LLM refusal error
type RefusalError struct {
	Message string
}

func (e *RefusalError) Error() string {
	return e.Message
}

// NewRefusalError creates a new refusal error (message is required)
func NewRefusalError(message string) *RefusalError {
	return &RefusalError{Message: message}
}

// EmptyResponseError represents an empty response error
type EmptyResponseError struct {
	Message string
}

func (e *EmptyResponseError) Error() string {
	return e.Message
}

// NewEmptyResponseError creates a new empty response error (message is required)
func NewEmptyResponseError(message string) *EmptyResponseError {
	return &EmptyResponseError{Message: message}
}