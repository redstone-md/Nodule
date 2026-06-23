// Package main is the entry point for the Nodule MCP server.
// Nodule provides a bounce_idea tool that delegates architectural critique
// to a configurable LLM provider (Bring Your Own Model).
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redstone-md/nodule/internal/config"
	"github.com/redstone-md/nodule/internal/llm"
	"github.com/redstone-md/nodule/internal/mcpserver"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	client, err := createLLMClient(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LLM client error: %v\n", err)
		os.Exit(1)
	}

	log.Printf("nodule: provider=%s model=%s temperature=%.2f", client.Name(), cfg.ModelName, cfg.Temperature)

	srv := mcpserver.New(client, cfg.ModelName, cfg.Temperature, cfg.MaxTokens, cfg.ServerName, cfg.ServerVersion)

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("nodule: server error: %v", err)
	}
}

// createLLMClient builds the appropriate LLM client based on provider config.
func createLLMClient(ctx context.Context, cfg *config.Config) (llm.Client, error) {
	switch cfg.Provider {
	case "gemini":
		return llm.NewGeminiClient(ctx, cfg.APIKey)
	case "openai":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return llm.NewOpenAIClient(cfg.APIKey, baseURL), nil
	default:
		return nil, fmt.Errorf("unknown provider %q: use 'gemini' or 'openai'", cfg.Provider)
	}
}
