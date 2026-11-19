package services

import "doter/internal/agents"

// EventAgents is emitted after every monitor sample with an agents.Snapshot
// payload.
const EventAgents = "agents:update"

// AgentsService exposes the Claude/Codex session monitor to the frontend.
type AgentsService struct {
	monitor *agents.Monitor
}

// NewAgentsService creates the service.
func NewAgentsService(monitor *agents.Monitor) *AgentsService {
	return &AgentsService{monitor: monitor}
}

// GetSnapshot returns the latest cached session list. The UI calls this once
// on mount and then follows EventAgents for live updates.
func (s *AgentsService) GetSnapshot() agents.Snapshot {
	return s.monitor.Current()
}
