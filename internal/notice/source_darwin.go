//go:build darwin

package notice

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"
)

// selfBundleID matches singleInstanceID in main.go; Doter's own alerts must
// not come back into the notification panel.
const selfBundleID = "com.example.doter"

func platformFetch(after string, limit int) ([]Entry, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, after, err
	}
	path := filepath.Join(home, "Library", "Group Containers",
		"group.com.apple.usernoted", "db2", "db")
	if _, err := os.Stat(path); err != nil {
		return nil, after, err
	}

	since, _ := strconv.ParseInt(after, 10, 64)
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, after, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT r.rec_id, r.data, r.delivered_date, COALESCE(a.identifier, '')
		FROM record r LEFT JOIN app a ON a.app_id = r.app_id
		WHERE r.rec_id > ?
		ORDER BY r.rec_id DESC
		LIMIT ?`, since, limit)
	if err != nil {
		return nil, after, err
	}
	defer rows.Close()

	entries := make([]Entry, 0, limit)
	cursor := since
	for rows.Next() {
		var (
			recID      int64
			data       []byte
			delivered  sql.NullFloat64
			identifier string
		)
		if err := rows.Scan(&recID, &data, &delivered, &identifier); err != nil {
			continue
		}
		if recID > cursor {
			cursor = recID
		}

		payload, ok := decodeDarwin(data)
		if !ok {
			continue
		}
		app := payload.App
		if app == "" {
			app = identifier
		}
		if app == selfBundleID || (payload.Req.Title == "" && payload.Req.Body == "") {
			continue
		}
		date := appleToUnixMilli(payload.Date)
		if date == 0 && delivered.Valid {
			date = appleToUnixMilli(delivered.Float64)
		}
		entries = append(entries, Entry{
			ID:    "mac:" + strconv.FormatInt(recID, 10),
			App:   app,
			Title: payload.Req.Title,
			Body:  payload.Req.Body,
			Date:  date,
		})
	}
	return entries, strconv.FormatInt(cursor, 10), rows.Err()
}
