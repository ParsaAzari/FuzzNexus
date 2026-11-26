//go:build !darwin

package dock

// Init is a no-op on platforms without a macOS-style dynamic dock icon.
func Init(_ func(isDark bool)) {}

// SetChangeHook is a no-op on platforms without a macOS-style dynamic dock icon.
func SetChangeHook(_ func(isDark bool)) {}

// IsDarkMode returns false as a stub on non-Darwin platforms.
func IsDarkMode() bool {
	return false
}

// SetDockTheme is a no-op on non-Darwin platforms.
func SetDockTheme(isDark bool) {
	mu.Lock()
	currentDark = isDark
	mu.Unlock()
}

// Cleanup is a no-op on non-Darwin platforms.
func Cleanup() {}

// HideTrafficLights is a no-op on non-Darwin platforms.
func HideTrafficLights() {}

// ShowTrafficLights is a no-op on non-Darwin platforms.
func ShowTrafficLights() {}

// CenterWindow is a no-op on non-Darwin platforms.
func CenterWindow() {}

// GeometricCenter is a no-op on non-Darwin platforms.
func GeometricCenter(width, height int) {}
