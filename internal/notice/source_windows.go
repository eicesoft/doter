//go:build windows

package notice

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

func platformFetch(after string, limit int) ([]Entry, string, error) {
	path := filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows",
		"Notifications", "wpndatabase.db")
	if _, err := os.Stat(path); err != nil {
		return nil, after, err
	}

	since, _ := strconv.ParseInt(after, 10, 64)
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, after, err
	}
	defer db.Close()

	// ponytail: the wpndatabase schema has shifted between Windows releases;
	// this is the Win10/11 shape. If the join fails on an older build, fall
	// back to reading Notification without the handler table.
	rows, err := db.Query(`
		SELECT n.Id, n.Payload, n.ArrivalTime, COALESCE(h.PrimaryId, '')
		FROM Notification n
		LEFT JOIN NotificationHandler h ON h.Id = n.HandlerId
		WHERE n.Id > ?
		ORDER BY n.Id DESC
		LIMIT ?`, since, limit)
	if err != nil {
		return nil, after, err
	}
	defer rows.Close()

	entries := make([]Entry, 0, limit)
	cursor := since
	for rows.Next() {
		var (
			id      int64
			payload []byte
			arrival sql.NullInt64
			aumid   string
		)
		if err := rows.Scan(&id, &payload, &arrival, &aumid); err != nil {
			continue
		}
		if id > cursor {
			cursor = id
		}

		title, body := parseToastPayload(payload)
		if (title == "" && body == "") || strings.Contains(strings.ToLower(aumid), "doter") {
			continue
		}
		var date int64
		if arrival.Valid {
			date = filetimeToUnixMilli(arrival.Int64)
		}
		entries = append(entries, Entry{
			ID:    "win:" + strconv.FormatInt(id, 10),
			App:   aumid,
			Title: title,
			Body:  body,
			Date:  date,
		})
	}
	return entries, strconv.FormatInt(cursor, 10), rows.Err()
}
