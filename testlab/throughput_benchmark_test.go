//go:build testlab

package main

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/assembly"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

// BenchmarkEngineThroughput measures pure in-memory RPS of the ffuf engine with CircuitBreaker active.
func BenchmarkEngineThroughput(b *testing.B) {
	// Spin up ultra-fast mock HTTP server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer srv.Close()

	// Generate wordlist of size b.N
	tempDir := b.TempDir()
	wlPath := filepath.Join(tempDir, "bench_wl.txt")
	f, err := os.Create(wlPath)
	if err != nil {
		b.Fatal(err)
	}
	bw := bufio.NewWriter(f)
	count := b.N
	if count < 1000 {
		count = 1000
	}
	for i := 0; i < count; i++ {
		_, _ = bw.WriteString("w" + strconv.Itoa(i) + "\n")
	}
	_ = bw.Flush()
	_ = f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := fuzznexus.NewConfigOptions()
	opts.HTTP.URL = srv.URL + "/FUZZ"
	opts.Input.Wordlists = []string{wlPath}
	opts.General.Threads = 40 // High concurrency
	opts.General.Quiet = true
	opts.General.Noninteractive = true
	opts.Filter.Status = "200" // Filter 200s so terminal stdout does not bottleneck high load

	conf, err := fuzznexus.ConfigFromOptions(opts, ctx, cancel)
	if err != nil {
		b.Fatalf("ConfigFromOptions failed: %v", err)
	}

	mm, err := filter.FromConfig(opts, true)
	if err != nil {
		b.Fatalf("filter.FromConfig failed: %v", err)
	}
	conf.MatcherManager = mm

	job, err := assembly.BuildJob(conf)
	if err != nil {
		b.Fatalf("BuildJob failed: %v", err)
	}

	b.ResetTimer()
	job.Start()
	b.StopTimer()
}
