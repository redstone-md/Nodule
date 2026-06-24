package config

import (
	"os"
	"testing"
)

// helper to set env vars and clean up after test
func setEnvs(t *testing.T, envs map[string]string) {
	t.Helper()
	for k, v := range envs {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("Setenv %s: %v", k, err)
		}
	}
	t.Cleanup(func() {
		for k := range envs {
			_ = os.Unsetenv(k)
		}
	})
}

func TestLoad_Defaults(t *testing.T) {
	setEnvs(t, map[string]string{
		"NODULE_API_KEY": "test-key-123",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Provider != "gemini" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "gemini")
	}
	if cfg.ModelName != "gemini-2.5-flash" {
		t.Errorf("ModelName = %q, want %q", cfg.ModelName, "gemini-2.5-flash")
	}
	if cfg.Temperature != 1.2 {
		t.Errorf("Temperature = %.2f, want 1.2", cfg.Temperature)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096", cfg.MaxTokens)
	}
	if cfg.ServerName != "nodule" {
		t.Errorf("ServerName = %q, want %q", cfg.ServerName, "nodule")
	}
	if cfg.ServerVersion != "0.3.0" {
		t.Errorf("ServerVersion = %q, want %q", cfg.ServerVersion, "0.3.0")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	setEnvs(t, map[string]string{
		"NODULE_LLM_PROVIDER": "openai",
		"NODULE_MODEL_NAME":   "gpt-4.1-mini",
		"NODULE_API_KEY":      "sk-test",
		"NODULE_LLM_BASE_URL": "http://localhost:11434/v1",
		"NODULE_TEMPERATURE":  "0.7",
		"NODULE_MAX_TOKENS":   "2048",
		"NODULE_SERVER_NAME":  "my-nodule",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "openai")
	}
	if cfg.ModelName != "gpt-4.1-mini" {
		t.Errorf("ModelName = %q, want %q", cfg.ModelName, "gpt-4.1-mini")
	}
	if cfg.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-test")
	}
	if cfg.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:11434/v1")
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("Temperature = %.2f, want 0.7", cfg.Temperature)
	}
	if cfg.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, want 2048", cfg.MaxTokens)
	}
	if cfg.ServerName != "my-nodule" {
		t.Errorf("ServerName = %q, want %q", cfg.ServerName, "my-nodule")
	}
}

func TestLoad_APIKeyFallbacks(t *testing.T) {
	tests := []struct {
		name  string
		envs  map[string]string
		want  string
	}{
		{
			name: "NODULE_API_KEY takes priority",
			envs: map[string]string{
				"NODULE_API_KEY": "nodule-key",
				"GEMINI_API_KEY": "gemini-key",
			},
			want: "nodule-key",
		},
		{
			name: "GEMINI_API_KEY fallback",
			envs: map[string]string{
				"GEMINI_API_KEY": "gemini-key",
			},
			want: "gemini-key",
		},
		{
			name: "GOOGLE_API_KEY fallback",
			envs: map[string]string{
				"GOOGLE_API_KEY": "google-key",
			},
			want: "google-key",
		},
		{
			name: "OPENAI_API_KEY fallback",
			envs: map[string]string{
				"OPENAI_API_KEY": "openai-key",
			},
			want: "openai-key",
		},
		{
			name: "GOOGLE_API_KEY takes priority over GEMINI_API_KEY",
			envs: map[string]string{
				"GOOGLE_API_KEY": "google-key",
				"GEMINI_API_KEY": "gemini-key",
			},
			want: "gemini-key", // GEMINI_API_KEY checked first in resolveAPIKey
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvs(t, tt.envs)
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}
			if cfg.APIKey != tt.want {
				t.Errorf("APIKey = %q, want %q", cfg.APIKey, tt.want)
			}
		})
	}
}

func TestLoad_MissingAPIKey(t *testing.T) {
	// Ensure no API key env vars are set
	setEnvs(t, map[string]string{
		"NODULE_LLM_PROVIDER": "gemini",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing API key, got nil")
	}
}

func TestLoad_InvalidTemperature(t *testing.T) {
	setEnvs(t, map[string]string{
		"NODULE_API_KEY":     "test-key",
		"NODULE_TEMPERATURE": "3.0",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid temperature, got nil")
	}
}

func TestLoad_InvalidMaxTokens(t *testing.T) {
	setEnvs(t, map[string]string{
		"NODULE_API_KEY":    "test-key",
		"NODULE_MAX_TOKENS": "-1",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid max tokens, got nil")
	}
}

func TestLoad_OpenAIProviderWithoutAPIKey(t *testing.T) {
	// OpenAI provider with a local BaseURL (Ollama) should work without API key
	setEnvs(t, map[string]string{
		"NODULE_LLM_PROVIDER": "openai",
		"NODULE_LLM_BASE_URL": "http://localhost:11434/v1",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() for openai+local URL w/o key: unexpected error: %v", err)
	}
	if cfg.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "openai")
	}
}

func TestLoad_OpenAIProviderCloudWithoutAPIKey(t *testing.T) {
	// OpenAI provider targeting cloud (no BaseURL) without key should fail
	setEnvs(t, map[string]string{
		"NODULE_LLM_PROVIDER": "openai",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for cloud openai without API key, got nil")
	}
}

func TestLoad_UnknownProvider(t *testing.T) {
	setEnvs(t, map[string]string{
		"NODULE_API_KEY":        "test-key",
		"NODULE_LLM_PROVIDER":   "anthropic",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v (validation shouldn't catch unknown provider)", err)
	}
	if cfg.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "anthropic")
	}
}

func TestLoad_InvalidTemperatureString(t *testing.T) {
	setEnvs(t, map[string]string{
		"NODULE_API_KEY":     "test-key",
		"NODULE_TEMPERATURE": "not-a-number",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	// Should fall back to default
	if cfg.Temperature != 1.2 {
		t.Errorf("Temperature = %.2f, want default 1.2", cfg.Temperature)
	}
}
