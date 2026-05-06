package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const appName = "weazlchat"

type Config struct {
	ActiveProvider string              `json:"active_provider"`
	Providers      map[string]Provider `json:"providers"`
	Database       Database            `json:"database"`
	UI             UI                  `json:"ui"`
	Tools          Tools               `json:"tools"`
}

type Provider struct {
	Type          string `json:"type"`
	ServerURL     string `json:"server_url"`
	Model         string `json:"model"`
	APIKey        string `json:"api_key,omitempty"`
	ContextWindow int    `json:"context_window,omitempty"`
}

type Database struct {
	Path string `json:"path"`
}

type UI struct {
	ResumeLastSession bool `json:"resume_last_session"`
}

type Tools struct {
	Enabled         bool     `json:"enabled"`
	AutoExecute     bool     `json:"auto_execute_safe"`
	AlphaVantageKey string   `json:"alpha_vantage_api_key,omitempty"`
	BraveAPIKey     string   `json:"brave_api_key,omitempty"`
	WorkspaceRoots  []string `json:"workspace_roots,omitempty"`
	MaxOutputChars  int      `json:"max_output_chars,omitempty"`
	MaxFileBytes    int64    `json:"max_file_bytes,omitempty"`
}

func Load() (Config, string, error) {
	path := configPath()
	cfg := Default()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return cfg, path, err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0o700); err != nil {
		return cfg, path, err
	}

	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, path, Save(path, cfg)
	}
	if err != nil {
		return cfg, path, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, path, err
	}
	cfg.withDefaults()
	return cfg, path, nil
}

func Save(path string, cfg Config) error {
	cfg.withDefaults()
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func Default() Config {
	dataDir := dataDir()
	return Config{
		ActiveProvider: "local-vllm",
		Providers: map[string]Provider{
			"local-vllm": {
				Type:          "vllm",
				ServerURL:     "http://localhost:8000",
				Model:         "local-model",
				ContextWindow: 32768,
			},
			"local-ollama": {
				Type:          "ollama",
				ServerURL:     "http://localhost:11434",
				Model:         "llama3.1",
				ContextWindow: 32768,
			},
		},
		Database: Database{Path: filepath.Join(dataDir, "weazlchat.sqlite3")},
		UI:       UI{ResumeLastSession: true},
		Tools: Tools{
			Enabled:        false,
			AutoExecute:    true,
			MaxOutputChars: 12000,
			MaxFileBytes:   1024 * 1024,
		},
	}
}

func (c *Config) Active() Provider {
	if c.Providers == nil {
		c.Providers = map[string]Provider{}
	}
	p, ok := c.Providers[c.ActiveProvider]
	if !ok {
		return Provider{}
	}
	return p
}

func (c *Config) withDefaults() {
	def := Default()
	if c.ActiveProvider == "" {
		c.ActiveProvider = def.ActiveProvider
	}
	if c.Providers == nil || len(c.Providers) == 0 {
		c.Providers = def.Providers
	}
	for name, provider := range c.Providers {
		if provider.ContextWindow <= 0 {
			provider.ContextWindow = 32768
			c.Providers[name] = provider
		}
	}
	if c.Database.Path == "" {
		c.Database.Path = def.Database.Path
	}
	if c.Tools.MaxOutputChars <= 0 {
		c.Tools.MaxOutputChars = def.Tools.MaxOutputChars
	}
	if c.Tools.MaxFileBytes <= 0 {
		c.Tools.MaxFileBytes = def.Tools.MaxFileBytes
	}
}

func configPath() string {
	if p := os.Getenv("WEAZLCHAT_CONFIG"); p != "" {
		return p
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appName, "config.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", appName, "config.json")
}

func dataDir() string {
	if p := os.Getenv("WEAZLCHAT_DATA"); p != "" {
		return p
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", appName)
}
