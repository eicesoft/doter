// Package fullscreen detects whether the frontmost application is running in
// a fullscreen or presentation state. The island uses it to avoid covering
// video, games and slides.
package fullscreen

// Active reports whether the frontmost application is fullscreen.
func Active() bool {
	return active()
}
