package weather

import (
	"testing"
)

const sampleForecast = `{
  "current": {
    "temperature_2m": 10.3,
    "relative_humidity_2m": 37,
    "apparent_temperature": 6.8,
    "is_day": 1,
    "weather_code": 0,
    "wind_speed_10m": 10.8
  },
  "daily": {
    "time": ["2026-11-19", "2026-11-20"],
    "weather_code": [0, 61],
    "temperature_2m_max": [14.1, 12.0],
    "temperature_2m_min": [7.2, 8.5],
    "precipitation_probability_max": [0, 80]
  }
}`

func TestParseForecast(t *testing.T) {
	loc := Location{Name: "上海", Lat: 31.23, Lon: 121.47}
	f, err := parseForecast([]byte(sampleForecast), loc)
	if err != nil {
		t.Fatalf("parseForecast: %v", err)
	}
	if f.Current.Temperature != 10.3 || f.Current.Description != "晴" || !f.Current.IsDay {
		t.Errorf("current = %+v", f.Current)
	}
	if len(f.Days) != 2 {
		t.Fatalf("days = %d, want 2", len(f.Days))
	}
	if f.Days[1].Description != "小雨" || f.Days[1].PrecipProb != 80 {
		t.Errorf("day[1] = %+v", f.Days[1])
	}
	if f.Location.Name != "上海" {
		t.Errorf("location = %+v", f.Location)
	}
}

func TestParseForecastRaggedDaily(t *testing.T) {
	// A short daily series must not panic or produce out-of-range entries.
	body := []byte(`{"current":{"weather_code":2},"daily":{"time":["a","b"],"weather_code":[2]}}`)
	f, err := parseForecast(body, Location{Name: "x"})
	if err != nil {
		t.Fatalf("parseForecast: %v", err)
	}
	if len(f.Days) != 0 {
		t.Errorf("days = %+v, want none", f.Days)
	}
}

func TestDescribe(t *testing.T) {
	if got := Describe(95); got != "雷阵雨" {
		t.Errorf("Describe(95) = %q", got)
	}
	if got := Describe(1234); got != "未知" {
		t.Errorf("Describe(1234) = %q", got)
	}
}

func TestClockOf(t *testing.T) {
	cases := map[string]string{
		"2026-11-19T06:24": "06:24",
		"2026-11-19T16:54": "16:54",
		"":                 "",
		"garbage":          "garbage",
	}
	for in, want := range cases {
		if got := clockOf(in); got != want {
			t.Errorf("clockOf(%q) = %q, want %q", in, got, want)
		}
	}
}
