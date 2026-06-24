package mcpserver

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/redstone-md/nodule/internal/llm"
	"github.com/redstone-md/nodule/internal/llm/llmmock"
	"github.com/redstone-md/nodule/internal/prompt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestBounceIdeaHandler_Success verifies the handler calls the LLM and
// assembles a correct markdown result with provider/model/focus header.
func TestBounceIdeaHandler_Success(t *testing.T) {
	mock := llmmock.New("test-provider", &llm.GenerateResponse{
		Content: "CRITICAL: Race condition on shared mutex at line 42",
		Model:   "test-model",
	})

	srv := New(mock, "test-model", 1.0, 1024, "test-nodule", "0.0.0")

	result, _, err := srv.handleBounceIdea(context.Background(), nil, BounceIdeaInput{
		Context:          "func Process(items []int) { for _, i := range items { go handle(i) } }",
		ProposedSolution: "Use sync.WaitGroup to wait for goroutines",
		Focus:            "edge_cases",
	})
	if err != nil {
		t.Fatalf("handleBounceIdea error: %v", err)
	}

	if result.IsError {
		t.Error("IsError should be false on success")
	}
	if len(result.Content) == 0 {
		t.Fatal("Result has no content")
	}

	text := resultText(result)
	if !strings.Contains(text, "Race condition") {
		t.Errorf("Result text missing mock response content; got: %s", truncate(text, 200))
	}
	if !strings.Contains(text, "test-provider") {
		t.Error("Result text missing provider name in header")
	}
	if !strings.Contains(text, "test-model") {
		t.Error("Result text missing model name in header")
	}
	if !strings.Contains(text, "edge_cases") {
		t.Error("Result text missing focus mode in header")
	}

	// Verify the LLM was called with the correct focus mode in system prompt
	lastReq := mock.LastReq()
	if !strings.Contains(lastReq.SystemPrompt, "race-condition") {
		t.Errorf("System prompt should contain edge_cases focus; got: %s", truncate(lastReq.SystemPrompt, 100))
	}
	if !strings.Contains(lastReq.UserPrompt, "WaitGroup") {
		t.Errorf("User prompt should contain the proposed solution; got: %s", truncate(lastReq.UserPrompt, 100))
	}
}

// TestBounceIdeaHandler_DefaultFocus verifies that omitting focus defaults to "performance".
func TestBounceIdeaHandler_DefaultFocus(t *testing.T) {
	mock := llmmock.New("test-provider", &llm.GenerateResponse{
		Content: "O(N²) detected",
		Model:   "test-model",
	})

	srv := New(mock, "test-model", 1.0, 1024, "test-nodule", "0.0.0")

	_, _, _ = srv.handleBounceIdea(context.Background(), nil, BounceIdeaInput{
		Context:          "code here",
		ProposedSolution: "solution here",
		Focus:            "", // empty — should default to performance
	})

	lastReq := mock.LastReq()
	if !strings.Contains(lastReq.SystemPrompt, "алло") {
		t.Errorf("Default focus should be performance; system prompt: %s", truncate(lastReq.SystemPrompt, 100))
	}
}

// TestBounceIdeaHandler_AllFocusModes verifies every focus mode produces
// a system prompt with mode-specific directives.
func TestBounceIdeaHandler_AllFocusModes(t *testing.T) {
	focusChecks := map[string]string{
		"performance":  "алло",
		"architecture": "инкапсуля",
		"security":     "injection",
		"edge_cases":   "race-condition",
	}

	for focus, mustContain := range focusChecks {
		t.Run(focus, func(t *testing.T) {
			mock := llmmock.New("test", &llm.GenerateResponse{Content: "ok", Model: "m"})
			srv := New(mock, "m", 1.0, 1024, "test", "0.0.0")

			_, _, _ = srv.handleBounceIdea(context.Background(), nil, BounceIdeaInput{
				Context:          "c",
				ProposedSolution: "s",
				Focus:            focus,
			})

			lastReq := mock.LastReq()
			if !strings.Contains(lastReq.SystemPrompt, mustContain) {
				t.Errorf("Focus %q: system prompt missing %q", focus, mustContain)
			}
		})
	}
}

// TestBounceIdeaHandler_LLMError verifies graceful error handling
// when the LLM client returns an error.
func TestBounceIdeaHandler_LLMError(t *testing.T) {
	mock := llmmock.NewWithErr("test-provider", context.DeadlineExceeded)

	srv := New(mock, "test-model", 1.0, 1024, "test-nodule", "0.0.0")

	result, _, err := srv.handleBounceIdea(context.Background(), nil, BounceIdeaInput{
		Context:          "c",
		ProposedSolution: "s",
		Focus:            "performance",
	})
	if err != nil {
		t.Fatalf("handleBounceIdea returned error (should be in result): %v", err)
	}

	if !result.IsError {
		t.Error("IsError should be true when LLM fails")
	}

	text := resultText(result)
	if !strings.Contains(text, "failed") {
		t.Errorf("Error result should contain failure message; got: %s", text)
	}
}

// TestBounceIdeaHandler_TemperatureAndTokensPassed verifies that the handler
// passes the configured temperature and max tokens to the LLM client.
func TestBounceIdeaHandler_TemperatureAndTokensPassed(t *testing.T) {
	mock := llmmock.New("test", &llm.GenerateResponse{Content: "ok", Model: "m"})

	srv := New(mock, "m", 1.3, 2048, "test", "0.0.0")

	_, _, _ = srv.handleBounceIdea(context.Background(), nil, BounceIdeaInput{
		Context:          "c",
		ProposedSolution: "s",
	})

	lastReq := mock.LastReq()
	if lastReq.Temperature != 1.3 {
		t.Errorf("Temperature = %.2f, want 1.3", lastReq.Temperature)
	}
	if lastReq.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, want 2048", lastReq.MaxTokens)
	}
	if lastReq.ModelName != "m" {
		t.Errorf("ModelName = %q, want %q", lastReq.ModelName, "m")
	}
}

// TestParseFocusAcceptance verifies ParseFocus is correctly wired
// via a cross-package sanity check.
func TestParseFocusAcceptance(t *testing.T) {
	f := prompt.ParseFocus("security")
	if f != prompt.FocusSecurity {
		t.Errorf("ParseFocus(security) = %q, want %q", f, prompt.FocusSecurity)
	}
}

// helpers

func resultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// --- BOM-safe reader tests ---

// fakeCloser wraps a byte slice as an io.ReadCloser.
type fakeCloser struct {
	data []byte
	pos  int
}

func (f *fakeCloser) Read(p []byte) (int, error) {
	if f.pos >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.pos:])
	f.pos += n
	return n, nil
}

func (f *fakeCloser) Close() error { return nil }

func TestBOMSafeReader_StripsSingleBOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	src := &fakeCloser{data: append(bom, []byte(`{"jsonrpc":"2.0","id":1}`)...)}
	r := newBOMSafeReader(src)

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}
	got := string(buf[:n])
	if got != `{"jsonrpc":"2.0","id":1}` {
		t.Errorf("Expected BOM to be stripped, got: %q", got)
	}
}

func TestBOMSafeReader_StripsMultipleBOMs(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	// PowerShell StreamWriter prepends BOM on every write — 3 writes = 3 BOMs.
	src := &fakeCloser{data: append(append(append(bom, bom...), bom...), []byte(`{"jsonrpc":"2.0"}`)...)}
	r := newBOMSafeReader(src)

	all := []byte{}
	buf := make([]byte, 16)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			all = append(all, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}
	got := string(all)
	if got != `{"jsonrpc":"2.0"}` {
		t.Errorf("Expected all 3 BOMs to be stripped, got: %q", got)
	}
}

func TestBOMSafeReader_NoBOMUnchanged(t *testing.T) {
	src := &fakeCloser{data: []byte(`{"jsonrpc":"2.0","id":1}`)}
	r := newBOMSafeReader(src)

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}
	got := string(buf[:n])
	if got != `{"jsonrpc":"2.0","id":1}` {
		t.Errorf("Expected clean input to pass through, got: %q", got)
	}
}

func TestBOMSafeReader_EmptyInput(t *testing.T) {
	src := &fakeCloser{data: nil}
	r := newBOMSafeReader(src)

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if n != 0 || err != io.EOF {
		t.Errorf("Expected EOF with 0 bytes, got n=%d err=%v", n, err)
	}
}

func TestBOMSafeReader_Close(t *testing.T) {
	src := &fakeCloser{data: []byte(`{}`)}
	r := newBOMSafeReader(src)
	if err := r.Close(); err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}
