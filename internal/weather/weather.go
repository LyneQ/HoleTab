package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type WeatherData struct {
	Current struct {
		Temperature float64 `json:"temperature_2m"`
		WeatherCode int     `json:"weathercode"`
		WindSpeed   float64 `json:"windspeed_10m"`
	} `json:"current"`
}

type WeatherInfo struct {
	Temp      float64
	WindSpeed float64
	Emoji     string
	Label     string
}

func GetWeather(lat, lon string) (*WeatherInfo, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%s&longitude=%s&current=temperature_2m,weathercode,windspeed_10m&wind_speed_unit=kmh", lat, lon)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather api returned status %d", resp.StatusCode)
	}

	var data WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	emoji, label := MapWMO(data.Current.WeatherCode)

	return &WeatherInfo{
		Temp:      data.Current.Temperature,
		WindSpeed: data.Current.WindSpeed,
		Emoji:     emoji,
		Label:     label,
	}, nil
}

func MapWMO(code int) (string, string) {
	mapping := map[int]struct {
		Emoji string
		Label string
	}{
		0:  {"☀️", "Clear sky"},
		1:  {"🌤️", "Mainly clear"},
		2:  {"⛅", "Partly cloudy"},
		3:  {"☁️", "Overcast"},
		45: {"🌫️", "Fog"},
		48: {"🌫️", "Depositing rime fog"},
		51: {"🌦️", "Drizzle: Light"},
		53: {"🌦️", "Drizzle: Moderate"},
		55: {"🌦️", "Drizzle: Dense"},
		56: {"🌨️", "Freezing Drizzle: Light"},
		57: {"🌨️", "Freezing Drizzle: Dense"},
		61: {"🌧️", "Rain: Slight"},
		63: {"🌧️", "Rain: Moderate"},
		65: {"🌧️", "Rain: Heavy"},
		66: {"🌨️", "Freezing Rain: Light"},
		67: {"🌨️", "Freezing Rain: Heavy"},
		71: {"❄️", "Snow fall: Slight"},
		73: {"❄️", "Snow fall: Moderate"},
		75: {"❄️", "Snow fall: Heavy"},
		77: {"❄️", "Snow grains"},
		80: {"🌦️", "Rain showers: Slight"},
		81: {"🌦️", "Rain showers: Moderate"},
		82: {"🌦️", "Rain showers: Violent"},
		85: {"🌨️", "Snow showers: Slight"},
		86: {"🌨️", "Snow showers: Heavy"},
		95: {"⛈️", "Thunderstorm: Slight or moderate"},
		96: {"⛈️", "Thunderstorm with slight hail"},
		99: {"⛈️", "Thunderstorm with heavy hail"},
	}

	if info, ok := mapping[code]; ok {
		return info.Emoji, info.Label
	}
	return "❓", "Unknown"
}
