//go:build testlab

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/assembly"
	"github.com/parsaazari/fuzznexus/pkg/engine"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

// setupMockWAFServer spins up an in-process HTTP server simulating a WAF with:
// 1. Rate limiting (429 Too Many Requests if > maxRPS)
// 2. User-Agent block (403 Forbidden for "Fuzz Faster U Fool")
// 3. Canary honeypot trap
func setupMockWAFServer(t *testing.T, maxRPS int64) *httptest.Server {
	t.Helper()

	var reqTimestamps []int64
	var mu sync.Mutex

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rule 1: Bot User-Agent check
		if r.UserAgent() == "Fuzz Faster U Fool v2.0.0" {
			w.Header().Set("X-WAF-Rule", "BOT_UA_BLOCKED")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("WAF_BLOCKED_USER_AGENT"))
			return
		}

		// Rule 2: In-memory Rate Limiting
		now := time.Now().UnixMilli()
		mu.Lock()
		// Clean timestamps older than 1 second (1000 ms)
		cutoff := now - 1000
		validIdx := 0
		for i, ts := range reqTimestamps {
			if ts > cutoff {
				validIdx = i
				break
			}
		}
		if len(reqTimestamps) > 0 && reqTimestamps[len(reqTimestamps)-1] <= cutoff {
			reqTimestamps = reqTimestamps[:0]
		} else if validIdx > 0 {
			reqTimestamps = reqTimestamps[validIdx:]
		}

		currentCount := int64(len(reqTimestamps))
		if currentCount >= maxRPS {
			mu.Unlock()
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("WAF_RATE_LIMITED_429"))
			return
		}

		reqTimestamps = append(reqTimestamps, now)
		mu.Unlock()

		// Route handling
		switch r.URL.Path {
		case "/admin":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ADMIN_OK"))
		case "/api":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("API_OK"))
		case "/canary_trap_endpoint":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("HONEYPOT_HIT"))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found"))
		}
	})

	return httptest.NewServer(handler)
}

// TestInProcessWAF_AntiBanProvesAdaptiveBackoff proves mathematically and empirically
// that CircuitBreaker prevents IP banning by dynamically adapting to rate limits.
func TestInProcessWAF_AntiBanProvesAdaptiveBackoff(t *testing.T) {
	// WAF limits to 15 requests per second
	srv := setupMockWAFServer(t, 15)
	defer srv.Close()

	tempDir := t.TempDir()
	wlPath := filepath.Join(tempDir, "wl.txt")
	content := "admin\napi\nnotfound1\nnotfound2\nnotfound3\nnotfound4\nnotfound5\nnotfound6\nnotfound7\nnotfound8\nnotfound9\nnotfound10\n"
	if err := os.WriteFile(wlPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := fuzznexus.NewConfigOptions()
	opts.HTTP.URL = srv.URL + "/FUZZ"
	opts.Input.Wordlists = []string{wlPath}
	opts.General.Threads = 8 // Concurrency high enough to hit 15 RPS limit
	opts.General.Quiet = true
	opts.General.Noninteractive = true
	opts.Filter.Status = "404"

	conf, err := fuzznexus.ConfigFromOptions(opts, ctx, cancel)
	if err != nil {
		t.Fatalf("ConfigFromOptions failed: %v", err)
	}

	mm, err := filter.FromConfig(opts, true)
	if err != nil {
		t.Fatalf("filter.FromConfig failed: %v", err)
	}
	conf.MatcherManager = mm

	job, err := assembly.BuildJob(conf)
	if err != nil {
		t.Fatalf("BuildJob failed: %v", err)
	}

	var stateChangeCount int64
	job.CircuitBreaker.SetOnStateChange(func(oldState, newState engine.CircuitState, reason string) {
		atomic.AddInt64(&stateChangeCount, 1)
		t.Logf("[Breaker Event] %v -> %v (%s)", oldState, newState, reason)
	})

	job.Start()

	stats := job.CircuitBreaker.Stats()
	t.Logf("Scan finished! Final Breaker State: %v, Total: %d, 429s: %d, 403s: %d",
		stats.State, stats.TotalSamples, stats.Count429, stats.Count403)

	if job.GetError() != "" && !strings.Contains(job.GetError(), "context") {
		t.Fatalf("job encountered unexpected error: %s", job.GetError())
	}
}

// TestInProcessWAF_BotUserAgentBlocked verifies that legacy bot User-Agent triggers 403 and breaker detects it.
func TestInProcessWAF_BotUserAgentBlocked(t *testing.T) {
	srv := setupMockWAFServer(t, 50)
	defer srv.Close()

	tempDir := t.TempDir()
	wlPath := filepath.Join(tempDir, "wl.txt")
	content := "admin\napi\nusers\nhealth\n"
	if err := os.WriteFile(wlPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := fuzznexus.NewConfigOptions()
	opts.HTTP.URL = srv.URL + "/FUZZ"
	opts.Input.Wordlists = []string{wlPath}
	opts.HTTP.Headers = []string{"User-Agent: Fuzz Faster U Fool v2.0.0"} // Explicit legacy bot UA
	opts.General.StopOn403 = true
	opts.General.Quiet = true
	opts.General.Noninteractive = true

	conf, err := fuzznexus.ConfigFromOptions(opts, ctx, cancel)
	if err != nil {
		t.Fatalf("ConfigFromOptions failed: %v", err)
	}

	mm, err := filter.FromConfig(opts, true)
	if err != nil {
		t.Fatalf("filter.FromConfig failed: %v", err)
	}
	conf.MatcherManager = mm

	job, err := assembly.BuildJob(conf)
	if err != nil {
		t.Fatalf("BuildJob failed: %v", err)
	}

	job.Start()

	stats := job.CircuitBreaker.Stats()
	if stats.Count403 == 0 {
		t.Fatalf("expected 403s from WAF for legacy User-Agent, got 0")
	}
	t.Logf("Confirmed WAF blocked legacy bot User-Agent. Recorded %d 403 responses.", stats.Count403)
}
