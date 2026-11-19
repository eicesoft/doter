//go:build windows

package fullscreen

import (
	"syscall"
	"unsafe"
)

var (
	shell32                      = syscall.NewLazyDLL("shell32.dll")
	shQueryUserNotificationState = shell32.NewProc("SHQueryUserNotificationState")
)

// SHQueryUserNotificationState values that mean the user is watching a
// fullscreen app or presenting.
const (
	qunsBusy                 = 2
	qunsRunningD3DFullScreen = 3
	qunsPresentationMode     = 4
)

func active() bool {
	var state int32
	hr, _, _ := shQueryUserNotificationState.Call(uintptr(unsafe.Pointer(&state)))
	if int32(hr) != 0 {
		return false
	}
	switch state {
	case qunsBusy, qunsRunningD3DFullScreen, qunsPresentationMode:
		return true
	default:
		return false
	}
}
