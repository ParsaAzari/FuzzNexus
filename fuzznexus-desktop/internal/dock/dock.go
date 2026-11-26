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
	mu          sync.Mutex
	currentDark bool
)

// GetCurrentState returns the last recorded dark state.
func GetCurrentState() bool {
	mu.Lock()
	defer mu.Unlock()
	return currentDark
}
