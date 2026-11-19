package clipboard

import (
	"path/filepath"
	"testing"
)

func TestExcludedSourceMatchesBundleIDCaseInsensitively(t *testing.T) {
	if !isExcludedSource(appInfo{ID: "com.1Password.1password", Name: "1Password"}) {
		t.Fatal("1Password should be excluded")
	}
	if isExcludedSource(appInfo{ID: "com.apple.Safari", Name: "Safari"}) {
		t.Fatal("Safari should not be excluded")
	}
	if isExcludedSource(appInfo{Name: "Unbundled"}) {
		t.Fatal("an app without a bundle id should not match")
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
