package translate

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveTranslate hits the real endpoints. Skipped unless DOTER_LIVE_TRANSLATE=1.
func TestLiveTranslate(t *testing.T) {
	if os.Getenv("DOTER_LIVE_TRANSLATE") != "1" {
		t.Skip("set DOTER_LIVE_TRANSLATE=1 to hit the live API")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	c := NewClient()
	res, err := c.Translate(ctx, "hello world", "zh-CN")
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	t.Logf("text=%q source=%q provider=%q", res.Text, res.Source, res.Provider)
	if res.Text == "" {
		t.Fatal("empty translation")
	}
}
