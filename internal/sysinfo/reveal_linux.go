//go:build linux

package sysinfo

import (
	"os/exec"
	"path/filepath"
)

func revealFile(path string) bool {
	cmd := exec.Command("xdg-open", filepath.Dir(path))
	if err := cmd.Start(); err != nil {
		return false
	}
	go cmd.Wait()
	return true
}
