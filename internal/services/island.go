package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"doter/internal/notch"
)

// IslandService lets the UI grow and shrink the native window around the
// island animation. Wails serialises these calls onto the UI thread.
type IslandService struct {
	ctrl *notch.Controller
	app  *application.App
}

func NewIslandService(ctrl *notch.Controller, app *application.App) *IslandService {
	return &IslandService{ctrl: ctrl, app: app}
}

// SetExpanded asks the native window to switch between pill and dashboard size.
func (s *IslandService) SetExpanded(expanded bool) bool {
	s.ctrl.SetExpanded(expanded)
	return s.ctrl.Expanded()
}

// ToggleExpanded flips the window size and returns the new state.
func (s *IslandService) ToggleExpanded() bool {
	return s.ctrl.ToggleExpanded()
}

// Show reveals the island.
func (s *IslandService) Show() { s.ctrl.Show() }

// Hide hides the island until the user brings it back from the tray.
func (s *IslandService) Hide() { s.ctrl.Hide() }

// ToggleVisible shows or hides the island.
func (s *IslandService) ToggleVisible() bool {
	s.ctrl.ToggleVisible()
	return s.ctrl.Visible()
}

// IsVisible reports whether the island window is shown.
func (s *IslandService) IsVisible() bool { return s.ctrl.Visible() }

// Focus activates the window so the webview receives keyboard events.
func (s *IslandService) Focus() { s.ctrl.Focus() }

// Quit exits the application.
func (s *IslandService) Quit() { s.app.Quit() }
