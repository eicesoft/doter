//go:build (!darwin && !windows && !linux) || (darwin && !cgo)

package fullscreen

func active() bool {
	return false
}
