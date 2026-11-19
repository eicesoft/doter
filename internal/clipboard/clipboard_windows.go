//go:build windows

package clipboard

import "syscall"

var getClipboardSequenceNumber = syscall.NewLazyDLL("user32.dll").NewProc("GetClipboardSequenceNumber")

// changeCount uses the native clipboard sequence number, which increments on
// every write and costs one user32 call.
func changeCount() int64 {
	value, _, _ := getClipboardSequenceNumber.Call()
	return int64(value)
}
