package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/storage"
	"github.com/bprendie/weazlchat/internal/tools"
	"github.com/bprendie/weazlchat/internal/tui"
)

func main() {
	cfg, cfgPath, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	store, err := storage.Open(cfg.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		fmt.Fprintf(os.Stderr, "database migration: %v\n", err)
		os.Exit(1)
	}

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(tools.NewCalculatorTool())
	toolRegistry.Register(tools.NewDateTimeTool())
	if cfg.Tools.AlphaVantageKey != "" {
		toolRegistry.Register(tools.NewStockPriceTool(cfg.Tools.AlphaVantageKey))
	}
	if cfg.Tools.BraveAPIKey != "" {
		toolRegistry.Register(tools.NewWebSearchTool(cfg.Tools.BraveAPIKey))
	}

	p := tea.NewProgram(tui.New(cfg, cfgPath, store, toolRegistry), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
