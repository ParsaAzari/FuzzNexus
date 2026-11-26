package fuzznexus

import (
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// BrowserProfile represents a realistic, internally coherent browser persona.
type BrowserProfile struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Platform  string            `json:"platform"`
	UserAgent string            `json:"user_agent"`
	Headers   map[string]string `json:"headers"`
}

// DefaultBrowserProfiles returns the curated set of authentic modern browser personas.
func DefaultBrowserProfiles() []BrowserProfile {
	return []BrowserProfile{
		{
			ID:        "chrome-macos",
			Name:      "Chrome (macOS)",
			Platform:  "macOS",
			UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			Headers: map[string]string{
				"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
				"Accept-Language":           "en-US,en;q=0.9",
				"Sec-Ch-Ua":                 `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
				"Sec-Ch-Ua-Mobile":          "?0",
				"Sec-Ch-Ua-Platform":        `"macOS"`,
				"Sec-Fetch-Dest":            "document",
				"Sec-Fetch-Mode":            "navigate",
				"Sec-Fetch-Site":            "none",
				"Sec-Fetch-User":            "?1",
				"Upgrade-Insecure-Requests": "1",
			},
		},
		{
			ID:        "chrome-windows",
			Name:      "Chrome (Windows)",
			Platform:  "Windows",
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			Headers: map[string]string{
				"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
				"Accept-Language":           "en-US,en;q=0.9",
				"Sec-Ch-Ua":                 `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
				"Sec-Ch-Ua-Mobile":          "?0",
				"Sec-Ch-Ua-Platform":        `"Windows"`,
				"Sec-Fetch-Dest":            "document",
				"Sec-Fetch-Mode":            "navigate",
				"Sec-Fetch-Site":            "none",
				"Sec-Fetch-User":            "?1",
				"Upgrade-Insecure-Requests": "1",
			},
		},
		{
			ID:        "safari-macos",
			Name:      "Safari (macOS)",
			Platform:  "macOS",
			UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Safari/605.1.15",
			Headers: map[string]string{
				"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
				"Accept-Language": "en-US,en;q=0.9",
				"Sec-Fetch-Dest":  "document",
				"Sec-Fetch-Mode":  "navigate",
				"Sec-Fetch-Site":  "none",
			},
		},
		{
			ID:        "firefox-windows",
			Name:      "Firefox (Windows)",
			Platform:  "Windows",
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
			Headers: map[string]string{
				"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
				"Accept-Language":           "en-US,en;q=0.5",
				"Sec-Fetch-Dest":            "document",
				"Sec-Fetch-Mode":            "navigate",
				"Sec-Fetch-Site":            "none",
				"Sec-Fetch-User":            "?1",
				"Upgrade-Insecure-Requests": "1",
			},
		},
		{
			ID:        "edge-windows",
			Name:      "Edge (Windows)",
			Platform:  "Windows",
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
			Headers: map[string]string{
				"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
				"Accept-Language":           "en-US,en;q=0.9",
				"Sec-Ch-Ua":                 `"Microsoft Edge";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
				"Sec-Ch-Ua-Mobile":          "?0",
				"Sec-Ch-Ua-Platform":        `"Windows"`,
				"Sec-Fetch-Dest":            "document",
				"Sec-Fetch-Mode":            "navigate",
				"Sec-Fetch-Site":            "none",
				"Sec-Fetch-User":            "?1",
				"Upgrade-Insecure-Requests": "1",
			},
		},
	}
}

// FindBrowserProfile looks up a profile by ID or common alias, falling back to Chrome macOS.
func FindBrowserProfile(idOrName string) BrowserProfile {
	profiles := DefaultBrowserProfiles()
	norm := strings.ToLower(strings.TrimSpace(idOrName))
	if norm == "" || norm == "auto" || norm == "adaptive" || norm == "default" {
		return profiles[0]
	}
	for _, p := range profiles {
		if strings.ToLower(p.ID) == norm || strings.ToLower(p.Name) == norm {
			return p
		}
	}
	// Fallback alias matching
	switch {
	case strings.Contains(norm, "safari"):
		return profiles[2]
	case strings.Contains(norm, "firefox"):
		return profiles[3]
	case strings.Contains(norm, "edge"):
		return profiles[4]
	case strings.Contains(norm, "win"):
		return profiles[1]
	default:
		return profiles[0]
	}
}

// ProfileRotator manages stateful or randomized browser persona cycling.
type ProfileRotator struct {
	mu           sync.RWMutex
	profiles     []BrowserProfile
	currentIndex int
	mode         string
	rnd          *rand.Rand
}

// NewProfileRotator initializes a thread-safe rotator.
func NewProfileRotator(mode, initialProfile string) *ProfileRotator {
	profiles := DefaultBrowserProfiles()
	initialIdx := 0
	norm := strings.ToLower(strings.TrimSpace(initialProfile))
	for i, p := range profiles {
		if strings.ToLower(p.ID) == norm {
			initialIdx = i
			break
		}
	}

	normMode := strings.ToLower(strings.TrimSpace(mode))
	if normMode == "" {
		normMode = "adaptive"
	}

	return &ProfileRotator{
		profiles:     profiles,
		currentIndex: initialIdx,
		mode:         normMode,
		rnd:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Current returns the currently active browser persona.
func (r *ProfileRotator) Current() BrowserProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.profiles) == 0 {
		return DefaultBrowserProfiles()[0]
	}
	return r.profiles[r.currentIndex]
}

// Rotate switches to the next persona atomically and returns it.
func (r *ProfileRotator) Rotate() BrowserProfile {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.profiles) <= 1 {
		return r.profiles[0]
	}
	if r.mode == "random" {
		next := r.rnd.Intn(len(r.profiles))
		if next == r.currentIndex {
			next = (r.currentIndex + 1) % len(r.profiles)
		}
		r.currentIndex = next
	} else {
		r.currentIndex = (r.currentIndex + 1) % len(r.profiles)
	}
	return r.profiles[r.currentIndex]
}

// ApplyToHTTPRequest sets profile headers on an http.Request without overwriting explicit custom headers.
func (r *ProfileRotator) ApplyToHTTPRequest(req *http.Request, customHeaders map[string]string) {
	profile := r.Current()
	profile.ApplyToRequest(req, customHeaders)
}

// ApplyToHeadersMap merges profile headers into a headers map, preserving user-defined headers.
func (p BrowserProfile) ApplyToHeadersMap(headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	hasHeaderCI := func(target string) bool {
		for k := range headers {
			if strings.EqualFold(k, target) {
				return true
			}
		}
		return false
	}
	// 1. Set User-Agent if not explicitly customized (case-insensitive)
	if !hasHeaderCI("User-Agent") {
		headers["User-Agent"] = p.UserAgent
	}
	// 2. Set all profile-specific headers without overwriting user overrides
	for k, v := range p.Headers {
		if !hasHeaderCI(k) {
			headers[k] = v
		}
	}
	return headers
}

// ApplyToRequest applies the profile to an http.Request, respecting user custom headers.
func (p BrowserProfile) ApplyToRequest(req *http.Request, customHeaders map[string]string) {
	if req.Header == nil {
		req.Header = make(http.Header)
	}

	// 1. User-Agent
	if customUA, ok := customHeaders["User-Agent"]; ok && customUA != "" {
		req.Header.Set("User-Agent", customUA)
	} else if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", p.UserAgent)
	}

	// 2. Persona headers (Accept, Accept-Language, Sec-*, etc.)
	for k, v := range p.Headers {
		if customVal, ok := customHeaders[k]; ok && customVal != "" {
			req.Header.Set(k, customVal)
		} else if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}
}
