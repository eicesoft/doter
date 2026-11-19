//go:build !darwin || !cgo

package clipboard

// changeCount returns the pasteboard's change counter. Platforms without a
// native implementation report -1 so the watcher never sees a change.
func changeCount() int64 {
	return -1
}

// readText returns the pasteboard's plain-text content.
func readText() (string, bool) {
	return "", false
}

// hasURL reports whether the pasteboard holds a URL.
func hasURL() bool {
	return false
}

// isSensitive reports whether the pasteboard is marked as concealed.
func isSensitive() bool {
	return false
}

// frontmostApp returns the app that most likely performed the copy.
func frontmostApp() appInfo {
	return appInfo{}
}

// imagePrimary reports whether the pasteboard's primary content is an image.
func imagePrimary() bool {
	return false
}

// readImage returns the pasteboard image encoded as PNG.
func readImage() ([]byte, bool) {
	return nil, false
}

// thumbnailPNG scales PNG data so its longest side is at most maxDim.
func thumbnailPNG(png []byte, maxDim int) ([]byte, bool) {
	return nil, false
}

// readFormats returns the pasteboard's extra representations as an opaque
// plist blob. rich reports whether styled text is included.
func readFormats() (blob []byte, rich bool, ok bool) {
	return nil, false, false
}

// WriteFormats restores representations captured by readFormats.
func WriteFormats(blob []byte) bool {
	return false
}

// WriteImageFile restores a stored image to the pasteboard.
func WriteImageFile(path string) bool {
	return false
}
