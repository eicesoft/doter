package weather

import (
	"os"
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

func TestLegacyLocationLoad(t *testing.T) {
	path := t.TempDir() + "/weather.json"
	legacy := `{"name":"上海","region":"上海市","country":"中国","lat":31.23,"lon":121.47}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}
	c := NewClient(path)
	if len(c.cities) != 1 || c.active != 0 {
		t.Fatalf("cities=%+v active=%d, want the legacy location active", c.cities, c.active)
	}
	if got := c.cities[0].Name; got != "上海" {
		t.Errorf("city name = %q, want 上海", got)
	}
}

func TestCityListEditing(t *testing.T) {
	path := t.TempDir() + "/weather.json"
	c := NewClient(path)
	beijing := Location{Name: "北京", Region: "北京市", Lat: 39.9042, Lon: 116.4074}
	shanghai := Location{Name: "上海", Region: "上海市", Lat: 31.2304, Lon: 121.4737}

	if st := c.AddCity(beijing); len(st.Cities) != 1 || st.Active != 0 {
		t.Fatalf("after first add: cities=%d active=%d", len(st.Cities), st.Active)
	}
	if st := c.AddCity(shanghai); len(st.Cities) != 2 || st.Active != 1 {
		t.Fatalf("after second add: cities=%d active=%d", len(st.Cities), st.Active)
	}
	// Adding an existing city selects it instead of duplicating it.
	if st := c.AddCity(beijing); len(st.Cities) != 2 || st.Active != 0 {
		t.Fatalf("after duplicate add: cities=%d active=%d", len(st.Cities), st.Active)
	}

	if st := c.StepCity(-1); st.Active != 1 {
		t.Fatalf("step -1 wraps to %d, want 1", st.Active)
	}
	if st := c.StepCity(1); st.Active != 0 {
		t.Fatalf("step +1 wraps to %d, want 0", st.Active)
	}

	if st := c.RemoveCity(0); len(st.Cities) != 1 || st.Active != 0 {
		t.Fatalf("after removing first: cities=%d active=%d", len(st.Cities), st.Active)
	}
	if st := c.RemoveCity(0); len(st.Cities) != 0 || st.Active != 0 {
		t.Fatalf("after removing last: cities=%d active=%d", len(st.Cities), st.Active)
	}

	// The empty list must survive a restart instead of falling back to the
	// previous saved city.
	reloaded := NewClient(path)
	if len(reloaded.cities) != 0 {
		t.Fatalf("reloaded cities=%+v, want none", reloaded.cities)
	}
}

func TestCityListPersistence(t *testing.T) {
	path := t.TempDir() + "/weather.json"
	c := NewClient(path)
	c.AddCity(Location{Name: "北京", Lat: 39.9042, Lon: 116.4074})
	c.AddCity(Location{Name: "上海", Lat: 31.2304, Lon: 121.4737})
	c.StepCity(-1)

	reloaded := NewClient(path)
	if len(reloaded.cities) != 2 || reloaded.active != 0 {
		t.Fatalf("reloaded cities=%+v active=%d", reloaded.cities, reloaded.active)
	}
	if reloaded.cities[1].Name != "上海" {
		t.Errorf("second city = %q, want 上海", reloaded.cities[1].Name)
	}
}

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
