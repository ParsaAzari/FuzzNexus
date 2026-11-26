package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockPausableJob struct {
	mu      sync.Mutex
	paused  bool
	resumed bool
	stopped bool
}

func (m *mockPausableJob) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paused = true
}

func (m *mockPausableJob) Resume() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paused = false
	m.resumed = true
}

func (m *mockPausableJob) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
}

// 1. Test IPProvider anonymity
func TestIPProvider_Anonymity(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("198.51.100.42\n"))
	}))
	defer ts.Close()

	provider := NewManualVPNProvider(ts.URL)
	ctx := context.Background()
	hash, err := provider.CurrentIPHash(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(hash) != 8 {
		t.Fatalf("expected 8-char hash, got: %s (len: %d)", hash, len(hash))
	}
	if strings.Contains(hash, "198") || strings.Contains(hash, "51") {
		t.Fatalf("raw IP leaked in hash: %s", hash)
	}
}

// 2. Test BanDetector exact locked rules
func TestBanDetector_Rules(t *testing.T) {
	bd := NewBanDetector(DefaultBanDetectorConfig(), nil)

	// RateLimit rule: >= 3 429 in window
	statsRL := RollingStats{
		TotalSamples: 50,
		Count429:     3,
	}
	if cat := bd.ClassifyMetrics(statsRL, false); cat != BanRateLimit {
		t.Errorf("expected BanRateLimit, got: %v", cat)
	}

	// Outage rule: >= 3 5xx and 0 403
	statsOutage := RollingStats{
		TotalSamples: 50,
		Count5xx:     3,
		Count403:     0,
	}
	if cat := bd.ClassifyMetrics(statsOutage, false); cat != BanOutage {
		t.Errorf("expected BanOutage, got: %v", cat)
	}

	// Outage with 403 should NOT be Outage
	statsMixed := RollingStats{
		TotalSamples: 50,
		Count5xx:     3,
		Count403:     1,
	}
	if cat := bd.ClassifyMetrics(statsMixed, false); cat == BanOutage {
		t.Errorf("expected non-Outage when 403 > 0, got: %v", cat)
	}

	// IPBlock rule: >= 5 403/resets and ratio >= 50%
	statsIPBlock := RollingStats{
		TotalSamples: 10,
		Count403:     5,
		CountErrors:  1,
	}
	if cat := bd.ClassifyMetrics(statsIPBlock, false); cat != BanIPBlock {
		t.Errorf("expected BanIPBlock, got: %v", cat)
	}
}

// 3. Test Persistence with AES-256-GCM cookies encryption
func TestPersistence_AESGCM(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "resume_state.json")

	rawCookies := "session_token=secret_xyz123; role=admin"
	encCookies, nonce, err := EncryptCookies(rawCookies)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if encCookies == rawCookies || strings.Contains(encCookies, "secret_xyz123") {
		t.Fatalf("cookies not encrypted: %s", encCookies)
	}

	decrypted, err := DecryptCookies(encCookies, nonce)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}
	if decrypted != rawCookies {
		t.Fatalf("expected %s, got %s", rawCookies, decrypted)
	}

	state := ResumeState{
		CurrentIndex:     50,
		IPHash:           "a1b2c3d4",
		Headers:          map[string]string{"Authorization": "Bearer token"},
		EncryptedCookies: encCookies,
		CookiesNonce:     nonce,
		Timestamp:        time.Now(),
	}

	if err := SaveResumeState(filePath, state); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify file content does not leak cookies or raw IPs
	fileBytes, _ := os.ReadFile(filePath)
	if strings.Contains(string(fileBytes), "secret_xyz123") {
		t.Fatalf("raw cookie leaked in file: %s", string(fileBytes))
	}

	loaded, err := LoadResumeState(filePath)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.CurrentIndex != 50 || loaded.IPHash != "a1b2c3d4" {
		t.Fatalf("loaded state mismatch: %+v", loaded)
	}

	// Test deletion
	if err := DeleteResumeState(filePath); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("file should be deleted")
	}
}

// 4. Test SafeStop Workflow State Machine
func TestSafeStop_Workflow(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	bd := NewBanDetector(DefaultBanDetectorConfig(), nil)
	bd.SetCooloffDuration(10 * time.Millisecond) // fast cooloff for test

	targetBlocked := true
	bd.SetProbeFunc(func(ctx context.Context) (bool, error) {
		return targetBlocked, nil
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if targetBlocked {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	mockJob := &mockPausableJob{}
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")

	wf := NewSafeStopWorkflow(cb, bd, NewManualVPNProvider(ts.URL), mockJob, ts.URL, stateFile)
	wf.SetCurrentIndex(50)
	wf.SetSessionContext("session=abc", map[string]string{"User-Agent": "test"})

	var popupShown bool
	var continueEnabled bool
	wf.OnPopupShown = func(title, msg string, buttons []string) {
		popupShown = true
	}
	wf.OnContinueEnabled = func() {
		continueEnabled = true
	}

	// Simulate 5 403s in Circuit Breaker
	for i := 0; i < 10; i++ {
		cb.Record(403, false)
	}

	ctx := context.Background()
	handled, _ := wf.CheckAndHandleBlock(ctx, false)
	if !handled {
		t.Fatalf("expected CheckAndHandleBlock to handle IPBlock")
	}

	if wf.State() != StatePausedIPBlock {
		t.Fatalf("expected StatePausedIPBlock, got: %v", wf.State())
	}
	if !mockJob.paused {
		t.Fatalf("expected job to be paused")
	}
	if !popupShown {
		t.Fatalf("expected popup to be shown")
	}

	// Test connection while still blocked
	ok, _, _ := wf.TestConnection(ctx)
	if ok {
		t.Fatalf("TestConnection should fail when targetBlocked is true")
	}

	// Now unblock target (simulating user changed IP)
	targetBlocked = false

	ok, ipHash, err := wf.TestConnection(ctx)
	if !ok || err != nil {
		t.Fatalf("TestConnection should succeed: %v", err)
	}
	if len(ipHash) != 8 {
		t.Fatalf("expected 8-char ip hash, got %s", ipHash)
	}
	if !continueEnabled {
		t.Fatalf("expected OnContinueEnabled to fire")
	}

	// Continue
	resumed, err := wf.Continue(ctx)
	if !resumed || err != nil {
		t.Fatalf("Continue failed: %v", err)
	}

	if wf.State() != StateClosed {
		t.Fatalf("expected StateClosed after continue, got: %v", wf.State())
	}
	if !mockJob.resumed {
		t.Fatalf("expected job to be resumed")
	}

	// Verify state file deleted on successful continue
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Fatalf("state file should be deleted after successful continue")
	}
}
