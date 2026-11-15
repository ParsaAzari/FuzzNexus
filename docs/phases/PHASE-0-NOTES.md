# Phase 0 Handoff Note

- **Current Position:** Phase 0 Completed / Transition to Phase 1 (Step 1.0 Finished)
- **Completed Phases:** Phase 0 (Wails v2 Desktop Spike & Infrastructure)
- **Date:** 2026-11-15
- **Toolchain:** Go 1.27.3 darwin/arm64, Node v26.8.2, Wails v2.17.1, Vite 6.2.0, Svelte 5.55.7

---

## 1. Previous Phase Changes

The following files and components were established in Phase 0:

- **`.github/workflows/desktop-ci.yml`:** Created 3-OS CI matrix workflow (macOS, Ubuntu, Windows) executing lint, core engine tests, frontend verification, production CSP validation, desktop Go tests, and Wails binary packaging.
- **`docs/ARCHITECTURE.md`:** Comprehensive architectural specification, pipeline diagrams, and reproducible codebase audit outputs (`os.Exit`, global state, `signal.Notify`, mutexes, `FFUFHASH`).
- **`docs/UPSTREAM_PATCHES.md`:** Documented minimal draft patch for `Job.interruptMonitor()` signal leak resolution and confirmed `MatcherManager.mu` thread safety.
- **`docs/THIRD_PARTY_LICENSES.md`:** Recorded SIL Open Font License 1.1 for Inter and JetBrains Mono, along with MIT and BSD licenses for Go and npm dependencies.
- **`ffuf-desktop/`:** Scaffolding initialized using official Wails v2 template (`svelte-ts`).
- **`ffuf-desktop/go.mod`:** Local replacement linked to parent module (`replace github.com/ffuf/ffuf/v2 => ../`) and added runtime dependencies: `modernc.org/sqlite v1.62.0`, `github.com/klauspost/compress v1.20.4`, `go.uber.org/goleak v1.3.0`.
- **`ffuf-desktop/frontend/src/assets/fonts/`:** Downloaded and bundled 7 local `.woff2` font files:
  - `Inter-Regular.woff2`, `Inter-Medium.woff2`, `Inter-SemiBold.woff2`, `Inter-Bold.woff2`
  - `JetBrainsMono-Regular.woff2`, `JetBrainsMono-Medium.woff2`, `JetBrainsMono-Bold.woff2`
- **`ffuf-desktop/frontend/vite.config.ts`:** Implemented Vite HTML transform plugin separating Development CSP (`ws://localhost:*` & `ipc:`) from Production CSP (`default-src 'self' ... connect-src 'self' wails: http://wails.localhost`).
- **`ffuf-desktop/frontend/src/style.css`:** Added relative `@font-face` definitions, Apple HIG tokens for `Neutral` and `Ocean` themes (Light/Dark), and selective `user-select` rules (`user-select: text` on table/code/inputs; `user-select: none` on window chrome).

---

## 2. Decisions & Plan Deviations

1. **Vite Version Pinning:** In `ffuf-desktop/frontend/package.json`, Vite was adjusted from `^8.0.13` (which resolved to Vite 8.3.8) to `^6.2.0` because `@sveltejs/vite-plugin-svelte@6.2.4` requires peer dependency `vite@^6.0.0 || ^7.0.0`. This resolved `ERESOLVE` npm errors cleanly without `--force`.
2. **Git Repository Initialization:** Initialized Git repository on `main` branch with comprehensive `.gitignore` filtering build artifacts, binaries, and `node_modules`.

---

## 3. Test Results with Citations

### 3.1 `go build ./...`
```bash
$ go build ./...
# Exit code: 0
```

### 3.2 `go doc ./pkg/ffuf OutputProvider`
```bash
$ go doc ./pkg/ffuf OutputProvider
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
    OutputProvider is responsible of providing output from the RunnerProvider
```

### 3.3 `go doc ./pkg/ffuf Response.GetRedirectLocation`
```bash
$ go doc ./pkg/ffuf Response.GetRedirectLocation
package ffuf // import "github.com/ffuf/ffuf/v2/pkg/ffuf"

func (resp *Response) GetRedirectLocation(absolute bool) string
    GetRedirectLocation returns the redirect location for a 3xx redirect HTTP
    response
```

### 3.4 `go test -race -count=1 ./pkg/...`
```bash
$ go test -race -count=1 ./pkg/...
?   	github.com/ffuf/ffuf/v2/pkg/assembly	[no test files]
ok  	github.com/ffuf/ffuf/v2/pkg/engine	4.501s
ok  	github.com/ffuf/ffuf/v2/pkg/ffuf	1.944s
ok  	github.com/ffuf/ffuf/v2/pkg/filter	2.030s
ok  	github.com/ffuf/ffuf/v2/pkg/input	1.819s
ok  	github.com/ffuf/ffuf/v2/pkg/interactive	1.813s
ok  	github.com/ffuf/ffuf/v2/pkg/output	1.833s
ok  	github.com/ffuf/ffuf/v2/pkg/runner	11.701s
ok  	github.com/ffuf/ffuf/v2/pkg/scraper	1.749s
ok  	github.com/ffuf/ffuf/v2/pkg/testtarget	1.787s
```

### 3.5 Frontend Build & Asset Hashing (`npm run build`)
```bash
$ cd ffuf-desktop/frontend && npm run build
vite v6.4.6 building for production...
✓ 113 modules transformed.
dist/index.html                                     0.70 kB │ gzip:  0.42 kB
dist/assets/JetBrainsMono-Regular-CA-Os4ii.woff2   92.38 kB
dist/assets/JetBrainsMono-Medium-DoKWA_AF.woff2    94.28 kB
dist/assets/JetBrainsMono-Bold-CUogYd9I.woff2      94.63 kB
dist/assets/Inter-Regular-COLGFB3M.woff2          111.27 kB
dist/assets/Inter-Medium-CDhBSFyE.woff2           114.35 kB
dist/assets/Inter-SemiBold-BAEEcJ4E.woff2         114.81 kB
dist/assets/Inter-Bold-BOs3KVhN.woff2             114.84 kB
dist/assets/logo-universal-BdCeFAbM.png           139.70 kB
dist/assets/index-DftLQKfm.css                      3.81 kB │ gzip:  1.23 kB
dist/assets/index-B4OIhUTk.js                      28.57 kB │ gzip: 11.26 kB
✓ built in 450ms
```

### 3.6 Wails Desktop Binary Packaging (`wails build`)
```bash
$ cd ffuf-desktop && ~/go/bin/wails build
# Building target: darwin/arm64
  • Generating bindings: Done.
  • Installing frontend dependencies: Done.
  • Compiling frontend: Done.
  • Compiling application: Done.
  • Packaging application: Done.
  • Self-signing application: Done.
Built 'build/bin/ffuf-desktop.app/Contents/MacOS/ffuf-desktop' in 19.221s.
```

### 3.7 Production CSP Verification
```bash
$ grep -i "Content-Security-Policy" ffuf-desktop/frontend/dist/index.html
<meta http-equiv="Content-Security-Policy" content="default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data:; connect-src 'self' wails: http://wails.localhost; object-src 'none'; base-uri 'none'; form-action 'none'; frame-src 'none';"/>
```

---

## 4. Observed Wails Origin / Scheme Mapping

| Operating System | Webview Engine | Observed IPC Scheme / Origin | Connect-src Required |
| :--- | :--- | :--- | :--- |
| **macOS** | WebKit (WKWebView) | `wails://` | `'self' wails:` |
| **Windows** | Microsoft Edge WebView2 | `http://wails.localhost` | `'self' http://wails.localhost` |
| **Linux** | WebKitGTK | `wails://` / `http://wails.localhost` | `'self' wails: http://wails.localhost` |

---

## 5. Open Risks Table

| Risk Description | Probability | Impact | Mitigation Action | Owner | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Signal handler intercept in Wails `OnShutdown` | Medium | Medium | Use `OnBeforeClose` callback in `main.go` to stop scan and finalize provider cleanly before shutdown. | Backend Lead | Mitigated / In Progress |
| WebKitGTK sandbox quirks on Linux | Low | Medium | Test empty sandbox iframe in Linux CI runner; verify no network leakage. | Frontend Lead | Open (Phase 5) |
| High rate lock contention in SQLite WAL | Low | High | Use single writer goroutine, 80ms batch commits, and PRAGMA synchronous=NORMAL. | Backend Lead | Mitigated (Phase 1) |

---

## 6. Next Steps (Phase 1 Execution)

1. Create `docs/phases/PHASE-1-NOTES.md` defining Phase 1 implementation targets and test cases.
2. Implement `ffuf-desktop/internal/scan/session.go` (Central State Machine with deterministic shutdown).
3. Implement `ffuf-desktop/internal/output/gui_output.go` (Deterministic single-writer, bounded backpressure, `sync.Cond`, trailing-edge emit).
4. Implement `ffuf-desktop/internal/store/db.go` (SQLite WAL mode, migrations, clamp-bounded `GetResultsSince`).
5. Implement `ffuf-desktop/internal/blobs/storage.go` (Disk payloads, 0700 permissions, zstd quota).
6. Implement `internal/scan/safego.go` (Panic recovery with token masking).
7. Execute Phase 1 unit and integration tests with `go test -race -count=1` and `goleak.VerifyTestMain(m)`.
