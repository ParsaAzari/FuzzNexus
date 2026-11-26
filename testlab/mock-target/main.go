package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	requestCount int64
	countMutex   sync.Mutex
)

func main() {
	mux := http.NewServeMux()

	// Legitimate target endpoints
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		countMutex.Lock()
		requestCount++
		countMutex.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body><h1>Corporate Web App v2.0</h1></body></html>"))
	})

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		countMutex.Lock()
		requestCount++
		countMutex.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ADMIN_DASHBOARD_AUTHORIZED"))
	})

	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		countMutex.Lock()
		requestCount++
		countMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","uptime":"99.99%"}`))
	})

	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		countMutex.Lock()
		requestCount++
		countMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":1,"name":"admin"},{"id":2,"name":"auditor"}]`))
	})

	mux.HandleFunc("/secret_backup.zip", func(w http.ResponseWriter, r *http.Request) {
		countMutex.Lock()
		requestCount++
		countMutex.Unlock()
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("PK\x03\x04MOCK_ZIP_DATA"))
	})

	// Honeypot Canary Endpoint
	mux.HandleFunc("/canary_trap_endpoint", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[HONEYPOT] Canary endpoint hit from %s! User-Agent: %s", r.RemoteAddr, r.UserAgent())
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("HONEYPOT_TRIGGERED: IP flagged for inspection."))
	})

	// Body processing endpoint for Phase 3 deep body tests
	mux.HandleFunc("/api/v1/process-body", func(w http.ResponseWriter, r *http.Request) {
		countMutex.Lock()
		requestCount++
		countMutex.Unlock()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		totalBytes := len(body)
		content := string(body)

		// Look for standard command payload
		payloadKey := "ACTION_COMMAND:EXECUTE_TARGET_TASK"
		offset := strings.Index(content, payloadKey)

		if offset >= 0 {
			log.Printf("[TARGET-ACTION-PROCESSED] Successfully executed action '%s' found at offset %d (Total body length: %d bytes, TransferEncoding: %v)",
				payloadKey, offset, totalBytes, r.TransferEncoding)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{"status":"PROCESSED","action":"%s","offset":%d,"total_bytes":%d}`,
				payloadKey, offset, totalBytes)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"RECEIVED_NO_COMMAND","total_bytes":%d}`, totalBytes)
	})

	// Server stats endpoint
	mux.HandleFunc("/_stats", func(w http.ResponseWriter, r *http.Request) {
		countMutex.Lock()
		c := requestCount
		countMutex.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "total_requests=%d", c)
	})

	server := &http.Server{
		Addr:         ":9090",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Println("[MOCK-TARGET] Listening on :9090...")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
