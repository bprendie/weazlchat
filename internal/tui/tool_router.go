package tui

import (
	"regexp"
	"strings"

	"github.com/bprendie/weazlchat/internal/tools"
)

var stockSymbolPattern = regexp.MustCompile(`\b[A-Z]{1,5}\b`)

func forcedToolForPrompt(prompt string, registry *tools.Registry) string {
	if registry == nil || strings.TrimSpace(prompt) == "" {
		return ""
	}
	text := strings.ToLower(prompt)
	switch {
	case shouldForceWeather(text):
		return registeredToolName(registry, "get_weather")
	case shouldForceStock(prompt, text):
		return registeredToolName(registry, "get_stock_price")
	case shouldForceDateTime(text):
		return registeredToolName(registry, "get_current_time")
	case shouldForceWebSearch(text):
		return registeredToolName(registry, "web_search")
	default:
		return ""
	}
}

func registeredToolName(registry *tools.Registry, name string) string {
	if _, ok := registry.Get(name); ok {
		return name
	}
	return ""
}

func shouldForceStock(prompt, text string) bool {
	if !(strings.Contains(text, "stock") || strings.Contains(text, "share price") || strings.Contains(text, "ticker")) {
		return false
	}
	if containsAny(text, "price", "quote", "today", "current", "now", "latest") {
		return true
	}
	return stockSymbolPattern.MatchString(prompt)
}

func shouldForceWebSearch(text string) bool {
	return strings.Contains(text, "latest on ") ||
		strings.Contains(text, "latest news") ||
		strings.Contains(text, "recent news") ||
		strings.Contains(text, "breaking news") ||
		strings.Contains(text, "news today") ||
		strings.Contains(text, "what's happening") ||
		strings.Contains(text, "what is happening")
}

func shouldForceWeather(text string) bool {
	return strings.Contains(text, "weather") ||
		strings.Contains(text, "forecast") ||
		(strings.Contains(text, "wear") && strings.Contains(text, "outside"))
}

func shouldForceDateTime(text string) bool {
	return strings.Contains(text, "what time") ||
		strings.Contains(text, "current time") ||
		strings.Contains(text, "time is it") ||
		strings.Contains(text, "today's date") ||
		strings.Contains(text, "what date")
}

func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
