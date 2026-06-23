# Contributing to Nodule

## Prerequisites

- Go 1.23+
- A Gemini or OpenAI API key for manual testing

## Development Workflow

1. Fork and clone the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make changes with tests
4. Run the full check suite:
   ```bash
   go test ./...
   go vet ./...
   go build ./cmd/nodule/
   ```
5. Commit with [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat: add new focus mode`
   - `fix: resolve empty context crash`
   - `test: add config validation coverage`
6. Push and open a Pull Request

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep files focused and under 300 lines
- Every exported type and function needs a doc comment
- Use interfaces for cross-boundary dependencies (see `llm.Client`)
- No global mutable state

## Adding a New LLM Provider

1. Implement the `llm.Client` interface in `internal/llm/`
2. Add provider name to `cmd/nodule/main.go` switch
3. Add config validation for the new provider in `internal/config/`
4. Add tests using `internal/llm/llmmock`

## Adding a New Focus Mode

1. Add constant in `internal/prompt/builder.go`
2. Add case to `ParseFocus`
3. Add system prompt and user prompt directives
4. Add test case to `TestBounceIdeaHandler_AllFocusModes`

## Reporting Issues

- Include Go version, OS, and environment variables (redact API keys)
- Attach logs from stderr
