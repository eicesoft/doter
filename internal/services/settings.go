package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"doter/internal/clipboard"
	"doter/internal/prefs"
)

// SettingsService exposes the backend preferences that affect behaviour
// outside the webview: desktop alerts and clipboard capture.
type SettingsService struct {
	app             *application.App
	store           *prefs.Store
	history         *clipboard.History
	watcher         *clipboard.Watcher
	onAlertsEnabled func()
	onHotkeyChanged func(bool)
}

// NewSettingsService creates the service and applies the persisted clipboard
// settings to the history and watcher before they start working.
// onAlertsEnabled is called when the user turns alerts back on, so the platform
// can ask for notification permission. onHotkeyChanged binds or releases the
// global clipboard shortcut.
func NewSettingsService(
	app *application.App,
	store *prefs.Store,
	history *clipboard.History,
	watcher *clipboard.Watcher,
	onAlertsEnabled func(),
	onHotkeyChanged func(bool),
) *SettingsService {
	service := &SettingsService{
		app:             app,
		store:           store,
		history:         history,
		watcher:         watcher,
		onAlertsEnabled: onAlertsEnabled,
		onHotkeyChanged: onHotkeyChanged,
	}
	if watcher != nil {
		watcher.SetPaused(store.Get().ClipboardPaused)
	}
	service.applyClipboardSettings(store.Get())
	return service
}

// applyClipboardSettings pushes the limits and exclusions into the runtime
// components.
func (s *SettingsService) applyClipboardSettings(settings prefs.Settings) {
	if s.history != nil {
		s.history.SetLimits(settings.ClipboardHistoryLimit, settings.ClipboardPreviewBytes)
	}
	if s.watcher != nil {
		s.watcher.SetImageMaxBytes(settings.ClipboardImageMaxBytes)
		s.watcher.SetExcludedApps(settings.ClipboardExcludedApps)
	}
}

// notifyClipboardConfig tells the open clipboard panel to reload its list after
// a limit or exclusion change.
func (s *SettingsService) notifyClipboardConfig() {
	if s.app != nil {
		s.app.Event.Emit(EventClipboardConfig)
	}
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

// SetClipboardHistoryLimit changes how many entries the history keeps.
func (s *SettingsService) SetClipboardHistoryLimit(limit int) prefs.Settings {
	settings := s.store.SetClipboardHistoryLimit(limit)
	s.applyClipboardSettings(settings)
	s.notifyClipboardConfig()
	return settings
}

// SetClipboardPreviewBytes changes the largest text shipped to the clipboard
// panel.
func (s *SettingsService) SetClipboardPreviewBytes(bytes int) prefs.Settings {
	settings := s.store.SetClipboardPreviewBytes(bytes)
	s.applyClipboardSettings(settings)
	s.notifyClipboardConfig()
	return settings
}

// SetClipboardImageMaxBytes changes the largest image accepted into history.
func (s *SettingsService) SetClipboardImageMaxBytes(bytes int64) prefs.Settings {
	settings := s.store.SetClipboardImageMaxBytes(bytes)
	s.applyClipboardSettings(settings)
	s.notifyClipboardConfig()
	return settings
}

// SetClipboardAutoPaste toggles pasting the selected entry into the previous
// application.
func (s *SettingsService) SetClipboardAutoPaste(enabled bool) prefs.Settings {
	return s.store.SetClipboardAutoPaste(enabled)
}

// SetClipboardExcludedApps replaces the exclusion list.
func (s *SettingsService) SetClipboardExcludedApps(apps []string) prefs.Settings {
	settings := s.store.SetClipboardExcludedApps(apps)
	s.applyClipboardSettings(settings)
	s.notifyClipboardConfig()
	return settings
}

// SetAutostart registers or removes the login item. The persisted value is
// only updated after the platform accepted the change.
func (s *SettingsService) SetAutostart(enabled bool) (prefs.Settings, error) {
	if s.app == nil {
		return s.store.SetAutostart(enabled), nil
	}
	var err error
	if enabled {
		err = s.app.Autostart.Enable()
	} else {
		err = s.app.Autostart.Disable()
	}
	if err != nil {
		return s.store.Get(), err
	}
	return s.store.SetAutostart(enabled), nil
}

// SetHideOnFullscreen toggles hiding the island over fullscreen apps.
func (s *SettingsService) SetHideOnFullscreen(enabled bool) prefs.Settings {
	return s.store.SetHideOnFullscreen(enabled)
}
