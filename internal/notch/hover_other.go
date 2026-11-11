//go:build !darwin || !cgo

package notch

import "unsafe"

// watchHover is a no-op where the native pointer poll is unavailable.
func watchHover(nsWindow unsafe.Pointer, handler func(bool)) {}
