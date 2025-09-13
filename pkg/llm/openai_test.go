package llm_test

import (
	"testing"

	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestNewOpenAIClient(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		config      llm.Config
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid http URL",
			apiKey:      "",
			config:      llm.Config{BaseURL: "http://localhost:11434", Model: "llama2:7b"},
			shouldError: false,
		},
		{
			name:        "valid https URL",
			apiKey:      "test-key",
			config:      llm.Config{BaseURL: "https://api.example.com", Model: "gpt-3.5-turbo"},
			shouldError: false,
		},
		{
			name:        "URL with existing v1 path",
			apiKey:      "",
			config:      llm.Config{BaseURL: "http://localhost:8080/v1", Model: "test-model"},
			shouldError: false,
		},
		{
			name:        "empty base URL (uses OpenAI)",
			apiKey:      "key",
			config:      llm.Config{Model: "model"},
			shouldError: false,
		},
		{
			name:        "invalid URL format",
			apiKey:      "",
			config:      llm.Config{BaseURL: "not-a-url", Model: "model"},
			shouldError: true,
			errorMsg:    "baseURL must include scheme",
		},
		{
			name:        "URL without http/https scheme",
			apiKey:      "",
			config:      llm.Config{BaseURL: "localhost:8080", Model: "model"},
			shouldError: true,
			errorMsg:    "baseURL must use http:// or https:// scheme",
		},
		{
			name:        "default model when empty",
			apiKey:      "",
			config:      llm.Config{BaseURL: "http://localhost:8080"}, // Should default to gpt-3.5-turbo
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := llm.NewOpenAIClient(tt.apiKey, tt.config)

			if tt.shouldError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				assert.NoError(t, client.Close())
			}
		})
	}
}

func TestOpenAICompatibleServices(t *testing.T) {
	t.Run("OllamaClient", func(t *testing.T) {
		// Test with custom URL
		client, err := llm.NewOpenAIClient("", llm.Config{BaseURL: "http://localhost:11434", Model: "llama2:7b"})
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NoError(t, client.Close())

		// Test with default URL pattern
		client2, err := llm.NewOpenAIClient("", llm.Config{BaseURL: "http://localhost:11434", Model: "llama2:7b"})
		require.NoError(t, err)
		assert.NotNil(t, client2)
		assert.NoError(t, client2.Close())
	})

	t.Run("LocalAIClient", func(t *testing.T) {
		client, err := llm.NewOpenAIClient("", llm.Config{BaseURL: "http://localhost:8080", Model: "gpt-3.5-turbo"})
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NoError(t, client.Close())

		// Test with another LocalAI instance
		client2, err := llm.NewOpenAIClient("", llm.Config{BaseURL: "http://localhost:9090", Model: "gpt-3.5-turbo"})
		require.NoError(t, err)
		assert.NotNil(t, client2)
		assert.NoError(t, client2.Close())
	})

	t.Run("VLLMClient", func(t *testing.T) {
		client, err := llm.NewOpenAIClient("", llm.Config{BaseURL: "http://vllm-server:8000", Model: "microsoft/DialoGPT-medium"})
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NoError(t, client.Close())
	})

	t.Run("TextGenerationInferenceClient", func(t *testing.T) {
		client, err := llm.NewOpenAIClient("", llm.Config{BaseURL: "http://tgi-server:3000", Model: "bigscience/bloom"})
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NoError(t, client.Close())
	})
}

func TestHasAPIPath(t *testing.T) {
	// Note: hasAPIPath is not exported, so we test it indirectly through client creation
	tests := []struct {
		name    string
		baseURL string
		// We can't directly test the internal hasAPIPath function,
		// but we can verify the client is created successfully
	}{
		{"URL with /v1", "http://localhost:8080/v1"},
		{"URL with /api", "http://localhost:8080/api"},
		{"URL with /v1/", "http://localhost:8080/v1/"},
		{"URL with /api/", "http://localhost:8080/api/"},
		{"URL without path", "http://localhost:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := llm.NewOpenAIClient("", llm.Config{BaseURL: tt.baseURL, Model: "test-model"})
			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.NoError(t, client.Close())
		})
	}
}

func TestClientConfiguration(t *testing.T) {
	config := llm.Config{
		BaseURL:     "http://localhost:8080",
		Model:       "test-model",
		Temperature: &[]float32{0.8}[0],
		MaxTokens:   &[]int{1000}[0],
		TopP:        &[]float32{0.9}[0],
		Stop:        []string{"</s>", "\n\n"},
	}

	client, err := llm.NewOpenAIClient("test-key", config)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NoError(t, client.Close())
}

// Example test showing how to use the client (though it won't actually make requests in tests)
func TestClientUsageExample(t *testing.T) {
	// This test demonstrates usage patterns but doesn't make actual API calls
	client, err := llm.NewOpenAIClient("", llm.Config{
		BaseURL:     "http://localhost:11434",
		Model:       "llama2:7b",
		Temperature: &[]float32{0.7}[0],
		MaxTokens:   &[]int{500}[0],
	})
	require.NoError(t, err)
	assert.NotNil(t, client)

	// In a real scenario, you would use:
	// messages := []llm.Message{
	//     llm.NewUserMessage("Hello, how are you?"),
	// }
	// response, err := client.Chat(context.Background(), messages)

	assert.NoError(t, client.Close())
}
