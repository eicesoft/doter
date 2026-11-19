//go:build windows || linux

package clipboard

import (
	"bytes"
	"context"
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
// Wayland. Rich pasteboard formats stay macOS-only; Windows and Linux get
// text and PNG images, which cover the common copy/paste flows.

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

func frontmostApp() appInfo {
	// Neither backend exposes the source application reliably. The sensitive
	// marker still covers password managers on both platforms.
	return appInfo{}
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

// Rich pasteboard formats are not captured on Windows/Linux yet. Returning
// no blob makes WriteBack fall back to plain text, which is lossless for the
// text itself.
func readFormats() (blob []byte, rich bool, ok bool) {
	return nil, false, false
}

func WriteFormats([]byte) bool {
	return false
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
