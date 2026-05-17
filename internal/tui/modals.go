package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renameWorkspaceView() string {
	w := max(20, m.width-6)
	popupWidth := min(64, max(24, w-4))
	prompt := m.styles.help.Render("shown in picker as <name>: timestamp")
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Width(popupWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(crushPink).
		Background(panel).
		Padding(1, 2).
		Render("Rename workspace\n\n"+m.input.View()+"\n"+prompt))
}

func (m model) clearContextView() string {
	w := max(20, m.width-6)
	popupWidth := min(66, max(30, w-4))
	count := len(m.messages)
	copy := fmt.Sprintf("Clear this session's active context?\n\nThis deletes %d message(s), tool-call history, and context checkpoints from SQLite for the current session.\n\n[ enter yes ]   [ esc no ]", count)
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, lipgloss.NewStyle().
		Width(popupWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(crushPink).
		Background(panel).
		Padding(1, 2).
		Render(copy))
}
