package app

import (
	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/storage"
	"github.com/bprendie/weazlchat/internal/tools"
)

type Runtime struct {
	Config       config.Config
	ConfigPath   string
	Store        *storage.Store
	ToolRegistry *tools.Registry
}

func LoadDefault() (Runtime, error) {
	cfg, cfgPath, err := config.Load()
	if err != nil {
		return Runtime{}, err
	}
	return NewRuntime(cfg, cfgPath, cfg.Database.Path)
}

func NewRuntime(cfg config.Config, cfgPath, dbPath string) (Runtime, error) {
	store, err := storage.Open(dbPath)
	if err != nil {
		return Runtime{}, err
	}
	if err := store.Migrate(); err != nil {
		store.Close()
		return Runtime{}, err
	}
	return Runtime{
		Config:       cfg,
		ConfigPath:   cfgPath,
		Store:        store,
		ToolRegistry: NewToolRegistry(cfg, store),
	}, nil
}

func NewToolRegistry(cfg config.Config, store *storage.Store) *tools.Registry {
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
	return toolRegistry
}
