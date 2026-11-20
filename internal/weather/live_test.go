package weather

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveForecast hits the real APIs. Skipped unless DOTER_LIVE_WEATHER=1.
func TestLiveForecast(t *testing.T) {
	if os.Getenv("DOTER_LIVE_WEATHER") != "1" {
		t.Skip("set DOTER_LIVE_WEATHER=1 to hit the live API")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := NewClient(t.TempDir() + "/weather.json")
	st, err := c.State(ctx)
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if st.Forecast == nil {
		t.Fatal("no forecast")
	}
	f := st.Forecast
	t.Logf("cities=%+v active=%d current=%+v days=%d", st.Cities, st.Active, f.Current, len(f.Days))
	if len(f.Days) == 0 {
		t.Fatal("no daily forecast")
	}

	cities, err := c.Search(ctx, "杭州")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(cities) == 0 {
		t.Fatal("no cities found")
	}
	t.Logf("first city=%+v", cities[0])
}
