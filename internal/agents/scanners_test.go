package agents

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestAgentProcessMatching(t *testing.T) {
	for name, want := range map[string]string{
		"claude": "claude", "codex": "codex", "opencode": "opencode", "omp": "omp", "node": "",
	} {
		if got := agentForName(name); got != want {
			t.Errorf("agentForName(%q) = %q, want %q", name, got, want)
		}
	}
	if !isPiCommand("/Users/x/.nvm/versions/node/v24/bin/node /Users/x/.npm-global/bin/pi") {
		t.Error("a node process running bin/pi must be recognised")
	}
	if isPiCommand("/usr/bin/python3 /work/pipeline.py") {
		t.Error("an unrelated python process must not be recognised")
	}
}

func TestPiParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	content := `{"type":"session","version":3,"id":"pi-1","timestamp":"2026-11-19T10:00:00Z","cwd":"/work/demo"}
{"type":"model_change","timestamp":"2026-11-19T10:00:01Z","provider":"magpie","modelId":"deepseek-v4"}
{"type":"session_info","timestamp":"2026-11-19T10:00:02Z","name":"Fix the tests"}
{"type":"message","timestamp":"2026-11-19T10:00:03Z","message":{"role":"user","content":[{"type":"text","text":"Fix the failing tests"}]}}
{"type":"message","timestamp":"2026-11-19T10:00:04Z","message":{"role":"assistant","model":"deepseek-v4","stopReason":"toolUse","usage":{"input":100,"output":20,"cacheRead":50,"cacheWrite":5}}}
`
	state := &piState{}
	state.reset("pi")
	parsePiFile(path, writeLog(t, path, content), state)

	if state.session.Agent != "pi" || state.session.ID != "pi-1" || state.session.ProjectName != "demo" {
		t.Errorf("identity = %+v", state.session)
	}
	if title := piTitle(state); title != "Fix the tests" {
		t.Errorf("title = %q", title)
	}
	if state.session.Model != "deepseek-v4" {
		t.Errorf("model = %q", state.session.Model)
	}
	if state.session.InputTokens != 100 || state.session.OutputTokens != 20 || state.session.CachedTokens != 50 {
		t.Errorf("tokens = %+v", state.session)
	}
	if state.session.ContextTokens != 155 || state.session.Turns != 1 {
		t.Errorf("context/turns = %d/%d", state.session.ContextTokens, state.session.Turns)
	}
	if state.waiting {
		t.Error("toolUse must leave the session running")
	}
	if !state.awaiting {
		t.Error("toolUse must mark the session blocked")
	}

	appendFile(t, path, `{"type":"message","timestamp":"2026-11-19T10:00:05Z","message":{"role":"assistant","model":"deepseek-v4","stopReason":"stop","usage":{"input":10,"output":5,"cacheRead":200}}}
`)
	parsePiFile(path, mustStat(t, path), state)

	if !state.waiting {
		t.Error("stop must mark the session idle")
	}
	if state.session.InputTokens != 110 || state.session.OutputTokens != 25 || state.session.CachedTokens != 250 {
		t.Errorf("tokens after append = %+v", state.session)
	}
	if state.session.ContextTokens != 210 {
		t.Errorf("context = %d", state.session.ContextTokens)
	}
}

func TestOmpParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	content := `{"type":"session","version":3,"id":"omp-1","timestamp":"2026-11-19T11:00:00Z","cwd":"/work/demo"}
{"type":"title","title":"","updatedAt":"2026-11-19T11:00:00Z"}
{"type":"title_change","title":"Ship the release","timestamp":"2026-11-19T11:00:01Z"}
{"type":"model_change","timestamp":"2026-11-19T11:00:02Z","model":"magpie/glm-5"}
{"type":"message","timestamp":"2026-11-19T11:00:03Z","message":{"role":"user","content":[{"type":"text","text":"ship it"}]}}
{"type":"message","timestamp":"2026-11-19T11:00:04Z","message":{"role":"assistant","model":"glm-5","stopReason":"stop","usage":{"input":30,"output":10}}}
`
	state := &piState{}
	state.reset("omp")
	parsePiFile(path, writeLog(t, path, content), state)

	if state.session.Agent != "omp" || state.session.ProjectName != "demo" {
		t.Errorf("identity = %+v", state.session)
	}
	if title := piTitle(state); title != "Ship the release" {
		t.Errorf("title = %q", title)
	}
	if !state.waiting {
		t.Error("a stopped turn must be idle")
	}
}

func TestOpencodeAwaiting(t *testing.T) {
	pending := `{"content":[{"type":"text"},{"type":"tool","state":{"status":"pending"}}]}`
	if !opencodeAwaiting(pending) {
		t.Error("a pending tool part must mark the session blocked")
	}
	if opencodeAwaiting(`{"content":[{"type":"tool","state":{"status":"running"}}]}`) {
		t.Error("a running tool part must not mark the session blocked")
	}
	if opencodeAwaiting("") {
		t.Error("an empty reply must not mark the session blocked")
	}
}

func TestOpencodeParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`create table session_v2 (id text primary key, parent_id text, title text, slug text, directory text, model text, version text, tokens_input integer, tokens_output integer, tokens_cache_read integer, time_created integer, time_updated integer)`); err != nil {
		t.Fatalf("schema session: %v", err)
	}
	if _, err := db.Exec(`create table session_message (id text primary key, session_id text, type text, seq integer, time_created integer, data text)`); err != nil {
		t.Fatalf("schema message: %v", err)
	}

	if _, err := db.Exec(`insert into session_v2 values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, "ses_1", nil, "Fix login", "calm-fox", "/work/demo", `{"id":"glm-5"}`, "2.4.0", 1000, 50, 400, 100, 200); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := db.Exec(`insert into session_v2 values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, "ses_child", "ses_1", "Sub", "x", "/work/demo", "", "2.4.0", 0, 0, 0, 100, 200); err != nil {
		t.Fatalf("insert child: %v", err)
	}
	if _, err := db.Exec(`insert into session_message values (?, ?, ?, ?, ?, ?)`, "m1", "ses_1", "assistant", 1, 150, `{"tokens":{"input":700,"cache":{"read":300,"write":10}}}`); err != nil {
		t.Fatalf("insert message: %v", err)
	}
	if _, err := db.Exec(`insert into session_message values (?, ?, ?, ?, ?, ?)`, "m2", "ses_1", "idle", 2, 200, "{}"); err != nil {
		t.Fatalf("insert idle: %v", err)
	}

	entries := scanOpencodeDB(path, time.UnixMilli(0))
	if len(entries) != 1 {
		t.Fatalf("entries = %d", len(entries))
	}
	session := entries[0].session
	if session.Agent != "opencode" || session.ProjectName != "demo" || session.Title != "Fix login" {
		t.Errorf("identity = %+v", session)
	}
	if session.Model != "glm-5" {
		t.Errorf("model = %q", session.Model)
	}
	if session.InputTokens != 1000 || session.OutputTokens != 50 || session.CachedTokens != 400 {
		t.Errorf("tokens = %+v", session)
	}
	if session.ContextTokens != 1010 || session.Turns != 1 {
		t.Errorf("context/turns = %d/%d", session.ContextTokens, session.Turns)
	}
	if !entries[0].waiting {
		t.Error("an idle message must mark the session waiting")
	}
}
