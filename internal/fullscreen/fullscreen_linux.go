//go:build linux

package fullscreen

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// active asks the X11 window manager for the active window and then checks its
// EWMH state. Wayland sessions without XWayland cannot answer this; the island
// simply stays visible there.
func active() bool {
	out, err := xprop("-root", "_NET_ACTIVE_WINDOW")
	if err != nil {
		return false
	}
	id := activeWindowID(string(out))
	if id == "" || id == "0x0" {
		return false
	}

	out, err = xprop("-id", id, "_NET_WM_STATE")
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "_NET_WM_STATE_FULLSCREEN")
}

func xprop(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "xprop", args...).Output()
}

func activeWindowID(output string) string {
	index := strings.LastIndex(output, "#")
	if index < 0 {
		return ""
	}
	return strings.TrimSpace(output[index+1:])
}
