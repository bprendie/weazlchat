package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit, true
	case "ctrl+n":
		if m.mode == modeChat {
			updated, cmd := m.newSession()
			return updated, cmd, true
		}
	case "ctrl+s":
		if m.mode == modeChat {
			m.saveWorkspace()
			return m, nil, true
		}
	case "ctrl+t":
		if m.mode == modeChat && !m.thinking {
			updated, cmd := m.trimContext(false, "", 0, 0)
			return updated, cmd, true
		}
	case "ctrl+u":
		if m.mode == modeChat && !m.thinking {
			updated, cmd := m.startClearContextConfirm()
			return updated, cmd, true
		}
	case "ctrl+m":
		if m.mode == modeChat {
			updated, cmd := m.toggleMouseMode()
			return updated, cmd, true
		}
	case "ctrl+r", "ctrl+w":
		if m.mode == modeChat {
			updated, cmd := m.showWorkspaces()
			return updated, cmd, true
		}
	case "ctrl+d":
		if m.mode == modeSessions {
			updated, cmd := m.deleteSelectedSession()
			return updated, cmd, true
		}
		if m.mode == modeWorkspace {
			updated, cmd := m.deleteSelectedWorkspace()
			return updated, cmd, true
		}
	case "ctrl+e":
		if (m.mode == modeChat && !m.thinking) || m.mode == modeWorkspace {
			updated, cmd := m.startRenameWorkspace()
			return updated, cmd, true
		}
	case "pgup":
		if m.mode == modeChat {
			m.viewport.PageUp()
			return m, nil, true
		}
	case "pgdown":
		if m.mode == modeChat {
			m.viewport.PageDown()
			return m, nil, true
		}
	case "home":
		if m.mode == modeChat {
			m.viewport.GotoTop()
			return m, nil, true
		}
	case "end":
		if m.mode == modeChat {
			m.viewport.GotoBottom()
			return m, nil, true
		}
	case "esc":
		if m.mode == modeSessions || m.mode == modeWorkspace {
			m.mode = modeChat
			m.input.Focus()
			return m, nil, true
		}
		if m.mode == modeRenameWorkspace {
			updated, cmd := m.cancelRenameWorkspace()
			return updated, cmd, true
		}
		if m.mode == modeClearContext {
			updated, cmd := m.cancelClearContext()
			return updated, cmd, true
		}
	case "enter":
		updated, cmd := m.handleEnter()
		return updated, cmd, true
	}
	return m, nil, false
}

func looksLikePaste(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes {
		return false
	}
	return len(msg.Runes) > 16 || strings.ContainsRune(string(msg.Runes), '\n')
}
