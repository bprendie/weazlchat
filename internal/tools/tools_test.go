package tools

import (
	"context"
	"strings"
	"testing"
)

func TestRegistryListIsSorted(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewStockPriceTool("key"))
	registry.Register(NewCalculatorTool())
	registry.Register(NewDateTimeTool())
	registry.Register(NewWeatherTool())

	got := registry.List()
	names := make([]string, len(got))
	for i, tool := range got {
		names[i] = tool.Name()
	}

	want := []string{"calculate", "get_current_time", "get_stock_price", "get_weather"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names[%d] = %q, want %q; all names: %v", i, names[i], want[i], names)
		}
	}
}

func TestRegistryExecuteParsesRawArguments(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewCalculatorTool())

	result := registry.Execute(context.Background(), ToolCall{
		ID:      "call_1",
		Name:    "calculate",
		RawArgs: `{"operation":"multiply","a":6,"b":7}`,
	})

	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.ToolCallID != "call_1" {
		t.Fatalf("ToolCallID = %q, want call_1", result.ToolCallID)
	}
	if !strings.Contains(result.Content, "42") {
		t.Fatalf("Content = %q, want result containing 42", result.Content)
	}
}

func TestDateTimeToolUTC(t *testing.T) {
	result, err := NewDateTimeTool().Execute(context.Background(), map[string]any{
		"timezone": "UTC",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(result, "Current time in UTC:") {
		t.Fatalf("result = %q, want UTC time output", result)
	}
}

func TestDateTimeToolInvalidTimezone(t *testing.T) {
	_, err := NewDateTimeTool().Execute(context.Background(), map[string]any{
		"timezone": "No/SuchZone",
	})
	if err == nil {
		t.Fatal("Execute returned nil error for invalid timezone")
	}
}

func TestWebSearchToolRequiresQuery(t *testing.T) {
	_, err := NewWebSearchTool("key").Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("Execute returned nil error without query")
	}
}

func TestWebSearchToolRequiresAPIKey(t *testing.T) {
	_, err := NewWebSearchTool("").Execute(context.Background(), map[string]any{
		"query": "weazlchat",
	})
	if err == nil {
		t.Fatal("Execute returned nil error without API key")
	}
}

func TestWeatherToolRequiresLatitude(t *testing.T) {
	_, err := NewWeatherTool().Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("Execute returned nil error without latitude")
	}
}

func TestWeatherToolRejectsInvalidCoordinates(t *testing.T) {
	_, err := NewWeatherTool().Execute(context.Background(), map[string]any{
		"latitude":  100,
		"longitude": -75,
	})
	if err == nil {
		t.Fatal("Execute returned nil error for invalid latitude")
	}
}

func TestFormatNWSForecast(t *testing.T) {
	var point nwsPoint
	point.Properties.GridID = "PHI"
	point.Properties.GridX = 10
	point.Properties.GridY = 20
	point.Properties.RelativeLoc.Properties.City = "Philadelphia"
	point.Properties.RelativeLoc.Properties.State = "PA"

	var forecast nwsForecast
	forecast.Properties.GeneratedAt = "2026-05-06T12:00:00Z"
	forecast.Properties.Periods = append(forecast.Properties.Periods, struct {
		Name             string `json:"name"`
		StartTime        string `json:"startTime"`
		EndTime          string `json:"endTime"`
		IsDaytime        bool   `json:"isDaytime"`
		Temperature      int    `json:"temperature"`
		TemperatureUnit  string `json:"temperatureUnit"`
		WindSpeed        string `json:"windSpeed"`
		WindDirection    string `json:"windDirection"`
		ShortForecast    string `json:"shortForecast"`
		DetailedForecast string `json:"detailedForecast"`
	}{
		Name:             "Today",
		Temperature:      72,
		TemperatureUnit:  "F",
		WindSpeed:        "5 mph",
		WindDirection:    "W",
		ShortForecast:    "Sunny",
		DetailedForecast: "Sunny and mild.",
	})

	got := formatNWSForecast("Home", point, forecast, 1)
	if !strings.Contains(got, "NWS forecast for Home near Philadelphia, PA") {
		t.Fatalf("forecast output missing location: %q", got)
	}
	if !strings.Contains(got, "Today: 72°F, Sunny") {
		t.Fatalf("forecast output missing period: %q", got)
	}
}
