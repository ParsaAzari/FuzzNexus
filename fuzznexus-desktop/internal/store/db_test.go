package store

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"fuzznexus-desktop/internal/output"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func setupTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "ffuf-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	st, err := NewStore(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	return st, tempDir
}

func TestStoreMigrationAndCRUD(t *testing.T) {
	st, dir := setupTestStore(t)
	defer func() {
		st.Close()
		os.RemoveAll(dir)
	}()

	// 1. Create Session
	sessionID, err := st.CreateSession(`{"url":"http://test"}`, `{"url":"http://test"}`)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if sessionID != 1 {
		t.Errorf("Expected session ID 1, got %d", sessionID)
	}

	// 2. Insert Batch
	batch := []output.ResultDTO{
		{
			ID:          1,
			Position:    1,
			Inputs:      map[string]string{"FUZZ": "admin"},
			Status:      200,
			Length:      1024,
			Words:       100,
			Lines:       20,
			DurationMs:  45,
			URL:         "http://test/admin",
			ContentType: "text/html",
			Hash:        "abc123",
			Host:        "test",
		},
		{
			ID:          2,
			Position:    2,
			Inputs:      map[string]string{"FUZZ": "login"},
			Status:      302,
			Length:      256,
			Words:       20,
			Lines:       5,
			DurationMs:  30,
			URL:         "http://test/login",
			ContentType: "text/html",
			Hash:        "abc456",
			Host:        "test",
		},
	}

	err = st.InsertResults(sessionID, batch)
	if err != nil {
		t.Fatalf("InsertResults failed: %v", err)
	}

	// 3. Query Results
	results, hasMore, err := st.GetResultsSince(sessionID, 0, 10)
	if err != nil {
		t.Fatalf("GetResultsSince failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
	if hasMore {
		t.Error("Expected hasMore to be false")
	}

	if results[0].Inputs["FUZZ"] != "admin" {
		t.Errorf("Expected input 'admin', got %s", results[0].Inputs["FUZZ"])
	}
	if results[1].Status != 302 {
		t.Errorf("Expected status 302, got %d", results[1].Status)
	}

	// 4. Update state to completed
	err = st.UpdateSessionState(sessionID, "Completed", "")
	if err != nil {
		t.Fatalf("UpdateSessionState failed: %v", err)
	}

	sess, err := st.GetSession(sessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if sess.State != "Completed" {
		t.Errorf("Expected state 'Completed', got %s", sess.State)
	}
	if sess.TotalRequests != 2 {
		t.Errorf("Expected TotalRequests 2, got %d", sess.TotalRequests)
	}
}

func TestGetResultsSinceClampingAndPagination(t *testing.T) {
	st, dir := setupTestStore(t)
	defer func() {
		st.Close()
		os.RemoveAll(dir)
	}()

	sessionID, _ := st.CreateSession(`{}`, `{}`)

	// Insert 15 items
	var batch []output.ResultDTO
	for i := 1; i <= 15; i++ {
		batch = append(batch, output.ResultDTO{
			ID:         int64(i),
			Position:   i,
			Status:     200,
			Length:     int64(i * 100),
			URL:        fmt.Sprintf("http://test/%d", i),
			DurationMs: 20,
		})
	}
	_ = st.InsertResults(sessionID, batch)

	// Test pagination: page 1 (limit 10)
	res1, hasMore1, err := st.GetResultsSince(sessionID, 0, 10)
	if err != nil {
		t.Fatalf("GetResultsSince page 1 failed: %v", err)
	}
	if len(res1) != 10 {
		t.Fatalf("Expected 10 items, got %d", len(res1))
	}
	if !hasMore1 {
		t.Error("Expected hasMore1 to be true")
	}

	// Page 2: since last ID (10)
	lastID := res1[len(res1)-1].ID
	res2, hasMore2, err := st.GetResultsSince(sessionID, lastID, 10)
	if err != nil {
		t.Fatalf("GetResultsSince page 2 failed: %v", err)
	}
	if len(res2) != 5 {
		t.Fatalf("Expected 5 items, got %d", len(res2))
	}
	if hasMore2 {
		t.Error("Expected hasMore2 to be false")
	}

	// Test boundary clamping: limit = -1 should clamp to default (1000)
	resClamp1, _, _ := st.GetResultsSince(sessionID, -5, -1)
	if len(resClamp1) != 15 {
		t.Errorf("Expected 15 items with clamped negative bounds, got %d", len(resClamp1))
	}
}

func TestConcurrentReadersWriter(t *testing.T) {
	st, dir := setupTestStore(t)
	defer func() {
		st.Close()
		os.RemoveAll(dir)
	}()

	sessionID, _ := st.CreateSession(`{}`, `{}`)

	// Start writer goroutine inserting 20 batches
	doneWriter := make(chan struct{})
	go func() {
		defer close(doneWriter)
		for b := 0; b < 20; b++ {
			batch := make([]output.ResultDTO, 10)
			for i := 0; i < 10; i++ {
				id := int64(b*10 + i + 1)
				batch[i] = output.ResultDTO{
					ID:         id,
					Position:   int(id),
					Status:     200,
					Length:     100,
					DurationMs: 15,
					URL:        fmt.Sprintf("http://concurrent/%d", id),
				}
			}
			if err := st.InsertResults(sessionID, batch); err != nil {
				t.Errorf("InsertResults error: %v", err)
				return
			}
		}
	}()

	// Start 4 concurrent readers
	var readWg sync.WaitGroup
	for r := 0; r < 4; r++ {
		readWg.Add(1)
		go func(readerID int) {
			defer readWg.Done()
			var lastID int64
			for {
				select {
				case <-doneWriter:
					// Drain remaining results
					_, _, _ = st.GetResultsSince(sessionID, lastID, 50)
					return
				default:
					res, _, err := st.GetResultsSince(sessionID, lastID, 25)
					if err != nil {
						t.Errorf("Reader %d query error: %v", readerID, err)
						return
					}
					if len(res) > 0 {
						lastID = res[len(res)-1].ID
					}
				}
			}
		}(r)
	}

	<-doneWriter
	readWg.Wait()

	// Verify total count in session
	sess, err := st.GetSession(sessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if sess.TotalRequests != 200 {
		t.Errorf("Expected TotalRequests 200, got %d", sess.TotalRequests)
	}
}

func TestGetSessions(t *testing.T) {
	st, dir := setupTestStore(t)
	defer func() {
		st.Close()
		os.RemoveAll(dir)
	}()

	s1, _ := st.CreateSession(`{"target":"s1"}`, `{"target":"s1"}`)
	s2, _ := st.CreateSession(`{"target":"s2"}`, `{"target":"s2"}`)
	_ = st.UpdateSessionState(s1, "Completed", "")

	sessions, err := st.GetSessions(10)
	if err != nil {
		t.Fatalf("GetSessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].ID != s2 {
		t.Errorf("Expected most recent session %d first, got %d", s2, sessions[0].ID)
	}
}

func BenchmarkBatchInsert(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "ffuf-store-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	st, err := NewStore(tempDir)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer st.Close()

	sessionID, _ := st.CreateSession(`{}`, `{}`)

	batchSize := 250
	sampleBatch := make([]output.ResultDTO, batchSize)
	for i := 0; i < batchSize; i++ {
		sampleBatch[i] = output.ResultDTO{
			ID:          int64(i + 1),
			Position:    i + 1,
			Inputs:      map[string]string{"FUZZ": "payload"},
			Status:      200,
			Length:      1024,
			Words:       100,
			Lines:       20,
			DurationMs:  30,
			URL:         "http://bench/target",
			ContentType: "text/html",
			Host:        "bench",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = st.InsertResults(sessionID, sampleBatch)
	}
}
