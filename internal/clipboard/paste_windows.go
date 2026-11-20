//go:build windows

package clipboard

import (
	"errors"
	"os"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procIsWindow                 = user32.NewProc("IsWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procKeybdEvent               = user32.NewProc("keybd_event")

	pasteTargetWindow atomic.Uintptr
)

const (
	vkControl   = 0x11
	vkV         = 0x56
	keyeventKey = 0x0002
)

func rememberTarget() {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return
	}
	var pid uint32
	procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 || pid == uint32(os.Getpid()) {
		return
	}
	pasteTargetWindow.Store(hwnd)
}

func pasteTarget() error {
	hwnd := pasteTargetWindow.Load()
	if hwnd == 0 {
		return errors.New("没有可粘贴的目标窗口")
	}
	if ok, _, _ := procIsWindow.Call(hwnd); ok == 0 {
		return errors.New("目标窗口已关闭")
	}
	if ok, _, _ := procSetForegroundWindow.Call(hwnd); ok == 0 {
		return errors.New("无法激活目标窗口")
	}

	// Let the target finish its activation transition before the keys arrive.
	time.Sleep(120 * time.Millisecond)
	procKeybdEvent.Call(vkControl, 0, 0, 0)
	procKeybdEvent.Call(vkV, 0, 0, 0)
	procKeybdEvent.Call(vkV, 0, keyeventKey, 0)
	procKeybdEvent.Call(vkControl, 0, keyeventKey, 0)
	return nil
}
