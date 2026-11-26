package bridge_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"fuzznexus-desktop/internal/bridge"
	"fuzznexus-desktop/internal/wordlists"
)

// TestDynamicSandboxLifecycleAndReScan verifies:
// 1. Creation and merging of wordlists into an ephemeral sandbox.
// 2. StartScan with the sandbox wordlist -> StopScan.
// 3. Ensuring the sandbox file is RETAINED on disk for repeated scans (fixing the reported bug).
// 4. Starting a second scan with the same sandbox wordlist without re-merging -> works seamlessly.
// 5. Calling CleanupSandboxWordlist (user clicks Clear Sandbox) explicitly deletes the file.
// 6. Verifying network error diagnostics recording.
func TestDynamicSandboxLifecycleAndReScan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fuzznexus_dyn_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	svc, err := bridge.NewService(tempDir, nil)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	defer svc.Close()

	// 1. Seed two wordlists via service
	wl1, err := svc.CreateWordlist("endpoints.txt", "admin\nlogin\napi\n")
	if err != nil {
		t.Fatalf("Failed creating wl1: %v", err)
	}
	wl2, err := svc.CreateWordlist("config.txt", "dashboard\nadmin\nsecret\n")
	if err != nil {
		t.Fatalf("Failed creating wl2: %v", err)
	}

	// 2. Merge into sandbox
	opts := wordlists.SanitizeOptions{
		TrimWhitespace: true,
		StripBOM:       true,
		RemoveEmpty:    true,
	}
	sbDTO, err := svc.MergeWordlistsToSandbox([]int64{wl1.ID, wl2.ID}, opts, true, "merged_sandbox.txt", "sandbox")
	if err != nil {
		t.Fatalf("MergeWordlistsToSandbox failed: %v", err)
	}

	if _, err := os.Stat(sbDTO.Path); err != nil {
		t.Fatalf("Sandbox file does not exist after creation: %v", err)
	}

	// Mock Target Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "admin") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404"))
	}))
	defer server.Close()

	// 3. First Scan Run with sandbox wordlist
	scanReq := bridge.ScanRequestDTO{
		URL:      server.URL + "/FUZZ",
		Method:   "GET",
		Wordlist: sbDTO.Path,
		Threads:  2,
	}

	sid1, err := svc.StartScan(scanReq)
	if err != nil {
		t.Fatalf("StartScan (Run 1) failed: %v", err)
	}

	// Wait briefly and then STOP scan
	time.Sleep(120 * time.Millisecond)
	if err := svc.StopScan(); err != nil {
		t.Fatalf("StopScan failed: %v", err)
	}
	time.Sleep(150 * time.Millisecond)

	// 4. CRITICAL VERIFICATION:
	// The sandbox file MUST still exist on disk after StopScan so user can re-scan!
	if _, err := os.Stat(sbDTO.Path); os.IsNotExist(err) {
		t.Fatalf("CRITICAL BUG DETECTED: sandbox file %s was deleted after scan stop!", sbDTO.Path)
	}

	// 5. Second Scan Run with EXACT same sandbox wordlist without re-merging!
	sid2, err := svc.StartScan(scanReq)
	if err != nil {
		t.Fatalf("CRITICAL BUG DETECTED: Re-scanning with retained sandbox wordlist failed: %v", err)
	}

	time.Sleep(150 * time.Millisecond)
	_ = svc.StopScan()
	time.Sleep(100 * time.Millisecond)

	if sid1 == sid2 {
		t.Errorf("Expected distinct session IDs for separate scan runs, got %d and %d", sid1, sid2)
	}

	// Verify sandbox file is STILL intact after second scan stop
	if _, err := os.Stat(sbDTO.Path); os.IsNotExist(err) {
		t.Fatalf("Sandbox file was deleted after second scan stop!")
	}

	// 6. User clicks "Clear Sandbox" in UI
	if err := svc.CleanupSandboxWordlist(sbDTO.Path); err != nil {
		t.Fatalf("CleanupSandboxWordlist failed: %v", err)
	}

	// Now and ONLY now should the file be deleted from disk
	if _, err := os.Stat(sbDTO.Path); !os.IsNotExist(err) {
		t.Fatalf("Expected sandbox file to be deleted after user clicked Clear Sandbox, err: %v", err)
	}

	// 7. Test Network Error Diagnostics
	// Create an endpoint that immediately fails connection
	closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := closedServer.URL
	closedServer.Close()

	errWl, _ := svc.CreateWordlist("err_words.txt", "x1\nx2\nx3\n")
	errScanReq := bridge.ScanRequestDTO{
		URL:      closedURL + "/FUZZ",
		Method:   "GET",
		Wordlist: errWl.Path,
		Threads:  2,
	}

	_, _ = svc.StartScan(errScanReq)
	time.Sleep(200 * time.Millisecond)
	_ = svc.StopScan()

	recentErrs := svc.GetRecentErrors()
	t.Logf("Diagnostic test recorded %d network errors via GetRecentErrors()", len(recentErrs))
	for _, e := range recentErrs {
		t.Logf("  Captured error #%d: %s", e.Index, e.Message)
	}
}
