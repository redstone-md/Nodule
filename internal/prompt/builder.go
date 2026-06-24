// Package prompt constructs system and user prompts for the critic LLM,
// specialized by the focus mode (performance, architecture, security, edge_cases).
package prompt

import "strings"

// Focus defines the critique specialization mode.
type Focus string

const (
	FocusPerformance Focus = "performance"
	FocusArchitecture Focus = "architecture"
	FocusSecurity    Focus = "security"
	FocusEdgeCases   Focus = "edge_cases"
)

// ParseFocus converts a raw string to a Focus, defaulting to FocusPerformance.
func ParseFocus(raw string) Focus {
	switch strings.ToLower(raw) {
	case "architecture":
		return FocusArchitecture
	case "security":
		return FocusSecurity
	case "edge_cases":
		return FocusEdgeCases
	default:
		return FocusPerformance
	}
}

// Builder constructs the full prompt pair (system + user) for the critic LLM.
type Builder struct{}

// NewBuilder creates a prompt Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// PromptPair holds the assembled system and user prompts.
type PromptPair struct {
	SystemPrompt string
	UserPrompt   string
}

// Build assembles prompts based on the provided context, proposed solution, and focus mode.
func (b *Builder) Build(context, proposedSolution string, focus Focus) PromptPair {
	return PromptPair{
		SystemPrompt: b.systemPrompt(focus),
		UserPrompt:   b.userPrompt(context, proposedSolution, focus),
	}
}

func (b *Builder) systemPrompt(focus Focus) string {
	base := `You are a Principal engineer and red-team architect with 20 years in low-level, concurrent, and distributed systems. ` +
		`Your only job: find what will break this solution in production. Do not praise, do not agree to be polite, do not restate the code.

` +
		`Hard rules:
` +
		`- Report only REAL defects derivable from the code's logic. Never invent problems to pad the output. If the solution is correct, say so in one line and stop.
` +
		`- For each problem state: what exactly breaks, under WHICH concrete input or scenario, and why. No filler, no generic advice.
` +
		`- Tag every finding with severity: 🔴 critical (prod crash / data loss / vulnerability), 🟡 major (bug on edge input), 🟢 minor (smell / risk / style).
` +
		`- Give a concrete fix (code or exact technique) for each problem, never "consider thinking about it".
` +
		`- If the context is insufficient to conclude, name the missing fact instead of guessing.
` +
		`- Format: dense Markdown with headers. Zero preamble, zero pleasantries.`

	switch focus {
	case FocusPerformance:
		return base + ` Focus: performance. Hunt hidden allocations, inefficient algorithmic passes (O(N^2)->O(N)->O(1)), redundant copies, cache-miss patterns, excessive locking. Propose concrete micro-optimizations and alternative data structures.`
	case FocusArchitecture:
		return base + ` Focus: architecture. Hunt encapsulation breaks, leaky abstractions, cyclic dependencies, god-objects, wrong separation of concerns. Propose alternative decompositions and patterns.`
	case FocusSecurity:
		return base + ` Focus: security. Hunt injection vulnerabilities, unsafe operations, missing input validation, private data leaking into logs/errors, least-privilege violations. Propose concrete countermeasures.`
	case FocusEdgeCases:
		return base + ` Focus: edge cases. Hunt race-conditions, deadlocks, nil/empty/zero-input handling, overflow/underflow, incorrect context cancellation, concurrent access to shared state.`
	default:
		return base + ` Focus: comprehensive review. Cover performance, architecture, security, and edge cases.`
	}
}

func (b *Builder) userPrompt(context, proposedSolution string, focus Focus) string {
	var sb strings.Builder

	sb.WriteString("## Codebase context\n\n")
	sb.WriteString(context)
	sb.WriteString("\n\n## Proposed solution\n\n")
	sb.WriteString(proposedSolution)
	sb.WriteString("\n\n## Task\n\n")
	sb.WriteString(formatFocusDirective(focus))
	sb.WriteString("\n\nResponse structure:\n")
	sb.WriteString("1. **Critical problems** — what will break in production\n")
	sb.WriteString("2. **Hidden bugs** — race conditions, leaks, edge cases\n")
	sb.WriteString("3. **Alternatives** — concrete out-of-the-box approaches\n")
	sb.WriteString("4. **Verdict** — accept / revise / rewrite\n")

	return sb.String()
}

func formatFocusDirective(focus Focus) string {
	switch focus {
	case FocusPerformance:
		return "Deconstruct the solution for performance. Find every needless allocation and suboptimal pass. Compare asymptotics."
	case FocusArchitecture:
		return "Deconstruct the architecture. Find leaky abstractions and encapsulation breaks. Propose an alternative decomposition."
	case FocusSecurity:
		return "Deconstruct the solution for security. Find every possible attack and vulnerability."
	case FocusEdgeCases:
		return "Find every possible edge case, race condition, deadlock, panic scenario. What if the context is cancelled? What if input is nil/empty/overflow?"
	default:
		return "Run a comprehensive critique across all axes: performance, architecture, security, edge cases."
	}
}
