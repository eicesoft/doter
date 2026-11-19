package services

import (
	"doter/internal/clipboard"
	"doter/internal/prefs"
)

// SettingsService exposes the backend preferences that affect behaviour
// outside the webview: desktop alerts and clipboard capture.
type SettingsService struct {
	store           *prefs.Store
	watcher         *clipboard.Watcher
	onAlertsEnabled func()
	onHotkeyChanged func(bool)
}

// NewSettingsService creates the service and applies the persisted clipboard
// pause flag to the watcher before it starts polling. onAlertsEnabled is
// called when the user turns alerts back on, so the platform can ask for
// notification permission. onHotkeyChanged binds or releases the global
// clipboard shortcut.
func NewSettingsService(
	store *prefs.Store,
	watcher *clipboard.Watcher,
	onAlertsEnabled func(),
	onHotkeyChanged func(bool),
) *SettingsService {
	service := &SettingsService{
		store:           store,
		watcher:         watcher,
		onAlertsEnabled: onAlertsEnabled,
		onHotkeyChanged: onHotkeyChanged,
	}
	if watcher != nil {
		watcher.SetPaused(store.Get().ClipboardPaused)
	}
	return service
}

// Get returns the current backend preferences.
func (s *SettingsService) Get() prefs.Settings {
	return s.store.Get()
}

// SetAlertsEnabled toggles desktop notifications from the alert engine.
func (s *SettingsService) SetAlertsEnabled(enabled bool) prefs.Settings {
	settings := s.store.SetAlertsEnabled(enabled)
	if enabled && s.onAlertsEnabled != nil {
		s.onAlertsEnabled()
	}
	return settings
}

// SetClipboardPaused stops or resumes recording new clipboard entries.
func (s *SettingsService) SetClipboardPaused(paused bool) prefs.Settings {
	settings := s.store.SetClipboardPaused(paused)
	if s.watcher != nil {
		s.watcher.SetPaused(paused)
	}
	return settings
}

// SetClipboardHotkeyEnabled binds or releases the global clipboard shortcut.
func (s *SettingsService) SetClipboardHotkeyEnabled(enabled bool) prefs.Settings {
	settings := s.store.SetClipboardHotkeyEnabled(enabled)
	if s.onHotkeyChanged != nil {
		s.onHotkeyChanged(enabled)
	}
	return settings
}
