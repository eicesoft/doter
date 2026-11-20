package prefs

import (
	"os"
	"path/filepath"
	"testing"

	"doter/internal/clipboard"
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

func TestClipboardSettingsClampAndPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := New(path)

	got := store.SetClipboardHistoryLimit(1)
	if got.ClipboardHistoryLimit != clipboard.MinMaxEntries {
		t.Fatalf("history limit = %d, want %d", got.ClipboardHistoryLimit, clipboard.MinMaxEntries)
	}
	store.SetClipboardPreviewBytes(1)
	store.SetClipboardImageMaxBytes(1)
	apps := store.SetClipboardExcludedApps([]string{" Slack ", "slack", "", "1Password"})

	if apps.ClipboardPreviewBytes != clipboard.MinPreviewBytes {
		t.Fatalf("preview bytes = %d", apps.ClipboardPreviewBytes)
	}
	if apps.ClipboardImageMaxBytes != clipboard.MinImageMaxBytes {
		t.Fatalf("image max = %d", apps.ClipboardImageMaxBytes)
	}
	if len(apps.ClipboardExcludedApps) != 2 {
		t.Fatalf("excluded apps = %v", apps.ClipboardExcludedApps)
	}

	reloaded := New(path).Get()
	if reloaded.ClipboardHistoryLimit != clipboard.MinMaxEntries ||
		reloaded.ClipboardPreviewBytes != clipboard.MinPreviewBytes ||
		reloaded.ClipboardImageMaxBytes != clipboard.MinImageMaxBytes ||
		len(reloaded.ClipboardExcludedApps) != 2 {
		t.Fatalf("reloaded clipboard settings = %+v", reloaded)
	}
}

func TestClipboardExcludedAppsCopyIsIsolated(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "settings.json"))
	apps := store.Get().ClipboardExcludedApps
	apps[0] = "mutated"
	if store.Get().ClipboardExcludedApps[0] != clipboard.DefaultExcludedApps()[0] {
		t.Fatal("Get must not expose the backing slice")
	}
}
