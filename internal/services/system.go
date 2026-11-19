// Package services exposes backend capabilities to the React frontend.
package services

import (
	"doter/internal/sysinfo"
)

// EventSnapshot is emitted once per sample with a sysinfo.Snapshot payload.
const EventSnapshot = "sysinfo:update"

// SystemService is bound into the frontend by Wails.
type SystemService struct {
	sampler *sysinfo.Sampler
}

func NewSystemService(sampler *sysinfo.Sampler) *SystemService {
	return &SystemService{sampler: sampler}
}

// GetSnapshot returns the latest cached reading. The UI calls this once on
// mount and then follows EventSnapshot for live updates.
func (s *SystemService) GetSnapshot() sysinfo.Snapshot {
	return s.sampler.Current()
}

// GetStaticInfo returns hardware and OS details captured at startup.
func (s *SystemService) GetStaticInfo() sysinfo.StaticInfo {
	return s.sampler.Static()
}

// SearchProcesses matches every running process by name or executable path.
func (s *SystemService) SearchProcesses(query string) []sysinfo.ProcessStats {
	return s.sampler.SearchProcesses(query, 30)
}

// KillProcess terminates a process after verifying its start time.
func (s *SystemService) KillProcess(pid int32, createTime int64) error {
	return sysinfo.KillProcess(pid, createTime)
}

// RevealProcess opens the file manager at the process executable.
func (s *SystemService) RevealProcess(pid int32, createTime int64) bool {
	return sysinfo.RevealProcess(pid, createTime)
}
