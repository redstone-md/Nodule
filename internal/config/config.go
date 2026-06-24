// Package config provides environment-based configuration for the Nodule MCP server.
// All configuration is loaded from environment variables with sensible defaults.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all server configuration loaded from environment variables.
type Config struct {
	// LLM settings
	Provider    string  // NODULE_LLM_PROVIDER: "gemini" | "openai" (default: "gemini")
	ModelName   string  // NODULE_MODEL_NAME: model identifier (default: "gemini-2.5-flash")
	APIKey      string  // NODULE_API_KEY (fallback: GEMINI_API_KEY, OPENAI_API_KEY)
	BaseURL     string  // NODULE_LLM_BASE_URL: for OpenAI-compatible endpoints
	Temperature float32 // NODULE_TEMPERATURE (default: 1.2)
	MaxTokens   int     // NODULE_MAX_TOKENS (default: 4096)

	// Server settings
	ServerName    string // NODULE_SERVER_NAME (default: "nodule")
	ServerVersion string // NODULE_SERVER_VERSION (default: "0.1.0")
}

// Load reads configuration from environment variables.
// Returns an error if required values are missing.
func Load() (*Config, error) {
	cfg := &Config{
		Provider:      getEnv("NODULE_LLM_PROVIDER", "gemini"),
		ModelName:     getEnv("NODULE_MODEL_NAME", "gemini-2.5-flash"),
		APIKey:        resolveAPIKey(),
		BaseURL:       getEnv("NODULE_LLM_BASE_URL", ""),
		Temperature:   getEnvFloat("NODULE_TEMPERATURE", 1.2),
		MaxTokens:     getEnvInt("NODULE_MAX_TOKENS", 4096),
		ServerName:    getEnv("NODULE_SERVER_NAME", "nodule"),
		ServerVersion: getEnv("NODULE_SERVER_VERSION", "0.4.0"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate ensures required configuration values are present.
func (c *Config) validate() error {
	if c.APIKey == "" {
		// Allow no key for local OpenAI-compatible servers (Ollama, vLLM, LM Studio)
		// where a BaseURL is configured. Cloud providers need a key.
		if c.Provider == "openai" && c.BaseURL != "" {
			// local server, key optional
		} else {
			return fmt.Errorf("NODULE_API_KEY is required for provider %q (set NODULE_API_KEY, GEMINI_API_KEY, or OPENAI_API_KEY)", c.Provider)
		}
	}
	if c.Temperature < 0 || c.Temperature > 2.0 {
		return fmt.Errorf("NODULE_TEMPERATURE must be between 0 and 2.0, got %.2f", c.Temperature)
	}
	if c.MaxTokens < 1 {
		return fmt.Errorf("NODULE_MAX_TOKENS must be positive, got %d", c.MaxTokens)
	}
	return nil
}

// resolveAPIKey finds the API key from NODULE_API_KEY or a provider-specific fallback.
func resolveAPIKey() string {
	if key := os.Getenv("NODULE_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key
	}
	return ""
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvFloat(key string, fallback float32) float32 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return fallback
	}
	return float32(parsed)
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}
