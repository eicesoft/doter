//go:build !darwin || !cgo

package notch

import "github.com/wailsapp/wails/v3/pkg/application"

// cameraHousingWidth is always 0 on platforms without a camera housing.
func cameraHousingWidth() int { return 0 }

// builtinScreenBounds is unavailable on platforms without a built-in display.
func builtinScreenBounds() (application.Rect, bool) {
	return application.Rect{}, false
}
