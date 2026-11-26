package engine

import (
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_DefaultClosed(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	stats := cb.Stats()

	if stats.State != StateClosed {
		t.Fatalf("expected initial state to be StateClosed, got %v", stats.State)
	}
	if stats.TotalSamples != 0 || stats.Count403 != 0 || stats.Count429 != 0 || stats.CountErrors != 0 {
		t.Fatalf("expected all counters to be 0, got %+v", stats)
	}

	backoff, delay := cb.CheckBackoff()
	if backoff || delay != 0 {
		t.Fatalf("expected no backoff in StateClosed, got backoff=%v delay=%v", backoff, delay)
	}
}

func TestCircuitBreaker_SlidingWindowEviction(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.WindowSize = 10
	cb := NewCircuitBreaker(cfg)

	// Record 10 403 responses
	for i := 0; i < 10; i++ {
		cb.Record(403, false)
	}

	stats := cb.Stats()
	if stats.TotalSamples != 10 || stats.Count403 != 10 {
		t.Fatalf("expected 10 403 samples, got %+v", stats)
	}

	// Now record 10 200 OK responses -> should completely evict all 403s!
	for i := 0; i < 10; i++ {
		cb.Record(200, false)
	}

	stats = cb.Stats()
	if stats.TotalSamples != 10 {
		t.Fatalf("expected TotalSamples to stay at window size 10, got %d", stats.TotalSamples)
	}
	if stats.Count403 != 0 {
		t.Fatalf("expected Count403 to be 0 after eviction, got %d", stats.Count403)
	}
	if stats.Ratio403 != 0 {
		t.Fatalf("expected Ratio403 to be 0, got %f", stats.Ratio403)
	}
}

func TestCircuitBreaker_StateTransitions_429(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.WindowSize = 20
	cfg.Threshold429Ratio = 0.20 // 20%
	cb := NewCircuitBreaker(cfg)

	var mu sync.Mutex
	var lastTransition string
	cb.SetOnStateChange(func(oldState, newState CircuitState, reason string) {
		mu.Lock()
		lastTransition = reason
		mu.Unlock()
	})

	// Record 15 200 OKs and 5 429s (5/20 = 25% > 20%)
	for i := 0; i < 15; i++ {
		cb.Record(200, false)
	}
	for i := 0; i < 5; i++ {
		cb.Record(429, false)
	}

	stats := cb.Stats()
	if stats.State != StateHalfOpen {
		t.Fatalf("expected state to transition to StateHalfOpen, got %v", stats.State)
	}

	backoff, delay := cb.CheckBackoff()
	if !backoff || delay <= 0 {
		t.Fatalf("expected backoff in StateHalfOpen, got backoff=%v delay=%v", backoff, delay)
	}

	// Now record 20 200 OK responses -> target recovered!
	for i := 0; i < 20; i++ {
		cb.Record(200, false)
	}

	stats = cb.Stats()
	if stats.State != StateClosed {
		t.Fatalf("expected state to recover to StateClosed, got %v", stats.State)
	}

	mu.Lock()
	_ = lastTransition
	mu.Unlock()
}

func TestCircuitBreaker_StateTransitions_403(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.WindowSize = 20
	cfg.Threshold403Ratio = 0.40
	cfg.CooloffPeriod = 50 * time.Millisecond
	cb := NewCircuitBreaker(cfg)

	// Record 10 200s and 10 403s (10/20 = 50% > 40%)
	for i := 0; i < 10; i++ {
		cb.Record(200, false)
	}
	for i := 0; i < 10; i++ {
		cb.Record(403, false)
	}

	stats := cb.Stats()
	if stats.State != StateOpen {
		t.Fatalf("expected state to transition to StateOpen, got %v", stats.State)
	}

	// While Open within cooloff period, CheckBackoff should report true and remaining delay
	backoff, delay := cb.CheckBackoff()
	if !backoff || delay <= 0 {
		t.Fatalf("expected backoff while Open, got backoff=%v delay=%v", backoff, delay)
	}

	// Sleep past cooloff
	time.Sleep(60 * time.Millisecond)

	// CheckBackoff should now transition to HalfOpen
	backoff, _ = cb.CheckBackoff()
	if !backoff {
		t.Fatalf("expected backoff in HalfOpen, got false")
	}

	stats = cb.Stats()
	if stats.State != StateHalfOpen {
		t.Fatalf("expected state to be StateHalfOpen after cooloff, got %v", stats.State)
	}
}

func TestCircuitBreaker_ShouldStop(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.WindowSize = 30
	cfg.HardStop403Ratio = 0.80
	cb := NewCircuitBreaker(cfg)

	// Before 20 samples, ShouldStop should be false
	for i := 0; i < 15; i++ {
		cb.Record(403, false)
	}
	stop, _ := cb.ShouldStop(true, false, false)
	if stop {
		t.Fatalf("expected ShouldStop=false before min samples, got true")
	}

	// Record 15 more 403s (Total 30/30 = 100% > 80%)
	for i := 0; i < 15; i++ {
		cb.Record(403, false)
	}

	stop, reason := cb.ShouldStop(true, false, false)
	if !stop {
		t.Fatalf("expected ShouldStop=true with 100%% 403s, got false")
	}
	if reason == "" {
		t.Fatalf("expected non-empty reason on stop")
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	var wg sync.WaitGroup

	// Run 20 writer goroutines and 10 reader goroutines concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				status := 200
				isErr := false
				if j%5 == 0 {
					status = 403
				} else if j%7 == 0 {
					status = 429
				} else if j%11 == 0 {
					isErr = true
				}
				cb.Record(status, isErr)
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = cb.Stats()
				_, _ = cb.CheckBackoff()
				_, _ = cb.ShouldStop(true, true, true)
			}
		}()
	}

	wg.Wait()

	stats := cb.Stats()
	if stats.TotalSamples != 50 {
		t.Fatalf("expected total samples to be window size 50, got %d", stats.TotalSamples)
	}
}

func TestCircuitBreaker_StateTransitions_5xx(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.Threshold5xxRatio = 0.20 // 20% threshold
	cb := NewCircuitBreaker(cfg)

	var transitionSample int
	cb.SetOnStateChange(func(oldState, newState CircuitState, reason string) {
	})

	// Send 8 200s and 2 502s (2/10 = 20% 5xx)
	for i := 1; i <= 10; i++ {
		if i <= 8 {
			cb.Record(200, false)
		} else {
			cb.Record(502, false) // 502 Bad Gateway
		}
		if cb.Stats().State == StateHalfOpen && transitionSample == 0 {
			transitionSample = i
		}
	}

	stats := cb.Stats()
	if stats.State != StateHalfOpen {
		t.Fatalf("expected StateHalfOpen with 20%% 5xx errors, got %v", stats.State)
	}
	if transitionSample > 10 || transitionSample == 0 {
		t.Fatalf("expected transition at sample <= 10, got %d", transitionSample)
	}
	t.Logf("CONFIRMED: Triggered state: Half-Open at request sample: %d (<= 10)", transitionSample)
}

func TestCircuitBreaker_AddOnStateChangePreservesExistingCallbacks(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	var first, second int
	cb.SetOnStateChange(func(CircuitState, CircuitState, string) { first++ })
	cb.AddOnStateChange(func(CircuitState, CircuitState, string) { second++ })

	for i := 0; i < 8; i++ {
		cb.Record(403, false)
	}

	if first != 1 || second != 1 {
		t.Fatalf("callbacks were not both invoked: first=%d second=%d", first, second)
	}
}
