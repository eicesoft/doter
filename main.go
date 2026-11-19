package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"doter/internal/agents"
	"doter/internal/alerts"
	"doter/internal/clipboard"
	"doter/internal/notch"
	"doter/internal/prefs"
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

// clipboardHotkey is the global shortcut that opens the clipboard tab.
const clipboardHotkey = "CmdOrCtrl+Shift+V"

func init() {
	// Typed custom event: the binding generator exposes matching TypeScript
	// types for `sysinfo:update`.
	application.RegisterEvent[sysinfo.Snapshot](services.EventSnapshot)
	// The same for `clipboard:update`.
	application.RegisterEvent[clipboard.Entry](services.EventClipboard)
	// And for `agents:update`, the Claude/Codex monitor.
	application.RegisterEvent[agents.Snapshot](services.EventAgents)
	// Alerts are also forwarded to the frontend so the panel can stay in sync
	// with the native notification.
	application.RegisterEvent[alerts.Alert](services.EventAlert)
	// A shortcut or notification click asks the frontend to open a tab.
	application.RegisterEvent[string](services.EventOpenTab)
}

func main() {
	sampler := sysinfo.NewSampler()
	systemService := services.NewSystemService(sampler)

	prefsStore := prefs.New(prefsPath())
	clipboardHistory := clipboard.NewHistory(clipboardHistoryPath())
	clipboardWatcher := clipboard.NewWatcher(clipboardHistory)
	notificationService := notifications.New()

	app := application.New(application.Options{
		Name:        "Doter",
		Description: "A system monitor that lives in the notch",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(systemService),
			application.NewService(notificationService),
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

	alertsEnabled := func() bool { return prefsStore.Get().AlertsEnabled }
	alertEngine := alerts.New(alertsEnabled, func(alert alerts.Alert) {
		deliverAlert(app, notificationService, alert)
	})

	openClipboard := func() { openTab(app, island, "clipboard") }
	setClipboardHotkey := func(enabled bool) {
		if enabled {
			if app.GlobalShortcut.IsRegistered(clipboardHotkey) {
				return
			}
			if err := app.GlobalShortcut.Register(clipboardHotkey, openClipboard); err != nil {
				log.Printf("could not register clipboard shortcut: %v", err)
			}
			return
		}
		if !app.GlobalShortcut.IsRegistered(clipboardHotkey) {
			return
		}
		if err := app.GlobalShortcut.Unregister(clipboardHotkey); err != nil {
			log.Printf("could not unregister clipboard shortcut: %v", err)
		}
	}

	settingsService := services.NewSettingsService(
		prefsStore,
		clipboardWatcher,
		func() { go requestNotificationPermission(notificationService) },
		setClipboardHotkey,
	)
	app.RegisterService(application.NewService(settingsService))

	weatherClient := weather.NewClient(weatherLocationPath())
	app.RegisterService(application.NewService(services.NewWeatherService(weatherClient)))
	app.RegisterService(application.NewService(services.NewMusicService(app)))

	agentMonitor := agents.NewMonitor()
	app.RegisterService(application.NewService(services.NewAgentsService(agentMonitor)))

	// A notification click should bring the island back and open the tab that
	// explains the alert.
	notificationService.OnNotificationResponse(func(result notifications.NotificationResult) {
		if result.Error != nil {
			log.Printf("notification response: %v", result.Error)
			return
		}
		tab, _ := result.Response.UserInfo["tab"].(string)
		if tab == "" {
			tab = "overview"
		}
		openTab(app, island, tab)
	})

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
	// CmdOrCtrl is the convention for a clipboard manager. Registration can
	// fail when another app owns the shortcut; the rest of the app still works.
	if prefsStore.Get().ClipboardHotkeyEnabled {
		setClipboardHotkey(true)
	}

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		island.PlaceAndShow()
		sampler.Start(context.Background())
		go pumpSnapshots(app, sampler, alertEngine)
		clipboardWatcher.Start(context.Background())
		go pumpClipboard(app, clipboardWatcher)
		agentMonitor.Start(context.Background())
		go pumpAgents(app, agentMonitor, alertEngine)
		if alertsEnabled() {
			go requestNotificationPermission(notificationService)
		}
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

// prefsPath is where backend preferences (alerts, clipboard pause and hotkey)
// live.
func prefsPath() string {
	return configPath("settings.json")
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

func pumpSnapshots(app *application.App, sampler *sysinfo.Sampler, engine *alerts.Engine) {
	for snapshot := range sampler.Updates() {
		app.Event.Emit(services.EventSnapshot, snapshot)
		engine.ObserveSnapshot(snapshot)
	}
}

// pumpClipboard forwards new clipboard entries to the frontend.
func pumpClipboard(app *application.App, watcher *clipboard.Watcher) {
	for entry := range watcher.Updates() {
		app.Event.Emit(services.EventClipboard, entry)
	}
}

// pumpAgents forwards agent session snapshots to the frontend.
func pumpAgents(app *application.App, monitor *agents.Monitor, engine *alerts.Engine) {
	for snapshot := range monitor.Updates() {
		app.Event.Emit(services.EventAgents, snapshot)
		engine.ObserveAgents(snapshot)
	}
}

// openTab shows the island, focuses it and asks the frontend to pin the
// requested tab. The frontend owns the resize animation, so this only asks.
func openTab(app *application.App, island *notch.Controller, tab string) {
	island.Show()
	island.Focus()
	app.Event.Emit(services.EventOpenTab, tab)
}

// deliverAlert fans one alert out to the UI and the native notification
// centre. The notification is best-effort: a missing permission must not
// break the sampler.
func deliverAlert(app *application.App, service *notifications.NotificationService, alert alerts.Alert) {
	app.Event.Emit(services.EventAlert, alert)

	// Dispatch away from the sampler goroutine: a slow platform notification
	// daemon must not delay the next system reading.
	go func() {
		if err := service.SendNotification(notifications.NotificationOptions{
			ID:       alert.ID,
			Title:    alert.Title,
			Body:     alert.Body,
			ThreadID: "doter-alerts",
			Data:     map[string]interface{}{"tab": alert.Tab},
		}); err != nil {
			log.Printf("send notification %q: %v", alert.Kind, err)
		}
	}()
}

// requestNotificationPermission asks the platform once per call site. The
// result is only logged: the app keeps running when notifications are denied.
func requestNotificationPermission(service *notifications.NotificationService) {
	allowed, err := service.RequestNotificationAuthorization()
	if err != nil {
		log.Printf("notification authorization: %v", err)
		return
	}
	if !allowed {
		log.Printf("notifications are not authorized; alerts will stay silent")
	}
}
