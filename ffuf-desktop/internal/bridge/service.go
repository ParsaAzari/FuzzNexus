package bridge

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"ffuf-desktop/internal/blobs"
	"ffuf-desktop/internal/output"
	"ffuf-desktop/internal/scan"
	"ffuf-desktop/internal/store"

	"github.com/ffuf/ffuf/v2/pkg/ffuf"
	"github.com/ffuf/ffuf/v2/pkg/filter"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ScanRequestDTO represents parameters submitted from the frontend UI to start a fuzzing scan.
type ScanRequestDTO struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Data            string            `json:"data"`
	Wordlist        string            `json:"wordlist"`
	Threads         int               `json:"threads"`
	Rate            int               `json:"rate"`
	Matchers        string            `json:"matchers"`
	Filters         string            `json:"filters"`
	Timeout         int               `json:"timeout"`
	FollowRedirects bool              `json:"follow_redirects"`
}

// ResultsPageDTO represents a chunk of paginated results returned for UI consumption.
type ResultsPageDTO struct {
	Results []output.ResultDTO `json:"results"`
	HasMore bool               `json:"has_more"`
	LastID  int64              `json:"last_id"`
}

// StateEvent represents a scan state transition broadcast to the frontend.
type StateEvent struct {
	SessionID int64      `json:"session_id"`
	State     scan.State `json:"state"`
	Error     string     `json:"error,omitempty"`
}

// Service provides the Wails bridge methods exposed to the frontend TypeScript layer.
type Service struct {
	wailsCtx      context.Context
	wailsCtxMu    sync.RWMutex
	emitter       func(name string, data interface{})
	emitterMu     sync.RWMutex
	store         *store.Store
	blobs         *blobs.Storage
	activeSession *scan.Session
	sessionMu     sync.Mutex
	dataDir       string
	logger        *slog.Logger
}

// NewService initializes the bridge service with persistent store and blobs storage.
func NewService(dataDir string, logger *slog.Logger) (*Service, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".ffuf-desktop")
	}

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create app data directory: %w", err)
	}

	st, err := store.NewStore(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SQLite store: %w", err)
	}

	bl, err := blobs.NewStorage(dataDir, blobs.DefaultSessionQuota)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("failed to initialize blob storage: %w", err)
	}

	return &Service{
		store:   st,
		blobs:   bl,
		dataDir: dataDir,
		logger:  logger,
	}, nil
}

// SetContext assigns the Wails runtime context once the window initializes.
func (s *Service) SetContext(ctx context.Context) {
	s.wailsCtxMu.Lock()
	defer s.wailsCtxMu.Unlock()
	s.wailsCtx = ctx
}

// SetEventEmitter allows providing a custom event emitter (e.g. for testing).
func (s *Service) SetEventEmitter(fn func(name string, data interface{})) {
	s.emitterMu.Lock()
	defer s.emitterMu.Unlock()
	s.emitter = fn
}

func (s *Service) emitEvent(name string, data interface{}) {
	s.emitterMu.RLock()
	custom := s.emitter
	s.emitterMu.RUnlock()

	if custom != nil {
		custom(name, data)
		return
	}

	s.wailsCtxMu.RLock()
	ctx := s.wailsCtx
	s.wailsCtxMu.RUnlock()

	if ctx != nil {
		wailsRuntime.EventsEmit(ctx, name, data)
	}
}

// StartScan launches a new in-process scan session.
func (s *Service) StartScan(req ScanRequestDTO) (int64, error) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	if s.activeSession != nil {
		state, _ := s.activeSession.State()
		if state == scan.StateRunning || state == scan.StatePaused || state == scan.StateStopping {
			return 0, errors.New("a scan session is already currently active")
		}
	}

	if req.URL == "" {
		return 0, errors.New("target URL cannot be empty")
	}
	if req.Wordlist == "" {
		return 0, errors.New("wordlist path cannot be empty")
	}
	if _, err := os.Stat(req.Wordlist); err != nil {
		return 0, fmt.Errorf("wordlist file not accessible: %w", err)
	}

	// Prepare config records
	configBytes, _ := json.Marshal(req)
	redactedReq := req
	redactedHeaders := make(map[string]string)
	for k, v := range req.Headers {
		lk := strings.ToLower(k)
		if lk == "authorization" || lk == "cookie" || lk == "set-cookie" || strings.Contains(lk, "token") {
			redactedHeaders[k] = "***"
		} else {
			redactedHeaders[k] = v
		}
	}
	redactedReq.Headers = redactedHeaders
	redactedBytes, _ := json.Marshal(redactedReq)

	sessionID, err := s.store.CreateSession(string(configBytes), string(redactedBytes))
	if err != nil {
		return 0, fmt.Errorf("failed to initialize session record: %w", err)
	}

	scanCtx, cancel := context.WithCancel(context.Background())

	// Build ffuf Config
	opts := ffuf.NewConfigOptions()
	opts.HTTP.URL = req.URL
	opts.Input.Wordlists = []string{req.Wordlist}
	if req.Threads > 0 {
		opts.General.Threads = req.Threads
	} else {
		opts.General.Threads = 10
	}
	opts.General.Rate = req.Rate
	opts.General.Quiet = true
	opts.General.Noninteractive = true
	opts.HTTP.Method = req.Method
	if opts.HTTP.Method == "" {
		opts.HTTP.Method = "GET"
	}
	opts.HTTP.Data = req.Data
	opts.HTTP.FollowRedirects = req.FollowRedirects
	opts.HTTP.Timeout = req.Timeout

	// Add custom headers
	for k, v := range req.Headers {
		opts.HTTP.Headers = append(opts.HTTP.Headers, fmt.Sprintf("%s: %s", k, v))
	}

	// Matcher & filter options
	if req.Matchers != "" {
		opts.Matcher.Status = req.Matchers
	}
	if req.Filters != "" {
		opts.Filter.Status = req.Filters
	}

	conf, err := ffuf.ConfigFromOptions(opts, scanCtx, cancel)
	if err != nil {
		cancel()
		return 0, fmt.Errorf("failed to parse ffuf configuration: %w", err)
	}

	// Install matcher manager
	mm, err := filter.FromConfig(opts, req.Matchers == "")
	if err != nil {
		cancel()
		return 0, fmt.Errorf("failed to construct matcher manager: %w", err)
	}
	conf.MatcherManager = mm

	sess, err := scan.NewSession(scan.SessionOptions{
		SessionID: sessionID,
		Config:    conf,
		Store:     s.store,
		Blobs:     s.blobs,
		Logger:    s.logger,
		CancelCtx: cancel,
		EmitCommit: func(e output.CommittedEvent) {
			s.emitEvent("results:committed", e)
		},
		EmitProgress: func(p scan.ProgressDTO) {
			s.emitEvent("scan:progress", p)
		},
		EmitState: func(st scan.State, errMsg string) {
			s.emitEvent("scan:state", StateEvent{
				SessionID: sessionID,
				State:     st,
				Error:     errMsg,
			})
		},
	})
	if err != nil {
		cancel()
		return 0, fmt.Errorf("failed to create scan session: %w", err)
	}

	if err := sess.Start(); err != nil {
		cancel()
		return 0, fmt.Errorf("failed to start scan session: %w", err)
	}

	s.activeSession = sess
	return sessionID, nil
}

// PauseScan pauses the currently active scan.
func (s *Service) PauseScan() error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.activeSession == nil {
		return errors.New("no active scan session")
	}
	return s.activeSession.Pause()
}

// ResumeScan resumes the currently paused scan.
func (s *Service) ResumeScan() error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.activeSession == nil {
		return errors.New("no active scan session")
	}
	return s.activeSession.Resume()
}

// StopScan terminates the active scan session.
func (s *Service) StopScan() error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.activeSession == nil {
		return nil
	}
	return s.activeSession.Stop()
}

// GetResultsSince retrieves committed results after lastID for a session.
func (s *Service) GetResultsSince(sessionID int64, lastID int64, limit int) (ResultsPageDTO, error) {
	results, hasMore, err := s.store.GetResultsSince(sessionID, lastID, limit)
	if err != nil {
		return ResultsPageDTO{}, err
	}

	newLastID := lastID
	if len(results) > 0 {
		newLastID = results[len(results)-1].ID
	}

	return ResultsPageDTO{
		Results: results,
		HasMore: hasMore,
		LastID:  newLastID,
	}, nil
}

// GetResultPayload fetches raw request/response preview with 500KB truncation.
func (s *Service) GetResultPayload(sessionID int64, resultID int64, maxBytes int) (*blobs.PayloadDTO, error) {
	return s.blobs.GetResultPayload(sessionID, resultID, maxBytes)
}

// AddLiveFilter adds a status, size, line, or word filter during runtime execution.
func (s *Service) AddLiveFilter(name, option string) error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.activeSession == nil {
		return errors.New("no active scan session")
	}
	return s.activeSession.AddLiveFilter(name, option)
}

// UpdateRate dynamically changes the request rate limit on the active scan.
func (s *Service) UpdateRate(rps int) error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.activeSession == nil {
		return errors.New("no active scan session")
	}
	s.activeSession.ChangeRate(rps)
	return nil
}

// GetSessions lists scan history sessions.
func (s *Service) GetSessions(limit int) ([]store.SessionDTO, error) {
	return s.store.GetSessions(limit)
}

// GetSession retrieves metadata for a single session.
func (s *Service) GetSession(sessionID int64) (*store.SessionDTO, error) {
	return s.store.GetSession(sessionID)
}

// ExportResults exports scan results to disk in CSV or JSON format.
func (s *Service) ExportResults(sessionID int64, format string, targetPath string) error {
	results, _, err := s.store.GetResultsSince(sessionID, 0, 50000)
	if err != nil {
		return fmt.Errorf("failed to read results: %w", err)
	}

	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer f.Close()

	if strings.ToLower(format) == "csv" {
		w := csv.NewWriter(f)
		defer w.Flush()

		_ = w.Write([]string{"ID", "Position", "Status", "Length", "Words", "Lines", "DurationMs", "URL", "Redirect", "ContentType", "Hash"})
		for _, r := range results {
			_ = w.Write([]string{
				strconv.FormatInt(r.ID, 10),
				strconv.Itoa(r.Position),
				strconv.FormatInt(r.Status, 10),
				strconv.FormatInt(r.Length, 10),
				strconv.FormatInt(r.Words, 10),
				strconv.FormatInt(r.Lines, 10),
				strconv.FormatInt(r.DurationMs, 10),
				r.URL,
				r.RedirectLocation,
				r.ContentType,
				r.Hash,
			})
		}
		return nil
	}

	// Default: JSON export
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

// Close gracefully terminates any running scan and releases store/blob resources.
func (s *Service) Close() error {
	s.sessionMu.Lock()
	if s.activeSession != nil {
		_ = s.activeSession.Stop()
	}
	s.sessionMu.Unlock()

	var errs []error
	if s.blobs != nil {
		if err := s.blobs.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.store != nil {
		if err := s.store.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing bridge service: %v", errs)
	}
	return nil
}
