package output

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func mockResponse(url string, status int64, length int64) fuzznexus.Response {
	return fuzznexus.Response{
		StatusCode:    status,
		ContentLength: length,
		ContentWords:  10,
		ContentLines:  5,
		Duration:      50 * time.Millisecond,
		Request: &fuzznexus.Request{
			Url:      url,
			Host:     "localhost",
			Position: 1,
			Input: map[string][]byte{
				"FUZZ":   []byte("test"),
				"FNHASH": []byte("hash123"),
			},
		},
	}
}

// Q-01 & Q-02: Test that Finalize can run concurrently with ticker and Result without deadlock or closed channel panic.
func TestFinalizeConcurrency(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var committedCount int64
	dbWriter := func(sessionID int64, batch []ResultDTO) error {
		atomic.AddInt64(&committedCount, int64(len(batch)))
		return nil
	}

	provider := NewGUIOutputProvider(
		ctx,
		1,
		slog.Default(),
		dbWriter,
		func(e CommittedEvent) {},
		func(e FinishedEvent) {},
		func(c, m string) {},
		func(err error) {},
	)

	var wg sync.WaitGroup
	// Concurrently pump results
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				provider.Result(mockResponse(fmt.Sprintf("http://target/%d/%d", workerID, j), 200, 100))
			}
		}(i)
	}

	// Trigger Finalize concurrently
	time.Sleep(10 * time.Millisecond)
	finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer finalizeCancel()

	err := provider.FinalizeWithContext(finalizeCtx)
	if err != nil {
		t.Fatalf("Finalize returned error: %v", err)
	}

	wg.Wait()

	// Calling Finalize a second time must be completely safe and idempotent
	err2 := provider.FinalizeWithContext(finalizeCtx)
	if err2 != nil {
		t.Fatalf("Second Finalize call returned error: %v", err2)
	}
}

// Q-03: Test that persistent DB failure drains writeCh, triggers onFatal, and Finalize does not hang.
func TestDBFailureHaltsScan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var fatalCalled atomic.Bool
	var fatalErr error
	var fatalMu sync.Mutex

	dbWriterAlwaysFails := func(sessionID int64, batch []ResultDTO) error {
		return errors.New("simulated fatal disk I/O error")
	}

	provider := NewGUIOutputProvider(
		ctx,
		2,
		slog.Default(),
		dbWriterAlwaysFails,
		func(e CommittedEvent) {},
		func(e FinishedEvent) {},
		func(c, m string) {},
		func(err error) {
			fatalMu.Lock()
			fatalErr = err
			fatalMu.Unlock()
			fatalCalled.Store(true)
		},
	)

	// Pump items
	for i := 0; i < 20; i++ {
		provider.Result(mockResponse(fmt.Sprintf("http://target/%d", i), 200, 100))
	}
	provider.FlushTick()

	// Wait for retries to exhaust and onFatal to fire
	deadline := time.Now().Add(2 * time.Second)
	for !fatalCalled.Load() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if !fatalCalled.Load() {
		t.Fatal("Expected onFatal callback to be called upon persistent DB failure")
	}

	fatalMu.Lock()
	if fatalErr == nil {
		t.Error("Expected non-nil fatalErr")
	}
	fatalMu.Unlock()

	finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer finalizeCancel()

	// Finalize must return cleanly without hanging
	_ = provider.FinalizeWithContext(finalizeCtx)
}

func TestDBFailureFatalCallbackCanFinalizeProvider(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	finalized := make(chan error, 1)
	var provider *GUIOutputProvider
	provider = NewGUIOutputProvider(
		ctx,
		3,
		slog.Default(),
		func(int64, []ResultDTO) error { return errors.New("simulated persistent sqlite failure") },
		func(CommittedEvent) {},
		func(FinishedEvent) {},
		func(string, string) {},
		func(error) {
			finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), time.Second)
			defer finalizeCancel()
			finalized <- provider.FinalizeWithContext(finalizeCtx)
		},
	)

	provider.Result(mockResponse("http://target/fatal", 200, 100))
	provider.FlushTick()

	select {
	case err := <-finalized:
		if err == nil || err.Error() != "simulated persistent sqlite failure" {
			t.Fatalf("FinalizeWithContext() error = %v, want persistent writer error", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fatal callback could not finalize provider without blocking")
	}
}

// Q-04: Test that slow writer applies bounded backpressure on Result() and IDs are contiguous.
func TestBackpressureBufferBound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var committedCount int64
	var committedIDs []int64
	var idMu sync.Mutex

	dbWriterSlow := func(sessionID int64, batch []ResultDTO) error {
		time.Sleep(10 * time.Millisecond)
		idMu.Lock()
		for _, item := range batch {
			committedIDs = append(committedIDs, item.ID)
		}
		idMu.Unlock()
		atomic.AddInt64(&committedCount, int64(len(batch)))
		return nil
	}

	provider := NewGUIOutputProvider(
		ctx,
		3,
		slog.Default(),
		dbWriterSlow,
		func(e CommittedEvent) {},
		func(e FinishedEvent) {},
		func(c, m string) {},
		func(err error) {},
	)
	// Set small buffer limit to test backpressure threshold
	provider.SetMaxBuffered(50)

	var wg sync.WaitGroup
	totalItems := 500
	numWorkers := 5

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			for i := 0; i < totalItems/numWorkers; i++ {
				provider.Result(mockResponse(fmt.Sprintf("http://target/%d/%d", wID, i), 200, int64(i)))
			}
		}(w)
	}

	wg.Wait()

	finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer finalizeCancel()

	err := provider.FinalizeWithContext(finalizeCtx)
	if err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	idMu.Lock()
	defer idMu.Unlock()

	if len(committedIDs) != totalItems {
		t.Fatalf("Expected %d committed IDs, got %d", totalItems, len(committedIDs))
	}

	// Verify IDs are strictly 1..totalItems
	seen := make(map[int64]bool)
	for _, id := range committedIDs {
		if id < 1 || id > int64(totalItems) {
			t.Errorf("ID out of bounds: %d", id)
		}
		if seen[id] {
			t.Errorf("Duplicate ID observed: %d", id)
		}
		seen[id] = true
	}
}

// Q-05: Test trailing-edge event delivery.
func TestTrailingEdgeCoalescedEmit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var commitEvents []CommittedEvent
	var eventMu sync.Mutex

	dbWriter := func(sessionID int64, batch []ResultDTO) error {
		return nil
	}

	provider := NewGUIOutputProvider(
		ctx,
		4,
		slog.Default(),
		dbWriter,
		func(e CommittedEvent) {
			eventMu.Lock()
			commitEvents = append(commitEvents, e)
			eventMu.Unlock()
		},
		func(e FinishedEvent) {},
		func(c, m string) {},
		func(err error) {},
	)

	provider.Result(mockResponse("http://target/single", 200, 100))
	provider.FlushTick()

	finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer finalizeCancel()

	err := provider.FinalizeWithContext(finalizeCtx)
	if err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	eventMu.Lock()
	defer eventMu.Unlock()

	if len(commitEvents) == 0 {
		t.Fatal("Expected trailing-edge CommittedEvent, got none")
	}

	lastEvent := commitEvents[len(commitEvents)-1]
	if lastEvent.LatestID != 1 {
		t.Errorf("Expected LatestID=1, got %d", lastEvent.LatestID)
	}
}
