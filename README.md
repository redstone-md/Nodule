# Nodule

A local MCP (Model Context Protocol) server that provides a **bounce_idea** tool — an independent second opinion for your coding agent. It sends your current solution to a configurable LLM critic for architectural critique, bug hunting, and alternative approaches.

## How It Works

```
[Your Agent] ──bounce_idea──> [Nodule MCP Server] ──prompt──> [Your LLM]
                                     │                              │
                                     └───── critique ◄──────────────┘
```

Nodule implements a **Double-Layer Reasoning** pattern: your primary agent does the implementation, while an independently configured LLM plays the role of a ruthless senior critic. This breaks local optima and catches issues before they reach production.

## BYOM / BYOK

Nodule is fully local. You choose the critic model and provide your own API key:

- **Gemini** (default) — fast, long-context, cheap
- **OpenAI** — GPT-4.1-mini, GPT-5, etc.
- **Ollama / vLLM / LM Studio** — any local model with an OpenAI-compatible `/v1/chat/completions` endpoint

## Quick Start

### Go Install

```bash
go install github.com/redstone-md/nodule/cmd/nodule@latest
```

### npm

```bash
npx @redstone-md/nodule
```

### Configuration (Environment Variables)

| Variable | Default | Description |
|---|---|---|
| `NODULE_LLM_PROVIDER` | `gemini` | LLM provider: `gemini` or `openai` |
| `NODULE_MODEL_NAME` | `gemini-2.5-flash` | Model identifier |
| `NODULE_API_KEY` | — | API key (fallbacks: `GEMINI_API_KEY`, `GOOGLE_API_KEY`, `OPENAI_API_KEY`) |
| `NODULE_LLM_BASE_URL` | — | Base URL for OpenAI-compatible servers (e.g. `http://localhost:11434/v1`) |
| `NODULE_TEMPERATURE` | `1.2` | Critic temperature (higher = more creative) |
| `NODULE_MAX_TOKENS` | `4096` | Maximum output tokens |
| `NODULE_SERVER_NAME` | `nodule` | MCP server name |
| `NODULE_SERVER_VERSION` | `0.4.1` | MCP server version |

### Examples

**Gemini (default):**
```bash
export NODULE_API_KEY="your-gemini-key"
export NODULE_MODEL_NAME="gemini-2.5-flash"
npx @redstone-md/nodule
```

**OpenAI:**
```bash
export NODULE_LLM_PROVIDER="openai"
export NODULE_API_KEY="sk-..."
export NODULE_MODEL_NAME="gpt-4.1-mini"
npx @redstone-md/nodule
```

**Local Ollama (no API key needed):**
```bash
export NODULE_LLM_PROVIDER="openai"
export NODULE_LLM_BASE_URL="http://localhost:11434/v1"
export NODULE_MODEL_NAME="qwen3:8b"
npx @redstone-md/nodule
```

## MCP Tool: `bounce_idea`

| Parameter | Type | Required | Description |
|---|---|---|---|
| `context` | string | Yes | Current codebase state, function signatures, benchmarks, or error logs |
| `proposed_solution` | string | Yes | Draft idea or code the agent plans to apply |
| `focus` | string | No | Critique specialization: `performance`, `architecture`, `security`, `edge_cases` (default: `performance`) |

### Focus Modes

- **`performance`** — Hidden allocations, O(N^2) paths, cache misses, lock contention
- **`architecture`** — Encapsulation leaks, cyclic dependencies, god-objects, SRP violations
- **`security`** — Injection vulnerabilities, unsafe operations, input validation gaps
- **`edge_cases`** — Race conditions, deadlocks, nil/empty inputs, context cancellation

## MCP Client Configuration

> **For AI agents:** a step-by-step, copy-paste install guide written for an LLM to
> follow lives in [`docs/INSTALL_FOR_LLM.md`](docs/INSTALL_FOR_LLM.md).

### Claude Code

Fastest — one command (user scope, available in every project):

```bash
claude mcp add nodule --scope user \
  -e NODULE_LLM_PROVIDER=gemini \
  -e NODULE_API_KEY=your-key \
  -e NODULE_MODEL_NAME=gemini-2.5-flash \
  -- npx -y @redstone-md/nodule@latest
```

Verify: `claude mcp list` should show `nodule … ✔ Connected`.

Or edit `~/.claude.json` (top-level `mcpServers`) by hand:

```json
{
  "mcpServers": {
    "nodule": {
      "command": "npx",
      "args": ["-y", "@redstone-md/nodule@latest"],
      "env": {
        "NODULE_LLM_PROVIDER": "gemini",
        "NODULE_API_KEY": "your-key",
        "NODULE_MODEL_NAME": "gemini-2.5-flash"
      }
    }
  }
}
```

### OpenCode (`~/.config/opencode/opencode.json`)

OpenCode uses a single `command` array, not `command` + `args`:

```json
{
  "mcp": {
    "nodule": {
      "type": "local",
      "command": ["npx", "-y", "@redstone-md/nodule@latest"],
      "enabled": true,
      "env": {
        "NODULE_LLM_PROVIDER": "openai",
        "NODULE_LLM_BASE_URL": "https://api.openai.com/v1",
        "NODULE_API_KEY": "sk-...",
        "NODULE_MODEL_NAME": "gpt-4.1-mini"
      }
    }
  }
}
```

### Cursor / VS Code

```json
{
  "mcp.servers": {
    "nodule": {
      "command": "npx",
      "args": ["-y", "@redstone-md/nodule@latest"],
      "env": {
        "NODULE_LLM_PROVIDER": "openai",
        "NODULE_API_KEY": "your-key",
        "NODULE_MODEL_NAME": "gpt-4.1-mini"
      }
    }
  }
}
```

### First-run note (read this if it "doesn't connect")

`npx` downloads a ~16 MB platform binary on first launch. A cold start can exceed an MCP
client's startup timeout, and a **stale npx cache** from an older version is the most
common failure. Pre-warm once in a terminal, then (re)start your client:

```bash
npm cache clean --force            # only if a previous version misbehaved
npx -y @redstone-md/nodule@latest  # wait for "nodule: ready", then Ctrl+C
```

## Skill (Claude Code)

A bundled skill teaches the agent *when* and *how* to call `bounce_idea` well. Install:

```bash
mkdir -p ~/.claude/skills/nodule
cp skills/nodule/SKILL.md ~/.claude/skills/nodule/SKILL.md
```

## Development

```bash
# Build
go build ./cmd/nodule/

# Test
go test ./...

# Vet
go vet ./...
```

## Architecture

```
cmd/nodule/main.go          # Entry point: config → provider → server
internal/
  config/config.go          # Env-based BYOM/BYOK configuration
  llm/
    client.go               # llm.Client interface (Bring Your Own Model)
    gemini.go               # Gemini provider (google.golang.org/genai)
    openai.go               # OpenAI-compatible provider (Ollama, vLLM, etc.)
    llmmock/mock.go         # Test double for llm.Client
  prompt/builder.go         # System/user prompt assembly by focus mode
  mcpserver/server.go       # MCP server + bounce_idea tool handler
```

## License

MIT
