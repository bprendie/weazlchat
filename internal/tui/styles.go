package tui

import "github.com/charmbracelet/lipgloss"

const (
	crushPink   = lipgloss.Color("#F25D94")
	crushPurple = lipgloss.Color("#7D56F4")
	crushMint   = lipgloss.Color("#04B575")
	crushGold   = lipgloss.Color("#F7D774")
	ink         = lipgloss.Color("#FAFAFA")
	muted       = lipgloss.Color("#8E8E93")
	panel       = lipgloss.Color("#181820")
	border      = lipgloss.Color("#3D315B")
)

type styles struct {
	frame        lipgloss.Style
	header       lipgloss.Style
	panel        lipgloss.Style
	status       lipgloss.Style
	statusLabel  lipgloss.Style
	statusValue  lipgloss.Style
	statusWarn   lipgloss.Style
	help         lipgloss.Style
	helpKey      lipgloss.Style
	user         lipgloss.Style
	assistant    lipgloss.Style
	system       lipgloss.Style
	roleUser     lipgloss.Style
	roleAI       lipgloss.Style
	roleTool     lipgloss.Style
	roleSystem   lipgloss.Style
	input        lipgloss.Style
	inputCopy    lipgloss.Style
	inputWorking lipgloss.Style
	sidebar      lipgloss.Style
	sidebarSel   lipgloss.Style
}

func newStyles() styles {
	return styles{
		frame: lipgloss.NewStyle().
			Foreground(ink).
			Background(lipgloss.Color("#0D0D12")).
			Padding(1, 2),
		header: lipgloss.NewStyle().
			Foreground(crushPink).
			Bold(true),
		panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(border).
			Background(panel).
			Padding(0, 1),
		status: lipgloss.NewStyle().
			Foreground(crushMint).
			Bold(true),
		statusLabel: lipgloss.NewStyle().
			Foreground(crushPurple).
			Bold(true),
		statusValue: lipgloss.NewStyle().
			Foreground(ink),
		statusWarn: lipgloss.NewStyle().
			Foreground(crushGold).
			Bold(true),
		help:    lipgloss.NewStyle().Foreground(muted),
		helpKey: lipgloss.NewStyle().Foreground(crushPurple).Bold(true),
		user:    lipgloss.NewStyle().Foreground(crushGold).Bold(true),
		assistant: lipgloss.NewStyle().
			Foreground(crushMint).
			Bold(true),
		system: lipgloss.NewStyle().Foreground(crushPurple).Bold(true),
		roleUser: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0D0D12")).
			Background(crushGold).
			Bold(true).
			Padding(0, 1),
		roleAI: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0D0D12")).
			Background(crushMint).
			Bold(true).
			Padding(0, 1),
		roleTool: lipgloss.NewStyle().
			Foreground(ink).
			Background(crushPurple).
			Bold(true).
			Padding(0, 1),
		roleSystem: lipgloss.NewStyle().
			Foreground(ink).
			Background(border).
			Bold(true).
			Padding(0, 1),
		input: lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(crushPurple).
			Padding(0, 1),
		inputCopy: lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(crushGold).
			Padding(0, 1),
		inputWorking: lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(crushMint).
			Padding(0, 1),
		sidebar: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(border).
			Padding(0, 1),
		sidebarSel: lipgloss.NewStyle().
			Foreground(crushPink).
			Bold(true),
	}
}
