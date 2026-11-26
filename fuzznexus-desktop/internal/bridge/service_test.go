package bridge

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func setupTestBridge(t *testing.T) (*Service, string, string, *httptest.Server) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "ffuf-bridge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	wordlistPath := filepath.Join(tempDir, "words.txt")
	words := "item1\nitem2\nitem3\nadmin\n"
	if err := os.WriteFile(wordlistPath, []byte(words), 0600); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write wordlist: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ADMIN OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("404"))
		}
	}))

	svc, err := NewService(tempDir, nil)
	if err != nil {
		server.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("NewService failed: %v", err)
	}

	return svc, tempDir, wordlistPath, server
}

func TestBridgeStartAndQuery(t *testing.T) {
	svc, tempDir, wordlistPath, server := setupTestBridge(t)
	defer func() {
		_ = svc.Close()
		server.Close()
		os.RemoveAll(tempDir)
	}()

	var eventsMu sync.Mutex
	var receivedEvents []string
	svc.SetEventEmitter(func(name string, data interface{}) {
		eventsMu.Lock()
		receivedEvents = append(receivedEvents, name)
		eventsMu.Unlock()
	})

	req := ScanRequestDTO{
		URL:      server.URL + "/FUZZ",
		Method:   "GET",
		Wordlist: wordlistPath,
		Threads:  2,
	}

	sessionID, err := svc.StartScan(req)
	if err != nil {
		t.Fatalf("StartScan failed: %v", err)
	}
	if sessionID != 1 {
		t.Errorf("Expected session ID 1, got %d", sessionID)
	}

	// Poll until results appear or complete
	var lastID int64
	var foundAdmin bool
	for attempts := 0; attempts < 30; attempts++ {
		page, err := svc.GetResultsSince(sessionID, lastID, 50)
		if err == nil && len(page.Results) > 0 {
			lastID = page.LastID
			for _, r := range page.Results {
				if r.Inputs["FUZZ"] == "admin" {
					foundAdmin = true
				}
			}
		}
		if foundAdmin {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !foundAdmin {
		t.Error("Expected to find 'admin' match in results")
	}

	// Test GetSessions
	sessions, err := svc.GetSessions(10)
	if err != nil {
		t.Fatalf("GetSessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	// Test ExportResults JSON
	jsonPath := filepath.Join(tempDir, "export.json")
	if err := svc.ExportResults(sessionID, "json", jsonPath); err != nil {
		t.Fatalf("ExportResults JSON failed: %v", err)
	}
	if data, err := os.ReadFile(jsonPath); err != nil || len(data) == 0 {
		t.Errorf("Expected non-empty JSON export file")
	}

	// Test ExportResults CSV
	csvPath := filepath.Join(tempDir, "export.csv")
	if err := svc.ExportResults(sessionID, "csv", csvPath); err != nil {
		t.Fatalf("ExportResults CSV failed: %v", err)
	}
	if data, err := os.ReadFile(csvPath); err != nil || len(data) == 0 {
		t.Errorf("Expected non-empty CSV export file")
	}

	// Test ExportResults TXT
	txtPath := filepath.Join(tempDir, "export.txt")
	if err := svc.ExportResults(sessionID, "txt", txtPath); err != nil {
		t.Fatalf("ExportResults TXT failed: %v", err)
	}
	if data, err := os.ReadFile(txtPath); err != nil || len(data) == 0 {
		t.Errorf("Expected non-empty TXT export file")
	}

	// Test ExportResults HTML
	htmlPath := filepath.Join(tempDir, "export.html")
	if err := svc.ExportResults(sessionID, "html", htmlPath); err != nil {
		t.Fatalf("ExportResults HTML failed: %v", err)
	}
	if data, err := os.ReadFile(htmlPath); err != nil || !strings.Contains(string(data), "FuzzNexus Scan Report") {
		t.Errorf("Expected valid HTML export file")
	}

	// Test ExportResults Markdown
	mdPath := filepath.Join(tempDir, "export.md")
	if err := svc.ExportResults(sessionID, "md", mdPath); err != nil {
		t.Fatalf("ExportResults MD failed: %v", err)
	}
	if data, err := os.ReadFile(mdPath); err != nil || !strings.Contains(string(data), "| # | Status |") {
		t.Errorf("Expected valid Markdown export file")
	}
}

func TestBridgeValidationAndControls(t *testing.T) {
	svc, tempDir, _, server := setupTestBridge(t)
	defer func() {
		_ = svc.Close()
		server.Close()
		os.RemoveAll(tempDir)
	}()

	// 1. Missing URL
	_, err := svc.StartScan(ScanRequestDTO{Wordlist: "some/path"})
	if err == nil {
		t.Error("Expected error on empty URL")
	}

	// 2. Missing Wordlist
	_, err = svc.StartScan(ScanRequestDTO{URL: "http://example.com/FUZZ"})
	if err == nil {
		t.Error("Expected error on empty Wordlist")
	}

	// 3. Pause/Resume when no session active
	if err := svc.PauseScan(); err == nil {
		t.Error("Expected error on PauseScan without active session")
	}
	if err := svc.ResumeScan(); err == nil {
		t.Error("Expected error on ResumeScan without active session")
	}
	if err := svc.StopScan(); err != nil {
		t.Errorf("Expected nil error on StopScan when idle, got %v", err)
	}
}

func TestBridgeAdvancedFeaturesAndHelpers(t *testing.T) {
	svc, tempDir, wordlistPath, server := setupTestBridge(t)
	defer func() {
		_ = svc.Close()
		server.Close()
		os.RemoveAll(tempDir)
	}()

	// 1. Encoders list contains standard pencode encoders
	encs := svc.GetAvailableEncoders()
	if len(encs) == 0 {
		t.Fatal("Expected non-empty GetAvailableEncoders result")
	}
	var hasURLEncode bool
	for _, e := range encs {
		if e == "urlencode" {
			hasURLEncode = true
			break
		}
	}
	if !hasURLEncode {
		t.Errorf("Expected 'urlencode' in GetAvailableEncoders, got %v", encs)
	}

	// 2. ParseRawHTTPRequest parses method, host, headers (dropping Content-Length), and body
	rawHTTP := "POST /api/login?next=FUZZ HTTP/1.1\r\nHost: target.local\r\nContent-Type: application/json\r\nContent-Length: 99\r\n\r\n{\"user\":\"FUZZ\"}"
	parsed, err := svc.ParseRawHTTPRequest(rawHTTP, "https")
	if err != nil {
		t.Fatalf("ParseRawHTTPRequest failed: %v", err)
	}
	if parsed.Method != "POST" || parsed.URL != "https://target.local/api/login?next=FUZZ" {
		t.Errorf("Unexpected parsed request: %+v", parsed)
	}
	if _, hasCL := parsed.Headers["Content-Length"]; hasCL {
		t.Error("Expected Content-Length to be stripped by ParseRawHTTPRequest")
	}
	if parsed.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header, got %v", parsed.Headers)
	}

	// 3. Invalid replay proxy URL rejected cleanly
	_, err = svc.StartScan(ScanRequestDTO{
		URL:            server.URL + "/FUZZ",
		Wordlist:       wordlistPath,
		ReplayProxyURL: "://bad-proxy",
	})
	if err == nil {
		t.Error("Expected invalid replay proxy URL to return error")
	}

	// 4. Recursion validation requires URL ending in FUZZ
	_, err = svc.StartScan(ScanRequestDTO{
		URL:       server.URL + "/FUZZ/extra",
		Wordlist:  wordlistPath,
		Recursion: true,
	})
	if err == nil {
		t.Error("Expected recursion with non-trailing FUZZ URL to return error")
	}

	// 5. applyMatchersAndFilters preserves CLI SetupFilters semantics
	opts := fuzznexus.NewConfigOptions()
	if addDefault := applyMatchersAndFilters(opts, ScanRequestDTO{MatcherSize: "123"}); addDefault {
		t.Error("Expected addDefaultStatusMatcher=false when only size matcher is set")
	}
	if opts.Matcher.Size != "123" {
		t.Errorf("Expected Matcher.Size=123, got %q", opts.Matcher.Size)
	}
	opts2 := fuzznexus.NewConfigOptions()
	if addDefault := applyMatchersAndFilters(opts2, ScanRequestDTO{Matchers: "all", FilterWords: "10"}); !addDefault {
		t.Error("Expected addDefaultStatusMatcher=true when status matcher is explicitly set")
	}
	if opts2.Filter.Words != "10" {
		t.Errorf("Expected Filter.Words=10, got %q", opts2.Filter.Words)
	}

	// 6. Multi-wordlist + Extensions + OutputDirectory end-to-end scan
	wl2 := filepath.Join(tempDir, "exts.txt")
	if err := os.WriteFile(wl2, []byte("admin\n"), 0600); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(tempDir, "od_out")
	sid, err := svc.StartScan(ScanRequestDTO{
		URL:             server.URL + "/W1",
		Wordlists:       []string{wl2 + ":W1"},
		InputMode:       "clusterbomb",
		Threads:         2,
		OutputDirectory: outDir,
		Extensions:      ".php",
		MatcherStatus:   "200,404",
	})
	if err != nil {
		t.Fatalf("Multi-wordlist StartScan failed: %v", err)
	}

	// Wait briefly for scan to record results and unlock wordlists
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		entries, _ := os.ReadDir(outDir)
		if len(entries) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := os.Stat(outDir); err != nil {
		t.Errorf("Expected OutputDirectory %q to be created by -od, err=%v", outDir, err)
	}
	_ = sid
	_ = svc.StopScan()
}
