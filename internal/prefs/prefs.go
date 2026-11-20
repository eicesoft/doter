// Package prefs stores the small set of backend preferences that the
// dashboard and the clipboard watcher need before the webview exists.
package prefs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"doter/internal/clipboard"
)

// Settings is the persisted backend configuration.
type Settings struct {
	// AlertsEnabled controls desktop notifications from the alert engine.
	AlertsEnabled bool `json:"alertsEnabled"`
	// ClipboardPaused stops new clipboard entries without clearing history.
	ClipboardPaused bool `json:"clipboardPaused"`
	// ClipboardHotkeyEnabled controls the global clipboard shortcut.
	ClipboardHotkeyEnabled bool `json:"clipboardHotkeyEnabled"`
	// ClipboardHistoryLimit bounds how many entries are kept.
	ClipboardHistoryLimit int `json:"clipboardHistoryLimit"`
	// ClipboardPreviewBytes is the largest text shipped to the webview.
	ClipboardPreviewBytes int `json:"clipboardPreviewBytes"`
	// ClipboardImageMaxBytes rejects larger images at capture time.
	ClipboardImageMaxBytes int64 `json:"clipboardImageMaxBytes"`
	// ClipboardAutoPaste pastes the chosen entry into the previous app.
	ClipboardAutoPaste bool `json:"clipboardAutoPaste"`
	// ClipboardExcludedApps lists bundle ids or process names never recorded.
	ClipboardExcludedApps []string `json:"clipboardExcludedApps"`
	// Autostart launches Doter at login.
	Autostart bool `json:"autostart"`
	// HideOnFullscreen hides the island while another app is fullscreen.
	HideOnFullscreen bool `json:"hideOnFullscreen"`
}

// Default returns the defaults for a fresh install.
func Default() Settings {
	return Settings{
		AlertsEnabled:          true,
		ClipboardPaused:        false,
		ClipboardHotkeyEnabled: true,
		ClipboardHistoryLimit:  clipboard.DefaultMaxEntries,
		ClipboardPreviewBytes:  clipboard.DefaultPreviewBytes,
		ClipboardImageMaxBytes: clipboard.DefaultImageMaxBytes,
		ClipboardAutoPaste:     false,
		ClipboardExcludedApps:  clipboard.DefaultExcludedApps(),
		Autostart:              false,
		HideOnFullscreen:       true,
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
	s.settings = normalizeSettings(loaded)
	return s
}

// Get returns a copy of the current settings.
func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	settings := s.settings
	settings.ClipboardExcludedApps = append([]string(nil), s.settings.ClipboardExcludedApps...)
	return settings
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

// SetClipboardHistoryLimit changes the number of entries kept and clamps the
// value to the supported range.
func (s *Store) SetClipboardHistoryLimit(limit int) Settings {
	limit = clampInt(limit, clipboard.MinMaxEntries, clipboard.MaxMaxEntries)
	return s.update(func(settings *Settings) {
		settings.ClipboardHistoryLimit = limit
	})
}

// SetClipboardPreviewBytes changes the preview text size and clamps it.
func (s *Store) SetClipboardPreviewBytes(bytes int) Settings {
	bytes = clampInt(bytes, clipboard.MinPreviewBytes, clipboard.MaxPreviewBytes)
	return s.update(func(settings *Settings) {
		settings.ClipboardPreviewBytes = bytes
	})
}

// SetClipboardImageMaxBytes changes the image cap and clamps it.
func (s *Store) SetClipboardImageMaxBytes(bytes int64) Settings {
	bytes = clampInt64(bytes, clipboard.MinImageMaxBytes, clipboard.MaxImageMaxBytes)
	return s.update(func(settings *Settings) {
		settings.ClipboardImageMaxBytes = bytes
	})
}

// SetClipboardAutoPaste toggles pasting the selected entry into the previous
// app.
func (s *Store) SetClipboardAutoPaste(enabled bool) Settings {
	return s.update(func(settings *Settings) {
		settings.ClipboardAutoPaste = enabled
	})
}

// SetClipboardExcludedApps replaces the exclusion list. Blank entries are
// dropped, duplicates are removed case-insensitively, and the list is capped.
func (s *Store) SetClipboardExcludedApps(apps []string) Settings {
	apps = normalizeApps(apps)
	return s.update(func(settings *Settings) {
		settings.ClipboardExcludedApps = apps
	})
}

// SetAutostart records the login-item state.
func (s *Store) SetAutostart(enabled bool) Settings {
	return s.update(func(settings *Settings) {
		settings.Autostart = enabled
	})
}

// SetHideOnFullscreen toggles hiding the island over fullscreen apps.
func (s *Store) SetHideOnFullscreen(enabled bool) Settings {
	return s.update(func(settings *Settings) {
		settings.HideOnFullscreen = enabled
	})
}

func (s *Store) update(fn func(*Settings)) Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.settings)
	s.saveLocked()
	return s.settingsCopyLocked()
}

func (s *Store) settingsCopyLocked() Settings {
	settings := s.settings
	settings.ClipboardExcludedApps = append([]string(nil), s.settings.ClipboardExcludedApps...)
	return settings
}

// normalizeSettings clamps hand-edited values so the runtime never sees an
// out-of-range limit or an unbounded exclusion list.
func normalizeSettings(settings Settings) Settings {
	settings.ClipboardHistoryLimit = clampInt(settings.ClipboardHistoryLimit, clipboard.MinMaxEntries, clipboard.MaxMaxEntries)
	settings.ClipboardPreviewBytes = clampInt(settings.ClipboardPreviewBytes, clipboard.MinPreviewBytes, clipboard.MaxPreviewBytes)
	settings.ClipboardImageMaxBytes = clampInt64(settings.ClipboardImageMaxBytes, clipboard.MinImageMaxBytes, clipboard.MaxImageMaxBytes)
	if settings.ClipboardExcludedApps == nil {
		settings.ClipboardExcludedApps = clipboard.DefaultExcludedApps()
	} else {
		settings.ClipboardExcludedApps = normalizeApps(settings.ClipboardExcludedApps)
	}
	return settings
}

const maxExcludedApps = 64

func normalizeApps(apps []string) []string {
	result := make([]string, 0, len(apps))
	seen := make(map[string]struct{}, len(apps))
	for _, app := range apps {
		app = strings.TrimSpace(app)
		if app == "" {
			continue
		}
		key := strings.ToLower(app)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, app)
		if len(result) == maxExcludedApps {
			break
		}
	}
	return result
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampInt64(value, min, max int64) int64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
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
