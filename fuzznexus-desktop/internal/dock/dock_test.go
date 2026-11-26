package dock

import (
	"testing"
)

func TestDockIconAssets(t *testing.T) {
	if len(IconDarkPNG) == 0 {
		t.Fatal("IconDarkPNG is empty")
	}
	if len(IconLightPNG) == 0 {
		t.Fatal("IconLightPNG is empty")
	}
	// Verify PNG signatures (first 8 bytes: 137, 80, 78, 71, 13, 10, 26, 10)
	pngSig := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	for i, b := range pngSig {
		if IconDarkPNG[i] != b {
			t.Errorf("IconDarkPNG invalid header at %d: got %d, want %d", i, IconDarkPNG[i], b)
		}
		if IconLightPNG[i] != b {
			t.Errorf("IconLightPNG invalid header at %d: got %d, want %d", i, IconLightPNG[i], b)
		}
	}
}

func TestDockSetTheme(t *testing.T) {
	var received bool
	SetChangeHook(func(isDark bool) {
		received = isDark
	})

	SetDockTheme(true)
	if !GetCurrentState() {
		t.Errorf("expected currentDark true, got false")
	}

	SetDockTheme(false)
	if GetCurrentState() {
		t.Errorf("expected currentDark false, got true")
	}

	_ = received
}
