package scan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ffuf-desktop/internal/blobs"
	"ffuf-desktop/internal/output"
	"ffuf-desktop/internal/store"
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
