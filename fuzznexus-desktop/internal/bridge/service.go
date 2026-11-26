package bridge

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fuzznexus-desktop/internal/blobs"
	"fuzznexus-desktop/internal/output"
	"fuzznexus-desktop/internal/scan"
	"fuzznexus-desktop/internal/store"
	"fuzznexus-desktop/internal/wordlists"

	"github.com/parsaazari/fuzznexus/pkg/engine"
	"github.com/parsaazari/fuzznexus/pkg/filter"
	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
	"github.com/parsaazari/fuzznexus/pkg/input"
	"github.com/parsaazari/fuzznexus/pkg/runner"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ScanRequestDTO represents parameters submitted from the frontend UI to start a fuzzing scan.
type ScanRequestDTO struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Data            string            `json:"data"`
	Wordlist        string            `json:"wordlist"`
	Wordlists       []string          `json:"wordlists,omitempty"`
	InputMode       string            `json:"input_mode,omitempty"`
	Threads         int               `json:"threads"`
	Rate            int               `json:"rate"`
	Matchers        string            `json:"matchers"`
	Filters         string            `json:"filters"`
	Timeout         int               `json:"timeout"`
	FollowRedirects bool              `json:"follow_redirects"`
	AutoCalibration bool              `json:"auto_calibration"`
	ProxyURL        string            `json:"proxy_url"`
	TLSProfile      string            `json:"tls_profile"`
	BrowserProfile  string            `json:"browser_profile"`
	RotateOnBlock   bool              `json:"rotate_on_block"`
	ResumeFromIndex int               `json:"resume_from_index,omitempty"`

	// Phase 1: HTTP, Timing, Input Processing, Matchers/Filters & Safety
	ReplayProxyURL            string   `json:"replay_proxy_url,omitempty"`
	Raw                       bool     `json:"raw,omitempty"`
	Http2                     bool     `json:"http2,omitempty"`
	IgnoreBody                bool     `json:"ignore_body,omitempty"`
	CloseConnOnThrottle       bool     `json:"close_conn_on_throttle,omitempty"`
	DetectSoftBlock           *bool    `json:"detect_soft_block,omitempty"`
	Delay                     string   `json:"delay,omitempty"`
	MaxTime                   int      `json:"max_time,omitempty"`
	MaxTimeJob                int      `json:"max_time_job,omitempty"`
	StopOn403                 bool     `json:"stop_on_403,omitempty"`
	StopOnErrors              bool     `json:"stop_on_errors,omitempty"`
	StopOnAll                 bool     `json:"stop_on_all,omitempty"`
	Extensions                string   `json:"extensions,omitempty"`
	DirSearchCompat           bool     `json:"dirsearch_compat,omitempty"`
	IgnoreWordlistComments    bool     `json:"ignore_wordlist_comments,omitempty"`
	Encoders                  []string `json:"encoders,omitempty"`
	MatcherMode               string   `json:"matcher_mode,omitempty"`
	MatcherStatus             string   `json:"matcher_status,omitempty"`
	MatcherSize               string   `json:"matcher_size,omitempty"`
	MatcherWords              string   `json:"matcher_words,omitempty"`
	MatcherLines              string   `json:"matcher_lines,omitempty"`
	MatcherRegexp             string   `json:"matcher_regexp,omitempty"`
	MatcherTime               string   `json:"matcher_time,omitempty"`
	FilterMode                string   `json:"filter_mode,omitempty"`
	FilterStatus              string   `json:"filter_status,omitempty"`
	FilterSize                string   `json:"filter_size,omitempty"`
	FilterWords               string   `json:"filter_words,omitempty"`
	FilterLines               string   `json:"filter_lines,omitempty"`
	FilterRegexp              string   `json:"filter_regexp,omitempty"`
	FilterTime                string   `json:"filter_time,omitempty"`
	AutoCalibrationPerHost    bool     `json:"auto_calibration_per_host,omitempty"`
	AutoCalibrationKeyword    string   `json:"auto_calibration_keyword,omitempty"`
	AutoCalibrationStrategies []string `json:"auto_calibration_strategies,omitempty"`
	AutoCalibrationStrings    []string `json:"auto_calibration_strings,omitempty"`

	// Phase 2: Recursion & Raw Request File
	Recursion         bool   `json:"recursion,omitempty"`
	RecursionDepth    int    `json:"recursion_depth,omitempty"`
	RecursionStrategy string `json:"recursion_strategy,omitempty"`
	RequestFile       string `json:"request_file,omitempty"`
	RequestProto      string `json:"request_proto,omitempty"`

	// Phase 3: Advanced TLS, Command Input, Scrapers, Audit/Output & Preflights
	SNI              string                      `json:"sni,omitempty"`
	ClientCert       string                      `json:"client_cert,omitempty"`
	ClientKey        string                      `json:"client_key,omitempty"`
	InputCommands    []string                    `json:"input_commands,omitempty"`
	InputNum         int                         `json:"input_num,omitempty"`
	InputShell       string                      `json:"input_shell,omitempty"`
	Scrapers         string                      `json:"scrapers,omitempty"`
	ScraperFile      string                      `json:"scraper_file,omitempty"`
	AuditLog         string                      `json:"audit_log,omitempty"`
	DebugLog         string                      `json:"debug_log,omitempty"`
	OutputDirectory  string                      `json:"output_directory,omitempty"`
	Preflights       []fuzznexus.PreflightConfig `json:"preflights,omitempty"`
	Postflights      []fuzznexus.PreflightConfig `json:"postflights,omitempty"`
	PreflightMode    string                      `json:"preflight_mode,omitempty"`
	PreflightError   string                      `json:"preflight_error,omitempty"`
	PreflightAnyHost bool                        `json:"preflight_anyhost,omitempty"`
}

// ParsedRawHTTPRequestDTO represents a raw HTTP request (e.g. from Burp/Caido) parsed into UI fields.
type ParsedRawHTTPRequestDTO struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Data    string            `json:"data"`
}

// PendingResumeDTO summarizes an interrupted scan saved on disk for cross-restart recovery.
type PendingResumeDTO struct {
	SessionID      int64     `json:"session_id"`
	TargetURL      string    `json:"target_url"`
	Method         string    `json:"method"`
	Wordlist       string    `json:"wordlist"`
	BrowserProfile string    `json:"browser_profile"`
	TLSProfile     string    `json:"tls_profile"`
	CurrentIndex   int       `json:"current_index"`
	IPHash         string    `json:"ip_hash"`
	Timestamp      time.Time `json:"timestamp"`
}

// ResultsPageDTO represents a chunk of paginated results returned for UI consumption.
type ResultsPageDTO struct {
	Results []output.ResultDTO `json:"results"`
	HasMore bool               `json:"has_more"`
	LastID  int64              `json:"last_id"`
}

// FilteredResultsPageDTO represents paginated results matching a filter along with total count.
type FilteredResultsPageDTO struct {
	Results      []output.ResultDTO `json:"results"`
	TotalMatches int64              `json:"total_matches"`
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
	wordlists     *wordlists.Manager
	activeSession *scan.Session
	lastErrors    []scan.NetworkErrorDTO
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
		dataDir = filepath.Join(home, ".fuzznexus")
		legacyDir := filepath.Join(home, ".ffuf-desktop")
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			if _, err := os.Stat(legacyDir); err == nil {
				// Migrate legacy directory to new location
				_ = os.Rename(legacyDir, dataDir)
			}
		}
	}

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create app data directory: %w", err)
	}

	// Ensure FuzzNexus configuration directories (autocalibration strategies, scraper dir) exist on clean machines.
	if err := fuzznexus.CheckOrCreateConfigDir(); err != nil {
		logger.Warn("Failed to initialize default FuzzNexus config directories", "err", err)
	}

	st, err := store.NewStore(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SQLite store: %w", err)
	}

	// Auto-heal and migrate any legacy paths in SQLite store (.ffuf-desktop -> .fuzznexus)
	_, _ = st.WriterDB().Exec(`
		UPDATE wordlists SET path = REPLACE(path, '/.fuzznexus-desktop/', '/.fuzznexus/') WHERE path LIKE '%/.fuzznexus-desktop/%';
		UPDATE sessions SET config_json = REPLACE(config_json, '/.fuzznexus-desktop/', '/.fuzznexus/') WHERE config_json LIKE '%/.fuzznexus-desktop/%';
	`)

	bl, err := blobs.NewStorage(dataDir, blobs.DefaultSessionQuota)
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("failed to initialize blob storage: %w", err)
	}

	wm, err := wordlists.NewManager(dataDir, st.WriterDB())
	if err != nil {
		_ = bl.Close()
		_ = st.Close()
		return nil, fmt.Errorf("failed to initialize wordlists manager: %w", err)
	}

	return &Service{
		store:     st,
		blobs:     bl,
		wordlists: wm,
		dataDir:   dataDir,
		logger:    logger,
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

// extractWordlistPath returns the file path portion of a wordlist spec (stripping optional :KEYWORD).
func extractWordlistPath(spec string) string {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" || trimmed == "-" {
		return trimmed
	}
	if _, err := os.Stat(trimmed); err == nil {
		return trimmed
	}
	// Handle path:KEYWORD (including Windows drive letter C:\path\to\file.txt:KEY)
	idx := strings.LastIndex(trimmed, ":")
	if idx > 0 {
		if runtime.GOOS == "windows" && idx == 1 {
			return trimmed
		}
		candidate := trimmed[:idx]
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return candidate
	}
	return trimmed
}

func validateProxyURL(raw, label string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid %s %q: must include scheme and host (e.g. http://127.0.0.1:8080)", label, trimmed)
	}
	return trimmed, nil
}

func applyMatchersAndFilters(opts *fuzznexus.ConfigOptions, req ScanRequestDTO) bool {
	statusSet := false
	matcherSet := false

	matcherStatus := strings.TrimSpace(req.MatcherStatus)
	if matcherStatus == "" {
		matcherStatus = strings.TrimSpace(req.Matchers)
	}
	if matcherStatus != "" {
		opts.Matcher.Status = matcherStatus
		statusSet = true
	}
	if strings.TrimSpace(req.MatcherSize) != "" {
		opts.Matcher.Size = strings.TrimSpace(req.MatcherSize)
		matcherSet = true
	}
	if strings.TrimSpace(req.MatcherWords) != "" {
		opts.Matcher.Words = strings.TrimSpace(req.MatcherWords)
		matcherSet = true
	}
	if strings.TrimSpace(req.MatcherLines) != "" {
		opts.Matcher.Lines = strings.TrimSpace(req.MatcherLines)
		matcherSet = true
	}
	if strings.TrimSpace(req.MatcherRegexp) != "" {
		opts.Matcher.Regexp = strings.TrimSpace(req.MatcherRegexp)
		matcherSet = true
	}
	if strings.TrimSpace(req.MatcherTime) != "" {
		opts.Matcher.Time = strings.TrimSpace(req.MatcherTime)
		matcherSet = true
	}
	if mMode := strings.ToLower(strings.TrimSpace(req.MatcherMode)); mMode == "and" || mMode == "or" {
		opts.Matcher.Mode = mMode
	}

	var statusFilters []string
	if strings.TrimSpace(req.Filters) != "" {
		statusFilters = append(statusFilters, strings.TrimSpace(req.Filters))
	}
	if strings.TrimSpace(req.FilterStatus) != "" && strings.TrimSpace(req.FilterStatus) != strings.TrimSpace(req.Filters) {
		statusFilters = append(statusFilters, strings.TrimSpace(req.FilterStatus))
	}
	if len(statusFilters) > 0 {
		opts.Filter.Status = strings.Join(statusFilters, ",")
	}
	if strings.TrimSpace(req.FilterSize) != "" {
		opts.Filter.Size = strings.TrimSpace(req.FilterSize)
	}
	if strings.TrimSpace(req.FilterWords) != "" {
		opts.Filter.Words = strings.TrimSpace(req.FilterWords)
	}
	if strings.TrimSpace(req.FilterLines) != "" {
		opts.Filter.Lines = strings.TrimSpace(req.FilterLines)
	}
	if strings.TrimSpace(req.FilterRegexp) != "" {
		opts.Filter.Regexp = strings.TrimSpace(req.FilterRegexp)
	}
	if strings.TrimSpace(req.FilterTime) != "" {
		opts.Filter.Time = strings.TrimSpace(req.FilterTime)
	}
	if fMode := strings.ToLower(strings.TrimSpace(req.FilterMode)); fMode == "and" || fMode == "or" {
		opts.Filter.Mode = fMode
	}

	return statusSet || !matcherSet
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

	req.RequestFile = strings.TrimSpace(req.RequestFile)
	if req.RequestFile != "" {
		if _, err := os.Stat(req.RequestFile); err != nil {
			return 0, fmt.Errorf("raw request file not accessible: %w", err)
		}
	}

	if strings.TrimSpace(req.URL) == "" && req.RequestFile == "" {
		return 0, errors.New("target URL cannot be empty")
	}

	// Resolve input providers first so NormalizeTargetWithOptions knows all custom keywords / sniper mode
	var effectiveWordlists []string
	var lockedWordlists []string
	var activeKeywords []string
	if len(req.InputCommands) == 0 {
		for _, w := range req.Wordlists {
			if strings.TrimSpace(w) != "" {
				effectiveWordlists = append(effectiveWordlists, strings.TrimSpace(w))
			}
		}
		if len(effectiveWordlists) == 0 && strings.TrimSpace(req.Wordlist) != "" {
			effectiveWordlists = append(effectiveWordlists, strings.TrimSpace(req.Wordlist))
		}
		if len(effectiveWordlists) == 0 {
			return 0, errors.New("wordlist path cannot be empty")
		}
		for _, wlSpec := range effectiveWordlists {
			wlPath := extractWordlistPath(wlSpec)
			if wlPath != "-" {
				if _, err := os.Stat(wlPath); err != nil {
					return 0, fmt.Errorf("wordlist file not accessible: %w", err)
				}
				lockedWordlists = append(lockedWordlists, wlPath)
			}
			if len(wlSpec) > len(wlPath)+1 && wlSpec[len(wlPath)] == ':' {
				activeKeywords = append(activeKeywords, wlSpec[len(wlPath)+1:])
			}
		}
	}

	if strings.TrimSpace(req.URL) != "" {
		norm, err := scan.NormalizeTargetWithOptions(req.URL, req.Headers, req.Data, req.Method, req.InputMode, activeKeywords)
		if err != nil {
			return 0, fmt.Errorf("invalid target URL: %w", err)
		}
		req.URL = norm.Normalized
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
	opts := fuzznexus.NewConfigOptions()
	opts.HTTP.URL = req.URL
	opts.Input.Wordlists = effectiveWordlists
	if len(req.InputCommands) > 0 {
		opts.Input.Inputcommands = req.InputCommands
		if req.InputNum > 0 {
			opts.Input.InputNum = req.InputNum
		}
		if strings.TrimSpace(req.InputShell) != "" {
			opts.Input.InputShell = strings.TrimSpace(req.InputShell)
		}
	}
	if mode := strings.ToLower(strings.TrimSpace(req.InputMode)); mode != "" {
		opts.Input.InputMode = mode
	}
	opts.Input.Extensions = strings.TrimSpace(req.Extensions)
	opts.Input.DirSearchCompat = req.DirSearchCompat
	opts.Input.IgnoreWordlistComments = req.IgnoreWordlistComments
	if len(req.Encoders) > 0 {
		opts.Input.Encoders = req.Encoders
	}
	if req.RequestFile != "" {
		opts.Input.Request = req.RequestFile
		if strings.TrimSpace(req.RequestProto) != "" {
			opts.Input.RequestProto = strings.TrimSpace(req.RequestProto)
		}
	}

	if req.Threads > 0 {
		opts.General.Threads = req.Threads
	} else {
		opts.General.Threads = 10
	}
	opts.General.Rate = req.Rate
	opts.General.Quiet = true
	opts.General.Noninteractive = true
	opts.General.Delay = strings.TrimSpace(req.Delay)
	opts.General.MaxTime = req.MaxTime
	opts.General.MaxTimeJob = req.MaxTimeJob
	opts.General.StopOn403 = req.StopOn403
	opts.General.StopOnErrors = req.StopOnErrors
	opts.General.StopOnAll = req.StopOnAll

	opts.HTTP.Method = req.Method
	if opts.HTTP.Method == "" && req.RequestFile == "" {
		opts.HTTP.Method = "GET"
	}
	opts.HTTP.Data = req.Data
	opts.HTTP.FollowRedirects = req.FollowRedirects
	if req.Timeout > 0 {
		opts.HTTP.Timeout = req.Timeout
	}
	opts.HTTP.Raw = req.Raw
	opts.HTTP.Http2 = req.Http2
	opts.HTTP.IgnoreBody = req.IgnoreBody
	opts.HTTP.CloseConnOnThrottle = req.CloseConnOnThrottle
	if req.DetectSoftBlock != nil {
		opts.HTTP.DetectSoftBlock = *req.DetectSoftBlock
	} else {
		opts.HTTP.DetectSoftBlock = true
	}
	opts.HTTP.Recursion = req.Recursion
	opts.HTTP.RecursionDepth = req.RecursionDepth
	if rs := strings.ToLower(strings.TrimSpace(req.RecursionStrategy)); rs != "" {
		opts.HTTP.RecursionStrategy = rs
	}
	opts.HTTP.SNI = strings.TrimSpace(req.SNI)
	opts.HTTP.ClientCert = strings.TrimSpace(req.ClientCert)
	opts.HTTP.ClientKey = strings.TrimSpace(req.ClientKey)
	if len(req.Preflights) > 0 {
		opts.HTTP.Preflights = req.Preflights
	}
	if len(req.Postflights) > 0 {
		opts.HTTP.Postflights = req.Postflights
	}
	if pm := strings.TrimSpace(req.PreflightMode); pm != "" {
		opts.HTTP.PreflightMode = pm
	}
	if pe := strings.TrimSpace(req.PreflightError); pe != "" {
		opts.HTTP.PreflightError = pe
	}
	opts.HTTP.PreflightAnyHost = req.PreflightAnyHost

	// Auto-Calibration options
	opts.General.AutoCalibration = req.AutoCalibration
	opts.General.AutoCalibrationPerHost = req.AutoCalibrationPerHost
	if ack := strings.TrimSpace(req.AutoCalibrationKeyword); ack != "" {
		opts.General.AutoCalibrationKeyword = ack
	}
	if len(req.AutoCalibrationStrategies) > 0 {
		opts.General.AutoCalibrationStrategies = req.AutoCalibrationStrategies
	}
	if len(req.AutoCalibrationStrings) > 0 {
		opts.General.AutoCalibrationStrings = req.AutoCalibrationStrings
	}

	// Scrapers & Forensics Output
	if sc := strings.TrimSpace(req.Scrapers); sc != "" {
		opts.General.Scrapers = sc
	}
	opts.General.ScraperFile = strings.TrimSpace(req.ScraperFile)
	opts.Output.AuditLog = strings.TrimSpace(req.AuditLog)
	opts.Output.DebugLog = strings.TrimSpace(req.DebugLog)
	opts.Output.OutputDirectory = strings.TrimSpace(req.OutputDirectory)

	proxyURL, err := validateProxyURL(req.ProxyURL, "proxy URL")
	if err != nil {
		cancel()
		return 0, err
	}
	opts.HTTP.ProxyURL = proxyURL

	replayProxyURL, err := validateProxyURL(req.ReplayProxyURL, "replay proxy URL")
	if err != nil {
		cancel()
		return 0, err
	}
	opts.HTTP.ReplayProxyURL = replayProxyURL

	// Add custom headers
	for k, v := range req.Headers {
		opts.HTTP.Headers = append(opts.HTTP.Headers, fmt.Sprintf("%s: %s", k, v))
	}

	// Matcher & filter options (full parity with the shared CLI engine)
	addDefaultStatusMatcher := applyMatchersAndFilters(opts, req)

	conf, err := fuzznexus.ConfigFromOptions(opts, scanCtx, cancel)
	if err != nil {
		cancel()
		return 0, fmt.Errorf("failed to parse FuzzNexus configuration: %w", err)
	}
	conf.BrowserProfile = req.BrowserProfile
	conf.RotateBrowserProfile = req.RotateOnBlock
	conf.TLSProfile = strings.TrimSpace(req.TLSProfile)
	if req.ResumeFromIndex > 0 {
		conf.StartIndex = req.ResumeFromIndex
	}

	// Install matcher manager
	mm, err := filter.FromConfig(opts, addDefaultStatusMatcher)
	if err != nil {
		cancel()
		return 0, fmt.Errorf("failed to construct matcher manager: %w", err)
	}
	conf.MatcherManager = mm

	unlockAll := func() {
		for _, p := range lockedWordlists {
			s.wordlists.UnlockWordlist(p)
		}
	}

	sess, err := scan.NewSession(scan.SessionOptions{
		SessionID:      sessionID,
		Config:         conf,
		Store:          s.store,
		Blobs:          s.blobs,
		Logger:         s.logger,
		ResumeFilePath: s.resumeFilePath(),
		CancelCtx:      cancel,
		EmitCommit: func(e output.CommittedEvent) {
			s.emitEvent("results:committed", e)
		},
		EmitProgress: func(p scan.ProgressDTO) {
			s.emitEvent("scan:progress", p)
		},
		EmitState: func(st scan.State, errMsg string) {
			if st == scan.StateCompleted || st == scan.StateFailed || st == scan.StateStopped {
				unlockAll()
			}
			s.emitEvent("scan:state", StateEvent{
				SessionID: sessionID,
				State:     st,
				Error:     errMsg,
			})
		},
		EmitSafeStop: func(prompt scan.SafeStopPromptDTO) {
			s.emitEvent("scan:safe_stop_prompt", prompt)
		},
		EmitAdvisory: func(alert scan.AdvisoryAlertDTO) {
			s.emitEvent("scan:advisory_alert", alert)
		},
		EmitError: func(code string, msg string) {
			s.emitEvent("scan:error", msg)
		},
		EmitCircuitState: func(state string, reason string) {
			s.emitEvent("scan:circuit_state", map[string]string{
				"session_id": fmt.Sprintf("%d", sessionID),
				"state":      state,
				"reason":     reason,
			})
		},
	})
	if err != nil {
		cancel()
		return 0, fmt.Errorf("failed to create scan session: %w", err)
	}

	for _, p := range lockedWordlists {
		s.wordlists.LockWordlist(p)
	}

	if err := sess.Start(); err != nil {
		unlockAll()
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

// GetRecentErrors returns the recent network diagnostic errors for the active or most recent session.
func (s *Service) GetRecentErrors() []scan.NetworkErrorDTO {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.activeSession != nil {
		errs := s.activeSession.GetRecentErrors()
		if len(errs) > 0 {
			s.lastErrors = errs
		}
		return errs
	}
	if s.lastErrors != nil {
		return s.lastErrors
	}
	return []scan.NetworkErrorDTO{}
}

// TestConnection tests reachability with new IP while paused in Safe Stop.
func (s *Service) TestConnection() (scan.SafeStopTestResultDTO, error) {
	s.sessionMu.Lock()
	sess := s.activeSession
	s.sessionMu.Unlock()
	if sess == nil {
		return scan.SafeStopTestResultDTO{Success: false, Error: "no active scan session"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return sess.TestConnection(ctx)
}

// ResumeSafeStop unpauses scanning after IP change verification.
func (s *Service) ResumeSafeStop() error {
	s.sessionMu.Lock()
	sess := s.activeSession
	s.sessionMu.Unlock()
	if sess == nil {
		return errors.New("no active scan session")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return sess.ResumeSafeStop(ctx)
}

// ResumeAnyway forces resumption even if IP hasn't changed.
func (s *Service) ResumeAnyway() error {
	s.sessionMu.Lock()
	sess := s.activeSession
	s.sessionMu.Unlock()
	if sess == nil {
		return errors.New("no active scan session")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return sess.ResumeAnyway(ctx)
}

// CancelSafeStop cancels execution and saves resume state.
func (s *Service) CancelSafeStop() error {
	s.sessionMu.Lock()
	sess := s.activeSession
	s.sessionMu.Unlock()
	if sess == nil {
		return nil
	}
	return sess.CancelSafeStop()
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

// DeleteSessions deletes selected historical sessions (skipping any currently active scan).
func (s *Service) DeleteSessions(ids []int64) (int64, error) {
	s.sessionMu.Lock()
	var activeID int64
	if s.activeSession != nil {
		state, _ := s.activeSession.State()
		if state == scan.StateRunning || state == scan.StatePaused || state == scan.StateStopping {
			activeID = s.activeSession.ID()
		}
	}
	s.sessionMu.Unlock()

	filtered := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && id != activeID {
			filtered = append(filtered, id)
		}
	}
	return s.store.DeleteSessions(filtered)
}

// ClearSessions removes all historical scan sessions except any currently active scan.
func (s *Service) ClearSessions() (int64, error) {
	s.sessionMu.Lock()
	var activeID int64
	if s.activeSession != nil {
		state, _ := s.activeSession.State()
		if state == scan.StateRunning || state == scan.StatePaused || state == scan.StateStopping {
			activeID = s.activeSession.ID()
		}
	}
	s.sessionMu.Unlock()

	return s.store.ClearSessions(activeID)
}

// GetResultsFiltered queries results matching a filter specification.
func (s *Service) GetResultsFiltered(sessionID int64, group store.FilterGroup, limit int, offset int) (FilteredResultsPageDTO, error) {
	results, total, err := s.store.GetResultsFiltered(sessionID, group, limit, offset)
	if err != nil {
		return FilteredResultsPageDTO{}, err
	}
	return FilteredResultsPageDTO{
		Results:      results,
		TotalMatches: total,
	}, nil
}

// SaveFilterPreset saves or updates a named filter preset in SQLite.
func (s *Service) SaveFilterPreset(name, filterJSON string) error {
	return s.store.SaveFilterPreset(name, filterJSON)
}

// GetFilterPresets retrieves all saved filter presets from SQLite.
func (s *Service) GetFilterPresets() ([]store.FilterPresetDTO, error) {
	return s.store.GetFilterPresets()
}

// DeleteFilterPreset removes a filter preset by name.
func (s *Service) DeleteFilterPreset(name string) error {
	return s.store.DeleteFilterPreset(name)
}

// ExportResults exports scan results to disk in JSON, CSV, TXT, HTML, or Markdown format.
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

	formatLower := strings.ToLower(strings.TrimSpace(format))
	switch formatLower {
	case "csv":
		w := csv.NewWriter(f)
		defer w.Flush()

		_ = w.Write([]string{"ID", "Position", "Payloads", "Status", "Length", "Words", "Lines", "DurationMs", "URL", "Redirect", "ContentType", "Scraper", "Hash"})
		for _, r := range results {
			_ = w.Write([]string{
				strconv.FormatInt(r.ID, 10),
				strconv.Itoa(r.Position),
				formatSortedMap(r.Inputs),
				strconv.FormatInt(r.Status, 10),
				strconv.FormatInt(r.Length, 10),
				strconv.FormatInt(r.Words, 10),
				strconv.FormatInt(r.Lines, 10),
				strconv.FormatInt(r.DurationMs, 10),
				r.URL,
				r.RedirectLocation,
				r.ContentType,
				formatSortedMap(r.ScraperData),
				r.Hash,
			})
		}
		return nil

	case "txt":
		w := bufio.NewWriter(f)
		defer w.Flush()

		for _, r := range results {
			redirectPart := ""
			if r.RedirectLocation != "" {
				redirectPart = fmt.Sprintf(" -> %s", r.RedirectLocation)
			}
			line := fmt.Sprintf("[%d] %s%s (Length: %d, Words: %d, Lines: %d, Duration: %dms)\n",
				r.Status, r.URL, redirectPart, r.Length, r.Words, r.Lines, r.DurationMs)
			if _, err := w.WriteString(line); err != nil {
				return fmt.Errorf("failed writing txt export line: %w", err)
			}
		}
		return nil

	case "md", "markdown":
		return writeMarkdownExport(f, sessionID, results)

	case "html":
		return writeHTMLExport(f, sessionID, results)

	case "ejson":
		encoded := make([]output.ResultDTO, 0, len(results))
		for _, r := range results {
			copyR := r
			copyR.URL = base64.StdEncoding.EncodeToString([]byte(r.URL))
			if r.RedirectLocation != "" {
				copyR.RedirectLocation = base64.StdEncoding.EncodeToString([]byte(r.RedirectLocation))
			}
			encInputs := make(map[string]string, len(r.Inputs))
			for k, v := range r.Inputs {
				encInputs[k] = base64.StdEncoding.EncodeToString([]byte(v))
			}
			copyR.Inputs = encInputs
			encoded = append(encoded, copyR)
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		return enc.Encode(encoded)

	case "ecsv":
		w := csv.NewWriter(f)
		defer w.Flush()
		_ = w.Write([]string{"ID", "Position", "Payloads_B64", "Status", "Length", "Words", "Lines", "DurationMs", "URL_B64", "Redirect_B64", "ContentType", "Scraper", "Hash"})
		for _, r := range results {
			_ = w.Write([]string{
				strconv.FormatInt(r.ID, 10),
				strconv.Itoa(r.Position),
				base64.StdEncoding.EncodeToString([]byte(formatSortedMap(r.Inputs))),
				strconv.FormatInt(r.Status, 10),
				strconv.FormatInt(r.Length, 10),
				strconv.FormatInt(r.Words, 10),
				strconv.FormatInt(r.Lines, 10),
				strconv.FormatInt(r.DurationMs, 10),
				base64.StdEncoding.EncodeToString([]byte(r.URL)),
				base64.StdEncoding.EncodeToString([]byte(r.RedirectLocation)),
				r.ContentType,
				formatSortedMap(r.ScraperData),
				r.Hash,
			})
		}
		return nil

	default:
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
}

func formatSortedMap(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, m[k]))
	}
	return strings.Join(parts, "; ")
}

func writeMarkdownExport(f *os.File, sessionID int64, results []output.ResultDTO) error {
	w := bufio.NewWriter(f)
	defer w.Flush()
	escMD := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ")
	}
	if _, err := fmt.Fprintf(w, "# FuzzNexus Scan Report (Session #%d)\n\nGenerated: %s | Total Results: %d\n\n",
		sessionID, time.Now().UTC().Format(time.RFC3339), len(results)); err != nil {
		return err
	}
	if _, err := w.WriteString("| # | Status | Payloads | URL | Size | Words | Lines | Duration | Redirect | Scraper |\n| -: | -: | :-- | :-- | -: | -: | -: | -: | :-- | :-- |\n"); err != nil {
		return err
	}
	for _, r := range results {
		if _, err := fmt.Fprintf(w, "| %d | %d | %s | %s | %d | %d | %d | %dms | %s | %s |\n",
			r.ID, r.Status, escMD(formatSortedMap(r.Inputs)), escMD(r.URL),
			r.Length, r.Words, r.Lines, r.DurationMs, escMD(r.RedirectLocation), escMD(formatSortedMap(r.ScraperData))); err != nil {
			return err
		}
	}
	return nil
}

func writeHTMLExport(f *os.File, sessionID int64, results []output.ResultDTO) error {
	w := bufio.NewWriter(f)
	defer w.Flush()
	if _, err := fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>FuzzNexus Report - Session #%d</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 24px; background: #0f1117; color: #e5e7eb; }
  h1 { font-size: 20px; margin-bottom: 4px; }
  .meta { color: #9ca3af; font-size: 13px; margin-bottom: 16px; }
  input#q { width: 320px; padding: 8px 10px; border-radius: 6px; border: 1px solid #374151; background: #181b23; color: #f3f4f6; margin-bottom: 14px; }
  table { width: 100%%; border-collapse: collapse; font-size: 13px; }
  th, td { text-align: left; padding: 8px 10px; border-bottom: 1px solid #262a36; word-break: break-word; }
  th { background: #161922; color: #9ca3af; font-weight: 600; position: sticky; top: 0; }
  .badge { display: inline-block; padding: 2px 7px; border-radius: 4px; font-family: monospace; font-weight: 600; }
  .s2 { background: rgba(16,185,129,0.18); color: #34d399; }
  .s3 { background: rgba(59,130,246,0.18); color: #60a5fa; }
  .s4 { background: rgba(245,158,11,0.18); color: #fbbf24; }
  .s5 { background: rgba(239,68,68,0.18); color: #f87171; }
  a { color: #60a5fa; text-decoration: none; }
  a:hover { text-decoration: underline; }
  code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
</style>
</head>
<body>
<h1>FuzzNexus Scan Report &mdash; Session #%d</h1>
<div class="meta">Generated %s &bull; %d matched results</div>
<input id="q" type="text" placeholder="Filter rows..." oninput="for(const tr of document.querySelectorAll('#tbl tbody tr')){tr.style.display=tr.textContent.toLowerCase().includes(this.value.toLowerCase())?'':'none';}" />
<table id="tbl">
<thead><tr><th>#</th><th>Status</th><th>Payloads</th><th>URL</th><th>Size</th><th>Words</th><th>Lines</th><th>Duration</th><th>Redirect / Scraper</th></tr></thead>
<tbody>
`, sessionID, sessionID, html.EscapeString(time.Now().UTC().Format(time.RFC3339)), len(results)); err != nil {
		return err
	}
	for _, r := range results {
		cls := "s4"
		switch {
		case r.Status >= 200 && r.Status < 300:
			cls = "s2"
		case r.Status >= 300 && r.Status < 400:
			cls = "s3"
		case r.Status >= 500:
			cls = "s5"
		}
		extra := r.RedirectLocation
		if sc := formatSortedMap(r.ScraperData); sc != "" {
			if extra != "" {
				extra += " | "
			}
			extra += sc
		}
		if _, err := fmt.Fprintf(w, `<tr><td>%d</td><td><span class="badge %s">%d</span></td><td><code>%s</code></td><td><a href="%s" target="_blank" rel="noreferrer">%s</a></td><td>%d</td><td>%d</td><td>%d</td><td>%dms</td><td><code>%s</code></td></tr>`+"\n",
			r.ID, cls, r.Status,
			html.EscapeString(formatSortedMap(r.Inputs)),
			html.EscapeString(r.URL), html.EscapeString(r.URL),
			r.Length, r.Words, r.Lines, r.DurationMs,
			html.EscapeString(extra)); err != nil {
			return err
		}
	}
	_, err := w.WriteString("</tbody></table></body></html>\n")
	return err
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
	if s.wordlists != nil {
		if err := s.wordlists.CleanupAllSandboxes(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing bridge service: %v", errs)
	}
	return nil
}

// ListWordlists returns all registered wordlists.
func (s *Service) ListWordlists() ([]wordlists.WordlistDTO, error) {
	return s.wordlists.ListWordlists()
}

// CreateWordlist creates a new managed wordlist.
func (s *Service) CreateWordlist(name, initialContent string) (*wordlists.WordlistDTO, error) {
	return s.wordlists.CreateWordlist(name, initialContent)
}

// CreateWordlistWithCategory creates a new managed wordlist with a specific category.
func (s *Service) CreateWordlistWithCategory(name, initialContent, category string) (*wordlists.WordlistDTO, error) {
	return s.wordlists.CreateWordlistWithCategory(name, initialContent, category)
}

// GetWordlistCategories returns all wordlist categories.
func (s *Service) GetWordlistCategories() ([]wordlists.CategoryDTO, error) {
	return s.wordlists.GetCategories()
}

// CreateWordlistCategory creates a new category record.
func (s *Service) CreateWordlistCategory(id, name, color string) (*wordlists.CategoryDTO, error) {
	return s.wordlists.CreateCategory(id, name, color)
}

// UpdateWordlistCategory modifies an existing category's name and color.
func (s *Service) UpdateWordlistCategory(id, name, color string) error {
	return s.wordlists.UpdateCategory(id, name, color)
}

// DeleteWordlistCategory removes a custom category and reassigns its wordlists to 'general'.
func (s *Service) DeleteWordlistCategory(id string) error {
	return s.wordlists.DeleteCategory(id)
}

// UpdateWordlist renames a wordlist metadata record.
func (s *Service) UpdateWordlist(id int64, name string) error {
	return s.wordlists.UpdateWordlist(id, name)
}

// UpdateWordlistMetadata updates both name and category of a wordlist.
func (s *Service) UpdateWordlistMetadata(id int64, name string, category string) error {
	return s.wordlists.UpdateWordlistMetadata(id, name, category)
}

// MergeWordlists combines multiple wordlists into a single deduplicated new wordlist.
func (s *Service) MergeWordlists(ids []int64, targetName string, category string) (*wordlists.WordlistDTO, error) {
	return s.wordlists.MergeWordlists(ids, targetName, category)
}

// MergeWordlistsToSandbox combines multiple wordlists with high-precision sanitization into an isolated sandbox file or persistent library wordlist.
func (s *Service) MergeWordlistsToSandbox(ids []int64, opts wordlists.SanitizeOptions, isEphemeral bool, targetName string, category string) (*wordlists.WordlistDTO, error) {
	return s.wordlists.MergeWordlistsToSandbox(ids, opts, isEphemeral, targetName, category)
}

// PreviewMergeStats computes a dry-run analysis of combining multiple wordlists without writing output.
func (s *Service) PreviewMergeStats(ids []int64, opts wordlists.SanitizeOptions) (*wordlists.MergeReportDTO, error) {
	return s.wordlists.PreviewMergeStats(ids, opts)
}

// CleanupSandboxWordlist removes an unlocked sandbox wordlist file.
func (s *Service) CleanupSandboxWordlist(path string) error {
	return s.wordlists.CleanupSandboxWordlist(path)
}

// DeleteWordlist removes a wordlist and unlinks or deletes its file.
func (s *Service) DeleteWordlist(id int64) error {
	return s.wordlists.DeleteWordlist(id)
}

// DeleteWordlistsBatch removes multiple wordlists and unlinks or deletes their files.
func (s *Service) DeleteWordlistsBatch(ids []int64) (int, error) {
	return s.wordlists.DeleteWordlistsBatch(ids)
}

// ResetBuiltinWordlist restores a built-in wordlist back to its original factory state.
func (s *Service) ResetBuiltinWordlist(id int64) (*wordlists.WordlistDTO, error) {
	return s.wordlists.ResetBuiltinWordlist(id)
}

// RestoreAllBuiltinWordlists re-seeds any missing built-in wordlists without touching custom ones.
func (s *Service) RestoreAllBuiltinWordlists() ([]*wordlists.WordlistDTO, error) {
	return s.wordlists.RestoreAllBuiltinWordlists()
}

// NormalizeTargetURL parses and sanitizes raw user input into an analysis object with safe URL.
func (s *Service) NormalizeTargetURL(rawURL string) (*scan.TargetAnalysis, error) {
	return scan.NormalizeTarget(rawURL, nil, "")
}

// ReadWordlistPage reads a slice of lines for the virtual editor.
func (s *Service) ReadWordlistPage(id int64, offset int, limit int) (wordlists.WordlistPageDTO, error) {
	return s.wordlists.ReadWordlistPage(id, offset, limit)
}

// SaveWordlistContent saves new content to a wordlist atomically.
func (s *Service) SaveWordlistContent(id int64, content string) error {
	return s.wordlists.SaveWordlistContent(id, content)
}

// ImportWordlistFromPath copies an external file into the app wordlists store.
func (s *Service) ImportWordlistFromPath(sourcePath string) (*wordlists.WordlistDTO, error) {
	return s.wordlists.ImportWordlistFromPath(sourcePath)
}

// LinkWordlistPath links an external wordlist file in place without copying.
func (s *Service) LinkWordlistPath(sourcePath string) (*wordlists.WordlistDTO, error) {
	return s.wordlists.LinkWordlistPath(sourcePath)
}

// RelocateWordlist updates the file location of a linked wordlist.
func (s *Service) RelocateWordlist(id int64, newPath string) error {
	return s.wordlists.RelocateWordlist(id, newPath)
}

// GetWordlistStats computes analytical metrics for a wordlist.
func (s *Service) GetWordlistStats(id int64) (*wordlists.WordlistStatsDTO, error) {
	return s.wordlists.GetWordlistStats(id)
}

// ExportWordlist exports a wordlist to a destination file.
func (s *Service) ExportWordlist(id int64, targetPath string) error {
	return s.wordlists.ExportWordlist(id, targetPath)
}

// VerifyWordlist inspects a wordlist for blank lines and whitespace defects.
func (s *Service) VerifyWordlist(id int64) (*wordlists.WordlistHealthDTO, error) {
	return s.wordlists.VerifyWordlist(id)
}

// VerifyWordlistPath inspects any file path for blank lines, whitespace, and duplicates.
func (s *Service) VerifyWordlistPath(path string) (*wordlists.WordlistHealthDTO, error) {
	return wordlists.VerifyWordlistFile(path)
}

// CleanWordlist strips blank lines and trims whitespace from a wordlist.
func (s *Service) CleanWordlist(id int64) (*wordlists.CleanResultDTO, error) {
	return s.wordlists.CleanWordlist(id)
}

// CleanWordlistPath cleans blank lines, trims whitespace, and removes duplicates from any file path.
func (s *Service) CleanWordlistPath(path string) (*wordlists.CleanResultDTO, error) {
	return wordlists.CleanWordlistFile(path)
}

// AnalyzeWordlistRisk performs passive pre-scan WAF heuristic risk analysis on a wordlist file.
func (s *Service) AnalyzeWordlistRisk(path string) (*wordlists.WordlistRiskReportDTO, error) {
	return wordlists.AnalyzeWordlistRisks(path)
}

// CreateSanitizedSandboxWordlist builds a non-destructive ephemeral sandbox copy with traversal patterns removed.
func (s *Service) CreateSanitizedSandboxWordlist(sourcePath string) (*wordlists.WordlistDTO, error) {
	return s.wordlists.CreateSanitizedSandboxWordlist(sourcePath)
}

// SelectWordlistDialog displays a native file chooser dialog.
func (s *Service) SelectWordlistDialog(title string) (string, error) {
	s.wailsCtxMu.RLock()
	ctx := s.wailsCtx
	s.wailsCtxMu.RUnlock()

	if ctx == nil {
		return "", errors.New("native window context not available")
	}

	if title == "" {
		title = "Select Wordlist File"
	}

	return wailsRuntime.OpenFileDialog(ctx, wailsRuntime.OpenDialogOptions{
		Title: title,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Wordlists & Text Files (*.txt, *.dic, *.lst)", Pattern: "*.txt;*.dic;*.lst;*.*"},
		},
	})
}

// SaveWordlistDialog displays a native save file dialog.
func (s *Service) SaveWordlistDialog(defaultName string) (string, error) {
	s.wailsCtxMu.RLock()
	ctx := s.wailsCtx
	s.wailsCtxMu.RUnlock()

	if ctx == nil {
		return "", errors.New("native window context not available")
	}

	return wailsRuntime.SaveFileDialog(ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Export Wordlist File",
		DefaultFilename: defaultName,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
		},
	})
}

// SaveExportDialog displays a native save file dialog for exporting results.
func (s *Service) SaveExportDialog(defaultName, format string) (string, error) {
	s.wailsCtxMu.RLock()
	ctx := s.wailsCtx
	s.wailsCtxMu.RUnlock()

	if ctx == nil {
		return "", errors.New("native window context not available")
	}

	ext := strings.ToLower(format)
	pattern := "*." + ext
	displayName := fmt.Sprintf("%s Files (*.%s)", strings.ToUpper(ext), ext)

	return wailsRuntime.SaveFileDialog(ctx, wailsRuntime.SaveDialogOptions{
		Title:           fmt.Sprintf("Export Scan Results as %s", strings.ToUpper(ext)),
		DefaultFilename: defaultName,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: displayName, Pattern: pattern},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}

// SelectAuxiliaryFileDialog opens a native file chooser for certs, keys, raw requests, scrapers, or audit logs.
func (s *Service) SelectAuxiliaryFileDialog(title, filterName, filterPattern string) (string, error) {
	s.wailsCtxMu.RLock()
	ctx := s.wailsCtx
	s.wailsCtxMu.RUnlock()

	if ctx == nil {
		return "", errors.New("native window context not available")
	}
	if strings.TrimSpace(title) == "" {
		title = "Select File"
	}
	var filters []wailsRuntime.FileFilter
	if strings.TrimSpace(filterPattern) != "" {
		name := strings.TrimSpace(filterName)
		if name == "" {
			name = "Supported Files"
		}
		filters = append(filters, wailsRuntime.FileFilter{DisplayName: name, Pattern: filterPattern})
	}
	filters = append(filters, wailsRuntime.FileFilter{DisplayName: "All Files (*.*)", Pattern: "*.*"})

	return wailsRuntime.OpenFileDialog(ctx, wailsRuntime.OpenDialogOptions{
		Title:   title,
		Filters: filters,
	})
}

// SelectDirectoryDialog opens a native directory chooser (e.g. for -od Output Directory).
func (s *Service) SelectDirectoryDialog(title string) (string, error) {
	s.wailsCtxMu.RLock()
	ctx := s.wailsCtx
	s.wailsCtxMu.RUnlock()

	if ctx == nil {
		return "", errors.New("native window context not available")
	}
	if strings.TrimSpace(title) == "" {
		title = "Select Directory"
	}
	return wailsRuntime.OpenDirectoryDialog(ctx, wailsRuntime.OpenDialogOptions{
		Title: title,
	})
}

// GetAvailableEncoders returns all encoder names supported by the underlying pencode engine.
func (s *Service) GetAvailableEncoders() []string {
	return input.AvailableEncoders()
}

// ImportTOMLConfig reads a standard ffuf TOML config file (-config / .fuzznexusrc) into a ScanRequestDTO.
func (s *Service) ImportTOMLConfig(path string) (*ScanRequestDTO, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, errors.New("config path cannot be empty")
	}
	opts, err := fuzznexus.ReadConfig(trimmed)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	headers := make(map[string]string)
	for _, h := range opts.HTTP.Headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	primaryWL := ""
	if len(opts.Input.Wordlists) > 0 {
		primaryWL = opts.Input.Wordlists[0]
	}
	detectSB := opts.HTTP.DetectSoftBlock
	dto := &ScanRequestDTO{
		URL:                       opts.HTTP.URL,
		Method:                    opts.HTTP.Method,
		Headers:                   headers,
		Data:                      opts.HTTP.Data,
		Wordlist:                  primaryWL,
		Wordlists:                 opts.Input.Wordlists,
		InputMode:                 opts.Input.InputMode,
		Threads:                   opts.General.Threads,
		Rate:                      opts.General.Rate,
		Timeout:                   opts.HTTP.Timeout,
		FollowRedirects:           opts.HTTP.FollowRedirects,
		AutoCalibration:           opts.General.AutoCalibration,
		AutoCalibrationPerHost:    opts.General.AutoCalibrationPerHost,
		AutoCalibrationKeyword:    opts.General.AutoCalibrationKeyword,
		AutoCalibrationStrategies: opts.General.AutoCalibrationStrategies,
		AutoCalibrationStrings:    opts.General.AutoCalibrationStrings,
		ProxyURL:                  opts.HTTP.ProxyURL,
		ReplayProxyURL:            opts.HTTP.ReplayProxyURL,
		TLSProfile:                opts.HTTP.TLSProfile,
		Raw:                       opts.HTTP.Raw,
		Http2:                     opts.HTTP.Http2,
		IgnoreBody:                opts.HTTP.IgnoreBody,
		CloseConnOnThrottle:       opts.HTTP.CloseConnOnThrottle,
		DetectSoftBlock:           &detectSB,
		Delay:                     opts.General.Delay,
		MaxTime:                   opts.General.MaxTime,
		MaxTimeJob:                opts.General.MaxTimeJob,
		StopOn403:                 opts.General.StopOn403,
		StopOnErrors:              opts.General.StopOnErrors,
		StopOnAll:                 opts.General.StopOnAll,
		Extensions:                opts.Input.Extensions,
		DirSearchCompat:           opts.Input.DirSearchCompat,
		IgnoreWordlistComments:    opts.Input.IgnoreWordlistComments,
		Encoders:                  opts.Input.Encoders,
		MatcherMode:               opts.Matcher.Mode,
		MatcherStatus:             opts.Matcher.Status,
		MatcherSize:               opts.Matcher.Size,
		MatcherWords:              opts.Matcher.Words,
		MatcherLines:              opts.Matcher.Lines,
		MatcherRegexp:             opts.Matcher.Regexp,
		MatcherTime:               opts.Matcher.Time,
		FilterMode:                opts.Filter.Mode,
		FilterStatus:              opts.Filter.Status,
		FilterSize:                opts.Filter.Size,
		FilterWords:               opts.Filter.Words,
		FilterLines:               opts.Filter.Lines,
		FilterRegexp:              opts.Filter.Regexp,
		FilterTime:                opts.Filter.Time,
		Recursion:                 opts.HTTP.Recursion,
		RecursionDepth:            opts.HTTP.RecursionDepth,
		RecursionStrategy:         opts.HTTP.RecursionStrategy,
		RequestFile:               opts.Input.Request,
		RequestProto:              opts.Input.RequestProto,
		SNI:                       opts.HTTP.SNI,
		ClientCert:                opts.HTTP.ClientCert,
		ClientKey:                 opts.HTTP.ClientKey,
		InputCommands:             opts.Input.Inputcommands,
		InputNum:                  opts.Input.InputNum,
		InputShell:                opts.Input.InputShell,
		Scrapers:                  opts.General.Scrapers,
		ScraperFile:               opts.General.ScraperFile,
		AuditLog:                  opts.Output.AuditLog,
		DebugLog:                  opts.Output.DebugLog,
		OutputDirectory:           opts.Output.OutputDirectory,
		Preflights:                opts.HTTP.Preflights,
		Postflights:               opts.HTTP.Postflights,
		PreflightMode:             opts.HTTP.PreflightMode,
		PreflightError:            opts.HTTP.PreflightError,
		PreflightAnyHost:          opts.HTTP.PreflightAnyHost,
	}
	return dto, nil
}

// SearchFNHASH reproduces the raw HTTP request for a FNHASH from the local history store (-search).
func (s *Service) SearchFNHASH(hash string) (string, error) {
	trimmed := strings.TrimSpace(hash)
	if trimmed == "" {
		return "", errors.New("FNHASH cannot be empty")
	}
	coptions, pos, err := engine.SearchHash(trimmed)
	if err != nil {
		return "", err
	}
	if len(coptions) == 0 {
		return "", fmt.Errorf("no history record found for hash %q", trimmed)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var sb strings.Builder
	for _, copt := range coptions {
		conf, cerr := fuzznexus.ConfigFromOptions(&copt.ConfigOptions, ctx, cancel)
		if cerr != nil {
			continue
		}
		ok, reason := engine.HistoryReplayable(conf)
		if !ok {
			sb.WriteString(fmt.Sprintf("[ERR] Hash cannot be replayed: %s\n", reason))
			continue
		}
		inp, merr := input.NewInputProvider(conf)
		if merr.ErrorOrNil() != nil {
			sb.WriteString(fmt.Sprintf("[ERR] Input provider error: %v\n", merr.ErrorOrNil()))
			continue
		}
		inp.SetPosition(pos)
		inputdata := inp.Value()
		inputdata["FNHASH"] = []byte(trimmed)
		basereq := fuzznexus.BaseRequest(conf)
		r := runner.NewSimpleRunner(conf, false)
		ffufreq, _ := r.Prepare(inputdata, &basereq)
		rawreq, _ := r.Dump(&ffufreq)
		sb.WriteString(fmt.Sprintf("--- Session Time: %s ---\n%s\n", copt.Time.Format(time.RFC3339), string(rawreq)))
	}
	if sb.Len() == 0 {
		return "", fmt.Errorf("could not reconstruct request for hash %q", trimmed)
	}
	return sb.String(), nil
}

// ParseRawHTTPRequest parses raw HTTP request text (e.g. pasted from Burp/Caido) into structured target fields.
func (s *Service) ParseRawHTTPRequest(rawText, defaultProto string) (*ParsedRawHTTPRequestDTO, error) {
	normalized := strings.ReplaceAll(rawText, "\r\n", "\n")
	if strings.TrimSpace(normalized) == "" {
		return nil, errors.New("raw request text cannot be empty")
	}
	proto := strings.ToLower(strings.TrimSpace(defaultProto))
	if proto != "http" && proto != "https" {
		proto = "https"
	}

	var headPart, bodyPart string
	if idx := strings.Index(normalized, "\n\n"); idx >= 0 {
		headPart = normalized[:idx]
		bodyPart = normalized[idx+2:]
	} else {
		headPart = normalized
	}

	lines := strings.Split(headPart, "\n")
	if len(lines) == 0 {
		return nil, errors.New("missing HTTP request line")
	}
	parts := strings.Fields(strings.TrimSpace(lines[0]))
	if len(parts) < 2 {
		return nil, fmt.Errorf("malformed HTTP request line: %q", lines[0])
	}

	method := strings.ToUpper(parts[0])
	targetPath := parts[1]
	headers := make(map[string]string)
	host := ""

	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		kv := strings.SplitN(trimmed, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if strings.EqualFold(k, "content-length") {
			continue
		}
		if strings.EqualFold(k, "host") {
			host = v
			continue
		}
		headers[k] = v
	}

	targetURL := ""
	if strings.HasPrefix(strings.ToLower(targetPath), "http://") || strings.HasPrefix(strings.ToLower(targetPath), "https://") {
		targetURL = targetPath
	} else {
		if host == "" {
			return nil, errors.New("Host header is required when request path is relative")
		}
		if !strings.HasPrefix(targetPath, "/") {
			targetPath = "/" + targetPath
		}
		targetURL = fmt.Sprintf("%s://%s%s", proto, host, targetPath)
	}

	bodyPart = strings.TrimSuffix(bodyPart, "\n")
	return &ParsedRawHTTPRequestDTO{
		Method:  method,
		URL:     targetURL,
		Headers: headers,
		Data:    bodyPart,
	}, nil
}

func (s *Service) resumeFilePath() string {
	return filepath.Join(s.dataDir, "resume_state.json")
}

// GetPendingResumeState checks if an interrupted scan state exists on disk.
func (s *Service) GetPendingResumeState() (*PendingResumeDTO, error) {
	path := s.resumeFilePath()
	state, err := engine.LoadResumeState(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if state.TargetURL == "" || state.Wordlist == "" {
		return nil, nil
	}
	return &PendingResumeDTO{
		SessionID:      state.SessionID,
		TargetURL:      state.TargetURL,
		Method:         state.Method,
		Wordlist:       state.Wordlist,
		BrowserProfile: state.BrowserProfile,
		TLSProfile:     state.TLSProfile,
		CurrentIndex:   state.CurrentIndex,
		IPHash:         state.IPHash,
		Timestamp:      state.Timestamp,
	}, nil
}

// DiscardPendingResumeState removes any saved interrupted scan checkpoint.
func (s *Service) DiscardPendingResumeState() error {
	return engine.DeleteResumeState(s.resumeFilePath())
}

// ResumePendingScan decrypts any saved session cookies and resumes from the saved wordlist index.
func (s *Service) ResumePendingScan() (int64, error) {
	path := s.resumeFilePath()
	state, err := engine.LoadResumeState(path)
	if err != nil {
		return 0, fmt.Errorf("failed to load pending resume state: %w", err)
	}
	if state.TargetURL == "" || state.Wordlist == "" {
		return 0, errors.New("pending resume state is missing target URL or wordlist")
	}

	headers := make(map[string]string, len(state.Headers))
	for k, v := range state.Headers {
		headers[k] = v
	}
	if state.EncryptedCookies != "" && state.CookiesNonce != "" {
		cookies, err := engine.DecryptCookies(state.EncryptedCookies, state.CookiesNonce)
		if err != nil {
			return 0, fmt.Errorf("failed to decrypt saved cookies: %w", err)
		}
		if cookies != "" {
			hasCookie := false
			for k := range headers {
				if strings.EqualFold(k, "cookie") {
					hasCookie = true
					break
				}
			}
			if !hasCookie {
				headers["Cookie"] = cookies
			}
		}
	}

	method := state.Method
	if method == "" {
		method = "GET"
	}

	req := ScanRequestDTO{
		URL:             state.TargetURL,
		Method:          method,
		Headers:         headers,
		Wordlist:        state.Wordlist,
		Threads:         10,
		Timeout:         10,
		AutoCalibration: true,
		BrowserProfile:  state.BrowserProfile,
		TLSProfile:      state.TLSProfile,
		ResumeFromIndex: state.CurrentIndex,
	}

	sessionID, err := s.StartScan(req)
	if err != nil {
		return 0, err
	}
	_ = engine.DeleteResumeState(path)
	return sessionID, nil
}
