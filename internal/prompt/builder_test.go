package prompt

import (
	"strings"
	"testing"
)

func TestParseFocus(t *testing.T) {
	tests := []struct {
		input string
		want  Focus
	}{
		{"performance", FocusPerformance},
		{"Performance", FocusPerformance},
		{"PERFORMANCE", FocusPerformance},
		{"architecture", FocusArchitecture},
		{"Architecture", FocusArchitecture},
		{"security", FocusSecurity},
		{"Security", FocusSecurity},
		{"edge_cases", FocusEdgeCases},
		{"Edge_Cases", FocusEdgeCases},
		{"edge cases", FocusPerformance}, // space not underscore => default
		{"", FocusPerformance},           // empty => default
		{"unknown", FocusPerformance},    // unknown => default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseFocus(tt.input)
			if got != tt.want {
				t.Errorf("ParseFocus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuilder_Build_SystemPromptContainsBaseDirective(t *testing.T) {
	b := NewBuilder()

	for _, focus := range []Focus{FocusPerformance, FocusArchitecture, FocusSecurity, FocusEdgeCases} {
		pair := b.Build("test-context", "test-solution", focus)
		if !strings.Contains(pair.SystemPrompt, "what will break this solution in production") {
			t.Errorf("SystemPrompt for %s missing base directive", focus)
		}
	}
}

func TestBuilder_Build_SystemPromptHasFocusSpecificDirective(t *testing.T) {
	b := NewBuilder()

	tests := []struct {
		focus       Focus
		mustContain string
	}{
		{FocusPerformance, "alloc"},
		{FocusArchitecture, "encapsulation"},
		{FocusSecurity, "injection"},
		{FocusEdgeCases, "race-condition"},
	}

	for _, tt := range tests {
		t.Run(string(tt.focus), func(t *testing.T) {
			pair := b.Build("ctx", "sol", tt.focus)
			if !strings.Contains(pair.SystemPrompt, tt.mustContain) {
				t.Errorf("SystemPrompt for %s must contain %q\nGot: %s", tt.focus, tt.mustContain, pair.SystemPrompt)
			}
		})
	}
}

func TestBuilder_Build_UserPromptContainsContext(t *testing.T) {
	b := NewBuilder()

	pair := b.Build("my-codebase-state", "my-proposed-fix", FocusPerformance)

	if !strings.Contains(pair.UserPrompt, "my-codebase-state") {
		t.Error("UserPrompt must contain the context")
	}
	if !strings.Contains(pair.UserPrompt, "my-proposed-fix") {
		t.Error("UserPrompt must contain the proposed solution")
	}
	if !strings.Contains(pair.UserPrompt, "Codebase context") {
		t.Error("UserPrompt must contain section header 'Codebase context'")
	}
	if !strings.Contains(pair.UserPrompt, "Proposed solution") {
		t.Error("UserPrompt must contain section header 'Proposed solution'")
	}
}

func TestBuilder_Build_UserPromptHasStructure(t *testing.T) {
	b := NewBuilder()
	pair := b.Build("c", "s", FocusPerformance)

	requiredSections := []string{
		"Critical problems",
		"Hidden bugs",
		"Alternatives",
		"Verdict",
	}

	for _, section := range requiredSections {
		if !strings.Contains(pair.UserPrompt, section) {
			t.Errorf("UserPrompt missing section: %q", section)
		}
	}
}

func TestBuilder_Build_FocusDirectivePerMode(t *testing.T) {
	b := NewBuilder()

	tests := []struct {
		focus       Focus
		mustContain string
	}{
		{FocusPerformance, "asymptotics"},
		{FocusArchitecture, "decomposition"},
		{FocusSecurity, "attack and vulnerab"},
		{FocusEdgeCases, "deadlock"},
	}

	for _, tt := range tests {
		t.Run(string(tt.focus), func(t *testing.T) {
			pair := b.Build("c", "s", tt.focus)
			if !strings.Contains(pair.UserPrompt, tt.mustContain) {
				t.Errorf("UserPrompt for %s must contain %q", tt.focus, tt.mustContain)
			}
		})
	}
}

func TestBuilder_Build_NonEmptyPrompts(t *testing.T) {
	b := NewBuilder()

	for _, focus := range []Focus{FocusPerformance, FocusArchitecture, FocusSecurity, FocusEdgeCases} {
		pair := b.Build("ctx", "sol", focus)
		if pair.SystemPrompt == "" {
			t.Errorf("SystemPrompt empty for focus %s", focus)
		}
		if pair.UserPrompt == "" {
			t.Errorf("UserPrompt empty for focus %s", focus)
		}
	}
}
