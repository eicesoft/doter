package agents

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

const (
	// sampleInterval is how often the session logs and processes are rescanned.
	// Logs are read incrementally, so only newly appended bytes cost anything.
	sampleInterval = 3 * time.Second
	// sessionWindow bounds the sessions shown: logs older than this are ignored.
	sessionWindow = 24 * time.Hour
	// maxSessions caps the panel payload; the UI scrolls the most recent ones.
	maxSessions = 30
	// activeGrace covers the gap between an agent writing to its log and the
	// process scan seeing it.
	activeGrace = 90 * time.Second
)

// Monitor tails the agent session logs and matches them against live agent
// processes. It owns the only goroutine that touches the parser state.
type Monitor struct {
	mu       sync.RWMutex
	snapshot Snapshot
	updates  chan Snapshot
	started  bool

	home string
	// claude, codex and pi keep the per-file read offset and running
	// aggregates, so a sample only decodes freshly appended lines.
	claude map[string]*claudeState
	codex  map[string]*codexState
	pi     map[string]*piState
}

// NewMonitor builds a monitor for the current user.
func NewMonitor() *Monitor {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return &Monitor{
		updates: make(chan Snapshot, 1),
		home:    home,
		claude:  make(map[string]*claudeState),
		codex:   make(map[string]*codexState),
		pi:      make(map[string]*piState),
		snapshot: Snapshot{
			Sessions: []Session{},
		},
	}
}

// Start begins sampling until ctx is cancelled.
func (m *Monitor) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	go func() {
		m.sampleOnce()
		m.publish()

		ticker := time.NewTicker(sampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.sampleOnce()
				m.publish()
			}
		}
	}()
}

// Updates yields a snapshot after every sample. The channel is never closed.
func (m *Monitor) Updates() <-chan Snapshot { return m.updates }

// Current returns the most recent snapshot.
func (m *Monitor) Current() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap := m.snapshot
	snap.Sessions = append([]Session(nil), m.snapshot.Sessions...)
	return snap
}

func (m *Monitor) publish() {
	snap := m.Current()
	select {
	case m.updates <- snap:
	default:
		// The UI is behind; drop this frame rather than queueing.
	}
}

// sessionEntry pairs a parsed session with the parser-only lifecycle flags.
type sessionEntry struct {
	session Session
	// waiting is true when the agent finished its turn and is waiting for the
	// user. It is intentionally not part of the JSON payload.
	waiting bool
	// awaiting is true when a live turn is blocked on a tool call the user has
	// to approve or answer. Parse-only, like waiting.
	awaiting bool
}

func (m *Monitor) sampleOnce() {
	now := time.Now()
	entries := make([]*sessionEntry, 0, 16)

	if m.home != "" {
		entries = append(entries, m.scanClaude(now)...)
		entries = append(entries, m.scanCodex(now)...)
		entries = append(entries, m.scanPi(now)...)
		entries = append(entries, m.scanOpencode(now)...)
	}

	assignProcesses(entries)

	for _, entry := range entries {
		entry.session.Status = sessionStatus(entry, now)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].session.LastActivity != entries[j].session.LastActivity {
			return entries[i].session.LastActivity > entries[j].session.LastActivity
		}
		return entries[i].session.ID > entries[j].session.ID
	})

	snapshot := Snapshot{
		Sessions:  make([]Session, 0, len(entries)),
		UpdatedAt: now.UnixMilli(),
	}
	for _, entry := range entries {
		switch entry.session.Status {
		case StatusRunning:
			snapshot.Running++
		case StatusWaiting:
			snapshot.Waiting++
		case StatusEnded:
			snapshot.Ended++
		default:
			snapshot.Done++
		}
		snapshot.InputTokens += entry.session.InputTokens
		snapshot.OutputTokens += entry.session.OutputTokens
		if len(snapshot.Sessions) < maxSessions {
			snapshot.Sessions = append(snapshot.Sessions, entry.session)
		}
	}

	m.mu.Lock()
	m.snapshot = snapshot
	m.mu.Unlock()

	m.prune(now)
}

// sessionStatus maps the parser flags onto the panel's status. A finished turn
// is done while its CLI stays open; once no process serves the session it is
// ended. A live turn is running, or waiting when blocked on a user answer.
func sessionStatus(entry *sessionEntry, now time.Time) string {
	// A log that is still being appended to is live even when the process scan
	// missed it (agents may run behind a wrapper binary).
	live := entry.session.PID != 0 ||
		now.Sub(time.UnixMilli(entry.session.LastActivity)) < activeGrace
	if !live {
		return StatusEnded
	}
	if entry.awaiting {
		return StatusWaiting
	}
	if entry.waiting {
		return StatusDone
	}
	return StatusRunning
}

// agentProc is a live agent process.
type agentProc struct {
	pid   int32
	agent string
	cwd   string
}

// scanAgentProcesses finds running agent CLIs. The desktop app's main process
// shares the "claude" name, so its /Applications path is filtered out.
func scanAgentProcesses() []agentProc {
	processes, err := process.Processes()
	if err != nil {
		return nil
	}
	procs := make([]agentProc, 0, 8)
	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		agent := agentForName(name)
		if agent == "" && strings.EqualFold(filepath.Base(strings.TrimSuffix(name, ".exe")), "node") {
			// pi ships as a node script, so it only shows up on the command line.
			cmdline, _ := p.Cmdline()
			if isPiCommand(cmdline) {
				agent = "pi"
			}
		}
		if agent == "" {
			continue
		}
		exe, _ := p.Exe()
		if isDesktopApp(exe) {
			continue
		}
		cwd, _ := p.Cwd()
		procs = append(procs, agentProc{pid: p.Pid, agent: agent, cwd: cwd})
	}
	return procs
}

func agentForName(name string) string {
	base := strings.ToLower(filepath.Base(strings.TrimSuffix(name, ".exe")))
	switch base {
	case "claude":
		return "claude"
	case "codex":
		return "codex"
	case "opencode":
		return "opencode"
	case "omp":
		return "omp"
	default:
		return ""
	}
}

// isPiCommand looks for the pi entry script among the process arguments.
func isPiCommand(cmdline string) bool {
	for _, field := range strings.Fields(cmdline) {
		if strings.Contains(field, "/") && filepath.Base(field) == "pi" {
			return true
		}
	}
	return false
}

func isDesktopApp(exe string) bool {
	low := strings.ToLower(filepath.ToSlash(exe))
	return strings.HasPrefix(low, "/applications/claude.app/")
}

// assignProcesses matches processes to sessions by working directory. When
// several sessions share a project, the most recently active one gets the
// live process.
func assignProcesses(entries []*sessionEntry) {
	buckets := make(map[string][]int32)
	for _, proc := range scanAgentProcesses() {
		if proc.cwd == "" {
			continue
		}
		key := proc.agent + "\x00" + filepath.Clean(proc.cwd)
		buckets[key] = append(buckets[key], proc.pid)
	}
	if len(buckets) == 0 {
		return
	}

	sorted := append([]*sessionEntry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].session.LastActivity > sorted[j].session.LastActivity
	})
	for _, entry := range sorted {
		if entry.session.Project == "" {
			continue
		}
		key := entry.session.Agent + "\x00" + filepath.Clean(entry.session.Project)
		pids := buckets[key]
		if len(pids) == 0 {
			continue
		}
		entry.session.PID = pids[0]
		buckets[key] = pids[1:]
	}
}

func (m *Monitor) prune(now time.Time) {
	cutoff := now.Add(-sessionWindow).UnixMilli()
	for path, state := range m.claude {
		if state.session.LastActivity < cutoff {
			delete(m.claude, path)
		}
	}
	for path, state := range m.codex {
		if state.session.LastActivity < cutoff {
			delete(m.codex, path)
		}
	}
	for path, state := range m.pi {
		if state.session.LastActivity < cutoff {
			delete(m.pi, path)
		}
	}
}

// cleanTitle flattens a prompt into a single display line.
func cleanTitle(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) > 72 {
		text = string(runes[:71]) + "…"
	}
	return text
}

// looksLikePrompt filters out injected context before it becomes a title.
func looksLikePrompt(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(text, "<") {
		return false
	}
	return !strings.Contains(text, "<INSTRUCTIONS>")
}
