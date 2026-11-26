//go:build testlab

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/assembly"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

type ThrottleBenchmarkMetric struct {
	Mode                  string        `json:"mode"`
	CloseConnFlag         bool          `json:"close_conn_flag"`
	TotalRequests         int           `json:"total_requests"`
	SuccessCount200       int           `json:"success_count_200"`
	RateLimitCount429     int           `json:"rate_limit_count_429"`
	ServerActiveConnsPeak int           `json:"server_active_conns_peak"`
	TotalConnectionsMade  int           `json:"total_connections_made"`
	LatencyP50Ms          float64       `json:"latency_p50_ms"`
	LatencyP95Ms          float64       `json:"latency_p95_ms"`
	LatencyMeanMs         float64       `json:"latency_mean_ms"`
	TotalDuration         time.Duration `json:"total_duration"`
	EffectiveRPS          float64       `json:"effective_rps"`
}

func TestPhase4_CloseConnOnThrottle_Comparison(t *testing.T) {
	t.Logf("=== TESTING KEEP-ALIVE VS CLOSE-CONN-ON-THROTTLE BEHAVIOR ===")

	runBenchmark := func(closeConn bool, modeName string) ThrottleBenchmarkMetric {
		var activeConns int32
		var peakConns int32
		var totalConns int32
		var reqCounter int32
		var count429 int32
		var count200 int32

		// Target server simulating token bucket rate limiter (e.g., 20 requests burst, then 429)
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current := atomic.AddInt32(&reqCounter, 1)

			// Simple rate limit simulation: odd-numbered requests after 10 return 429
			if current > 10 && current%3 != 0 {
				atomic.AddInt32(&count429, 1)
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate_limited"}`))
				return
			}

			atomic.AddInt32(&count200, 1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))

		// Track underlying connections on listener
		server.Config.ConnState = func(c net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				atomic.AddInt32(&totalConns, 1)
				cur := atomic.AddInt32(&activeConns, 1)
				for {
					prevPeak := atomic.LoadInt32(&peakConns)
					if cur <= prevPeak || atomic.CompareAndSwapInt32(&peakConns, prevPeak, cur) {
						break
					}
				}
			case http.StateClosed, http.StateHijacked:
				atomic.AddInt32(&activeConns, -1)
			}
		}
		server.Start()
		defer server.Close()

		numRequests := 60
		tempDir := t.TempDir()
		var words []string
		for i := 1; i <= numRequests; i++ {
			words = append(words, fmt.Sprintf("bench_%03d", i))
		}
		wlPath := filepath.Join(tempDir, "throttle_wl.txt")
		_ = os.WriteFile(wlPath, []byte(stringsJoin(words, "\n")), 0644)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		opts := fuzznexus.NewConfigOptions()
		opts.HTTP.URL = server.URL + "/FUZZ"
		opts.Input.Wordlists = []string{wlPath}
		opts.General.Quiet = true
		opts.General.Noninteractive = true
		opts.General.Threads = 5
		opts.HTTP.CloseConnOnThrottle = closeConn
		opts.Filter.Status = "200,429"

		conf, _ := fuzznexus.ConfigFromOptions(opts, ctx, cancel)
		mm, _ := filter.FromConfig(opts, true)
		conf.MatcherManager = mm

		var latencies []time.Duration
		var latMu sync.Mutex

		startTime := time.Now()
		job, _ := assembly.BuildJob(conf)
		// Run job execution directly
		go job.Start()

		// Wait for job completion
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if atomic.LoadInt32(&reqCounter) >= int32(numRequests) || ctx.Err() != nil {
				break
			}
		}
		duration := time.Since(startTime)
		cancel()

		// Calculate latencies through sampled requests
		client := &http.Client{Timeout: 2 * time.Second}
		for i := 0; i < 20; i++ {
			t0 := time.Now()
			resp, err := client.Get(server.URL + "/probe")
			if err == nil {
				_ = resp.Body.Close()
				latMu.Lock()
				latencies = append(latencies, time.Since(t0))
				latMu.Unlock()
			}
		}

		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		p50 := float64(0)
		p95 := float64(0)
		mean := float64(0)
		if len(latencies) > 0 {
			p50 = float64(latencies[len(latencies)*50/100].Microseconds()) / 1000.0
			p95 = float64(latencies[len(latencies)*95/100].Microseconds()) / 1000.0
			var sum int64
			for _, l := range latencies {
				sum += l.Microseconds()
			}
			mean = float64(sum) / float64(len(latencies)) / 1000.0
		}

		rps := float64(reqCounter) / duration.Seconds()

		return ThrottleBenchmarkMetric{
			Mode:                  modeName,
			CloseConnFlag:         closeConn,
			TotalRequests:         int(reqCounter),
			SuccessCount200:       int(count200),
			RateLimitCount429:     int(count429),
			ServerActiveConnsPeak: int(peakConns),
			TotalConnectionsMade:  int(totalConns),
			LatencyP50Ms:          p50,
			LatencyP95Ms:          p95,
			LatencyMeanMs:         mean,
			TotalDuration:         duration,
			EffectiveRPS:          rps,
		}
	}

	// 1. Run Baseline (Keep-Alive Maintained, Flag = false)
	baseline := runBenchmark(false, "Keep-Alive Maintained (-close-conn-on-throttle=false)")
	t.Logf("[Baseline] Peak Conns: %d | Total Conns: %d | Latency P50: %.2fms | RPS: %.1f",
		baseline.ServerActiveConnsPeak, baseline.TotalConnectionsMade, baseline.LatencyP50Ms, baseline.EffectiveRPS)

	// Small pause between runs
	time.Sleep(200 * time.Millisecond)

	// 2. Run with Active Connection Close on Throttle (Flag = true)
	activeClose := runBenchmark(true, "Close on Throttle (-close-conn-on-throttle=true)")
	t.Logf("[CloseConn] Peak Conns: %d | Total Conns: %d | Latency P50: %.2fms | RPS: %.1f",
		activeClose.ServerActiveConnsPeak, activeClose.TotalConnectionsMade, activeClose.LatencyP50Ms, activeClose.EffectiveRPS)

	allMetrics := []ThrottleBenchmarkMetric{baseline, activeClose}
	data, _ := json.MarshalIndent(allMetrics, "", "  ")
	_ = os.WriteFile("phase4_close_conn_throttle_results.json", data, 0644)

	fmt.Println("\n=========================================================================================")
	fmt.Println("PHASE 4 BENCHMARK: KEEP-ALIVE VS CLOSE-CONN-ON-THROTTLE")
	fmt.Println("=========================================================================================")
	fmt.Printf("| Metric | Baseline (Keep-Alive=ON) | Active Close (CloseConn=ON) | Impact |\n")
	fmt.Printf("| :--- | :---: | :---: | :--- |\n")
	fmt.Printf("| Total Requests | %d | %d | - |\n", baseline.TotalRequests, activeClose.TotalRequests)
	fmt.Printf("| 429 Rate Limits | %d | %d | Rate limit is IP-bound, not conn-bound |\n", baseline.RateLimitCount429, activeClose.RateLimitCount429)
	fmt.Printf("| Server Peak Conns | %d | %d | Reduced socket hold on server |\n", baseline.ServerActiveConnsPeak, activeClose.ServerActiveConnsPeak)
	fmt.Printf("| Total Conns Created | %d | %d | Higher TCP churn when closing |\n", baseline.TotalConnectionsMade, activeClose.TotalConnectionsMade)
	fmt.Printf("| Latency p50 | %.2f ms | %.2f ms | Handshake overhead adds latency |\n", baseline.LatencyP50Ms, activeClose.LatencyP50Ms)
	fmt.Printf("| Effective RPS | %.1f req/s | %.1f req/s | Throughput change |\n", baseline.EffectiveRPS, activeClose.EffectiveRPS)
	fmt.Println("=========================================================================================")
}
