package services

import (
	"context"

	"doter/internal/weather"
)

// WeatherService exposes the weather forecast and city list to the frontend.
type WeatherService struct {
	client *weather.Client
}

// NewWeatherService creates the service.
func NewWeatherService(client *weather.Client) *WeatherService {
	return &WeatherService{client: client}
}

// GetState returns the saved cities, the active index and the active city's
// forecast. The forecast is null when the network is unavailable; the panel
// shows that as "no data" while keeping the city list usable.
func (s *WeatherService) GetState() *weather.State {
	st, _ := s.client.State(context.Background())
	return &st
}

// SearchCities looks up cities by name so they can be added in settings.
func (s *WeatherService) SearchCities(query string) *[]weather.Location {
	locations, err := s.client.Search(context.Background(), query)
	if err != nil {
		return nil
	}
	return &locations
}

// AddCity adds a city to the saved list and selects it.
func (s *WeatherService) AddCity(loc weather.Location) *weather.State {
	st := s.client.AddCity(loc)
	return &st
}

// RemoveCity drops a saved city by index.
func (s *WeatherService) RemoveCity(index int) *weather.State {
	st := s.client.RemoveCity(index)
	return &st
}

// StepCity moves the active city by delta, wrapping around the list.
func (s *WeatherService) StepCity(delta int) *weather.State {
	st := s.client.StepCity(delta)
	return &st
}

// GetCityForecast returns the forecast for the city offset steps from the
// active one without selecting it. The panel preloads its neighbouring pages
// with this so a swipe never waits on the network halfway through.
func (s *WeatherService) GetCityForecast(offset int) *weather.Forecast {
	f, err := s.client.Peek(context.Background(), offset)
	if err != nil {
		return nil
	}
	return &f
}
