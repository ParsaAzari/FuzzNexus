package scan

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"ffuf-desktop/internal/blobs"
	"ffuf-desktop/internal/output"
	"ffuf-desktop/internal/store"

	"github.com/ffuf/ffuf/v2/pkg/assembly"
	"github.com/ffuf/ffuf/v2/pkg/engine"
	"github.com/ffuf/ffuf/v2/pkg/ffuf"
	"github.com/ffuf/ffuf/v2/pkg/filter"
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
	ErrSessionNotIdle     = errors.New("scan: session is not in Idle state")
	ErrSessionNotRunning  = errors.New("scan: session is not in Running state")
	ErrSessionNotPaused   = errors.New("scan: session is not in Paused state")
	ErrSessionTerminated  = errors.New("scan: session has already terminated")
)

// ProgressDTO contains live metrics emitted to the frontend during a scan.
type ProgressDTO struct {
	SessionID  int64     `json:"session_id"`
	StartedAt  time.Time `json:"started_at"`
	ReqCount   int       `json:"req_count"`
	ReqTotal   int       `json:"req_total"`
	ReqSec     int64     `json:"req_sec"`
	QueuePos   int       `json:"queue_pos"`
	QueueTotal int       `json:"queue_total"`
	ErrorCount int       `json:"error_count"`
	Percentage float64   `json:"percentage"`
}

// Session manages a single scan lifecycle, orchestrating ffuf engine, SQLite store,
// blob storage, and Wails events with strict state transitions and deterministic teardown.
type Session struct {
	id             int64
	config         *ffuf.Config
	job            *engine.Job
	outProvider    *output.GUIOutputProvider
	st             *store.Store
	blobs          *blobs.Storage
	logger         *slog.Logger
	state          State
	stateMu        sync.RWMutex
	lastErr        string
	doneChan       chan struct{}
	cancelCtx      context.CancelFunc
	teardownOnce   sync.Once
	emitCommitFn   func(output.CommittedEvent)
	emitProgressFn func(ProgressDTO)
	emitStateFn    func(State, string)
	jobDone        chan struct{}
}

// SessionOptions configures creation of a Session.
type SessionOptions struct {
	SessionID      int64
	Config         *ffuf.Config
	Store          *store.Store
	Blobs          *blobs.Storage
	Logger         *slog.Logger
	CancelCtx      context.CancelFunc
	EmitCommit     func(output.CommittedEvent)
	EmitProgress   func(ProgressDTO)
	EmitState      func(State, string)
}

// NewSession builds and wires a new Session instance with its dependencies.
func NewSession(opts SessionOptions) (*Session, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	s := &Session{
		id:             opts.SessionID,
		config:         opts.Config,
		st:             opts.Store,
		blobs:          opts.Blobs,
		logger:         opts.Logger,
		state:          StateIdle,
		doneChan:       make(chan struct{}),
		jobDone:        make(chan struct{}),
		cancelCtx:      opts.CancelCtx,
		emitCommitFn:   opts.EmitCommit,
		emitProgressFn: opts.EmitProgress,
		emitStateFn:    opts.EmitState,
	}

	// 1. Construct ffuf.Job via assembly
	job, err := assembly.BuildJob(opts.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to build ffuf job: %w", err)
	}
	s.job = job

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
		nil,
		func(fatalErr error) {
			s.handleFatal(fatalErr)
		},
	)

	// Configure progress listener
	out.SetOnProgress(func(p ffuf.Progress) {
		pct := 0.0
		if p.ReqTotal > 0 {
			pct = (float64(p.ReqCount) / float64(p.ReqTotal)) * 100.0
		}
		dto := ProgressDTO{
			SessionID:  opts.SessionID,
			StartedAt:  p.StartedAt,
			ReqCount:   p.ReqCount,
			ReqTotal:   p.ReqTotal,
			ReqSec:     p.ReqSec,
			QueuePos:   p.QueuePos,
			QueueTotal: p.QueueTotal,
			ErrorCount: p.ErrorCount,
			Percentage: pct,
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

	s.outProvider = out
	s.job.Output = out

	return s, nil
}

// Start kicks off the ffuf engine execution in a background goroutine.
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
		s.teardown(StateCompleted)
	})

	return nil
}

// Pause pauses the running ffuf scan workers.
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

// Resume resumes a paused ffuf scan.
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

// Stop requests the ffuf engine to halt execution and executes teardown.
func (s *Session) Stop() error {
	s.stateMu.Lock()
	if s.state == StateCompleted || s.state == StateFailed || s.state == StateStopped {
		s.stateMu.Unlock()
		return nil
	}
	s.state = StateStopping
	s.stateMu.Unlock()

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
// 2. Finalize GUIOutputProvider with independent context & timeout.
// 3. Commit/checkpoint SQLite WAL and record final state.
// 4. Mark session complete and close doneChan.
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

		// Step 2: Finalize output provider with independent 5-second context
		finCtx, finCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer finCancel()
		if s.outProvider != nil {
			if finErr := s.outProvider.FinalizeWithContext(finCtx); finErr != nil {
				s.logger.Warn("Output provider Finalize returned error", "error", finErr)
			}
		}

		// Determine final state
		s.stateMu.Lock()
		finalState := desiredState
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

func (s *Session) notifyState(state State, msg string) {
	if s.emitStateFn != nil {
		s.emitStateFn(state, msg)
	}
}

// BuildFFUFConfig creates an *ffuf.Config from serializable UI options.
func BuildFFUFConfig(ctx context.Context, cancel context.CancelFunc, url string, wordlist string, threads int, rate int) (*ffuf.Config, error) {
	opts := ffuf.NewConfigOptions()
	opts.HTTP.URL = url
	opts.Input.Wordlists = []string{wordlist}
	if threads > 0 {
		opts.General.Threads = threads
	} else {
		opts.General.Threads = 10
	}
	opts.General.Rate = rate
	opts.General.Quiet = true
	opts.General.Noninteractive = true

	conf, err := ffuf.ConfigFromOptions(opts, ctx, cancel)
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
