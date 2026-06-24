// Package mcpserver wires the Nodule MCP server, registers the bounce_idea tool,
// and implements the tool handler that delegates to an LLM client.
package mcpserver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/redstone-md/nodule/internal/llm"
	"github.com/redstone-md/nodule/internal/prompt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// BounceIdeaInput defines the typed input for the bounce_idea tool.
type BounceIdeaInput struct {
	Context          string `json:"context"                jsonschema:"Current codebase state, function signatures, benchmarks or error logs. Required."`
	ProposedSolution string `json:"proposed_solution"       jsonschema:"Draft idea or code the main agent plans to apply. Required."`
	Focus            string `json:"focus"                   jsonschema:"Critique specialization: performance, architecture, security, edge_cases. Default: performance."`
}

// Server wraps the MCP server and its dependencies.
type Server struct {
	mcpServer   *mcp.Server
	llmClient   llm.Client
	promptBldr  *prompt.Builder
	modelName   string
	temperature float32
	maxTokens   int
}

// New creates a Server with the given LLM client and generation parameters.
func New(client llm.Client, modelName string, temperature float32, maxTokens int, serverName, serverVersion string) *Server {
	s := &Server{
		mcpServer: mcp.NewServer(&mcp.Implementation{
			Name:    serverName,
			Version: serverVersion,
		}, nil),
		llmClient:   client,
		promptBldr:  prompt.NewBuilder(),
		modelName:   modelName,
		temperature: temperature,
		maxTokens:   maxTokens,
	}

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "bounce_idea",
		Description: "Calls an independent external architect to critique your current solution, find hidden bugs, race conditions, edge-cases, or optimize algorithm performance.",
	}, s.handleBounceIdea)

	return s
}

// handleBounceIdea is the typed tool handler for bounce_idea.
func (s *Server) handleBounceIdea(ctx context.Context, req *mcp.CallToolRequest, input BounceIdeaInput) (*mcp.CallToolResult, any, error) {
	focus := prompt.ParseFocus(input.Focus)

	prompts := s.promptBldr.Build(input.Context, input.ProposedSolution, focus)

	llmResp, err := s.llmClient.Generate(ctx, llm.GenerateRequest{
		SystemPrompt: prompts.SystemPrompt,
		UserPrompt:   prompts.UserPrompt,
		Temperature:  s.temperature,
		MaxTokens:    s.maxTokens,
		ModelName:    s.modelName,
	})
	if err != nil {
		log.Printf("bounce_idea: LLM generation failed: %v", err)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Critic generation failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	header := fmt.Sprintf("# Critic Analysis [%s / %s / %s]\n\n", s.llmClient.Name(), s.modelName, string(focus))
	markdown := header + llmResp.Content

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: markdown}},
	}, nil, nil
}

// Run starts the MCP server on stdio transport (blocking).
// Uses a BOM-safe reader for Windows compatibility — some MCP clients
// and PowerShell pipes emit a UTF-8 BOM that breaks JSON-RPC parsing.
func (s *Server) Run(ctx context.Context) error {
	transport := &mcp.IOTransport{
		Reader: newBOMSafeReader(os.Stdin),
		Writer: nopWriteCloser{os.Stdout},
	}
	return s.mcpServer.Run(ctx, transport)
}

// --- BOM-safe reader ---

// UTF-8 BOM emitted by Windows tools and PowerShell pipes.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// bomSafeReader wraps an io.ReadCloser and strips a leading UTF-8 BOM.
// Uses bufio.Reader to Peek at the first 3 bytes without consuming them.
type bomSafeReader struct {
	br      *bufio.Reader
	src     io.ReadCloser
	checked bool
}

// newBOMSafeReader creates a reader that transparently skips a leading
// UTF-8 BOM if present, then passes all reads through to the underlying source.
func newBOMSafeReader(src io.ReadCloser) *bomSafeReader {
	return &bomSafeReader{
		br:  bufio.NewReaderSize(src, 4096),
		src: src,
	}
}

func (r *bomSafeReader) Read(p []byte) (int, error) {
	// On the first read, strip all leading UTF-8 BOM sequences.
	// Some Windows pipes emit multiple BOMs (e.g. PowerShell StreamWriter
	// prepends BOM on each write).
	if !r.checked {
		r.checked = true
		for {
			peek, _ := r.br.Peek(3)
			if len(peek) == 3 && peek[0] == utf8BOM[0] && peek[1] == utf8BOM[1] && peek[2] == utf8BOM[2] {
				r.br.Discard(3)
				continue
			}
			break
		}
	}
	n, err := r.br.Read(p)
	// Defensive: strip a BOM that lands at the start of a fresh buffer.
	for n >= 3 && p[0] == utf8BOM[0] && p[1] == utf8BOM[1] && p[2] == utf8BOM[2] {
		copy(p, p[3:])
		n -= 3
	}
	return n, err
}

func (r *bomSafeReader) Close() error {
	return r.src.Close()
}

// --- nopWriteCloser ---

// nopWriteCloser wraps an io.Writer with a no-op Close.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
