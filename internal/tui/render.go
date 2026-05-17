package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bprendie/weazlchat/internal/storage"
)

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
	case modeLoading:
		body = m.styles.panel.Render(fmt.Sprintf("%s loading previous session", m.working.View()))
	case modeRenameWorkspace:
		body = m.renameWorkspaceView()
	case modeClearContext:
		body = m.clearContextView()
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

// renderMessages updates the viewport with the current message history and streaming state
func (m *model) renderMessages() {
	var b strings.Builder
	b.WriteString(m.renderTranscript(m.messages))
	if m.thinking {
		b.WriteString(m.styles.assistant.Render("ai"))
		b.WriteString("\n")
		if len(m.pendingTools) > 0 {
			b.WriteString(m.styles.system.Render("🔧 using tools"))
			b.WriteString("\n")
		}
		if m.streamText == "" {
			b.WriteString(m.thinkingView())
		} else {
			b.WriteString(wrapText(m.streamText, m.viewport.Width))
		}
		b.WriteString("\n\n")
	}
	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}

func (m *model) renderTranscript(messages []storage.Message) string {
	var b strings.Builder
	if len(messages) == 0 {
		b.WriteString(m.styles.system.Render("W34Zl Ch4T is ready. Local providers only."))
		if m.cfg.Tools.Enabled {
			b.WriteString("\n")
			b.WriteString(m.styles.system.Render("Tools enabled: " + strings.Join(m.getToolNames(), ", ")))
		}
	} else {
		for _, msg := range messages {
			if msg.Role == "tool" {
				continue
			}
			if msg.Role == "assistant" && strings.TrimSpace(msg.Content) == "" {
				if label := toolCallLabel(msg.ToolCalls); label != "" {
					b.WriteString(m.styles.system.Render(label))
					b.WriteString("\n\n")
				}
				continue
			}

			label := m.styles.user.Render("you")
			if msg.Role == "assistant" {
				label = m.styles.assistant.Render("ai")
			}
			b.WriteString(label)
			b.WriteString("\n")

			if msg.Content != "" {
				b.WriteString(m.renderContent(msg.Role, msg.Content))
			}
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func toolCallLabel(raw string) string {
	if raw == "" {
		return ""
	}
	var calls []struct {
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := json.Unmarshal([]byte(raw), &calls); err != nil || len(calls) == 0 {
		return "🔧 used tools"
	}
	names := make([]string, 0, len(calls))
	for _, call := range calls {
		if call.Function.Name != "" {
			names = append(names, call.Function.Name)
		}
	}
	if len(names) == 0 {
		return "🔧 used tools"
	}
	return "🔧 used " + strings.Join(names, ", ")
}

// getToolNames returns a list of registered tool names
func (m model) getToolNames() []string {
	if m.toolRegistry == nil {
		return nil
	}
	tools := m.toolRegistry.List()
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name()
	}
	return names
}

// ansiHeader returns the ASCII art header
func ansiHeader() string {
	return ` __      __          _______________.__  _________ .__        ________________
/  \    /  \ ____   /  |  \____    /|  | \_   ___ \|  |__    /  |  \__    ___/
\   \/\/   // __ \ /   |  |_/     / |  | /    \  \/|  |  \  /   |  |_|    |   
 \        /\  ___//    ^   /     /_ |  |_\     \___|   Y  \/    ^   /|    |   
  \__/\  /  \___  >____   /_______ \|____/\______  /___|  /\____   | |____|   
       \/       \/     |__|       \/             \/     \/      |__|`
}

// trimTitle shortens a title to fit display constraints
func trimTitle(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= 48 {
		return s
	}
	return s[:45] + "..."
}

// Made with Bob
