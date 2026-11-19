//go:build linux

package clipboard

import (
	"context"
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
