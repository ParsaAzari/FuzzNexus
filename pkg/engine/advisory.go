package engine

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
	utls "github.com/refraction-networking/utls"
)

// AdvisoryVerdict represents the definitive classification of elevated block responses.
type AdvisoryVerdict string

const (
	VerdictNone            AdvisoryVerdict = "NONE"
	VerdictIPBan           AdvisoryVerdict = "IP_BAN"
	VerdictWordlistTrigger AdvisoryVerdict = "WORDLIST_WAF_TRIGGER"
	VerdictRateLimit       AdvisoryVerdict = "RATE_LIMIT"
	VerdictEndpointRule    AdvisoryVerdict = "ENDPOINT_PROTECTED"
)

// CanaryEvidence records live probe results used to distinguish IP bans from wordlist WAF rules.
type CanaryEvidence struct {
	RootURL          string `json:"root_url"`
	RootStatus       int    `json:"root_status"`
	RootWAFHeader    string `json:"root_waf_header,omitempty"`
	RootError        string `json:"root_error,omitempty"`
	BenignPathURL    string `json:"benign_path_url,omitempty"`
	BenignPathStatus int    `json:"benign_path_status,omitempty"`
	BenignWAFHeader  string `json:"benign_waf_header,omitempty"`
	BenignError      string `json:"benign_error,omitempty"`
	CheckedAt        string `json:"checked_at"`
}

// DiagnosticVerdict is the structured outcome emitted to session control & desktop UI.
type DiagnosticVerdict struct {
	Verdict          AdvisoryVerdict `json:"verdict"`
	Confidence       int             `json:"confidence"` // 0-100
	Title            string          `json:"title"`
	Summary          string          `json:"summary"`
	EvidenceHeader   string          `json:"evidence_header,omitempty"`
	FixedBodyLength  int64           `json:"fixed_body_length,omitempty"`
	SamplePayloads   []string        `json:"sample_payloads,omitempty"`
	Canary           CanaryEvidence  `json:"canary"`
	SuggestedActions []string        `json:"suggested_actions,omitempty"`
}

// CanaryProbeConfig configures authentic browser/TLS-aligned canary probes.
type CanaryProbeConfig struct {
	RootCanaryURL     string
	BenignPathURL     string
	IsLocalTarget     bool
	AllowNon200Origin bool
	Headers           map[string]string
	Cookies           string
	BrowserProfile    string
	TLSProfile        string
	RateLimitFunc     func()
	Timeout           time.Duration
}

// ProbeOutcome holds the sanitized evaluation of a single health or canary HTTP probe.
type ProbeOutcome struct {
	URL         string
	StatusCode  int
	WAFHeader   string
	SoftBlocked bool
	IsBlocked   bool
	Healthy     bool
	Error       error
}

// AppendCacheBuster appends a unique query parameter to bypass edge CDN caches.
func AppendCacheBuster(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := parsed.Query()
	q.Set("_fn_canary", fmt.Sprintf("%d", time.Now().UnixNano()))
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

// RandomCanarySlug generates a benign lowercase alphanumeric slug unlikely to hit real routes or traversal rules.
func RandomCanarySlug() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("fn_canary_%d", time.Now().UnixNano()%100000)
	}
	return "fn_canary_" + hex.EncodeToString(buf)
}

func newCanaryHTTPClient(cfg CanaryProbeConfig) *http.Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	tr := &http.Transport{
		DisableKeepAlives: true,
		DialContext: (&net.Dialer{
			Timeout: timeout,
		}).DialContext,
		TLSHandshakeTimeout: timeout,
	}

	if cfg.TLSProfile != "" && cfg.TLSProfile != "default" {
		var helloID utls.ClientHelloID
		switch strings.ToLower(cfg.TLSProfile) {
		case "chrome", "chrome120", "chrome-120":
			helloID = utls.HelloChrome_120
		case "firefox", "firefox120", "firefox121", "firefox-120":
			helloID = utls.HelloFirefox_120
		default:
			helloID = utls.HelloGolang
		}
		dialer := &net.Dialer{Timeout: timeout}
		tr.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			rawConn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			host, _, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				host = addr
			}
			uConn := utls.UClient(rawConn, &utls.Config{
				ServerName:         host,
				InsecureSkipVerify: true,
			}, helloID)
			if hsErr := uConn.HandshakeContext(ctx); hsErr != nil {
				_ = rawConn.Close()
				return nil, hsErr
			}
			return uConn, nil
		}
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ProbeHTTPTarget executes a single browser-coherent HTTP GET probe and inspects status, WAF headers, and soft-blocks.
func ProbeHTTPTarget(ctx context.Context, rawURL string, cfg CanaryProbeConfig, cacheBust bool) ProbeOutcome {
	client := newCanaryHTTPClient(cfg)
	defer client.CloseIdleConnections()
	return probeHTTPTargetWithClient(ctx, client, rawURL, cfg, cacheBust)
}

func probeHTTPTargetWithClient(ctx context.Context, client *http.Client, rawURL string, cfg CanaryProbeConfig, cacheBust bool) ProbeOutcome {
	out := ProbeOutcome{URL: rawURL}
	if rawURL == "" {
		out.Error = fmt.Errorf("empty probe URL")
		return out
	}
	targetURL := rawURL
	if cacheBust && !cfg.IsLocalTarget {
		targetURL = AppendCacheBuster(rawURL)
	}
	out.URL = targetURL

	if cfg.RateLimitFunc != nil {
		cfg.RateLimitFunc()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		out.Error = err
		return out
	}

	profile := fuzznexus.FindBrowserProfile(cfg.BrowserProfile)
	profile.ApplyToRequest(req, cfg.Headers)
	if cfg.Cookies != "" && req.Header.Get("Cookie") == "" {
		req.Header.Set("Cookie", cfg.Cookies)
	}

	resp, err := client.Do(req)
	if err != nil {
		out.Error = err
		out.IsBlocked = true
		return out
	}
	defer resp.Body.Close()

	out.StatusCode = resp.StatusCode
	out.WAFHeader = ExtractWAFSignal(resp.StatusCode, resp.Header)

	var bodyReader io.Reader = io.LimitReader(resp.Body, 64*1024)
	switch strings.ToLower(resp.Header.Get("Content-Encoding")) {
	case "gzip":
		if zr, zerr := gzip.NewReader(bodyReader); zerr == nil {
			defer zr.Close()
			bodyReader = io.LimitReader(zr, 64*1024)
		}
	case "br":
		bodyReader = io.LimitReader(brotli.NewReader(bodyReader), 64*1024)
	case "deflate":
		zr := flate.NewReader(bodyReader)
		defer zr.Close()
		bodyReader = io.LimitReader(zr, 64*1024)
	}
	bodyBytes, _ := io.ReadAll(bodyReader)

	sbd := NewSoftBlockDetector(true)
	if isSoft, reason := sbd.Check(resp.StatusCode, resp.Header, bodyBytes); isSoft {
		out.SoftBlocked = true
		if out.WAFHeader == "" {
			out.WAFHeader = reason
		}
	}

	if out.SoftBlocked || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		out.IsBlocked = true
		out.Healthy = false
		return out
	}

	// Redirects to challenge pages indicate block rather than clean origin reachability
	if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
		loc := strings.ToLower(resp.Header.Get("Location"))
		if strings.Contains(loc, "challenge") || strings.Contains(loc, "captcha") || strings.Contains(loc, "__cf_chl") {
			out.IsBlocked = true
			out.Healthy = false
			if out.WAFHeader == "" {
				out.WAFHeader = "redirect-challenge"
			}
			return out
		}
	}

	// Clean 2xx/3xx/404 (on non-local origins or benign path probes) indicate origin is reachable & unbanned
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		out.Healthy = true
		return out
	}
	if cfg.AllowNon200Origin && resp.StatusCode == http.StatusNotFound && out.WAFHeader == "" {
		out.Healthy = true
		return out
	}

	return out
}

// EvaluateBlockCause runs dual differential canary probing and combines it with rolling
// forensics to definitively distinguish IP/client bans from payload-specific wordlist triggers.
func EvaluateBlockCause(
	ctx context.Context,
	stats RollingStats,
	forensics RollingForensics,
	probeCfg CanaryProbeConfig,
) DiagnosticVerdict {
	nowStr := time.Now().UTC().Format(time.RFC3339)
	evidence := CanaryEvidence{
		RootURL:   probeCfg.RootCanaryURL,
		CheckedAt: nowStr,
	}

	// Reuse a single transport across Probe 1 and Probe 2 to halve TLS overhead,
	// closing all idle connections before returning.
	sharedCfg := probeCfg
	sharedClient := newCanaryHTTPClient(sharedCfg)
	if tr, ok := sharedClient.Transport.(*http.Transport); ok {
		tr.DisableKeepAlives = false
	}
	defer sharedClient.CloseIdleConnections()

	// Step 1: Probe Root/Origin (with cache-busting on remote targets)
	rootOutcome := probeHTTPTargetWithClient(ctx, sharedClient, probeCfg.RootCanaryURL, probeCfg, true)
	evidence.RootURL = rootOutcome.URL
	evidence.RootStatus = rootOutcome.StatusCode
	evidence.RootWAFHeader = rootOutcome.WAFHeader
	if rootOutcome.Error != nil {
		evidence.RootError = rootOutcome.Error.Error()
	}

	// If root origin itself returns 403/429/SoftBlock or drops connections, IP/client is blocked.
	if rootOutcome.Error != nil || rootOutcome.IsBlocked {
		verdictType := VerdictIPBan
		title := "Target IP / Client Block Confirmed"
		summary := fmt.Sprintf("Origin canary probe to %s failed (%d %s). The target firewall appears to be rejecting traffic from your current IP or route.",
			rootOutcome.URL, rootOutcome.StatusCode, rootOutcome.WAFHeader)
		if rootOutcome.StatusCode == http.StatusTooManyRequests || (stats.Count429 >= 3 && stats.Count403 == 0) {
			verdictType = VerdictRateLimit
			title = "Target Rate Limit Enforced (429)"
		}
		return DiagnosticVerdict{
			Verdict:          verdictType,
			Confidence:       98,
			Title:            title,
			Summary:          summary,
			EvidenceHeader:   coalesceString(rootOutcome.WAFHeader, forensics.DominantWAFHeader),
			FixedBodyLength:  forensics.DominantBodyLength,
			SamplePayloads:   forensics.SamplePayloads,
			Canary:           evidence,
			SuggestedActions: []string{"rotate_ip", "reduce_rate"},
		}
	}

	// Step 2: Probe Benign Random Path at FUZZ route (when safe, e.g. path/query fuzzing)
	benignHealthy := true
	if probeCfg.BenignPathURL != "" && probeCfg.BenignPathURL != probeCfg.RootCanaryURL {
		benignCfg := probeCfg
		benignCfg.AllowNon200Origin = true // Random non-existent slug should return clean 404 or 200
		benignOutcome := probeHTTPTargetWithClient(ctx, sharedClient, probeCfg.BenignPathURL, benignCfg, false)
		evidence.BenignPathURL = benignOutcome.URL
		evidence.BenignPathStatus = benignOutcome.StatusCode
		evidence.BenignWAFHeader = benignOutcome.WAFHeader
		if benignOutcome.Error != nil {
			evidence.BenignError = benignOutcome.Error.Error()
		}

		if benignOutcome.Error != nil || benignOutcome.IsBlocked {
			benignHealthy = false
		}
	}

	// If origin was healthy (e.g. /health 200 in local testlab) OR benign path was blocked (e.g. /scan/fn_canary 403),
	// then the route itself is rejecting even harmless requests -> genuine IP/route WAF block.
	if !benignHealthy {
		return DiagnosticVerdict{
			Verdict:          VerdictIPBan,
			Confidence:       95,
			Title:            "Route / WAF Block Confirmed",
			Summary:          fmt.Sprintf("Origin responded %d, but even a benign test path (%s) was rejected with %d (%s). Your IP or session is blocked on this route.", evidence.RootStatus, evidence.BenignPathURL, evidence.BenignPathStatus, evidence.BenignWAFHeader),
			EvidenceHeader:   coalesceString(evidence.BenignWAFHeader, forensics.DominantWAFHeader),
			FixedBodyLength:  forensics.DominantBodyLength,
			SamplePayloads:   forensics.SamplePayloads,
			Canary:           evidence,
			SuggestedActions: []string{"rotate_ip", "test_connection"},
		}
	}

	// Both Root Canary and Benign Path Canary succeeded cleanly!
	// Inspect rolling forensics to verify whether the wordlist triggered WAF payload rules.
	hasFixedTinyWAFBody := forensics.BlockedCount >= 3 && forensics.DistinctBodyLengths == 1 && forensics.DominantBodyLength > 0 && forensics.DominantBodyLength <= 256
	hasPayloadWAFSignal := forensics.TraversalPayloadHit > 0 || forensics.DominantWAFHeader != "" || hasFixedTinyWAFBody

	if hasPayloadWAFSignal {
		conf := 85
		if forensics.TraversalPayloadHit > 0 && forensics.DominantWAFHeader != "" {
			conf = 99
		} else if forensics.TraversalPayloadHit > 0 || (forensics.DominantWAFHeader != "" && hasFixedTinyWAFBody) {
			conf = 96
		}

		return DiagnosticVerdict{
			Verdict:    VerdictWordlistTrigger,
			Confidence: conf,
			Title:      "Wordlist Payloads Triggering WAF — Your IP Is Healthy",
			Summary: fmt.Sprintf(
				"Differential canary verification confirmed your IP is NOT banned (origin returned %d, benign path returned %d). Recent 403 blocks correlate with wordlist patterns or WAF payload rules (%s).",
				evidence.RootStatus,
				coalesceInt(evidence.BenignPathStatus, evidence.RootStatus),
				coalesceString(forensics.DominantWAFHeader, "traversal/bypass tokens"),
			),
			EvidenceHeader:   forensics.DominantWAFHeader,
			FixedBodyLength:  forensics.DominantBodyLength,
			SamplePayloads:   forensics.SamplePayloads,
			Canary:           evidence,
			SuggestedActions: []string{"sanitize_wordlist", "switch_wordlist", "ignore"},
		}
	}

	// Healthy probes without traversal tokens/uniform header -> endpoint-level access restrictions
	return DiagnosticVerdict{
		Verdict:    VerdictEndpointRule,
		Confidence: 75,
		Title:      "Endpoint-Specific Access Restrictions Detected",
		Summary: fmt.Sprintf(
			"Target origin (%d) and benign paths (%d) are reachable. Some requested endpoints returned 403 due to protected endpoint rules rather than an IP ban.",
			evidence.RootStatus,
			coalesceInt(evidence.BenignPathStatus, evidence.RootStatus),
		),
		SamplePayloads:   forensics.SamplePayloads,
		Canary:           evidence,
		SuggestedActions: []string{"ignore", "switch_wordlist"},
	}
}

func coalesceString(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func coalesceInt(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}
