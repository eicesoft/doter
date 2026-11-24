//go:build !darwin || !cgo

package notch

import "unsafe"

// watchHover is a no-op where the native pointer poll is unavailable.
func watchHover(nsWindow unsafe.Pointer, bottomInset func() float64, handler func(bool)) {}
