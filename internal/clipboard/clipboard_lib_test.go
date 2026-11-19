//go:build windows || linux

package clipboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestThumbnailPNGScalesToLongestSide(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 800, 400))
	for x := range 800 {
		for y := range 400 {
			src.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	var input bytes.Buffer
	if err := png.Encode(&input, src); err != nil {
		t.Fatal(err)
	}

	out, ok := thumbnailPNG(input.Bytes(), 320)
	if !ok {
		t.Fatal("thumbnail generation failed")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 320 || cfg.Height != 160 {
		t.Fatalf("thumbnail = %dx%d, want 320x160", cfg.Width, cfg.Height)
	}
}
