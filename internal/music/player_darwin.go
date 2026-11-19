//go:build darwin

package music

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The adapter is a prebuilt, universal, ad-hoc-signed framework driven by a
// perl script. Apple restricted the MediaRemote framework to com.apple.*
// processes in macOS 15.4; /usr/bin/perl carries that signing identifier, so
// the helper still sees the system-wide now-playing session.
const adapterVersion = "0.1.0"

//go:embed assets/mediaremote-adapter/mediaremote-adapter.pl
var adapterScript []byte

//go:embed assets/mediaremote-adapter/MediaRemoteAdapter.framework/Versions/A/MediaRemoteAdapter
var adapterFramework []byte

//go:embed assets/mediaremote-adapter/MediaRemoteAdapter.framework/Versions/A/Resources/Info.plist
var adapterInfoPlist []byte

//go:embed assets/mediaremote-adapter/MediaRemoteAdapter.framework/Versions/A/_CodeSignature/CodeResources
var adapterCodeResources []byte

//go:embed assets/mediaremote-adapter/LICENSE
var adapterLicense []byte

type adapterPaths struct {
	script    string
	framework string
}

type adapterPlayer struct {
	mu sync.RWMutex
	// state is the merged now-playing dictionary. The adapter emits diffs by
	// default, with null values meaning "this key went away".
	state map[string]any
	now   NowPlaying
	err   string

	artwork     map[string]string
	artworkKey  map[string]string
	lastArtData string
	lastArtKey  string

	paths     adapterPaths
	streamCtx context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	closeMu   sync.Mutex
	closed    bool
}

type adapterMessage struct {
	Type    string         `json:"type"`
	Diff    bool           `json:"diff"`
	Payload map[string]any `json:"payload"`
}

func newPlayer() Player {
	return &adapterPlayer{
		state:      map[string]any{},
		artwork:    map[string]string{},
		artworkKey: map[string]string{},
	}
}

func (p *adapterPlayer) Start(ctx context.Context) error {
	paths, err := ensureAdapterAssets()
	if err != nil {
		return err
	}
	// Probe with `get`: an exit code of 0 means the framework loaded and the
	// adapter can talk to MediaRemote (no media is simply "null" on stdout).
	if err := p.run(paths, "get"); err != nil {
		return fmt.Errorf("读取系统媒体失败: %w", err)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	p.closeMu.Lock()
	if p.closed {
		p.closeMu.Unlock()
		cancel()
		return context.Canceled
	}
	p.streamCtx = streamCtx
	p.cancel = cancel
	p.paths = paths
	p.closeMu.Unlock()

	cmd := exec.CommandContext(streamCtx, "/usr/bin/perl", paths.script, paths.framework, "stream")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return err
	}
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("启动媒体适配器失败: %w", err)
	}

	p.closeMu.Lock()
	p.done = make(chan struct{})
	closed := p.closed
	p.closeMu.Unlock()
	go p.readStream(cmd, stdout)

	if closed {
		cancel()
		return context.Canceled
	}
	return nil
}
func (p *adapterPlayer) readStream(cmd *exec.Cmd, stdout io.Reader) {
	defer close(p.done)
	defer p.cancel()

	scanner := bufio.NewScanner(stdout)
	// Cover art arrives as a base64 JPEG inline; a 64 KiB scanner would fail
	// on the first track with artwork.
	scanner.Buffer(make([]byte, 64*1024), 32*1024*1024)

	for scanner.Scan() {
		p.applyLine(scanner.Bytes())
	}

	waitErr := cmd.Wait()
	if p.streamCtx.Err() != nil {
		return // Close asked for it.
	}
	if err := scanner.Err(); err != nil {
		p.fail(fmt.Errorf("媒体更新中断: %w", err))
		return
	}
	if waitErr != nil {
		p.fail(fmt.Errorf("媒体适配器已退出: %w", waitErr))
		return
	}
	p.fail(errors.New("媒体适配器已退出"))
}

func (p *adapterPlayer) applyLine(line []byte) {
	var msg adapterMessage
	if err := json.Unmarshal(line, &msg); err != nil || msg.Type != "data" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if !msg.Diff {
		p.state = make(map[string]any, len(msg.Payload))
	}
	for key, value := range msg.Payload {
		if value == nil {
			delete(p.state, key)
			continue
		}
		p.state[key] = value
	}
	p.now = p.buildLocked()
}

func (p *adapterPlayer) buildLocked() NowPlaying {
	title := adapterString(p.state, "title")
	if title == "" {
		return NowPlaying{Error: p.err}
	}

	artist := adapterString(p.state, "artist")
	album := adapterString(p.state, "album")
	bundle := adapterString(p.state, "bundleIdentifier")
	if bundle == "" || strings.Contains(strings.ToLower(bundle), "webkit") {
		if parent := adapterString(p.state, "parentApplicationBundleIdentifier"); parent != "" {
			bundle = parent
		}
	}

	trackID := TrackID(adapterString(p.state, "uniqueIdentifier"), bundle, title, artist, album)
	p.cacheArtworkLocked(trackID)

	np := NowPlaying{
		Available: true,
		TrackID:   trackID,
		Title:     title,
		Artist:    artist,
		Album:     album,
		Source:    FriendlySource(bundle),

		IsPlaying: adapterBool(p.state, "playing"),
		Position:  adapterFloat(p.state, "elapsedTime"),
		Duration:  adapterFloat(p.state, "duration"),
		Shuffle:   adapterInt(p.state, "shuffleMode") >= 2,
		Repeat:    adapterRepeat(p.state),

		CanPlayPause: true,
		CanPrev:      !adapterBool(p.state, "prohibitsSkip"),
		CanNext:      !adapterBool(p.state, "prohibitsSkip"),
		CanSeek:      adapterFloat(p.state, "duration") > 0,
		CanShuffle:   true,
		CanRepeat:    true,

		ArtworkKey: p.artworkKey[trackID],
	}

	if stamp := adapterString(p.state, "timestamp"); stamp != "" {
		if t, err := time.Parse(time.RFC3339Nano, stamp); err == nil {
			np.UpdatedAt = t.UnixMilli()
		}
	}
	if np.UpdatedAt == 0 {
		np.UpdatedAt = time.Now().UnixMilli()
	}
	return np
}

func (p *adapterPlayer) cacheArtworkLocked(trackID string) {
	data := adapterString(p.state, "artworkData")
	if data == "" {
		return
	}
	if data == p.lastArtData {
		if p.artworkKey[trackID] == "" {
			p.artworkKey[trackID] = p.lastArtKey
		}
		return
	}
	key := shortHash(data)
	p.lastArtData = data
	p.lastArtKey = key
	if p.artworkKey[trackID] == key {
		return
	}
	p.artworkKey[trackID] = key
	p.artwork[trackID] = ArtworkDataURI(adapterString(p.state, "artworkMimeType"), data)
}

func (p *adapterPlayer) Snapshot() NowPlaying {
	p.mu.RLock()
	defer p.mu.RUnlock()
	np := p.now
	if !np.Available && np.Error == "" {
		np.Error = p.err
	}
	return np
}

func (p *adapterPlayer) Artwork(trackID string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.artwork[trackID]
}

func (p *adapterPlayer) fail(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.err = err.Error()
	p.now = NowPlaying{Error: p.err}
}

func (p *adapterPlayer) PlayPause() error {
	return p.command("send", "2")
}

func (p *adapterPlayer) Next() error {
	return p.command("send", "4")
}

func (p *adapterPlayer) Previous() error {
	return p.command("send", "5")
}

func (p *adapterPlayer) Seek(seconds float64) error {
	micros := int64(seconds * 1e6)
	if micros < 0 {
		return errors.New("播放位置无效")
	}
	return p.command("seek", strconv.FormatInt(micros, 10))
}

func (p *adapterPlayer) ToggleShuffle() error {
	return p.command("send", "6")
}

func (p *adapterPlayer) CycleRepeat() error {
	return p.command("send", "7")
}

func (p *adapterPlayer) command(args ...string) error {
	p.mu.RLock()
	paths := p.paths
	p.mu.RUnlock()
	if paths.script == "" {
		return errors.New("媒体适配器尚未就绪")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/bin/perl", append([]string{paths.script, paths.framework}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			return fmt.Errorf("媒体控制失败: %w", err)
		}
		return fmt.Errorf("媒体控制失败: %s", message)
	}
	return nil
}

func (p *adapterPlayer) run(paths adapterPaths, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/perl", append([]string{paths.script, paths.framework}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			return err
		}
		return errors.New(message)
	}
	return nil
}

func (p *adapterPlayer) Close() {
	p.closeMu.Lock()
	if p.closed {
		p.closeMu.Unlock()
		return
	}
	p.closed = true
	cancel := p.cancel
	done := p.done
	p.closeMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// ensureAdapterAssets writes the embedded helper to the user cache once per
// adapter version. The framework is rebuilt with its canonical symlink layout
// so codesign can still validate it and dlopen resolves the usual path.
func ensureAdapterAssets() (adapterPaths, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		root = os.TempDir()
	}
	dir := filepath.Join(root, "doter", "mediaremote-adapter", adapterVersion)
	framework := filepath.Join(dir, "MediaRemoteAdapter.framework")

	paths := adapterPaths{
		script:    filepath.Join(dir, "mediaremote-adapter.pl"),
		framework: framework,
	}
	marker := filepath.Join(dir, ".ready")
	if _, err := os.Stat(marker); err == nil {
		return paths, nil
	}

	versionDir := filepath.Join(framework, "Versions", "A")
	if err := os.MkdirAll(filepath.Join(versionDir, "_CodeSignature"), 0o755); err != nil {
		return adapterPaths{}, err
	}
	if err := os.MkdirAll(filepath.Join(versionDir, "Resources"), 0o755); err != nil {
		return adapterPaths{}, err
	}
	files := []struct {
		path string
		data []byte
		mode os.FileMode
	}{
		{paths.script, adapterScript, 0o644},
		{filepath.Join(versionDir, "MediaRemoteAdapter"), adapterFramework, 0o755},
		{filepath.Join(versionDir, "Resources", "Info.plist"), adapterInfoPlist, 0o644},
		{filepath.Join(versionDir, "_CodeSignature", "CodeResources"), adapterCodeResources, 0o644},
		{filepath.Join(dir, "LICENSE"), adapterLicense, 0o644},
	}
	for _, file := range files {
		if err := writeAtomic(file.path, file.data, file.mode); err != nil {
			return adapterPaths{}, err
		}
	}

	links := []struct{ path, target string }{
		{filepath.Join(framework, "Versions", "Current"), "A"},
		{filepath.Join(framework, "MediaRemoteAdapter"), filepath.Join("Versions", "Current", "MediaRemoteAdapter")},
		{filepath.Join(framework, "Resources"), filepath.Join("Versions", "Current", "Resources")},
	}
	for _, link := range links {
		if _, err := os.Lstat(link.path); err == nil {
			continue
		}
		if err := os.Symlink(link.target, link.path); err != nil {
			return adapterPaths{}, err
		}
	}
	if err := writeAtomic(marker, []byte(adapterVersion), 0o644); err != nil {
		return adapterPaths{}, err
	}
	return paths, nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Chmod(tmp, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func adapterString(state map[string]any, key string) string {
	switch value := state[key].(type) {
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	default:
		return ""
	}
}

func adapterFloat(state map[string]any, key string) float64 {
	switch value := state[key].(type) {
	case float64:
		return value
	case string:
		v, _ := strconv.ParseFloat(value, 64)
		return v
	default:
		return 0
	}
}

func adapterInt(state map[string]any, key string) int {
	switch value := state[key].(type) {
	case float64:
		return int(value)
	case string:
		v, _ := strconv.Atoi(value)
		return v
	default:
		return 0
	}
}

func adapterBool(state map[string]any, key string) bool {
	switch value := state[key].(type) {
	case bool:
		return value
	case float64:
		return value != 0
	case string:
		return value == "1" || strings.EqualFold(value, "true")
	default:
		return false
	}
}

func adapterRepeat(state map[string]any) RepeatMode {
	switch adapterInt(state, "repeatMode") {
	case 2:
		return RepeatTrack
	case 3:
		return RepeatAll
	default:
		return RepeatOff
	}
}
