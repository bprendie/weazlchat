package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/bprendie/weazlchat/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	reader := bufio.NewReader(os.Stdin)
	cfg, cfgPath, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Println("WeazlChat provider setup")
	providerType := askChoice(reader, "Provider", []string{"vllm", "ollama"}, "vllm")
	defaultURL := "http://localhost:8000"
	if providerType == "ollama" {
		defaultURL = "http://localhost:11434"
	}

	fmt.Println(urlHelp(providerType))
	serverURL := normalizeServerURL(providerType, askString(reader, "Base URL", defaultURL))
	fmt.Printf("Using base URL: %s\n", serverURL)
	models, err := fetchModels(providerType, serverURL)
	if err != nil {
		fmt.Printf("Could not query models: %v\n", err)
		model := askString(reader, "Model name", defaultModel(providerType))
		contextWindow := askContextWindow(reader)
		return writeConfig(cfgPath, configureTools(reader, cfg), providerType, serverURL, model, contextWindow)
	}
	if len(models) == 0 {
		fmt.Println("Provider returned no models.")
		model := askString(reader, "Model name", defaultModel(providerType))
		contextWindow := askContextWindow(reader)
		return writeConfig(cfgPath, configureTools(reader, cfg), providerType, serverURL, model, contextWindow)
	}

	model := askModel(reader, models)
	contextWindow := askContextWindow(reader)
	return writeConfig(cfgPath, configureTools(reader, cfg), providerType, serverURL, model, contextWindow)
}
