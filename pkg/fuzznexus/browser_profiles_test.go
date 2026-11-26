package fuzznexus

import (
	"net/http"
	"strings"
	"testing"
)

func TestBrowserProfiles_Consistency(t *testing.T) {
	profiles := DefaultBrowserProfiles()
	if len(profiles) < 5 {
		t.Fatalf("expected at least 5 default profiles, got %d", len(profiles))
	}

	for _, p := range profiles {
		if p.ID == "" || p.Name == "" || p.UserAgent == "" {
			t.Errorf("profile %+v has empty critical fields", p)
		}
		if _, ok := p.Headers["Accept"]; !ok {
			t.Errorf("profile %s missing essential Accept header", p.ID)
		}
		if _, ok := p.Headers["Accept-Language"]; !ok {
			t.Errorf("profile %s missing essential Accept-Language header", p.ID)
		}

		// Consistency rules:
		// 1. Safari must NOT have Sec-Ch-Ua
		if strings.Contains(p.ID, "safari") {
			if _, hasSec := p.Headers["Sec-Ch-Ua"]; hasSec {
				t.Errorf("safari profile should not have Sec-Ch-Ua header")
			}
		}

		// 2. Chrome must have Sec-Ch-Ua matching platform
		if strings.Contains(p.ID, "chrome") {
			secPlatform, ok := p.Headers["Sec-Ch-Ua-Platform"]
			if !ok {
				t.Errorf("chrome profile %s missing Sec-Ch-Ua-Platform", p.ID)
			}
			if p.Platform == "macOS" && !strings.Contains(secPlatform, "macOS") {
				t.Errorf("chrome macOS platform mismatch: %s", secPlatform)
			}
			if p.Platform == "Windows" && !strings.Contains(secPlatform, "Windows") {
				t.Errorf("chrome Windows platform mismatch: %s", secPlatform)
			}
		}
	}
}

func TestBrowserProfiles_ApplyToRequest(t *testing.T) {
	profile := FindBrowserProfile("chrome-macos")

	// Case 1: Default application without custom overrides
	req, _ := http.NewRequest("GET", "https://example.com/", nil)
	profile.ApplyToRequest(req, nil)

	if req.Header.Get("User-Agent") != profile.UserAgent {
		t.Errorf("expected UA %s, got %s", profile.UserAgent, req.Header.Get("User-Agent"))
	}
	if req.Header.Get("Accept") != profile.Headers["Accept"] {
		t.Errorf("expected Accept %s, got %s", profile.Headers["Accept"], req.Header.Get("Accept"))
	}
	if req.Header.Get("Sec-Ch-Ua-Platform") != profile.Headers["Sec-Ch-Ua-Platform"] {
		t.Errorf("expected Sec-Ch-Ua-Platform %s, got %s", profile.Headers["Sec-Ch-Ua-Platform"], req.Header.Get("Sec-Ch-Ua-Platform"))
	}

	// Case 2: Custom user headers override defaults
	req2, _ := http.NewRequest("GET", "https://example.com/", nil)
	custom := map[string]string{
		"User-Agent": "Custom-Scanner/1.0",
		"Accept":     "application/json",
	}
	profile.ApplyToRequest(req2, custom)

	if req2.Header.Get("User-Agent") != "Custom-Scanner/1.0" {
		t.Errorf("custom UA was overwritten")
	}
	if req2.Header.Get("Accept") != "application/json" {
		t.Errorf("custom Accept was overwritten")
	}
	// Non-overridden headers should still be present
	if req2.Header.Get("Sec-Ch-Ua-Platform") != profile.Headers["Sec-Ch-Ua-Platform"] {
		t.Errorf("expected non-overridden Sec-Ch-Ua-Platform to be preserved")
	}
}

func TestProfileRotator_Cycling(t *testing.T) {
	rotator := NewProfileRotator("adaptive", "chrome-macos")
	first := rotator.Current()
	if first.ID != "chrome-macos" {
		t.Fatalf("expected initial profile chrome-macos, got %s", first.ID)
	}

	second := rotator.Rotate()
	if second.ID == first.ID {
		t.Fatalf("expected rotated profile to be different from first")
	}

	third := rotator.Rotate()
	if third.ID == second.ID {
		t.Fatalf("expected rotated profile to be different from second")
	}
}
