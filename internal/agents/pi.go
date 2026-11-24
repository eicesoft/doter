package agents

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// piState is the incremental parse state of one pi/omp session log.
type piState struct {
	offset   int64
	waiting  bool
	awaiting bool
	session  Session
	title    string
	prompt   string
}

func (s *piState) reset(agent string) {
	s.offset = 0
	s.waiting = false
	s.awaiting = false
	s.session = Session{Agent: agent}
	s.title = ""
	s.prompt = ""
}

// piRecord is one JSONL line of a pi or omp session log. Both projects are
// descendants of the same harness, so one decoder covers their shared records;
// the fields that differ (title vs session_info, model vs provider/modelId)
// are simply optional here.
type piRecord struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	ID        string `json:"id"`
	Cwd       string `json:"cwd"`
	Runtime   struct {
		Version string `json:"version"`
	} `json:"runtime"`

	// session_info (pi) and title/title_change (omp).
	Name  string `json:"name"`
	Title string `json:"title"`

	// model_change: pi splits provider and modelId, omp joins them into model.
	Provider string `json:"provider"`
	ModelID  string `json:"modelId"`
	Model    string `json:"model"`

	Message piMessage `json:"message"`
}

type piMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	Model      string          `json:"model"`
	StopReason string          `json:"stopReason"`
	Usage      struct {
		Input      int64 `json:"input"`
		Output     int64 `json:"output"`
		CacheRead  int64 `json:"cacheRead"`
		CacheWrite int64 `json:"cacheWrite"`
	} `json:"usage"`
}

// scanPi reads every pi and omp session log touched within the window.
func (m *Monitor) scanPi(now time.Time) []*sessionEntry {
	roots := []struct{ agent, root string }{
		{"pi", filepath.Join(m.home, ".pi", "agent", "sessions")},
		{"omp", filepath.Join(m.home, ".omp", "agent", "sessions")},
	}

	cutoff := now.Add(-sessionWindow)
	seen := make(map[string]bool)
	entries := make([]*sessionEntry, 0, 16)

	for _, candidate := range roots {
		projectDirs, err := os.ReadDir(candidate.root)
		if err != nil {
			continue
		}
		for _, projectDir := range projectDirs {
			if !projectDir.IsDir() {
				continue
			}
			dir := filepath.Join(candidate.root, projectDir.Name())
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

				state := m.pi[path]
				if state == nil {
					state = &piState{}
					state.reset(candidate.agent)
					m.pi[path] = state
				}
				if state.session.ID == "" {
					state.session.ID = strings.TrimSuffix(file.Name(), ".jsonl")
				}
				parsePiFile(path, info, state)

				if state.session.LastActivity == 0 {
					state.session.LastActivity = info.ModTime().UnixMilli()
				}
				if state.session.StartedAt == 0 {
					state.session.StartedAt = info.ModTime().UnixMilli()
				}
				state.session.Title = piTitle(state)
				entries = append(entries, &sessionEntry{session: state.session, waiting: state.waiting, awaiting: state.awaiting})
			}
		}
	}

	for path := range m.pi {
		if !seen[path] {
			delete(m.pi, path)
		}
	}
	return entries
}

// parsePiFile decodes only the bytes appended since the last sample.
func parsePiFile(path string, info os.FileInfo, state *piState) {
	if info.Size() < state.offset {
		state.reset(state.session.Agent)
	}
	readAppended(path, state.offset, func(offset int64, data []byte) {
		for _, line := range bytes.Split(data, []byte{'\n'}) {
			state.consume(line)
		}
		state.offset = offset
	})
}

func (st *piState) consume(line []byte) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	var record piRecord
	if json.Unmarshal(line, &record) != nil {
		return
	}
	touch(&st.session, parseTimestamp(record.Timestamp))

	switch record.Type {
	case "session":
		if record.ID != "" {
			st.session.ID = record.ID
		}
		if record.Cwd != "" {
			setProject(&st.session, record.Cwd)
		}
		if record.Runtime.Version != "" {
			st.session.Version = record.Runtime.Version
		}

	case "model_change":
		switch {
		case record.ModelID != "":
			st.session.Model = record.ModelID
		case record.Model != "":
			st.session.Model = record.Model
		}

	case "session_info":
		if record.Name != "" {
			st.title = cleanTitle(record.Name)
		}

	case "title", "title_change":
		if record.Title != "" {
			st.title = cleanTitle(record.Title)
		}

	case "message":
		st.consumeMessage(record.Message)
	}
}

func (st *piState) consumeMessage(message piMessage) {
	switch message.Role {
	case "user":
		st.waiting = false
		st.awaiting = false
		if st.prompt == "" {
			st.prompt = firstPiText(message.Content)
		}

	case "assistant":
		if reason := message.StopReason; reason != "" {
			// pi/omp keep the turn open while they call tools; anything else
			// (stop, length, error, aborted) ends it and waits for the user.
			tool := reason == "toolUse" || reason == "toolCall"
			st.waiting = !tool
			st.awaiting = tool
		}
		if message.Model != "" {
			st.session.Model = message.Model
		}
		usage := message.Usage
		if usage.Input+usage.Output+usage.CacheRead+usage.CacheWrite > 0 {
			st.session.InputTokens += usage.Input
			st.session.OutputTokens += usage.Output
			st.session.CachedTokens += usage.CacheRead
			st.session.ContextTokens = usage.Input + usage.CacheRead + usage.CacheWrite
			st.session.Turns++
		}
	}
}

func piTitle(state *piState) string {
	for _, candidate := range []string{state.title, state.prompt} {
		if candidate != "" {
			return candidate
		}
	}
	if len(state.session.ID) >= 8 {
		return state.session.ID[:8]
	}
	return state.session.ID
}

// firstPiText pulls the first human-readable block out of a message content.
func firstPiText(raw json.RawMessage) string {
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
