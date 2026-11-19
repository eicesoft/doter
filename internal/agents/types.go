// Package agents discovers Claude Code and Codex CLI sessions from their
// on-disk logs and reports which ones are running.
package agents

// Status describes whether an agent session is still working.
const (
	// StatusRunning means the agent is mid-turn: streaming, calling tools or
	// waiting on a model response.
	StatusRunning = "running"
	// StatusIdle means the process is alive but waiting for the next prompt.
	StatusIdle = "idle"
	// StatusDone means no live process could be matched to the session.
	StatusDone = "done"
)

// Session is one Claude Code or Codex conversation.
type Session struct {
	ID string `json:"id"`
	// Agent is "claude" or "codex".
	Agent       string `json:"agent"`
	Project     string `json:"project"`
	ProjectName string `json:"projectName"`
	// Title is the session's custom title, fallback to the first user prompt.
	Title   string `json:"title"`
	Model   string `json:"model"`
	Branch  string `json:"branch"`
	Version string `json:"version"`
	Status  string `json:"status"`

	// StartedAt and LastActivity are Unix milliseconds.
	StartedAt    int64 `json:"startedAt"`
	LastActivity int64 `json:"lastActivity"`

	// InputTokens and OutputTokens are cumulative for the session.
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
	CachedTokens int64 `json:"cachedTokens"`
	// ContextTokens is the prompt size of the most recent request.
	ContextTokens int64 `json:"contextTokens"`
	// Turns counts assistant responses.
	Turns int `json:"turns"`
	// PID is the live process serving the session, zero when none matched.
	PID int32 `json:"pid,omitempty"`
}

// Snapshot is the agent panel's view of every recently active session.
type Snapshot struct {
	Sessions []Session `json:"sessions"`
	Running  int       `json:"running"`
	Idle     int       `json:"idle"`
	Done     int       `json:"done"`
	// InputTokens and OutputTokens sum every session in the window.
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
	UpdatedAt    int64 `json:"updatedAt"`
}
