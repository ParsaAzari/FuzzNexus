package scan

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"fuzznexus-desktop/internal/blobs"
	"fuzznexus-desktop/internal/output"
	"fuzznexus-desktop/internal/store"

	"github.com/parsaazari/fuzznexus/pkg/assembly"
	"github.com/parsaazari/fuzznexus/pkg/engine"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

type State string

const (
	StateIdle      State = "Idle"
	StateRunning   State = "Running"
	StatePaused    State = "Paused"
	StateStopping  State = "Stopping"
	StateCompleted State = "Completed"
	StateFailed    State = "Failed"
	StateStopped   State = "Stopped"
)

var (
	ErrSessionNotIdle    = errors.New("scan: session is not in Idle state")
	ErrSessionNotRunning = errors.New("scan: session is not in Running state")
	ErrSessionNotPaused  = errors.New("scan: session is not in Paused state")
	ErrSessionTerminated = errors.New("scan: session has already terminated")
)

// SafeStopPromptDTO contains parameters needed to display the Safe Stop IP Block modal.
type SafeStopPromptDTO struct {
	SessionID     int64    `json:"session_id"`
	Title         string   `json:"title"`
	Message       string   `json:"message"`
	IPHash        string   `json:"ip_hash"`
	RawIP         string   `json:"raw_ip,omitempty"`
	Reason        string   `json:"reason"`
	CanarySummary string   `json:"canary_summary,omitempty"`
	Buttons       []string `json:"buttons"`
}

// AdvisoryAlertDTO notifies the UI when non-IP-ban WAF triggers or wordlist anomalies are detected.
type AdvisoryAlertDTO struct {
	SessionID        int64                 `json:"session_id"`
	Verdict          string                `json:"verdict"`
	Confidence       int                   `json:"confidence"`
	Title            string                `json:"title"`
	Summary          string                `json:"summary"`
	EvidenceHeader   string                `json:"evidence_header,omitempty"`
	FixedBodyLength  int64                 `json:"fixed_body_length,omitempty"`
	SamplePayloads   []string              `json:"sample_payloads,omitempty"`
	Canary           engine.CanaryEvidence `json:"canary"`
	SuggestedActions []string              `json:"suggested_actions,omitempty"`
}

type probeTargets struct {
	RootCanaryURL     string
	BenignTemplateURL string
	CanProbeBenign    bool
	IsLocalTarget     bool
	AllowNon200Origin bool
}

// SafeStopTestResultDTO returns the status of an IP change connection test.
type SafeStopTestResultDTO struct {
	Success   bool   `json:"success"`
	NewIPHash string `json:"new_ip_hash"`
	NewRawIP  string `json:"new_raw_ip,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ProgressDTO contains live metrics emitted to the frontend during a scan.
type ProgressDTO struct {
	SessionID     int64     `json:"session_id"`
	StartedAt     time.Time `json:"started_at"`
	ReqCount      int       `json:"req_count"`
	ReqTotal      int       `json:"req_total"`
	ReqSec        int64     `json:"req_sec"`
	QueuePos      int       `json:"queue_pos"`
	QueueTotal    int       `json:"queue_total"`
	ErrorCount    int       `json:"error_count"`
	Percentage    float64   `json:"percentage"`
	BytesSent     int64     `json:"bytes_sent"`
	BytesRecv     int64     `json:"bytes_recv"`
	CircuitState  string    `json:"circuit_state,omitempty"`
	CircuitReason string    `json:"circuit_reason,omitempty"`
}

// NetworkErrorDTO represents a recorded network diagnostic error during scanning.
type NetworkErrorDTO struct {
	Index     int       `json:"index"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Session manages a single scan lifecycle, orchestrating the FuzzNexus engine, SQLite store,
// blob storage, and Wails events with strict state transitions and deterministic teardown.
type Session struct {
	id                 int64
	config             *fuzznexus.Config
	job                *engine.Job
	outProvider        *output.GUIOutputProvider
	st                 *store.Store
	blobs              *blobs.Storage
	logger             *slog.Logger
	state              State
	stateMu            sync.RWMutex
	lastErr            string
	doneChan           chan struct{}
	cancelCtx          context.CancelFunc
	teardownOnce       sync.Once
	emitCommitFn       func(output.CommittedEvent)
	emitProgressFn     func(ProgressDTO)
	emitStateFn        func(State, string)
	emitSafeStopFn     func(SafeStopPromptDTO)
	emitAdvisoryFn     func(AdvisoryAlertDTO)
	emitErrorFn        func(string, string)
	emitCircuitStateFn func(string, string)
	errorsMu           sync.RWMutex
	recentErrors       []NetworkErrorDTO
	nextErrorIndex     int
	jobDone            chan struct{}
	safeStopWF         *engine.SafeStopWorkflow
	blockHandled       bool
	blockMu            sync.Mutex
	targets            probeTargets
	verifyBaseCtx      context.Context
	verifyBaseCancel   context.CancelFunc
	verifyMu           sync.Mutex
	verifyWg           sync.WaitGroup
	verifyInFlight     bool
	lastVerifyAt       time.Time
	lastVerdict        *engine.DiagnosticVerdict
	debugLogFile       *os.File
}

// SessionOptions configures creation of a Session.
type SessionOptions struct {
	SessionID        int64
	Config           *fuzznexus.Config
	Store            *store.Store
	Blobs            *blobs.Storage
	Logger           *slog.Logger
	ResumeFilePath   string
	CancelCtx        context.CancelFunc
	EmitCommit       func(output.CommittedEvent)
	EmitProgress     func(ProgressDTO)
	EmitState        func(State, string)
	EmitSafeStop     func(SafeStopPromptDTO)
	EmitAdvisory     func(AdvisoryAlertDTO)
	EmitError        func(string, string)
	EmitCircuitState func(string, string)
}

// NewSession builds and wires a new Session instance with its dependencies.
func NewSession(opts SessionOptions) (*Session, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	verifyBaseCtx, verifyBaseCancel := context.WithCancel(context.Background())

	s := &Session{
		id:                 opts.SessionID,
		config:             opts.Config,
		st:                 opts.Store,
		blobs:              opts.Blobs,
		logger:             opts.Logger,
		state:              StateIdle,
		doneChan:           make(chan struct{}),
		jobDone:            make(chan struct{}),
		cancelCtx:          opts.CancelCtx,
		verifyBaseCtx:      verifyBaseCtx,
		verifyBaseCancel:   verifyBaseCancel,
		emitCommitFn:       opts.EmitCommit,
		emitProgressFn:     opts.EmitProgress,
		emitStateFn:        opts.EmitState,
		emitSafeStopFn:     opts.EmitSafeStop,
		emitAdvisoryFn:     opts.EmitAdvisory,
		emitErrorFn:        opts.EmitError,
		emitCircuitStateFn: opts.EmitCircuitState,
	}

	// 1. Construct fuzznexus.Job via assembly
	job, err := assembly.BuildJob(opts.Config)
	if err != nil {
		verifyBaseCancel()
		return nil, fmt.Errorf("failed to build FuzzNexus job: %w", err)
	}
	s.job = job

	// 1.5. Wire SafeStopWorkflow & Differential Canary Verifier for IP block vs wordlist resilience
	targets := deriveProbeTargets(opts.Config.Url, opts.Config.Headers, opts.Config.Data)
	s.targets = targets

	ipProvider := engine.NewManualVPNProvider("")
	if targets.IsLocalTarget {
		ipProvider = engine.NewManualVPNProvider(targets.RootCanaryURL)
	}
	bdConfig := engine.DefaultBanDetectorConfig()
	bdConfig.CooloffDuration = 1 * time.Second

	cookieHdr := ""
	for k, v := range opts.Config.Headers {
		if strings.EqualFold(k, "cookie") {
			cookieHdr = v
			break
		}
	}

	bd := engine.NewBanDetector(bdConfig, func(ctx context.Context) (bool, error) {
		outcome := engine.ProbeHTTPTarget(ctx, targets.RootCanaryURL, engine.CanaryProbeConfig{
			RootCanaryURL:     targets.RootCanaryURL,
			IsLocalTarget:     targets.IsLocalTarget,
			AllowNon200Origin: targets.AllowNon200Origin,
			Headers:           opts.Config.Headers,
			Cookies:           cookieHdr,
			BrowserProfile:    opts.Config.BrowserProfile,
			TLSProfile:        opts.Config.TLSProfile,
			Timeout:           3 * time.Second,
		}, true)
		if outcome.Error != nil {
			return true, outcome.Error
		}
		return outcome.IsBlocked || !outcome.Healthy, nil
	})

	wf := engine.NewSafeStopWorkflow(s.job.CircuitBreaker, bd, ipProvider, s.job, targets.RootCanaryURL, opts.ResumeFilePath)
	wf.SetBrowserProfile(opts.Config.BrowserProfile)
	wf.SetTLSProfile(opts.Config.TLSProfile)
	wf.SetTargetProbeFlags(targets.IsLocalTarget, targets.AllowNon200Origin)
	wf.SetSessionContext(cookieHdr, opts.Config.Headers)
	wordlistPath := ""
	if len(opts.Config.Wordlists) > 0 {
		wordlistPath = opts.Config.Wordlists[0]
	}
	wf.SetScanMetadata(opts.SessionID, opts.Config.Url, opts.Config.Method, wordlistPath)
	s.safeStopWF = wf

	wf.OnSafeStopPrompt = func(title, message, ipHash, rawIP, reason string, buttons []string) {
		canarySummary := ""
		s.verifyMu.Lock()
		if s.lastVerdict != nil && s.lastVerdict.Canary.RootURL != "" {
			if s.lastVerdict.Canary.RootError != "" {
				canarySummary = fmt.Sprintf("Origin Canary (%s) -> error: %s", s.lastVerdict.Canary.RootURL, s.lastVerdict.Canary.RootError)
			} else if s.lastVerdict.Canary.BenignPathStatus == 403 {
				canarySummary = fmt.Sprintf("Benign Path Canary (%s) -> 403 Blocked", s.lastVerdict.Canary.BenignPathURL)
			} else {
				canarySummary = fmt.Sprintf("Origin Canary (%s) -> %d %s", s.lastVerdict.Canary.RootURL, s.lastVerdict.Canary.RootStatus, s.lastVerdict.Canary.RootWAFHeader)
			}
		}
		s.verifyMu.Unlock()

		prompt := SafeStopPromptDTO{
			SessionID:     s.id,
			Title:         title,
			Message:       message,
			IPHash:        ipHash,
			RawIP:         rawIP,
			Reason:        reason,
			CanarySummary: strings.TrimSpace(canarySummary),
			Buttons:       buttons,
		}
		if s.emitSafeStopFn != nil {
			s.emitSafeStopFn(prompt)
		}
	}

	// Listen for CircuitBreaker state changes: run Differential Canary instead of blind IP-ban popup
	s.job.CircuitBreaker.AddOnStateChange(func(oldState, newState engine.CircuitState, reason string) {
		if s.job != nil && s.job.Output != nil {
			s.job.Output.Warning(fmt.Sprintf("[CircuitBreaker] %v -> %v: %s", oldState, newState, reason))
		}
		if s.emitCircuitStateFn != nil {
			s.emitCircuitStateFn(newState.String(), reason)
		}
		if newState == engine.StateOpen && strings.Contains(reason, "403") {
			s.launchBlockCauseVerification(reason)
		}
	})

	// 2. Wire GUIOutputProvider
	out := output.NewGUIOutputProvider(
		opts.Config.Context,
		opts.SessionID,
		opts.Logger,
		func(sessionID int64, batch []output.ResultDTO) error {
			if s.st == nil {
				return nil
			}
			return s.st.InsertResults(sessionID, batch)
		},
		func(event output.CommittedEvent) {
			if s.emitCommitFn != nil {
				s.emitCommitFn(event)
			}
		},
		nil, // emitFinished handled directly during teardown
		func(code string, msg string) {
			s.RecordError(msg)
			if s.emitErrorFn != nil {
				s.emitErrorFn(code, msg)
			}
		},
		func(fatalErr error) {
			s.handleFatal(fatalErr)
		},
	)

	// Configure progress listener
	out.SetOnProgress(func(p fuzznexus.Progress) {
		pct := 0.0
		if p.ReqTotal > 0 {
			pct = (float64(p.ReqCount) / float64(p.ReqTotal)) * 100.0
		}
		cState := ""
		if s.job != nil && s.job.CircuitBreaker != nil {
			cState = s.job.CircuitBreaker.State().String()
		}
		dto := ProgressDTO{
			SessionID:    opts.SessionID,
			StartedAt:    p.StartedAt,
			ReqCount:     p.ReqCount,
			ReqTotal:     p.ReqTotal,
			ReqSec:       p.ReqSec,
			QueuePos:     p.QueuePos,
			QueueTotal:   p.QueueTotal,
			ErrorCount:   p.ErrorCount,
			Percentage:   pct,
			BytesSent:    p.BytesSent,
			BytesRecv:    p.BytesRecv,
			CircuitState: cState,
		}
		if s.emitProgressFn != nil {
			s.emitProgressFn(dto)
		}
	})

	// Configure blob payload saver
	if opts.Blobs != nil {
		out.SetPayloadSaver(func(resultID int64, req []byte, resp []byte) {
			_ = opts.Blobs.SavePayload(opts.SessionID, resultID, req, resp)
		})
	}
	if opts.Config != nil && opts.Config.OutputDirectory != "" {
		out.SetOutputDirectory(opts.Config.OutputDirectory)
	}

	s.outProvider = out
	s.job.Output = out

	if opts.Config != nil && strings.TrimSpace(opts.Config.Debuglog) != "" {
		f, derr := os.OpenFile(strings.TrimSpace(opts.Config.Debuglog), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if derr != nil {
			s.logger.Warn("Failed to open debug log file", "path", opts.Config.Debuglog, "error", derr)
		} else {
			log.SetOutput(f)
			s.debugLogFile = f
		}
	}

	return s, nil
}

// Start kicks off the FuzzNexus engine execution in a background goroutine.
func (s *Session) Start() error {
	s.stateMu.Lock()
	if s.state != StateIdle {
		s.stateMu.Unlock()
		return ErrSessionNotIdle
	}
	s.state = StateRunning
	s.stateMu.Unlock()

	s.notifyState(StateRunning, "")

	SafeGo(s.logger, "scan-runner", func(panicErr error) {
		s.handleFatal(fmt.Errorf("fatal panic during scan execution: %w", panicErr))
	}, func() {
		defer close(s.jobDone)
		s.job.Start()
	})

	// Background monitor for natural completion or stop
	SafeGo(s.logger, "scan-teardown-monitor", func(panicErr error) {
		s.handleFatal(fmt.Errorf("fatal panic during scan teardown: %w", panicErr))
	}, func() {
		<-s.jobDone
		targetState := StateCompleted
		if s.job != nil && s.job.GetError() != "" {
			targetState = StateFailed
		}
		s.teardown(targetState)
	})

	return nil
}

// Pause pauses the running FuzzNexus scan workers.
func (s *Session) Pause() error {
	s.stateMu.Lock()
	if s.state != StateRunning {
		s.stateMu.Unlock()
		return ErrSessionNotRunning
	}
	s.state = StatePaused
	s.stateMu.Unlock()

	s.job.Pause()
	s.notifyState(StatePaused, "")
	return nil
}

// Resume resumes a paused FuzzNexus scan.
func (s *Session) Resume() error {
	s.stateMu.Lock()
	if s.state != StatePaused {
		s.stateMu.Unlock()
		return ErrSessionNotPaused
	}
	s.state = StateRunning
	s.stateMu.Unlock()

	s.job.Resume()
	s.notifyState(StateRunning, "")
	return nil
}

// Stop requests the FuzzNexus engine to halt execution and executes teardown.
func (s *Session) Stop() error {
	s.stateMu.Lock()
	if s.state == StateCompleted || s.state == StateFailed || s.state == StateStopped {
		s.stateMu.Unlock()
		return nil
	}
	s.state = StateStopping
	s.stateMu.Unlock()

	if s.verifyBaseCancel != nil {
		s.verifyBaseCancel()
	}
	s.notifyState(StateStopping, "")
	s.job.Stop()
	if s.cancelCtx != nil {
		s.cancelCtx()
	}

	// Teardown with StateStopped
	s.teardown(StateStopped)
	return nil
}

// ChangeRate dynamically adjusts the request rate limit on the running job.
func (s *Session) ChangeRate(rps int) {
	if s.job != nil && s.job.Rate != nil {
		s.job.Rate.ChangeRate(rps)
	}
}

// AddLiveFilter dynamically registers a new filter on MatcherManager during execution.
func (s *Session) AddLiveFilter(name, option string) error {
	if s.config == nil || s.config.MatcherManager == nil {
		return errors.New("scan: matcher manager is not initialized")
	}
	return s.config.MatcherManager.AddFilter(name, option, false)
}

// ID returns the SQLite session identifier.
func (s *Session) ID() int64 {
	return s.id
}

// State returns current session state and last error message.
func (s *Session) State() (State, string) {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state, s.lastErr
}

// Wait blocks until the session has completely finished and tore down.
func (s *Session) Wait() {
	<-s.doneChan
}

// handleFatal handles persistent database or engine panics.
func (s *Session) handleFatal(err error) {
	s.stateMu.Lock()
	s.lastErr = err.Error()
	s.state = StateFailed
	s.stateMu.Unlock()

	if s.verifyBaseCancel != nil {
		s.verifyBaseCancel()
	}
	s.logger.Error("Fatal scan failure encountered", "error", err)
	s.notifyState(StateFailed, err.Error())

	// Stop engine workers
	if s.job != nil {
		s.job.Stop()
	}
	if s.cancelCtx != nil {
		s.cancelCtx()
	}

	// Teardown with StateFailed
	s.teardown(StateFailed)
}

// teardown executes the mandatory 4-step teardown sequence:
// 1. Stop job and wait for worker goroutines to join.
// 2. Wait for any in-flight Differential Canary verifier to complete cleanly.
// 3. Finalize GUIOutputProvider with independent context & timeout.
// 4. Commit/checkpoint SQLite WAL and record final state.
// 5. Mark session complete and close doneChan.
func (s *Session) teardown(desiredState State) {
	s.teardownOnce.Do(func() {
		s.logger.Info("Executing deterministic session teardown", "sessionID", s.id, "desiredState", desiredState)

		// Step 1: Ensure job is stopped and workers exited
		if s.job != nil {
			s.job.Stop()
		}
		select {
		case <-s.jobDone:
		case <-time.After(2 * time.Second):
			s.logger.Warn("Timed out waiting for job workers to join", "sessionID", s.id)
		}

		// Close idle HTTP keep-alive connections once workers finish
		if s.job != nil && s.job.Runner != nil {
			if c, ok := s.job.Runner.(interface{ CloseIdleConnections() }); ok {
				c.CloseIdleConnections()
			}
		}

		// Step 1.5: Wait for in-flight canary verification (already canceled if Stopped/Failed)
		if desiredState != StateCompleted && s.verifyBaseCancel != nil {
			s.verifyBaseCancel()
		}
		s.verifyWg.Wait()
		if s.verifyBaseCancel != nil {
			s.verifyBaseCancel()
		}

		// Step 2: Finalize output provider with independent 5-second context
		finCtx, finCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer finCancel()
		if s.outProvider != nil {
			if finErr := s.outProvider.FinalizeWithContext(finCtx); finErr != nil {
				s.logger.Warn("Output provider Finalize returned error", "error", finErr)
			}
		}
		if s.job != nil && s.job.AuditLogger != nil {
			s.job.AuditLogger.Close()
		}
		if s.debugLogFile != nil {
			log.SetOutput(os.Stderr)
			_ = s.debugLogFile.Close()
			s.debugLogFile = nil
		}

		// Determine final state
		s.stateMu.Lock()
		finalState := desiredState
		if s.job != nil && s.job.GetError() != "" {
			finalState = StateFailed
			if s.lastErr == "" {
				s.lastErr = s.job.GetError()
			}
		}
		if s.state == StateFailed {
			finalState = StateFailed
		} else if s.state == StateStopping || desiredState == StateStopped {
			finalState = StateStopped
		}
		s.state = finalState
		lastErrMsg := s.lastErr
		s.stateMu.Unlock()

		// Step 3: Checkpoint SQLite and update session state
		if s.st != nil {
			_ = s.st.UpdateSessionState(s.id, string(finalState), lastErrMsg)
			_ = s.st.CheckpointWAL()
		}

		// Step 4: Emit final state notification and close doneChan
		s.notifyState(finalState, lastErrMsg)
		close(s.doneChan)
		s.logger.Info("Deterministic session teardown complete", "sessionID", s.id, "finalState", finalState)
	})
}

// TriggerSafeStop explicitly halts fuzzing workers, enters PausedIPBlock, and emits the modal prompt.
func (s *Session) TriggerSafeStop(reason string) {
	s.blockMu.Lock()
	if s.blockHandled {
		s.blockMu.Unlock()
		return
	}
	s.blockHandled = true
	s.blockMu.Unlock()

	s.logger.Warn("IP block triggered Safe Stop Workflow", "reason", reason)

	s.stateMu.Lock()
	if s.state != StateRunning {
		s.stateMu.Unlock()
		return
	}
	s.state = StatePaused
	s.stateMu.Unlock()

	// 1. Immediately pause job so 0 requests leave the machine
	s.job.Pause()
	s.notifyState(StatePaused, "Safe Stop: IP block suspected ("+reason+")")

	// 2. Invoke EnterPausedIPBlock
	if s.safeStopWF != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.safeStopWF.EnterPausedIPBlock(ctx, reason)
	}
}

// TestConnection tests whether the target connection has been restored with a new IP.
func (s *Session) TestConnection(ctx context.Context) (SafeStopTestResultDTO, error) {
	if s.safeStopWF == nil {
		return SafeStopTestResultDTO{Success: false, Error: "Safe stop workflow is not initialized"}, nil
	}
	success, newHash, err := s.safeStopWF.TestConnection(ctx)
	newRaw := s.safeStopWF.NewRawIP()
	if err != nil {
		return SafeStopTestResultDTO{
			Success:   false,
			NewIPHash: newHash,
			NewRawIP:  newRaw,
			Error:     err.Error(),
		}, nil
	}
	return SafeStopTestResultDTO{Success: success, NewIPHash: newHash, NewRawIP: newRaw}, nil
}

// ResumeSafeStop continues scanning from the exact paused index after IP change.
func (s *Session) ResumeSafeStop(ctx context.Context) error {
	if s.safeStopWF == nil {
		return errors.New("safe stop workflow is not initialized")
	}
	success, err := s.safeStopWF.Continue(ctx)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("cannot resume: connection probe failed")
	}

	s.stateMu.Lock()
	s.state = StateRunning
	s.blockMu.Lock()
	s.blockHandled = false
	s.blockMu.Unlock()
	s.stateMu.Unlock()

	s.notifyState(StateRunning, "")
	return nil
}

// ResumeAnyway forces continuation even if IP hasn't changed.
func (s *Session) ResumeAnyway(ctx context.Context) error {
	if s.safeStopWF == nil {
		return errors.New("safe stop workflow is not initialized")
	}
	if err := s.safeStopWF.ResumeAnyway(ctx); err != nil {
		return err
	}

	s.stateMu.Lock()
	s.state = StateRunning
	s.blockMu.Lock()
	s.blockHandled = false
	s.blockMu.Unlock()
	s.stateMu.Unlock()

	s.notifyState(StateRunning, "")
	return nil
}

// CancelSafeStop cancels execution, saves encrypted state, and halts the session.
func (s *Session) CancelSafeStop() error {
	if s.safeStopWF != nil {
		_ = s.safeStopWF.Cancel()
	}
	return s.Stop()
}

func (s *Session) launchBlockCauseVerification(triggerReason string) {
	s.verifyMu.Lock()
	if s.verifyInFlight {
		s.verifyMu.Unlock()
		return
	}
	if !s.lastVerifyAt.IsZero() && time.Since(s.lastVerifyAt) < 30*time.Second {
		prev := s.lastVerdict
		s.verifyMu.Unlock()
		if prev != nil && (prev.Verdict == engine.VerdictWordlistTrigger || prev.Verdict == engine.VerdictEndpointRule) {
			s.job.CircuitBreaker.SetWordlistAdvisoryBypass(true, prev.EvidenceHeader, prev.FixedBodyLength)
		}
		return
	}
	st, _ := s.State()
	if st != StateRunning {
		s.verifyMu.Unlock()
		return
	}
	s.verifyInFlight = true
	s.verifyWg.Add(1)
	s.verifyMu.Unlock()

	SafeGo(s.logger, "block-cause-verifier", func(panicErr error) {
		s.logger.Warn("Block cause verifier panicked", "error", panicErr)
	}, func() {
		defer s.verifyWg.Done()
		defer func() {
			s.verifyMu.Lock()
			s.verifyInFlight = false
			s.lastVerifyAt = time.Now()
			s.verifyMu.Unlock()
		}()

		baseCtx := s.verifyBaseCtx
		if baseCtx == nil {
			baseCtx = context.Background()
		}
		verifyCtx, cancel := context.WithTimeout(baseCtx, 12*time.Second)
		defer cancel()

		hdrCopy := make(map[string]string, len(s.config.Headers))
		cookieHdr := ""
		for k, v := range s.config.Headers {
			hdrCopy[k] = v
			if strings.EqualFold(k, "cookie") {
				cookieHdr = v
			}
		}

		canaryCfg := engine.CanaryProbeConfig{
			RootCanaryURL:     s.targets.RootCanaryURL,
			IsLocalTarget:     s.targets.IsLocalTarget,
			AllowNon200Origin: s.targets.AllowNon200Origin,
			Headers:           hdrCopy,
			Cookies:           cookieHdr,
			BrowserProfile:    s.config.BrowserProfile,
			TLSProfile:        s.config.TLSProfile,
			Timeout:           5 * time.Second,
		}
		if s.targets.CanProbeBenign && strings.Contains(s.targets.BenignTemplateURL, "FUZZ") {
			canaryCfg.BenignPathURL = strings.ReplaceAll(s.targets.BenignTemplateURL, "FUZZ", engine.RandomCanarySlug())
		}

		stats := s.job.CircuitBreaker.Stats()
		forensics := s.job.CircuitBreaker.ForensicsSnapshot()
		verdict := engine.EvaluateBlockCause(verifyCtx, stats, forensics, canaryCfg)

		if verifyCtx.Err() != nil {
			s.logger.Warn("Differential Canary verification did not finish before context end",
				"sessionID", s.id,
				"err", verifyCtx.Err(),
			)
			return
		}

		curState, _ := s.State()
		if curState == StateStopping || curState == StateStopped || curState == StateFailed {
			return
		}

		s.verifyMu.Lock()
		vCopy := verdict
		s.lastVerdict = &vCopy
		s.verifyMu.Unlock()

		s.logger.Info("Differential Canary verdict reached",
			"sessionID", s.id,
			"verdict", verdict.Verdict,
			"confidence", verdict.Confidence,
			"rootStatus", verdict.Canary.RootStatus,
			"benignStatus", verdict.Canary.BenignPathStatus,
			"triggerReason", triggerReason,
		)

		alert := AdvisoryAlertDTO{
			SessionID:        s.id,
			Verdict:          string(verdict.Verdict),
			Confidence:       verdict.Confidence,
			Title:            verdict.Title,
			Summary:          verdict.Summary,
			EvidenceHeader:   verdict.EvidenceHeader,
			FixedBodyLength:  verdict.FixedBodyLength,
			SamplePayloads:   verdict.SamplePayloads,
			Canary:           verdict.Canary,
			SuggestedActions: verdict.SuggestedActions,
		}

		switch verdict.Verdict {
		case engine.VerdictIPBan:
			jobAlreadyDone := false
			select {
			case <-s.jobDone:
				jobAlreadyDone = true
			default:
			}
			if !jobAlreadyDone && curState == StateRunning {
				s.TriggerSafeStop(verdict.Summary)
			} else if s.emitAdvisoryFn != nil {
				s.emitAdvisoryFn(alert)
			}

		case engine.VerdictWordlistTrigger:
			s.job.CircuitBreaker.SetWordlistAdvisoryBypass(true, verdict.EvidenceHeader, verdict.FixedBodyLength)
			if s.emitCircuitStateFn != nil {
				s.emitCircuitStateFn("Closed", "WAF payload rule triggered; IP healthy")
			}
			if s.emitAdvisoryFn != nil {
				s.emitAdvisoryFn(alert)
			}

		case engine.VerdictEndpointRule:
			s.job.CircuitBreaker.SetWordlistAdvisoryBypass(true, "", 0)
			if s.emitCircuitStateFn != nil {
				s.emitCircuitStateFn("Closed", "Endpoint rule detected; IP healthy")
			}
			if s.emitAdvisoryFn != nil {
				s.emitAdvisoryFn(alert)
			}

		case engine.VerdictRateLimit:
			s.job.CircuitBreaker.SetState(engine.StateHalfOpen, verdict.Title)
			if s.emitAdvisoryFn != nil {
				s.emitAdvisoryFn(alert)
			}
		}
	})
}

func deriveProbeTargets(rawURL string, headers map[string]string, data string) probeTargets {
	target := rawURL
	var analysis *TargetAnalysis
	if norm, err := NormalizeTarget(rawURL, headers, data); err == nil && norm != nil {
		target = norm.Normalized
		analysis = norm
	} else if !strings.Contains(target, "://") {
		target = "http://" + target
	}

	u, err := url.Parse(target)
	if err != nil || u.Host == "" {
		return probeTargets{
			RootCanaryURL:     target,
			AllowNon200Origin: true,
		}
	}

	isExplicitHealth := strings.Contains(u.Host, "8090") || strings.Contains(strings.ToLower(u.Path), "/health")
	isLocalHost := strings.Contains(u.Host, "127.0.0.1") || strings.Contains(u.Host, "localhost") || strings.Contains(u.Host, "[::1]")

	res := probeTargets{
		IsLocalTarget: isLocalHost || isExplicitHealth,
	}

	if isExplicitHealth {
		res.RootCanaryURL = u.Scheme + "://" + u.Host + "/health"
		res.AllowNon200Origin = false
	} else {
		res.RootCanaryURL = u.Scheme + "://" + u.Host + "/"
		res.AllowNon200Origin = true
	}

	canProbe := strings.Contains(target, "FUZZ")
	if analysis != nil && analysis.FuzzLocation != "path" && analysis.FuzzLocation != "query" {
		canProbe = false
	}
	if canProbe {
		res.CanProbeBenign = true
		res.BenignTemplateURL = target
	}
	return res
}

func (s *Session) notifyState(state State, msg string) {
	if s.emitStateFn != nil {
		s.emitStateFn(state, msg)
	}
}

// BuildFFUFConfig creates an *fuzznexus.Config from serializable UI options.
func BuildFFUFConfig(ctx context.Context, cancel context.CancelFunc, targetURL string, wordlist string, threads int, rate int) (*fuzznexus.Config, error) {
	norm, err := NormalizeTarget(targetURL, nil, "")
	if err == nil && norm != nil {
		targetURL = norm.Normalized
	}

	opts := fuzznexus.NewConfigOptions()
	opts.HTTP.URL = targetURL
	opts.HTTP.DetectSoftBlock = true
	opts.Input.Wordlists = []string{wordlist}
	if threads > 0 {
		opts.General.Threads = threads
	} else {
		opts.General.Threads = 10
	}
	opts.General.Rate = rate
	opts.General.Quiet = true
	opts.General.Noninteractive = true

	conf, err := fuzznexus.ConfigFromOptions(opts, ctx, cancel)
	if err != nil {
		return nil, err
	}

	// Install default status matchers (200,204,301,302,307,401,403,405,500)
	mm, err := filter.FromConfig(opts, true)
	if err != nil {
		return nil, err
	}
	conf.MatcherManager = mm

	return conf, nil
}

// RecordError logs a network error to the in-memory ring buffer.
func (s *Session) RecordError(msg string) {
	if msg == "" || strings.Contains(msg, "context canceled") || strings.Contains(msg, "operation was canceled") {
		return
	}
	s.errorsMu.Lock()
	defer s.errorsMu.Unlock()
	s.nextErrorIndex++
	entry := NetworkErrorDTO{
		Index:     s.nextErrorIndex,
		Message:   msg,
		Timestamp: time.Now().UTC(),
	}
	if len(s.recentErrors) >= 100 {
		s.recentErrors = s.recentErrors[1:]
	}
	s.recentErrors = append(s.recentErrors, entry)
}

// GetRecentErrors returns a copy of recently recorded network errors.
func (s *Session) GetRecentErrors() []NetworkErrorDTO {
	s.errorsMu.RLock()
	defer s.errorsMu.RUnlock()
	res := make([]NetworkErrorDTO, len(s.recentErrors))
	copy(res, s.recentErrors)
	return res
}
