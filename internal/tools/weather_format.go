package tools

import (
	"fmt"
	"strings"
	"time"
)

func formatWeather(place weatherPlace, forecast weatherForecast, tempUnit string) string {
	unit := forecast.CurrentUnits.Temperature2m
	if unit == "" {
		if tempUnit == "fahrenheit" {
			unit = "°F"
		} else {
			unit = "°C"
		}
	}

	location := weatherLocation(place)
	requestDate := weatherRequestDate(place.Timezone)

	var b strings.Builder
	fmt.Fprintf(&b, "Weather for %s\n", location)
	fmt.Fprintf(&b, "Request date: %s", requestDate)
	if place.Timezone != "" {
		fmt.Fprintf(&b, " (%s)", place.Timezone)
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "Current: %.0f%s, feels like %.0f%s, %s\n",
		forecast.Current.Temperature2m,
		unit,
		forecast.Current.ApparentTemperature,
		unit,
		weatherCodeText(forecast.Current.WeatherCode),
	)
	fmt.Fprintf(&b, "Humidity: %d%%\n", forecast.Current.RelativeHumidity2m)
	fmt.Fprintf(&b, "Wind: %.0f %s from %d°\n", forecast.Current.WindSpeed10m, forecast.CurrentUnits.WindSpeed10m, forecast.Current.WindDirection10m)
	fmt.Fprintf(&b, "Precipitation: %.2f %s\n", forecast.Current.Precipitation, forecast.CurrentUnits.Precipitation)
	if forecast.Current.Time != "" {
		fmt.Fprintf(&b, "Observed: %s\n", forecast.Current.Time)
	}

	writeDailyForecast(&b, forecast, unit, requestDate)
	return strings.TrimSpace(b.String())
}

func weatherLocation(place weatherPlace) string {
	location := place.Name
	if place.Admin1 != "" {
		location += ", " + place.Admin1
	}
	if place.Country != "" {
		location += ", " + place.Country
	}
	return location
}

func weatherRequestDate(timezone string) string {
	if timezone != "" {
		if loc, err := time.LoadLocation(timezone); err == nil {
			return time.Now().In(loc).Format("2006-01-02")
		}
	}
	return time.Now().Format("2006-01-02")
}

func writeDailyForecast(b *strings.Builder, forecast weatherForecast, unit, requestDate string) {
	if len(forecast.Daily.Time) == 0 {
		return
	}
	b.WriteString("\nForecast:\n")
	for i, day := range forecast.Daily.Time {
		if i >= len(forecast.Daily.TemperatureMax) || i >= len(forecast.Daily.TemperatureMin) || i >= len(forecast.Daily.WeatherCode) {
			break
		}
		precip := 0.0
		if i < len(forecast.Daily.PrecipitationSum) {
			precip = forecast.Daily.PrecipitationSum[i]
		}
		label := day
		if day == requestDate {
			label += " (today)"
		}
		fmt.Fprintf(b, "%s: %.0f%s/%.0f%s, %s, precip %.2f %s\n",
			label,
			forecast.Daily.TemperatureMax[i],
			unit,
			forecast.Daily.TemperatureMin[i],
			unit,
			weatherCodeText(forecast.Daily.WeatherCode[i]),
			precip,
			forecast.DailyUnits.PrecipitationSum,
		)
	}
}

func weatherCodeText(code int) string {
	switch code {
	case 0:
		return "clear sky"
	case 1, 2, 3:
		return "partly cloudy"
	case 45, 48:
		return "fog"
	case 51, 53, 55:
		return "drizzle"
	case 56, 57:
		return "freezing drizzle"
	case 61, 63, 65:
		return "rain"
	case 66, 67:
		return "freezing rain"
	case 71, 73, 75:
		return "snow"
	case 77:
		return "snow grains"
	case 80, 81, 82:
		return "rain showers"
	case 85, 86:
		return "snow showers"
	case 95:
		return "thunderstorm"
	case 96, 99:
		return "thunderstorm with hail"
	default:
		return fmt.Sprintf("weather code %d", code)
	}
}
