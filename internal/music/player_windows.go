//go:build windows && cgo

package music

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc/control"
	"github.com/xiaowumin-mark/smtc-suite-go/pkg/smtc/monitor"
)

type windowsPlayer struct {
	mu      sync.RWMutex
	current *smtc.SessionInfo
	now     NowPlaying
	err     string

	artwork    map[string]string
	artworkKey map[string]string

	ctrlMu sync.Mutex
	ctrl   *control.Controller
	ctrlID string

	monitor *monitor.Monitor
	cancel  context.CancelFunc
	done    chan struct{}
	closeMu sync.Mutex
	closed  bool
}

func newPlayer() Player {
	return &windowsPlayer{
		artwork:    map[string]string{},
		artworkKey: map[string]string{},
	}
}

func (p *windowsPlayer) Start(ctx context.Context) error {
	m, err := monitor.New(nil)
	if err != nil {
		return fmt.Errorf("读取系统媒体失败: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	p.closeMu.Lock()
	if p.closed {
		p.closeMu.Unlock()
		cancel()
		_ = m.Close()
		return context.Canceled
	}
	p.monitor = m
	p.cancel = cancel
	p.done = make(chan struct{})
	p.closeMu.Unlock()

	p.refresh()
	go p.loop(ctx)

	return nil
}

func (p *windowsPlayer) loop(ctx context.Context) {
	defer close(p.done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	events := p.monitor.Events()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-events:
			if !ok {
				return
			}
			p.refresh()
		case <-ticker.C:
			// SMTC does not push timeline ticks; refresh so the panel's
			// interpolation can be re-anchored.
			p.refresh()
		}
	}
}

func (p *windowsPlayer) refresh() {
	current := p.monitor.CurrentSession()
	if current == nil {
		p.mu.Lock()
		p.current = nil
		if p.err != "" {
			p.now = NowPlaying{Error: p.err}
		} else {
			p.now = NowPlaying{}
		}
		p.mu.Unlock()
		return
	}

	info := *current
	title := info.MediaInfo.Title
	if title == "" {
		p.mu.Lock()
		p.current = nil
		p.now = NowPlaying{Error: p.err}
		p.mu.Unlock()
		return
	}

	trackID := TrackID("", info.SourceAppUserModelID, title, info.MediaInfo.Artist, info.MediaInfo.AlbumTitle)
	p.cacheArtwork(trackID, info.MediaInfo)

	duration := (info.TimelineInfo.EndTime - info.TimelineInfo.StartTime).Seconds()
	if duration < 0 {
		duration = 0
	}
	position := info.TimelineInfo.Position.Seconds()
	if position < 0 {
		position = 0
	}

	np := NowPlaying{
		Available: true,
		TrackID:   trackID,
		Title:     title,
		Artist:    info.MediaInfo.Artist,
		Album:     info.MediaInfo.AlbumTitle,
		Source:    FriendlySource(info.SourceAppUserModelID),

		IsPlaying: info.PlaybackStatus == smtc.PlaybackStatusPlaying,
		Position:  position,
		Duration:  duration,
		UpdatedAt: time.Now().UnixMilli(),
		Shuffle:   info.IsShuffleActive,
		Repeat:    repeatMode(info.AutoRepeatMode),

		CanPlayPause: info.PlaybackControls.PlayPauseToggle || info.PlaybackControls.Play || info.PlaybackControls.Pause,
		CanPrev:      info.PlaybackControls.Previous,
		CanNext:      info.PlaybackControls.Next,
		CanSeek:      info.PlaybackControls.PlaybackPosition && duration > 0,
		CanShuffle:   info.PlaybackControls.Shuffle,
		CanRepeat:    info.PlaybackControls.Repeat,
	}

	p.mu.Lock()
	p.current = &info
	p.now = np
	p.mu.Unlock()
}

func (p *windowsPlayer) cacheArtwork(trackID string, media smtc.MediaInfo) {
	if len(media.ThumbnailData) == 0 {
		return
	}
	key := media.ThumbnailHash
	if key == "" {
		key = shortHash(string(media.ThumbnailData))
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.artworkKey[trackID] == key {
		return
	}
	p.artworkKey[trackID] = key
	p.artwork[trackID] = ArtworkDataURI(
		http.DetectContentType(media.ThumbnailData),
		base64.StdEncoding.EncodeToString(media.ThumbnailData),
	)
}

func (p *windowsPlayer) Snapshot() NowPlaying {
	p.mu.RLock()
	defer p.mu.RUnlock()
	np := p.now
	if !np.Available && np.Error == "" {
		np.Error = p.err
	}
	if np.Available {
		np.ArtworkKey = p.artworkKey[np.TrackID]
	}
	return np
}

func (p *windowsPlayer) Artwork(trackID string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.artwork[trackID]
}

func (p *windowsPlayer) PlayPause() error {
	return p.withController(func(c *control.Controller) error {
		return c.TogglePlayPause()
	})
}

func (p *windowsPlayer) Next() error {
	return p.withController(func(c *control.Controller) error {
		return c.Next()
	})
}

func (p *windowsPlayer) Previous() error {
	return p.withController(func(c *control.Controller) error {
		return c.Previous()
	})
}

func (p *windowsPlayer) Seek(seconds float64) error {
	if seconds < 0 {
		return errors.New("播放位置无效")
	}
	position := time.Duration(seconds * float64(time.Second))
	return p.withController(func(c *control.Controller) error {
		return c.Seek(position)
	})
}

func (p *windowsPlayer) ToggleShuffle() error {
	p.mu.RLock()
	active := false
	if p.current != nil {
		active = p.current.IsShuffleActive
	}
	p.mu.RUnlock()

	return p.withController(func(c *control.Controller) error {
		return c.SetShuffle(!active)
	})
}

func (p *windowsPlayer) CycleRepeat() error {
	p.mu.RLock()
	mode := smtc.AutoRepeatNone
	if p.current != nil {
		mode = p.current.AutoRepeatMode
	}
	p.mu.RUnlock()

	next := smtc.AutoRepeatTrack
	switch mode {
	case smtc.AutoRepeatNone:
		next = smtc.AutoRepeatTrack
	case smtc.AutoRepeatTrack:
		next = smtc.AutoRepeatList
	default:
		next = smtc.AutoRepeatNone
	}
	return p.withController(func(c *control.Controller) error {
		return c.SetRepeatMode(next)
	})
}

func (p *windowsPlayer) withController(fn func(*control.Controller) error) error {
	p.ctrlMu.Lock()
	defer p.ctrlMu.Unlock()

	p.mu.RLock()
	sessionID := ""
	if p.current != nil {
		sessionID = p.current.SessionID
	}
	p.mu.RUnlock()
	if sessionID == "" {
		return errors.New("没有正在播放的媒体")
	}

	if p.ctrl == nil || p.ctrlID != sessionID {
		if p.ctrl != nil {
			_ = p.ctrl.Close()
			p.ctrl = nil
			p.ctrlID = ""
		}
		c, err := control.New(sessionID)
		if err != nil {
			return fmt.Errorf("连接媒体会话失败: %w", err)
		}
		p.ctrl = c
		p.ctrlID = sessionID
	}

	if err := fn(p.ctrl); err != nil {
		return fmt.Errorf("媒体控制失败: %w", err)
	}
	p.refresh()
	return nil
}

func (p *windowsPlayer) Close() {
	p.closeMu.Lock()
	if p.closed {
		p.closeMu.Unlock()
		return
	}
	p.closed = true
	cancel := p.cancel
	done := p.done
	m := p.monitor
	p.closeMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}

	p.ctrlMu.Lock()
	if p.ctrl != nil {
		_ = p.ctrl.Close()
		p.ctrl = nil
	}
	p.ctrlMu.Unlock()

	if m != nil {
		_ = m.Close()
	}
}

func repeatMode(mode smtc.AutoRepeatMode) RepeatMode {
	switch mode {
	case smtc.AutoRepeatTrack:
		return RepeatTrack
	case smtc.AutoRepeatList:
		return RepeatAll
	default:
		return RepeatOff
	}
}
