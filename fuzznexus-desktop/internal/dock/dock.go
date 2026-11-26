package dock

import (
	_ "embed"
	"sync"
)

//go:embed assets/appicon-dark.png
var IconDarkPNG []byte

//go:embed assets/appicon-light.png
var IconLightPNG []byte

var (
	mu           sync.Mutex
	onChangeHook func(isDark bool)
	currentDark  bool
)

// SetChangeHook registers a callback invoked when OS appearance changes.
func SetChangeHook(fn func(isDark bool)) {
	mu.Lock()
	defer mu.Unlock()
	onChangeHook = fn
}

// GetCurrentState returns the last recorded dark state.
func GetCurrentState() bool {
	mu.Lock()
	defer mu.Unlock()
	return currentDark
}
