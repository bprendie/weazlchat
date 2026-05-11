package tui

import (
	"testing"

	"github.com/bprendie/weazlchat/internal/tools"
)

func TestForcedToolForPrompt(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.NewStockPriceTool("key"))
	registry.Register(tools.NewWebSearchTool("key"))
	registry.Register(tools.NewWeatherTool())
	registry.Register(tools.NewDateTimeTool())

	tests := []struct {
		prompt string
		want   string
	}{
		{prompt: "what is ibms stock price today", want: "get_stock_price"},
		{prompt: "what is IBM stock price today", want: "get_stock_price"},
		{prompt: "what is the latest on iran", want: "web_search"},
		{prompt: "what should i wear in manchester nh today if i'm going to be outside?", want: "get_weather"},
		{prompt: "what time is it in Tokyo?", want: "get_current_time"},
		{prompt: "explain how stock markets work", want: ""},
	}

	for _, tt := range tests {
		if got := forcedToolForPrompt(tt.prompt, registry); got != tt.want {
			t.Fatalf("forcedToolForPrompt(%q) = %q, want %q", tt.prompt, got, tt.want)
		}
	}
}

func TestForcedToolForPromptRequiresRegisteredTool(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.NewCalculatorTool())

	if got := forcedToolForPrompt("what is IBM stock price today", registry); got != "" {
		t.Fatalf("forcedToolForPrompt with missing stock tool = %q, want empty", got)
	}
}
