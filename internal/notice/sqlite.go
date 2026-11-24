package notice

import (
	"net/url"
	"path/filepath"
)

// sqliteDSN opens the store read-only: the platform notification daemon owns
// the file and the app must never write to it.
func sqliteDSN(path string) string {
	u := url.URL{Path: filepath.ToSlash(path)}
	return "file:" + u.EscapedPath() + "?mode=ro"
}
