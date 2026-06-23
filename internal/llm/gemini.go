// Package llm — Gemini provider implementation using google.golang.org/genai.
package llm

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// GeminiClient implements Client for the Google Gemini API.
type GeminiClient struct {
	apiKey     string
	httpClient *genai.Client
}

// NewGeminiClient creates a Gemini-backed LLM client.
func NewGeminiClient(ctx context.Context, apiKey string) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini client init: %w", err)
	}
	return &GeminiClient{apiKey: apiKey, httpClient: client}, nil
}

// Generate calls the Gemini GenerateContent API with the given request.
func (g *GeminiClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	config := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr[float32](req.Temperature),
		MaxOutputTokens: int32(req.MaxTokens),
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: req.SystemPrompt}},
		},
	}

	result, err := g.httpClient.Models.GenerateContent(
		ctx,
		req.ModelName,
		genai.Text(req.UserPrompt),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("gemini generate: %w", err)
	}

	text := result.Text()
	if text == "" {
		return nil, fmt.Errorf("gemini returned empty response")
	}

	return &GenerateResponse{
		Content: text,
		Model:   req.ModelName,
	}, nil
}

// Name returns the provider identifier.
func (g *GeminiClient) Name() string {
	return "gemini"
}
