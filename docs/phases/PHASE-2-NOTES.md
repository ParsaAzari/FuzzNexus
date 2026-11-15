# Phase 2 Handoff Note: High-Performance Data Grid, Svelte 5 Stores & Apple HIG Foundations

- **Current Position:** Phase 2 Pre-Start (Planning & Architecture Handoff)
- **Completed Phases:**
  - Phase 0: Wails v2 Desktop Spike & Infrastructure (`phase-0-done`, commit `2f30b28`)
  - Phase 1: Core Engine, Storage, State Machine & Wails Bridge (`phase-1-done`, commit `132793c`)
- **Date:** 2026-11-15
- **Toolchain:** Go 1.27.3 darwin/arm64, Node v26.8.2, Wails v2.17.1, Vite v6.4.6, Svelte 5.55.7

---

## 1. Summary of Changes in Phase 1 (Commit `132793c`)

The following files and components were developed, audited, and strictly verified in Phase 1:
- `ffuf-desktop/internal/scan/safego.go` & `safego_test.go`:
  - Goroutine panic safety wrapper with token redaction (`Authorization`, `Cookie`, `Set-Cookie`, `Bearer`).
- `ffuf-desktop/internal/output/gui_output.go` & `gui_output_test.go`:
  - Single-writer goroutine with `sync.Cond` backpressure gate (max 50,000 items).
  - Monotonic contiguous IDs assigned under lock.
  - Trailing-edge coalesced event emission (`80ms` window).
  - Idempotent `Finalize()` and `FinalizeWithContext(ctx)`.
  - Zero goroutine leaks on persistent DB failure via `onFatal` callback and drain loop.
- `ffuf-desktop/internal/store/db.go` & `db_test.go`:
  - Pure-Go SQLite WAL store (`modernc.org/sqlite`) with single writer / 4 concurrent readers.
  - Indexed schema, bounded `GetResultsSince` with clamping (1..5000), session CRUD.
  - Benchmark: 19.1 us/batch (~13M items/sec batch insert capacity).
- `ffuf-desktop/internal/blobs/storage.go` & `storage_test.go`:
  - Zstandard compressed blob storage under `~/.ffuf-desktop/blobs/` with `0700`/`0600` permissions.
  - 500KB response truncation on safe UTF-8 boundaries.
  - Session quota limit (2GB) enforcement.
- `ffuf-desktop/internal/scan/session.go` & `session_test.go`:
  - Central state machine (`Idle`, `Running`, `Paused`, `Stopping`, `Completed`, `Failed`, `Stopped`).
  - Strict 4-step teardown: job stop -> workers join -> Finalize (5s timeout) -> WAL checkpoint & DB update -> notify state.
  - Upstream patch in `pkg/engine/job.go` eliminating `Job.interruptMonitor()` goroutine leak.
- `ffuf-desktop/internal/bridge/service.go` & `service_test.go`:
  - Typed bridge methods bound to Wails frontend, non-blocking event emitters, clean shutdown.
- `ffuf-desktop/app.go` & `main.go`:
  - Integrated bridge with Wails lifecycle (`OnStartup`, `OnShutdown`).

---

## 2. Evidence of Phase 1 Verification (Base Commit: `132793c`)

```bash
$ cd ffuf-desktop && go test -race -count=1 ./...
ok  	ffuf-desktop/internal/blobs	1.783s
ok  	ffuf-desktop/internal/bridge	1.821s
ok  	ffuf-desktop/internal/output	2.478s
ok  	ffuf-desktop/internal/scan	7.098s
ok  	ffuf-desktop/internal/store	1.777s

$ go test -race ./pkg/...
ok  	github.com/ffuf/ffuf/v2/pkg/engine	4.362s
ok  	github.com/ffuf/ffuf/v2/pkg/ffuf	1.572s
ok  	github.com/ffuf/ffuf/v2/pkg/filter	1.663s
ok  	github.com/ffuf/ffuf/v2/pkg/input	1.456s
ok  	github.com/ffuf/ffuf/v2/pkg/interactive	1.417s
ok  	github.com/ffuf/ffuf/v2/pkg/output	1.430s
ok  	github.com/ffuf/ffuf/v2/pkg/runner	11.816s
ok  	github.com/ffuf/ffuf/v2/pkg/scraper	1.552s
ok  	github.com/ffuf/ffuf/v2/pkg/testtarget	1.574s

$ go vet ./... && cd ffuf-desktop && go vet ./...
# Exit code 0, 100% clean
```

---

## 3. Decisions & Architecture Deviations for Phase 2

1. **Svelte 5 Runes Only:** No legacy Svelte 3/4 stores (`writable`, `derived`). All reactive state implemented using `$state`, `$derived`, `$effect`, and TypeScript classes or module runes.
2. **Virtual Data Grid Implementation:** Built natively with Svelte 5 runes rather than bulky external libraries. Fixed 32px row height enables \(O(1)\) scroll calculations (`Math.floor(scrollTop / rowHeight)`).
3. **Pull-Based Synchronization:** Frontend never buffers raw results from Go events directly. The `"results:committed"` event triggers pull queries (`GetResultsSince(sessionID, lastID, 1000)`), ensuring deterministic client memory usage.
4. **Apple HIG Design System:**
   - Strict CSS custom property palette: `--bg-primary`, `--bg-secondary`, `--bg-surface`, `--accent-blue`, `--accent-green`, `--accent-orange`, `--accent-red`, `--border-subtle`, `--text-primary`, `--text-secondary`, `--text-tertiary`.
   - Typography: Local Inter for interface, JetBrains Mono for tabular numbers and inspector code.
   - Selective text selection: Window chrome and table containers are `user-select: none`; table cells and inspector bodies are `user-select: text`.
   - macOS Traffic Light window controls accommodation with padding-top on the title bar.

---

## 4. Open Risks Table

| Risk Description | Probability | Impact | Mitigation Action | Owner | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Scroll jank or frame drops at 100,000+ rows | Medium | High | Fixed 32px row height virtualizer, transform-based row positioning, overscan bounded to 20 rows. | Frontend Lead | Open (To be verified in Phase 2) |
| Memory bloat on massive result lists in browser | Low | Medium | Pull batches of 1,000 items, store items in a flat array, recycle DOM nodes. | Frontend Lead | Open (To be verified in Phase 2) |
| DOM layout thrashing during live filter changes | Low | Medium | Filter and sort computations performed using `$derived` runes with stable references. | Frontend Lead | Open (To be verified in Phase 2) |

---

## 5. Next Steps (Phase 2 Execution Steps)

1. **Step 2.1: Svelte 5 Reactive Stores (`frontend/src/lib/stores/`):**
   - `scan.svelte.ts`: Scan state machine, progress metrics, rate/filter actions.
   - `results.svelte.ts`: Pull synchronization loop, monotonic ID tracking, filtering, sorting.
   - `theme.svelte.ts`: Light / Dark / System theme preference using Apple HIG color tokens.
2. **Step 2.2: Apple HIG Virtual Grid Component (`frontend/src/lib/components/VirtualGrid.svelte`):**
   - Virtual scroll container with dynamic slice calculation based on `scrollTop` and container height.
   - Fixed 32px row height with alternating subtle tint.
   - Resizable, sortable columns with status badges (2xx green, 3xx blue, 4xx yellow/orange, 5xx red).
   - Keyboard navigation (ArrowUp, ArrowDown, Enter to inspect).
3. **Step 2.3: Window Chrome & Top Navigation (`TitleBar.svelte`, `ControlBar.svelte`):**
   - macOS traffic lights inset, draggable header.
   - Live scan controls (Start, Pause, Resume, Stop, Rate slider, Live quick filter input).
   - Progress bar with percentage, current req/sec, and error badge.
4. **Step 2.4: Verification & Performance Benchmark:**
   - Validate 60fps scrolling with 10,000 synthetic rows in Vitest or component test.
   - Run `npm run build` and `npm run check`.
   - Commit and tag `phase-2-done`.
