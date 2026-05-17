package main

import (
	"bufio"
	"fmt"

	"github.com/bprendie/weazlchat/internal/config"
)

func writeConfig(cfgPath string, cfg config.Config, providerType, serverURL, model string, contextWindow int) error {
	providerID := "primary-" + providerType
	serverURL = normalizeServerURL(providerType, serverURL)
	if cfg.Providers == nil {
		cfg.Providers = map[string]config.Provider{}
	}
	if contextWindow <= 0 {
		contextWindow = 32768
	}
	cfg.ActiveProvider = providerID
	cfg.Providers[providerID] = config.Provider{
		Type:          providerType,
		ServerURL:     serverURL,
		Model:         model,
		ContextWindow: contextWindow,
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}
	fmt.Printf("Wrote config: %s\n", cfgPath)
	fmt.Printf("Active provider: %s (%s / %s)\n", providerID, providerType, model)
	if cfg.Tools.Enabled {
		fmt.Println("Tools enabled")
	}
	return nil
}

func configureTools(reader *bufio.Reader, cfg config.Config) config.Config {
	fmt.Println("")
	fmt.Println("Optional tool API keys")
	fmt.Println("Leave a key blank to keep the current value.")

	alphaKey := askSecret(reader, "Alpha Vantage API key for stock lookups", cfg.Tools.AlphaVantageKey)
	braveKey := askSecret(reader, "Brave Search API key for web search", cfg.Tools.BraveAPIKey)
	workspaceRoots := askRoots(reader, cfg.Tools.WorkspaceRoots)
	enableTools := askChoice(reader, "Enable tools", []string{"yes", "no"}, "yes") == "yes"

	cfg.Tools.AlphaVantageKey = alphaKey
	cfg.Tools.BraveAPIKey = braveKey
	cfg.Tools.WorkspaceRoots = workspaceRoots
	cfg.Tools.AutoExecute = true
	cfg.Tools.Enabled = enableTools
	return cfg
}
