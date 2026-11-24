package agents

import (
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// scanOpencode reads OpenCode's SQLite session store. The database is opened
// read-only for each sample; WAL mode lets that coexist with a running CLI.
func (m *Monitor) scanOpencode(now time.Time) []*sessionEntry {
	dbPath := filepath.Join(m.home, ".local", "share", "opencode", "opencode.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}
	return scanOpencodeDB(dbPath, now.Add(-sessionWindow))
}

func scanOpencodeDB(dbPath string, cutoff time.Time) []*sessionEntry {
	dsn := url.URL{Scheme: "file", Path: dbPath, RawQuery: "mode=ro&_pragma=busy_timeout(2000)"}

	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	rows, err := db.Query("select s.id, coalesce(s.title, ''), s.slug, s.directory, coalesce(s.model, ''), s.version, s.tokens_input, s.tokens_output, s.tokens_cache_read, s.time_created, s.time_updated, coalesce((select m.type from session_message m where m.session_id = s.id order by m.seq desc limit 1), ''), coalesce((select count(*) from session_message m where m.session_id = s.id and m.type = 'assistant'), 0), coalesce((select m.data from session_message m where m.session_id = s.id and m.type = 'assistant' order by m.seq desc limit 1), '') from session_v2 s where s.parent_id is null and s.time_updated >= ? order by s.time_updated desc limit 60", cutoff.UnixMilli())
	if err != nil {
		return nil
	}
	defer rows.Close()

	entries := make([]*sessionEntry, 0, 8)
	for rows.Next() {
		var (
			id, title, slug, directory, model, version string
			input, output, cached                      int64
			created, updated                           int64
			lastType, lastReply                        string
			turns                                      int
		)
		if err := rows.Scan(&id, &title, &slug, &directory, &model, &version,
			&input, &output, &cached, &created, &updated, &lastType, &turns, &lastReply); err != nil {
			continue
		}

		session := Session{
			ID:           id,
			Agent:        "opencode",
			Version:      version,
			Title:        cleanTitle(title),
			Model:        opencodeModel(model),
			StartedAt:    created,
			LastActivity: updated,
			InputTokens:  input,
			OutputTokens: output,
			CachedTokens: cached,
			Turns:        turns,
		}
		if session.Title == "" {
			session.Title = cleanTitle(slug)
		}
		if directory != "" {
			setProject(&session, directory)
		}

		entries = append(entries, &sessionEntry{
			session:  session,
			waiting:  lastType == "idle",
			awaiting: opencodeAwaiting(lastReply),
		})
	}
	rows.Close()

	// The context probe runs after the cursor is released: the read-only DSN
	// serialises the connection, so a second statement cannot overlap it.
	for _, entry := range entries {
		entry.session.ContextTokens = opencodeContext(db, entry.session.ID)
	}
	return entries
}

// opencodeAwaiting reports whether the reply is blocked on a tool call the
// user has to approve: the tool part stays pending until the CLI gets it.
func opencodeAwaiting(raw string) bool {
	var reply struct {
		Content []struct {
			Type  string `json:"type"`
			State struct {
				Status string `json:"status"`
			} `json:"state"`
		} `json:"content"`
	}
	if json.Unmarshal([]byte(raw), &reply) != nil {
		return false
	}
	for _, part := range reply.Content {
		if part.Type == "tool" && part.State.Status == "pending" {
			return true
		}
	}
	return false
}

// opencodeContext reads the prompt size of the session's last assistant reply of the session's last assistant reply.
func opencodeContext(db *sql.DB, sessionID string) int64 {
	var raw string
	row := db.QueryRow("select data from session_message where session_id = ? and type = 'assistant' order by seq desc limit 1", sessionID)
	if row.Scan(&raw) != nil {
		return 0
	}

	var message struct {
		Tokens struct {
			Input int64 `json:"input"`
			Cache struct {
				Read  int64 `json:"read"`
				Write int64 `json:"write"`
			} `json:"cache"`
		} `json:"tokens"`
	}
	if json.Unmarshal([]byte(raw), &message) != nil {
		return 0
	}
	return message.Tokens.Input + message.Tokens.Cache.Read + message.Tokens.Cache.Write
}

// opencodeModel extracts the model id from the stored JSON blob.
func opencodeModel(raw string) string {
	if raw == "" {
		return ""
	}
	var model struct {
		ID string `json:"id"`
	}
	if json.Unmarshal([]byte(raw), &model) != nil {
		return ""
	}
	return model.ID
}
