package agents

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// codexState is the incremental parse state of one Codex rollout log.
type codexState struct {
	offset  int64
	waiting bool
	session Session
	prompt  string
}

func (s *codexState) reset() {
	s.offset = 0
	s.waiting = false
	s.session = Session{Agent: "codex"}
	s.prompt = ""
}

// codexRecord is the outer JSONL envelope; payload is decoded per record type.
type codexRecord struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexUsage struct {
	Input  int64 `json:"input_tokens"`
	Cached int64 `json:"cached_input_tokens"`
	Output int64 `json:"output_tokens"`
	Total  int64 `json:"total_tokens"`
}

type codexEvent struct {
	Type           string `json:"type"`
	ThreadSettings struct {
		Model string `json:"model"`
	} `json:"thread_settings"`
	Info *struct {
		Total *codexUsage `json:"total_token_usage"`
		Last  *codexUsage `json:"last_token_usage"`
	} `json:"info"`
	Item *struct {
		Type    string `json:"type"`
		Phase   string `json:"phase"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"item"`
}

// scanCodex reads every rollout log touched within the window.
func (m *Monitor) scanCodex(now time.Time) []*sessionEntry {
	root := filepath.Join(m.home, ".codex", "sessions")
	if _, err := os.Stat(root); err != nil {
		return nil
	}

	cutoff := now.Add(-sessionWindow)
	seen := make(map[string]bool)
	entries := make([]*sessionEntry, 0, 4)

	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			return nil
		}
		seen[path] = true

		state := m.codex[path]
		if state == nil {
			state = &codexState{}
			state.reset()
			m.codex[path] = state
		}
		if state.session.ID == "" {
			state.session.ID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
		}
		parseCodexFile(path, info, state)

		if state.session.LastActivity == 0 {
			state.session.LastActivity = info.ModTime().UnixMilli()
		}
		if state.session.StartedAt == 0 {
			state.session.StartedAt = info.ModTime().UnixMilli()
		}
		if state.session.Title == "" {
			state.session.Title = state.prompt
		}
		if state.session.Title == "" {
			state.session.Title = state.session.ID
		}
		entries = append(entries, &sessionEntry{session: state.session, waiting: state.waiting})
		return nil
	})

	for path := range m.codex {
		if !seen[path] {
			delete(m.codex, path)
		}
	}
	return entries
}

// parseCodexFile decodes only the bytes appended since the last sample.
func parseCodexFile(path string, info os.FileInfo, state *codexState) {
	if info.Size() < state.offset {
		state.reset()
	}
	readAppended(path, state.offset, func(offset int64, data []byte) {
		for _, line := range bytes.Split(data, []byte{'\n'}) {
			state.consume(line)
		}
		state.offset = offset
	})
}

func (st *codexState) consume(line []byte) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	var record codexRecord
	if json.Unmarshal(line, &record) != nil {
		return
	}
	touch(&st.session, parseTimestamp(record.Timestamp))

	switch record.Type {
	case "session_meta":
		var meta struct {
			ID         string `json:"id"`
			SessionID  string `json:"session_id"`
			Cwd        string `json:"cwd"`
			CliVersion string `json:"cli_version"`
			Timestamp  string `json:"timestamp"`
		}
		if json.Unmarshal(record.Payload, &meta) != nil {
			return
		}
		if meta.ID != "" {
			st.session.ID = meta.ID
		} else if meta.SessionID != "" {
			st.session.ID = meta.SessionID
		}
		if meta.Cwd != "" {
			setProject(&st.session, meta.Cwd)
		}
		if meta.CliVersion != "" {
			st.session.Version = meta.CliVersion
		}
		touch(&st.session, parseTimestamp(meta.Timestamp))

	case "turn_context":
		var ctx struct {
			Model string `json:"model"`
			Cwd   string `json:"cwd"`
		}
		if json.Unmarshal(record.Payload, &ctx) == nil {
			if ctx.Model != "" {
				st.session.Model = ctx.Model
			}
			if ctx.Cwd != "" {
				setProject(&st.session, ctx.Cwd)
			}
		}

	case "token_usage_record":
		var usage struct {
			Thread *codexUsage `json:"thread_token_usage"`
		}
		if json.Unmarshal(record.Payload, &usage) == nil && usage.Thread != nil {
			st.applyUsage(*usage.Thread, nil)
		}

	case "event_msg":
		var event codexEvent
		if json.Unmarshal(record.Payload, &event) != nil {
			return
		}
		st.consumeEvent(event)
	}
}

func (st *codexState) consumeEvent(event codexEvent) {
	switch event.Type {
	case "thread_settings_applied":
		if event.ThreadSettings.Model != "" {
			st.session.Model = event.ThreadSettings.Model
		}
	case "task_started", "turn_started":
		st.waiting = false
	case "task_complete", "turn_complete":
		st.waiting = true
	case "token_count":
		if event.Info != nil {
			var last *codexUsage
			if event.Info.Last != nil {
				last = event.Info.Last
			}
			if event.Info.Total != nil {
				st.applyUsage(*event.Info.Total, last)
			}
		}
	case "item_completed":
		item := event.Item
		if item == nil {
			return
		}
		switch item.Type {
		case "UserMessage":
			st.waiting = false
			if st.prompt == "" {
				st.prompt = firstCodexText(item.Content)
			}
		case "AgentMessage":
			if item.Phase == "final_answer" {
				st.waiting = true
				st.session.Turns++
			}
		}
	}
}

// applyUsage stores cumulative counters. The last request's total is the
// context size; Codex reports it separately from the session totals.
func (st *codexState) applyUsage(total codexUsage, last *codexUsage) {
	st.session.InputTokens = total.Input
	st.session.CachedTokens = total.Cached
	st.session.OutputTokens = total.Output
	if last != nil && last.Total > 0 {
		st.session.ContextTokens = last.Total
	} else if total.Total > 0 {
		st.session.ContextTokens = total.Total
	}
}

func firstCodexText(content []struct {
	Type string `json:"type"`
	Text string `json:"text"`
}) string {
	for _, block := range content {
		if block.Text != "" && looksLikePrompt(block.Text) {
			return cleanTitle(block.Text)
		}
	}
	return ""
}
