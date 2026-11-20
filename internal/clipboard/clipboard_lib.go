//go:build windows || linux

package clipboard

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"strings"
	"sync"
	"time"

	libclip "golang.design/x/clipboard"
	"golang.org/x/image/draw"
)

// The golang.design/x/clipboard backend is pure Go on Windows, X11 and
// Wayland. Text, image, HTML and RTF representations are captured; the source
// app comes from the platform-specific helper.

var (
	libInitOnce sync.Once
	libInitErr  error
)

func ensureClipboard() error {
	libInitOnce.Do(func() {
		libInitErr = libclip.Init()
	})
	return libInitErr
}

func readText() (string, bool) {
	if ensureClipboard() != nil {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	data, err := libclip.Read(ctx, libclip.FmtText)
	if err != nil || len(data) == 0 {
		return "", false
	}
	return string(data), true
}

func hasURL() bool {
	// classify already recognises http(s) URLs from the text itself.
	return false
}

func isSensitive() bool {
	if ensureClipboard() != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sensitive, err := libclip.Sensitive(ctx)
	return err == nil && sensitive
}

func imagePrimary() bool {
	// Prefer text when both representations exist: copying rich text or a
	// browser selection often also publishes a rendered image.
	if text, ok := readText(); ok && strings.TrimSpace(text) != "" {
		return false
	}
	if ensureClipboard() != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := libclip.Read(ctx, libclip.FmtImage)
	return err == nil
}

func readImage() ([]byte, bool) {
	if ensureClipboard() != nil {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	data, err := libclip.Read(ctx, libclip.FmtImage)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

// thumbnailPNG scales a PNG down to maxDim on its longest side. The macOS
// implementation uses ImageIO; this one keeps the Windows/Linux path pure Go.
func thumbnailPNG(data []byte, maxDim int) ([]byte, bool) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= maxDim && height <= maxDim {
		return data, true
	}
	scale := float64(maxDim) / float64(width)
	if hscale := float64(maxDim) / float64(height); hscale < scale {
		scale = hscale
	}
	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}

// storedFormats is the platform-independent rich-text payload kept with an
// entry. HTML and RTF are opaque native bytes: Windows stores CF_HTML, X11
// and Wayland store text/html, and both round-trip unchanged.
type storedFormats struct {
	Text string `json:"text,omitempty"`
	HTML []byte `json:"html,omitempty"`
	RTF  []byte `json:"rtf,omitempty"`
}

// maxRichBytes keeps a pathological clipboard (a whole web archive) from
// bloating the history file.
const maxRichBytes = 1 << 20

func rtfFormat() libclip.Format {
	return libclip.Register("text/rtf")
}

// readFormats captures styled representations. Plain text is included so
// WriteFormats can publish all representations in one clipboard write.
func readFormats() (blob []byte, rich bool, ok bool) {
	if ensureClipboard() != nil {
		return nil, false, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var formats storedFormats
	if text, err := libclip.Read(ctx, libclip.FmtText); err == nil {
		formats.Text = string(text)
	}
	if html, err := libclip.Read(ctx, htmlFormat()); err == nil && len(html) <= maxRichBytes {
		formats.HTML = html
	}
	if rtf, err := libclip.Read(ctx, rtfFormat()); err == nil && len(rtf) <= maxRichBytes {
		formats.RTF = rtf
	}
	if len(formats.HTML) == 0 && len(formats.RTF) == 0 {
		return nil, false, false
	}

	data, err := json.Marshal(formats)
	if err != nil {
		return nil, false, false
	}
	return data, true, true
}

// WriteFormats publishes the plain, HTML and RTF representations together so
// each consumer picks the richest type it understands.
func WriteFormats(blob []byte) bool {
	if len(blob) == 0 || ensureClipboard() != nil {
		return false
	}
	var formats storedFormats
	if err := json.Unmarshal(blob, &formats); err != nil {
		return false
	}

	items := make([]libclip.Option, 0, 3)
	if len(formats.HTML) > 0 {
		items = append(items, libclip.Item{Format: htmlFormat(), Bytes: formats.HTML})
	}
	if len(formats.RTF) > 0 {
		items = append(items, libclip.Item{Format: rtfFormat(), Bytes: formats.RTF})
	}
	if formats.Text != "" {
		items = append(items, libclip.Item{Format: libclip.FmtText, Bytes: []byte(formats.Text)})
	}
	if len(items) == 0 {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := libclip.WriteAll(ctx, items...)
	return err == nil
}

func WriteImageFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || ensureClipboard() != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = libclip.Write(ctx, libclip.FmtImage, data)
	return err == nil
}
