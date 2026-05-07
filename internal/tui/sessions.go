package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
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

func (i workspaceItem) Title() string { return storage.WorkspaceSave(i).Name }
func (i workspaceItem) Description() string {
	return storage.WorkspaceSave(i).CreatedAt.Format("2006-01-02 15:04")
}
func (i workspaceItem) FilterValue() string { return storage.WorkspaceSave(i).Name }

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
