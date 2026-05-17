package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/llm"
	"github.com/bprendie/weazlchat/internal/storage"
	"github.com/bprendie/weazlchat/internal/tools"
)

type mode int

const (
	modeVault mode = iota
	modeServer
	modeLoading
	modeChat
	modeSessions
	modeWorkspace
	modeRenameWorkspace
	modeClearContext
)

type model struct {
	cfg                 config.Config
	cfgPath             string
	store               *storage.Store
	toolRegistry        *tools.Registry
	styles              styles
	mode                mode
	width               int
	height              int
	input               textinput.Model
	viewport            viewport.Model
	markdown            markdownRenderer
	sessions            list.Model
	workspaces          list.Model
	working             spinner.Model
	contextBar          progress.Model
	activeWorkspaceID   int64
	activeWorkspaceName string
	activeWorkspaceAt   time.Time
	renameWorkspaceID   int64
	renameReturnMode    mode
	renameDraft         string
	session             storage.Session
	messages            []storage.Message
	checkpoint          storage.ContextCheckpoint
	hasCheckpoint       bool
	err                 string
	status              string
	thinking            bool
	trimming            bool
	mouseScroll         bool
	stream              <-chan streamEvent
	streamText          string
	streamAt            time.Time
	reqIn               int
	reqOut              int
	pasteText           string
	pasteLines          int
	historyIdx          int
	historyDraft        string
	pendingTools        []llm.ToolCall
	toolResults         []string
}

type streamEvent struct {
	eventType string // "content", "tool_call", "done"
	chunk     string
	toolCalls []llm.ToolCall
	usage     llm.Usage
	err       error
	done      bool
}

type contextTrimMsg struct {
	auto            bool
	prompt          string
	currentPromptID int64
	throughID       int64
	summary         string
	err             error
}

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
		markdown:     markdownRenderer{enabled: cfg.UI.MarkdownEnabled(), style: cfg.UI.MarkdownStyle},
		sessions:     sessions,
		workspaces:   workspaces,
		working:      working,
		contextBar:   contextBar,
		mouseScroll:  true,
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
