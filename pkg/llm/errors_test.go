package llm_test

import (
	"testing"

	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitError(t *testing.T) {
	t.Run("default message", func(t *testing.T) {
		err := llm.NewRateLimitError()
		assert.Equal(t, "rate limit exceeded. Please try again later", err.Error())
	})

	t.Run("custom message", func(t *testing.T) {
		customMessage := "Custom rate limit message"
		err := llm.NewRateLimitError(customMessage)
		assert.Equal(t, customMessage, err.Error())
	})
}

func TestRefusalError(t *testing.T) {
	t.Run("message assignment", func(t *testing.T) {
		message := "The LLM refused to respond to this prompt."
		err := llm.NewRefusalError(message)
		assert.Equal(t, message, err.Error())
	})
}

func TestEmptyResponseError(t *testing.T) {
	t.Run("message assignment", func(t *testing.T) {
		message := "The LLM returned an empty response."
		err := llm.NewEmptyResponseError(message)
		assert.Equal(t, message, err.Error())
	})
}

func TestCommonErrors(t *testing.T) {
	t.Run("error constants", func(t *testing.T) {
		assert.NotNil(t, llm.ErrRateLimit)
		assert.NotNil(t, llm.ErrRefusal)
		assert.NotNil(t, llm.ErrEmptyResponse)
		assert.NotNil(t, llm.ErrInvalidModel)

		assert.Contains(t, llm.ErrRateLimit.Error(), "rate limit")
		assert.Contains(t, llm.ErrRefusal.Error(), "refused")
		assert.Contains(t, llm.ErrEmptyResponse.Error(), "empty")
		assert.Contains(t, llm.ErrInvalidModel.Error(), "invalid model")
	})
}
