package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// thinkingView returns the spinner view with appropriate status text
func (m model) thinkingView() string {
	if m.trimming {
		return fmt.Sprintf("%s compacting_context_checkpoint", m.working.View())
	}
	return fmt.Sprintf("%s %s", m.working.View(), m.thinkingPhrase())
}

// thinkingPhrase returns a cyberpunk-themed status phrase that changes slowly during generation
func (m model) thinkingPhrase() string {
	if len(modelThinkingPhrases) == 0 || m.streamAt.IsZero() {
		return "model_is_thinking"
	}
	phase := min(2, int(time.Since(m.streamAt)/(20*time.Second)))
	start := int((m.streamAt.UnixNano() / int64(time.Millisecond)) % int64(len(modelThinkingPhrases)))
	idx := (start + phase) % len(modelThinkingPhrases)
	return modelThinkingPhrases[idx]
}

// metricsView returns the formatted metrics bar showing context usage and token counts
func (m model) metricsView() string {
	totalIn := m.session.InputTokens + m.reqIn
	totalOut := m.session.OutputTokens + m.reqOut
	tps := 0.0
	if m.thinking && !m.streamAt.IsZero() {
		elapsed := time.Since(m.streamAt).Seconds()
		if elapsed > 0 {
			tps = float64(m.reqOut) / elapsed
		}
	}
	budget := m.contextBudget()
	contextTokens := m.contextTokenEstimate()
	pct := min(1.0, float64(contextTokens)/float64(budget))
	text := fmt.Sprintf("ctx %s %d/%d  in %d  out %d  %.1f t/s", m.contextBar.ViewAs(pct), contextTokens, budget, totalIn, totalOut, tps)
	width := max(20, m.width-6)
	if len(text) < width {
		text = strings.Repeat(" ", width-len(text)) + text
	}
	return m.styles.help.Render(text)
}

// helpText returns context-appropriate help text for the current mode
func (m model) helpText() string {
	if m.mode == modeLoading {
		return "loading previous session | ctrl+c quit"
	}
	if m.mode == modeRenameWorkspace {
		return "enter save rename | esc cancel | ctrl+c quit"
	}
	if m.mode == modeClearContext {
		return "enter clear context | esc cancel | ctrl+c quit"
	}
	if m.mode == modeSessions {
		return "enter resume | ctrl+d delete session | esc back | ctrl+c quit"
	}
	if m.mode == modeWorkspace {
		return "enter replay | ctrl+e rename | ctrl+d delete | esc back | ctrl+c quit"
	}
	mouseHelp := "ctrl+m copy"
	if !m.mouseScroll {
		mouseHelp = "ctrl+m mouse"
	}
	renameHelp := ""
	if m.activeWorkspaceID != 0 {
		renameHelp = " | ctrl+e rename"
	}
	return "enter send/select | wheel/pgup/pgdn scroll | " + mouseHelp + " | ctrl+n new | ctrl+t trim | ctrl+u clear | ctrl+r workspaces | ctrl+s save" + renameHelp + " | ctrl+c quit"
}

// inputView returns the input field view with paste indicator if applicable
func (m model) inputView() string {
	if m.pasteText == "" {
		return m.input.View()
	}
	prefix := strings.TrimSpace(m.input.Value())
	badge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0D0D12")).
		Background(lipgloss.Color("#F3E600")).
		Bold(true).
		Padding(0, 1).
		Render(fmt.Sprintf("[PASTED %d lines]", m.pasteLines))
	if prefix == "" {
		return badge
	}
	return m.input.View() + " " + badge
}

// resize updates component dimensions based on terminal size
func (m *model) resize() {
	w := max(20, m.width-6)
	h := max(5, m.height-16)
	m.viewport.Width = w
	m.viewport.Height = h
	m.sessions.SetSize(w, h+4)
	m.workspaces.SetSize(w, h+4)
	m.contextBar.Width = min(28, max(10, w/4))
	m.markdown.Resize(w)
}

func (m *model) renderContent(role, content string) string {
	if role == "assistant" {
		return m.markdown.Render(content, m.viewport.Width)
	}
	return wrapText(content, m.viewport.Width)
}
