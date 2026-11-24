// Package notch owns the top-of-screen window that hosts the island UI.
//
// The window is a frameless, transparent, always-on-top panel that shrinks to
// a notification-like pill when idle and grows into a dashboard when the user
// hovers it. macOS uses a non-activating NSPanel above the menu bar, while
// Windows and Linux use frameless transparent windows; all three share the
// same positioning rules so the frontend does not need to care about the host.
package notch

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	// EventHover is emitted with a bool when the pointer enters or leaves the
	// window, including while the app is inactive.
	EventHover = "island:hover"

	// The window keeps a 12pt inset on each side for the frontend's inverted
	// top corners (--island-shoulder in styles.css), so both sizes are the
	// island body plus that room.
	DefaultCollapsedWidth  = 212
	DefaultCollapsedHeight = 46
	DefaultExpandedWidth   = 664
	DefaultExpandedHeight  = 560

	// notchSideWidth is the room kept on each side of the camera housing for
	// the idle pill's content. Keep it tight: every extra point covers the
	// menu bar on a notched display.
	notchSideWidth = 48

	// ToastHeight is the extra window room below the pill for an agent
	// notification bubble.
	ToastHeight = 96
)

// Options configures the island window.
type Options struct {
	// Name uniquely identifies the window inside Wails.
	Name string
	// Title is used by the OS window list and accessibility tools.
	Title string
	// URL is the asset route rendered by the webview.
	URL string

	// CollapsedWidth/Height describe the window size while the pill is idle.
	CollapsedWidth  int
	CollapsedHeight int
	// ExpandedWidth/Height describe the window size while the dashboard is open.
	ExpandedWidth  int
	ExpandedHeight int

	// TopGap is the distance from the top edge of the display (not the work
	// area), so the island docks over the menu bar by default.
	TopGap int
}

// Controller manages the island window and remembers its expansion state.
type Controller struct {
	app  *application.App
	win  *application.WebviewWindow
	opts Options

	mu       sync.Mutex
	expanded bool
	visible  bool
	toast    bool
}

// New creates the island window. The window starts hidden; call PlaceAndShow
// once the application has started so screen information is available.
func New(app *application.App, opts Options) *Controller {
	opts = normalise(opts)

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   opts.Name,
		Title:  opts.Title,
		Width:  opts.CollapsedWidth,
		Height: opts.CollapsedHeight,

		Frameless:     true,
		DisableResize: true,
		AlwaysOnTop:   true,
		Hidden:        true,

		InitialPosition:  application.WindowCentered,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		URL:              opts.URL,

		// Do not steal keyboard focus or appear in the alt-tab list.
		Mac: application.MacWindow{
			Backdrop:      application.MacBackdropTransparent,
			DisableShadow: true,
			CornerType:    application.MacWindowCornerTypeSquare,
			TitleBar:      application.MacTitleBarHidden,
			WindowClass:   application.MacWindowClassPanel,
			PanelPreferences: application.MacPanelPreferences{
				// NonActivating lets the panel take mouse input without
				// activating the app or stealing the menu bar.
				NonActivating: true,
				FloatingPanel: true,
				// BecomesKeyOnlyIfNeeded must stay off. With it on, a click on
				// a control (tab, card, button) leaves the panel non-key, and
				// AppKit discards that first click instead of delivering it.
			},
			WindowLevel: application.MacWindowLevelStatus,
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces |
				application.MacWindowCollectionBehaviorFullScreenAuxiliary |
				application.MacWindowCollectionBehaviorStationary |
				application.MacWindowCollectionBehaviorIgnoresCycle,
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar:                   true,
			DisableFramelessWindowDecorations: true,
			Theme:                             application.Dark,
		},
		Linux: application.LinuxWindow{
			WindowIsTranslucent: true,
		},
	})

	return &Controller{app: app, win: win, opts: opts}
}

func normalise(opts Options) Options {
	if opts.Name == "" {
		opts.Name = "island"
	}
	if opts.Title == "" {
		opts.Title = "Doter"
	}
	if opts.URL == "" {
		opts.URL = "/"
	}
	if opts.CollapsedWidth <= 0 {
		opts.CollapsedWidth = DefaultCollapsedWidth
		// The camera housing hides the centre of the docked window, so on a
		// notched display the idle pill has to be wide enough to show content
		// on both sides of it.
		if housing := cameraHousingWidth(); housing > 0 {
			if widened := housing + 2*notchSideWidth; widened > opts.CollapsedWidth {
				opts.CollapsedWidth = widened
			}
		}
	}
	if opts.CollapsedHeight <= 0 {
		opts.CollapsedHeight = DefaultCollapsedHeight
	}
	if opts.ExpandedWidth <= 0 {
		opts.ExpandedWidth = DefaultExpandedWidth
	}
	if opts.ExpandedHeight <= 0 {
		opts.ExpandedHeight = DefaultExpandedHeight
	}
	return opts
}

// Window exposes the underlying webview window for tray attachment or tests.
func (c *Controller) Window() *application.WebviewWindow { return c.win }

// PlaceAndShow positions the island at the top centre of the built-in display
// and makes it visible. The built-in display is preferred over the primary one
// so a multi-monitor setup still starts the island on the laptop screen. It is
// safe to call more than once (for example after a display configuration
// change).
func (c *Controller) PlaceAndShow() {
	c.mu.Lock()
	c.visible = true
	c.mu.Unlock()

	c.applyFrameOn(c.defaultScreen())
	watchHover(c.win.NativeWindow(), c.toastInset, func(entered bool) {
		c.app.Event.Emit(EventHover, entered)
	})
	c.win.Show()
}

// SetToastVisible reserves room below the pill for an agent notification
// bubble (or releases it). Only the idle window grows; the dashboard covers
// that space itself.
func (c *Controller) SetToastVisible(visible bool) {
	c.mu.Lock()
	if c.toast == visible {
		c.mu.Unlock()
		return
	}
	c.toast = visible
	c.mu.Unlock()

	c.applyFrame()
}

// toastInset is the bottom strip of the window that belongs to the toast, so
// the pointer poll does not treat hovering the bubble as hovering the pill.
func (c *Controller) toastInset() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.toast && !c.expanded {
		return ToastHeight
	}
	return 0
}

// SetExpanded resizes and repositions the native window. The frontend drives
// this before running its CSS transition so the expanded content is never
// clipped by the smaller idle window.
func (c *Controller) SetExpanded(expanded bool) {
	c.mu.Lock()
	if c.expanded == expanded {
		c.mu.Unlock()
		return
	}
	c.expanded = expanded
	c.mu.Unlock()

	c.applyFrame()
}

// SetGlassVisible shows or hides the native vibrancy layer. The frontend owns
// the timing: the layer is only revealed once the island has finished growing,
// and hidden the moment a collapse starts, so it never covers empty space.
func (c *Controller) SetGlassVisible(visible bool) {
	setNativeGlass(c.win.NativeWindow(), visible)
}

// ToggleExpanded flips the expansion state and reports the new value.
func (c *Controller) ToggleExpanded() bool {
	c.mu.Lock()
	c.expanded = !c.expanded
	expanded := c.expanded
	c.mu.Unlock()

	c.applyFrame()
	// The glass layer covers the whole island, so it can come back with the
	// window rather than waiting for the frontend.
	setNativeGlass(c.win.NativeWindow(), expanded)
	return expanded
}

// Expanded reports whether the dashboard is open.
func (c *Controller) Expanded() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.expanded
}

// Hide hides the island without destroying it.
func (c *Controller) Hide() {
	c.mu.Lock()
	c.visible = false
	c.mu.Unlock()
	c.win.Hide()
}

// Show reveals the island, keeping the remembered expansion state.
func (c *Controller) Show() {
	c.mu.Lock()
	c.visible = true
	c.mu.Unlock()
	c.win.Show()
}

// ToggleVisible hides a visible island and shows a hidden one.
func (c *Controller) ToggleVisible() {
	c.mu.Lock()
	visible := c.visible
	c.mu.Unlock()

	if visible {
		c.Hide()
		return
	}
	c.Show()
}

// Visible reports whether the island is currently shown.
func (c *Controller) Visible() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.visible
}

// Reposition re-centres the window, e.g. after the user changes displays.
func (c *Controller) Reposition() { c.applyFrame() }

// Focus activates the island so keyboard input (Escape) reaches the webview.
func (c *Controller) Focus() { c.win.Focus() }

// applyFrame reads state under the mutex, then talks to the native window
// without holding it: Wails window calls can synchronise with the UI thread,
// and holding the lock across that boundary risks a deadlock with tray-menu
// callbacks.
func (c *Controller) applyFrame() { c.applyFrameOn(c.currentScreen()) }

func (c *Controller) applyFrameOn(screen *application.Screen) {
	c.mu.Lock()
	expanded := c.expanded
	c.mu.Unlock()

	width, height := c.sizeFor(expanded)

	if screen == nil {
		c.win.SetSize(width, height)
		return
	}

	x := screen.Bounds.X + (screen.Bounds.Width-width)/2
	// Docked to the top edge of the display, over the menu bar.
	y := screen.Bounds.Y + c.opts.TopGap

	c.win.SetBounds(application.Rect{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	})
}

func (c *Controller) sizeFor(expanded bool) (int, int) {
	if expanded {
		return c.opts.ExpandedWidth, c.opts.ExpandedHeight
	}
	c.mu.Lock()
	toast := c.toast
	c.mu.Unlock()
	if toast {
		return c.opts.CollapsedWidth, c.opts.CollapsedHeight + ToastHeight
	}
	return c.opts.CollapsedWidth, c.opts.CollapsedHeight
}

// currentScreen prefers the display the island is already on so resizing does
// not teleport it across monitors.
func (c *Controller) currentScreen() *application.Screen {
	if screen, err := c.win.GetScreen(); err == nil && screen != nil {
		return screen
	}
	return c.defaultScreen()
}

// defaultScreen prefers the built-in display, so startup lands on the laptop
// screen even when an external display is primary. Wails has not loaded its
// screen list yet when the app starts, so macOS supplies the frame directly.
func (c *Controller) defaultScreen() *application.Screen {
	if bounds, ok := builtinScreenBounds(); ok {
		return &application.Screen{Bounds: bounds}
	}
	return c.app.Screen.GetPrimary()
}
