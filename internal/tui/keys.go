package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+n":
		if m.mode == modeChat {
			return m.newSession()
		}
	case "ctrl+s":
		if m.mode == modeChat {
			m.saveWorkspace()
		}
	case "ctrl+t":
		if m.mode == modeChat && !m.thinking {
			return m.trimContext(false, "", 0, 0)
		}
	case "ctrl+u":
		if m.mode == modeChat && !m.thinking {
			return m.startClearContextConfirm()
		}
	case "ctrl+m":
		if m.mode == modeChat {
			return m.toggleMouseMode()
		}
	case "ctrl+r", "ctrl+w":
		if m.mode == modeChat {
			return m.showWorkspaces()
		}
	case "ctrl+d":
		if m.mode == modeSessions {
			return m.deleteSelectedSession()
		}
		if m.mode == modeWorkspace {
			return m.deleteSelectedWorkspace()
		}
	case "ctrl+e":
		if (m.mode == modeChat && !m.thinking) || m.mode == modeWorkspace {
			return m.startRenameWorkspace()
		}
	case "pgup":
		if m.mode == modeChat {
			m.viewport.PageUp()
			return m, nil
		}
	case "pgdown":
		if m.mode == modeChat {
			m.viewport.PageDown()
			return m, nil
		}
	case "home":
		if m.mode == modeChat {
			m.viewport.GotoTop()
			return m, nil
		}
	case "end":
		if m.mode == modeChat {
			m.viewport.GotoBottom()
			return m, nil
		}
	case "esc":
		if m.mode == modeSessions || m.mode == modeWorkspace {
			m.mode = modeChat
			m.input.Focus()
		}
		if m.mode == modeRenameWorkspace {
			return m.cancelRenameWorkspace()
		}
		if m.mode == modeClearContext {
			return m.cancelClearContext()
		}
	case "enter":
		return m.handleEnter()
	}
	return m, nil
}

func looksLikePaste(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes {
		return false
	}
	return len(msg.Runes) > 16 || strings.ContainsRune(string(msg.Runes), '\n')
}
