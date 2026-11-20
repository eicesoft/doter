package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"doter/internal/clipboard"
)

// EventClipboard is emitted with a clipboard entry whenever the pasteboard
// changes.
const EventClipboard = "clipboard:update"

// EventClipboardConfig is emitted when limits or exclusions change, so an
// open clipboard panel reloads its list.
const EventClipboardConfig = "clipboard:config"

// ClipboardService exposes the clipboard history to the frontend.
//
// The watcher itself is not part of this service: binding a method that
// returns a channel or takes a context is not supported, and the frontend
// only ever needs the history it already owns.
type ClipboardService struct {
	history *clipboard.History
	app     *application.App
}

// NewClipboardService creates the service.
func NewClipboardService(history *clipboard.History, app *application.App) *ClipboardService {
	return &ClipboardService{history: history, app: app}
}

// GetHistory returns the clipboard entries, newest first.
func (s *ClipboardService) GetHistory() []clipboard.Entry {
	return s.history.List()
}

// Pin toggles the pinned flag of an entry, keeping it at the top of the list.
func (s *ClipboardService) Pin(id string) bool {
	return s.history.Pin(id)
}

// Delete removes an entry from the history.
func (s *ClipboardService) Delete(id string) bool {
	return s.history.Delete(id)
}

// Clear empties the history.
func (s *ClipboardService) Clear() {
	s.history.Clear()
}

// WriteBack puts an entry's content back on the clipboard, so it can be
// pasted straight away.
func (s *ClipboardService) WriteBack(id string) bool {
	if path, ok := s.history.ImagePath(id); ok {
		return clipboard.WriteImageFile(path)
	}
	// Restoring the rich representations keeps the source app's data and
	// styling; fall back to plain text when there are none.
	if formats, ok := s.history.Formats(id); ok && clipboard.WriteFormats(formats) {
		return true
	}
	content, ok := s.history.Full(id)
	if !ok {
		return false
	}
	return s.app.Clipboard.SetText(content)
}

// Paste sends the platform's paste shortcut to the application that was
// frontmost when the clipboard was opened. The caller writes the entry back
// first, then collapses the island, then calls Paste.
func (s *ClipboardService) Paste() error {
	return clipboard.Paste()
}
