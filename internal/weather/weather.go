// Package weather fetches current conditions and a short forecast from
// Open-Meteo, which needs no API key, plus a coarse location from the
// caller's IP address.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	forecastURL  = "https://api.open-meteo.com/v1/forecast"
	geocodingURL = "https://geocoding-api.open-meteo.com/v1/search"
	locateURL    = "https://ipwho.is/"

	requestTimeout = 10 * time.Second
	// CacheTTL is how long a forecast is reused. The panel is opened often
	// and neither the weather nor the free API's rate limit warrants a
	// request every time.
	CacheTTL = 10 * time.Minute
	// ForecastDays is how many daily entries are requested, today included.
	ForecastDays = 5
)

// Location is where the forecast is for.
type Location struct {
	Name    string  `json:"name"`
	Region  string  `json:"region,omitempty"`
	Country string  `json:"country,omitempty"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

// Current is the right-now weather.
type Current struct {
	Temperature float64 `json:"temperature"`
	Apparent    float64 `json:"apparent"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"windSpeed"`
	Code        int     `json:"code"`
	Description string  `json:"description"`
	IsDay       bool    `json:"isDay"`
}

// Day is one daily forecast entry.
type Day struct {
	Date        string  `json:"date"`
	Code        int     `json:"code"`
	Description string  `json:"description"`
	Max         float64 `json:"max"`
	Min         float64 `json:"min"`
	PrecipProb  int     `json:"precipProb"`
}

// Forecast is what the UI renders.
type Forecast struct {
	Location  Location `json:"location"`
	Current   Current  `json:"current"`
	Days      []Day    `json:"days"`
	UpdatedAt int64    `json:"updatedAt"`
}

// Client fetches forecasts and remembers the chosen location between runs.
type Client struct {
	http *http.Client
	// path is where the chosen location is persisted.
	path string

	mu       sync.Mutex
	location *Location
	forecast *Forecast
}

// NewClient creates a client that persists its location at path.
func NewClient(path string) *Client {
	c := &Client{
		http: &http.Client{Timeout: requestTimeout},
		path: path,
	}
	c.load()
	return c
}

// Forecast returns the current weather, fetching it when the cache expired.
// The location is guessed from the IP address on first use.
func (c *Client) Forecast(ctx context.Context) (Forecast, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.forecast != nil && time.Since(time.UnixMilli(c.forecast.UpdatedAt)) < CacheTTL {
		return *c.forecast, nil
	}

	if c.location == nil {
		loc, err := c.locate(ctx)
		if err != nil {
			return Forecast{}, err
		}
		c.location = loc
		c.saveLocked()
	}

	f, err := c.fetch(ctx, *c.location)
	if err != nil {
		return Forecast{}, err
	}
	c.forecast = &f
	return f, nil
}

// SetLocation pins the location by hand and drops the cached forecast.
func (c *Client) SetLocation(loc Location) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.location = &loc
	c.forecast = nil
	c.saveLocked()
}

// Search looks up cities by name.
func (c *Client) Search(ctx context.Context, query string) ([]Location, error) {
	if query == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("name", query)
	params.Set("count", "8")
	params.Set("language", "zh")

	var payload struct {
		Results []struct {
			Name      string  `json:"name"`
			Admin1    string  `json:"admin1"`
			Country   string  `json:"country"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := c.getJSON(ctx, geocodingURL+"?"+params.Encode(), &payload); err != nil {
		return nil, err
	}

	locations := make([]Location, 0, len(payload.Results))
	for _, r := range payload.Results {
		locations = append(locations, Location{
			Name:    r.Name,
			Region:  r.Admin1,
			Country: r.Country,
			Lat:     r.Latitude,
			Lon:     r.Longitude,
		})
	}
	return locations, nil
}

// locate guesses the city from the public IP address.
func (c *Client) locate(ctx context.Context) (*Location, error) {
	var payload struct {
		Success   bool    `json:"success"`
		City      string  `json:"city"`
		Region    string  `json:"region"`
		Country   string  `json:"country"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := c.getJSON(ctx, locateURL, &payload); err != nil {
		return nil, err
	}
	if !payload.Success || payload.City == "" {
		return nil, fmt.Errorf("weather: could not determine location from IP")
	}
	return &Location{
		Name:    payload.City,
		Region:  payload.Region,
		Country: payload.Country,
		Lat:     payload.Latitude,
		Lon:     payload.Longitude,
	}, nil
}

// fetch downloads the forecast for a location.
func (c *Client) fetch(ctx context.Context, loc Location) (Forecast, error) {
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%.4f", loc.Lat))
	params.Set("longitude", fmt.Sprintf("%.4f", loc.Lon))
	params.Set("current", "temperature_2m,relative_humidity_2m,apparent_temperature,is_day,weather_code,wind_speed_10m")
	params.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max")
	params.Set("timezone", "auto")
	params.Set("forecast_days", fmt.Sprintf("%d", ForecastDays))

	body, err := c.get(ctx, forecastURL+"?"+params.Encode())
	if err != nil {
		return Forecast{}, err
	}
	return parseForecast(body, loc)
}

// parseForecast turns an Open-Meteo response into a Forecast.
func parseForecast(body []byte, loc Location) (Forecast, error) {
	var payload struct {
		Current struct {
			Temperature float64 `json:"temperature_2m"`
			Humidity    int     `json:"relative_humidity_2m"`
			Apparent    float64 `json:"apparent_temperature"`
			IsDay       int     `json:"is_day"`
			Code        int     `json:"weather_code"`
			WindSpeed   float64 `json:"wind_speed_10m"`
		} `json:"current"`
		Daily struct {
			Time       []string  `json:"time"`
			Code       []int     `json:"weather_code"`
			Max        []float64 `json:"temperature_2m_max"`
			Min        []float64 `json:"temperature_2m_min"`
			PrecipProb []int     `json:"precipitation_probability_max"`
		} `json:"daily"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Forecast{}, fmt.Errorf("weather: decode forecast: %w", err)
	}
	if len(payload.Daily.Time) == 0 {
		return Forecast{}, fmt.Errorf("weather: empty forecast")
	}

	days := make([]Day, 0, len(payload.Daily.Time))
	for i, date := range payload.Daily.Time {
		if i >= len(payload.Daily.Code) || i >= len(payload.Daily.Max) || i >= len(payload.Daily.Min) {
			break
		}
		days = append(days, Day{
			Date:        date,
			Code:        payload.Daily.Code[i],
			Description: Describe(payload.Daily.Code[i]),
			Max:         payload.Daily.Max[i],
			Min:         payload.Daily.Min[i],
			PrecipProb:  at(payload.Daily.PrecipProb, i),
		})
	}

	return Forecast{
		Location: loc,
		Current: Current{
			Temperature: payload.Current.Temperature,
			Apparent:    payload.Current.Apparent,
			Humidity:    payload.Current.Humidity,
			WindSpeed:   payload.Current.WindSpeed,
			Code:        payload.Current.Code,
			Description: Describe(payload.Current.Code),
			IsDay:       payload.Current.IsDay == 1,
		},
		Days:      days,
		UpdatedAt: time.Now().UnixMilli(),
	}, nil
}

// at returns items[i], or zero when i is past the end.
func at(items []int, i int) int {
	if i < len(items) {
		return items[i]
	}
	return 0
}

func (c *Client) getJSON(ctx context.Context, rawURL string, out any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("weather: decode %s: %w", rawURL, err)
	}
	return nil
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weather: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("weather: read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather: %s returned %s", rawURL, resp.Status)
	}
	return body, nil
}

func (c *Client) load() {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return
	}
	var loc Location
	if err := json.Unmarshal(data, &loc); err != nil || loc.Name == "" {
		return
	}
	c.location = &loc
}

func (c *Client) saveLocked() {
	if c.location == nil {
		return
	}
	data, err := json.Marshal(c.location)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, c.path)
}
