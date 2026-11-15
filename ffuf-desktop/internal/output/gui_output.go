package output

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ffuf/ffuf/v2/pkg/ffuf"
)

// ResultDTO represents a sanitized, lightweight result payload sent to SQLite and the UI.
// Headers and raw response bodies are strictly omitted from this DTO.
type ResultDTO struct {
	ID               int64             `json:"id"`
	Position         int               `json:"pos"`
	Inputs           map[string]string `json:"inputs"` // Multi-keyword input values (Clusterbomb/Pitchfork)
	Status           int64             `json:"status"`
	Length           int64             `json:"length"`
	Words            int64             `json:"words"`
	Lines            int64             `json:"lines"`
	DurationMs       int64             `json:"duration_ms"`
	URL              string            `json:"url"`
	RedirectLocation string            `json:"redirect_location,omitempty"`
	ContentType      string            `json:"content_type,omitempty"`
	Hash             string            `json:"hash,omitempty"` // FFUFHASH
	Host             string            `json:"host"`
	ScraperData      map[string]string `json:"scraper_data,omitempty"`
	Cancelled        bool              `json:"cancelled"` // Displayed as "Timeout/Cancelled" in UI
}

// CommittedEvent is a lightweight coalesced notification emitted to the UI when results are committed.
type CommittedEvent struct {
	LatestID int64 `json:"latest_id"`
	Count    int   `json:"count"`
	Ts       int64 `json:"ts"`
}

// FinishedEvent indicates scan finalization with total committed rows.
type FinishedEvent struct {
	LastID int64  `json:"last_id"`
	Total  int64  `json:"total"`
	State  string `json:"state"`
}

// GUIOutputProvider implements ffuf.OutputProvider with bounded backpressure, single-writer SQLite commits,
// and guaranteed trailing-edge event delivery.
type GUIOutputProvider struct {
	mu          sync.Mutex
	cond        *sync.Cond
	buf         []ResultDTO
	maxBuffered int
	nextID      int64
	finalized   bool
	failed      atomic.Bool
	failErrMu   sync.RWMutex
	failErr     error

	writeCh      chan []ResultDTO
	tickerStop   chan struct{}
	tickerDone   chan struct{}
	writerDone   chan struct{}
	finalizeOnce sync.Once
	finalizeErr  error

	sessionID      int64
	ctx            context.Context
	cancel         context.CancelFunc
	logger         *slog.Logger

	dbWriterFn     func(sessionID int64, batch []ResultDTO) error
	emitCommitFn   func(event CommittedEvent)
	emitFinishedFn func(event FinishedEvent)
	emitErrorFn    func(code string, msg string)
	onFatal        func(err error)
	onProgress     func(status ffuf.Progress)
	payloadSaver   func(resultID int64, req []byte, resp []byte)
}

// NewGUIOutputProvider initializes and starts the background loops for GUIOutputProvider.
func NewGUIOutputProvider(
	ctx context.Context,
	sessionID int64,
	logger *slog.Logger,
	dbWriter func(int64, []ResultDTO) error,
	emitCommit func(CommittedEvent),
	emitFinished func(FinishedEvent),
	emitError func(string, string),
	onFatal func(error),
) *GUIOutputProvider {
	if logger == nil {
		logger = slog.Default()
	}
	cCtx, cancel := context.WithCancel(ctx)
	p := &GUIOutputProvider{
		buf:            make([]ResultDTO, 0, 1024),
		maxBuffered:    50000, // Maximum queued DTOs before applying backpressure to ffuf workers
		writeCh:        make(chan []ResultDTO, 50),
		tickerStop:     make(chan struct{}),
		tickerDone:     make(chan struct{}),
		writerDone:     make(chan struct{}),
		sessionID:      sessionID,
		ctx:            cCtx,
		cancel:         cancel,
		logger:         logger,
		dbWriterFn:     dbWriter,
		emitCommitFn:   emitCommit,
		emitFinishedFn: emitFinished,
		emitErrorFn:    emitError,
		onFatal:        onFatal,
	}
	p.cond = sync.NewCond(&p.mu)

	go p.tickerLoop()
	go p.singleWriterLoop()
	return p
}

// SetMaxBuffered overrides the default maximum buffer threshold (primarily for testing).
func (p *GUIOutputProvider) SetMaxBuffered(limit int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxBuffered = limit
	p.cond.Broadcast()
}

// SetOnProgress configures an optional callback for ffuf.Progress events.
func (p *GUIOutputProvider) SetOnProgress(fn func(ffuf.Progress)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onProgress = fn
}

// SetPayloadSaver configures an optional callback for storing raw request/response payloads.
func (p *GUIOutputProvider) SetPayloadSaver(fn func(resultID int64, req []byte, resp []byte)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.payloadSaver = fn
}

// Result implements ffuf.OutputProvider. ID allocation and append occur in a single locked critical section.
func (p *GUIOutputProvider) Result(resp ffuf.Response) {
	p.mu.Lock()

	// Q-04: Bounded Backpressure gate. Blocks workers when buffer exceeds maxBuffered.
	for len(p.buf) >= p.maxBuffered && !p.finalized && !p.failed.Load() && p.ctx.Err() == nil {
		p.cond.Wait()
	}

	if p.finalized || p.failed.Load() || p.ctx.Err() != nil {
		p.mu.Unlock()
		return
	}

	p.nextID++
	id := p.nextID
	p.buf = append(p.buf, p.toDTO(id, resp))
	saver := p.payloadSaver
	p.mu.Unlock()

	if saver != nil {
		var reqBytes []byte
		if resp.Request != nil {
			if len(resp.Request.Raw) > 0 {
				reqBytes = []byte(resp.Request.Raw)
			} else {
				reqBytes = []byte(fmt.Sprintf("%s %s\r\nHost: %s\r\n\r\n%s", resp.Request.Method, resp.Request.Url, resp.Request.Host, string(resp.Request.Data)))
			}
		}
		saver(id, reqBytes, resp.Data)
	}
}

func (p *GUIOutputProvider) toDTO(id int64, resp ffuf.Response) ResultDTO {
	url := ""
	host := ""
	ffufHash := ""
	inputs := make(map[string]string)
	pos := 0

	if resp.Request != nil {
		url = sanitizeUTF8(resp.Request.Url, 2048)
		host = sanitizeUTF8(resp.Request.Host, 256)
		pos = resp.Request.Position
		for k, v := range resp.Request.Input {
			if k == "FFUFHASH" {
				ffufHash = string(v)
			} else {
				inputs[k] = sanitizeUTF8(string(v), 512)
			}
		}
	}

	scraperData := make(map[string]string)
	for k, v := range resp.ScraperData {
		if len(v) > 0 {
			scraperData[k] = sanitizeUTF8(strings.Join(v, ", "), 1024)
		}
	}

	return ResultDTO{
		ID:               id,
		Position:         pos,
		Inputs:           inputs,
		Status:           resp.StatusCode,
		Length:           resp.ContentLength,
		Words:            resp.ContentWords,
		Lines:            resp.ContentLines,
		DurationMs:       resp.Duration.Milliseconds(),
		URL:              url,
		RedirectLocation: sanitizeUTF8(resp.GetRedirectLocation(false), 2048),
		ContentType:      sanitizeUTF8(resp.ContentType, 128),
		Hash:             ffufHash,
		Host:             host,
		ScraperData:      scraperData,
		Cancelled:        resp.Cancelled,
	}
}

// takeBuffer extracts the buffer slice under lock and broadcasts condition to unblock workers.
func (p *GUIOutputProvider) takeBuffer() []ResultDTO {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.buf) == 0 {
		return nil
	}
	b := make([]ResultDTO, len(p.buf))
	copy(b, p.buf)
	p.buf = p.buf[:0]
	p.cond.Broadcast() // Unblock workers waiting for buffer capacity
	return b
}

func (p *GUIOutputProvider) tickerLoop() {
	defer close(p.tickerDone)
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.tickerStop:
			return
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.FlushTick()
		}
	}
}

// FlushTick sends pending buffer to writeCh outside mutex.
func (p *GUIOutputProvider) FlushTick() {
	b := p.takeBuffer()
	if len(b) == 0 || p.failed.Load() {
		return
	}
	select {
	case p.writeCh <- b:
	case <-p.ctx.Done():
	}
}

// singleWriterLoop executes deterministic sequential writes with retry, trailing-edge coalescing, and failure drain.
func (p *GUIOutputProvider) singleWriterLoop() {
	defer close(p.writerDone)
	var latestID int64
	var pendingCount int
	var dirty bool

	emitTicker := time.NewTicker(80 * time.Millisecond)
	defer emitTicker.Stop()

	for {
		select {
		case <-emitTicker.C:
			// Q-05: Trailing-edge coalesced notification
			if dirty {
				if p.emitCommitFn != nil {
					p.emitCommitFn(CommittedEvent{
						LatestID: latestID,
						Count:    pendingCount,
						Ts:       time.Now().UnixMilli(),
					})
				}
				dirty = false
				pendingCount = 0
			}

		case batch, ok := <-p.writeCh:
			if !ok {
				// Channel closed by Finalize. Guaranteed trailing-edge emit.
				if dirty && p.emitCommitFn != nil {
					p.emitCommitFn(CommittedEvent{
						LatestID: latestID,
						Count:    pendingCount,
						Ts:       time.Now().UnixMilli(),
					})
				}
				return
			}

			if len(batch) == 0 {
				continue
			}

			// Q-03: Commit with retry (3 attempts with exponential backoff)
			var err error
			for attempt := 1; attempt <= 3; attempt++ {
				err = p.dbWriterFn(p.sessionID, batch)
				if err == nil {
					break
				}
				p.logger.Warn("SQLite commit failed, retrying", "attempt", attempt, "err", err)
				time.Sleep(time.Duration(attempt*25) * time.Millisecond)
			}

			if err != nil {
				// Q-03: Persistent failure handling
				p.logger.Error("SQLite commit persistently failed; halting scan", "err", err)
				p.failErrMu.Lock()
				p.failErr = err
				p.failErrMu.Unlock()
				p.failed.Store(true)

				// Unblock any workers in Result()
				p.mu.Lock()
				p.cond.Broadcast()
				p.mu.Unlock()

				if p.emitErrorFn != nil {
					p.emitErrorFn("DB_COMMIT_FAILURE", "Failed to persist scan results to disk.")
				}
				if p.onFatal != nil {
					p.onFatal(err) // Halts job and sets StateFailed in scan.Session
				}

				// Drain remaining channel items to prevent blocking senders
				for range p.writeCh {
				}
				return
			}

			latestID = batch[len(batch)-1].ID
			pendingCount += len(batch)
			dirty = true
		}
	}
}

// Finalize implements ffuf.OutputProvider with a default 5-second timeout context.
func (p *GUIOutputProvider) Finalize() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.FinalizeWithContext(ctx)
}

// FinalizeWithContext executes deterministic teardown without deadlock or race.
func (p *GUIOutputProvider) FinalizeWithContext(ctx context.Context) error {
	p.finalizeOnce.Do(func() {
		// 1. Mark finalized under lock and unblock waiting workers
		p.mu.Lock()
		p.finalized = true
		p.cond.Broadcast()
		p.mu.Unlock()

		// 2. Q-02: Join ticker loop before touching writeCh
		close(p.tickerStop)
		<-p.tickerDone

		// 3. Final buffer flush (outside mutex)
		p.FlushTick()

		// 4. Close writeCh (exclusively here)
		close(p.writeCh)

		// 5. Wait for single writer loop to drain writeCh
		select {
		case <-p.writerDone:
		case <-ctx.Done():
			p.finalizeErr = fmt.Errorf("timeout draining database writer")
		}

		// 6. Q-06: Emit scan:finished or report failure
		if p.failed.Load() {
			p.failErrMu.RLock()
			p.finalizeErr = p.failErr
			p.failErrMu.RUnlock()
		} else if p.finalizeErr == nil {
			p.mu.Lock()
			total := p.nextID
			p.mu.Unlock()
			if p.emitFinishedFn != nil {
				p.emitFinishedFn(FinishedEvent{
					LastID: total,
					Total:  total,
					State:  "Completed",
				})
			}
		}
		p.cancel()
	})
	return p.finalizeErr
}

// Required ffuf.OutputProvider interface stubs
func (p *GUIOutputProvider) Banner()                                      {}
func (p *GUIOutputProvider) Progress(status ffuf.Progress) {
	p.mu.Lock()
	fn := p.onProgress
	p.mu.Unlock()
	if fn != nil {
		fn(status)
	}
}
func (p *GUIOutputProvider) Info(infostring string)                       {}
func (p *GUIOutputProvider) Error(errstring string)                       {}
func (p *GUIOutputProvider) Raw(output string)                            {}
func (p *GUIOutputProvider) Warning(warnstring string)                    {}
func (p *GUIOutputProvider) PrintResult(res ffuf.Result)                  {}
func (p *GUIOutputProvider) SaveFile(filename, format string) error       { return nil }
func (p *GUIOutputProvider) GetCurrentResults() []ffuf.Result              { return nil }
func (p *GUIOutputProvider) SetCurrentResults(results []ffuf.Result)      {}
func (p *GUIOutputProvider) FilterCurrentResults(keep func(ffuf.Result) bool) {}
func (p *GUIOutputProvider) SetPaused(paused bool)                        {}
func (p *GUIOutputProvider) PendingResults() int                          { return 0 }
func (p *GUIOutputProvider) Reset()                                       {}
func (p *GUIOutputProvider) Cycle()                                       {}

func sanitizeUTF8(s string, maxLen int) string {
	valid := strings.ToValidUTF8(s, "")
	if len(valid) > maxLen {
		return valid[:maxLen]
	}
	return valid
}
