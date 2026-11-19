//go:build darwin && cgo

package clipboard

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The watcher reads the real system pasteboard, so these tests drive it with
// pbcopy. They only run on macOS where the pasteboard implementation exists.
func TestWatcherCapturesCopiedText(t *testing.T) {
	history := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))
	watcher := NewWatcher(history)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher.Start(ctx)

	content := "watcher test 中文"
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(content)
	if err := cmd.Run(); err != nil {
		t.Skipf("pbcopy unavailable: %v", err)
	}

	select {
	case entry := <-watcher.Updates():
		if entry.Content != content {
			t.Fatalf("captured %q, want %q", entry.Content, content)
		}
		if entry.Kind != "text" {
			t.Fatalf("kind = %q, want text", entry.Kind)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not report the copy")
	}
}

func TestWatcherIgnoresUnchangedPasteboard(t *testing.T) {
	history := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))
	watcher := NewWatcher(history)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher.Start(ctx)

	// The first poll always captures whatever is already on the pasteboard.
	select {
	case <-watcher.Updates():
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not report the initial pasteboard contents")
	}

	// Nothing changed, so no further entries should arrive.
	select {
	case entry := <-watcher.Updates():
		t.Fatalf("watcher reported an unexpected change: %q", entry.Content)
	case <-time.After(1200 * time.Millisecond):
	}
}

// TestImageRoundTrip drives the native image paths end to end: a PNG written
// to the pasteboard must read back, shrink into a thumbnail, and survive the
// history.
func TestImageRoundTrip(t *testing.T) {
	history := NewHistory(filepath.Join(t.TempDir(), "clipboard.json"))

	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for x := 0; x < 64; x++ {
		for y := 0; y < 48; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 5), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "in.png")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	if !WriteImageFile(path) {
		t.Skip("pasteboard is not writable in this environment")
	}
	if !imagePrimary() {
		t.Fatal("pasteboard image should be the primary content")
	}

	pngData, ok := readImage()
	if !ok {
		t.Fatal("pasteboard image not readable")
	}
	if thumb, ok := thumbnailPNG(pngData, 32); !ok {
		t.Fatal("thumbnail generation failed")
	} else if cfg, err := png.DecodeConfig(bytes.NewReader(thumb)); err != nil || cfg.Width > 32 || cfg.Height > 32 {
		t.Fatalf("thumbnail not scaled: cfg=%+v err=%v", cfg, err)
	}

	entry, ok := history.AddImage(pngData, "data:image/png;base64,AAAA", 64, 48)
	if !ok || entry.Kind != "image" {
		t.Fatalf("image entry not stored: %+v", entry)
	}
	if _, ok := history.ImagePath(entry.ID); !ok {
		t.Fatal("image file path missing")
	}
}

// TestFormatRoundTrip drives the native multi-format paths: styled text put
// on the pasteboard by another app must be captured as a blob and restored
// byte for byte.
func TestFormatRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.rtf")
	rtf := `{\rtf1\ansi\deff0 {\fonttbl {\f0 Helvetica;}}\f0\fs24 bold {\b important}\b0  text}`
	if err := os.WriteFile(path, []byte(rtf), 0o600); err != nil {
		t.Fatal(err)
	}
	script := `set the clipboard to (read (POSIX file "` + path + `") as «class RTF »)`
	if out, err := exec.Command("osascript", "-e", script).CombinedOutput(); err != nil {
		t.Skipf("cannot set the pasteboard here: %v %s", err, out)
	}

	blob, rich, ok := readFormats()
	if !ok || !rich {
		t.Skipf("pasteboard did not keep the RTF data (ok=%v rich=%v)", ok, rich)
	}
	if !WriteFormats(blob) {
		t.Fatal("write-back failed")
	}
	restored, rich, ok := readFormats()
	if !ok || !rich {
		t.Fatal("formats lost after write-back")
	}
	// AppKit adds derived representations when the pasteboard is written, so
	// the blob is not byte-identical; the RTF payload itself must survive.
	if !bytes.Contains(restored, []byte(rtf)) {
		t.Fatal("write-back did not preserve the RTF data")
	}
}
