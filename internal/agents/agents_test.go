package agents

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeLog(t *testing.T, path, content string) os.FileInfo {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat log: %v", err)
	}
	return info
}

func TestClaudeIncrementalParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	content := `{"type":"user","timestamp":"2026-11-19T08:00:00Z","cwd":"/work/demo","sessionId":"abc12345","gitBranch":"main","version":"2.0.0","message":{"role":"user","content":"Fix the login bug"}}
{"type":"custom-title","customTitle":"Login fix","sessionId":"abc12345"}
{"type":"assistant","timestamp":"2026-11-19T08:00:01Z","message":{"model":"claude-sonnet","stop_reason":"tool_use","usage":{"input_tokens":100,"output_tokens":20,"cache_read_input_tokens":50,"cache_creation_input_tokens":5}}}
`
	state := &claudeState{}
	state.reset()
	parseClaudeFile(path, writeLog(t, path, content), state)

	if title := claudeTitle(state); title != "Login fix" {
		t.Errorf("title = %q", title)
	}
	if state.session.ProjectName != "demo" || state.session.Project != "/work/demo" {
		t.Errorf("project = %q (%q)", state.session.Project, state.session.ProjectName)
	}
	if state.session.InputTokens != 100 || state.session.OutputTokens != 20 || state.session.CachedTokens != 50 {
		t.Errorf("tokens = %+v", state.session)
	}
	if state.session.ContextTokens != 155 || state.session.Turns != 1 {
		t.Errorf("context/turns = %d/%d", state.session.ContextTokens, state.session.Turns)
	}
	if state.waiting {
		t.Error("tool_use must leave the session running")
	}
	if !state.awaiting {
		t.Error("a tool call without its result must mark the session blocked")
	}

	appendFile(t, path, `{"type":"assistant","timestamp":"2026-11-19T08:00:02Z","message":{"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":200}}}
{"type":"assistant","timestamp":"2026-11-19T08:00:03Z","message":{"stop_reason":"end_turn"}}
`)
	parseClaudeFile(path, mustStat(t, path), state)

	if !state.waiting {
		t.Error("end_turn must mark the session idle")
	}
	if state.awaiting {
		t.Error("end_turn must clear the blocked flag")
	}
	if state.session.InputTokens != 110 || state.session.OutputTokens != 25 || state.session.CachedTokens != 250 {
		t.Errorf("tokens after append = %+v", state.session)
	}
	if state.session.ContextTokens != 210 {
		t.Errorf("context = %d", state.session.ContextTokens)
	}
	if state.session.LastActivity != parseTimestamp("2026-11-19T08:00:03Z") {
		t.Errorf("last activity = %d", state.session.LastActivity)
	}

	// Claude Code writes last-prompt after the turn's end_turn; it must not
	// flip the finished session back to running.
	appendFile(t, path, `{"type":"last-prompt","lastPrompt":"Fix the login bug"}
`)
	parseClaudeFile(path, mustStat(t, path), state)
	if !state.waiting {
		t.Error("last-prompt after end_turn must keep the session idle")
	}
}

func TestCodexParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	content := `{"timestamp":"2026-11-19T09:00:00Z","type":"session_meta","payload":{"id":"thread-1","cwd":"/work/demo","cli_version":"0.170.0"}}
{"timestamp":"2026-11-19T09:00:01Z","type":"turn_context","payload":{"model":"gpt-5-codex","cwd":"/work/demo"}}
{"timestamp":"2026-11-19T09:00:02Z","type":"event_msg","payload":{"type":"task_started"}}
{"timestamp":"2026-11-19T09:00:03Z","type":"event_msg","payload":{"type":"item_completed","item":{"type":"UserMessage","content":[{"type":"text","text":"Ship the release"}]}}}
{"timestamp":"2026-11-19T09:00:04Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":400,"output_tokens":50,"total_tokens":1050},"last_token_usage":{"input_tokens":700,"cached_input_tokens":300,"output_tokens":10,"total_tokens":710}}}}
`
	state := &codexState{}
	state.reset()
	parseCodexFile(path, writeLog(t, path, content), state)

	if state.session.ID != "thread-1" || state.session.ProjectName != "demo" {
		t.Errorf("identity = %+v", state.session)
	}
	if state.session.Model != "gpt-5-codex" {
		t.Errorf("model = %q", state.session.Model)
	}
	if state.prompt != "Ship the release" {
		t.Errorf("title = %q", state.prompt)
	}
	if state.session.InputTokens != 1000 || state.session.CachedTokens != 400 || state.session.OutputTokens != 50 {
		t.Errorf("tokens = %+v", state.session)
	}
	if state.session.ContextTokens != 710 {
		t.Errorf("context = %d", state.session.ContextTokens)
	}
	if state.waiting {
		t.Error("a started task must count as running")
	}

	appendFile(t, path, `{"timestamp":"2026-11-19T09:00:05Z","type":"event_msg","payload":{"type":"task_complete"}}
`)
	parseCodexFile(path, mustStat(t, path), state)
	if !state.waiting {
		t.Error("task_complete must mark the session idle")
	}
	if state.session.ProjectName != "demo" {
		t.Error("a later cwd must not replace the project root")
	}

	appendFile(t, path, `{"timestamp":"2026-11-19T09:00:06Z","type":"event_msg","payload":{"type":"task_started"}}
{"timestamp":"2026-11-19T09:00:07Z","type":"response_item","payload":{"type":"function_call","name":"shell"}}
`)
	parseCodexFile(path, mustStat(t, path), state)
	if !state.awaiting {
		t.Error("a function call without its output must mark the session blocked")
	}

	appendFile(t, path, `{"timestamp":"2026-11-19T09:00:08Z","type":"response_item","payload":{"type":"function_call_output","output":"ok"}}
`)
	parseCodexFile(path, mustStat(t, path), state)
	if state.awaiting {
		t.Error("a function call output must clear the blocked flag")
	}
}

func TestSessionStatus(t *testing.T) {
	now := time.Now()
	running := &sessionEntry{session: Session{PID: 42, LastActivity: now.UnixMilli()}}
	if got := sessionStatus(running, now); got != StatusRunning {
		t.Errorf("live working session = %q", got)
	}
	awaiting := &sessionEntry{session: Session{PID: 42, LastActivity: now.UnixMilli()}, awaiting: true}
	if got := sessionStatus(awaiting, now); got != StatusWaiting {
		t.Errorf("live blocked session = %q", got)
	}
	// A finished turn is done even while its CLI stays open.
	finished := &sessionEntry{session: Session{PID: 42, LastActivity: now.UnixMilli()}, waiting: true}
	if got := sessionStatus(finished, now); got != StatusDone {
		t.Errorf("finished turn = %q", got)
	}
	stale := &sessionEntry{session: Session{LastActivity: now.Add(-time.Hour).UnixMilli()}, awaiting: true}
	if got := sessionStatus(stale, now); got != StatusDone {
		t.Errorf("dead session = %q", got)
	}
}

func appendFile(t *testing.T, path, content string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("append log: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close log: %v", err)
	}
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat log: %v", err)
	}
	return info
}
