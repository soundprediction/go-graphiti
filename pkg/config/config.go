package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Log configuration
	Log LogConfig `mapstructure:"log"`

	// Server configuration
	Server ServerConfig `mapstructure:"server"`

	// Database configuration
	Database DatabaseConfig `mapstructure:"database"`

	// LLM configuration
	LLM LLMConfig `mapstructure:"llm"`

	// Embedding configuration
	Embedding EmbeddingConfig `mapstructure:"embedding"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // gin mode: debug, release, test
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"` // neo4j, falkordb
	URI      string `mapstructure:"uri"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

// LLMConfig holds LLM configuration
type LLMConfig struct {
	Provider    string  `mapstructure:"provider"` // openai, anthropic, etc.
	Model       string  `mapstructure:"model"`
	APIKey      string  `mapstructure:"api_key"`
	BaseURL     string  `mapstructure:"base_url"`
	Temperature float32 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
}

// EmbeddingConfig holds embedding configuration
type EmbeddingConfig struct {
	Provider string `mapstructure:"provider"` // openai, etc.
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Set defaults
	setDefaults()

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// Override with environment variables if present
	overrideWithEnv(config)

	return config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")

	// Server defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")

	// Database defaults
	viper.SetDefault("database.driver", "kuzu")
	viper.SetDefault("database.uri", "./kuzu_db")
	viper.SetDefault("database.username", "")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.database", "")

	// LLM defaults
	viper.SetDefault("llm.provider", "openai")
	viper.SetDefault("llm.model", "gpt-4")
	viper.SetDefault("llm.temperature", 0.1)
	viper.SetDefault("llm.max_tokens", 2048)

	// Embedding defaults
	viper.SetDefault("embedding.provider", "openai")
	viper.SetDefault("embedding.model", "text-embedding-3-small")
}

// overrideWithEnv overrides config with environment variables
func overrideWithEnv(config *Config) {
	// LLM API Key
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
		config.Embedding.APIKey = apiKey
	}
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" && config.LLM.Provider == "anthropic" {
		config.LLM.APIKey = apiKey
	}

	// Database credentials
	if uri := os.Getenv("NEO4J_URI"); uri != "" {
		config.Database.URI = uri
	}
	if user := os.Getenv("NEO4J_USER"); user != "" {
		config.Database.Username = user
	}
	if pass := os.Getenv("NEO4J_PASSWORD"); pass != "" {
		config.Database.Password = pass
	}

	// Kuzu database path
	if dbPath := os.Getenv("KUZU_DB_PATH"); dbPath != "" {
		config.Database.URI = dbPath
	}

	// Generic database settings
	if dbDriver := os.Getenv("DB_DRIVER"); dbDriver != "" {
		config.Database.Driver = dbDriver
	}
	if dbURI := os.Getenv("DB_URI"); dbURI != "" {
		config.Database.URI = dbURI
	}

	// Server settings
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		viper.Set("server.port", port)
	}
}
