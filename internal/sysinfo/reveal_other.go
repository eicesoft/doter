//go:build !darwin && !windows && !linux

package sysinfo

func revealFile(string) bool {
	return false
}
