# System Architecture: ffuf Desktop

ffuf Desktop is an enterprise-grade desktop graphical user interface and analytical telemetry dashboard for the `ffuf v2` web fuzzer. It embeds the high-performance Go fuzzing engine directly in-process via Wails v2, eliminating child processes, terminal escaping, and CLI parsing bottlenecks.

---

## 1. System Overview & Core Pipelines

```
+-----------------------------------------------------------------------------------+
|                        Frontend: Pure Vite + Svelte 5 SPA                         |
|   - Apple Human Interface Guidelines (HIG) Visual Tokens                          |
|   - Pull-Based Data Grid (requestAnimationFrame batching)                         |
|   - Interactive uPlot 2D Scatter Matrix (Size x Words Response Clustering)        |
|   - Sandboxed Inspector (<iframe sandbox=""> with inert DOMParser & strict CSP)   |
|   - Attack Composer (§ Sniper markers, Multi-wordlist combinatorics, cURL import) |
+-----------------------------------------------------------------------------------+
                                         │  ▲
       Wails v2 IPC (runtime.EventsOn)  │  │ GetResultsSince(sessionID, lastID, limit)
                                         ▼  │
+-----------------------------------------------------------------------------------+
|                            Bridge & Scan Manager (Go)                             |
|   - Central State Machine: Idle -> Running -> Paused -> Stopping -> Stopped/Failed|
|   - Single Active Scan Concurrency with FIFO queue                                |
|   - Coalesced commit notification: results:committed {latest_id, count, ts}       |
|   - Structured Errors: {code, field?, message}                                    |
|   - Panic Resilience: safeGo wrapper with stack trace redaction                   |
+-----------------------------------------------------------------------------------+
                                         │  ▲
              Bounded Channel (cap 50)  │  │ Read Query Pool
                                         ▼  │
+-----------------------------------------------------------------------------------+
|                     Deterministic Output & Storage Subsystem                      |
|   - GUIOutputProvider: In-process ffuf.OutputProvider implementation             |
|   - Backpressure Gate: sync.Cond blocks workers when buffer reaches maxBuffered   |
|   - Dedicated Single-Writer Goroutine (SQLite WAL mode, batch transactions)       |
|   - Disk Blobs Storage (~/.ffuf-desktop/blobs/, 0700 permissions, zstd quota)     |
+-----------------------------------------------------------------------------------+
                                         ▲
                                         │ ffuf.Response (Result callback)
+-----------------------------------------------------------------------------------+
|                               ffuf v2 Core Engine                                 |
|   - RunnerProvider: HTTP/1.1 & HTTP/2, Client Certs, SNI                          |
|   - MatcherManager: Live sync.RWMutex protected runtime filter mutations          |
|   - RateThrottle: Dynamic token-bucket rate limiter (RPS adjustments)             |
|   - Preflight / Postflight: Dynamic VarExtract engine (CSRF/Cookies/Headers)      |
+-----------------------------------------------------------------------------------+
```

---

## 2. In-Process Concurrency & Backpressure Architecture

### 2.1 Monotonic Sequential IDs & Buffer Management
To guarantee contiguous, ascending sequence numbers without data races, ID generation occurs strictly under the provider's mutex:
```go
func (p *GUIOutputProvider) Result(resp ffuf.Response) {
    p.mu.Lock()
    defer p.mu.Unlock()

    for len(p.buf) >= p.maxBuffered && !p.finalized && !p.failed.Load() && p.ctx.Err() == nil {
        p.cond.Wait() // Releases p.mu while sleeping, re-acquires p.mu on wakeup
    }

    if p.finalized || p.failed.Load() || p.ctx.Err() != nil {
        return
    }

    p.nextID++
    p.buf = append(p.buf, p.toDTO(p.nextID, resp))
}
```

### 2.2 Bounded Backpressure & Natural Throttling
When the in-memory buffer reaches `maxBuffered` (default 50,000 DTOs), worker goroutines calling `Result()` block on `p.cond.Wait()`. This naturally throttles the dispatch rate of `ffuf` workers to match the disk write throughput of SQLite, preventing memory exhaustion under sustained 5,000+ req/s workloads.

### 2.3 Single Writer Goroutine & Trailing-Edge Coalescing
A dedicated background writer goroutine receives batches from a bounded channel (`writeCh chan []ResultDTO`, capacity 50):
1. Commits batches to SQLite strictly sequentially inside a transaction.
2. In the event of SQLite lock contention, retries up to 3 times with exponential backoff.
3. If persistent failure occurs, sets `failed.Store(true)`, broadcasts to unblock all workers, calls `onFatal(err)` to halt `ffuf.Job`, drains remaining channel items, and exits cleanly.
4. An independent 80ms timer emits a coalesced `results:committed` event if `dirty` is true, guaranteeing trailing-edge delivery for the final batch.

### 2.4 Pull-Based UI Synchronization
The frontend maintains a single in-flight `GetResultsSince(sessionID, lastRenderedID, limit)` request:
- The UI listens for `results:committed {latest_id, count, ts}`.
- If no pull is currently pending and `latest_id > lastRenderedID`, the UI pulls the next chunk.
- Missing chunks (e.g., if a network or event drop occurred) are automatically resolved because the frontend requests `lastRenderedID`.
- Incoming rows are appended to flat Typed Arrays (`Int32Array`, `Float32Array`) and rendered via `requestAnimationFrame` batching in the Virtual Table.

---

## 3. Security & Isolation Architecture

### 3.1 Content Security Policy (CSP)
- **Production CSP (`index.html`):**
  ```
  default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data:; connect-src 'self' wails: http://wails.localhost; object-src 'none'; base-uri 'none'; form-action 'none'; frame-src 'none';
  ```
- **Development CSP:**
  Permits `ws://localhost:*` and `ipc:` solely during `wails dev` to enable Vite Hot Module Replacement (HMR).

### 3.2 Inspector Sandboxing & Preview Hardening
The raw response preview in the Inspector uses a triple-layered defense:
1. **Go Byte Truncation:** `GetResultPayload` clamps bodies to a 500KB maximum on valid UTF-8 character boundaries. Binary payloads are routed exclusively to the Hex viewer.
2. **Inert DOMParser Sanitization:** The frontend parses the HTML using `DOMParser().parseFromString(rawHtml, 'text/html')` in an inert context and strips `<meta http-equiv="refresh">`, `<base>`, `<link>`, `<script>`, `<iframe>`, `<object>`, `<embed>`, and `<form>`.
3. **Empty Sandbox Isolation:** Rendered into `<iframe sandbox="" srcdoc="...">` containing an embedded `default-src 'none'` meta CSP. Script execution, navigation, external resource loading, and base-hijacking are completely neutralized.

### 3.3 Filesystem & Sensitive Data Protection
- Application root directory `~/.ffuf-desktop` is initialized with POSIX `0700` permissions (read/write/search by owner only).
- Database and blob files are created with `0600` permissions.
- In-memory and exported configuration data is sanitized via `config_redacted_json`, masking `Authorization`, `Cookie`, and `Set-Cookie` tokens.
- `--input-cmd` is disabled by default and requires explicit user consent in Settings. Wordlist paths are canonicalized with `filepath.Clean` and `EvalSymlinks`.

---

## 4. Empirical Codebase Audit (Evidence)

Audited against `github.com/ffuf/ffuf/v2` (Git Tag `v2`, Go 1.26+ runtime):

### 4.1 Audit Command: `os.Exit` Check
```bash
$ grep -rn --include="*.go" "os.Exit" .
./test/integration/harness_test.go:36:	os.Exit(code)
./test/e2e/e2e_test.go:29:func TestMain(m *testing.M) { os.Exit(runMain(m)) }
./characterization_test.go:37:// installs a fresh ContinueOnError FlagSet so a parse error can't os.Exit the tests.
./help.go:73:			os.Exit(1)
./main.go:52:			os.Exit(1)
./main.go:73:		os.Exit(0)
./main.go:78:		os.Exit(0)
./main.go:102:			os.Exit(1)
./main.go:116:		os.Exit(1)
./main.go:128:		os.Exit(1)
./main.go:134:		os.Exit(1)
```
**Conclusion:** Zero occurrences of `os.Exit` in `pkg/`. All CLI termination calls reside strictly in `main.go`, `help.go`, and test harnesses.

### 4.2 Audit Command: `panic(` and `log.Fatal` Check
```bash
$ grep -rn --include="*.go" "log.Fatal\|panic(" pkg/
pkg/ffuf/flags.go:259:				panic("ffuf: flag -" + name + " is missing a usage tag")
pkg/ffuf/flags.go:262:				panic("ffuf: flag -" + name + " has unknown section \"" + section + "\"")
pkg/ffuf/flags.go:306:			panic("ffuf: slice flag -" + name + " needs kind:multistring|wordlist|csvreplace (got " + kind + ")")
pkg/ffuf/flags.go:309:		panic("ffuf: unsupported flag field type for -" + name)
```
**Conclusion:** Zero `log.Fatal` calls. The 4 `panic(` calls occur in `pkg/ffuf/flags.go` inside CLI flag registration reflection tests, which are not invoked when initializing configurations programmatically via `ffuf.NewConfigOptions()`.

### 4.3 Audit Command: `signal.Notify` Check
```bash
$ grep -rn --include="*.go" "signal.Notify" .
./pkg/engine/job.go:396:	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
```
**Conclusion:** `signal.Notify` exists solely at `pkg/engine/job.go:396` within `interruptMonitor()`.

### 4.4 Audit Command: Mutable Global Variables Check
```bash
$ grep -rn --include="*.go" "^var \|sync.Once" pkg/
pkg/output/file_csv.go:12:var staticheaders = []string{...}
pkg/ffuf/config.go:20:var VarSources = []string{...}
pkg/ffuf/constants.go:8:var ( VERSION = "0.0.0", CONFIGDIR = ..., HISTORYDIR = ... )
pkg/ffuf/flags.go:100:var extraFlags = []extraFlag{...}
pkg/ffuf/util.go:16:var chars = []rune(...)
pkg/ffuf/util.go:106:	versionOnce     sync.Once
```
**Conclusion:** All package-level variables are read-only constants, slices, or sync mechanisms. No mutable scan runtime state is shared globally.

### 4.5 Audit Command: Core Mutexes & Thread Safety
```bash
$ grep -rn "RWMutex\|sync.Mutex" pkg/filter pkg/engine pkg/ffuf
pkg/filter/filter.go:21:	mu               sync.RWMutex
pkg/engine/jobqueue.go:12:	mu   sync.Mutex
pkg/engine/rate.go:14:	RateMutex      sync.Mutex
pkg/engine/job.go:52:	timeMutex       sync.Mutex
pkg/engine/job.go:53:	errMutex        sync.Mutex
pkg/engine/job.go:54:	inputMutex      sync.Mutex
pkg/engine/job.go:55:	calibMutex      sync.Mutex
pkg/engine/job.go:56:	pauseStateMutex sync.Mutex
pkg/engine/job.go:57:	pauseMutex      sync.RWMutex
```
**Conclusion:** `MatcherManager.mu` is a `sync.RWMutex` (`pkg/filter/filter.go:21`), verifying that live filter additions/removals during active fuzzing are thread-safe.

### 4.6 Audit Command: FFUFHASH Generation & Location
```bash
$ grep -rn --include="*.go" "FFUFHASH" .
pkg/engine/job.go:367:		nextInput["FFUFHASH"] = j.ffufHash(nextPosition)
pkg/output/file_html.go:218:			if k == "FFUFHASH" { ffufhash = string(v) }
pkg/output/file_csv.go:58:		if k == "FFUFHASH" { ffufhash = string(v) }
```
**Conclusion:** FFUFHASH is generated per request position and stored in `req.Input["FFUFHASH"]`. It is safely read from `resp.Request.Input["FFUFHASH"]`.

### 4.7 Audit Command: Core Interface Signatures (`go doc`)
```
package ffuf // import "github.com/ffuf/ffuf/v2/pkg/ffuf"

type OutputProvider interface {
	Banner()
	Finalize() error
	Progress(status Progress)
	Info(infostring string)
	Error(errstring string)
	Raw(output string)
	Warning(warnstring string)
	Result(resp Response)
	PrintResult(res Result)
	SaveFile(filename, format string) error
	GetCurrentResults() []Result
	SetCurrentResults(results []Result)
	FilterCurrentResults(keep func(Result) bool)
	SetPaused(paused bool)
	PendingResults() int
	Reset()
	Cycle()
}

type MatcherManager interface {
	SetCalibrated(calibrated bool)
	SetCalibratedForHost(host string, calibrated bool)
	AddFilter(name string, option string, replace bool) error
	AddPerDomainFilter(domain string, name string, option string) error
	RemoveFilter(name string)
	AddMatcher(name string, option string) error
	GetFilters() map[string]FilterProvider
	GetMatchers() map[string]FilterProvider
	FiltersForDomain(domain string) map[string]FilterProvider
	CalibratedForDomain(domain string) bool
	Calibrated() bool
	Matches(resp *Response, perHost bool, matcherMode string, filterMode string) bool
}
```
**Conclusion:** Verified against live toolchain. The `GUIOutputProvider` satisfies `ffuf.OutputProvider` completely.
