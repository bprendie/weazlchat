package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/llm"
)

var contextWindowChoices = []struct {
	name   string
	tokens int
	note   string
}{
	{name: "small", tokens: 8192},
	{name: "medium", tokens: 16384},
	{name: "large", tokens: 32768},
	{name: "xl", tokens: 128000, note: "large local servers only"},
}

func (m model) isLLMConfigMode() bool {
	return m.mode == modeLLMProvider ||
		m.mode == modeLLMServer ||
		m.mode == modeLLMLoading ||
		m.mode == modeLLMModel ||
		m.mode == modeLLMContext
}

func (m model) startLLMConfig() (tea.Model, tea.Cmd) {
	p := m.cfg.Active()
	providerType := p.Type
	if providerType != "ollama" {
		providerType = "vllm"
	}
	m.llmDraft = llmConfigDraft{
		ProviderType:  providerType,
		ServerURL:     p.ServerURL,
		Model:         p.Model,
		ContextWindow: m.contextBudget(),
		ProviderIndex: providerIndex(providerType),
		ContextIndex:  contextChoiceIndex(m.contextBudget()),
		PreviousInput: m.input.Value(),
	}
	m.input.Reset()
	m.input.Blur()
	m.mode = modeLLMProvider
	m.status = "select llm provider"
	m.err = ""
	return m, nil
}

func (m model) handleLLMConfigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m.cancelLLMConfig()
	}
	switch m.mode {
	case modeLLMProvider:
		return m.handleLLMProviderKey(msg)
	case modeLLMServer:
		return m.handleLLMServerKey(msg)
	case modeLLMLoading:
		return m, nil
	case modeLLMModel:
		return m.handleLLMModelKey(msg)
	case modeLLMContext:
		return m.handleLLMContextKey(msg)
	}
	return m, nil
}

func (m model) handleLLMProviderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "left":
		m.llmDraft.ProviderIndex = max(0, m.llmDraft.ProviderIndex-1)
	case "down", "right":
		m.llmDraft.ProviderIndex = minInt(1, m.llmDraft.ProviderIndex+1)
	case "1":
		m.llmDraft.ProviderIndex = 0
	case "2":
		m.llmDraft.ProviderIndex = 1
	case "enter":
		m.llmDraft.ProviderType = []string{"vllm", "ollama"}[m.llmDraft.ProviderIndex]
		if m.llmDraft.ServerURL == "" || m.cfg.Active().Type != m.llmDraft.ProviderType {
			m.llmDraft.ServerURL = llm.DefaultServerURL(m.llmDraft.ProviderType)
		}
		m.input.SetValue(llm.NormalizeServerURL(m.llmDraft.ProviderType, m.llmDraft.ServerURL))
		m.input.CursorEnd()
		m.input.Placeholder = llm.DefaultServerURL(m.llmDraft.ProviderType)
		m.input.Focus()
		m.mode = modeLLMServer
		m.status = "set llm server"
	}
	return m, nil
}

func (m model) handleLLMServerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() != "enter" {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	serverURL := llm.NormalizeServerURL(m.llmDraft.ProviderType, m.input.Value())
	if serverURL == "" {
		serverURL = llm.DefaultServerURL(m.llmDraft.ProviderType)
	}
	m.llmDraft.ServerURL = serverURL
	m.llmDraft.Models = nil
	m.llmDraft.FetchErr = ""
	m.llmDraft.ModelIndex = 0
	m.input.Reset()
	m.input.Blur()
	m.mode = modeLLMLoading
	m.status = "querying models"
	m.working.Spinner = contextTrimSpinner
	return m, tea.Batch(m.fetchLLMModels(), m.working.Tick)
}

func (m model) fetchLLMModels() tea.Cmd {
	providerType := m.llmDraft.ProviderType
	serverURL := m.llmDraft.ServerURL
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		models, err := llm.FetchModels(ctx, providerType, serverURL)
		return llmModelsMsg{models: models, err: err}
	}
}

func (m model) handleLLMModelsMsg(msg llmModelsMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeLLMLoading {
		return m, nil
	}
	m.llmDraft.Models = msg.models
	if msg.err != nil {
		m.llmDraft.FetchErr = msg.err.Error()
		m.input.SetValue(defaultDraftModel(m.llmDraft.ProviderType, m.cfg.Active().Model))
		m.input.CursorEnd()
		m.input.Placeholder = "model name"
		m.input.Focus()
		m.status = "enter model manually"
	} else if len(msg.models) == 0 {
		m.llmDraft.FetchErr = "provider returned no models"
		m.input.SetValue(defaultDraftModel(m.llmDraft.ProviderType, m.cfg.Active().Model))
		m.input.CursorEnd()
		m.input.Placeholder = "model name"
		m.input.Focus()
		m.status = "enter model manually"
	} else {
		m.llmDraft.FetchErr = ""
		m.llmDraft.ModelIndex = modelChoiceIndex(msg.models, m.cfg.Active().Model)
		m.status = "select model"
	}
	m.mode = modeLLMModel
	return m, nil
}

func (m model) handleLLMModelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.llmDraft.FetchErr != "" || len(m.llmDraft.Models) == 0 {
		if msg.String() == "enter" {
			model := strings.TrimSpace(m.input.Value())
			if model == "" {
				m.err = "model name is required"
				return m, nil
			}
			m.llmDraft.Model = model
			m.input.Reset()
			m.input.Blur()
			m.mode = modeLLMContext
			m.status = "select context window"
			m.err = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "up", "left":
		m.llmDraft.ModelIndex = max(0, m.llmDraft.ModelIndex-1)
	case "down", "right":
		m.llmDraft.ModelIndex = minInt(len(m.llmDraft.Models)-1, m.llmDraft.ModelIndex+1)
	case "enter":
		m.llmDraft.Model = m.llmDraft.Models[m.llmDraft.ModelIndex]
		m.mode = modeLLMContext
		m.status = "select context window"
	default:
		if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= len(m.llmDraft.Models) {
			m.llmDraft.ModelIndex = n - 1
		}
	}
	return m, nil
}

func (m model) handleLLMContextKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "left":
		m.llmDraft.ContextIndex = max(0, m.llmDraft.ContextIndex-1)
	case "down", "right":
		m.llmDraft.ContextIndex = minInt(len(contextWindowChoices)-1, m.llmDraft.ContextIndex+1)
	case "enter":
		m.llmDraft.ContextWindow = contextWindowChoices[m.llmDraft.ContextIndex].tokens
		return m.saveLLMConfig()
	default:
		if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= len(contextWindowChoices) {
			m.llmDraft.ContextIndex = n - 1
		}
	}
	return m, nil
}

func (m model) saveLLMConfig() (tea.Model, tea.Cmd) {
	providerType := m.llmDraft.ProviderType
	providerID := "primary-" + providerType
	if m.cfg.Providers == nil {
		m.cfg.Providers = map[string]config.Provider{}
	}
	m.cfg.ActiveProvider = providerID
	m.cfg.Providers[providerID] = config.Provider{
		Type:          providerType,
		ServerURL:     llm.NormalizeServerURL(providerType, m.llmDraft.ServerURL),
		Model:         m.llmDraft.Model,
		ContextWindow: m.llmDraft.ContextWindow,
	}
	if err := config.Save(m.cfgPath, m.cfg); err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.input.SetValue(m.llmDraft.PreviousInput)
	m.input.CursorEnd()
	m.input.Placeholder = "message " + m.cfg.Active().Model
	m.input.Focus()
	m.llmDraft = llmConfigDraft{}
	m.mode = modeChat
	m.err = ""
	m.status = fmt.Sprintf("llm set: %s %s", providerType, m.cfg.Active().Model)
	return m, nil
}

func (m model) cancelLLMConfig() (tea.Model, tea.Cmd) {
	previous := m.llmDraft.PreviousInput
	m.llmDraft = llmConfigDraft{}
	m.input.SetValue(previous)
	m.input.CursorEnd()
	m.input.Placeholder = "message " + m.cfg.Active().Model
	m.input.Focus()
	m.mode = modeChat
	m.status = "llm config canceled"
	m.err = ""
	return m, nil
}

func providerIndex(providerType string) int {
	if providerType == "ollama" {
		return 1
	}
	return 0
}

func contextChoiceIndex(tokens int) int {
	best := 0
	for i, choice := range contextWindowChoices {
		if choice.tokens == tokens {
			return i
		}
		if choice.tokens <= tokens {
			best = i
		}
	}
	return best
}

func modelChoiceIndex(models []string, current string) int {
	for i, model := range models {
		if model == current {
			return i
		}
	}
	return 0
}

func defaultDraftModel(providerType, current string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	return llm.DefaultModel(providerType)
}
