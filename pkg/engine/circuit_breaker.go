package engine

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// CircuitState represents the operational state of the Circuit Breaker.
type CircuitState int

const (
	StateClosed        CircuitState = iota // Normal operation: 100% throughput
	StateHalfOpen                          // Warning / Backoff state: throttled rate, probing
	StateOpen                              // Tripped: WAF block / rate limit detected, pausing
	StatePausedIPBlock                     // Safe Stop: IP Block confirmed, job paused with popup
	StateResuming                          // Resuming: executing final probe to healthy endpoint
	StateAborted                           // Aborted by user cancel; state preserved
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateHalfOpen:
		return "Half-Open (Throttled)"
	case StateOpen:
		return "Open (Blocked)"
	case StatePausedIPBlock:
		return "PausedIPBlock"
	case StateResuming:
		return "Resuming"
	case StateAborted:
		return "Aborted"
	default:
		return "Unknown"
	}
}

// SampleObservation captures forensic response context alongside status codes
// so the engine can distinguish IP bans from payload-triggered WAF blocks.
type SampleObservation struct {
	StatusCode  int
	IsError     bool
	BodyLength  int64
	WAFHeader   string
	PayloadHint string
	SoftBlocked bool
}

// RollingForensics summarizes blocked samples currently inside the sliding window.
type RollingForensics struct {
	BlockedCount        int
	DistinctBodyLengths int
	DominantBodyLength  int64
	DominantWAFHeader   string
	TraversalPayloadHit int
	SamplePayloads      []string
	HasSoftBlock        bool
}

// ExtractWAFSignal extracts normalized WAF/firewall block indicators from HTTP response headers.
func ExtractWAFSignal(statusCode int, headers map[string][]string) string {
	if len(headers) == 0 {
		return ""
	}
	priorityKeys := []string{
		"cf-mitigated",
		"x-blocked-by",
		"x-gate",
		"x-waf-rule",
		"x-waf-event",
		"x-arvan-waf",
		"x-sucuri-block",
		"x-denied-reason",
	}
	for _, want := range priorityKeys {
		for k, vs := range headers {
			if strings.EqualFold(k, want) && len(vs) > 0 {
				val := strings.TrimSpace(vs[0])
				if val != "" {
					return strings.ToLower(want) + ": " + val
				}
			}
		}
	}
	if statusCode == 403 || statusCode == 429 {
		for k, vs := range headers {
			if strings.EqualFold(k, "server") && len(vs) > 0 {
				srv := strings.ToLower(strings.TrimSpace(vs[0]))
				if strings.Contains(srv, "cloudflare") || strings.Contains(srv, "arvan") || strings.Contains(srv, "akamai") || strings.Contains(srv, "awselb") {
					return "server: " + strings.TrimSpace(vs[0])
				}
			}
		}
	}
	return ""
}

// IsRiskyTraversalPayload reports whether a payload string contains common traversal,
// normalization, or WAF-signature bypass sequences (e.g., .//, ./., ../, .;/).
func IsRiskyTraversalPayload(payload string) bool {
	if payload == "" {
		return false
	}
	lower := strings.ToLower(payload)
	riskyTokens := []string{
		".//", "./.", "../", "..\\", ".;/", "//", `\`, "%2e", "%2f", "%5c", "%00",
		"union select", "or 1=1", "<script", "javascript:", "${jndi:",
	}
	for _, tok := range riskyTokens {
		if strings.Contains(lower, tok) {
			return true
		}
	}
	if strings.HasSuffix(lower, "/.") || strings.Contains(lower, ";") {
		return true
	}
	return false
}

// responseSample holds the outcome of a single request in the rolling window.
type responseSample struct {
	statusCode  int
	isError     bool
	suppressed  bool
	bodyLength  int64
	wafHeader   string
	payloadHint string
	softBlocked bool
	timestamp   time.Time
}

// CircuitBreakerConfig configures the sliding window anomaly detector and circuit breaker.
type CircuitBreakerConfig struct {
	WindowSize          int           // Number of samples in rolling window (default 50)
	Threshold429Ratio   float64       // Ratio of 429 in window to trigger HalfOpen (default 0.15 = 15%)
	Threshold403Ratio   float64       // Ratio of 403 in window to trigger Open/Alert (default 0.40 = 40%)
	ThresholdErrorRatio float64       // Ratio of network errors in window to trigger HalfOpen (default 0.20 = 20%)
	Threshold5xxRatio   float64       // Ratio of 5xx errors (502/503/504) in window to trigger HalfOpen (default 0.20 = 20%)
	HardStop403Ratio    float64       // Ratio of 403 in window for definitive stop when StopOn403 is enabled (default 0.80)
	HardStop429Ratio    float64       // Ratio of 429 in window for definitive stop when StopOnAll is enabled (default 0.50)
	HardStopErrorRatio  float64       // Ratio of network errors for definitive stop when StopOnErrors is enabled (default 0.60)
	CooloffPeriod       time.Duration // Time to wait when Open before entering HalfOpen (default 2s)
	BackoffDelay        time.Duration // Delay per request during HalfOpen (default 100ms)
}

// DefaultCircuitBreakerConfig returns sensible defaults for web fuzzing resilience.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		WindowSize:          50,
		Threshold429Ratio:   0.15,
		Threshold403Ratio:   0.40,
		ThresholdErrorRatio: 0.20,
		Threshold5xxRatio:   0.20,
		HardStop403Ratio:    0.80,
		HardStop429Ratio:    0.50,
		HardStopErrorRatio:  0.60,
		CooloffPeriod:       2 * time.Second,
		BackoffDelay:        100 * time.Millisecond,
	}
}

// RollingStats represents a point-in-time snapshot of the window metrics.
type RollingStats struct {
	State        CircuitState
	TotalSamples int
	Count403     int
	Count429     int
	Count5xx     int
	CountErrors  int
	Ratio403     float64
	Ratio429     float64
	Ratio5xx     float64
	RatioErrors  float64
}

// CircuitBreaker manages sliding window metrics and adaptive state transitions.
type CircuitBreaker struct {
	mu              sync.RWMutex
	config          CircuitBreakerConfig
	samples         []responseSample
	head            int
	count           int
	state           CircuitState
	lastStateChange time.Time

	// Real-time window counters (maintained in O(1))
	count403 int
	count429 int
	count5xx int
	countErr int

	// Wordlist WAF bypass suppression (prevents re-triggering IP-ban backoff on known traversal noise)
	wordlistBypassActive  bool
	acknowledgedWAFHeader string
	acknowledgedBodyLen   int64

	// Callbacks
	onStateChange func(oldState, newState CircuitState, reason string)
}

// NewCircuitBreaker constructs a CircuitBreaker with the given configuration.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = 50
	}
	return &CircuitBreaker{
		config:          cfg,
		samples:         make([]responseSample, cfg.WindowSize),
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// SetOnStateChange installs a callback fired when the circuit state changes.
func (cb *CircuitBreaker) SetOnStateChange(fn func(oldState, newState CircuitState, reason string)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// AddOnStateChange appends a callback without replacing callbacks already registered.
// It is used when several layers need to observe the same circuit transitions.
func (cb *CircuitBreaker) AddOnStateChange(fn func(oldState, newState CircuitState, reason string)) {
	if fn == nil {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()
	previous := cb.onStateChange
	if previous == nil {
		cb.onStateChange = fn
		return
	}
	cb.onStateChange = func(oldState, newState CircuitState, reason string) {
		previous(oldState, newState, reason)
		fn(oldState, newState, reason)
	}
}

// Record records a response status code and whether it was a transport error.
func (cb *CircuitBreaker) Record(statusCode int, isError bool) {
	cb.RecordSample(SampleObservation{
		StatusCode: statusCode,
		IsError:    isError,
	})
}

// RecordSample records a response with forensic headers, body length, and payload metadata.
func (cb *CircuitBreaker) RecordSample(obs SampleObservation) {
	var transitionFn func(CircuitState, CircuitState, string)
	var oldState, newState CircuitState
	var reason string

	cb.mu.Lock()

	// 1. Evict oldest sample at head if ring buffer is full
	if cb.count == cb.config.WindowSize {
		old := cb.samples[cb.head]
		if old.isError {
			cb.countErr--
		} else {
			if old.statusCode == 403 && !old.suppressed {
				cb.count403--
			} else if old.statusCode == 429 {
				cb.count429--
			} else if old.statusCode >= 500 && old.statusCode <= 599 {
				cb.count5xx--
			}
		}
	} else {
		cb.count++
	}

	// Determine if this 403 is already-classified wordlist WAF noise
	suppressed := false
	if cb.wordlistBypassActive && !obs.IsError && obs.StatusCode == 403 && !obs.SoftBlocked {
		headerMatches := cb.acknowledgedWAFHeader == "" || strings.EqualFold(obs.WAFHeader, cb.acknowledgedWAFHeader)
		payloadOrLenMatches := IsRiskyTraversalPayload(obs.PayloadHint) || (cb.acknowledgedBodyLen > 0 && obs.BodyLength == cb.acknowledgedBodyLen)
		if headerMatches && payloadOrLenMatches {
			suppressed = true
		}
	}

	// 2. Insert new sample
	cb.samples[cb.head] = responseSample{
		statusCode:  obs.StatusCode,
		isError:     obs.IsError,
		suppressed:  suppressed,
		bodyLength:  obs.BodyLength,
		wafHeader:   obs.WAFHeader,
		payloadHint: obs.PayloadHint,
		softBlocked: obs.SoftBlocked,
		timestamp:   time.Now(),
	}
	if obs.IsError {
		cb.countErr++
	} else {
		if obs.StatusCode == 403 && !suppressed {
			cb.count403++
		} else if obs.StatusCode == 429 {
			cb.count429++
		} else if obs.StatusCode >= 500 && obs.StatusCode <= 599 {
			cb.count5xx++
		}
	}

	// 3. Advance head pointer
	cb.head = (cb.head + 1) % cb.config.WindowSize

	// 4. Update state transitions based on window ratios
	oldState, newState, reason = cb.evaluateStateTransitions()
	if newState != oldState && cb.onStateChange != nil {
		transitionFn = cb.onStateChange
	}

	cb.mu.Unlock()

	// Call notification callback outside the mutex
	if transitionFn != nil {
		transitionFn(oldState, newState, reason)
	}
}

// SetWordlistAdvisoryBypass acknowledges confirmed wordlist-specific WAF blocks so subsequent
// traversal/matched blocks do not loop into StateOpen/SafeStop while preserving detection for
// genuine IP bans or non-traversal 403 escalations.
func (cb *CircuitBreaker) SetWordlistAdvisoryBypass(active bool, wafHeader string, bodyLen int64) {
	cb.mu.Lock()
	cb.wordlistBypassActive = active
	cb.acknowledgedWAFHeader = wafHeader
	cb.acknowledgedBodyLen = bodyLen
	if active {
		recounted403 := 0
		for i := 0; i < cb.count; i++ {
			s := &cb.samples[i]
			if s.statusCode == 403 && !s.isError && !s.softBlocked {
				headerMatches := wafHeader == "" || strings.EqualFold(s.wafHeader, wafHeader)
				payloadOrLenMatches := IsRiskyTraversalPayload(s.payloadHint) || (bodyLen > 0 && s.bodyLength == bodyLen)
				if headerMatches && payloadOrLenMatches {
					s.suppressed = true
					continue
				}
			}
			if s.statusCode == 403 && !s.isError && !s.suppressed {
				recounted403++
			}
		}
		cb.count403 = recounted403
		if cb.state == StateOpen || cb.state == StateHalfOpen {
			cb.state = StateClosed
			cb.lastStateChange = time.Now()
		}
	}
	cb.mu.Unlock()
}

// ForensicsSnapshot returns an aggregated view of all recent blocked samples in the window.
func (cb *CircuitBreaker) ForensicsSnapshot() RollingForensics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	out := RollingForensics{}
	lenCounts := make(map[int64]int)
	hdrCounts := make(map[string]int)
	seenPayloads := make(map[string]struct{})

	for i := 0; i < cb.count; i++ {
		// Walk chronologically starting from oldest when ring buffer is full
		idx := i
		if cb.count == cb.config.WindowSize {
			idx = (cb.head + i) % cb.config.WindowSize
		}
		s := cb.samples[idx]
		if s.isError || s.statusCode != 403 {
			continue
		}
		out.BlockedCount++
		if s.softBlocked {
			out.HasSoftBlock = true
		}
		lenCounts[s.bodyLength]++
		if s.wafHeader != "" {
			hdrCounts[s.wafHeader]++
		}
		if IsRiskyTraversalPayload(s.payloadHint) {
			out.TraversalPayloadHit++
		}
		if s.payloadHint != "" && len(out.SamplePayloads) < 6 {
			if _, ok := seenPayloads[s.payloadHint]; !ok {
				seenPayloads[s.payloadHint] = struct{}{}
				out.SamplePayloads = append(out.SamplePayloads, s.payloadHint)
			}
		}
	}

	out.DistinctBodyLengths = len(lenCounts)
	bestLenCount := 0
	for l, c := range lenCounts {
		if c > bestLenCount {
			bestLenCount = c
			out.DominantBodyLength = l
		}
	}
	bestHdrCount := 0
	for h, c := range hdrCounts {
		if c > bestHdrCount {
			bestHdrCount = c
			out.DominantWAFHeader = h
		}
	}

	return out
}

// evaluateStateTransitions evaluates window ratios and adjusts circuit state (must be called under lock).
func (cb *CircuitBreaker) evaluateStateTransitions() (CircuitState, CircuitState, string) {
	if cb.count < 10 {
		// Rule 3 Fast-Path: If >= 5 403s with >= 50% ratio, trigger StateOpen immediately
		if (cb.state == StateClosed || cb.state == StateHalfOpen) && cb.count403 >= 5 && float64(cb.count403)/float64(cb.count) >= 0.50 {
			oldState := cb.state
			newState := StateOpen
			reason := fmt.Sprintf("WAF Forbidden block detected (%d 403s)", cb.count403)
			cb.state = newState
			cb.lastStateChange = time.Now()
			return oldState, newState, reason
		}
		// Minimum samples before making other decisions
		return cb.state, cb.state, ""
	}

	r429 := float64(cb.count429) / float64(cb.count)
	r403 := float64(cb.count403) / float64(cb.count)
	rErr := float64(cb.countErr) / float64(cb.count)
	r5xx := float64(cb.count5xx) / float64(cb.count)

	oldState := cb.state
	newState := oldState
	var reason string

	switch cb.state {
	case StateClosed:
		if r429 >= cb.config.Threshold429Ratio {
			newState = StateHalfOpen
			reason = fmt.Sprintf("WAF Rate Limiting detected (%.1f%% 429s)", r429*100)
		} else if r403 >= cb.config.Threshold403Ratio {
			newState = StateOpen
			reason = fmt.Sprintf("WAF Forbidden block detected (%.1f%% 403s)", r403*100)
		} else if rErr >= cb.config.ThresholdErrorRatio {
			newState = StateHalfOpen
			reason = fmt.Sprintf("Elevated network errors detected (%.1f%% errors)", rErr*100)
		} else if r5xx >= cb.config.Threshold5xxRatio {
			newState = StateHalfOpen
			reason = fmt.Sprintf("Server gateway/5xx errors detected (%.1f%% 5xx)", r5xx*100)
		}

	case StateHalfOpen:
		if r403 >= cb.config.Threshold403Ratio {
			newState = StateOpen
			reason = fmt.Sprintf("Escalated to WAF Forbidden block (%.1f%% 403s)", r403*100)
		} else if r429 < 0.05 && r403 < 0.10 && rErr < 0.10 && r5xx < 0.10 {
			newState = StateClosed
			reason = "Target recovered: window error rates normalized"
		}

	case StateOpen:
		if time.Since(cb.lastStateChange) >= cb.config.CooloffPeriod {
			newState = StateHalfOpen
			reason = fmt.Sprintf("Cooloff period elapsed (%v); probing in Half-Open state", cb.config.CooloffPeriod)
		}
	}

	if newState != oldState {
		cb.state = newState
		cb.lastStateChange = time.Now()
	}

	return oldState, newState, reason
}

// Stats returns a copy of current rolling metrics.
func (cb *CircuitBreaker) Stats() RollingStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var r403, r429, r5xx, rErr float64
	if cb.count > 0 {
		r403 = float64(cb.count403) / float64(cb.count)
		r429 = float64(cb.count429) / float64(cb.count)
		r5xx = float64(cb.count5xx) / float64(cb.count)
		rErr = float64(cb.countErr) / float64(cb.count)
	}

	return RollingStats{
		State:        cb.state,
		TotalSamples: cb.count,
		Count403:     cb.count403,
		Count429:     cb.count429,
		Count5xx:     cb.count5xx,
		CountErrors:  cb.countErr,
		Ratio403:     r403,
		Ratio429:     r429,
		Ratio5xx:     r5xx,
		RatioErrors:  rErr,
	}
}

// State returns the current operational state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// SetState explicitly transitions the circuit breaker state with a notification reason.
func (cb *CircuitBreaker) SetState(newState CircuitState, reason string) {
	cb.mu.Lock()
	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()
	var transitionFn func(CircuitState, CircuitState, string)
	if cb.onStateChange != nil && oldState != newState {
		transitionFn = cb.onStateChange
	}
	cb.mu.Unlock()
	if transitionFn != nil {
		transitionFn(oldState, newState, reason)
	}
}

// CheckBackoff determines whether the dispatch loop should throttle or sleep.
func (cb *CircuitBreaker) CheckBackoff() (shouldBackoff bool, delay time.Duration) {
	var transitionFn func(CircuitState, CircuitState, string)
	var oldState, newState CircuitState
	var reason string

	cb.mu.Lock()

	// If safely paused or aborted due to IPBlock, keep pausing completely
	if cb.state == StatePausedIPBlock || cb.state == StateAborted {
		cb.mu.Unlock()
		return true, 500 * time.Millisecond
	}

	// Auto-check if Open state has cooled off
	if cb.state == StateOpen {
		if time.Since(cb.lastStateChange) >= cb.config.CooloffPeriod {
			oldState = cb.state
			cb.state = StateHalfOpen
			cb.lastStateChange = time.Now()
			newState = StateHalfOpen
			reason = "Cooloff period elapsed; resuming in Half-Open"
			if cb.onStateChange != nil {
				transitionFn = cb.onStateChange
			}
			cb.mu.Unlock()
			if transitionFn != nil {
				transitionFn(oldState, newState, reason)
			}
			return true, cb.config.BackoffDelay
		}
		remaining := cb.config.CooloffPeriod - time.Since(cb.lastStateChange)
		cb.mu.Unlock()
		return true, remaining
	}

	if cb.state == StateHalfOpen {
		delay = cb.config.BackoffDelay
		cb.mu.Unlock()
		return true, delay
	}

	cb.mu.Unlock()
	return false, 0
}

// ShouldStop evaluates whether hard-stop conditions are met based on rolling metrics.
func (cb *CircuitBreaker) ShouldStop(stopOn403, stopOnAll, stopOnErrors bool) (bool, string) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.count < 20 {
		// Need at least 20 samples in window
		return false, ""
	}

	r403 := float64(cb.count403) / float64(cb.count)
	r429 := float64(cb.count429) / float64(cb.count)
	rErr := float64(cb.countErr) / float64(cb.count)

	if stopOn403 || stopOnAll {
		if r403 >= cb.config.HardStop403Ratio {
			return true, fmt.Sprintf("WAF block confirmed: %.1f%% of recent %d requests returned 403 Forbidden.", r403*100, cb.count)
		}
	}

	if stopOnAll {
		if r429 >= cb.config.HardStop429Ratio {
			return true, fmt.Sprintf("WAF rate limit unrecoverable: %.1f%% of recent %d requests returned 429 Too Many Requests.", r429*100, cb.count)
		}
	}

	if stopOnErrors || stopOnAll {
		if rErr >= cb.config.HardStopErrorRatio {
			return true, fmt.Sprintf("Server degraded: %.1f%% of recent %d requests failed with transport errors.", rErr*100, cb.count)
		}
	}

	return false, ""
}
