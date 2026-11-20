//go:build darwin && cgo

package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices
#include <AppKit/AppKit.h>
#include <ApplicationServices/ApplicationServices.h>
#include <unistd.h>

static pid_t doterPasteTarget = 0;

// rememberTargetApp keeps the last external app that owned the screen, so
// auto-paste can return focus to it after the island has been opened.
static void rememberTargetApp(void) {
	@autoreleasepool {
		NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
		if (app == nil || [app processIdentifier] == getpid()) {
			return;
		}
		doterPasteTarget = [app processIdentifier];
	}
}

// pasteToTargetApp returns 0 on success, 1 when accessibility is not granted,
// 2 when no target was recorded, 3 when the target has quit, and 4 when the
// synthetic keyboard event could not be created.
static int pasteToTargetApp(void) {
	if (!AXIsProcessTrusted()) {
		return 1;
	}
	if (doterPasteTarget <= 0) {
		return 2;
	}

	@autoreleasepool {
		NSRunningApplication *app = [NSRunningApplication
			runningApplicationWithProcessIdentifier:doterPasteTarget];
		if (app == nil || [app isTerminated]) {
			return 3;
		}

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
		[app activateWithOptions:NSApplicationActivateIgnoringOtherApps];
#pragma clang diagnostic pop
		usleep(120000);

		CGEventRef down = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)0x09, true);
		CGEventRef up = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)0x09, false);
		if (down == NULL || up == NULL) {
			if (down != NULL) {
				CFRelease(down);
			}
			if (up != NULL) {
				CFRelease(up);
			}
			return 4;
		}
		CGEventSetFlags(down, kCGEventFlagMaskCommand);
		CGEventSetFlags(up, kCGEventFlagMaskCommand);
		CGEventPost(kCGHIDEventTap, down);
		CGEventPost(kCGHIDEventTap, up);
		CFRelease(down);
		CFRelease(up);
	}
	return 0;
}
*/
import "C"

import "errors"

func rememberTarget() {
	C.rememberTargetApp()
}

func pasteTarget() error {
	switch C.pasteToTargetApp() {
	case 0:
		return nil
	case 1:
		return errors.New("自动粘贴需要辅助功能权限")
	case 2:
		return errors.New("没有可粘贴的目标应用")
	case 3:
		return errors.New("目标应用已退出")
	default:
		return errors.New("无法发送粘贴快捷键")
	}
}
