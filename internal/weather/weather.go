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
	"strings"
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
	Temperature   float64 `json:"temperature"`
	Apparent      float64 `json:"apparent"`
	Humidity      int     `json:"humidity"`
	DewPoint      float64 `json:"dewPoint"`
	WindSpeed     float64 `json:"windSpeed"`
	WindGusts     float64 `json:"windGusts"`
	WindDir       int     `json:"windDir"`
	Precipitation float64 `json:"precipitation"`
	CloudCover    int     `json:"cloudCover"`
	Pressure      float64 `json:"pressure"`
	Visibility    float64 `json:"visibility"`
	UVIndex       float64 `json:"uvIndex"`
	Code          int     `json:"code"`
	Description   string  `json:"description"`
	IsDay         bool    `json:"isDay"`
}

// Day is one daily forecast entry.
type Day struct {
	Date        string  `json:"date"`
	Code        int     `json:"code"`
	Description string  `json:"description"`
	Max         float64 `json:"max"`
	Min         float64 `json:"min"`
	PrecipProb  int     `json:"precipProb"`
	PrecipSum   float64 `json:"precipSum"`
	Sunrise     string  `json:"sunrise"`
	Sunset      string  `json:"sunset"`
	UVMax       float64 `json:"uvMax"`
	WindMax     float64 `json:"windMax"`
}

// Forecast is what the UI renders.
type Forecast struct {
	Location  Location `json:"location"`
	Current   Current  `json:"current"`
	Days      []Day    `json:"days"`
	UpdatedAt int64    `json:"updatedAt"`
}

// State is the saved city list plus the forecast for the active city. The
// forecast is nil when the network request failed; the city list and active
// index still let the UI offer retry and city switching.
type State struct {
	Cities   []Location `json:"cities"`
	Active   int        `json:"active"`
	Forecast *Forecast  `json:"forecast,omitempty"`
}

type cachedForecast struct {
	forecast Forecast
	at       time.Time
}

// Client fetches forecasts and remembers the chosen cities between runs.
type Client struct {
	http *http.Client
	// path is where the city list and active index are persisted.
	path string

	mu        sync.Mutex
	cities    []Location
	active    int
	forecasts map[string]cachedForecast
}

// NewClient creates a client that persists its cities at path.
func NewClient(path string) *Client {
	c := &Client{
		http:      &http.Client{Timeout: requestTimeout},
		path:      path,
		forecasts: make(map[string]cachedForecast),
	}
	c.load()
	return c
}

// State returns the city list and the forecast for the active city. The
// location is guessed from the IP address on first use. The forecast pointer
// is nil when fetching failed, while the cities stay available.
func (c *Client) State(ctx context.Context) (State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cities) == 0 {
		loc, err := c.locate(ctx)
		if err != nil {
			return State{}, err
		}
		c.cities = []Location{*loc}
		c.active = 0
		c.saveLocked()
	}

	if err := c.fetchActiveLocked(ctx); err != nil {
		return c.stateLocked(), err
	}
	st := c.stateLocked()
	f := c.forecasts[cityKey(c.cities[c.active])].forecast
	st.Forecast = &f
	return st, nil
}

// AddCity adds a city and makes it the active one. An already-known city is
// only selected, so adding a search hit twice is harmless.
func (c *Client) AddCity(loc Location) State {
	c.mu.Lock()
	defer c.mu.Unlock()

	if loc.Name == "" {
		return c.stateLocked()
	}
	if index := c.cityIndexLocked(loc); index >= 0 {
		c.active = index
		c.saveLocked()
		return c.stateLocked()
	}
	c.cities = append(c.cities, loc)
	c.active = len(c.cities) - 1
	c.saveLocked()
	return c.stateLocked()
}

// RemoveCity drops a city by index. Removing the active city selects its
// successor, or the new last city when the removed one was at the end.
func (c *Client) RemoveCity(index int) State {
	c.mu.Lock()
	defer c.mu.Unlock()

	if index < 0 || index >= len(c.cities) {
		return c.stateLocked()
	}
	c.cities = append(c.cities[:index], c.cities[index+1:]...)
	switch {
	case len(c.cities) == 0:
		c.active = 0
	case c.active > index:
		c.active--
	case c.active >= len(c.cities):
		c.active = len(c.cities) - 1
	}
	c.saveLocked()
	return c.stateLocked()
}

// StepCity moves the active index by delta, wrapping around the city list.
func (c *Client) StepCity(delta int) State {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cities) < 2 {
		return c.stateLocked()
	}
	n := len(c.cities)
	c.active = ((c.active+delta)%n + n) % n
	c.saveLocked()
	return c.stateLocked()
}

// Peek returns the forecast for the city delta steps from the active one
// without changing the selection. The UI uses it to preload the neighbouring
// pages so a swipe can slide between ready cities instead of waiting on the
// network mid-gesture.
func (c *Client) Peek(ctx context.Context, delta int) (Forecast, error) {
	c.mu.Lock()
	if len(c.cities) == 0 {
		c.mu.Unlock()
		return Forecast{}, fmt.Errorf("no cities saved")
	}
	n := len(c.cities)
	loc := c.cities[((c.active+delta)%n+n)%n]
	key := cityKey(loc)
	if cached, ok := c.forecasts[key]; ok && time.Since(cached.at) < CacheTTL {
		c.mu.Unlock()
		return cached.forecast, nil
	}
	c.mu.Unlock()

	// Fetch outside the lock: a prefetch for a neighbour must never hold up a
	// StepCity or a refresh of the active city.
	f, err := c.fetch(ctx, loc)
	if err != nil {
		return Forecast{}, err
	}
	c.mu.Lock()
	c.forecasts[key] = cachedForecast{forecast: f, at: time.Now()}
	c.mu.Unlock()
	return f, nil
}

func (c *Client) stateLocked() State {
	cities := make([]Location, len(c.cities))
	copy(cities, c.cities)
	return State{Cities: cities, Active: c.active}
}

func (c *Client) cityIndexLocked(loc Location) int {
	key := cityKey(loc)
	for i, city := range c.cities {
		if cityKey(city) == key {
			return i
		}
	}
	return -1
}

func cityKey(loc Location) string {
	return fmt.Sprintf("%.4f,%.4f", loc.Lat, loc.Lon)
}

// fetchActiveLocked refreshes the active city's forecast when its cache
// expired. Callers must hold the mutex.
func (c *Client) fetchActiveLocked(ctx context.Context) error {
	loc := c.cities[c.active]
	key := cityKey(loc)
	if cached, ok := c.forecasts[key]; ok && time.Since(cached.at) < CacheTTL {
		return nil
	}
	f, err := c.fetch(ctx, loc)
	if err != nil {
		return err
	}
	c.forecasts[key] = cachedForecast{forecast: f, at: time.Now()}
	return nil
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
	params.Set("current", "temperature_2m,relative_humidity_2m,apparent_temperature,is_day,weather_code,wind_speed_10m,wind_gusts_10m,wind_direction_10m,precipitation,cloud_cover,pressure_msl,visibility,uv_index,dew_point_2m")
	params.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max,precipitation_sum,sunrise,sunset,uv_index_max,wind_speed_10m_max")
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
			DewPoint    float64 `json:"dew_point_2m"`
			IsDay       int     `json:"is_day"`
			Code        int     `json:"weather_code"`
			WindSpeed   float64 `json:"wind_speed_10m"`
			WindGusts   float64 `json:"wind_gusts_10m"`
			WindDir     int     `json:"wind_direction_10m"`
			Precip      float64 `json:"precipitation"`
			CloudCover  int     `json:"cloud_cover"`
			Pressure    float64 `json:"pressure_msl"`
			Visibility  float64 `json:"visibility"`
			UVIndex     float64 `json:"uv_index"`
		} `json:"current"`
		Daily struct {
			Time       []string  `json:"time"`
			Code       []int     `json:"weather_code"`
			Max        []float64 `json:"temperature_2m_max"`
			Min        []float64 `json:"temperature_2m_min"`
			PrecipProb []int     `json:"precipitation_probability_max"`
			PrecipSum  []float64 `json:"precipitation_sum"`
			Sunrise    []string  `json:"sunrise"`
			Sunset     []string  `json:"sunset"`
			UVMax      []float64 `json:"uv_index_max"`
			WindMax    []float64 `json:"wind_speed_10m_max"`
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
			PrecipSum:   atFloat(payload.Daily.PrecipSum, i),
			Sunrise:     clockOf(atString(payload.Daily.Sunrise, i)),
			Sunset:      clockOf(atString(payload.Daily.Sunset, i)),
			UVMax:       atFloat(payload.Daily.UVMax, i),
			WindMax:     atFloat(payload.Daily.WindMax, i),
		})
	}

	return Forecast{
		Location: loc,
		Current: Current{
			Temperature:   payload.Current.Temperature,
			Apparent:      payload.Current.Apparent,
			Humidity:      payload.Current.Humidity,
			DewPoint:      payload.Current.DewPoint,
			WindSpeed:     payload.Current.WindSpeed,
			WindGusts:     payload.Current.WindGusts,
			WindDir:       payload.Current.WindDir,
			Precipitation: payload.Current.Precip,
			CloudCover:    payload.Current.CloudCover,
			Pressure:      payload.Current.Pressure,
			Visibility:    payload.Current.Visibility,
			UVIndex:       payload.Current.UVIndex,
			Code:          payload.Current.Code,
			Description:   Describe(payload.Current.Code),
			IsDay:         payload.Current.IsDay == 1,
		},
		Days:      days,
		UpdatedAt: time.Now().UnixMilli(),
	}, nil
}

// clockOf trims an Open-Meteo timestamp to the local time, e.g.
// "2026-11-19T06:24" becomes "06:24". Empty input stays empty.
func clockOf(stamp string) string {
	if i := strings.IndexByte(stamp, 'T'); i >= 0 && len(stamp) >= i+6 {
		return stamp[i+1 : i+6]
	}
	return stamp
}

// at returns items[i], or zero when i is past the end.
func at(items []int, i int) int {
	if i < len(items) {
		return items[i]
	}
	return 0
}

func atFloat(items []float64, i int) float64 {
	if i < len(items) {
		return items[i]
	}
	return 0
}

func atString(items []string, i int) string {
	if i < len(items) {
		return items[i]
	}
	return ""
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

	// Current format: a city list with the active index.
	var stored struct {
		Cities []Location `json:"cities"`
		Active int        `json:"active"`
	}
	if err := json.Unmarshal(data, &stored); err == nil && len(stored.Cities) > 0 {
		c.cities = stored.Cities
		if stored.Active >= 0 && stored.Active < len(c.cities) {
			c.active = stored.Active
		}
		return
	}

	// Older builds persisted a single Location object; adopt it as the only
	// city so the first run after the upgrade keeps the chosen place.
	var loc Location
	if err := json.Unmarshal(data, &loc); err != nil || loc.Name == "" {
		return
	}
	c.cities = []Location{loc}
	c.active = 0
}

func (c *Client) saveLocked() {
	data, err := json.Marshal(struct {
		Cities []Location `json:"cities"`
		Active int        `json:"active"`
	}{Cities: c.cities, Active: c.active})
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
