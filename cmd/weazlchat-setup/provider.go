package main

import (
	"context"
	"time"

	"github.com/bprendie/weazlchat/internal/llm"
)

func fetchModels(providerType, serverURL string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return llm.FetchModels(ctx, providerType, serverURL)
}

func urlHelp(providerType string) string {
	switch providerType {
	case "vllm":
		return "Enter the base vLLM server URL only, without /v1. Example: http://localhost:8000"
	case "ollama":
		return "Enter the base Ollama server URL only, without /api. Example: http://localhost:11434"
	default:
		return "Enter the provider base URL."
	}
}

func normalizeServerURL(providerType, raw string) string {
	return llm.NormalizeServerURL(providerType, raw)
}

func defaultModel(providerType string) string {
	return llm.DefaultModel(providerType)
}
