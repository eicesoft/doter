//go:build !darwin || !cgo

package notch

import "unsafe"

// setNativeGlass is a no-op where the host has no behind-window vibrancy.
func setNativeGlass(nsWindow unsafe.Pointer, visible bool) {}

// setNativeGlassRect is a no-op where the host has no behind-window vibrancy.
func setNativeGlassRect(nsWindow unsafe.Pointer, visible bool, x, top, width, height float64) {}
