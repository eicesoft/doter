package notice

import (
	"context"
	"sync"
	"time"
)

const (
	// sampleInterval is how often the notification store is polled. The
	// platform daemons expose no change event, so polling is the only way to
	// notice new entries.
	sampleInterval = 2 * time.Second
	// maxEntries bounds the history kept in memory and in the panel.
	maxEntries = 50
)

// Monitor polls the platform notification store. One goroutine owns the
// cursor state; snapshots are published to the frontend through Updates.
type Monitor struct {
	mu       sync.RWMutex
	snapshot Snapshot
	updates  chan Snapshot
	started  bool

	fetch func(after string, limit int) ([]Entry, string, error)
	// cursor is the platform id of the newest entry already seen. It is only
	// touched by the polling goroutine.
	cursor string
	// primed is set after the first successful fetch, so a store that is
	// unavailable at startup does not flood the UI once it becomes readable.
	primed bool
}

// NewMonitor builds a monitor for the current user.
func NewMonitor() *Monitor {
	return &Monitor{
		updates: make(chan Snapshot, 1),
		fetch:   platformFetch,
		snapshot: Snapshot{
			Entries: []Entry{},
			New:     []Entry{},
		},
	}
}

// Start begins polling until ctx is cancelled.
func (m *Monitor) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	go func() {
		m.sample()

		ticker := time.NewTicker(sampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.sample()
			}
		}
	}()
}

// Updates yields a snapshot after every change. The channel is never closed.
func (m *Monitor) Updates() <-chan Snapshot { return m.updates }

// Current returns the most recent snapshot.
func (m *Monitor) Current() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap := m.snapshot
	snap.Entries = append([]Entry(nil), m.snapshot.Entries...)
	snap.New = append([]Entry(nil), m.snapshot.New...)
	return snap
}

// Clear drops the notification history shown in the panel. The platform
// store is left untouched, and notifications that arrive afterwards are
// still reported.
func (m *Monitor) Clear() {
	m.mu.Lock()
	m.snapshot = Snapshot{
		Entries:   []Entry{},
		New:       []Entry{},
		Error:     m.snapshot.Error,
		UpdatedAt: time.Now().UnixMilli(),
	}
	m.mu.Unlock()
	m.publish()
}

// Remove drops the entries with the given ids from the panel history. Like
// Clear, it leaves the platform store and the fetch cursor alone.
func (m *Monitor) Remove(ids []string) {
	if len(ids) == 0 {
		return
	}
	drop := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		drop[id] = struct{}{}
	}

	m.mu.Lock()
	changed := false
	entries := make([]Entry, 0, len(m.snapshot.Entries))
	for _, entry := range m.snapshot.Entries {
		if _, ok := drop[entry.ID]; ok {
			changed = true
			continue
		}
		entries = append(entries, entry)
	}
	if changed {
		newEntries := make([]Entry, 0, len(m.snapshot.New))
		for _, entry := range m.snapshot.New {
			if _, ok := drop[entry.ID]; !ok {
				newEntries = append(newEntries, entry)
			}
		}
		m.snapshot.Entries = entries
		m.snapshot.New = newEntries
		m.snapshot.UpdatedAt = time.Now().UnixMilli()
	}
	m.mu.Unlock()

	if changed {
		m.publish()
	}
}

func (m *Monitor) publish() {
	select {
	case m.updates <- m.Current():
	default:
		// The UI is behind; drop this frame rather than queueing.
	}
}

func (m *Monitor) sample() {
	initial := !m.primed
	entries, cursor, err := m.fetch(m.cursor, maxEntries)

	now := time.Now().UnixMilli()
	if err != nil {
		m.mu.Lock()
		if m.snapshot.Error == err.Error() {
			m.mu.Unlock()
			return
		}
		m.snapshot.Error = err.Error()
		m.snapshot.UpdatedAt = now
		m.mu.Unlock()
		m.publish()
		return
	}

	m.primed = true
	m.cursor = cursor

	m.mu.Lock()
	prev := m.snapshot
	if initial {
		m.snapshot = Snapshot{Entries: entries, New: []Entry{}, UpdatedAt: now}
		m.mu.Unlock()
		m.publish()
		return
	}
	if len(entries) == 0 && prev.Error == "" {
		m.mu.Unlock()
		return
	}
	merged := make([]Entry, 0, len(entries)+len(prev.Entries))
	merged = append(merged, entries...)
	merged = append(merged, prev.Entries...)
	if len(merged) > maxEntries {
		merged = merged[:maxEntries]
	}
	m.snapshot = Snapshot{Entries: merged, New: entries, UpdatedAt: now}
	m.mu.Unlock()

	m.publish()
}
