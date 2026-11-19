package prefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsWhenFileIsMissing(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "settings.json"))
	got := store.Get()
	if !got.AlertsEnabled || got.ClipboardPaused || !got.ClipboardHotkeyEnabled ||
		got.Autostart || !got.HideOnFullscreen {
		t.Fatalf("defaults = %+v", got)
	}
}

func TestSetPersistsAcrossReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := New(path)
	store.SetAlertsEnabled(false)
	store.SetClipboardPaused(true)
	store.SetClipboardHotkeyEnabled(false)
	store.SetAutostart(true)
	store.SetHideOnFullscreen(false)

	reloaded := New(path).Get()
	if reloaded.AlertsEnabled || !reloaded.ClipboardPaused || reloaded.ClipboardHotkeyEnabled ||
		!reloaded.Autostart || reloaded.HideOnFullscreen {
		t.Fatalf("reloaded = %+v", reloaded)
	}
}

func TestPartialFileKeepsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"clipboardPaused":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := New(path).Get()
	if !got.AlertsEnabled {
		t.Fatal("missing alertsEnabled should keep the default true")
	}
	if !got.ClipboardPaused {
		t.Fatal("clipboardPaused was not loaded")
	}
	if !got.ClipboardHotkeyEnabled {
		t.Fatal("missing clipboardHotkeyEnabled should keep the default true")
	}
	if got.Autostart || !got.HideOnFullscreen {
		t.Fatalf("missing new fields should keep defaults, got %+v", got)
	}
}

func TestCorruptFileFallsBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{not json`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := New(path).Get()
	if !got.AlertsEnabled || got.ClipboardPaused || !got.ClipboardHotkeyEnabled ||
		got.Autostart || !got.HideOnFullscreen {
		t.Fatalf("corrupt file should use defaults, got %+v", got)
	}
}
