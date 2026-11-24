//go:build !darwin && !windows

package notice

// AppIcon is not implemented on this platform.
func AppIcon(string) string { return "" }
