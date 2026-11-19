package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"doter/internal/music"
)

const (
	// EventMusic carries a music.NowPlaying snapshot.
	EventMusic = "music:update"
	// EventMusicArtwork carries a music.Artwork payload when a cover appears or
	// changes. Kept separate so the per-second snapshots stay small.
	EventMusicArtwork = "music:artwork"

	// musicIdleStop is how long the adapter keeps running after the panel
	// closes, so reopening the tab does not respawn it.
	musicIdleStop  = 20 * time.Second
	musicPollEvery = time.Second
)

type musicSession struct {
	player music.Player
	ctx    context.Context
	cancel context.CancelFunc
}

// MusicService exposes the system now-playing session to the music tab.
//
// The platform adapter only runs while the panel is open: Subscribe starts it,
// Unsubscribe stops it after a short grace period.
type MusicService struct {
	app *application.App

	mu       sync.Mutex
	subs     int
	session  *musicSession
	idle     *time.Timer
	now      music.NowPlaying
	lastJSON string
	lastArt  string
}

// NewMusicService creates the service.
func NewMusicService(app *application.App) *MusicService {
	return &MusicService{app: app}
}

// Subscribe starts the platform adapter when the music panel mounts.
func (s *MusicService) Subscribe() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	s.subs++
	if s.session != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &musicSession{player: music.NewPlayer(), ctx: ctx, cancel: cancel}
	s.session = session
	go s.run(session)
}

// Unsubscribe releases the panel's hold on the adapter. The helper stays
// alive briefly so switching tabs back and forth is free.
func (s *MusicService) Unsubscribe() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.subs > 0 {
		s.subs--
	}
	if s.subs > 0 || s.session == nil || s.idle != nil {
		return
	}
	s.idle = time.AfterFunc(musicIdleStop, s.idleStop)
}

func (s *MusicService) idleStop() {
	s.mu.Lock()
	if s.subs > 0 {
		// The panel reopened while the timer was firing.
		s.idle = nil
		s.mu.Unlock()
		return
	}
	session := s.session
	s.session = nil
	s.idle = nil
	s.resetStateLocked()
	s.mu.Unlock()

	if session != nil {
		session.cancel()
	}
}

// ServiceShutdown stops the platform adapter before the process exits. Without
// it the macOS stream helper would outlive the app.
func (s *MusicService) ServiceShutdown() error {
	s.mu.Lock()
	session := s.session
	s.session = nil
	s.subs = 0
	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	s.resetStateLocked()
	s.mu.Unlock()

	if session != nil {
		session.cancel()
		session.player.Close()
	}
	return nil
}

func (s *MusicService) run(session *musicSession) {
	defer session.player.Close()

	if err := session.player.Start(session.ctx); err != nil {
		if session.ctx.Err() != nil {
			return
		}
		s.mu.Lock()
		if s.session == session {
			s.now = music.NowPlaying{Error: err.Error()}
			s.lastJSON = ""
		}
		s.mu.Unlock()
		s.app.Event.Emit(EventMusic, music.NowPlaying{Error: err.Error()})
		return
	}

	s.poll(session)

	ticker := time.NewTicker(musicPollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-session.ctx.Done():
			return
		case <-ticker.C:
			s.poll(session)
		}
	}
}

func (s *MusicService) poll(session *musicSession) {
	now := session.player.Snapshot()
	payload, _ := json.Marshal(now)
	key := now.TrackID + "|" + now.ArtworkKey

	var artwork music.Artwork
	var emitArtwork bool
	s.mu.Lock()
	if s.session != session {
		s.mu.Unlock()
		return
	}
	changed := string(payload) != s.lastJSON
	if changed {
		s.lastJSON = string(payload)
		s.now = now
	}
	if now.Available && now.ArtworkKey != "" && key != s.lastArt {
		if data := session.player.Artwork(now.TrackID); data != "" {
			s.lastArt = key
			artwork = music.Artwork{TrackID: now.TrackID, Key: now.ArtworkKey, Data: data}
			emitArtwork = true
		}
	}
	s.mu.Unlock()

	if changed {
		s.app.Event.Emit(EventMusic, now)
	}
	if emitArtwork {
		s.app.Event.Emit(EventMusicArtwork, artwork)
	}
}

func (s *MusicService) resetStateLocked() {
	s.now = music.NowPlaying{}
	s.lastJSON = ""
	s.lastArt = ""
}

// GetNowPlaying returns the latest snapshot. The music panel calls it once on
// mount and then follows the event stream.
func (s *MusicService) GetNowPlaying() music.NowPlaying {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.now
}

// GetArtwork returns the cover data URI cached for a track, or "" while it is
// still loading.
func (s *MusicService) GetArtwork(trackID string) string {
	s.mu.Lock()
	session := s.session
	s.mu.Unlock()
	if session == nil {
		return ""
	}
	return session.player.Artwork(trackID)
}

// PlayPause toggles the current session's playback.
func (s *MusicService) PlayPause() bool {
	return s.command(func(p music.Player) error { return p.PlayPause() })
}

// Next skips to the next track.
func (s *MusicService) Next() bool {
	return s.command(func(p music.Player) error { return p.Next() })
}

// Previous returns to the previous track.
func (s *MusicService) Previous() bool {
	return s.command(func(p music.Player) error { return p.Previous() })
}

// Seek moves the current session to the given position in seconds.
func (s *MusicService) Seek(seconds float64) bool {
	return s.command(func(p music.Player) error { return p.Seek(seconds) })
}

// ToggleShuffle flips shuffle mode when the player supports it.
func (s *MusicService) ToggleShuffle() bool {
	return s.command(func(p music.Player) error { return p.ToggleShuffle() })
}

// CycleRepeat advances repeat through off, track, all.
func (s *MusicService) CycleRepeat() bool {
	return s.command(func(p music.Player) error { return p.CycleRepeat() })
}

func (s *MusicService) command(fn func(music.Player) error) bool {
	s.mu.Lock()
	session := s.session
	s.mu.Unlock()
	if session == nil {
		return false
	}
	if err := fn(session.player); err != nil {
		return false
	}
	s.poll(session)
	return true
}
