package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

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
		return m.resumeAfterAutoTrim(msg)
	}
	m.status = fmt.Sprintf("context trimmed through message %d", msg.throughID)
	m.renderMessages()
	return m, nil
}

func (m model) resumeAfterAutoTrim(msg contextTrimMsg) (tea.Model, tea.Cmd) {
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
