package notice

import (
	"testing"

	"howett.net/plist"
)

func TestParseToastPayload(t *testing.T) {
	raw := []byte(`<toast><visual><binding template="ToastGeneric"><text>标题</text><text>正文一</text><text>正文二</text></binding></visual></toast>`)
	title, body := parseToastPayload(raw)
	if title != "标题" || body != "正文一 正文二" {
		t.Fatalf("got %q / %q", title, body)
	}
}

func TestParseToastPayloadRejectsGarbage(t *testing.T) {
	title, body := parseToastPayload([]byte("not xml"))
	if title != "" || body != "" {
		t.Fatalf("got %q / %q", title, body)
	}
}

func TestAppleTime(t *testing.T) {
	// 2026-01-01T00:00:00Z is 788918400 seconds after 2001-01-01.
	if got := appleToUnixMilli(788918400); got != 1767225600000 {
		t.Fatalf("got %d", got)
	}
}

func TestFiletime(t *testing.T) {
	// 2026-01-01T00:00:00Z in FILETIME ticks.
	const ticks = 134116992000000000
	if got := filetimeToUnixMilli(ticks); got != 1767225600000 {
		t.Fatalf("got %d", got)
	}
}

func TestDecodeDarwin(t *testing.T) {
	data, err := plist.Marshal(map[string]any{
		"app":  "com.apple.mail",
		"date": 788918400.5,
		"req":  map[string]any{"titl": "主题", "body": "正文"},
	}, plist.BinaryFormat)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := decodeDarwin(data)
	if !ok {
		t.Fatal("decode failed")
	}
	if got.App != "com.apple.mail" || got.Req.Title != "主题" || got.Req.Body != "正文" {
		t.Fatalf("got %+v", got)
	}
	if ms := appleToUnixMilli(got.Date); ms != 1767225600500 {
		t.Fatalf("date = %d", ms)
	}
}

func TestSQLiteDSN(t *testing.T) {
	if got := sqliteDSN("/Users/a b/notifications.db"); got != "file:/Users/a%20b/notifications.db?mode=ro" {
		t.Fatalf("got %q", got)
	}
}

func TestMonitorBaselineIsSilent(t *testing.T) {
	m := NewMonitor()
	m.fetch = func(string, int) ([]Entry, string, error) {
		return []Entry{{ID: "1", Title: "旧"}}, "1", nil
	}

	m.sample()
	snap := m.Current()
	if len(snap.Entries) != 1 || len(snap.New) != 0 {
		t.Fatalf("baseline snapshot = %+v", snap)
	}
	drain(m.updates)
}

func TestMonitorReportsNewEntries(t *testing.T) {
	m := NewMonitor()
	step := 0
	m.fetch = func(string, int) ([]Entry, string, error) {
		step++
		if step == 1 {
			return []Entry{{ID: "1", Title: "旧"}}, "1", nil
		}
		return []Entry{{ID: "2", Title: "新"}}, "2", nil
	}

	m.sample()
	drain(m.updates)
	m.sample()

	snap := m.Current()
	if len(snap.Entries) != 2 || snap.Entries[0].ID != "2" || snap.Entries[1].ID != "1" {
		t.Fatalf("entries = %+v", snap.Entries)
	}
	if len(snap.New) != 1 || snap.New[0].ID != "2" {
		t.Fatalf("new = %+v", snap.New)
	}
	drain(m.updates)

	// A sample with nothing new must not wake the UI.
	m.fetch = func(string, int) ([]Entry, string, error) { return nil, "2", nil }
	m.sample()
	select {
	case <-m.updates:
		t.Fatal("quiet sample published a snapshot")
	default:
	}
}

func TestMonitorClear(t *testing.T) {
	m := NewMonitor()
	m.fetch = func(string, int) ([]Entry, string, error) {
		return []Entry{{ID: "1", Title: "旧"}}, "1", nil
	}
	m.sample()
	drain(m.updates)

	m.Clear()
	if entries := m.Current().Entries; len(entries) != 0 {
		t.Fatalf("entries after clear = %+v", entries)
	}

	// The cursor survives, so the next sample only reports genuinely new
	// notifications instead of reloading the cleared history.
	var after string
	m.fetch = func(cursor string, _ int) ([]Entry, string, error) {
		after = cursor
		return []Entry{{ID: "2", Title: "新"}}, "2", nil
	}
	m.sample()
	if after != "1" {
		t.Fatalf("cursor after clear = %q", after)
	}
	if snap := m.Current(); len(snap.New) != 1 || snap.New[0].ID != "2" {
		t.Fatalf("new after clear = %+v", snap.New)
	}
}

func TestMonitorRemove(t *testing.T) {
	m := NewMonitor()
	m.fetch = func(string, int) ([]Entry, string, error) {
		return []Entry{{ID: "1"}, {ID: "2", App: "a"}, {ID: "3", App: "a"}}, "3", nil
	}
	m.sample()
	drain(m.updates)

	m.Remove([]string{"1", "3"})
	entries := m.Current().Entries
	if len(entries) != 1 || entries[0].ID != "2" {
		t.Fatalf("entries after remove = %+v", entries)
	}
	drain(m.updates)

	// Re-removing an absent id must not publish another snapshot.
	m.Remove([]string{"1"})
	select {
	case <-m.updates:
		t.Fatal("removing an absent id published a snapshot")
	default:
	}
}

func drain(ch <-chan Snapshot) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
