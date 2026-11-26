//go:build darwin

package dock

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

extern void onThemeChangedCallback(int isDark);

static id gObserverToken = nil;

static void StartThemeObserver(void) {
    if (gObserverToken == nil) {
        gObserverToken = [[NSDistributedNotificationCenter defaultCenter]
            addObserverForName:@"AppleInterfaceThemeChangedNotification"
                        object:nil
                         queue:[NSOperationQueue mainQueue]
                    usingBlock:^(NSNotification *note) {
                        NSString *style = [[NSUserDefaults standardUserDefaults] stringForKey:@"AppleInterfaceStyle"];
                        int isDark = (style != nil && [style caseInsensitiveCompare:@"Dark"] == NSOrderedSame) ? 1 : 0;
                        onThemeChangedCallback(isDark);
                    }];
    }
}

static void StopThemeObserver(void) {
    if (gObserverToken != nil) {
        [[NSDistributedNotificationCenter defaultCenter] removeObserver:gObserverToken];
        gObserverToken = nil;
    }
}

static int NativeIsDarkMode(void) {
    @autoreleasepool {
        NSString *style = [[NSUserDefaults standardUserDefaults] stringForKey:@"AppleInterfaceStyle"];
        return (style != nil && [style caseInsensitiveCompare:@"Dark"] == NSOrderedSame) ? 1 : 0;
    }
}

static void NativeSetDockIcon(const void* bytes, int length) {
    if (bytes == NULL || length <= 0) return;
    @autoreleasepool {
        NSData* data = [NSData dataWithBytes:bytes length:length];
        NSImage* image = [[NSImage alloc] initWithData:data];
        if (image != nil) {
            dispatch_async(dispatch_get_main_queue(), ^{
                [NSApp setApplicationIconImage:image];
            });
        }
    }
}

static void NativeRestoreDockIcon(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setApplicationIconImage:nil];
    });
}

static void NativeHideTrafficLights(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        for (NSWindow *window in [NSApp windows]) {
            [[window standardWindowButton:NSWindowCloseButton] setHidden:YES];
            [[window standardWindowButton:NSWindowMiniaturizeButton] setHidden:YES];
            [[window standardWindowButton:NSWindowZoomButton] setHidden:YES];
        }
    });
}

static void NativeShowTrafficLights(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        for (NSWindow *window in [NSApp windows]) {
            [[window standardWindowButton:NSWindowCloseButton] setHidden:NO];
            [[window standardWindowButton:NSWindowMiniaturizeButton] setHidden:NO];
            [[window standardWindowButton:NSWindowZoomButton] setHidden:NO];
        }
    });
}

static void NativeGeometricCenter(int width, int height) {
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            NSPoint mouseLoc = [NSEvent mouseLocation];
            NSScreen *targetScreen = nil;
            for (NSScreen *screen in [NSScreen screens]) {
                if (NSPointInRect(mouseLoc, [screen frame])) {
                    targetScreen = screen;
                    break;
                }
            }
            if (targetScreen == nil) {
                targetScreen = [NSScreen mainScreen];
            }
            if (targetScreen == nil && [[NSScreen screens] count] > 0) {
                targetScreen = [[NSScreen screens] firstObject];
            }
            if (targetScreen == nil) return;

            NSRect visible = [targetScreen visibleFrame];
            CGFloat x = visible.origin.x + (visible.size.width - (CGFloat)width) / 2.0;
            CGFloat y = visible.origin.y + (visible.size.height - (CGFloat)height) / 2.0;

            for (NSWindow *window in [NSApp windows]) {
                NSRect newFrame = NSMakeRect(round(x), round(y), (CGFloat)width, (CGFloat)height);
                [window setFrame:newFrame display:YES animate:NO];
            }
        }
    });
}
*/
import "C"
import "unsafe"

//export onThemeChangedCallback
func onThemeChangedCallback(isDark C.int) {
	dark := isDark != 0
	SetDockTheme(dark)
	hook := changeHook()
	if hook != nil {
		hook(dark)
	}
}

// Init initializes the macOS dynamic dock icon observer and sets the initial icon.
func Init(onThemeChange func(isDark bool)) {
	if onThemeChange != nil {
		SetChangeHook(onThemeChange)
	}

	C.StartThemeObserver()
	initialDark := IsDarkMode()
	SetDockTheme(initialDark)
}

// IsDarkMode returns whether macOS is currently configured in dark appearance.
func IsDarkMode() bool {
	return C.NativeIsDarkMode() != 0
}

// SetDockTheme sets the macOS dock icon to either dark or light appearance.
func SetDockTheme(isDark bool) {
	mu.Lock()
	currentDark = isDark
	mu.Unlock()

	var data []byte
	if isDark {
		data = IconDarkPNG
	} else {
		data = IconLightPNG
	}

	if len(data) > 0 {
		C.NativeSetDockIcon(unsafe.Pointer(&data[0]), C.int(len(data)))
	}
}

// Cleanup removes observers and restores the default dock icon.
func Cleanup() {
	C.StopThemeObserver()
	C.NativeRestoreDockIcon()
}

// HideTrafficLights hides the macOS window buttons (close, minimize, zoom) during splash.
func HideTrafficLights() {
	C.NativeHideTrafficLights()
}

// ShowTrafficLights restores the macOS window buttons when the main app is ready.
func ShowTrafficLights() {
	C.NativeShowTrafficLights()
}

// GeometricCenter forces application windows to the exact geometric center of the active monitor's visible frame.
func GeometricCenter(width, height int) {
	C.NativeGeometricCenter(C.int(width), C.int(height))
}

// CenterWindow is maintained for backwards compatibility.
func CenterWindow() {
	GeometricCenter(640, 340)
}
