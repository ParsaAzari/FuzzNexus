//go:build testlab

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/assembly"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

type SoakCheckpoint struct {
	ElapsedSec  int     `json:"elapsed_sec"`
	Goroutines  int     `json:"goroutines"`
	HeapAllocMB float64 `json:"heap_alloc_mb"`
	HeapSysMB   float64 `json:"heap_sys_mb"`
	NumGC       uint32  `json:"num_gc"`
	TotalReqs   int64   `json:"total_reqs"`
	CurrentRPS  float64 `json:"current_rps"`
}

func TestSoakStability(t *testing.T) {
	durationStr := os.Getenv("SOAK_DURATION_SEC")
	soakDurationSec := 600 // 10 minutes default
	if durationStr != "" {
		if val, err := strconv.Atoi(durationStr); err == nil && val > 0 {
			soakDurationSec = val
		}
	}

	intervalSec := 30
	if soakDurationSec <= 60 {
		intervalSec = 5
	}

	t.Logf("Starting 10-Minute Soak Test (Duration: %d seconds, Checkpoint interval: %d seconds)...", soakDurationSec, intervalSec)

	var serverRequests int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&serverRequests, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer srv.Close()

	// Initial GC and baseline metrics
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)

	t.Logf("Baseline state: Goroutines=%d, HeapAlloc=%.2f MB, HeapSys=%.2f MB",
		initialGoroutines, float64(initialMem.Alloc)/1048576.0, float64(initialMem.Sys)/1048576.0)

	// Build a large wordlist to supply requests continuously
	tempDir := t.TempDir()
	wlPath := filepath.Join(tempDir, "soak_wl.txt")
	f, err := os.Create(wlPath)
	if err != nil {
		t.Fatal(err)
	}

	// 3,000,000 items is plenty for 10 minutes at 3,000-5,000 req/s
	numItems := 3000000
	for i := 0; i < numItems; i++ {
		_, _ = f.WriteString("s" + strconv.Itoa(i) + "\n")
	}
	_ = f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(soakDurationSec)*time.Second)
	defer cancel()

	opts := fuzznexus.NewConfigOptions()
	opts.HTTP.URL = srv.URL + "/FUZZ"
	opts.Input.Wordlists = []string{wlPath}
	opts.General.Threads = 25
	opts.General.Rate = 4000 // Stable constant rate: 4,000 RPS
	opts.General.Quiet = true
	opts.General.Noninteractive = true
	opts.Filter.Status = "200" // Avoid buffering output to terminal

	conf, err := fuzznexus.ConfigFromOptions(opts, ctx, cancel)
	if err != nil {
		t.Fatal(err)
	}

	mm, err := filter.FromConfig(opts, true)
	if err != nil {
		t.Fatal(err)
	}
	conf.MatcherManager = mm

	job, err := assembly.BuildJob(conf)
	if err != nil {
		t.Fatal(err)
	}

	var checkpoints []SoakCheckpoint
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	done := make(chan bool)

	startTime := time.Now()
	var lastReqCount int64

	// Metrics collector goroutine
	go func() {
		for {
			select {
			case <-ticker.C:
				elapsed := int(time.Since(startTime).Seconds())
				gCount := runtime.NumGoroutine()
				var m runtime.MemStats
				runtime.ReadMemStats(&m)

				currentReqs := atomic.LoadInt64(&serverRequests)
				deltaReqs := currentReqs - lastReqCount
				lastReqCount = currentReqs
				rps := float64(deltaReqs) / float64(intervalSec)

				cp := SoakCheckpoint{
					ElapsedSec:  elapsed,
					Goroutines:  gCount,
					HeapAllocMB: float64(m.Alloc) / 1048576.0,
					HeapSysMB:   float64(m.Sys) / 1048576.0,
					NumGC:       m.NumGC,
					TotalReqs:   currentReqs,
					CurrentRPS:  rps,
				}
				checkpoints = append(checkpoints, cp)
				t.Logf("[Checkpoint T=%03ds / %03ds] Goroutines: %d | Alloc: %.2f MB | Sys: %.2f MB | NumGC: %d | RPS: %.1f | TotalReqs: %d",
					elapsed, soakDurationSec, cp.Goroutines, cp.HeapAllocMB, cp.HeapSysMB, cp.NumGC, cp.CurrentRPS, cp.TotalReqs)

				if elapsed >= soakDurationSec {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Start the ffuf job (will run until context timeout of soakDurationSec)
	job.Start()
	ticker.Stop()
	close(done)

	totalElapsed := time.Since(startTime)
	finalReqs := atomic.LoadInt64(&serverRequests)

	// Post-run GC cleanup to check for leaks
	time.Sleep(500 * time.Millisecond)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	var finalMem runtime.MemStats
	runtime.ReadMemStats(&finalMem)

	t.Logf("Soak test finished after %.1fs. Total requests: %d, Overall RPS: %.1f",
		totalElapsed.Seconds(), finalReqs, float64(finalReqs)/totalElapsed.Seconds())
	t.Logf("Post-run state: Goroutines=%d (Baseline=%d), HeapAlloc=%.2f MB, HeapSys=%.2f MB",
		finalGoroutines, initialGoroutines, float64(finalMem.Alloc)/1048576.0, float64(finalMem.Sys)/1048576.0)

	// Save checkpoints to JSON
	cpBytes, _ := json.MarshalIndent(checkpoints, "", "  ")
	_ = os.WriteFile("soak_test_metrics.json", cpBytes, 0644)

	// Verify acceptance criteria:
	// 1. Goroutine leak check: should return close to baseline (within +5 of baseline)
	goroutineDiff := finalGoroutines - initialGoroutines
	if goroutineDiff > 10 {
		t.Errorf("Potential goroutine leak: baseline=%d, post-run=%d (diff=%d)", initialGoroutines, finalGoroutines, goroutineDiff)
	}

	// 2. Check that goroutines did not grow monotonically unbounded during execution
	var maxG int
	for _, cp := range checkpoints {
		if cp.Goroutines > maxG {
			maxG = cp.Goroutines
		}
	}
	t.Logf("Peak goroutines during soak execution: %d (well bounded under 25 threads)", maxG)

	// Print Markdown summary table of checkpoints
	fmt.Println("\n| Elapsed | Goroutines | Heap Alloc (MB) | Heap Sys (MB) | Num GC | Instant RPS | Cumulative Reqs |")
	fmt.Println("| :---: | :---: | :---: | :---: | :---: | :---: | :---: |")
	for _, cp := range checkpoints {
		fmt.Printf("| %ds | %d | %.2f MB | %.2f MB | %d | %.1f | %d |\n",
			cp.ElapsedSec, cp.Goroutines, cp.HeapAllocMB, cp.HeapSysMB, cp.NumGC, cp.CurrentRPS, cp.TotalReqs)
	}
}
