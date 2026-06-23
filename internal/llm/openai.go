// Package llm — OpenAI-compatible provider implementation.
// Supports any service with an /v1/chat/completions endpoint:
// OpenAI, Ollama, vLLM, LM Studio, Together AI, etc.
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

// OpenAIClient implements Client for any OpenAI-compatible API.
type OpenAIClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// openAIChatRequest mirrors the OpenAI chat/completions request body.
type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float32             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIChatResponse mirrors the OpenAI chat/completions response body.
type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
	Model   string         `json:"model"`
}

type openAIChoice struct {
	Message openAIChatMessage `json:"message"`
}

// NewOpenAIClient creates an OpenAI-compatible LLM client.
// baseURL should be the API root (e.g. "https://api.openai.com/v1" or "http://localhost:11434/v1").
func NewOpenAIClient(apiKey, baseURL string) *OpenAIClient {
	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Generate calls the /chat/completions endpoint with the given request.
func (o *OpenAIClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	chatReq := openAIChatRequest{
		Model: req.ModelName,
		Messages: []openAIChatMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserPrompt},
		},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("openai marshal: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", o.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("openai decode: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	return &GenerateResponse{
		Content: chatResp.Choices[0].Message.Content,
		Model:   chatResp.Model,
	}, nil
}

// Name returns the provider identifier.
func (o *OpenAIClient) Name() string {
	return "openai"
}
