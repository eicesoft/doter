//go:build darwin && cgo

package fullscreen

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework CoreGraphics
#include <AppKit/AppKit.h>
#include <CoreGraphics/CoreGraphics.h>
#include <math.h>

// frontmostIsFullscreen reports whether any normal window of the frontmost
// application exactly covers an active display. Fullscreen apps draw their
// window over the menu bar, so matching the display bounds avoids relying on
// private Space APIs.
static int frontmostIsFullscreen(void) {
	@autoreleasepool {
	NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
	if (app == nil) {
		return 0;
	}
	pid_t pid = [app processIdentifier];

	CFArrayRef windows = CGWindowListCopyWindowInfo(
		kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
		kCGNullWindowID);
	if (windows == NULL) {
		return 0;
	}

	CGDirectDisplayID displays[16];
	uint32_t displayCount = 0;
	if (CGGetActiveDisplayList(16, displays, &displayCount) != kCGErrorSuccess) {
		CFRelease(windows);
		return 0;
	}

	int found = 0;
	CFIndex count = CFArrayGetCount(windows);
	for (CFIndex i = 0; i < count && !found; i++) {
		CFDictionaryRef info = CFArrayGetValueAtIndex(windows, i);
		CFNumberRef owner = CFDictionaryGetValue(info, kCGWindowOwnerPID);
		CFNumberRef layer = CFDictionaryGetValue(info, kCGWindowLayer);
		CFDictionaryRef bounds = CFDictionaryGetValue(info, kCGWindowBounds);
		if (owner == NULL || layer == NULL || bounds == NULL) {
			continue;
		}

		int ownerPID = 0;
		int windowLayer = 0;
		CFNumberGetValue(owner, kCFNumberIntType, &ownerPID);
		CFNumberGetValue(layer, kCFNumberIntType, &windowLayer);
		if (ownerPID != pid || windowLayer != 0) {
			continue;
		}

		CGRect rect;
		if (!CGRectMakeWithDictionaryRepresentation(bounds, &rect)) {
			continue;
		}
		for (uint32_t d = 0; d < displayCount; d++) {
			CGRect screen = CGDisplayBounds(displays[d]);
			if (fabs(rect.origin.x - screen.origin.x) <= 2 &&
			    fabs(rect.origin.y - screen.origin.y) <= 2 &&
			    fabs(rect.size.width - screen.size.width) <= 2 &&
			    fabs(rect.size.height - screen.size.height) <= 2) {
				found = 1;
				break;
			}
		}
	}

	CFRelease(windows);
	return found;
	}
}
*/
import "C"

func active() bool {
	return C.frontmostIsFullscreen() != 0
}
