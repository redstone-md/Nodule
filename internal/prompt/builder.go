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
	base := `Ты — Senior R&D системный инженер и низкоуровневый архитектор. Твоя единственная цель — деконструировать и сломать предложенное решение. ` +
		`Никакой вежливости и вводных фраз — сразу к техническим инсайтам. ` +
		`Будь максимально лаконичен. Формат: Markdown с заголовками проблем.`

	switch focus {
	case FocusPerformance:
		return base + ` Фокус: производительность. Ищи скрытые аллокации, неэффективные алгоритмические проходы (O(N²)→O(N)→O(1)), лишние копирования, cache-miss паттерны, избыточные блокировки. Предлагай конкретные микрооптимизации и альтернативные структуры данных.`
	case FocusArchitecture:
		return base + ` Фокус: архитектура. Ищи нарушения инкапсуляции, утечки абстракций, циклические зависимости, god-objects, неправильное разделение ответственности. Предлагай альтернативные декомпозиции и паттерны.`
	case FocusSecurity:
		return base + ` Фокус: безопасность. Ищи injection-уязвимости, unsafe-операции, некорректную валидацию входных данных, утечки приватных данных в логи/ошибки, нарушение least-privilege. Предлагай конкретные контрмеры.`
	case FocusEdgeCases:
		return base + ` Фокус: edge-cases. Ищи race-conditions, deadlocks, проблемы при нулевых/пустых/nil-входах, overflow/underflow, некорректную обработку отмены контекста (context cancellation), проблемы при конкурентном доступе к shared state.`
	default:
		return base + ` Фокус: комплексный анализ. Рассмотри производительность, архитектуру, безопасность и edge-cases.`
	}
}

func (b *Builder) userPrompt(context, proposedSolution string, focus Focus) string {
	var sb strings.Builder

	sb.WriteString("## Контекст кодовой базы\n\n")
	sb.WriteString(context)
	sb.WriteString("\n\n## Предложенное решение\n\n")
	sb.WriteString(proposedSolution)
	sb.WriteString("\n\n## Задача\n\n")
	sb.WriteString(formatFocusDirective(focus))
	sb.WriteString("\n\nСтруктура ответа:\n")
	sb.WriteString("1. **Критические проблемы** — то, что сломает код в проде\n")
	sb.WriteString("2. **Скрытые баги** — race conditions, утечки, edge-cases\n")
	sb.WriteString("3. **Альтернативы** — конкретные out-of-the-box подходы\n")
	sb.WriteString("4. **Вердикт** — принять / доработать / переписать\n")

	return sb.String()
}

func formatFocusDirective(focus Focus) string {
	switch focus {
	case FocusPerformance:
		return "Деконструируй решение с точки зрения производительности. Найди каждый лишний аллокацию и неоптимальный проход. Сравни асимптотику."
	case FocusArchitecture:
		return "Деконструируй архитектурное решение. Найди утечки абстракций и нарушения инкапсуляции. Предложи альтернативную декомпозицию."
	case FocusSecurity:
		return "Деконструируй решение с точки зрения безопасности. Найди каждую возможную атаку и уязвимость."
	case FocusEdgeCases:
		return "Найди каждый возможный edge-case, race condition, deadlock, panic-scenario. Подумай: что если контекст отменят? Что если вход nil/empty/overflow?"
	default:
		return "Проведи комплексную критику решения по всем осям: производительность, архитектура, безопасность, edge-cases."
	}
}
