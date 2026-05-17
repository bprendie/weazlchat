package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		if m.mode == modeChat && !m.thinking {
			m.renderMessages()
		}
	case tea.MouseMsg:
		if m.mode == modeChat {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	case tea.KeyMsg:
		if m.mode == modeChat && !m.thinking {
			if updated, cmd, handled := m.handleChatKey(msg); handled {
				return updated, cmd
			}
		}
		if updated, cmd, handled := m.handleGlobalKey(msg); handled {
			return updated, cmd
		}
	case streamEvent:
		return m.handleStreamEvent(msg)
	case contextTrimMsg:
		return m.handleContextTrimMsg(msg)
	case previousSessionMsg:
		return m.handlePreviousSessionMsg(msg)
	case spinner.TickMsg:
		if m.thinking || m.mode == modeLoading {
			var cmd tea.Cmd
			m.working, cmd = m.working.Update(msg)
			if m.thinking {
				m.renderMessages()
			}
			return m, cmd
		}
	}

	switch m.mode {
	case modeSessions:
		var cmd tea.Cmd
		m.sessions, cmd = m.sessions.Update(msg)
		cmds = append(cmds, cmd)
	case modeWorkspace:
		var cmd tea.Cmd
		m.workspaces, cmd = m.workspaces.Update(msg)
		cmds = append(cmds, cmd)
	case modeRenameWorkspace:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	case modeClearContext:
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}
