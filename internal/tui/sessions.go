package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/bprendie/weazlchat/internal/storage"
)

type sessionItem struct {
	session storage.Session
}

func (i sessionItem) Title() string { return i.session.Title }
func (i sessionItem) Description() string {
	return fmt.Sprintf("%s / %s   in %d / out %d", i.session.Provider, i.session.Model, i.session.InputTokens, i.session.OutputTokens)
}
func (i sessionItem) FilterValue() string { return i.session.Title }

type workspaceItem storage.WorkspaceSave

type previousSessionMsg struct {
	session       storage.Session
	messages      []storage.Message
	checkpoint    storage.ContextCheckpoint
	hasCheckpoint bool
	rendered      string
	renderWidth   int
	err           error
}

func (m model) startChat() (tea.Model, tea.Cmd) {
	if m.cfg.UI.ResumeLastSession {
		m.mode = modeLoading
		m.status = "loading previous session"
		m.working.Spinner = spinner.Jump
		return m, tea.Batch(m.loadPreviousSession(), m.working.Tick)
	}
	return m.newSession()
}

func (m model) loadPreviousSession() tea.Cmd {
	renderer := m
	renderer.markdown.term = nil
	return func() tea.Msg {
		sess, ok, err := m.store.LatestSession()
		if err != nil || !ok {
			return previousSessionMsg{err: err}
		}
		msgs, err := m.store.Messages(sess.ID)
		if err != nil {
			return previousSessionMsg{err: err}
		}
		cp, hasCheckpoint, err := m.store.LatestContextCheckpoint(sess.ID)
		if err != nil {
			return previousSessionMsg{err: err}
		}
		rendered := ""
		if renderer.viewport.Width > 0 {
			rendered = renderer.renderTranscript(msgs)
		}
		return previousSessionMsg{
			session:       sess,
			messages:      msgs,
			checkpoint:    cp,
			hasCheckpoint: hasCheckpoint,
			rendered:      rendered,
			renderWidth:   renderer.viewport.Width,
		}
	}
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
	m.activeWorkspaceID = 0
	m.activeWorkspaceName = ""
	m.activeWorkspaceAt = time.Time{}
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
	m.activeWorkspaceID = 0
	m.activeWorkspaceName = ""
	m.activeWorkspaceAt = time.Time{}
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
