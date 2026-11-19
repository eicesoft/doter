package agents

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// claudeState is the incremental parse state of one session log.
type claudeState struct {
	offset   int64
	waiting  bool
	session  Session
	title    string
	prompt   string
	fallback string
}

func (s *claudeState) reset() {
	s.offset = 0
	s.waiting = false
	s.session = Session{Agent: "claude"}
	s.title = ""
	s.prompt = ""
	s.fallback = ""
}

// claudeRecord is one JSONL line of a Claude Code transcript.
type claudeRecord struct {
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	Cwd         string `json:"cwd"`
	GitBranch   string `json:"gitBranch"`
	Version     string `json:"version"`
	SessionID   string `json:"sessionId"`
	IsSidechain bool   `json:"isSidechain"`

	CustomTitle string `json:"customTitle"`
	AgentName   string `json:"agentName"`
	LastPrompt  string `json:"lastPrompt"`

	Message struct {
		Model      string          `json:"model"`
		StopReason string          `json:"stop_reason"`
		Content    json.RawMessage `json:"content"`
		Usage      struct {
			Input         int64 `json:"input_tokens"`
			Output        int64 `json:"output_tokens"`
			CacheRead     int64 `json:"cache_read_input_tokens"`
			CacheCreation int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// scanClaude reads every project transcript touched within the window.
func (m *Monitor) scanClaude(now time.Time) []*sessionEntry {
	root := filepath.Join(m.home, ".claude", "projects")
	projectDirs, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	cutoff := now.Add(-sessionWindow)
	seen := make(map[string]bool)
	entries := make([]*sessionEntry, 0, 16)

	for _, projectDir := range projectDirs {
		if !projectDir.IsDir() {
			continue
		}
		dir := filepath.Join(root, projectDir.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") {
				continue
			}
			path := filepath.Join(dir, file.Name())
			info, err := file.Info()
			if err != nil || info.ModTime().Before(cutoff) {
				continue
			}
			seen[path] = true

			state := m.claude[path]
			if state == nil {
				state = &claudeState{}
				state.reset()
				m.claude[path] = state
			}
			if state.session.ID == "" {
				state.session.ID = strings.TrimSuffix(file.Name(), ".jsonl")
			}
			parseClaudeFile(path, info, state)

			if state.session.LastActivity == 0 {
				state.session.LastActivity = info.ModTime().UnixMilli()
			}
			if state.session.StartedAt == 0 {
				state.session.StartedAt = info.ModTime().UnixMilli()
			}
			state.session.Title = claudeTitle(state)
			entries = append(entries, &sessionEntry{session: state.session, waiting: state.waiting})
		}
	}

	for path := range m.claude {
		if !seen[path] {
			delete(m.claude, path)
		}
	}
	return entries
}

// parseClaudeFile decodes only the bytes appended since the last sample.
func parseClaudeFile(path string, info os.FileInfo, state *claudeState) {
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

func (st *claudeState) consume(line []byte) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	var record claudeRecord
	if json.Unmarshal(line, &record) != nil {
		return
	}
	if record.IsSidechain {
		return
	}

	touch(&st.session, parseTimestamp(record.Timestamp))
	if record.SessionID != "" {
		st.session.ID = record.SessionID
	}
	if record.Cwd != "" {
		setProject(&st.session, record.Cwd)
	}
	if record.Version != "" {
		st.session.Version = record.Version
	}
	if record.GitBranch != "" {
		st.session.Branch = record.GitBranch
	}

	switch record.Type {
	case "custom-title":
		st.title = cleanTitle(record.CustomTitle)
	case "agent-name":
		if st.title == "" {
			st.title = cleanTitle(record.AgentName)
		}
	case "last-prompt":
		if st.fallback == "" {
			st.fallback = cleanTitle(record.LastPrompt)
		}
		st.waiting = false
	case "user":
		st.waiting = false
		if st.prompt == "" {
			st.prompt = firstClaudeText(record.Message.Content)
		}
	case "assistant":
		if reason := record.Message.StopReason; reason != "" {
			st.waiting = reason == "end_turn" || reason == "stop_sequence"
		}
		if model := record.Message.Model; model != "" && !strings.HasPrefix(model, "<") {
			st.session.Model = model
		}
		usage := record.Message.Usage
		if usage.Input+usage.Output+usage.CacheRead+usage.CacheCreation > 0 {
			st.session.InputTokens += usage.Input
			st.session.OutputTokens += usage.Output
			st.session.CachedTokens += usage.CacheRead
			st.session.ContextTokens = usage.Input + usage.CacheRead + usage.CacheCreation
			st.session.Turns++
		}
	case "api-request", "queue-operation":
		st.waiting = false
	}
}

func claudeTitle(state *claudeState) string {
	for _, candidate := range []string{state.title, state.prompt, state.fallback} {
		if candidate != "" {
			return candidate
		}
	}
	if len(state.session.ID) >= 8 {
		return state.session.ID[:8]
	}
	return state.session.ID
}

// firstClaudeText pulls the first human-readable block out of a message body.
func firstClaudeText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil && looksLikePrompt(text) {
		return cleanTitle(text)
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	for _, block := range blocks {
		if block.Type == "text" && looksLikePrompt(block.Text) {
			return cleanTitle(block.Text)
		}
	}
	return ""
}

// readAppended streams the bytes after offset and hands the caller complete
// lines only. A trailing partial line is left for the next sample.
func readAppended(path string, offset int64, consume func(next int64, lines []byte)) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.Size() <= offset {
		return
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return
	}
	end := bytes.LastIndexByte(data, '\n')
	if end < 0 {
		return
	}
	consume(offset+int64(end)+1, data[:end+1])
}

func parseTimestamp(value string) int64 {
	if value == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return 0
	}
	return parsed.UnixMilli()
}

func touch(session *Session, at int64) {
	if at == 0 {
		return
	}
	if session.StartedAt == 0 || at < session.StartedAt {
		session.StartedAt = at
	}
	if at > session.LastActivity {
		session.LastActivity = at
	}
}

func setProject(session *Session, cwd string) {
	// A session's transcript sometimes records a subdirectory after a tool
	// changes directory; the first cwd is the project root.
	if session.Project != "" {
		return
	}
	session.Project = cwd
	name := filepath.Base(cwd)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = cwd
	}
	session.ProjectName = name
}
