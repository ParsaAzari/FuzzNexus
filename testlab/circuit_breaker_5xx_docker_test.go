//go:build testlab

package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/assembly"
	"github.com/parsaazari/fuzznexus/pkg/engine"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

// TestCircuitBreaker_Docker502Integration tests Circuit Breaker reaction against real Docker target returning 502
func TestCircuitBreaker_Docker502Integration(t *testing.T) {
	targetURL := "http://127.0.0.1:8088/artificial-502"

	// Verify target is up
	resp, err := http.Get(targetURL)
	if err != nil {
		t.Skipf("Docker target not accessible: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("Expected 502 from docker target, got %d", resp.StatusCode)
	}

	cb := engine.NewCircuitBreaker(engine.DefaultCircuitBreakerConfig())
	var transitionSample int
	var transitionReason string

	cb.SetOnStateChange(func(oldState, newState engine.CircuitState, reason string) {
		if newState == engine.StateHalfOpen && transitionSample == 0 {
			transitionReason = reason
		}
	})

	client := &http.Client{Timeout: 3 * time.Second}
	for i := 1; i <= 15; i++ {
		r, err := client.Get(targetURL)
		if err != nil {
			cb.Record(0, true)
		} else {
			cb.Record(r.StatusCode, false)
			_ = r.Body.Close()
		}

		if cb.Stats().State == engine.StateHalfOpen && transitionSample == 0 {
			transitionSample = i
		}
	}

	t.Logf("=== 5xx DOCKER INTEGRATION TEST ===")
	t.Logf("Target: %s (Nginx 502 simulator)", targetURL)
	t.Logf("Triggered state: Half-Open at request sample: %d (<= 10: %v)", transitionSample, transitionSample <= 10 && transitionSample > 0)
	t.Logf("Transition Reason: %s", transitionReason)
	t.Logf("Circuit Stats: Total=%d, 5xx Count=%d (Ratio=%.2f), State=%s",
		cb.Stats().TotalSamples, cb.Stats().Count5xx, cb.Stats().Ratio5xx, cb.Stats().State)

	if transitionSample > 10 || transitionSample == 0 {
		t.Fatalf("Acceptance criteria failed: Expected transition at sample <= 10, got %d", transitionSample)
	}
}

// TestJob_Docker502BackoffExecution tests ffuf Job execution against the 502 endpoint
func TestJob_Docker502BackoffExecution(t *testing.T) {
	tempDir := t.TempDir()
	wlPath := filepath.Join(tempDir, "wl_502.txt")
	f, _ := os.Create(wlPath)
	for i := 0; i < 20; i++ {
		_, _ = f.WriteString("probe\n")
	}
	_ = f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := fuzznexus.NewConfigOptions()
	opts.HTTP.URL = "http://127.0.0.1:8088/artificial-502?q=FUZZ"
	opts.Input.Wordlists = []string{wlPath}
	opts.General.Threads = 2
	opts.General.Quiet = true
	opts.General.Noninteractive = true
	opts.Filter.Status = "200"

	conf, err := fuzznexus.ConfigFromOptions(opts, ctx, cancel)
	if err != nil {
		t.Fatal(err)
	}
	mm, _ := filter.FromConfig(opts, true)
	conf.MatcherManager = mm

	job, err := assembly.BuildJob(conf)
	if err != nil {
		t.Fatal(err)
	}

	job.Start()

	stats := job.CircuitBreaker.Stats()
	t.Logf("Job finished against 502 endpoint. CircuitBreaker State=%s, 5xx Count=%d", stats.State, stats.Count5xx)
	if stats.Count5xx < 10 {
		t.Errorf("Expected at least 10 5xx recorded, got %d", stats.Count5xx)
	}
	if stats.State != engine.StateHalfOpen {
		t.Errorf("Expected CircuitBreaker to be HalfOpen, got %s", stats.State)
	}
}
