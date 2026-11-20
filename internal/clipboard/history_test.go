package clipboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoryAddDeduplicatesAndMovesToTop(t *testing.T) {
	h := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))

	first, _ := h.Add("hello", "text")
	second, _ := h.Add("world", "text")
	if h.List()[0].ID != second.ID {
		t.Fatalf("newest entry should be first, got %s", h.List()[0].ID)
	}

	// Copying the same text again must not create a duplicate, and must
	// move that entry back to the top.
	h.Add("hello", "text")
	list := h.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}
	if list[0].ID != first.ID {
		t.Fatalf("re-copied entry should be first, got %s", list[0].ID)
	}
}

func TestHistoryPinKeepsEntryOnTop(t *testing.T) {
	h := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))

	pinned, _ := h.Add("pinned", "text")
	h.Add("newer", "text")

	if !h.Pin(pinned.ID) {
		t.Fatal("pin should succeed")
	}
	if h.List()[0].ID != pinned.ID {
		t.Fatal("pinned entry should be first")
	}

	h.Pin(pinned.ID)
	if h.List()[0].ID == pinned.ID {
		t.Fatal("unpinned entry should not stay first")
	}
}

func TestHistoryPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard.json")

	h := NewHistory(path)
	entry, _ := h.Add("persisted 中文", "text")

	reloaded := NewHistory(path)
	list := reloaded.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 entry after reload, got %d", len(list))
	}
	if list[0].ID != entry.ID {
		t.Fatal("entry id changed after reload")
	}
	if full, ok := reloaded.Full(entry.ID); !ok || full != "persisted 中文" {
		t.Fatalf("full content lost, got %q ok=%v", full, ok)
	}
}

func TestHistoryTruncatesContentButKeepsFullCopy(t *testing.T) {
	h := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))

	long := ""
	for i := 0; i < DefaultPreviewBytes+100; i++ {
		long += "a"
	}
	entry, _ := h.Add(long, "text")

	if len(entry.Content) != DefaultPreviewBytes {
		t.Fatalf("content should be truncated to %d bytes, got %d", DefaultPreviewBytes, len(entry.Content))
	}
	if entry.ContentLen != len(long) {
		t.Fatalf("content length should be %d, got %d", len(long), entry.ContentLen)
	}
	full, _ := h.Full(entry.ID)
	if len(full) != len(long) {
		t.Fatalf("full content should be kept, got %d bytes", len(full))
	}
}

func TestHistoryDeleteAndClear(t *testing.T) {
	h := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))
	entry, _ := h.Add("gone", "text")

	if !h.Delete(entry.ID) {
		t.Fatal("delete should succeed")
	}
	if len(h.List()) != 0 {
		t.Fatal("entry should be gone")
	}

	h.Add("a", "text")
	h.Clear()
	if len(h.List()) != 0 {
		t.Fatal("history should be empty")
	}
	if _, err := os.Stat(h.path); err != nil {
		t.Fatal("history file should still exist after clearing")
	}
}

func TestHistoryCapsEntries(t *testing.T) {
	h := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))
	for i := 0; i < DefaultMaxEntries+10; i++ {
		h.Add(string(rune('a'+i%26))+string(rune(i)), "text")
	}
	if got := len(h.List()); got > DefaultMaxEntries {
		t.Fatalf("history should hold at most %d entries, got %d", DefaultMaxEntries, got)
	}
}

func TestHistoryLimitsAreConfigurable(t *testing.T) {
	h := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))
	for i := range 20 {
		h.Add(string(rune('a'+i)), "text")
	}

	h.SetLimits(5, 512)
	entries := h.List()
	if len(entries) != 5 {
		t.Fatalf("configured cap should keep 5 entries, got %d", len(entries))
	}

	long := strings.Repeat("a", 800)
	entry, _ := h.Add(long, "text")
	if len(entry.Content) != 512 {
		t.Fatalf("configured preview should be 512 bytes, got %d", len(entry.Content))
	}

	// Enlarging the preview must recover the full stored text, not the old
	// truncated copy.
	h.SetLimits(5, 1024)
	if got := len(h.List()[0].Content); got != 800 {
		t.Fatalf("preview increase should expose %d bytes, got %d", 800, got)
	}
}

func TestTruncateKeepsValidUTF8(t *testing.T) {
	cut := truncate("中文测试内容", 5)
	for _, r := range cut {
		if r == '�' {
			t.Fatalf("truncation split a rune: %q", cut)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		content string
		url     bool
		want    string
	}{
		{"https://example.com", true, "url"},
		{"http://example.com/path", false, "url"},
		{"#ff00aa", false, "color"},
		{"plain text", false, "text"},
	}
	for _, c := range cases {
		if got := classify(c.content, c.url); got != c.want {
			t.Fatalf("classify(%q) = %q, want %q", c.content, got, c.want)
		}
	}
}

func TestHistoryImagePersistsAndDeletesFile(t *testing.T) {
	h := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))

	png := []byte("\x89PNG\r\n\x1a\n fake bytes")
	entry, ok := h.AddImage(png, "data:image/png;base64,AAAA", 64, 48, "")
	if !ok || entry.Kind != "image" {
		t.Fatalf("image not stored: %+v ok=%v", entry, ok)
	}
	if entry.ContentLen != len(png) {
		t.Fatalf("content length = %d, want %d", entry.ContentLen, len(png))
	}

	imgPath, ok := h.ImagePath(entry.ID)
	if !ok {
		t.Fatal("image path missing")
	}
	if data, err := os.ReadFile(imgPath); err != nil || string(data) != string(png) {
		t.Fatalf("image file not written: err=%v", err)
	}

	// The path must survive a restart and deleting the entry must remove the
	// file so images do not pile up on disk.
	reloaded := NewHistory(h.path)
	if _, ok := reloaded.ImagePath(entry.ID); !ok {
		t.Fatal("image path lost after reload")
	}
	if !reloaded.Delete(entry.ID) {
		t.Fatal("delete should succeed")
	}
	if _, err := os.Stat(imgPath); !os.IsNotExist(err) {
		t.Fatalf("image file should be removed, stat err=%v", err)
	}
}

func TestHistoryClearRemovesImageFiles(t *testing.T) {
	h := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))
	entry, _ := h.AddImage([]byte("image bytes"), "thumb", 1, 1, "")
	imgPath, _ := h.ImagePath(entry.ID)

	h.Clear()
	if _, err := os.Stat(imgPath); !os.IsNotExist(err) {
		t.Fatalf("image file should be removed by Clear, stat err=%v", err)
	}
}

func TestHistoryPersistsSourceApp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboard.json")
	h := NewHistory(path)
	if _, ok := h.AddRich("from safari", "text", nil, false, "Safari"); !ok {
		t.Fatal("entry was not stored")
	}
	reloaded := NewHistory(path)
	if got := reloaded.List(); len(got) != 1 || got[0].Source != "Safari" {
		t.Fatalf("source not persisted: %+v", got)
	}
}

func TestHistoryRichFormatsPersist(t *testing.T) {
	h := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))

	blob := []byte("opaque pasteboard blob")
	entry, ok := h.AddRich("styled text", "text", blob, true, "")
	if !ok || !entry.Rich {
		t.Fatalf("rich entry not stored: %+v ok=%v", entry, ok)
	}

	reloaded := NewHistory(h.path)
	got, ok := reloaded.Formats(entry.ID)
	if !ok || string(got) != string(blob) {
		t.Fatalf("formats lost after reload: %q ok=%v", got, ok)
	}
	if list := reloaded.List(); len(list) != 1 || !list[0].Rich {
		t.Fatalf("rich flag lost after reload: %+v", list)
	}

	// Plain entries have no formats to restore.
	plain, _ := reloaded.Add("plain", "text")
	if _, ok := reloaded.Formats(plain.ID); ok {
		t.Fatal("plain entry should have no formats")
	}
}
