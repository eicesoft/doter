package services

import (
	"path/filepath"
	"testing"

	"doter/internal/clipboard"
	"doter/internal/prefs"
)

func TestSettingsServicePersistsAndAppliesClipboardPause(t *testing.T) {
	dir := t.TempDir()
	store := prefs.New(filepath.Join(dir, "settings.json"))
	watcher := clipboard.NewWatcher(clipboard.NewHistory(filepath.Join(dir, "clipboard.json")))
	var hotkeyEnabled bool
	service := NewSettingsService(store, watcher, nil, func(enabled bool) {
		hotkeyEnabled = enabled
	})

	service.SetClipboardPaused(true)
	if !watcher.Paused() {
		t.Fatal("watcher was not paused")
	}
	if !store.Get().ClipboardPaused {
		t.Fatal("pause flag was not persisted")
	}

	service.SetClipboardPaused(false)
	if watcher.Paused() {
		t.Fatal("watcher was not resumed")
	}

	service.SetAlertsEnabled(false)
	if store.Get().AlertsEnabled {
		t.Fatal("alerts flag was not persisted")
	}

	service.SetClipboardHotkeyEnabled(false)
	if hotkeyEnabled {
		t.Fatal("hotkey callback did not receive the new state")
	}
	if store.Get().ClipboardHotkeyEnabled {
		t.Fatal("hotkey flag was not persisted")
	}
}
