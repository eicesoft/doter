// Package music reads the system-wide now-playing session and drives its
// transport controls. Metadata comes from the platform's own media session
// service; no player-specific integration is involved.
package music

// RepeatMode is the transport repeat state reported by the player.
type RepeatMode string

const (
	RepeatOff   RepeatMode = "off"
	RepeatTrack RepeatMode = "track"
	RepeatAll   RepeatMode = "all"
)

// NowPlaying is the snapshot the frontend renders. Position is valid at
// UpdatedAt, so a playing track is advanced locally between updates instead of
// polling for every frame.
type NowPlaying struct {
	Available bool `json:"available"`
	// Error explains why nothing is available when the media source itself
	// failed (as opposed to simply having no open session).
	Error string `json:"error,omitempty"`

	TrackID    string `json:"trackId"`
	ArtworkKey string `json:"artworkKey,omitempty"`

	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	// Source is the friendly name of the playing application, e.g. "Spotify".
	Source string `json:"source"`

	IsPlaying bool    `json:"isPlaying"`
	Position  float64 `json:"position"`
	Duration  float64 `json:"duration"`
	UpdatedAt int64   `json:"updatedAt"`

	Shuffle bool       `json:"shuffle"`
	Repeat  RepeatMode `json:"repeat"`

	CanPlayPause bool `json:"canPlayPause"`
	CanPrev      bool `json:"canPrev"`
	CanNext      bool `json:"canNext"`
	CanSeek      bool `json:"canSeek"`
	CanShuffle   bool `json:"canShuffle"`
	CanRepeat    bool `json:"canRepeat"`
}

// Artwork carries the album cover for one track. It is emitted separately from
// NowPlaying because cover data is hundreds of kilobytes and only changes when
// the track changes.
type Artwork struct {
	TrackID string `json:"trackId"`
	Key     string `json:"key"`
	Data    string `json:"data"`
}
