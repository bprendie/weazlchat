package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const weatherGovBaseURL = "https://api.weather.gov"

// WeatherTool fetches US weather forecasts from the National Weather Service.
type WeatherTool struct {
	client *http.Client
}

func NewWeatherTool() *WeatherTool {
	return &WeatherTool{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *WeatherTool) Name() string {
	return "get_weather"
}

func (t *WeatherTool) Description() string {
	return "Get a US National Weather Service forecast for latitude and longitude coordinates. The weather.gov API does not geocode city names"
}

func (t *WeatherTool) Parameters() []Parameter {
	return []Parameter{
		{
			Name:        "latitude",
			Type:        "number",
			Description: "Latitude for a US location, for example 40.7128",
			Required:    true,
		},
		{
			Name:        "longitude",
			Type:        "number",
			Description: "Longitude for a US location, for example -74.0060",
			Required:    true,
		},
		{
			Name:        "location_name",
			Type:        "string",
			Description: "Optional human-readable label for the coordinates",
			Required:    false,
		},
		{
			Name:        "periods",
			Type:        "number",
			Description: "Number of forecast periods to return, from 1 to 8. Defaults to 4",
			Required:    false,
		},
	}
}

func (t *WeatherTool) SafetyLevel() SafetyLevel {
	return SafetyLevelSafe
}

func (t *WeatherTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	lat, err := getNumber(params, "latitude")
	if err != nil {
		return "", err
	}
	lon, err := getNumber(params, "longitude")
	if err != nil {
		return "", err
	}
	if lat < -90 || lat > 90 {
		return "", fmt.Errorf("latitude must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return "", fmt.Errorf("longitude must be between -180 and 180")
	}

	periods := 4
	if _, ok := params["periods"]; ok {
		n, err := getNumber(params, "periods")
		if err != nil {
			return "", err
		}
		periods = int(n)
	}
	if periods < 1 {
		periods = 1
	}
	if periods > 8 {
		periods = 8
	}

	label, _ := params["location_name"].(string)
	label = strings.TrimSpace(label)
	if label == "" {
		label = fmt.Sprintf("%.4f, %.4f", lat, lon)
	}

	point, err := t.point(ctx, lat, lon)
	if err != nil {
		return "", err
	}
	forecast, err := t.forecast(ctx, point.Properties.Forecast)
	if err != nil {
		return "", err
	}
	return formatNWSForecast(label, point, forecast, periods), nil
}

type nwsPoint struct {
	Properties struct {
		Forecast       string `json:"forecast"`
		ForecastHourly string `json:"forecastHourly"`
		RelativeURL    string `json:"@id"`
		GridID         string `json:"gridId"`
		GridX          int    `json:"gridX"`
		GridY          int    `json:"gridY"`
		City           string `json:"city"`
		State          string `json:"state"`
		RelativeLoc    struct {
			Properties struct {
				City     string `json:"city"`
				State    string `json:"state"`
				Distance struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unitCode"`
				} `json:"distance"`
				Bearing struct {
					Value float64 `json:"value"`
					Unit  string  `json:"unitCode"`
				} `json:"bearing"`
			} `json:"properties"`
		} `json:"relativeLocation"`
	} `json:"properties"`
}

type nwsForecast struct {
	Properties struct {
		GeneratedAt string `json:"generatedAt"`
		Updated     string `json:"updated"`
		Periods     []struct {
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
		} `json:"periods"`
	} `json:"properties"`
}

func (t *WeatherTool) point(ctx context.Context, lat, lon float64) (nwsPoint, error) {
	var point nwsPoint
	url := fmt.Sprintf("%s/points/%.4f,%.4f", weatherGovBaseURL, lat, lon)
	if err := t.getJSON(ctx, url, &point); err != nil {
		return nwsPoint{}, err
	}
	if point.Properties.Forecast == "" {
		return nwsPoint{}, fmt.Errorf("weather.gov did not return a forecast URL for %.4f,%.4f", lat, lon)
	}
	return point, nil
}

func (t *WeatherTool) forecast(ctx context.Context, forecastURL string) (nwsForecast, error) {
	var forecast nwsForecast
	if err := t.getJSON(ctx, forecastURL, &forecast); err != nil {
		return nwsForecast{}, err
	}
	if len(forecast.Properties.Periods) == 0 {
		return nwsForecast{}, fmt.Errorf("weather.gov returned no forecast periods")
	}
	return forecast, nil
}

func (t *WeatherTool) getJSON(ctx context.Context, requestURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/geo+json")
	req.Header.Set("User-Agent", "weazlchat (https://github.com/bprendie/weazlchat)")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s returned %s: %s", requestURL, resp.Status, strings.TrimSpace(string(body)))
	}
	return json.Unmarshal(body, out)
}

func formatNWSForecast(label string, point nwsPoint, forecast nwsForecast, periods int) string {
	location := label
	near := point.Properties.RelativeLoc.Properties
	if near.City != "" && near.State != "" {
		location = fmt.Sprintf("%s near %s, %s", label, near.City, near.State)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "NWS forecast for %s\n", location)
	if point.Properties.GridID != "" {
		fmt.Fprintf(&b, "Grid: %s %d,%d\n", point.Properties.GridID, point.Properties.GridX, point.Properties.GridY)
	}
	if forecast.Properties.GeneratedAt != "" {
		fmt.Fprintf(&b, "Generated: %s\n", forecast.Properties.GeneratedAt)
	}

	maxPeriods := periods
	if maxPeriods > len(forecast.Properties.Periods) {
		maxPeriods = len(forecast.Properties.Periods)
	}
	for i := 0; i < maxPeriods; i++ {
		period := forecast.Properties.Periods[i]
		fmt.Fprintf(&b, "\n%s: %d°%s, %s, wind %s %s\n",
			period.Name,
			period.Temperature,
			period.TemperatureUnit,
			period.ShortForecast,
			period.WindSpeed,
			period.WindDirection,
		)
		if period.DetailedForecast != "" {
			b.WriteString(period.DetailedForecast)
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String())
}
