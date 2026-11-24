package services

import "doter/internal/notice"

// EventNotices is emitted with a notice.Snapshot when the platform
// notification store changes. Snapshot.New carries entries that just arrived,
// which the island shows as a bubble.
const EventNotices = "notices:update"

// NoticeService exposes the system notification monitor to the frontend.
type NoticeService struct {
	monitor *notice.Monitor
}

// NewNoticeService creates the service.
func NewNoticeService(monitor *notice.Monitor) *NoticeService {
	return &NoticeService{monitor: monitor}
}

// GetSnapshot returns the cached notification history. The UI calls this once
// on mount and then follows EventNotices for live updates.
func (s *NoticeService) GetSnapshot() notice.Snapshot {
	return s.monitor.Current()
}

// OpenApp activates the app that posted a notification.
func (s *NoticeService) OpenApp(app string) error {
	return notice.OpenApp(app)
}

// GetAppIcon returns a data URL for the app's icon, or an empty string when
// the platform cannot resolve one. The UI falls back to the app's initial.
func (s *NoticeService) GetAppIcon(app string) string {
	return notice.AppIcon(app)
}

// Clear empties the panel history.
func (s *NoticeService) Clear() {
	s.monitor.Clear()
}

// Remove drops the given notifications from the panel history. Removing a
// stack passes every id in that app's group.
func (s *NoticeService) Remove(ids []string) {
	s.monitor.Remove(ids)
}
