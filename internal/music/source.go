package music

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

// FriendlySource maps a platform media-session identifier (a macOS bundle ID
// or a Windows AUMID) to the name shown in the panel.
func FriendlySource(id string) string {
	lower := strings.ToLower(id)
	switch {
	case strings.Contains(lower, "spotify"):
		return "Spotify"
	case strings.Contains(lower, "apple.music"), strings.Contains(lower, "applemusic"), strings.Contains(lower, "itunes"):
		return "Apple Music"
	case strings.Contains(lower, "qqmusic"):
		return "QQ音乐"
	case strings.Contains(lower, "cloudmusic"), strings.Contains(lower, "163music"):
		return "网易云音乐"
	case strings.Contains(lower, "kugou"):
		return "酷狗音乐"
	case strings.Contains(lower, "kuwo"):
		return "酷我音乐"
	case strings.Contains(lower, "thebrowser"):
		return "Arc"
	case strings.Contains(lower, "safari"):
		return "Safari"
	case strings.Contains(lower, "chrome"):
		return "Chrome"
	case strings.Contains(lower, "msedge"), strings.Contains(lower, "edgemac"):
		return "Edge"
	case strings.Contains(lower, "firefox"):
		return "Firefox"
	case strings.Contains(lower, "zunemusic"), strings.Contains(lower, "mediaplayer"):
		return "媒体播放器"
	case strings.Contains(lower, "vlc"):
		return "VLC"
	case strings.Contains(lower, "iina"):
		return "IINA"
	}

	// Windows AUMIDs are "Package!Entry" strings; browser bundle IDs are
	// dotted. Show the most specific segment rather than the raw identifier.
	if i := strings.LastIndex(id, "!"); i >= 0 && i+1 < len(id) {
		id = id[i+1:]
	}
	id = strings.TrimSuffix(id, ".exe")
	if id == "" {
		return "媒体播放器"
	}
	return id
}

// TrackID identifies a playing item for artwork caching. Platform-specific
// unique identifiers are preferred; otherwise the metadata tuple is hashed.
func TrackID(unique string, parts ...string) string {
	if unique != "" {
		return "uid-" + shortHash(unique)
	}
	return shortHash(strings.Join(parts, "\x1f"))
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:8])
}

// ArtworkDataURI wraps the base64 cover payload reported by the platform
// adapters.
func ArtworkDataURI(mime, base64Data string) string {
	if base64Data == "" {
		return ""
	}
	mime = strings.TrimSpace(mime)
	if mime == "" || !strings.HasPrefix(mime, "image/") {
		mime = "image/jpeg"
	}
	return "data:" + mime + ";base64," + base64Data
}
