package clipboard

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PollInterval is how often the pasteboard's change counter is checked.
const PollInterval = 400 * time.Millisecond

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

	configMu sync.RWMutex
	excluded map[string]struct{}
	imageMax int64
}

// appInfo describes the frontmost application at copy time.
type appInfo struct {
	ID   string
	Name string
}

// NewWatcher creates a watcher over the given history.
func NewWatcher(history *History) *Watcher {
	w := &Watcher{
		history:  history,
		updates:  make(chan Entry, 8),
		lastSeen: -1,
		imageMax: DefaultImageMaxBytes,
	}
	w.SetExcludedApps(DefaultExcludedApps())
	return w
}

// SetExcludedApps replaces the list of apps whose clipboard content is never
// recorded. Entries match the source app's bundle identifier, process name or
// executable name, case-insensitively.
func (w *Watcher) SetExcludedApps(apps []string) {
	excluded := make(map[string]struct{}, len(apps))
	for _, app := range apps {
		if key := normalizeAppName(app); key != "" {
			excluded[key] = struct{}{}
		}
	}

	w.configMu.Lock()
	w.excluded = excluded
	w.configMu.Unlock()
}

// SetImageMaxBytes bounds what image data is stored.
func (w *Watcher) SetImageMaxBytes(limit int64) {
	if limit <= 0 {
		limit = DefaultImageMaxBytes
	}
	w.configMu.Lock()
	w.imageMax = limit
	w.configMu.Unlock()
}

func (w *Watcher) imageLimit() int64 {
	w.configMu.RLock()
	defer w.configMu.RUnlock()
	if w.imageMax <= 0 {
		return DefaultImageMaxBytes
	}
	return w.imageMax
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
	// Track the app the user copied from, so auto-paste can return focus to
	// it after the island closes.
	rememberTarget()

	if w.Paused() || isSensitive() {
		return
	}
	source := frontmostApp()
	if w.excludedSource(source) {
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

// excludedSource reports whether the source app is on the user's exclusion
// list. The list matches bundle identifiers, process names and executable
// names; .exe is ignored so one entry covers Windows and Unix conventions.
func (w *Watcher) excludedSource(source appInfo) bool {
	w.configMu.RLock()
	defer w.configMu.RUnlock()

	for _, candidate := range []string{source.ID, source.Name} {
		key := normalizeAppName(candidate)
		if key == "" {
			continue
		}
		if _, ok := w.excluded[key]; ok {
			return true
		}
	}
	return false
}

func normalizeAppName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimSuffix(name, ".exe")
	return name
}

// captureImage records the pasteboard image, if that is its primary content.
func (w *Watcher) captureImage(source string) (Entry, bool) {
	if !imagePrimary() {
		return Entry{}, false
	}
	data, ok := readImage()
	if !ok || len(data) > int(w.imageLimit()) {
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
