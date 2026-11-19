package clipboard

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"strings"
	"sync/atomic"
	"time"
)

// PollInterval is how often the pasteboard's change counter is checked.
const PollInterval = 400 * time.Millisecond

// MaxImageBytes bounds what is stored. Screenshots are a few megabytes;
// anything larger is not something worth keeping in a history.
const MaxImageBytes = 16 << 20

// ThumbnailSize is the longest side, in pixels, of the preview shipped to the
// frontend for an image entry.
const ThumbnailSize = 320

// Watcher polls the pasteboard and records new content into a history.
type Watcher struct {
	history *History
	updates chan Entry
	// lastSeen is the change counter of the last captured pasteboard
	// state. It starts at -1 so the first poll captures whatever is
	// already on the pasteboard.
	lastSeen int64
	paused   atomic.Bool
}

// appInfo describes the frontmost application at copy time.
type appInfo struct {
	ID   string
	Name string
}

// excludedSources are apps whose pasteboard content is never recorded, as a
// second line of defence for managers that do not set the standard macOS
// concealed/transient markers.
var excludedSources = map[string]bool{
	"com.1password.1password":    true,
	"com.agilebits.onepassword7": true,
	"com.apple.keychainaccess":   true,
	"com.apple.passwords":        true,
	"com.bitwarden.desktop":      true,
	"com.dashlane.dashlane":      true,
	"com.lastpass.lastpass":      true,
	"org.keepassxc.keepassxc":    true,
}

// NewWatcher creates a watcher over the given history.
func NewWatcher(history *History) *Watcher {
	return &Watcher{
		history:  history,
		updates:  make(chan Entry, 8),
		lastSeen: -1,
	}
}

// Updates yields an entry whenever the pasteboard changes.
func (w *Watcher) Updates() <-chan Entry { return w.updates }

// SetPaused controls whether new pasteboard content is recorded. The change
// counter is still advanced while paused, so resuming does not immediately
// capture whatever the user copied in the meantime.
func (w *Watcher) SetPaused(paused bool) { w.paused.Store(paused) }

// Paused reports whether capture is currently paused.
func (w *Watcher) Paused() bool { return w.paused.Load() }

// Start begins polling until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.poll()
			}
		}
	}()
}

// poll reads the pasteboard when its change counter moved.
func (w *Watcher) poll() {
	count := changeCount()
	if count < 0 || count == w.lastSeen {
		return
	}
	w.lastSeen = count

	if w.Paused() || isSensitive() {
		return
	}
	source := frontmostApp()
	if isExcludedSource(source) {
		return
	}

	if entry, ok := w.captureImage(source.Name); ok {
		w.emit(entry)
		return
	}

	content, ok := readText()
	if !ok || strings.TrimSpace(content) == "" {
		return
	}

	formats, rich, _ := readFormats()
	entry, changed := w.history.AddRich(content, classify(content, hasURL()), formats, rich, source.Name)
	if !changed {
		return
	}
	w.emit(entry)
}

// isExcludedSource reports whether the frontmost app is a password manager or
// another app whose clipboard content must never be recorded.
func isExcludedSource(source appInfo) bool {
	return source.ID != "" && excludedSources[strings.ToLower(source.ID)]
}

// captureImage records the pasteboard image, if that is its primary content.
func (w *Watcher) captureImage(source string) (Entry, bool) {
	if !imagePrimary() {
		return Entry{}, false
	}
	data, ok := readImage()
	if !ok || len(data) > MaxImageBytes {
		return Entry{}, false
	}
	thumbnail, ok := thumbnailPNG(data, ThumbnailSize)
	if !ok {
		return Entry{}, false
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Entry{}, false
	}
	return w.history.AddImage(
		data,
		"data:image/png;base64,"+base64.StdEncoding.EncodeToString(thumbnail),
		cfg.Width,
		cfg.Height,
		source,
	)
}

// emit hands an entry to the consumer, dropping it when the consumer is
// behind rather than queueing.
func (w *Watcher) emit(entry Entry) {
	select {
	case w.updates <- entry:
	default:
	}
}

// classify guesses what kind of content was copied. The pasteboard's own URL
// flag is authoritative; the text heuristics only cover the cases where the
// flag is missing.
func classify(content string, pasteboardURL bool) string {
	trimmed := strings.TrimSpace(content)
	if pasteboardURL || isURL(trimmed) {
		return "url"
	}
	if isColor(trimmed) {
		return "color"
	}
	return "text"
}

// isURL reports whether the text looks like an http(s) URL.
func isURL(s string) bool {
	lower := strings.ToLower(s)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return false
	}
	// A URL needs a host after the scheme.
	rest := s[strings.Index(s, "//")+2:]
	return rest != "" && !strings.ContainsAny(rest, " \t\n")
}

// isColor reports whether the text is a hex colour such as "#ff00aa".
func isColor(s string) bool {
	if !strings.HasPrefix(s, "#") || len(s) != 7 {
		return false
	}
	for _, c := range s[1:] {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
