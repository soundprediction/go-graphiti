package llm_test

import (
	"testing"

	"github.com/getzep/go-graphiti/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenAICompatibleClient(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		apiKey      string
		model       string
		config      llm.Config
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid http URL",
			baseURL:     "http://localhost:11434",
			apiKey:      "",
			model:       "llama2:7b",
			config:      llm.Config{},
			shouldError: false,
		},
		{
			name:        "valid https URL",
			baseURL:     "https://api.example.com",
			apiKey:      "test-key",
			model:       "gpt-3.5-turbo",
			config:      llm.Config{},
			shouldError: false,
		},
		{
			name:        "URL with existing v1 path",
			baseURL:     "http://localhost:8080/v1",
			apiKey:      "",
			model:       "test-model",
			config:      llm.Config{},
			shouldError: false,
		},
		{
			name:        "empty base URL",
			baseURL:     "",
			apiKey:      "key",
			model:       "model",
			config:      llm.Config{},
			shouldError: true,
			errorMsg:    "baseURL cannot be empty",
		},
		{
			name:        "invalid URL format",
			baseURL:     "not-a-url",
			apiKey:      "",
			model:       "model",
			config:      llm.Config{},
			shouldError: true,
			errorMsg:    "baseURL must include scheme",
		},
		{
			name:        "URL without http/https scheme",
			baseURL:     "localhost:8080",
			apiKey:      "",
			model:       "model",
			config:      llm.Config{},
			shouldError: true,
			errorMsg:    "baseURL must use http:// or https:// scheme",
		},
		{
			name:        "default model when empty",
			baseURL:     "http://localhost:8080",
			apiKey:      "",
			model:       "", // Should default to gpt-3.5-turbo
			config:      llm.Config{},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := llm.NewOpenAICompatibleClient(tt.baseURL, tt.apiKey, tt.model, tt.config)
			
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

func TestConvenienceFunctions(t *testing.T) {
	t.Run("NewOllamaClient", func(t *testing.T) {
		// Test with custom URL
		client, err := llm.NewOllamaClient("http://localhost:11434", "llama2:7b", llm.Config{})
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NoError(t, client.Close())
		
		// Test with default URL (empty string)
		client2, err := llm.NewOllamaClient("", "llama2:7b", llm.Config{})
		require.NoError(t, err)
		assert.NotNil(t, client2)
		assert.NoError(t, client2.Close())
	})

	t.Run("NewLocalAIClient", func(t *testing.T) {
		client, err := llm.NewLocalAIClient("http://localhost:8080", "gpt-3.5-turbo", llm.Config{})
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NoError(t, client.Close())
		
		// Test with default URL
		client2, err := llm.NewLocalAIClient("", "gpt-3.5-turbo", llm.Config{})
		require.NoError(t, err)
		assert.NotNil(t, client2)
		assert.NoError(t, client2.Close())
	})

	t.Run("NewVLLMClient", func(t *testing.T) {
		client, err := llm.NewVLLMClient("http://vllm-server:8000", "microsoft/DialoGPT-medium", llm.Config{})
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NoError(t, client.Close())
	})

	t.Run("NewTextGenerationInferenceClient", func(t *testing.T) {
		client, err := llm.NewTextGenerationInferenceClient("http://tgi-server:3000", "bigscience/bloom", llm.Config{})
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
			client, err := llm.NewOpenAICompatibleClient(tt.baseURL, "", "test-model", llm.Config{})
			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.NoError(t, client.Close())
		})
	}
}

func TestClientConfiguration(t *testing.T) {
	config := llm.Config{
		Model:       "test-model",
		Temperature: &[]float32{0.8}[0],
		MaxTokens:   &[]int{1000}[0],
		TopP:        &[]float32{0.9}[0],
		Stop:        []string{"</s>", "\n\n"},
	}

	client, err := llm.NewOpenAICompatibleClient("http://localhost:8080", "test-key", "test-model", config)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NoError(t, client.Close())
}

// Example test showing how to use the client (though it won't actually make requests in tests)
func TestClientUsageExample(t *testing.T) {
	// This test demonstrates usage patterns but doesn't make actual API calls
	client, err := llm.NewOllamaClient("http://localhost:11434", "llama2:7b", llm.Config{
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