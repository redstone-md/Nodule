# Nodule — Install Guide for LLM Agents

You are an AI coding agent. Follow these steps **in order** to install the Nodule MCP
server (the `bounce_idea` independent-critic tool) into the user's environment. Do not
skip verification. Stop and report if any step fails.

Nodule is a local MCP server: it shells out to a critic LLM you configure (BYOM/BYOK) and
exposes one tool, `bounce_idea`. It needs Node.js ≥ 18 and network access on first run.

---

## Step 0 — Gather the 3 required facts from the user

Before writing any config, confirm:

1. **Provider** — `gemini` or `openai` (openai = any OpenAI-compatible endpoint incl.
   Ollama/vLLM/LM Studio/OpenRouter/etc.).
2. **API key** — unless it's a local server (Ollama/vLLM) with no auth.
3. **Model name** — e.g. `gemini-2.5-flash`, `gpt-4.1-mini`, `qwen3:8b`.

If provider is `openai` against a non-OpenAI endpoint, you also need **`NODULE_LLM_BASE_URL`**
(the API root, ending in `/v1`, e.g. `https://api.openai.com/v1`, `http://localhost:11434/v1`).

Do not invent or reuse keys. Ask the user. Never print a full key back in logs.

## Step 1 — Pre-warm the binary (prevents the #1 failure)

`npx` downloads a ~16 MB platform binary on first run; a cold start can time out the MCP
client, and a stale cache from an old version is the most common "won't connect" cause.

```bash
npm cache clean --force            # only if a previous install misbehaved
npx -y @redstone-md/nodule@latest  # wait for "nodule: ready" on stderr, then Ctrl+C
```

## Step 2 — Detect the client and install

### If Claude Code

Prefer the CLI (it edits `~/.claude.json` correctly even while Claude is running):

```bash
claude mcp add nodule --scope user \
  -e NODULE_LLM_PROVIDER=<provider> \
  -e NODULE_API_KEY=<key> \
  -e NODULE_MODEL_NAME=<model> \
  -- npx -y @redstone-md/nodule@latest
```

Add `-e NODULE_LLM_BASE_URL=<url>` if provider is `openai` with a custom endpoint.
Scopes: `--scope user` (all projects) | `--scope project` (shared, writes `.mcp.json`) |
`--scope local` (this project only). Default to `user` unless the user says otherwise.

### If OpenCode

Edit `~/.config/opencode/opencode.json` and add under the `mcp` object. Note: OpenCode
uses a **single `command` array** (no separate `args` field):

```json
"nodule": {
  "type": "local",
  "command": ["npx", "-y", "@redstone-md/nodule@latest"],
  "enabled": true,
  "env": {
    "NODULE_LLM_PROVIDER": "<provider>",
    "NODULE_LLM_BASE_URL": "<url-if-openai-custom>",
    "NODULE_API_KEY": "<key>",
    "NODULE_MODEL_NAME": "<model>"
  }
}
```

Preserve existing JSON. Validate it parses before saving.

### If Cursor / VS Code / other MCP client

Use `command: "npx"`, `args: ["-y", "@redstone-md/nodule@latest"]`, and the same `env`
block. The env var names are identical across all clients.

## Step 3 — Verify the connection

- Claude Code: run `claude mcp list` → expect `nodule … ✔ Connected`.
- Any client: confirm the `bounce_idea` tool appears in the client's tool list.
- Manual smoke test (catches bad key/model fast):

```bash
NODULE_LLM_PROVIDER=<provider> NODULE_API_KEY=<key> NODULE_MODEL_NAME=<model> \
  printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}' \
  | npx -y @redstone-md/nodule@latest
```

Expect a JSON line containing `"serverInfo"`. Server logs (`nodule: provider=…`) go to
**stderr** and are normal — they do not corrupt the JSON-RPC stream on stdout.

## Step 4 — Install the skill (Claude Code only, optional but recommended)

```bash
mkdir -p ~/.claude/skills/nodule
cp skills/nodule/SKILL.md ~/.claude/skills/nodule/SKILL.md
```

It teaches the agent when/how to call `bounce_idea` (write a real `context` +
`proposed_solution`, pick one `focus`, and verify findings before acting).

---

## Environment variable reference

| Variable | Required | Default | Notes |
|---|---|---|---|
| `NODULE_LLM_PROVIDER` | yes | `gemini` | `gemini` \| `openai` |
| `NODULE_MODEL_NAME` | yes | `gemini-2.5-flash` | model id |
| `NODULE_API_KEY` | yes* | — | *optional for keyless local servers; fallbacks: `GEMINI_API_KEY`, `GOOGLE_API_KEY`, `OPENAI_API_KEY` |
| `NODULE_LLM_BASE_URL` | if custom openai | — | API root ending `/v1` |
| `NODULE_TEMPERATURE` | no | `1.2` | 0.0–2.0 |
| `NODULE_MAX_TOKENS` | no | `4096` | > 0 |

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Client shows server failed / not connected | stale npx cache or cold-start timeout | Step 1 pre-warm; `npm cache clean --force`; restart client |
| `NODULE_API_KEY is required for provider "…"` | no key and not a keyless local server | set the key, or set provider=openai + a local `NODULE_LLM_BASE_URL` |
| `openai HTTP 401/403` | wrong/expired key | re-check the key |
| `openai HTTP 404` / no choices | wrong model id or base URL | fix `NODULE_MODEL_NAME` / `NODULE_LLM_BASE_URL` |
| Garbage on stdout breaks JSON-RPC | client emits a UTF-8 BOM | handled automatically since v0.3.0; upgrade with `@latest` |
| `context deadline exceeded` | critic LLM slower than 30 s | use a faster model or smaller `context` |
