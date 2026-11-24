//go:build darwin

package notice

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"howett.net/plist"
)

// iconCache is bounded by the number of distinct apps that notify, so it is
// never pruned; each entry is a few kilobytes.
var iconCache = struct {
	sync.Mutex
	icons map[string]string
}{icons: make(map[string]string)}

// AppIcon returns a PNG data URL for the app's icon, or "" when it cannot be
// resolved.
func AppIcon(bundleID string) string {
	if bundleID == "" || strings.HasPrefix(bundleID, "_system_center_:") ||
		strings.ContainsAny(bundleID, "'\"") {
		return ""
	}

	iconCache.Lock()
	data, ok := iconCache.icons[bundleID]
	iconCache.Unlock()
	if ok {
		return data
	}

	data = loadAppIcon(bundleID)
	iconCache.Lock()
	iconCache.icons[bundleID] = data
	iconCache.Unlock()
	return data
}

func loadAppIcon(bundleID string) string {
	appPath := findApp(bundleID)
	if appPath == "" {
		return ""
	}
	icns := iconFile(appPath)
	if icns == "" {
		return ""
	}

	tmp, err := os.CreateTemp("", "doter-icon-*.png")
	if err != nil {
		return ""
	}
	out := tmp.Name()
	tmp.Close()
	os.Remove(out)
	defer os.Remove(out)

	// sips ships with macOS and converts the multi-resolution .icns to a
	// single small PNG that the webview can display.
	if err := exec.Command("sips", "-s", "format", "png", "-Z", "64", icns, "--out", out).Run(); err != nil {
		return ""
	}
	raw, err := os.ReadFile(out)
	if err != nil || len(raw) == 0 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)
}

func findApp(bundleID string) string {
	out, err := exec.Command("mdfind", "kMDItemCFBundleIdentifier == '"+bundleID+"'").Output()
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		if line = strings.TrimSpace(line); strings.HasSuffix(line, ".app") {
			return line
		}
	}
	return ""
}

// iconFile returns the bundle's icon resource, preferring the file named in
// Info.plist over an arbitrary .icns sibling.
func iconFile(appPath string) string {
	resources := filepath.Join(appPath, "Contents", "Resources")

	if data, err := os.ReadFile(filepath.Join(appPath, "Contents", "Info.plist")); err == nil {
		var info struct {
			Icon string `plist:"CFBundleIconFile"`
		}
		if _, err := plist.Unmarshal(data, &info); err == nil && info.Icon != "" {
			name := info.Icon
			if !strings.HasSuffix(name, ".icns") {
				name += ".icns"
			}
			if path := filepath.Join(resources, name); exists(path) {
				return path
			}
		}
	}

	entries, err := os.ReadDir(resources)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".icns") {
			return filepath.Join(resources, entry.Name())
		}
	}
	return ""
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
