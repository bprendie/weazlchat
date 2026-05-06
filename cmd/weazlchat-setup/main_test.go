package main

import (
	"bufio"
	"strings"
	"testing"

	"github.com/bprendie/weazlchat/internal/config"
)

func TestConfigureToolsKeepsExistingKeysOnBlank(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n\n\n\n"))
	cfg := config.Config{
		Tools: config.Tools{
			Enabled:         true,
			AutoExecute:     true,
			AlphaVantageKey: "alpha",
			BraveAPIKey:     "brave",
			WorkspaceRoots:  []string{"/tmp/work"},
		},
	}

	got := configureTools(reader, cfg)

	if got.Tools.AlphaVantageKey != "alpha" {
		t.Fatalf("AlphaVantageKey = %q, want existing key", got.Tools.AlphaVantageKey)
	}
	if got.Tools.BraveAPIKey != "brave" {
		t.Fatalf("BraveAPIKey = %q, want existing key", got.Tools.BraveAPIKey)
	}
	if len(got.Tools.WorkspaceRoots) != 1 || got.Tools.WorkspaceRoots[0] != "/tmp/work" {
		t.Fatalf("WorkspaceRoots = %#v, want existing roots", got.Tools.WorkspaceRoots)
	}
	if !got.Tools.Enabled {
		t.Fatal("Tools.Enabled = false, want true")
	}
}

func TestConfigureToolsClearsKeysWithDash(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("-\n-\n-\n2\n"))
	cfg := config.Config{
		Tools: config.Tools{
			Enabled:         true,
			AutoExecute:     true,
			AlphaVantageKey: "alpha",
			BraveAPIKey:     "brave",
			WorkspaceRoots:  []string{"/tmp/work"},
		},
	}

	got := configureTools(reader, cfg)

	if got.Tools.AlphaVantageKey != "" {
		t.Fatalf("AlphaVantageKey = %q, want empty", got.Tools.AlphaVantageKey)
	}
	if got.Tools.BraveAPIKey != "" {
		t.Fatalf("BraveAPIKey = %q, want empty", got.Tools.BraveAPIKey)
	}
	if len(got.Tools.WorkspaceRoots) != 0 {
		t.Fatalf("WorkspaceRoots = %#v, want empty", got.Tools.WorkspaceRoots)
	}
	if got.Tools.Enabled {
		t.Fatal("Tools.Enabled = true, want false")
	}
}
