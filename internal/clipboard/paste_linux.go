//go:build linux

package clipboard

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	pasteTargetMu     sync.Mutex
	pasteTargetWindow string
	lastRememberAt    atomic.Int64
)

func rememberTarget() {
	// Focus fires on every control hover; keep the X11 round-trip cheap.
	now := time.Now().UnixMilli()
	last := lastRememberAt.Load()
	if now-last < 250 {
		return
	}
	lastRememberAt.Store(now)

	id, pid := activeWindow()
	if id == "" || pid == 0 || pid == os.Getpid() {
		return
	}
	pasteTargetMu.Lock()
	pasteTargetWindow = id
	pasteTargetMu.Unlock()
}

func pasteTarget() error {
	pasteTargetMu.Lock()
	id := pasteTargetWindow
	pasteTargetMu.Unlock()
	if id == "" {
		return errors.New("没有可粘贴的目标窗口")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, "xdotool", "windowactivate", "--sync", id).Run(); err != nil {
		return fmt.Errorf("无法激活目标窗口（需要 xdotool）: %w", err)
	}
	if err := exec.CommandContext(ctx, "xdotool", "key", "--clearmodifiers", "ctrl+v").Run(); err != nil {
		return fmt.Errorf("无法发送粘贴快捷键: %w", err)
	}
	return nil
}

// activeWindow returns the focused X11 window id and its _NET_WM_PID.
// Wayland-only sessions do not expose either, so callers fall back to an empty
// target.
func activeWindow() (string, int) {
	out, err := xprop("-root", "_NET_ACTIVE_WINDOW")
	if err != nil {
		return "", 0
	}
	id := afterMarker(string(out), "#")
	if id == "" || id == "0x0" {
		return "", 0
	}

	out, err = xprop("-id", id, "_NET_WM_PID")
	if err != nil {
		return id, 0
	}
	pid, err := strconv.Atoi(afterMarker(string(out), "="))
	if err != nil {
		return id, 0
	}
	return id, pid
}

func xprop(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "xprop", args...).Output()
}

func afterMarker(output, marker string) string {
	index := strings.LastIndex(output, marker)
	if index < 0 {
		return ""
	}
	return strings.TrimSpace(output[index+len(marker):])
}
