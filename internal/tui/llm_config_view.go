package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) llmConfigView() string {
	width := max(28, m.width-10)
	switch m.mode {
	case modeLLMProvider:
		return m.llmPanel("LLM Provider", m.providerChoicesView(), width)
	case modeLLMServer:
		body := lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.help.Render(serverHint(m.llmDraft.ProviderType)),
			"",
			m.input.View(),
		)
		return m.llmPanel("LLM Server", body, width)
	case modeLLMLoading:
		body := fmt.Sprintf("%s fetching %s models from %s", m.working.View(), m.llmDraft.ProviderType, m.llmDraft.ServerURL)
		return m.llmPanel("LLM Models", body, width)
	case modeLLMModel:
		return m.llmPanel("LLM Model", m.modelChoicesView(), width)
	case modeLLMContext:
		return m.llmPanel("Context Window", m.contextChoicesView(), width)
	default:
		return ""
	}
}

func (m model) llmPanel(title, body string, width int) string {
	current := m.cfg.Active()
	summary := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.styles.statusLabel.Render("current"),
		" ",
		m.styles.statusValue.Render(fmt.Sprintf("%s / %s / %d", current.Type, current.Model, current.ContextWindow)),
	)
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.roleSystem.Render(title),
		"",
		body,
		"",
		summary,
	)
	return m.styles.panel.Width(width).Render(content)
}

func (m model) providerChoicesView() string {
	choices := []string{"vllm", "ollama"}
	rows := make([]string, 0, len(choices))
	for i, choice := range choices {
		rows = append(rows, m.selectRow(i == m.llmDraft.ProviderIndex, fmt.Sprintf("%d", i+1), choice, providerDescription(choice)))
	}
	return strings.Join(rows, "\n")
}

func (m model) modelChoicesView() string {
	if m.llmDraft.FetchErr != "" || len(m.llmDraft.Models) == 0 {
		body := []string{
			m.styles.statusWarn.Render(m.llmDraft.FetchErr),
			"",
			m.styles.help.Render("Type the model name exactly as your local server expects."),
			"",
			m.input.View(),
		}
		return strings.Join(body, "\n")
	}
	rows := make([]string, 0, len(m.llmDraft.Models))
	for i, model := range m.llmDraft.Models {
		rows = append(rows, m.selectRow(i == m.llmDraft.ModelIndex, fmt.Sprintf("%d", i+1), model, ""))
	}
	return strings.Join(rows, "\n")
}

func (m model) contextChoicesView() string {
	rows := make([]string, 0, len(contextWindowChoices))
	for i, choice := range contextWindowChoices {
		desc := fmt.Sprintf("%d tokens", choice.tokens)
		if choice.note != "" {
			desc += " - " + choice.note
		}
		rows = append(rows, m.selectRow(i == m.llmDraft.ContextIndex, fmt.Sprintf("%d", i+1), choice.name, desc))
	}
	return strings.Join(rows, "\n")
}

func (m model) selectRow(selected bool, key, label, desc string) string {
	marker := " "
	style := m.styles.help
	labelStyle := m.styles.statusValue
	if selected {
		marker = ">"
		style = m.styles.status
		labelStyle = m.styles.statusWarn
	}
	left := lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render(marker),
		" ",
		m.styles.helpKey.Render(key),
		" ",
		labelStyle.Render(label),
	)
	if desc == "" {
		return left
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", m.styles.help.Render(desc))
}

func providerDescription(providerType string) string {
	if providerType == "ollama" {
		return "local Ollama /api/chat"
	}
	return "OpenAI-compatible local /v1/chat/completions"
}

func serverHint(providerType string) string {
	if providerType == "ollama" {
		return "Base URL only, without /api. Example: http://localhost:11434"
	}
	return "Base URL only, without /v1. Example: http://localhost:8000"
}
