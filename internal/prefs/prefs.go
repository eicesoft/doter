// Package prefs stores the small set of backend preferences that the
// dashboard and the clipboard watcher need before the webview exists.
package prefs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Settings is the persisted backend configuration.
type Settings struct {
	// AlertsEnabled controls desktop notifications from the alert engine.
	AlertsEnabled bool `json:"alertsEnabled"`
	// ClipboardPaused stops new clipboard entries without clearing history.
	ClipboardPaused bool `json:"clipboardPaused"`
	// ClipboardHotkeyEnabled controls the global clipboard shortcut.
	ClipboardHotkeyEnabled bool `json:"clipboardHotkeyEnabled"`
}

// Default returns the defaults for a fresh install.
func Default() Settings {
	return Settings{
		AlertsEnabled:          true,
		ClipboardPaused:        false,
		ClipboardHotkeyEnabled: true,
	}
}

// Store is a concurrency-safe settings file.
type Store struct {
	mu       sync.RWMutex
	path     string
	settings Settings
}

// New loads the settings at path, falling back to defaults when the file is
// missing or unreadable.
func New(path string) *Store {
	s := &Store{path: path, settings: Default()}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	// Start from the defaults so a file written by an older version keeps
	// sane values for fields it does not know about yet.
	loaded := Default()
	if err := json.Unmarshal(data, &loaded); err != nil {
		return s
	}
	s.settings = loaded
	return s
}

// Get returns a copy of the current settings.
func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// SetAlertsEnabled toggles desktop alerts and persists the change.
func (s *Store) SetAlertsEnabled(enabled bool) Settings {
	return s.update(func(settings *Settings) {
		settings.AlertsEnabled = enabled
	})
}

// SetClipboardPaused toggles clipboard capture and persists the change.
func (s *Store) SetClipboardPaused(paused bool) Settings {
	return s.update(func(settings *Settings) {
		settings.ClipboardPaused = paused
	})
}

// SetClipboardHotkeyEnabled toggles the global clipboard shortcut.
func (s *Store) SetClipboardHotkeyEnabled(enabled bool) Settings {
	return s.update(func(settings *Settings) {
		settings.ClipboardHotkeyEnabled = enabled
	})
}

func (s *Store) update(fn func(*Settings)) Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.settings)
	s.saveLocked()
	return s.settings
}

func (s *Store) saveLocked() {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return
	}
	// Write beside the target and rename so a crash cannot leave a
	// half-written settings file behind.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, s.path)
}
