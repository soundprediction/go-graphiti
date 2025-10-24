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

// AzureOpenAIClient implements the Client interface for Azure OpenAI models.
type AzureOpenAIClient struct {
	config       *LLMConfig
	httpClient   *http.Client
	apiVersion   string
	deploymentID string
}

// AzureOpenAIConfig extends LLMConfig with Azure-specific settings.
type AzureOpenAIConfig struct {
	*LLMConfig
	APIVersion   string `json:"api_version,omitempty"`
	DeploymentID string `json:"deployment_id,omitempty"`
}

// NewAzureOpenAIClient creates a new Azure OpenAI client.
func NewAzureOpenAIClient(config *AzureOpenAIConfig) *AzureOpenAIClient {
	if config.APIVersion == "" {
		config.APIVersion = "2024-02-15-preview"
	}

	return &AzureOpenAIClient{
		config:       config.LLMConfig,
		apiVersion:   config.APIVersion,
		deploymentID: config.DeploymentID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// azureOpenAIRequest represents the request structure for Azure OpenAI API.
type azureOpenAIRequest struct {
	Messages    []azureOpenAIMessage `json:"messages"`
	MaxTokens   int                  `json:"max_tokens,omitempty"`
	Temperature float64              `json:"temperature,omitempty"`
	Stream      bool                 `json:"stream"`
}

// azureOpenAIMessage represents a message in Azure OpenAI format.
type azureOpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// azureOpenAIResponse represents the response from Azure OpenAI API.
type azureOpenAIResponse struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []azureOpenAIChoice `json:"choices"`
	Error   *azureOpenAIError   `json:"error,omitempty"`
}

// azureOpenAIChoice represents a choice in the response.
type azureOpenAIChoice struct {
	Index        int                `json:"index"`
	Message      azureOpenAIMessage `json:"message"`
	FinishReason string             `json:"finish_reason"`
}

// azureOpenAIError represents an error response.
type azureOpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Chat implements the Client interface for Azure OpenAI.
func (a *AzureOpenAIClient) Chat(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages provided")
	}

	if a.deploymentID == "" {
		return "", fmt.Errorf("deployment ID is required for Azure OpenAI")
	}

	// Convert messages to Azure OpenAI format
	azureMessages := make([]azureOpenAIMessage, 0, len(messages))
	for _, msg := range messages {
		azureMessages = append(azureMessages, azureOpenAIMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	req := azureOpenAIRequest{
		Messages:    azureMessages,
		MaxTokens:   a.config.MaxTokens,
		Temperature: float64(a.config.Temperature),
		Stream:      false,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Azure OpenAI URL format: https://{resource-name}.openai.azure.com/openai/deployments/{deployment-id}/chat/completions?api-version={api-version}
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		a.config.BaseURL, a.deploymentID, a.apiVersion)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", a.config.APIKey)

	resp, err := a.httpClient.Do(httpReq)
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

	var azureResp azureOpenAIResponse
	if err := json.Unmarshal(body, &azureResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if azureResp.Error != nil {
		return "", fmt.Errorf("API error: %s", azureResp.Error.Message)
	}

	if len(azureResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return azureResp.Choices[0].Message.Content, nil
}

// ChatWithStructuredOutput implements structured output for Azure OpenAI.
// Azure OpenAI supports structured output similar to OpenAI.
func (a *AzureOpenAIClient) ChatWithStructuredOutput(ctx context.Context, messages []Message, schema interface{}) (string, error) {
	// For now, use prompt engineering approach
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	modifiedMessages := append(messages, Message{
		Role:    RoleUser,
		Content: fmt.Sprintf("Please respond with valid JSON that matches this schema: %s", string(schemaBytes)),
	})

	return a.Chat(ctx, modifiedMessages)
}
