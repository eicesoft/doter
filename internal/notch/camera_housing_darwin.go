//go:build darwin && cgo

package notch

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static NSScreen* wailsBuiltinScreen(void) {
	for (NSScreen *screen in [NSScreen screens]) {
		NSNumber *number = [screen.deviceDescription objectForKey:@"NSScreenNumber"];
		if (number != nil && CGDisplayIsBuiltin([number unsignedIntValue])) {
			return screen;
		}
	}
	return nil;
}

static int cameraHousingWidth(void) {
	NSScreen *screen = wailsBuiltinScreen();
	if (screen == nil) {
		screen = [NSScreen mainScreen];
	}
	if (screen == nil) {
		return 0;
	}
	if (@available(macOS 12.0, *)) {
		if (screen.safeAreaInsets.top <= 0) {
			return 0;
		}
		NSRect left = screen.auxiliaryTopLeftArea;
		NSRect right = screen.auxiliaryTopRightArea;
		return (int)(screen.frame.size.width - left.size.width - right.size.width);
	}
	return 0;
}

// builtinScreenRect reports the built-in display in the same coordinate space
// as Wails window bounds: origin at the top-left of the primary display.
static int builtinScreenRect(int *x, int *y, int *width, int *height) {
	NSScreen *screen = wailsBuiltinScreen();
	NSArray<NSScreen*> *screens = [NSScreen screens];
	if (screen == nil || screens.count == 0) {
		return 0;
	}
	CGFloat primaryHeight = screens[0].frame.size.height;
	NSRect frame = screen.frame;
	*x = (int)frame.origin.x;
	*y = (int)(primaryHeight - (frame.origin.y + frame.size.height));
	*width = (int)frame.size.width;
	*height = (int)frame.size.height;
	return 1;
}
*/
import "C"

import "github.com/wailsapp/wails/v3/pkg/application"

// cameraHousingWidth reports the width of the camera housing (notch) in points
// on the built-in display, or 0 when that display has none.
func cameraHousingWidth() int {
	return int(C.cameraHousingWidth())
}

// builtinScreenBounds reports the built-in display in Wails window coordinates.
func builtinScreenBounds() (application.Rect, bool) {
	var x, y, width, height C.int
	if C.builtinScreenRect(&x, &y, &width, &height) == 0 {
		return application.Rect{}, false
	}
	return application.Rect{
		X:      int(x),
		Y:      int(y),
		Width:  int(width),
		Height: int(height),
	}, true
}
