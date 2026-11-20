package clipboard

import (
	"path/filepath"
	"testing"
)

func TestExcludedSourceMatchesBundleIDCaseInsensitively(t *testing.T) {
	watcher := NewWatcher(NewHistory(filepath.Join(t.TempDir(), "clipboard.json")))
	if !watcher.excludedSource(appInfo{ID: "com.1Password.1password", Name: "1Password"}) {
		t.Fatal("1Password should be excluded")
	}
	if watcher.excludedSource(appInfo{ID: "com.apple.Safari", Name: "Safari"}) {
		t.Fatal("Safari should not be excluded")
	}
	if watcher.excludedSource(appInfo{Name: "Unbundled"}) {
		t.Fatal("an app without a bundle id should not match")
	}
}

func TestExcludedSourceUsesConfiguredList(t *testing.T) {
	watcher := NewWatcher(NewHistory(filepath.Join(t.TempDir(), "clipboard.json")))
	watcher.SetExcludedApps([]string{"  Slack.exe  ", "com.example.Secret"})

	if !watcher.excludedSource(appInfo{ID: "slack", Name: "Slack"}) {
		t.Fatal("configured process name should match without .exe")
	}
	if !watcher.excludedSource(appInfo{ID: "com.example.secret"}) {
		t.Fatal("bundle ids should match case-insensitively")
	}
	if watcher.excludedSource(appInfo{ID: "com.1password.1password"}) {
		t.Fatal("replacing the list must drop the built-in defaults")
	}
}

func TestPauseAccessors(t *testing.T) {
	watcher := NewWatcher(NewHistory(filepath.Join(t.TempDir(), "clipboard.json")))
	if watcher.Paused() {
		t.Fatal("new watcher should not be paused")
	}
	watcher.SetPaused(true)
	if !watcher.Paused() {
		t.Fatal("SetPaused(true) was not observed")
	}
	watcher.SetPaused(false)
	if watcher.Paused() {
		t.Fatal("SetPaused(false) was not observed")
	}
}
