//go:build linux

package clipboard

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	libclip "golang.design/x/clipboard"
)

var (
	linuxSequence  atomic.Int64
	linuxWatchOnce sync.Once
)

// changeCount bridges the library's format watchers into the poll-based
// Watcher. One watch covers text, images and file lists.
func changeCount() int64 {
	linuxWatchOnce.Do(func() {
		if ensureClipboard() != nil {
			return
		}
		go func() {
			for range libclip.Watch(context.Background()) {
				linuxSequence.Add(1)
			}
		}()
	})
	return linuxSequence.Load()
}

// frontmostApp reports the active X11 window's process name. Wayland-only
// sessions do not expose a cross-process active window without a compositor
// specific protocol, so the source stays empty there.
func frontmostApp() appInfo {
	_, pid := activeWindow()
	if pid <= 0 {
		return appInfo{}
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return appInfo{}
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return appInfo{}
	}
	return appInfo{ID: strings.ToLower(name), Name: name}
}

// htmlFormat maps to the X11/Wayland text/html selection target.
func htmlFormat() libclip.Format {
	return libclip.Register("text/html")
}
