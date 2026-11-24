// Package notice reads the operating system's notification store and reports
// notifications as they arrive. Both macOS and Windows keep the store in a
// SQLite database, so the platform files only differ in schema and payload.
package notice

// Entry is one notification shown in the panel.
type Entry struct {
	// ID is unique per platform entry; the UI uses it as the list key.
	ID string `json:"id"`
	// App is the bundle id (macOS) or AUMID (Windows) that posted it.
	App   string `json:"app"`
	Title string `json:"title"`
	Body  string `json:"body"`
	// Date is the delivery time in Unix milliseconds; zero when unknown.
	Date int64 `json:"date"`
}

// Snapshot is the panel's view of the notification history.
type Snapshot struct {
	Entries []Entry `json:"entries"`
	// New lists entries discovered since the previous snapshot. It is
	// non-empty only on the update that found them.
	New []Entry `json:"new"`
	// Error explains why the store could not be read; empty when healthy.
	Error     string `json:"error"`
	UpdatedAt int64  `json:"updatedAt"`
}
