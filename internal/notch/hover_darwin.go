//go:build darwin && cgo

package notch

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static int islandMouseInside(void *nsWindow) {
	NSWindow *window = (NSWindow *)nsWindow;
	if (window == nil || !window.isVisible) {
		return 0;
	}
	NSRect frame = NSInsetRect(window.frame, -2, -2);
	NSPoint p = [NSEvent mouseLocation];
	return NSPointInRect(p, frame) ? 1 : 0;
}
*/
import "C"

import (
	"sync"
	"time"
	"unsafe"
)

var (
	hoverMu      sync.RWMutex
	hoverHandler func(entered bool)
	hoverStarted bool
)

// watchHover reports pointer enter/leave for the window. AppKit only delivers
// hover events to the webview while the window is key, and a non-activating
// panel never becomes key from hovering, so the pointer position is polled
// against the window frame instead.
func watchHover(nsWindow unsafe.Pointer, handler func(bool)) {
	if nsWindow == nil {
		return
	}

	hoverMu.Lock()
	hoverHandler = handler
	alreadyWatching := hoverStarted
	hoverStarted = true
	hoverMu.Unlock()

	if alreadyWatching {
		return
	}

	go func() {
		inside := false
		for {
			nowInside := C.islandMouseInside(nsWindow) != 0
			if nowInside != inside {
				inside = nowInside
				hoverMu.RLock()
				handler := hoverHandler
				hoverMu.RUnlock()
				if handler != nil {
					handler(nowInside)
				}
			}
			time.Sleep(80 * time.Millisecond)
		}
	}()
}
