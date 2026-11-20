package clipboard

// RememberTarget records the application that should receive an automatic
// paste. The watcher calls it when the clipboard changes; opening the clipboard
// tab calls it once more before the island takes focus.
func RememberTarget() {
	rememberTarget()
}

// Paste sends the platform paste shortcut to the remembered application.
// It returns an error when the platform has no synthetic-input support or the
// required permission is missing.
func Paste() error {
	return pasteTarget()
}
