package music

import "context"

// NewPlayer returns the platform media adapter.
func NewPlayer() Player {
	return newPlayer()
}

// Player is one platform's media-session adapter. Start is called when the
// music panel opens; Close releases the adapter when the panel stays closed.
// Snapshot is safe to call concurrently with transport methods.
type Player interface {
	// Start attaches to the system media session. It returns once the adapter
	// is ready to answer Snapshot; failures mean the platform source itself is
	// unavailable and are surfaced in the panel.
	Start(ctx context.Context) error
	Snapshot() NowPlaying
	// Artwork returns a data URI for the given track, or "" while the cover is
	// still loading. Callers cache it by ArtworkKey.
	Artwork(trackID string) string

	PlayPause() error
	Next() error
	Previous() error
	Seek(seconds float64) error
	ToggleShuffle() error
	CycleRepeat() error
	Close()
}
