package services

import (
	"context"

	"doter/internal/weather"
)

// WeatherService exposes the weather forecast to the frontend.
type WeatherService struct {
	client *weather.Client
}

// NewWeatherService creates the service.
func NewWeatherService(client *weather.Client) *WeatherService {
	return &WeatherService{client: client}
}

// GetForecast returns the current weather and a short daily forecast,
// locating the user by IP the first time it is called. It returns null when
// the network is unavailable; the panel shows that as "no data".
func (s *WeatherService) GetForecast() *weather.Forecast {
	f, err := s.client.Forecast(context.Background())
	if err != nil {
		return nil
	}
	return &f
}

// SearchCities looks up cities by name so the location can be pinned.
func (s *WeatherService) SearchCities(query string) *[]weather.Location {
	locations, err := s.client.Search(context.Background(), query)
	if err != nil {
		return nil
	}
	return &locations
}

// SetLocation pins the forecast to a city and returns its current weather.
func (s *WeatherService) SetLocation(loc weather.Location) *weather.Forecast {
	s.client.SetLocation(loc)
	return s.GetForecast()
}
