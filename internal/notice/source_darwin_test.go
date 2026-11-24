//go:build darwin

package notice

import (
	"strings"
	"testing"
)

// TestPlatformFetchDarwin reads the live notification store. It skips when
// the database is missing or the process lacks access to it.
func TestPlatformFetchDarwin(t *testing.T) {
	entries, cursor, err := platformFetch("", 5)
	if err != nil {
		t.Skipf("notification store unavailable: %v", err)
	}
	if cursor == "" {
		t.Fatal("empty cursor")
	}
	for _, entry := range entries {
		if entry.ID == "" || entry.App == "" {
			t.Fatalf("incomplete entry: %+v", entry)
		}
		if entry.Title == "" && entry.Body == "" {
			t.Fatalf("empty entry: %+v", entry)
		}
	}
}

func TestAppIconDarwin(t *testing.T) {
	icon := AppIcon("com.apple.mail")
	if icon == "" {
		t.Skip("icon lookup unavailable")
	}
	if !strings.HasPrefix(icon, "data:image/png;base64,") {
		t.Fatalf("unexpected icon: %.60s", icon)
	}
}
