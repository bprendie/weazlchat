package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/llm"
	"github.com/bprendie/weazlchat/internal/storage"
	"github.com/bprendie/weazlchat/internal/tools"
)

type mode int

const (
	modeVault mode = iota
	modeServer
	modeChat
	modeSessions
	modeWorkspace
)

type model struct {
	cfg           config.Config
	cfgPath       string
	store         *storage.Store
	toolRegistry  *tools.Registry
	styles        styles
	mode          mode
	width         int
	height        int
	input         textinput.Model
	viewport      viewport.Model
	sessions      list.Model
	workspaces    list.Model
	working       spinner.Model
	contextBar    progress.Model
	session       storage.Session
	messages      []storage.Message
	checkpoint    storage.ContextCheckpoint
	hasCheckpoint bool
	err           string
	status        string
	thinking      bool
	trimming      bool
	stream        <-chan streamEvent
	streamText    string
	streamAt      time.Time
	reqIn         int
	reqOut        int
	pasteText     string
	pasteLines    int
	historyIdx    int
	historyDraft  string
	pendingTools  []llm.ToolCall
	toolResults   []string
}

type streamEvent struct {
	eventType string // "content", "tool_call", "done"
	chunk     string
	toolCalls []llm.ToolCall
	usage     llm.Usage
	err       error
	done      bool
}

var contextTrimSpinner = spinner.Spinner{
	Frames: []string{
		"▰▱▱▱",
		"▰▰▱▱",
		"▰▰▰▱",
		"▰▰▰▰",
		"▱▰▰▰",
		"▱▱▰▰",
		"▱▱▱▰",
		"▱▱▱▱",
	},
	FPS: time.Second / 12,
}

var modelThinkingPhrases = []string{
	"hacking_the_gibson",
	"jacking_into_the_matrix",
	"breaching_corporate_ice",
	"overclocking_neural_link",
	"tracing_the_uplink",
	"decrypting_sector_7",
	"sniffing_data_packets",
	"bypassing_firewall_01",
	"rerouting_the_mainframe",
	"uploading_virus_payload",
	"accessing_black_ice",
	"mapping_the_grid",
	"ghosting_the_network",
	"prying_open_the_vault",
	"optimizing_cyberdeck",
	"draining_the_data_well",
	"spoofing_host_protocol",
	"syncing_with_the_construct",
	"scrambling_bio_signals",
	"system_reboot_imminent",
	"buuu_ddy",
	"wheezing_the_juice",
	"munching_the_grindage",
	"chilling_the_tokens",
	"chilling_up_on_here",
	"taxing_the_gig",
}

type contextTrimMsg struct {
	auto            bool
	prompt          string
	currentPromptID int64
	throughID       int64
	summary         string
	err             error
}

type sessionItem struct {
	session storage.Session
}

func (i sessionItem) Title() string { return i.session.Title }
func (i sessionItem) Description() string {
	return fmt.Sprintf("%s / %s   in %d / out %d", i.session.Provider, i.session.Model, i.session.InputTokens, i.session.OutputTokens)
}
func (i sessionItem) FilterValue() string { return i.session.Title }

type workspaceItem storage.WorkspaceSave

func (i workspaceItem) Title() string { return storage.WorkspaceSave(i).Name }
func (i workspaceItem) Description() string {
	return storage.WorkspaceSave(i).CreatedAt.Format("2006-01-02 15:04")
}
func (i workspaceItem) FilterValue() string { return storage.WorkspaceSave(i).Name }

func New(cfg config.Config, cfgPath string, store *storage.Store, toolRegistry *tools.Registry) tea.Model {
	ti := textinput.New()
	ti.Placeholder = "database password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	ti.CharLimit = 65535

	sessions := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	sessions.Title = "Sessions"
	workspaces := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	workspaces.Title = "Workspace Saves"

	s := newStyles()
	working := spinner.New(
		spinner.WithSpinner(spinner.Jump),
		spinner.WithStyle(s.assistant),
	)
	contextBar := progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage())

	return model{
		cfg:          cfg,
		cfgPath:      cfgPath,
		store:        store,
		toolRegistry: toolRegistry,
		styles:       s,
		mode:         modeVault,
		input:        ti,
		viewport:     viewport.New(0, 0),
		sessions:     sessions,
		workspaces:   workspaces,
		working:      working,
		contextBar:   contextBar,
		status:       "private local chat",
	}
}

func (m model) Init() tea.Cmd {
	has, err := m.store.HasVault()
	if err != nil {
		m.err = err.Error()
	}
	if !has {
		m.input.Placeholder = "create database password"
		m.status = "create encrypted local history"
		return textinput.Blink
	}
	m.status = "unlock encrypted local history"
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
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
		case "ctrl+r":
			if m.mode == modeChat {
				return m.showSessions()
			}
		case "ctrl+w":
			if m.mode == modeChat {
				return m.showWorkspaces()
			}
		case "ctrl+d":
			if m.mode == modeSessions {
				return m.deleteSelectedSession()
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
		case "enter":
			return m.handleEnter()
		}
	case streamEvent:
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

		// Handle tool calls if present
		if len(m.pendingTools) > 0 && m.cfg.Tools.Enabled {
			return m.executeTools(inputTokens, outputTokens)
		}

		// Save assistant message
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
	case contextTrimMsg:
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
	case spinner.TickMsg:
		if m.thinking {
			var cmd tea.Cmd
			m.working, cmd = m.working.Update(msg)
			m.renderMessages()
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
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	header := renderLogo(ansiHeader(), max(20, m.width-6))
	status := m.styles.status.Render(m.status)
	if m.err != "" {
		status = m.styles.system.Render("! " + m.err)
	}
	body := ""
	switch m.mode {
	case modeVault:
		body = m.styles.panel.Render("Encrypted history password\n\n" + m.input.View())
	case modeServer:
		p := m.cfg.Active()
		body = m.styles.panel.Render(fmt.Sprintf("Server URL for %s / %s\n\n%s", p.Type, p.Model, m.input.View()))
	case modeSessions:
		body = m.sessions.View()
	case modeWorkspace:
		body = m.workspaces.View()
	default:
		input := m.inputView()
		if m.thinking {
			input = m.thinkingView()
		}
		body = m.viewport.View() + "\n" + m.metricsView() + "\n" + m.styles.input.Width(max(20, m.width-6)).Render(input)
	}
	help := m.styles.help.Render(m.helpText())
	return m.styles.frame.Width(m.width).Height(m.height).Render(strings.Join([]string{header, status, body, help}, "\n"))
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeVault:
		pw := m.input.Value()
		if pw == "" {
			return m, nil
		}
		has, err := m.store.HasVault()
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		if has {
			err = m.store.Unlock(pw)
		} else {
			err = m.store.CreateVault(pw)
		}
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.err = ""
		m.input.Reset()
		m.input.EchoMode = textinput.EchoNormal
		m.input.Placeholder = m.cfg.Active().ServerURL
		m.mode = modeServer
		m.status = "confirm server url"
	case modeServer:
		if v := strings.TrimSpace(m.input.Value()); v != "" {
			p := m.cfg.Active()
			p.ServerURL = v
			m.cfg.Providers[m.cfg.ActiveProvider] = p
			if err := config.Save(m.cfgPath, m.cfg); err != nil {
				m.err = err.Error()
				return m, nil
			}
		}
		m.input.Reset()
		m.input.Placeholder = "message " + m.cfg.Active().Model
		return m.startChat()
	case modeSessions:
		item, ok := m.sessions.SelectedItem().(sessionItem)
		if ok {
			return m.loadSession(item.session)
		}
	case modeChat:
		if m.thinking {
			return m, nil
		}
		prompt := strings.TrimSpace(m.chatPrompt())
		if prompt == "" {
			return m, nil
		}
		m.input.Reset()
		m.pasteText = ""
		m.pasteLines = 0
		m.historyIdx = 0
		m.historyDraft = ""
		if err := m.store.AddMessage(m.session.ID, "user", prompt); err != nil {
			m.err = err.Error()
			return m, nil
		}
		title := m.session.Title
		if title == "New session" {
			title = trimTitle(prompt)
			m.session.Title = title
		}
		_ = m.store.TouchSession(m.session.ID, title)
		history, err := m.store.Messages(m.session.ID)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.messages = history
		m.renderMessages()
		currentPromptID := history[len(history)-1].ID
		if m.shouldAutoTrim(history) {
			return m.trimContext(true, prompt, currentPromptID, history[len(history)-2].ID)
		}
		contextHistory, err := m.contextHistoryForPrompt(currentPromptID)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.thinking = true
		m.working.Spinner = spinner.Jump
		m.streamText = ""
		m.streamAt = time.Now()
		m.reqIn = estimateMessages(contextHistory) + estimateTokens(prompt)
		m.reqOut = 0
		m.err = ""
		m.status = "streaming"
		ch := make(chan streamEvent, 64)
		m.stream = ch
		return m, tea.Batch(m.startStream(ch, prompt, contextHistory), waitStream(ch), m.working.Tick)
	}
	return m, nil
}

func (m model) startChat() (tea.Model, tea.Cmd) {
	if m.cfg.UI.ResumeLastSession {
		if sess, ok, err := m.store.LatestSession(); err != nil {
			m.err = err.Error()
		} else if ok {
			return m.loadSession(sess)
		}
	}
	return m.newSession()
}

func (m model) newSession() (tea.Model, tea.Cmd) {
	p := m.cfg.Active()
	sess := storage.Session{
		ID:       uuid.NewString(),
		Title:    "New session",
		Provider: m.cfg.ActiveProvider,
		Model:    p.Model,
	}
	if err := m.store.CreateSession(sess.ID, sess.Title, sess.Provider, sess.Model); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.session = sess
	m.messages = nil
	m.checkpoint = storage.ContextCheckpoint{}
	m.hasCheckpoint = false
	m.historyIdx = 0
	m.historyDraft = ""
	m.mode = modeChat
	m.status = fmt.Sprintf("%s %s", p.Type, p.Model)
	m.renderMessages()
	return m, nil
}

func (m model) loadSession(sess storage.Session) (tea.Model, tea.Cmd) {
	m.session = sess
	msgs, err := m.store.Messages(sess.ID)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.messages = msgs
	m.refreshCheckpoint()
	m.historyIdx = 0
	m.historyDraft = ""
	m.mode = modeChat
	m.status = "resumed " + sess.Title
	m.input.Focus()
	m.renderMessages()
	return m, nil
}

func (m model) showSessions() (tea.Model, tea.Cmd) {
	sessions, err := m.store.ListSessions(50)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items := make([]list.Item, 0, len(sessions))
	for _, sess := range sessions {
		items = append(items, sessionItem{session: sess})
	}
	m.sessions.SetItems(items)
	m.mode = modeSessions
	m.status = "select session | ctrl+d delete"
	return m, nil
}

func (m model) deleteSelectedSession() (tea.Model, tea.Cmd) {
	item, ok := m.sessions.SelectedItem().(sessionItem)
	if !ok {
		return m, nil
	}
	deletedCurrent := item.session.ID == m.session.ID
	if err := m.store.DeleteSession(item.session.ID); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.err = ""
	m.status = "deleted " + item.session.Title
	updated, cmd := m.showSessions()
	m = updated.(model)
	m.status = "deleted " + item.session.Title
	if deletedCurrent {
		m.session = storage.Session{}
		m.messages = nil
		return m.newSession()
	}
	return m, cmd
}

func (m model) showWorkspaces() (tea.Model, tea.Cmd) {
	saves, err := m.store.WorkspaceSaves(50)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	items := make([]list.Item, 0, len(saves))
	for _, save := range saves {
		items = append(items, workspaceItem(save))
	}
	m.workspaces.SetItems(items)
	m.mode = modeWorkspace
	return m, nil
}

func (m *model) saveWorkspace() {
	name := fmt.Sprintf("%s @ %s", m.session.Title, time.Now().Format("15:04:05"))
	if err := m.store.SaveWorkspace(name, m.session.ID, m.viewport.View()); err != nil {
		m.err = err.Error()
		return
	}
	m.status = "workspace saved"
	m.err = ""
}

func (m model) handleChatKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Paste || looksLikePaste(msg) {
		m.addPaste(string(msg.Runes))
		return m, nil, true
	}
	switch msg.String() {
	case "up":
		return m.recallHistory(-1), nil, true
	case "down":
		return m.recallHistory(1), nil, true
	case "backspace", "ctrl+h":
		if m.pasteText != "" && m.input.Value() == "" {
			m.pasteText = ""
			m.pasteLines = 0
			m.status = "paste cleared"
			return m, nil, true
		}
	case "ctrl+u":
		if m.pasteText != "" {
			m.pasteText = ""
			m.pasteLines = 0
			m.input.Reset()
			m.status = "input cleared"
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m model) executeTools(inputTokens, outputTokens int) (tea.Model, tea.Cmd) {
	// Save assistant message with tool calls
	toolCallsJSON, _ := json.Marshal(m.pendingTools)
	if err := m.store.AddMessageWithTools(m.session.ID, "assistant", strings.TrimSpace(m.streamText), string(toolCallsJSON), ""); err != nil {
		m.err = err.Error()
		return m, nil
	}

	// Execute tools
	m.toolResults = make([]string, 0, len(m.pendingTools))
	for _, call := range m.pendingTools {
		tool, ok := m.toolRegistry.Get(call.Function.Name)
		if !ok {
			result := fmt.Sprintf("Tool %q not found", call.Function.Name)
			m.toolResults = append(m.toolResults, result)
			if err := m.store.AddMessageWithTools(m.session.ID, "tool", result, "", call.ID); err != nil {
				m.err = err.Error()
			}
			continue
		}

		// Check if auto-execute is allowed
		if !m.cfg.Tools.AutoExecute && tool.SafetyLevel() != tools.SafetyLevelSafe {
			result := fmt.Sprintf("Tool %q requires manual approval (auto-execute disabled)", call.Function.Name)
			m.toolResults = append(m.toolResults, result)
			if err := m.store.AddMessageWithTools(m.session.ID, "tool", result, "", call.ID); err != nil {
				m.err = err.Error()
			}
			continue
		}

		// Parse arguments
		var args map[string]any
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			result := fmt.Sprintf("Failed to parse arguments: %v", err)
			m.toolResults = append(m.toolResults, result)
			if err := m.store.AddMessageWithTools(m.session.ID, "tool", result, "", call.ID); err != nil {
				m.err = err.Error()
			}
			continue
		}

		// Execute tool
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, err := tool.Execute(ctx, args)
		cancel()

		if err != nil {
			result = fmt.Sprintf("Tool error: %v", err)
		}
		result = limitToolOutput(result, m.cfg.Tools.MaxOutputChars)

		m.toolResults = append(m.toolResults, result)
		if err := m.store.AddMessageWithTools(m.session.ID, "tool", result, "", call.ID); err != nil {
			m.err = err.Error()
		}
	}

	// Update tokens
	if err := m.store.AddSessionTokens(m.session.ID, inputTokens, outputTokens); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.session.InputTokens += inputTokens
	m.session.OutputTokens += outputTokens

	// Reload messages and continue conversation with tool results
	m.messages, _ = m.store.Messages(m.session.ID)
	contextHistory, err := m.contextHistoryForContinuation()
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.streamText = ""
	m.reqIn = estimateMessages(contextHistory)
	m.reqOut = 0
	m.renderMessages()

	// Start new stream with tool results
	m.thinking = true
	m.working.Spinner = spinner.Points
	m.streamAt = time.Now()
	m.status = "processing tool results"
	ch := make(chan streamEvent, 64)
	m.stream = ch
	m.pendingTools = nil

	return m, tea.Batch(m.startStream(ch, "", contextHistory), waitStream(ch), m.working.Tick)
}

func (m model) recallHistory(delta int) model {
	history := m.userPrompts()
	if len(history) == 0 {
		return m
	}
	if m.historyIdx == 0 {
		m.historyDraft = m.input.Value()
	}
	next := m.historyIdx + delta
	if next < -len(history) {
		next = -len(history)
	}
	if next > 0 {
		next = 0
	}
	m.historyIdx = next
	if m.historyIdx == 0 {
		m.input.SetValue(m.historyDraft)
		m.historyDraft = ""
		return m
	}
	m.pasteText = ""
	m.pasteLines = 0
	m.input.SetValue(history[len(history)+m.historyIdx])
	m.input.CursorEnd()
	return m
}

func (m model) userPrompts() []string {
	prompts := make([]string, 0, len(m.messages))
	for _, msg := range m.messages {
		if msg.Role == "user" {
			prompts = append(prompts, msg.Content)
		}
	}
	return prompts
}

func (m *model) addPaste(s string) {
	if s == "" {
		return
	}
	if m.pasteText == "" {
		m.pasteText = s
	} else {
		m.pasteText += "\n" + s
	}
	m.pasteLines = countLines(m.pasteText)
	m.historyIdx = 0
	m.historyDraft = ""
	m.status = fmt.Sprintf("captured paste: %d lines", m.pasteLines)
}

func (m model) chatPrompt() string {
	prefix := strings.TrimSpace(m.input.Value())
	if m.pasteText == "" {
		return prefix
	}
	if prefix == "" {
		return m.pasteText
	}
	return prefix + "\n\n" + m.pasteText
}

func (m model) trimContext(auto bool, prompt string, currentPromptID, throughID int64) (tea.Model, tea.Cmd) {
	if m.session.ID == "" || len(m.messages) == 0 {
		m.status = "nothing to trim"
		return m, nil
	}
	if throughID == 0 {
		throughID = m.messages[len(m.messages)-1].ID
	}
	if throughID <= 0 {
		m.status = "nothing to trim"
		return m, nil
	}
	toSummarize := make([]storage.Message, 0, len(m.messages))
	if m.hasCheckpoint {
		if throughID <= m.checkpoint.ThroughMessageID {
			m.status = "context already trimmed"
			return m, nil
		}
		toSummarize = append(toSummarize, storage.Message{
			SessionID: m.session.ID,
			Role:      "system",
			Content:   "Previous checkpoint summary:\n" + m.checkpoint.Summary,
		})
	}
	for _, msg := range m.messages {
		if msg.ID <= throughID && (!m.hasCheckpoint || msg.ID > m.checkpoint.ThroughMessageID) {
			toSummarize = append(toSummarize, msg)
		}
	}
	if len(toSummarize) == 0 {
		m.status = "nothing to trim"
		return m, nil
	}
	m.thinking = true
	m.trimming = true
	m.working.Spinner = contextTrimSpinner
	m.streamText = ""
	m.streamAt = time.Now()
	m.reqIn = estimateMessages(toSummarize)
	m.reqOut = 0
	m.err = ""
	m.status = "trimming context"
	return m, summarizeContext(m.cfg.Active(), m.store, m.session.ID, toSummarize, auto, prompt, currentPromptID, throughID)
}

func summarizeContext(provider config.Provider, store *storage.Store, sessionID string, messages []storage.Message, auto bool, prompt string, currentPromptID, throughID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		client := llm.New(provider)
		summary, err := client.Summarize(ctx, transcriptForSummary(messages), summaryTargetTokens(provider.ContextWindow))
		if err == nil {
			err = store.SaveContextCheckpoint(sessionID, throughID, summary)
		}
		return contextTrimMsg{
			auto:            auto,
			prompt:          prompt,
			currentPromptID: currentPromptID,
			throughID:       throughID,
			summary:         summary,
			err:             err,
		}
	}
}

func (m *model) refreshCheckpoint() {
	m.checkpoint = storage.ContextCheckpoint{}
	m.hasCheckpoint = false
	if m.session.ID == "" {
		return
	}
	cp, ok, err := m.store.LatestContextCheckpoint(m.session.ID)
	if err != nil {
		m.err = err.Error()
		return
	}
	m.checkpoint = cp
	m.hasCheckpoint = ok
}

func (m model) contextHistoryForPrompt(currentPromptID int64) ([]storage.Message, error) {
	history, err := m.contextHistory()
	if err != nil {
		return nil, err
	}
	filtered := history[:0]
	for _, msg := range history {
		if msg.ID != currentPromptID {
			filtered = append(filtered, msg)
		}
	}
	return filtered, nil
}

func (m model) contextHistoryForContinuation() ([]storage.Message, error) {
	return m.contextHistory()
}

func (m model) contextHistory() ([]storage.Message, error) {
	if !m.hasCheckpoint {
		return append([]storage.Message(nil), m.messages...), nil
	}
	after, err := m.store.MessagesAfter(m.session.ID, m.checkpoint.ThroughMessageID)
	if err != nil {
		return nil, err
	}
	history := make([]storage.Message, 0, len(after)+1)
	history = append(history, storage.Message{
		SessionID: m.session.ID,
		Role:      "system",
		Content:   "Conversation checkpoint summary:\n" + m.checkpoint.Summary,
	})
	history = append(history, after...)
	return history, nil
}

func (m model) shouldAutoTrim(history []storage.Message) bool {
	if len(history) < 2 {
		return false
	}
	return float64(m.contextTokenEstimateFor(history))/float64(m.contextBudget()) >= 0.97
}

func (m model) contextTokenEstimate() int {
	return m.contextTokenEstimateFor(m.messages)
}

func (m model) contextTokenEstimateFor(messages []storage.Message) int {
	total := 0
	if m.hasCheckpoint {
		total += estimateTokens(m.checkpoint.Summary)
		for _, msg := range messages {
			if msg.ID > m.checkpoint.ThroughMessageID {
				total += estimateTokens(msg.Content)
				total += estimateTokens(msg.ToolCalls)
			}
		}
		return total
	}
	return estimateMessages(messages)
}

func (m model) contextBudget() int {
	if p := m.cfg.Active(); p.ContextWindow > 0 {
		return p.ContextWindow
	}
	return 32768
}

func summaryTargetTokens(contextWindow int) int {
	if contextWindow <= 0 {
		contextWindow = 32768
	}
	return min(2000, max(500, contextWindow/32))
}

func transcriptForSummary(messages []storage.Message) string {
	var b strings.Builder
	for _, msg := range messages {
		b.WriteString(strings.ToUpper(msg.Role))
		if msg.ToolCallID != "" {
			b.WriteString(" tool_call_id=")
			b.WriteString(msg.ToolCallID)
		}
		b.WriteString(":\n")
		if msg.Content != "" {
			b.WriteString(msg.Content)
			b.WriteString("\n")
		}
		if msg.ToolCalls != "" {
			b.WriteString("tool_calls: ")
			b.WriteString(msg.ToolCalls)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) startStream(ch chan<- streamEvent, prompt string, history []storage.Message) tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer close(ch)
			client := llm.New(m.cfg.Active())

			// Add tools if enabled
			if m.cfg.Tools.Enabled && m.toolRegistry != nil {
				toolDefs := m.toolRegistry.ToOpenAIFormat()
				if strings.ToLower(m.cfg.Active().Type) == "ollama" {
					toolDefs = m.toolRegistry.ToOllamaFormat()
				}
				client = client.WithTools(toolDefs)
			}

			usage, err := client.Stream(context.Background(), history, prompt, func(event llm.StreamEvent) {
				switch event.Type {
				case "content":
					ch <- streamEvent{eventType: "content", chunk: event.Content}
				case "tool_call":
					ch <- streamEvent{eventType: "tool_call", toolCalls: event.ToolCalls}
				}
			})
			ch <- streamEvent{eventType: "done", usage: usage, err: err, done: true}
		}()
		return nil
	}
}

func waitStream(ch <-chan streamEvent) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func looksLikePaste(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes {
		return false
	}
	return len(msg.Runes) > 16 || strings.ContainsRune(string(msg.Runes), '\n')
}
