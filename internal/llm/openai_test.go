package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIClient_Generate_Success(t *testing.T) {
	var receivedBody openAIChatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Path = %q, want /chat/completions", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("Authorization = %q, want 'Bearer test-key'", auth)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &receivedBody)

		resp := openAIChatResponse{
			Choices: []openAIChoice{
				{Message: openAIChatMessage{Role: "assistant", Content: "Critical: O(N²) loop at line 42"}},
			},
			Model: "test-model",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient("test-key", server.URL)

	resp, err := client.Generate(context.Background(), GenerateRequest{
		SystemPrompt: "You are a critic",
		UserPrompt:   "Review this code",
		Temperature:  0.9,
		MaxTokens:    1024,
		ModelName:    "gpt-4.1-mini",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	if resp.Content != "Critical: O(N²) loop at line 42" {
		t.Errorf("Content = %q, want critique text", resp.Content)
	}
	if resp.Model != "test-model" {
		t.Errorf("Model = %q, want %q", resp.Model, "test-model")
	}

	// Verify the request body was correctly assembled
	if receivedBody.Model != "gpt-4.1-mini" {
		t.Errorf("Request model = %q, want %q", receivedBody.Model, "gpt-4.1-mini")
	}
	if len(receivedBody.Messages) != 2 {
		t.Fatalf("Messages count = %d, want 2", len(receivedBody.Messages))
	}
	if receivedBody.Messages[0].Role != "system" || receivedBody.Messages[0].Content != "You are a critic" {
		t.Errorf("System message = %+v", receivedBody.Messages[0])
	}
	if receivedBody.Messages[1].Role != "user" || receivedBody.Messages[1].Content != "Review this code" {
		t.Errorf("User message = %+v", receivedBody.Messages[1])
	}
	if receivedBody.Temperature != 0.9 {
		t.Errorf("Temperature = %.2f, want 0.9", receivedBody.Temperature)
	}
	if receivedBody.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %d, want 1024", receivedBody.MaxTokens)
	}
}

func TestOpenAIClient_Generate_NoAPIKey(t *testing.T) {
	// Local servers (Ollama) don't require auth — verify no Authorization header sent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("Authorization should be empty for keyless client, got %q", auth)
		}
		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			Choices: []openAIChoice{
				{Message: openAIChatMessage{Content: "ok"}},
			},
		})
	}))
	defer server.Close()

	client := NewOpenAIClient("", server.URL)
	resp, err := client.Generate(context.Background(), GenerateRequest{
		SystemPrompt: "s",
		UserPrompt:   "u",
		ModelName:    "local-model",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("Content = %q, want %q", resp.Content, "ok")
	}
}

func TestOpenAIClient_Generate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"rate limited"}`, http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewOpenAIClient("key", server.URL)
	_, err := client.Generate(context.Background(), GenerateRequest{
		SystemPrompt: "s",
		UserPrompt:   "u",
		ModelName:    "m",
	})
	if err == nil {
		t.Fatal("Generate() expected error for HTTP 429, got nil")
	}
}

func TestOpenAIClient_Generate_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(openAIChatResponse{Choices: []openAIChoice{}})
	}))
	defer server.Close()

	client := NewOpenAIClient("key", server.URL)
	_, err := client.Generate(context.Background(), GenerateRequest{
		SystemPrompt: "s",
		UserPrompt:   "u",
		ModelName:    "m",
	})
	if err == nil {
		t.Fatal("Generate() expected error for empty choices, got nil")
	}
}

func TestOpenAIClient_Name(t *testing.T) {
	client := NewOpenAIClient("key", "http://localhost")
	if client.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", client.Name(), "openai")
	}
}
