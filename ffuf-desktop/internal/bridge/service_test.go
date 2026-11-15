package bridge

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

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
