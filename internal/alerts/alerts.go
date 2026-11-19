// Package alerts turns the live system and agent streams into a small number
// of high-signal notifications. Rules keep their own hysteresis and cooldown
// state so a sustained condition does not produce a stream of notifications.
package alerts

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"doter/internal/agents"
	"doter/internal/sysinfo"
)

// Alert is one notification-worthy event. Tab is the dashboard tab the
// notification should open when the user clicks it.
type Alert struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Tab       string `json:"tab"`
	CreatedAt int64  `json:"createdAt"`
}

const (
	KindCPU         = "cpu"
	KindMemory      = "memory"
	KindSwap        = "swap"
	KindDisk        = "disk"
	KindBattery     = "battery"
	KindAgentIdle   = "agent-idle"
	KindAgentDone   = "agent-done"
	tabOverview     = "overview"
	tabAgents       = "agents"
	cpuThreshold    = 90.0
	cpuClear        = 80.0
	memoryLimit     = 85.0
	memoryClear     = 80.0
	swapLimit       = 50.0
	swapClear       = 40.0
	batteryLimit    = 20.0
	diskFreeFloor   = 10 << 30 // 10 GiB
	diskUsedFloor   = 90.0
	cpuHold         = 30 * time.Second
	memoryHold      = 30 * time.Second
	swapHold        = 60 * time.Second
	cpuCooldown     = 15 * time.Minute
	memoryCooldown  = 30 * time.Minute
	swapCooldown    = 30 * time.Minute
	diskCooldown    = 6 * time.Hour
	batteryCooldown = 30 * time.Minute
	// agentMinRun filters out quick interactive turns: an agent that finishes
	// a short answer while the user is already watching the terminal should
	// not produce a notification on every turn.
	agentMinRun = 2 * time.Minute
)

type holdState struct {
	since     int64
	lastFired int64
}

type agentState struct {
	status       string
	runningSince int64
	notified     bool
}

// Engine evaluates snapshots. The enabled callback is read on every check so
// toggling alerts in the UI takes effect immediately without restarting.
type Engine struct {
	mu      sync.Mutex
	enabled func() bool
	notify  func(Alert)

	cpu      holdState
	memory   holdState
	swap     holdState
	lastDisk int64
	lastBatt int64

	agentStates map[string]*agentState
	seq         int
}

// New creates an engine. notify may be nil when alerts are only observed by
// tests.
func New(enabled func() bool, notify func(Alert)) *Engine {
	return &Engine{
		enabled:     enabled,
		notify:      notify,
		agentStates: make(map[string]*agentState),
	}
}

// ObserveSnapshot evaluates the system rules for one sample.
func (e *Engine) ObserveSnapshot(snapshot sysinfo.Snapshot) {
	now := snapshot.Timestamp
	if now <= 0 {
		now = time.Now().UnixMilli()
	}

	e.mu.Lock()
	var out []Alert
	if !e.on() {
		// Do not carry a half-elapsed hold across a disabled period: turning
		// alerts back on should give the user the full 30-second window.
		e.cpu.since = 0
		e.memory.since = 0
		e.swap.since = 0
	} else {
		if e.held(&e.cpu, snapshot.CPU.Usage, cpuThreshold, cpuClear, now, cpuHold, cpuCooldown) {
			out = append(out, e.alert(KindCPU, "CPU 占用过高",
				fmt.Sprintf("CPU 持续 30 秒高于 90%%，当前 %.0f%%。", snapshot.CPU.Usage), tabOverview, now))
		}
		if e.held(&e.memory, snapshot.Memory.UsedPercent, memoryLimit, memoryClear, now, memoryHold, memoryCooldown) {
			out = append(out, e.alert(KindMemory, "内存占用过高",
				fmt.Sprintf("内存占用 %.0f%%（%s / %s）。", snapshot.Memory.UsedPercent,
					formatBytes(snapshot.Memory.Used), formatBytes(snapshot.Memory.Total)), tabOverview, now))
		}
		if snapshot.Swap.Total > 0 && e.held(&e.swap, snapshot.Swap.UsedPercent, swapLimit, swapClear, now, swapHold, swapCooldown) {
			out = append(out, e.alert(KindSwap, "Swap 占用过高",
				fmt.Sprintf("Swap 已用 %.0f%%（%s / %s）。", snapshot.Swap.UsedPercent,
					formatBytes(snapshot.Swap.Used), formatBytes(snapshot.Swap.Total)), tabOverview, now))
		}
		if disk, ok := lowDisk(snapshot.Disks); ok && e.cooldown(&e.lastDisk, now, diskCooldown) {
			out = append(out, e.alert(KindDisk, "磁盘空间不足",
				fmt.Sprintf("%s 剩余 %s（已用 %.0f%%）。", disk.Mountpoint,
					formatBytes(disk.Free), disk.UsedPercent), tabOverview, now))
		}
		if b := snapshot.Battery; b != nil && !b.PluggedIn && b.Percent <= batteryLimit &&
			e.cooldown(&e.lastBatt, now, batteryCooldown) {
			out = append(out, e.alert(KindBattery, "电量过低",
				fmt.Sprintf("电量 %.0f%%，请连接电源。", b.Percent), tabOverview, now))
		}
	}
	e.mu.Unlock()

	e.deliver(out)
}

// ObserveAgents compares the current session list with the previous sample.
// The first snapshot only establishes a baseline, so sessions that were
// already running when the app started do not notify immediately.
func (e *Engine) ObserveAgents(snapshot agents.Snapshot) {
	now := snapshot.UpdatedAt
	if now <= 0 {
		now = time.Now().UnixMilli()
	}

	e.mu.Lock()
	var out []Alert
	seen := make(map[string]bool, len(snapshot.Sessions))
	for _, session := range snapshot.Sessions {
		key := session.Agent + ":" + session.ID
		seen[key] = true
		state, ok := e.agentStates[key]
		if !ok {
			state = &agentState{status: session.Status, runningSince: now}
			if session.Status == agents.StatusRunning {
				state.runningSince = now
			}
			e.agentStates[key] = state
			continue
		}

		if session.Status == agents.StatusRunning {
			if state.status != agents.StatusRunning {
				state.runningSince = now
				state.notified = false
			}
			state.status = session.Status
			continue
		}

		if state.status == agents.StatusRunning && !state.notified {
			state.notified = true
			if e.on() {
				switch session.Status {
				case agents.StatusIdle:
					if now-state.runningSince >= agentMinRun.Milliseconds() {
						out = append(out, e.agentAlert(session, KindAgentIdle, "等待输入",
							fmt.Sprintf("%s · %s", sessionProject(session), sessionLabel(session)), now))
					}
				case agents.StatusDone:
					out = append(out, e.agentAlert(session, KindAgentDone, "已完成",
						fmt.Sprintf("%s · %s\n本次 ↑%s ↓%s tokens", sessionProject(session),
							sessionLabel(session), formatTokens(session.InputTokens),
							formatTokens(session.OutputTokens)), now))
				}
			}
		}
		state.status = session.Status
	}

	for key := range e.agentStates {
		if !seen[key] {
			delete(e.agentStates, key)
		}
	}
	e.mu.Unlock()

	e.deliver(out)
}

func (e *Engine) on() bool {
	return e.enabled == nil || e.enabled()
}

// held reports whether a threshold has been true long enough and its cooldown
// has elapsed. clear is lower than limit so a value hovering around the limit
// does not reset the timer on every sample.
func (e *Engine) held(state *holdState, value, limit, clear float64, now int64, hold, cooldown time.Duration) bool {
	if value < clear {
		state.since = 0
		return false
	}
	if value < limit {
		return false
	}
	if state.since == 0 {
		state.since = now
	}
	if now-state.since < hold.Milliseconds() {
		return false
	}
	if state.lastFired != 0 && now-state.lastFired < cooldown.Milliseconds() {
		return false
	}
	state.lastFired = now
	return true
}

func (e *Engine) cooldown(last *int64, now int64, window time.Duration) bool {
	if *last != 0 && now-*last < window.Milliseconds() {
		return false
	}
	*last = now
	return true
}

func (e *Engine) alert(kind, title, body, tab string, now int64) Alert {
	e.seq++
	return Alert{
		ID:        fmt.Sprintf("doter-%s-%d-%d", kind, now, e.seq),
		Kind:      kind,
		Title:     title,
		Body:      body,
		Tab:       tab,
		CreatedAt: now,
	}
}

func (e *Engine) agentAlert(session agents.Session, kind, suffix, body string, now int64) Alert {
	label := "Claude Code"
	if session.Agent == "codex" {
		label = "Codex"
	}
	return e.alert(kind, label+" "+suffix, body, tabAgents, now)
}

func (e *Engine) deliver(alerts []Alert) {
	if e.notify == nil {
		return
	}
	for _, alert := range alerts {
		e.notify(alert)
	}
}

func lowDisk(disks []sysinfo.DiskStats) (sysinfo.DiskStats, bool) {
	var worst sysinfo.DiskStats
	found := false
	for _, disk := range disks {
		if disk.Total == 0 || disk.UsedPercent < diskUsedFloor || disk.Free >= diskFreeFloor {
			continue
		}
		if !found || disk.Free < worst.Free {
			worst = disk
			found = true
		}
	}
	return worst, found
}

func sessionProject(session agents.Session) string {
	if session.ProjectName != "" {
		return session.ProjectName
	}
	if session.Project != "" {
		return session.Project
	}
	return "Agent"
}

func sessionLabel(session agents.Session) string {
	if session.Title != "" {
		return session.Title
	}
	if session.ProjectName != "" {
		return session.ProjectName
	}
	if len(session.ID) > 8 {
		return session.ID[:8]
	}
	return session.ID
}

func formatTokens(value int64) string {
	switch {
	case value >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	case value >= 1_000:
		return fmt.Sprintf("%.1fk", float64(value)/1_000)
	default:
		return strconv.FormatInt(value, 10)
	}
}

func formatBytes(value uint64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	div, exp := uint64(unit), 0
	for n := value / unit; n >= unit && exp < 4; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(value)/float64(div), "KMGTP"[exp])
}
