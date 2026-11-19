//go:build darwin

package music

import "testing"

func newTestAdapter() *adapterPlayer {
	return &adapterPlayer{
		state:      map[string]any{},
		artwork:    map[string]string{},
		artworkKey: map[string]string{},
	}
}

func TestAdapterMergesDiffs(t *testing.T) {
	p := newTestAdapter()

	p.applyLine([]byte(`{"type":"data","diff":false,"payload":{
		"title":"Midnight City",
		"artist":"M83",
		"album":"Hurry Up, We're Dreaming",
		"bundleIdentifier":"com.spotify.client",
		"playing":true,
		"elapsedTime":84,
		"duration":243,
		"timestamp":"2026-11-19T04:00:00Z",
		"uniqueIdentifier":"track-a",
		"shuffleMode":3,
		"repeatMode":2,
		"artworkMimeType":"image/jpeg",
		"artworkData":"/9j/4AAQSkZJRg=="
	}}`))

	np := p.Snapshot()
	if !np.Available || np.Title != "Midnight City" || np.Artist != "M83" {
		t.Fatalf("unexpected metadata: %+v", np)
	}
	if np.Source != "Spotify" {
		t.Fatalf("source = %q, want Spotify", np.Source)
	}
	if !np.IsPlaying || np.Position != 84 || np.Duration != 243 {
		t.Fatalf("unexpected timeline: %+v", np)
	}
	if !np.Shuffle || np.Repeat != RepeatTrack {
		t.Fatalf("unexpected modes: shuffle=%v repeat=%q", np.Shuffle, np.Repeat)
	}
	if !np.CanSeek || !np.CanNext || !np.CanPrev {
		t.Fatalf("unexpected capabilities: %+v", np)
	}
	if np.UpdatedAt != 1795060800000 {
		t.Fatalf("updatedAt = %d", np.UpdatedAt)
	}
	if np.ArtworkKey == "" {
		t.Fatal("artwork key is empty")
	}
	if got := p.Artwork(np.TrackID); got != "data:image/jpeg;base64,/9j/4AAQSkZJRg==" {
		t.Fatalf("artwork = %q", got)
	}

	trackBefore := np.TrackID

	p.applyLine([]byte(`{"type":"data","diff":true,"payload":{"playing":false,"elapsedTime":120.5}}`))
	np = p.Snapshot()
	if np.IsPlaying || np.Position != 120.5 {
		t.Fatalf("diff not applied: %+v", np)
	}
	if np.TrackID != trackBefore {
		t.Fatalf("track id changed on a timeline diff")
	}
	if np.ArtworkKey == "" {
		t.Fatal("artwork key dropped when only the timeline changed")
	}

	p.applyLine([]byte(`{"type":"data","diff":false,"payload":{
		"title":"Other Song",
		"artist":"Other Artist",
		"bundleIdentifier":"com.apple.Music",
		"playing":true,
		"elapsedTime":0,
		"duration":200,
		"uniqueIdentifier":"track-b"
	}}`))
	np = p.Snapshot()
	if np.TrackID == trackBefore {
		t.Fatal("track id did not change for a new item")
	}
	if np.ArtworkKey != "" {
		t.Fatal("new track inherited the previous artwork")
	}
	if np.Source != "Apple Music" {
		t.Fatalf("source = %q, want Apple Music", np.Source)
	}
}

func TestAdapterEmptyPayloadIsUnavailable(t *testing.T) {
	p := newTestAdapter()
	p.applyLine([]byte(`{"type":"data","diff":false,"payload":{}}`))

	np := p.Snapshot()
	if np.Available {
		t.Fatalf("empty payload reported available: %+v", np)
	}
	if np.Error != "" {
		t.Fatalf("unexpected error: %q", np.Error)
	}
}

func TestAdapterAssetsProbe(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	paths, err := ensureAdapterAssets()
	if err != nil {
		t.Fatalf("ensureAdapterAssets: %v", err)
	}
	if err := newTestAdapter().run(paths, "get"); err != nil {
		t.Fatalf("adapter get failed: %v", err)
	}
}
