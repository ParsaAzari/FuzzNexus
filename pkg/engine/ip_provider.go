package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// IPProvider provides anonymized IP identification and change hints for network resilience.
type IPProvider interface {
	Name() string
	CurrentIPHash(ctx context.Context) (string, error)
	CurrentIP(ctx context.Context) (rawIP string, ipHash string, err error)
	CanChange() bool
	ChangeHint() string
}

// ManualVPNProvider prompts the user to change IP manually via VPN or proxy.
type ManualVPNProvider struct {
	ProbeEndpoint     string
	FallbackEndpoints []string
	HTTPClient        *http.Client
}

// NewManualVPNProvider creates a new ManualVPNProvider with default or custom probe endpoint.
func NewManualVPNProvider(probeEndpoint string) *ManualVPNProvider {
	fallbacks := []string{
		"https://checkip.amazonaws.com",
		"https://icanhazip.com",
		"https://api.ipify.org",
	}
	return &ManualVPNProvider{
		ProbeEndpoint:     probeEndpoint,
		FallbackEndpoints: fallbacks,
		HTTPClient: &http.Client{
			Timeout: 4 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

// Name returns the display name of the IP provider.
func (m *ManualVPNProvider) Name() string {
	return "Manual VPN"
}

// CanChange reports whether the user can switch IPs with this provider.
func (m *ManualVPNProvider) CanChange() bool {
	return true
}

// ChangeHint returns user-facing instructions for switching IP addresses.
func (m *ManualVPNProvider) ChangeHint() string {
	return "Please change your IP address via VPN or proxy, then click Test Connection."
}

// CurrentIP queries the probe endpoint and returns the raw IP alongside the 8-character SHA-256 hash.
func (m *ManualVPNProvider) CurrentIP(ctx context.Context) (string, string, error) {
	var endpoints []string
	if m.ProbeEndpoint != "" {
		endpoints = append(endpoints, m.ProbeEndpoint)
	}
	endpoints = append(endpoints, m.FallbackEndpoints...)

	var lastErr error
	for _, ep := range endpoints {
		if ep == "" {
			continue
		}
		rawIP, hash, err := m.queryEndpoint(ctx, ep)
		if err == nil && hash != "" {
			return rawIP, hash, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		// If probeEndpoint was a local target (127.0.0.1 or localhost), fallback gracefully to loopback identity
		if strings.Contains(m.ProbeEndpoint, "127.0.0.1") || strings.Contains(m.ProbeEndpoint, "localhost") {
			return "127.0.0.1", HashString8("127.0.0.1"), nil
		}
		return "", "", fmt.Errorf("all IP echo endpoints failed; last error: %w", lastErr)
	}
	return "", "", fmt.Errorf("no valid IP echo endpoints available")
}

// CurrentIPHash queries the probe endpoint and returns strictly the first 8 characters of SHA-256(IP).
func (m *ManualVPNProvider) CurrentIPHash(ctx context.Context) (string, error) {
	_, hash, err := m.CurrentIP(ctx)
	return hash, err
}

// HashString8 computes the 8-character SHA-256 hash of any input.
func HashString8(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])[:8]
}

func (m *ManualVPNProvider) queryEndpoint(ctx context.Context, endpoint string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("probe endpoint %s returned non-200 status: %d", endpoint, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", "", err
	}

	rawVal := strings.TrimSpace(string(bodyBytes))
	if rawVal == "" {
		return "", "", fmt.Errorf("probe endpoint %s returned empty body", endpoint)
	}

	// Reject HTML responses to avoid hashing web server pages (e.g. from misconfigured targets)
	if strings.Contains(rawVal, "<html") || strings.Contains(rawVal, "<!DOCTYPE") || strings.Contains(rawVal, "</") {
		return "", "", fmt.Errorf("probe endpoint %s returned HTML instead of an IP address", endpoint)
	}

	// Calculate SHA-256 and truncate to exactly 8 characters
	h := sha256.Sum256([]byte(rawVal))
	hashHex := hex.EncodeToString(h[:])
	return rawVal, hashHex[:8], nil
}
