package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreaker_FastPathStateConsistency(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	var openTransitions atomic.Int32
	cb.SetOnStateChange(func(oldState, newState CircuitState, reason string) {
		if newState == StateOpen {
			openTransitions.Add(1)
		}
	})

	// Send 8 consecutive 403s (all within count < 10 fast-path)
	for i := 0; i < 8; i++ {
		cb.Record(403, false)
	}

	if got := cb.State(); got != StateOpen {
		t.Fatalf("expected StateOpen after fast-path 403s, got %v", got)
	}
	if count := openTransitions.Load(); count != 1 {
		t.Fatalf("expected exactly 1 StateOpen transition, got %d", count)
	}
}

func TestDifferentialCanary_WordlistTrigger_Vs_IPBan(t *testing.T) {
	var canaryHits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Detect raw path traversal attempts (simulate bootstrap-security / ModSecurity rule)
		if strings.Contains(r.RequestURI, ".//") || strings.Contains(r.RequestURI, ".;/") {
			w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
			w.Header().Set("x-blocked-by", "bootstrap-security")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("403 Forbidden"))
			return
		}
		if r.URL.Path == "/" {
			canaryHits.Add(1)
			w.Header().Set("cf-cache-status", "DYNAMIC")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Parsa Azari Home</body></html>"))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/fn_canary_") {
			canaryHits.Add(1)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("<!DOCTYPE html><html><title>404 - Not Found</title></html>"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	payloads := []string{".//system", ".//manager", ".//v1", ".//internal", ".//admin"}
	for _, p := range payloads {
		cb.RecordSample(SampleObservation{
			StatusCode:  403,
			BodyLength:  13,
			WAFHeader:   "x-blocked-by: bootstrap-security",
			PayloadHint: p,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	verdict := EvaluateBlockCause(ctx, cb.Stats(), cb.ForensicsSnapshot(), CanaryProbeConfig{
		RootCanaryURL:     srv.URL + "/",
		BenignPathURL:     srv.URL + "/" + RandomCanarySlug(),
		AllowNon200Origin: true,
		BrowserProfile:    "chrome-macos",
	})

	if verdict.Verdict != VerdictWordlistTrigger {
		t.Fatalf("expected VerdictWordlistTrigger, got %s (%+v)", verdict.Verdict, verdict)
	}
	if verdict.Confidence < 95 {
		t.Fatalf("expected confidence >= 95, got %d", verdict.Confidence)
	}
	if verdict.EvidenceHeader != "x-blocked-by: bootstrap-security" {
		t.Fatalf("unexpected evidence header: %q", verdict.EvidenceHeader)
	}
	if verdict.FixedBodyLength != 13 {
		t.Fatalf("expected FixedBodyLength 13, got %d", verdict.FixedBodyLength)
	}
	if verdict.Canary.RootStatus != 200 || verdict.Canary.BenignPathStatus != 404 {
		t.Fatalf("unexpected canary statuses: root=%d benign=%d", verdict.Canary.RootStatus, verdict.Canary.BenignPathStatus)
	}
	if canaryHits.Load() != 2 {
		t.Fatalf("expected 2 canary probe requests (root + benign), got %d", canaryHits.Load())
	}
}

func TestDifferentialCanary_TrueIPBan_RootBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cf-mitigated", "challenge")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html><title>Just a moment...</title><div id=\"challenge-platform\">Verify you are human</div></html>"))
	}))
	defer srv.Close()

	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	for i := 0; i < 6; i++ {
		cb.RecordSample(SampleObservation{
			StatusCode:  403,
			BodyLength:  110,
			WAFHeader:   "cf-mitigated: challenge",
			PayloadHint: "about",
			SoftBlocked: true,
		})
	}

	verdict := EvaluateBlockCause(context.Background(), cb.Stats(), cb.ForensicsSnapshot(), CanaryProbeConfig{
		RootCanaryURL:     srv.URL + "/",
		BenignPathURL:     srv.URL + "/" + RandomCanarySlug(),
		AllowNon200Origin: true,
		BrowserProfile:    "chrome-macos",
	})

	if verdict.Verdict != VerdictIPBan {
		t.Fatalf("expected VerdictIPBan when root canary fails with 403/challenge, got %s", verdict.Verdict)
	}
}

func TestDifferentialCanary_SoftBlock200OnRoot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cf-mitigated", "challenge")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><title>Just a moment...</title><body><div id='challenge-platform'>Verify you are human</div></body></html>"))
	}))
	defer srv.Close()

	verdict := EvaluateBlockCause(context.Background(), RollingStats{Count403: 5, TotalSamples: 5}, RollingForensics{BlockedCount: 5, HasSoftBlock: true}, CanaryProbeConfig{
		RootCanaryURL:  srv.URL + "/",
		BrowserProfile: "chrome-macos",
	})

	if verdict.Verdict != VerdictIPBan {
		t.Fatalf("expected VerdictIPBan on 200 OK soft-block challenge page, got %s", verdict.Verdict)
	}
}

func TestCircuitBreaker_WordlistBypassDoesNotMaskRealIPBan(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	for i := 0; i < 6; i++ {
		cb.RecordSample(SampleObservation{
			StatusCode:  403,
			BodyLength:  13,
			WAFHeader:   "x-blocked-by: bootstrap-security",
			PayloadHint: ".//system",
		})
	}
	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen prior to bypass acknowledgment, got %v", cb.State())
	}

	// Acknowledge wordlist-specific WAF blocks -> restores Closed state & filters matching noise
	cb.SetWordlistAdvisoryBypass(true, "x-blocked-by: bootstrap-security", 13)
	if cb.State() != StateClosed {
		t.Fatalf("expected StateClosed after acknowledging wordlist bypass, got %v", cb.State())
	}

	// Subsequent traversal payloads matching same signature should NOT re-trip StateOpen
	for i := 0; i < 15; i++ {
		cb.RecordSample(SampleObservation{
			StatusCode:  403,
			BodyLength:  13,
			WAFHeader:   "x-blocked-by: bootstrap-security",
			PayloadHint: ".//internal",
		})
	}
	if cb.State() != StateClosed {
		t.Fatalf("expected StateClosed to hold on acknowledged traversal noise, got %v", cb.State())
	}

	// Now simulate a genuine Cloudflare IP ban affecting clean words -> MUST trip StateOpen!
	for i := 0; i < 30; i++ {
		cb.RecordSample(SampleObservation{
			StatusCode:  403,
			BodyLength:  5601,
			WAFHeader:   "cf-mitigated: challenge",
			PayloadHint: "contact",
			SoftBlocked: true,
		})
	}
	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen when genuine IP ban hits normal payloads, got %v", cb.State())
	}
}

func TestSoftBlockDetector_IgnoresCloudflarePassiveJSD(t *testing.T) {
	sbd := NewSoftBlockDetector(true)

	// Real 200 OK site footer from parsaazari.com carrying Cloudflare passive JSD telemetry
	passiveJSDHTML := []byte(`<!DOCTYPE html><html><head><title>Parsa Azari</title></head><body><h1>Hello</h1><script>var u='/cdn-cgi/challenge-platform/scripts/jsd/main.js?ray=a5fa6c0ccc79f5e8';</script></body></html>`)
	if isSoft, reason := sbd.Check(200, map[string][]string{"Server": {"cloudflare"}}, passiveJSDHTML); isSoft {
		t.Fatalf("expected passive Cloudflare JSD beacon on 200 OK page NOT to trigger soft-block, got reason: %s", reason)
	}

	// Genuine Turnstile / Managed Challenge without cf-mitigated header
	activeChallengeHTML := []byte(`<!DOCTYPE html><html><head><title>Just a moment...</title></head><body><script>window._cf_chl_opt={};var src='/cdn-cgi/challenge-platform/h/b/orchestrate/chl_page/v1';</script></body></html>`)
	if isSoft, _ := sbd.Check(200, nil, activeChallengeHTML); !isSoft {
		t.Fatalf("expected active Cloudflare challenge orchestrate page to trigger soft-block")
	}
}
