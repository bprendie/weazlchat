package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func fetchModels(providerType, serverURL string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	serverURL = normalizeServerURL(providerType, serverURL)

	switch providerType {
	case "vllm":
		return fetchVLLMModels(ctx, serverURL)
	case "ollama":
		return fetchOllamaModels(ctx, serverURL)
	default:
		return nil, fmt.Errorf("unsupported provider %q", providerType)
	}
}

func fetchVLLMModels(ctx context.Context, serverURL string) ([]string, error) {
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := getJSON(ctx, strings.TrimRight(serverURL, "/")+"/v1/models", &body); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(body.Data))
	for _, model := range body.Data {
		if model.ID != "" {
			models = append(models, model.ID)
		}
	}
	return models, nil
}

func fetchOllamaModels(ctx context.Context, serverURL string) ([]string, error) {
	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := getJSON(ctx, strings.TrimRight(serverURL, "/")+"/api/tags", &body); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(body.Models))
	for _, model := range body.Models {
		if model.Name != "" {
			models = append(models, model.Name)
		}
	}
	return models, nil
}

func getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
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
	u := strings.TrimRight(strings.TrimSpace(raw), "/")
	switch providerType {
	case "vllm":
		u = strings.TrimSuffix(u, "/v1")
	case "ollama":
		u = strings.TrimSuffix(u, "/api")
	}
	return u
}

func defaultModel(providerType string) string {
	if providerType == "ollama" {
		return "llama3.1"
	}
	return "local-model"
}
