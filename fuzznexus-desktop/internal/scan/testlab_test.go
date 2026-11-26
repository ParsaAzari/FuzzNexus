package scan_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fuzznexus-desktop/internal/blobs"
	"fuzznexus-desktop/internal/scan"
	"fuzznexus-desktop/internal/store"
	_ "modernc.org/sqlite"
)

// setupMiniFuzzLab spins up an in-process mock HTTP target for testing real fuzzing scenarios.
func setupMiniFuzzLab(t *testing.T) (string, string, *httptest.Server, *store.Store, *blobs.Storage) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route 1: Path fuzzing
		if r.URL.Path == "/admin" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ADMIN_DASHBOARD_OK"))
			return
		}
		if r.URL.Path == "/api" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("API_ROOT_OK"))
			return
		}

		// Route 2: Query param fuzzing (?q=secret)
		if r.URL.Path == "/search" && r.URL.Query().Get("q") == "secret" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("SEARCH_FOUND"))
			return
		}

		// Route 3: Extension fuzzing (/index.php)
		if r.URL.Path == "/index.php" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("PHP_EXECUTED"))
			return
		}

		// Route 4: Header fuzzing (X-Custom: secret)
		if r.Header.Get("X-Custom") == "secret" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("HEADER_MATCHED"))
			return
		}

		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
	}))

	tempDir := t.TempDir()
	wlPath := filepath.Join(tempDir, "testlab_wordlist.txt")
	content := "admin\napi\nsecret\nphp\nnotfound\n"
	if err := os.WriteFile(wlPath, []byte(content), 0600); err != nil {
		server.Close()
		t.Fatalf("failed to create test wordlist: %v", err)
	}

	st, err := store.NewStore(tempDir)
	if err != nil {
		server.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	bl, err := blobs.NewStorage(tempDir, blobs.DefaultSessionQuota)
	if err != nil {
		st.Close()
		server.Close()
		t.Fatalf("failed to create blobs: %v", err)
	}

	return tempDir, wlPath, server, st, bl
}

func TestMiniLab_BareURLAutoFuzz(t *testing.T) {
	_, wordlistPath, server, st, bl := setupMiniFuzzLab(t)
	defer server.Close()
	defer st.Close()
	defer bl.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Intentionally provide a bare server.URL without /FUZZ! (e.g. "http://127.0.0.1:xxxxx")
	bareURL := server.URL

	conf, err := scan.BuildFFUFConfig(ctx, cancel, bareURL, wordlistPath, 2, 0)
	if err != nil {
		t.Fatalf("BuildFFUFConfig failed with bare URL: %v", err)
	}

	sessionID, err := st.CreateSession(`{"target":"bare_url_test"}`, `{"target":"bare_url_test"}`)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	sess, err := scan.NewSession(scan.SessionOptions{
		SessionID: sessionID,
		Config:    conf,
		Store:     st,
		Blobs:     bl,
		CancelCtx: cancel,
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := sess.Start(); err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	sess.Wait()

	// Verify that /admin and /api were found (200 OK)
	results, _, err := st.GetResultsSince(sessionID, 0, 100)
	if err != nil {
		t.Fatalf("failed to fetch results: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results (admin, api), got %d", len(results))
	}

	foundAdmin := false
	foundAPI := false
	for _, r := range results {
		if strings.Contains(r.URL, "/admin") && r.Status == 200 {
			foundAdmin = true
		}
		if strings.Contains(r.URL, "/api") && r.Status == 200 {
			foundAPI = true
		}
	}

	if !foundAdmin {
		t.Errorf("expected /admin to be matched with 200 OK")
	}
	if !foundAPI {
		t.Errorf("expected /api to be matched with 200 OK")
	}
}

func TestMiniLab_QueryParameterFuzzing(t *testing.T) {
	_, wordlistPath, server, st, bl := setupMiniFuzzLab(t)
	defer server.Close()
	defer st.Close()
	defer bl.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Query parameter without explicit FUZZ
	queryTarget := server.URL + "/search?q="

	conf, err := scan.BuildFFUFConfig(ctx, cancel, queryTarget, wordlistPath, 2, 0)
	if err != nil {
		t.Fatalf("BuildFFUFConfig failed with query URL: %v", err)
	}

	sessionID, _ := st.CreateSession(`{}`, `{}`)
	sess, err := scan.NewSession(scan.SessionOptions{
		SessionID: sessionID,
		Config:    conf,
		Store:     st,
		Blobs:     bl,
		CancelCtx: cancel,
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := sess.Start(); err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	sess.Wait()

	results, _, err := st.GetResultsSince(sessionID, 0, 100)
	if err != nil {
		t.Fatalf("failed to fetch results: %v", err)
	}

	foundSecret := false
	for _, r := range results {
		if strings.Contains(r.URL, "q=secret") && r.Status == 200 {
			foundSecret = true
		}
	}
	if !foundSecret {
		t.Errorf("expected search query with secret to be found with 200 OK")
	}
}

func TestMiniLab_FileExtensionFuzzing(t *testing.T) {
	_, wordlistPath, server, st, bl := setupMiniFuzzLab(t)
	defer server.Close()
	defer st.Close()
	defer bl.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Extension fuzzing: /index.FUZZ
	extTarget := server.URL + "/index.FUZZ"

	conf, err := scan.BuildFFUFConfig(ctx, cancel, extTarget, wordlistPath, 2, 0)
	if err != nil {
		t.Fatalf("BuildFFUFConfig failed with extension URL: %v", err)
	}

	sessionID, _ := st.CreateSession(`{}`, `{}`)
	sess, err := scan.NewSession(scan.SessionOptions{
		SessionID: sessionID,
		Config:    conf,
		Store:     st,
		Blobs:     bl,
		CancelCtx: cancel,
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := sess.Start(); err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	sess.Wait()

	results, _, err := st.GetResultsSince(sessionID, 0, 100)
	if err != nil {
		t.Fatalf("failed to fetch results: %v", err)
	}

	foundPHP := false
	for _, r := range results {
		if strings.Contains(r.URL, "/index.php") && r.Status == 200 {
			foundPHP = true
		}
	}
	if !foundPHP {
		t.Errorf("expected /index.php to be found with 200 OK")
	}
}
