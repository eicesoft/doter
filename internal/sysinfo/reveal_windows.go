//go:build windows

package sysinfo

import "os/exec"

func revealFile(path string) bool {
	cmd := exec.Command("explorer.exe", "/select,"+path)
	if err := cmd.Start(); err != nil {
		return false
	}
	go cmd.Wait()
	return true
}
