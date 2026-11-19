//go:build !darwin || !cgo

package notch

import "unsafe"

// setNativeGlass is a no-op where the host has no behind-window vibrancy.
func setNativeGlass(nsWindow unsafe.Pointer, visible bool) {}
