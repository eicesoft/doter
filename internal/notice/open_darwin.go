//go:build darwin

package notice

import (
	"errors"
	"os/exec"
	"strings"
)

// OpenApp activates the app that posted a notification.
func OpenApp(bundleID string) error {
	if bundleID == "" || strings.HasPrefix(bundleID, "_system_center_:") {
		return errors.New("no app to open")
	}
	return exec.Command("open", "-b", bundleID).Run()
}
