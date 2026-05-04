package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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

	serverURL := askString(reader, "LLM provider URL", defaultURL)
	models, err := fetchModels(providerType, serverURL)
	if err != nil {
		fmt.Printf("Could not query models: %v\n", err)
		model := askString(reader, "Model name", defaultModel(providerType))
		return writeConfig(cfgPath, cfg, providerType, serverURL, model)
	}
	if len(models) == 0 {
		fmt.Println("Provider returned no models.")
		model := askString(reader, "Model name", defaultModel(providerType))
		return writeConfig(cfgPath, cfg, providerType, serverURL, model)
	}

	model := askModel(reader, models)
	return writeConfig(cfgPath, cfg, providerType, serverURL, model)
}

func fetchModels(providerType, serverURL string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

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

func writeConfig(cfgPath string, cfg config.Config, providerType, serverURL, model string) error {
	providerID := "primary-" + providerType
	if cfg.Providers == nil {
		cfg.Providers = map[string]config.Provider{}
	}
	cfg.ActiveProvider = providerID
	cfg.Providers[providerID] = config.Provider{
		Type:      providerType,
		ServerURL: serverURL,
		Model:     model,
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}
	fmt.Printf("Wrote config: %s\n", cfgPath)
	fmt.Printf("Active provider: %s (%s / %s)\n", providerID, providerType, model)
	return nil
}

func askChoice(reader *bufio.Reader, label string, choices []string, def string) string {
	for {
		fmt.Printf("%s:\n", label)
		for i, choice := range choices {
			fmt.Printf("  %d) %s\n", i+1, choice)
		}
		answer := askString(reader, "Select", def)
		if answer == "" {
			return def
		}
		if n, err := strconv.Atoi(answer); err == nil && n >= 1 && n <= len(choices) {
			return choices[n-1]
		}
		for _, choice := range choices {
			if strings.EqualFold(answer, choice) {
				return choice
			}
		}
		fmt.Println("Enter a menu number or provider name.")
	}
}

func askModel(reader *bufio.Reader, models []string) string {
	for {
		fmt.Println("Models:")
		for i, model := range models {
			fmt.Printf("  %d) %s\n", i+1, model)
		}
		answer := askString(reader, "Select model", "1")
		n, err := strconv.Atoi(answer)
		if err == nil && n >= 1 && n <= len(models) {
			return models[n-1]
		}
		if answer != "" && contains(models, answer) {
			return answer
		}
		fmt.Println("Enter a menu number or exact model name.")
	}
}

func askString(reader *bufio.Reader, label, def string) string {
	if def == "" {
		fmt.Printf("%s: ", label)
	} else {
		fmt.Printf("%s [%s]: ", label, def)
	}
	answer, err := reader.ReadString('\n')
	if err != nil && answer == "" {
		return def
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return def
	}
	return answer
}

func defaultModel(providerType string) string {
	if providerType == "ollama" {
		return "llama3.1"
	}
	return "local-model"
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
