//go:build windows

package clipboard

import (
	"strings"
	"syscall"
	"unsafe"

	"github.com/shirou/gopsutil/v4/process"
	libclip "golang.design/x/clipboard"
)

var (
	clipUser32                    = syscall.NewLazyDLL("user32.dll")
	getClipboardSequenceNumber    = clipUser32.NewProc("GetClipboardSequenceNumber")
	procGetClipboardOwner         = clipUser32.NewProc("GetClipboardOwner")
	procClipWindowThreadProcessID = clipUser32.NewProc("GetWindowThreadProcessId")
)

// changeCount uses the native clipboard sequence number, which increments on
// every write and costs one user32 call.
func changeCount() int64 {
	value, _, _ := getClipboardSequenceNumber.Call()
	return int64(value)
}

// frontmostApp uses the clipboard owner rather than the foreground window:
// the owner is still the app that copied after focus has moved elsewhere.
func frontmostApp() appInfo {
	hwnd, _, _ := procGetClipboardOwner.Call()
	if hwnd == 0 {
		return appInfo{}
	}
	var pid uint32
	procClipWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return appInfo{}
	}

	name, err := process.NewProcess(int32(pid))
	if err != nil {
		return appInfo{}
	}
	exeName, err := name.Name()
	if err != nil || exeName == "" {
		return appInfo{}
	}
	return appInfo{ID: strings.ToLower(exeName), Name: exeName}
}

// htmlFormat reads and writes CF_HTML, whose bytes are a header-wrapped HTML
// fragment rather than raw text/html. The library keeps custom formats verbatim,
// so the header survives a copy/paste round-trip.
func htmlFormat() libclip.Format {
	return libclip.Register("HTML Format")
}
