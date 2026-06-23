// Package llm defines the abstract LLM client interface and request/response types.
// Provider implementations (Gemini, OpenAI-compatible) are in separate files.
package llm

import "context"

// GenerateRequest is the provider-agnostic input for an LLM generation call.
type GenerateRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float32
	MaxTokens    int
	ModelName    string
}

// GenerateResponse is the provider-agnostic output from an LLM generation call.
type GenerateResponse struct {
	Content string
	Model   string
}

// Client is the interface that all LLM providers must implement.
// This enables BYOM (Bring Your Own Model) — any provider that satisfies
// this contract can be plugged into the Nodule server.
type Client interface {
	// Generate sends a prompt to the LLM and returns the generated text.
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)

	// Name returns the provider identifier (e.g. "gemini", "openai").
	Name() string
}
