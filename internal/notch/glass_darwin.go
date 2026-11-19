//go:build darwin && cgo

package notch

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore
#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>

static NSVisualEffectView *islandGlass = nil;

// islandGlassUpdate places a behind-window vibrancy view under the webview,
// sized to the island rather than the whole window: 20pt narrower and 14pt
// shorter than the expanded window, flush to its top. It is deferred to the
// main queue so it picks up the window frame Wails has just set.
//
// The frontend decides when to reveal it (after the island has grown) and
// hides it the moment a collapse starts, so the layer never covers area the
// webview's island is not occupying.
static void islandGlassUpdate(void *nsWindow, int visible) {
	if (nsWindow == nil) {
		return;
	}
	NSWindow *window = (NSWindow *)nsWindow;

	void (^update)(void) = ^{
		NSView *content = window.contentView;
		if (content == nil) {
			return;
		}

		CGFloat W = content.bounds.size.width;
		CGFloat H = content.bounds.size.height;
		NSRect target = NSMakeRect(10, 14, W - 20, H - 14);

		if (islandGlass == nil) {
			islandGlass = [[NSVisualEffectView alloc] initWithFrame:target];
			islandGlass.blendingMode = NSVisualEffectBlendingModeBehindWindow;
			islandGlass.material = NSVisualEffectMaterialUnderWindowBackground;
			islandGlass.state = NSVisualEffectStateActive;
			islandGlass.appearance = [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua];
			islandGlass.wantsLayer = YES;
			// The bottom corners match the island; the top edge stays square
			// because it sits against the screen edge.
			islandGlass.layer.cornerRadius = 16;
			islandGlass.layer.maskedCorners = kCALayerMinXMinYCorner | kCALayerMaxXMinYCorner;
			islandGlass.layer.masksToBounds = YES;
			islandGlass.alphaValue = 0;
			islandGlass.hidden = YES;
			[content addSubview:islandGlass positioned:NSWindowBelow relativeTo:nil];
		}

		if (!visible) {
			[islandGlass.layer removeAllAnimations];
			islandGlass.hidden = YES;
			islandGlass.alphaValue = 0;
			return;
		}

		islandGlass.frame = target;
		if (islandGlass.hidden) {
			islandGlass.alphaValue = 0;
		}
		islandGlass.hidden = NO;

		[NSAnimationContext beginGrouping];
		[[NSAnimationContext currentContext] setDuration:0.2];
		islandGlass.animator.alphaValue = 1.0;
		[NSAnimationContext endGrouping];
	};

	dispatch_async(dispatch_get_main_queue(), update);
}
*/
import "C"

import "unsafe"

// setNativeGlass shows or hides the native vibrancy layer behind the webview.
func setNativeGlass(nsWindow unsafe.Pointer, visible bool) {
	if nsWindow == nil {
		return
	}
	v := 0
	if visible {
		v = 1
	}
	C.islandGlassUpdate(nsWindow, C.int(v))
}
