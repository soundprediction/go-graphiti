package llm

// ModelSize represents the size/complexity of the model to use
type ModelSize string

const (
	// ModelSizeSmall represents a small, fast model for simple tasks
	ModelSizeSmall ModelSize = "small"
	// ModelSizeMedium represents a medium model for more complex tasks
	ModelSizeMedium ModelSize = "medium"
)

// Default configuration values
const (
	DefaultMaxTokens   = 8192
	DefaultTemperature = 1.0
)

// LLMConfig holds configuration for LLM clients, matching Python LLMConfig structure
type LLMConfig struct {
	// APIKey is the authentication key for accessing the LLM API
	APIKey string `json:"api_key,omitempty"`

	// Model is the specific LLM model to use for generating responses
	Model string `json:"model,omitempty"`

	// BaseURL is the base URL of the LLM API service
	BaseURL string `json:"base_url,omitempty"`

	// Temperature controls randomness in generation (0.0 to 2.0)
	Temperature float32 `json:"temperature,omitempty"`

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int `json:"max_tokens,omitempty"`

	// SmallModel is the model to use for simpler prompts
	SmallModel string `json:"small_model,omitempty"`
}

// NewLLMConfig creates a new LLMConfig with default values
func NewLLMConfig() *LLMConfig {
	return &LLMConfig{
		Temperature: DefaultTemperature,
		MaxTokens:   DefaultMaxTokens,
	}
}

// WithAPIKey sets the API key
func (c *LLMConfig) WithAPIKey(apiKey string) *LLMConfig {
	c.APIKey = apiKey
	return c
}

// WithModel sets the model
func (c *LLMConfig) WithModel(model string) *LLMConfig {
	c.Model = model
	return c
}

// WithBaseURL sets the base URL
func (c *LLMConfig) WithBaseURL(baseURL string) *LLMConfig {
	c.BaseURL = baseURL
	return c
}

// WithTemperature sets the temperature
func (c *LLMConfig) WithTemperature(temperature float32) *LLMConfig {
	c.Temperature = temperature
	return c
}

// WithMaxTokens sets the max tokens
func (c *LLMConfig) WithMaxTokens(maxTokens int) *LLMConfig {
	c.MaxTokens = maxTokens
	return c
}

// WithSmallModel sets the small model
func (c *LLMConfig) WithSmallModel(smallModel string) *LLMConfig {
	c.SmallModel = smallModel
	return c
}