//go:build darwin

package sysinfo

import (
	"os/exec"
	"strings"
)

func revealFile(path string) bool {
	// For an app bundle, reveal the .app rather than the binary deep inside
	// Contents/MacOS.
	if index := strings.Index(strings.ToLower(path), ".app/"); index >= 0 {
		path = path[:index+4]
	}
	cmd := exec.Command("open", "-R", path)
	if err := cmd.Start(); err != nil {
		return false
	}
	go cmd.Wait()
	return true
}
