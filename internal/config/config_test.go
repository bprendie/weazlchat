package config

import (
	"path/filepath"
	"testing"
)

func TestLoadVaultPathOverride(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	vaultPath := filepath.Join(dir, "vault_user.db")
	t.Setenv("WEAZLCHAT_CONFIG", configPath)
	t.Setenv("WEAZL_VAULT_PATH", vaultPath)

	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Database.Path != vaultPath {
		t.Fatalf("database path = %q, want %q", cfg.Database.Path, vaultPath)
	}
}

func TestLoadWithoutVaultPathKeepsConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	t.Setenv("WEAZLCHAT_CONFIG", configPath)
	t.Setenv("WEAZL_VAULT_PATH", "")

	cfg, _, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := filepath.Join(dir, "data", "weazlchat.sqlite3")
	cfg.Database.Path = want
	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cfg, _, err = Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if cfg.Database.Path != want {
		t.Fatalf("database path = %q, want %q", cfg.Database.Path, want)
	}
}

func TestLoadRejectsRelativeVaultPath(t *testing.T) {
	t.Setenv("WEAZLCHAT_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("WEAZL_VAULT_PATH", "vault_user.db")
	if _, _, err := Load(); err == nil {
		t.Fatal("Load accepted a relative WEAZL_VAULT_PATH")
	}
}
