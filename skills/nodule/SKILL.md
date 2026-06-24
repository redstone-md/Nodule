---
name: nodule
description: Use the Nodule MCP `bounce_idea` tool to get an independent, adversarial second opinion from a separate critic LLM before committing to a non-trivial design or fix. Trigger BEFORE applying a risky/irreversible change, when stuck between approaches, after writing concurrency/security/perf-sensitive code, or when the user asks to "bounce", "critique", "sanity-check", "second opinion", or "red-team" an idea. Do NOT use for trivial edits, formatting, or questions answerable from the code directly.
---

# Nodule — independent critic (`bounce_idea`)

Nodule runs a **separate** LLM as a ruthless senior critic. It does not see your repo —
only the `context` and `proposed_solution` you pass. Its value is breaking your local
optimum: catching bugs, race conditions, and weak architecture before they ship.

## When to call

Call `bounce_idea` when at least one is true:
- Change is hard to reverse (schema/migration, public API, data format, auth).
- Concurrency, security, or performance-critical code just written/changed.
- You're torn between 2+ approaches and want them stress-tested.
- A fix "works" but you're not confident it's correct under edge inputs.
- User explicitly asks for a critique / second opinion / red-team.

Do **not** call for: renames, formatting, docs, obvious one-liners, or anything you can
verify by reading the code or running a test. It costs an LLM round-trip — spend it where
wrongness is expensive.

## How to call it well

The critic is blind to everything except your two strings. Garbage in → useless critique.

**`context`** — give it enough to reason, no more:
- The real signatures / types / invariants involved (paste actual code, not prose).
- Constraints that matter: concurrency model, scale (N=?), latency budget, platform.
- What already failed, error logs, benchmark numbers — concrete facts beat description.

**`proposed_solution`** — the actual change you plan to apply (code or precise plan),
not "I'll refactor it nicely". The critic attacks what you write down.

**`focus`** — pick the axis where failure hurts most:
| focus | use when |
|---|---|
| `performance` | hot path, allocations, O(N²), locking, cache behavior |
| `architecture` | module boundaries, coupling, abstraction, extensibility |
| `security` | untrusted input, auth, secrets, injection, unsafe ops |
| `edge_cases` | concurrency, nil/empty/overflow, cancellation, error paths |

One focus per call. If two axes matter, make two calls — sharper than one vague pass.

## Reading the result

Nodule returns Markdown with severity tags: 🔴 critical, 🟡 major, 🟢 minor, and a verdict.

- **Don't blindly obey it.** The critic can be wrong, can hallucinate constraints, or
  miss repo context it never saw. Verify each 🔴/🟡 against the actual code before acting.
- Apply findings that hold up; explicitly reject ones that don't and say why.
- If the critique exposes a real flaw, fix it — then optionally bounce the revised version
  once more. Don't loop endlessly; 1–2 rounds is usually enough.
- Report to the user which findings you accepted vs. dismissed, not a raw dump.

## Example

```
bounce_idea(
  context: "Go HTTP handler, ~5k req/s. cache is map[string][]byte guarded by sync.RWMutex.\nfunc Get(k string) []byte { mu.RLock(); defer mu.RUnlock(); return cache[k] }",
  proposed_solution: "Drop the mutex and use sync.Map to cut lock contention on the read path.",
  focus: "edge_cases"
)
```
Then validate the returned 🔴/🟡 items against the code before changing anything.
