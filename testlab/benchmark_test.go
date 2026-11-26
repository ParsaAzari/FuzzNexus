//go:build testlab

package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/assembly"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

const (
	WafEndpoint = "http://127.0.0.1:8088"
)

// TestWAFSimulator_SignatureBlock verifies that legacy fuzzer User-Agent is immediately blocked with 403.
func TestWAFSimulator_SignatureBlock(t *testing.T) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", WafEndpoint+"/admin", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	req.Header.Set("User-Agent", "Fuzz Faster U Fool v2.0.0")
	resp, err := client.Do(req)
	if err != nil {
		t.Skipf("WAF simulator container not running on %s: %v", WafEndpoint, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for legacy bot User-Agent, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "BOT_SIGNATURE_USER_AGENT") {
		t.Fatalf("expected WAF rule response, got: %s", string(body))
	}
}

// TestWAFSimulator_BrowserAccess verifies that realistic browser User-Agent passes through to target.
func TestWAFSimulator_BrowserAccess(t *testing.T) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", WafEndpoint+"/admin", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		t.Skipf("WAF simulator container not running on %s: %v", WafEndpoint, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for browser User-Agent, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "ADMIN_DASHBOARD_AUTHORIZED") {
		t.Fatalf("expected admin content, got: %s", string(body))
	}
}

// TestWAFSimulator_AntiBanAdaptiveBackoff executes an automated scan with CircuitBreaker against the rate-limited WAF.
func TestWAFSimulator_AntiBanAdaptiveBackoff(t *testing.T) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(WafEndpoint + "/")
	if err != nil {
		t.Skipf("WAF simulator container not running on %s: %v", WafEndpoint, err)
	}
	_ = resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	wordlistPath := filepath.Join(t.TempDir(), "test_wordlist.txt")
	if err := os.WriteFile(wordlistPath, []byte("admin\nlogin\napi\nhealth\nstatus\n"), 0644); err != nil {
		t.Fatalf("failed to create test wordlist: %v", err)
	}

	opts := fuzznexus.NewConfigOptions()
	opts.HTTP.URL = WafEndpoint + "/FUZZ"
	opts.Input.Wordlists = []string{wordlistPath}
	opts.General.Threads = 5 // Concurrency to test rate limiter
	opts.General.Quiet = true
	opts.General.Noninteractive = true
	opts.Filter.Status = "404" // Filter out 404s

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

	// Verify CircuitBreaker is wired
	if job.CircuitBreaker == nil {
		t.Fatalf("expected CircuitBreaker to be initialized on Job")
	}

	// Start job
	job.Start()

	stats := job.CircuitBreaker.Stats()
	t.Logf("Scan finished! CircuitBreaker stats: State=%v TotalSamples=%d 429s=%d 403s=%d Errors=%d",
		stats.State, stats.TotalSamples, stats.Count429, stats.Count403, stats.CountErrors)

	if job.GetError() != "" && !strings.Contains(job.GetError(), "context") {
		t.Fatalf("unexpected fatal job error: %s", job.GetError())
	}
}
