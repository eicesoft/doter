package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"doter/internal/agents"
	"doter/internal/clipboard"
	"doter/internal/notch"
	"doter/internal/services"
	"doter/internal/sysinfo"
	"doter/internal/weather"
)

// The React frontend is embedded into the binary at build time.
//
//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/tray.png
var trayIcon []byte

func init() {
	// Typed custom event: the binding generator exposes matching TypeScript
	// types for `sysinfo:update`.
	application.RegisterEvent[sysinfo.Snapshot](services.EventSnapshot)
	// The same for `clipboard:update`.
	application.RegisterEvent[clipboard.Entry](services.EventClipboard)
	// And for `agents:update`, the Claude/Codex monitor.
	application.RegisterEvent[agents.Snapshot](services.EventAgents)
}

func main() {
	sampler := sysinfo.NewSampler()
	systemService := services.NewSystemService(sampler)

	clipboardHistory := clipboard.NewHistory(clipboardHistoryPath())
	clipboardWatcher := clipboard.NewWatcher(clipboardHistory)

	app := application.New(application.Options{
		Name:        "Doter",
		Description: "A system monitor that lives in the notch",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(systemService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// A menu-bar style utility: no Dock icon, no app switcher entry.
			ActivationPolicy: application.ActivationPolicyAccessory,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Windows: application.WindowsOptions{
			// Closing the island must not terminate the monitor.
			DisableQuitOnLastWindowClosed: true,
		},
		Linux: application.LinuxOptions{
			DisableQuitOnLastWindowClosed: true,
		},
	})

	island := notch.New(app, notch.Options{
		Name:  "island",
		Title: "Doter",
		URL:   "/",
	})
	app.RegisterService(application.NewService(services.NewIslandService(island, app)))
	app.RegisterService(application.NewService(services.NewClipboardService(clipboardHistory, app)))

	weatherClient := weather.NewClient(weatherLocationPath())
	app.RegisterService(application.NewService(services.NewWeatherService(weatherClient)))
	app.RegisterService(application.NewService(services.NewMusicService(app)))

	agentMonitor := agents.NewMonitor()
	app.RegisterService(application.NewService(services.NewAgentsService(agentMonitor)))

	tray := app.SystemTray.New()
	tray.SetTooltip("Doter — system monitor")
	tray.SetIcon(trayIcon)
	tray.SetMenu(buildTrayMenu(app, island))

	// A global shortcut is the quickest way to bring the island back after a
	// hide, especially on Windows where the tray overflow can swallow icons.
	if err := app.GlobalShortcut.Register("Ctrl+Shift+D", func() {
		island.ToggleVisible()
	}); err != nil {
		log.Printf("could not register global shortcut: %v", err)
	}

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		island.PlaceAndShow()
		sampler.Start(context.Background())
		go pumpSnapshots(app, sampler)
		clipboardWatcher.Start(context.Background())
		go pumpClipboard(app, clipboardWatcher)
		agentMonitor.Start(context.Background())
		go pumpAgents(app, agentMonitor)
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// clipboardHistoryPath is where the clipboard history is persisted between
// runs.
func clipboardHistoryPath() string {
	return configPath("clipboard.json")
}

// weatherLocationPath is where the pinned weather location is persisted.
func weatherLocationPath() string {
	return configPath("weather.json")
}

// configPath returns the path of a file in the app's config directory.
func configPath(name string) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "doter", name)
}

func buildTrayMenu(app *application.App, island *notch.Controller) *application.Menu {
	menu := application.NewMenu()

	menu.Add("Show / Hide island").OnClick(func(*application.Context) {
		island.ToggleVisible()
	})
	menu.AddSeparator()
	menu.Add("Quit Doter").OnClick(func(*application.Context) {
		app.Quit()
	})

	return menu
}

func pumpSnapshots(app *application.App, sampler *sysinfo.Sampler) {
	for snapshot := range sampler.Updates() {
		app.Event.Emit(services.EventSnapshot, snapshot)
	}
}

// pumpClipboard forwards new clipboard entries to the frontend.
func pumpClipboard(app *application.App, watcher *clipboard.Watcher) {
	for entry := range watcher.Updates() {
		app.Event.Emit(services.EventClipboard, entry)
	}
}

// pumpAgents forwards agent session snapshots to the frontend.
func pumpAgents(app *application.App, monitor *agents.Monitor) {
	for snapshot := range monitor.Updates() {
		app.Event.Emit(services.EventAgents, snapshot)
	}
}
