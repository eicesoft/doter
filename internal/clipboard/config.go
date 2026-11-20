package clipboard

// Defaults and bounds for the user-configurable clipboard settings. Limits are
// enforced again by the setters in prefs so a hand-edited settings file cannot
// put the history or the watcher into an unreasonable state.
const (
	DefaultMaxEntries = 100
	MinMaxEntries     = 10
	MaxMaxEntries     = 1000

	// DefaultPreviewBytes is the largest text shipped to the webview. The
	// full value stays on the Go side so a write-back is lossless.
	DefaultPreviewBytes = 4096
	MinPreviewBytes     = 512
	MaxPreviewBytes     = 64 << 10

	DefaultImageMaxBytes int64 = 16 << 20
	MinImageMaxBytes     int64 = 1 << 20
	MaxImageMaxBytes     int64 = 64 << 20
)

// defaultExcludedApps keeps password managers and other secret-bearing apps
// out of the history even when they do not set a concealed/transient marker.
// macOS entries are bundle identifiers; Windows and Linux entries are process
// names. Matching is case-insensitive and ignores .exe.
var defaultExcludedApps = []string{
	"com.1password.1password",
	"com.agilebits.onepassword7",
	"com.apple.keychainaccess",
	"com.apple.passwords",
	"com.bitwarden.desktop",
	"com.dashlane.dashlane",
	"com.lastpass.lastpass",
	"org.keepassxc.keepassxc",
	"1password",
	"bitwarden",
	"dashlane",
	"keepass",
	"keepassxc",
	"lastpass",
}

// DefaultExcludedApps returns a copy so callers cannot mutate the package
// default.
func DefaultExcludedApps() []string {
	return append([]string(nil), defaultExcludedApps...)
}
