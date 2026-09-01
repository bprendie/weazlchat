package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

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
	if err := unlockFromInheritedFD(store); err != nil {
		fmt.Fprintf(os.Stderr, "vault handoff: %v\n", err)
		os.Exit(1)
	}

	toolLimits := tools.Limits{
		WorkspaceRoots: cfg.Tools.WorkspaceRoots,
		MaxOutputChars: cfg.Tools.MaxOutputChars,
		MaxFileBytes:   cfg.Tools.MaxFileBytes,
	}
	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(tools.NewCalculatorTool())
	toolRegistry.Register(tools.NewDateTimeTool())
	toolRegistry.Register(tools.NewWeatherTool())
	toolRegistry.Register(tools.NewFetchURLTool(toolLimits))
	toolRegistry.Register(tools.NewListFilesTool(toolLimits))
	toolRegistry.Register(tools.NewReadFileTool(toolLimits))
	toolRegistry.Register(tools.NewSearchFilesTool(toolLimits))
	toolRegistry.Register(tools.NewCreateFileTool(toolLimits))
	toolRegistry.Register(tools.NewRunCommandTool(toolLimits))
	toolRegistry.Register(tools.NewSQLiteQueryTool(toolLimits))
	toolRegistry.Register(tools.NewRememberTool(store, toolLimits))
	toolRegistry.Register(tools.NewRecallTool(store, toolLimits))
	toolRegistry.Register(tools.NewListMemoriesTool(store, toolLimits))
	toolRegistry.Register(tools.NewForgetTool(store))
	if cfg.Tools.AlphaVantageKey != "" {
		toolRegistry.Register(tools.NewStockPriceTool(cfg.Tools.AlphaVantageKey))
	}
	if cfg.Tools.BraveAPIKey != "" {
		toolRegistry.Register(tools.NewWebSearchTool(cfg.Tools.BraveAPIKey))
	}

	p := tea.NewProgram(tui.New(cfg, cfgPath, store, toolRegistry), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}

func unlockFromInheritedFD(store *storage.Store) error {
	value := os.Getenv("WEAZL_VAULT_KEY_FD")
	if value == "" {
		return nil
	}
	_ = os.Unsetenv("WEAZL_VAULT_KEY_FD")
	fd, err := strconv.Atoi(value)
	if err != nil || fd < 3 {
		return fmt.Errorf("invalid key descriptor")
	}
	file := os.NewFile(uintptr(fd), "vault-key")
	if file == nil {
		return fmt.Errorf("open key descriptor")
	}
	defer file.Close()
	password, err := io.ReadAll(io.LimitReader(file, 64*1024))
	if err != nil {
		return err
	}
	if len(password) == 0 {
		return fmt.Errorf("empty vault key")
	}
	err = store.Unlock(string(password))
	for i := range password {
		password[i] = 0
	}
	return err
}
