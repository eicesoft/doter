// Package clipboard keeps a rolling history of the system pasteboard.
//
// macOS has no system-wide clipboard history: the pasteboard is a single
// slot that is overwritten on every copy. The history is therefore built
// here, by polling the pasteboard's change counter and recording what
// appears.
package clipboard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"
)

// MaxHistory is the number of entries kept. Older entries are dropped.
const MaxHistory = 100

// MaxContentBytes is the largest content shipped to the frontend. The full
// text stays in the Go side so writing an entry back is lossless.
const MaxContentBytes = 4096

// Entry is one clipboard item.
type Entry struct {
	// ID is a stable hash of the content, so the same text keeps the same
	// React key and can be de-duplicated.
	ID string `json:"id"`
	// Content is the clipboard text, truncated to MaxContentBytes.
	Content string `json:"content"`
	// Kind is "text", "url", "color" or "image". Image entries carry a
	// small PNG thumbnail as Content; the full image lives on disk.
	Kind string `json:"kind"`
	// Pinned marks entries the user wants to keep at the top.
	Pinned bool `json:"pinned"`
	// CreatedAt is when the entry was first captured, in milliseconds.
	CreatedAt int64 `json:"createdAt"`
	// ContentLen is the full length of the captured text in bytes.
	ContentLen int `json:"contentLen"`
	// Width and Height are the image size in pixels, zero for text.
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
	// Rich marks entries that kept styled representations (RTF, HTML, …)
	// and so paste back with their original formatting.
	Rich bool `json:"rich,omitempty"`
}

// storedEntry is the on-disk shape. The full content is kept so a restart
// can restore the history without losing anything.
type storedEntry struct {
	Entry
	Full string `json:"full"`
	// ImagePath is where an image entry's full-size PNG is stored.
	ImagePath string `json:"imagePath,omitempty"`
	// Formats is the opaque pasteboard blob captured alongside the text, so
	// a write-back restores the source app's original data and styling.
	Formats []byte `json:"formats,omitempty"`
}

// History is the clipboard history with its persisted backing file.
type History struct {
	mu sync.RWMutex
	// entries holds the newest first. Pinned entries always sort ahead of
	// the rest.
	entries []storedEntry
	path    string
	// imageDir holds the full-size PNGs of image entries, named by entry id.
	imageDir string
}

// NewHistory creates a history and loads any previously saved entries.
func NewHistory(path string) *History {
	h := &History{
		path:     path,
		imageDir: filepath.Join(filepath.Dir(path), "clipboard-images"),
	}
	h.load()
	return h
}

// List returns the entries: pinned ones first, then the rest by recency.
func (h *History) List() []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]Entry, 0, len(h.entries))
	for _, e := range h.entries {
		if e.Pinned {
			result = append(result, e.Entry)
		}
	}
	for _, e := range h.entries {
		if !e.Pinned {
			result = append(result, e.Entry)
		}
	}
	return result
}

// Add records a new clipboard value. It reports whether the history changed;
// a repeat of the current top entry only bumps that entry to the top.
func (h *History) Add(content string, kind string) (Entry, bool) {
	return h.AddRich(content, kind, nil, false)
}

// AddRich records a clipboard value along with the pasteboard's extra
// representations (RTF, HTML, …), which a write-back restores as-is so the
// content pastes into Word, Excel and browsers with its original styling.
func (h *History) AddRich(content, kind string, formats []byte, rich bool) (Entry, bool) {
	if content == "" {
		return Entry{}, false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	return h.insertLocked(storedEntry{
		Entry: Entry{
			ID:         hashID(content),
			Content:    truncate(content, MaxContentBytes),
			Kind:       kind,
			CreatedAt:  time.Now().UnixMilli(),
			ContentLen: len(content),
			Rich:       rich,
		},
		Full:    content,
		Formats: formats,
	}), true
}

// AddImage records a copied image. The PNG is written to the image directory
// and only its thumbnail (a data URL) is shipped to the frontend.
func (h *History) AddImage(png []byte, thumbnail string, width, height int) (Entry, bool) {
	if len(png) == 0 {
		return Entry{}, false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	id := hashBytes(png)
	path := filepath.Join(h.imageDir, id+".png")
	if err := os.MkdirAll(h.imageDir, 0o755); err != nil {
		return Entry{}, false
	}
	// The name is the content hash, so an existing file already holds these
	// bytes and rewriting a multi-megabyte PNG would be wasted work.
	if _, err := os.Stat(path); err != nil {
		if err := os.WriteFile(path, png, 0o600); err != nil {
			return Entry{}, false
		}
	}

	return h.insertLocked(storedEntry{
		Entry: Entry{
			ID:         id,
			Content:    thumbnail,
			Kind:       "image",
			CreatedAt:  time.Now().UnixMilli(),
			ContentLen: len(png),
			Width:      width,
			Height:     height,
		},
		ImagePath: path,
	}), true
}

// insertLocked moves an entry with the same id to the top, refreshing its
// timestamp, or prepends the given entry.
func (h *History) insertLocked(entry storedEntry) Entry {
	for i, e := range h.entries {
		if e.ID != entry.ID {
			continue
		}
		// Same content copied again: refresh its timestamp and move it to
		// the top, which is how every clipboard manager behaves.
		e.CreatedAt = time.Now().UnixMilli()
		h.entries = append(h.entries[:i], h.entries[i+1:]...)
		h.entries = append([]storedEntry{e}, h.entries...)
		h.trimLocked()
		h.saveLocked()
		return e.Entry
	}

	h.entries = append([]storedEntry{entry}, h.entries...)
	h.trimLocked()
	h.saveLocked()
	return entry.Entry
}

// Pin toggles the pinned flag of an entry. Pinned entries are listed first;
// the stored order always stays by recency, so unpinning restores it.
func (h *History) Pin(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.entries {
		if h.entries[i].ID != id {
			continue
		}
		h.entries[i].Pinned = !h.entries[i].Pinned
		h.saveLocked()
		return true
	}
	return false
}

// Delete removes an entry.
func (h *History) Delete(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.entries {
		if h.entries[i].ID != id {
			continue
		}
		removeImage(h.entries[i])
		h.entries = append(h.entries[:i], h.entries[i+1:]...)
		h.saveLocked()
		return true
	}
	return false
}

// Clear empties the history.
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, e := range h.entries {
		removeImage(e)
	}
	h.entries = nil
	h.saveLocked()
}

// Full returns the untruncated content of an entry.
func (h *History) Full(id string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, e := range h.entries {
		if e.ID == id {
			return e.Full, true
		}
	}
	return "", false
}

// ImagePath returns the file holding an image entry's full-size PNG.
func (h *History) ImagePath(id string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, e := range h.entries {
		if e.ID == id && e.ImagePath != "" {
			return e.ImagePath, true
		}
	}
	return "", false
}

// Formats returns the stored pasteboard representations of an entry.
func (h *History) Formats(id string) ([]byte, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, e := range h.entries {
		if e.ID == id && len(e.Formats) > 0 {
			return e.Formats, true
		}
	}
	return nil, false
}

// removeImage deletes an entry's stored image, if it has one.
func removeImage(e storedEntry) {
	if e.ImagePath != "" {
		_ = os.Remove(e.ImagePath)
	}
}

// trimLocked drops the oldest entries beyond MaxHistory.
func (h *History) trimLocked() {
	if len(h.entries) <= MaxHistory {
		return
	}
	for _, e := range h.entries[MaxHistory:] {
		removeImage(e)
	}
	h.entries = h.entries[:MaxHistory]
}

func (h *History) load() {
	data, err := os.ReadFile(h.path)
	if err != nil {
		return
	}
	var stored []storedEntry
	if err := json.Unmarshal(data, &stored); err != nil {
		return
	}
	h.entries = stored
	h.trimLocked()
}

func (h *History) saveLocked() {
	stored := make([]storedEntry, len(h.entries))
	copy(stored, h.entries)

	data, err := json.Marshal(stored)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(h.path), 0o755); err != nil {
		return
	}
	// Write to a temporary file and rename it, so an interrupted write can
	// never leave a truncated history behind.
	tmp := h.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, h.path)
}

// hashID derives a stable identifier from the content.
func hashID(content string) string {
	return hashBytes([]byte(content))
}

// hashBytes derives a stable identifier from raw content.
func hashBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:8])
}

// truncate shortens a string to at most max bytes without splitting a rune.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	end := 0
	for end < len(s) {
		_, size := utf8.DecodeRuneInString(s[end:])
		if end+size > max {
			break
		}
		end += size
	}
	return s[:end]
}

// sortStable sorts in place, keeping the relative order of equal elements.
func sortStable(items []storedEntry, less func(i, j int) bool) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && less(j, j-1); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

var _ = sortStable
