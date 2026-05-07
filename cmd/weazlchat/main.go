package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bprendie/weazlchat/internal/app"
	"github.com/bprendie/weazlchat/internal/tui"
)

func main() {
	rt, err := app.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup: %v\n", err)
		os.Exit(1)
	}
	defer rt.Store.Close()

	p := tea.NewProgram(tui.New(rt.Config, rt.ConfigPath, rt.Store, rt.ToolRegistry), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
