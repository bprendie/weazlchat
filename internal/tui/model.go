package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/llm"
	"github.com/bprendie/weazlchat/internal/storage"
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
	cfg        config.Config
	cfgPath    string
	store      *storage.Store
	styles     styles
	mode       mode
	width      int
	height     int
	input      textinput.Model
	viewport   viewport.Model
	sessions   list.Model
	workspaces list.Model
	session    storage.Session
	messages   []storage.Message
	err        string
	status     string
	thinking   bool
	tick       int
}

type responseMsg struct {
	content string
	err     error
}

type tickMsg time.Time

type sessionItem struct {
	session storage.Session
}

func (i sessionItem) Title() string { return i.session.Title }
func (i sessionItem) Description() string {
	return fmt.Sprintf("%s / %s", i.session.Provider, i.session.Model)
}
func (i sessionItem) FilterValue() string { return i.session.Title }

type workspaceItem storage.WorkspaceSave

func (i workspaceItem) Title() string { return storage.WorkspaceSave(i).Name }
func (i workspaceItem) Description() string {
	return storage.WorkspaceSave(i).CreatedAt.Format("2006-01-02 15:04")
}
func (i workspaceItem) FilterValue() string { return storage.WorkspaceSave(i).Name }

func New(cfg config.Config, cfgPath string, store *storage.Store) tea.Model {
	ti := textinput.New()
	ti.Placeholder = "database password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	ti.CharLimit = 4096

	sessions := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	sessions.Title = "Sessions"
	workspaces := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	workspaces.Title = "Workspace Saves"

	return model{
		cfg:        cfg,
		cfgPath:    cfgPath,
		store:      store,
		styles:     newStyles(),
		mode:       modeVault,
		input:      ti,
		viewport:   viewport.New(0, 0),
		sessions:   sessions,
		workspaces: workspaces,
		status:     "private local chat",
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
	case tea.KeyMsg:
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
		case "ctrl+r":
			if m.mode == modeChat {
				return m.showSessions()
			}
		case "ctrl+w":
			if m.mode == modeChat {
				return m.showWorkspaces()
			}
		case "esc":
			if m.mode == modeSessions || m.mode == modeWorkspace {
				m.mode = modeChat
				m.input.Focus()
			}
		case "enter":
			return m.handleEnter()
		}
	case responseMsg:
		m.thinking = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "request failed"
			return m, nil
		}
		if err := m.store.AddMessage(m.session.ID, "assistant", msg.content); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.messages, _ = m.store.Messages(m.session.ID)
		m.renderMessages()
		m.status = "ready"
	case tickMsg:
		if m.thinking {
			m.tick++
			return m, tea.Tick(110*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
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
	header := gradientLogo(ansiHeader())
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
		input := m.input.View()
		if m.thinking {
			input = m.thinkingView()
		}
		body = m.viewport.View() + "\n" + m.styles.input.Width(max(20, m.width-6)).Render(input)
	}
	help := m.styles.help.Render("enter send/select | ctrl+n new | ctrl+r sessions | ctrl+s save workspace | ctrl+w saves | esc back | ctrl+c quit")
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
		prompt := strings.TrimSpace(m.input.Value())
		if prompt == "" {
			return m, nil
		}
		m.input.Reset()
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
		m.thinking = true
		m.err = ""
		m.status = "thinking"
		return m, tea.Batch(m.complete(prompt, history[:len(history)-1]), tea.Tick(110*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) }))
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
	return m, nil
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

func (m model) complete(prompt string, history []storage.Message) tea.Cmd {
	return func() tea.Msg {
		client := llm.New(m.cfg.Active())
		content, err := client.Complete(context.Background(), history, prompt)
		return responseMsg{content: content, err: err}
	}
}

func (m *model) resize() {
	w := max(20, m.width-6)
	h := max(5, m.height-16)
	m.viewport.Width = w
	m.viewport.Height = h
	m.sessions.SetSize(w, h+4)
	m.workspaces.SetSize(w, h+4)
}

func (m *model) renderMessages() {
	var b strings.Builder
	if len(m.messages) == 0 {
		b.WriteString(m.styles.system.Render("W34Zl Ch4T is ready. Local providers only."))
	} else {
		for _, msg := range m.messages {
			label := m.styles.user.Render("you")
			if msg.Role == "assistant" {
				label = m.styles.assistant.Render("ai")
			}
			b.WriteString(label)
			b.WriteString("\n")
			b.WriteString(msg.Content)
			b.WriteString("\n\n")
		}
	}
	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}

func (m model) thinkingView() string {
	frames := []string{"<*>", "<+>", "<x>", "<#>", "<@>", "<#>", "<x>", "<+>"}
	bars := []string{"░▒▓", "▒▓█", "▓█▓", "█▓▒", "▓▒░", "▒░▒"}
	return fmt.Sprintf("%s %s model is thinking %s", m.styles.header.Render(frames[m.tick%len(frames)]), m.styles.assistant.Render(bars[m.tick%len(bars)]), m.styles.header.Render(frames[(m.tick+3)%len(frames)]))
}

func ansiHeader() string {
	return ` __      __          _______________.__  _________ .__        ________________
/  \    /  \ ____   /  |  \____    /|  | \_   ___ \|  |__    /  |  \__    ___/
\   \/\/   // __ \ /   |  |_/     / |  | /    \  \/|  |  \  /   |  |_|    |
 \        /\  ___//    ^   /     /_ |  |_\     \___|   Y  \/    ^   /|    |
  \__/\  /  \___  >____   /_______ \|____/\______  /___|  /\____   | |____|
       \/       \/     |__|       \/             \/     \/      |__|`
}

func trimTitle(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= 48 {
		return s
	}
	return s[:45] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
