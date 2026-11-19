package services

import (
	"context"
	"log"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// UpdateService exposes the app version and starts the built-in update flow.
type UpdateService struct {
	app     *application.App
	running atomic.Bool
}

// NewUpdateService creates the service.
func NewUpdateService(app *application.App) *UpdateService {
	return &UpdateService{app: app}
}

// Version returns the version the running binary was built with.
func (s *UpdateService) Version() string {
	return s.app.Updater.CurrentVersion()
}

// Check opens the update window and starts check + download. The window owns
// the user-facing states (checking, up to date, available, error), so the
// binding returns immediately.
func (s *UpdateService) Check() {
	if !s.running.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer s.running.Store(false)
		if err := s.app.Updater.CheckAndInstall(context.Background()); err != nil {
			log.Printf("update check: %v", err)
		}
	}()
}
