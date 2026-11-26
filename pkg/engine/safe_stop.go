package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PausableJob defines the minimal interface needed on a Job for safe pausing and resuming.
type PausableJob interface {
	Pause()
	Resume()
	Stop()
}

// SafeStopWorkflow manages the Phase 5 Safe Stop State Machine and UI events.
type SafeStopWorkflow struct {
	mu            sync.RWMutex
	circuit       *CircuitBreaker
	banDetector   *BanDetector
	ipProvider    IPProvider
	job           PausableJob
	stateFilePath string
	healthURL     string

	lastIPHash        string
	lastRawIP         string
	newRawIP          string
	currentIndex      int
	sessionID         int64
	targetURL         string
	method            string
	wordlist          string
	cookies           string
	headers           map[string]string
	browserProfile    string
	tlsProfile        string
	isLocalTarget     bool
	allowNon200Origin bool

	// Event Bus Subscriptions
	OnIPBlockSuspected func(reason string, ipHash string)
	OnStateChange      func(oldState, newState CircuitState, reason string)
	OnPopupShown       func(title, message string, buttons []string)
	OnSafeStopPrompt   func(title, message, ipHash, rawIP, reason string, buttons []string)
	OnContinueEnabled  func()

	// Internal state
	continueAllowed bool
}

// NewSafeStopWorkflow constructs a SafeStopWorkflow.
func NewSafeStopWorkflow(
	cb *CircuitBreaker,
	bd *BanDetector,
	provider IPProvider,
	job PausableJob,
	healthURL string,
	stateFilePath string,
) *SafeStopWorkflow {
	if stateFilePath == "" {
		stateFilePath = GetDefaultResumeFilePath()
	}
	wf := &SafeStopWorkflow{
		circuit:       cb,
		banDetector:   bd,
		ipProvider:    provider,
		job:           job,
		healthURL:     healthURL,
		stateFilePath: stateFilePath,
		headers:       make(map[string]string),
	}

	// Chain circuit state changes
	cb.AddOnStateChange(func(oldState, newState CircuitState, reason string) {
		if wf.OnStateChange != nil {
			wf.OnStateChange(oldState, newState, reason)
		}
	})

	return wf
}

// SetCurrentIndex updates the latest fuzzing index completed.
func (wf *SafeStopWorkflow) SetCurrentIndex(idx int) {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	wf.currentIndex = idx
}

// LastIPHash returns the last recorded IP hash.
func (wf *SafeStopWorkflow) LastIPHash() string {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	return wf.lastIPHash
}

// LastRawIP returns the last recorded raw IP address.
func (wf *SafeStopWorkflow) LastRawIP() string {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	return wf.lastRawIP
}

// NewRawIP returns the newly verified raw IP address after TestConnection.
func (wf *SafeStopWorkflow) NewRawIP() string {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	return wf.newRawIP
}

// SetSessionContext stores current cookies and headers for encrypted persistence.
func (wf *SafeStopWorkflow) SetSessionContext(cookies string, headers map[string]string) {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	wf.cookies = cookies
	wf.headers = headers
}

// SetBrowserProfile configures the active browser persona used for health and probe requests.
func (wf *SafeStopWorkflow) SetBrowserProfile(profileID string) {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	wf.browserProfile = profileID
}

// SetTLSProfile configures the TLS fingerprint used by canary and health probes.
func (wf *SafeStopWorkflow) SetTLSProfile(tlsProfile string) {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	wf.tlsProfile = tlsProfile
}

// SetScanMetadata records target, method, wordlist, and session ID for cross-restart resume.
func (wf *SafeStopWorkflow) SetScanMetadata(sessionID int64, targetURL, method, wordlist string) {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	wf.sessionID = sessionID
	wf.targetURL = targetURL
	wf.method = method
	wf.wordlist = wordlist
}

// SetTargetProbeFlags adjusts whether local /health rules or remote non-200 origin rules apply.
func (wf *SafeStopWorkflow) SetTargetProbeFlags(isLocal bool, allowNon200 bool) {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	wf.isLocalTarget = isLocal
	wf.allowNon200Origin = allowNon200
}

// State returns current circuit state.
func (wf *SafeStopWorkflow) State() CircuitState {
	return wf.circuit.State()
}

func (wf *SafeStopWorkflow) probeConfigLocked() CanaryProbeConfig {
	hdrCopy := make(map[string]string, len(wf.headers))
	for k, v := range wf.headers {
		hdrCopy[k] = v
	}
	return CanaryProbeConfig{
		RootCanaryURL:     wf.healthURL,
		IsLocalTarget:     wf.isLocalTarget,
		AllowNon200Origin: wf.allowNon200Origin,
		Headers:           hdrCopy,
		Cookies:           wf.cookies,
		BrowserProfile:    wf.browserProfile,
		TLSProfile:        wf.tlsProfile,
		Timeout:           5 * time.Second,
	}
}

// CheckAndHandleBlock evaluates rolling stats and performs the safe stop if IPBlock is confirmed.
// Rule: Transition to PausedIPBlock is ONLY allowed from Open.
func (wf *SafeStopWorkflow) CheckAndHandleBlock(ctx context.Context, hasRetryAfter bool) (bool, string) {
	stats := wf.circuit.Stats()
	category := wf.banDetector.ClassifyMetrics(stats, hasRetryAfter)

	if category != BanIPBlock {
		return false, ""
	}

	// Rule: Transition to PausedIPBlock ONLY permitted from Open
	if wf.circuit.State() != StateOpen {
		wf.circuit.SetState(StateOpen, "Preconditions for IPBlock met; entering Open state")
	}

	// Rule Condition (3): cooloff probe check
	// Sleep for the cooloff duration or until context cancelled
	cooloff := wf.banDetector.CooloffDuration()
	select {
	case <-ctx.Done():
		return false, "context cancelled during cooloff"
	case <-time.After(cooloff):
	}

	isBlocked, reason := wf.banDetector.VerifyCondition3(ctx)
	if !isBlocked {
		// Temporary 403 cleared after cooloff, return to HalfOpen, DO NOT show popup
		wf.circuit.SetState(StateHalfOpen, "Probe succeeded after cooloff; resuming in Half-Open")
		return false, ""
	}

	// Condition (3) confirmed! Enter StatePausedIPBlock and trigger Safe Stop Popup
	return wf.EnterPausedIPBlock(ctx, reason)
}

// EnterPausedIPBlock explicitly enters StatePausedIPBlock, saves state, and halts target requests.
func (wf *SafeStopWorkflow) EnterPausedIPBlock(ctx context.Context, reason string) (bool, string) {
	wf.mu.Lock()
	defer wf.mu.Unlock()

	// Strict Rule: Transition to PausedIPBlock ONLY allowed from StateOpen
	if wf.circuit.State() != StateOpen {
		wf.circuit.SetState(StateOpen, "Transitioning to Open prior to PausedIPBlock")
	}

	// Fetch current IP hash and raw IP
	ipHash := "unknown"
	rawIP := ""
	if wf.ipProvider != nil {
		if r, h, err := wf.ipProvider.CurrentIP(ctx); err == nil {
			rawIP = r
			ipHash = h
		}
	}
	wf.lastIPHash = ipHash
	wf.lastRawIP = rawIP

	// 1. Transition state
	wf.circuit.SetState(StatePausedIPBlock, fmt.Sprintf("IP block confirmed: %s", reason))

	// 2. Pause job (guaranteeing 0 requests to target)
	if wf.job != nil {
		wf.job.Pause()
	}

	// 3. Persist resume state with encrypted cookies
	encCookies, nonce, _ := EncryptCookies(wf.cookies)
	resumeState := ResumeState{
		SessionID:        wf.sessionID,
		TargetURL:        wf.targetURL,
		Method:           wf.method,
		Wordlist:         wf.wordlist,
		BrowserProfile:   wf.browserProfile,
		TLSProfile:       wf.tlsProfile,
		CurrentIndex:     wf.currentIndex,
		IPHash:           ipHash,
		Headers:          wf.headers,
		EncryptedCookies: encCookies,
		CookiesNonce:     nonce,
		Stats:            wf.circuit.Stats(),
		Timestamp:        time.Now(),
	}
	_ = SaveResumeState(wf.stateFilePath, resumeState)

	// 4. Trigger UI Events
	if wf.OnIPBlockSuspected != nil {
		wf.OnIPBlockSuspected(reason, ipHash)
	}

	title := "Your IP Address Appears to Be Blocked"
	message := "The target is returning 403 Forbidden responses. The scan has been safely paused.\n" +
		"Please change your IP using a VPN or proxy, then click Test Connection."
	buttons := []string{"Test Connection", "Cancel", "Resume anyway"}

	if wf.OnPopupShown != nil {
		wf.OnPopupShown(title, message, buttons)
	}

	if wf.OnSafeStopPrompt != nil {
		wf.OnSafeStopPrompt(title, message, ipHash, rawIP, reason, buttons)
	}

	return true, reason
}

// TestConnection executes a single probe to verify connectivity after user changes IP.
func (wf *SafeStopWorkflow) TestConnection(ctx context.Context) (success bool, newIPHash string, err error) {
	wf.mu.Lock()
	defer wf.mu.Unlock()

	if wf.circuit.State() != StatePausedIPBlock {
		return false, "", fmt.Errorf("TestConnection only allowed in StatePausedIPBlock")
	}

	// 1. ALWAYS query IP Provider first so the new IP address and hash are captured
	newRawIP := ""
	if wf.ipProvider != nil {
		if r, h, err := wf.ipProvider.CurrentIP(ctx); err == nil {
			newRawIP = r
			newIPHash = h
		} else if h, err := wf.ipProvider.CurrentIPHash(ctx); err == nil {
			newIPHash = h
		}
	}
	wf.newRawIP = newRawIP

	// 2. Probe endpoint (using health URL / origin canary if configured)
	probeURL := wf.healthURL
	if probeURL != "" {
		outcome := ProbeHTTPTarget(ctx, probeURL, wf.probeConfigLocked(), true)
		if outcome.Error != nil {
			return false, newIPHash, fmt.Errorf("probe failed: %w", outcome.Error)
		}
		if outcome.IsBlocked {
			if outcome.WAFHeader != "" {
				return false, newIPHash, fmt.Errorf("target is still blocking traffic (%d %s)", outcome.StatusCode, outcome.WAFHeader)
			}
			return false, newIPHash, fmt.Errorf("target is still returning %d Forbidden", outcome.StatusCode)
		}
		if !outcome.Healthy {
			return false, newIPHash, fmt.Errorf("probe failed with status: %d", outcome.StatusCode)
		}
	}

	// Check if IP changed
	if newIPHash != "" && wf.lastIPHash != "" && newIPHash == wf.lastIPHash {
		// Subnet/same IP warning
		_ = fmt.Sprintf("IP has not changed (hash: %s). Please check your VPN/proxy connection.", newIPHash)
	}

	wf.continueAllowed = true
	if wf.OnContinueEnabled != nil {
		wf.OnContinueEnabled()
	}

	return true, newIPHash, nil
}

// Continue performs the final probe and resumes scanning from the exact paused index.
func (wf *SafeStopWorkflow) Continue(ctx context.Context) (success bool, err error) {
	wf.mu.Lock()
	defer wf.mu.Unlock()

	if wf.circuit.State() != StatePausedIPBlock {
		return false, fmt.Errorf("Continue only allowed from StatePausedIPBlock")
	}

	// 1. Move to StateResuming
	wf.circuit.SetState(StateResuming, "Resuming execution: performing final probe")

	// 2. Exactly one final probe to healthy endpoint (not main scan target)
	probeURL := wf.healthURL
	if probeURL != "" {
		outcome := ProbeHTTPTarget(ctx, probeURL, wf.probeConfigLocked(), true)
		if outcome.Error != nil || outcome.IsBlocked || !outcome.Healthy {
			// Final probe failed -> back to PausedIPBlock
			wf.circuit.SetState(StatePausedIPBlock, "New IP is still blocked")
			if outcome.Error != nil {
				return false, outcome.Error
			}
			return false, fmt.Errorf("new IP is still blocked (%d)", outcome.StatusCode)
		}
	}

	// 3. Success! Move to Closed
	wf.circuit.SetState(StateClosed, "Connection verified; resuming fuzzing from paused index")

	// 4. Delete resume state file on successful continuation
	_ = DeleteResumeState(wf.stateFilePath)

	// 5. Resume job
	if wf.job != nil {
		wf.job.Resume()
	}

	wf.continueAllowed = false
	return true, nil
}

// Cancel aborts execution, preserves state file, and halts the job.
func (wf *SafeStopWorkflow) Cancel() error {
	wf.mu.Lock()
	defer wf.mu.Unlock()

	wf.circuit.SetState(StateAborted, "Scan cancelled by user; state preserved in resume_state.json")
	if wf.job != nil {
		wf.job.Stop()
	}
	return nil
}

// ResumeAnyway forces resumption into Open state with a warning.
func (wf *SafeStopWorkflow) ResumeAnyway(ctx context.Context) error {
	wf.mu.Lock()
	defer wf.mu.Unlock()

	wf.circuit.SetState(StateOpen, "User chose 'Resume anyway'; running in Open state under WAF block risk")
	if wf.job != nil {
		wf.job.Resume()
	}
	return nil
}
