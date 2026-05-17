package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
		return m.handleGlobalKey(msg)
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

func (m model) handleStreamEvent(msg streamEvent) (tea.Model, tea.Cmd) {
	if msg.eventType == "content" && msg.chunk != "" {
		m.streamText += msg.chunk
		m.reqOut = estimateTokens(m.streamText)
		m.renderMessages()
	}
	if msg.eventType == "tool_call" && len(msg.toolCalls) > 0 {
		m.pendingTools = msg.toolCalls
		m.status = fmt.Sprintf("executing %d tool(s)", len(msg.toolCalls))
		m.renderMessages()
	}
	if !msg.done {
		return m, waitStream(m.stream)
	}
	m.thinking = false
	m.stream = nil
	if msg.err != nil {
		m.err = msg.err.Error()
		m.status = "request failed"
		m.renderMessages()
		return m, nil
	}
	inputTokens := msg.usage.InputTokens
	outputTokens := msg.usage.OutputTokens
	if inputTokens == 0 {
		inputTokens = m.reqIn
	}
	if outputTokens == 0 {
		outputTokens = max(1, estimateTokens(m.streamText))
	}

	if len(m.pendingTools) > 0 && m.cfg.Tools.Enabled {
		return m.executeTools(inputTokens, outputTokens)
	}

	toolCallsJSON := ""
	if len(m.pendingTools) > 0 {
		b, _ := json.Marshal(m.pendingTools)
		toolCallsJSON = string(b)
	}
	if err := m.store.AddMessageWithTools(m.session.ID, "assistant", strings.TrimSpace(m.streamText), toolCallsJSON, ""); err != nil {
		m.err = err.Error()
		return m, nil
	}
	if err := m.store.AddSessionTokens(m.session.ID, inputTokens, outputTokens); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.session.InputTokens += inputTokens
	m.session.OutputTokens += outputTokens
	m.messages, _ = m.store.Messages(m.session.ID)
	m.streamText = ""
	m.pendingTools = nil
	m.toolResults = nil
	m.reqIn = 0
	m.reqOut = 0
	m.renderMessages()
	m.status = "ready"
	return m, nil
}

func (m model) handleContextTrimMsg(msg contextTrimMsg) (tea.Model, tea.Cmd) {
	m.trimming = false
	m.thinking = false
	if msg.err != nil {
		m.err = msg.err.Error()
		m.status = "context trim failed"
		m.renderMessages()
		return m, nil
	}
	m.err = ""
	m.refreshCheckpoint()
	if msg.auto {
		m.status = "streaming"
		m.thinking = true
		m.working.Spinner = spinner.Jump
		m.streamText = ""
		m.streamAt = time.Now()
		m.reqIn = m.contextTokenEstimate()
		m.reqOut = 0
		ch := make(chan streamEvent, 64)
		m.stream = ch
		history, err := m.contextHistoryForPrompt(msg.currentPromptID)
		if err != nil {
			m.thinking = false
			m.err = err.Error()
			m.status = "request failed"
			return m, nil
		}
		return m, tea.Batch(m.startStream(ch, msg.prompt, history), waitStream(ch), m.working.Tick)
	}
	m.status = fmt.Sprintf("context trimmed through message %d", msg.throughID)
	m.renderMessages()
	return m, nil
}

func (m model) handlePreviousSessionMsg(msg previousSessionMsg) (tea.Model, tea.Cmd) {
	m.thinking = false
	if msg.err != nil {
		m.err = msg.err.Error()
		return m.newSession()
	}
	if msg.session.ID == "" {
		return m.newSession()
	}
	m.session = msg.session
	m.messages = msg.messages
	m.checkpoint = msg.checkpoint
	m.hasCheckpoint = msg.hasCheckpoint
	m.activeWorkspaceID = 0
	m.activeWorkspaceName = ""
	m.activeWorkspaceAt = time.Time{}
	m.historyIdx = 0
	m.historyDraft = ""
	m.mode = modeChat
	m.status = "resumed " + msg.session.Title
	m.input.Focus()
	if msg.rendered != "" && msg.renderWidth == m.viewport.Width {
		m.viewport.SetContent(msg.rendered)
		m.viewport.GotoBottom()
	} else {
		m.renderMessages()
	}
	return m, tea.ClearScreen
}

func looksLikePaste(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes {
		return false
	}
	return len(msg.Runes) > 16 || strings.ContainsRune(string(msg.Runes), '\n')
}
