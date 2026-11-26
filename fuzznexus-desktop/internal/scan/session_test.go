package scan

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

	"fuzznexus-desktop/internal/blobs"
	"fuzznexus-desktop/internal/output"
	"fuzznexus-desktop/internal/store"
)

func setupMockScanEnvironment(t *testing.T) (string, string, *httptest.Server, *store.Store, *blobs.Storage) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "ffuf-scan-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test wordlist
	wordlistPath := filepath.Join(tempDir, "wordlist.txt")
	words := "admin\napi\nsecret\nlogin\ntest1\ntest2\ntest3\ntest4\ntest5\ntest6\n"
	if err := os.WriteFile(wordlistPath, []byte(words), 0600); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write wordlist: %v", err)
	}

	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html>Admin Portal</html>"))
		case "/api":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found"))
		}
	}))

	st, err := store.NewStore(tempDir)
	if err != nil {
		server.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	bl, err := blobs.NewStorage(tempDir, blobs.DefaultSessionQuota)
	if err != nil {
		st.Close()
		server.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create blobs storage: %v", err)
	}

	return tempDir, wordlistPath, server, st, bl
}

func TestFullScanLifecycle(t *testing.T) {
	tempDir, wordlistPath, server, st, bl := setupMockScanEnvironment(t)
	defer func() {
		st.Close()
		bl.Close()
		server.Close()
		os.RemoveAll(tempDir)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetURL := server.URL + "/FUZZ"
	conf, err := BuildFFUFConfig(ctx, cancel, targetURL, wordlistPath, 2, 0)
	if err != nil {
		t.Fatalf("BuildFFUFConfig failed: %v", err)
	}

	sessionID, err := st.CreateSession(`{"target":"test"}`, `{"target":"test"}`)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	var stateMu sync.Mutex
	var statesReceived []State
	var commitCount atomic.Int64

	sess, err := NewSession(SessionOptions{
		SessionID: sessionID,
		Config:    conf,
		Store:     st,
		Blobs:     bl,
		CancelCtx: cancel,
		EmitCommit: func(e output.CommittedEvent) {
			commitCount.Add(int64(e.Count))
		},
		EmitState: func(s State, msg string) {
			stateMu.Lock()
			statesReceived = append(statesReceived, s)
			stateMu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if err := sess.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for completion
	sess.Wait()

	state, lastErr := sess.State()
	if state != StateCompleted {
		t.Errorf("Expected final state Completed, got %s (lastErr: %s)", state, lastErr)
	}

	// Verify results in SQLite
	results, _, err := st.GetResultsSince(sessionID, 0, 50)
	if err != nil {
		t.Fatalf("GetResultsSince failed: %v", err)
	}

	// Only 200s (admin and api) should match the default matcher (200, 204, 301, 302, 307, 401, 403, 405, 500)
	if len(results) != 2 {
		t.Errorf("Expected 2 matched results (admin, api), got %d", len(results))
	}

	// Verify blobs stored for matched results
	payload, err := bl.GetResultPayload(sessionID, 1, 1024)
	if err != nil {
		t.Fatalf("GetResultPayload failed: %v", err)
	}
	if payload.ResponseText == "" {
		t.Error("Expected non-empty response text in blob payload")
	}
}

func TestPauseResumeAndStop(t *testing.T) {
	tempDir, wordlistPath, server, st, bl := setupMockScanEnvironment(t)
	defer func() {
		st.Close()
		bl.Close()
		server.Close()
		os.RemoveAll(tempDir)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetURL := server.URL + "/FUZZ"
	conf, err := BuildFFUFConfig(ctx, cancel, targetURL, wordlistPath, 1, 5) // Low rate 5 req/s
	if err != nil {
		t.Fatalf("BuildFFUFConfig failed: %v", err)
	}

	sessionID, _ := st.CreateSession(`{}`, `{}`)

	sess, err := NewSession(SessionOptions{
		SessionID: sessionID,
		Config:    conf,
		Store:     st,
		Blobs:     bl,
		CancelCtx: cancel,
	})
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if err := sess.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Test Pause
	time.Sleep(50 * time.Millisecond)
	if err := sess.Pause(); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
	s, _ := sess.State()
	if s != StatePaused {
		t.Errorf("Expected state Paused, got %s", s)
	}

	// Test Resume
	if err := sess.Resume(); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	s, _ = sess.State()
	if s != StateRunning {
		t.Errorf("Expected state Running, got %s", s)
	}

	// Test ChangeRate
	sess.ChangeRate(50)

	// Test Stop
	if err := sess.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	sess.Wait()

	finalState, _ := sess.State()
	if finalState != StateStopped {
		t.Errorf("Expected state Stopped, got %s", finalState)
	}
}

func TestFatalDBFailureHaltsScanAndFinalizes(t *testing.T) {
	tempDir, wordlistPath, server, st, bl := setupMockScanEnvironment(t)
	defer func() {
		bl.Close()
		server.Close()
		os.RemoveAll(tempDir)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetURL := server.URL + "/FUZZ"
	conf, err := BuildFFUFConfig(ctx, cancel, targetURL, wordlistPath, 2, 0)
	if err != nil {
		t.Fatalf("BuildFFUFConfig failed: %v", err)
	}

	sessionID, _ := st.CreateSession(`{}`, `{}`)

	// Close store prematurely to induce persistent DB write failure
	st.Close()

	sess, err := NewSession(SessionOptions{
		SessionID: sessionID,
		Config:    conf,
		Store:     st,
		Blobs:     bl,
		CancelCtx: cancel,
	})
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if err := sess.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	sess.Wait()

	finalState, lastErr := sess.State()
	if finalState != StateFailed {
		t.Errorf("Expected state Failed on DB error, got %s (err: %s)", finalState, lastErr)
	}
}

func TestSession_SafeStopWorkflowTrigger(t *testing.T) {
	tempDir, wordlistPath, mockSrv, st, bl := setupMockScanEnvironment(t)
	defer func() {
		mockSrv.Close()
		bl.Close()
		st.Close()
		os.RemoveAll(tempDir)
	}()

	// Server that returns 403 Forbidden to trigger Safe Stop
	blockingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer blockingServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetURL := blockingServer.URL + "/FUZZ"
	conf, err := BuildFFUFConfig(ctx, cancel, targetURL, wordlistPath, 2, 0)
	if err != nil {
		t.Fatalf("BuildFFUFConfig failed: %v", err)
	}

	sessionID, _ := st.CreateSession(`{}`, `{}`)

	promptChan := make(chan SafeStopPromptDTO, 1)

	sess, err := NewSession(SessionOptions{
		SessionID: sessionID,
		Config:    conf,
		Store:     st,
		Blobs:     bl,
		CancelCtx: cancel,
		EmitSafeStop: func(prompt SafeStopPromptDTO) {
			select {
			case promptChan <- prompt:
			default:
			}
		},
	})
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if err := sess.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Expect prompt within 5 seconds
	select {
	case prompt := <-promptChan:
		if prompt.SessionID != sessionID {
			t.Errorf("Expected session ID %d, got %d", sessionID, prompt.SessionID)
		}
		if prompt.Title == "" {
			t.Errorf("Expected non-empty title in prompt")
		}
		t.Logf("Successfully captured SafeStopPrompt: %s | buttons: %v", prompt.Title, prompt.Buttons)

		// Test TestConnection
		testRes, err := sess.TestConnection(context.Background())
		if err != nil {
			t.Fatalf("TestConnection returned unexpected error: %v", err)
		}
		// Server returns 403, so testRes.Success should be false
		if testRes.Success {
			t.Errorf("Expected TestConnection success to be false on 403 server")
		}

		// Test CancelSafeStop
		if err := sess.CancelSafeStop(); err != nil {
			t.Fatalf("CancelSafeStop failed: %v", err)
		}

	case <-time.After(10 * time.Second):
		t.Fatalf("Timed out waiting for SafeStopPrompt to be emitted")
	}

	sess.Wait()
}

func TestSession_WordlistWAFTrigger_NoFalseIPBanPopup(t *testing.T) {
	tempDir, _, mockSrv, st, bl := setupMockScanEnvironment(t)
	defer func() {
		mockSrv.Close()
		bl.Close()
		st.Close()
		os.RemoveAll(tempDir)
	}()

	// Write a wordlist simulating builtin_bypasses_heuristics.txt (.// and .;/ prefixes)
	wlPath := filepath.Join(tempDir, "bypass_wl.txt")
	wlContent := ".//system\n.//manager\n.//internal\n.//v1\n.//portal\n.//root\n.;/admin\n.;/api\n"
	if err := os.WriteFile(wlPath, []byte(wlContent), 0600); err != nil {
		t.Fatalf("failed writing bypass wordlist: %v", err)
	}

	// Simulate parsaazari.com:
	// - Root / -> 200 OK
	// - /health -> 404 (not defined)
	// - Any path containing .// or .;/ -> 403 x-blocked-by: bootstrap-security (13 bytes)
	// - Clean random path (/fn_canary_*) -> 404 HTML
	wafServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, ".//") || strings.Contains(r.RequestURI, ".;/") {
			w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
			w.Header().Set("x-blocked-by", "bootstrap-security")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("403 Forbidden"))
			return
		}
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html>Home Page</html>"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("<html>404 Not Found</html>"))
	}))
	defer wafServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf, err := BuildFFUFConfig(ctx, cancel, wafServer.URL+"/FUZZ", wlPath, 2, 0)
	if err != nil {
		t.Fatalf("BuildFFUFConfig failed: %v", err)
	}

	sessionID, _ := st.CreateSession(`{}`, `{}`)

	var safeStopTriggered atomic.Bool
	advisoryChan := make(chan AdvisoryAlertDTO, 4)

	sess, err := NewSession(SessionOptions{
		SessionID: sessionID,
		Config:    conf,
		Store:     st,
		Blobs:     bl,
		CancelCtx: cancel,
		EmitSafeStop: func(prompt SafeStopPromptDTO) {
			safeStopTriggered.Store(true)
		},
		EmitAdvisory: func(alert AdvisoryAlertDTO) {
			advisoryChan <- alert
		},
	})
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if err := sess.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	sess.Wait()

	if safeStopTriggered.Load() {
		t.Fatalf("expected NO false SafeStop IP-ban popup when origin and benign paths are healthy")
	}

	select {
	case alert := <-advisoryChan:
		if alert.Verdict != "WORDLIST_WAF_TRIGGER" {
			t.Fatalf("expected WORDLIST_WAF_TRIGGER verdict, got %q", alert.Verdict)
		}
		if alert.EvidenceHeader != "x-blocked-by: bootstrap-security" {
			t.Fatalf("expected x-blocked-by header evidence, got %q", alert.EvidenceHeader)
		}
		if alert.FixedBodyLength != 13 {
			t.Fatalf("expected fixed body length 13, got %d", alert.FixedBodyLength)
		}
		if alert.Canary.RootStatus != 200 || alert.Canary.BenignPathStatus != 404 {
			t.Fatalf("expected Canary Root=200 & Benign=404, got root=%d benign=%d", alert.Canary.RootStatus, alert.Canary.BenignPathStatus)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("expected AdvisoryAlertDTO to be emitted for WORDLIST_WAF_TRIGGER")
	}

	finalState, _ := sess.State()
	if finalState != StateCompleted {
		t.Errorf("expected scan to complete cleanly in non-blocking advisory mode, got %s", finalState)
	}
}

func TestDeriveProbeTargets_SafetyRules(t *testing.T) {
	realTargets := deriveProbeTargets("https://parsaazari.com/FUZZ", nil, "")
	if realTargets.RootCanaryURL != "https://parsaazari.com/" {
		t.Errorf("unexpected real root canary: %s", realTargets.RootCanaryURL)
	}
	if !realTargets.AllowNon200Origin || !realTargets.CanProbeBenign || realTargets.IsLocalTarget {
		t.Errorf("unexpected flags for real site: %+v", realTargets)
	}

	subTargets := deriveProbeTargets("https://FUZZ.parsaazari.com/", nil, "")
	if subTargets.CanProbeBenign {
		t.Errorf("subdomain FUZZ must skip benign path DNS probing")
	}

	dockerTargets := deriveProbeTargets("http://127.0.0.1:8090/scan/FUZZ", nil, "")
	if dockerTargets.RootCanaryURL != "http://127.0.0.1:8090/health" || !dockerTargets.IsLocalTarget || dockerTargets.AllowNon200Origin {
		t.Errorf("unexpected testlab targets: %+v", dockerTargets)
	}
}

func TestSession_RecordErrorMonotonicRingBuffer(t *testing.T) {
	sess := &Session{}
	for i := 1; i <= 120; i++ {
		sess.RecordError("network timeout")
	}

	errs := sess.GetRecentErrors()
	if len(errs) != 100 {
		t.Fatalf("expected 100 buffered errors, got %d", len(errs))
	}
	if errs[0].Index != 21 || errs[99].Index != 120 {
		t.Fatalf("expected monotonic indices 21..120, got %d..%d", errs[0].Index, errs[99].Index)
	}
	seen := make(map[int]bool, len(errs))
	for _, e := range errs {
		if seen[e.Index] {
			t.Fatalf("duplicate error index %d in ring buffer", e.Index)
		}
		seen[e.Index] = true
	}
}
