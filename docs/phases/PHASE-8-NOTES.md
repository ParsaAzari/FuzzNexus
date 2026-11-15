# Phase 8 Handoff Note: UI/UX Polish + Advanced Filters + Wordlist Manager

- **Current Position:** Phase 8 Pre-Start (Planning & Architecture Handoff)
- **Completed Phases:**
  - Phase 0: Wails v2 Desktop Spike & Infrastructure (`phase-0-done`, commit `2f30b28`)
  - Phase 1: Core Engine, Storage, State Machine & Wails Bridge (`phase-1-done`, commit `132793c`)
  - Phase 2: High-Performance Data Grid, Svelte 5 Stores & Apple HIG Foundations (`phase-2-done`, commit `df47e4d`)
  - Phase 3: Dual Inspector, Sandboxed HTML Preview & Analytics Charts (`phase-3-done`, commit `b8d327d`)
  - Phase 4: Multi-OS Verification, Package Bundling & Final Delivery (`phase-4-done`, commit `0dab49f`)
- **Date:** 2026-11-15
- **Toolchain:** Go 1.27.3 darwin/arm64, Node v26.8.2, Wails v2.17.1, Vite v6.4.6, Svelte 5.55.7, Vitest v5.1.0

---

## 1. Summary of Prior Phase Foundations (Phases 0–4)

- **Phase 0:** Offline font bundling (`Inter`, `JetBrains Mono` woff2), hardened CSP separation (dev vs prod), CI matrix workflow.
- **Phase 1:** In-process Go engine integration (`replace github.com/ffuf/ffuf/v2 => ../`), deterministic single-writer `GUIOutputProvider` with `sync.Cond` backpressure gate, monotonic sequential IDs, pure-Go SQLite WAL store (`modernc.org/sqlite`), Zstandard compressed blob storage with UTF-8 boundary truncation, 4-step teardown state machine, upstream patch for `interruptMonitor()` signal leak.
- **Phase 2:** Svelte 5 runes reactive stores (`theme`, `results`, `scan`), 32px virtual data grid with 15-row overscan (10ms for 10,000 items), Apple HIG window chrome with 68px macOS traffic light inset, live controls and telemetry bar.
- **Phase 3:** Hardened dual inspector with sandboxed `<iframe sandbox="">` preview and CSP `default-src 'none'`, 500KB truncation alert, pure SVG analytics dashboard (KPI cards, status code segmented bar, size histogram, word cluster filter injector).
- **Phase 4:** Standalone macOS application bundle (`ffuf-desktop.app` 16.8MB arm64), CI pipeline verification, zero race conditions, zero goroutine leaks.

---

## 2. Phase 8 Scope & Detailed Implementation Plan

### Step 8.0: Initial UI Audit & Baseline Screenshots
- Inspect current UI across Neutral and Ocean themes in both Light and Dark modes.
- Grep codebase for all emoji, Unicode symbol characters (`✓`, `✗`, `▶`, `⚙`, `⚲`, `⚡`, `🔍`, `📋`, `✕`, etc.).
- Audit spacing, typography, focus rings, keyboard accessibility, empty/loading states.
- Generate `docs/phases/UI-AUDIT.md` with prioritized findings (P0/P1/P2).

### Step 8.1: Unified SVG Icon System
- Install `lucide-svelte` (tree-shaken per-icon imports).
- Implement wrapper component `Icon.svelte` (`name`, `size`: 14/16/20/24, `strokeWidth`: 2, `currentColor`).
- Eradicate 100% of emojis and raw Unicode symbols from all `.svelte` and `.ts` files.
- Produce `docs/ICONS.md` mapping functions to icon identifiers.
- Distinct visual icons for HTTP status classes (2xx check, 3xx arrow, 4xx alert-triangle, 5xx x-circle).
- Add CI anti-emoji validation script to prevent regression.

### Step 8.2: Advanced Dual-Layer Filters & Query Builder
- **Data Model:** `FilterRule` & `FilterGroup` (AND/OR, up to 2 nested levels).
- **Dual Layers:**
  1. *View Filter (Client):* Live filtering over cached results in memory and parameterized SQLite queries for 500k+ rows.
  2. *Engine Filter (Live):* Forwarded to `ffuf.Job` live via `AddLiveFilter` (`fc`, `fs`, `fl`, `fw`, `fr`, `ft`). Unsupported operators disabled with informative tooltips.
- Filter Chips bar with quick-add from table cells (right click / context menu: "Hide this status", "Hide this size", "Only this host").
- Advanced Visual Query Builder modal with live match counts ("Hides 3,412 of 52,000").
- Filter Presets (Save, Load, Export/Import JSON) stored in SQLite.
- Undo/Redo history stack (minimum 20 levels).
- Security: ReDoS protection on regex inputs (length cap, complexity/timeout check) and strictly parameterized SQLite queries.
- Differential test: in-memory vs SQLite query results consistency test.

### Step 8.3: Wordlist Manager
- **Go Backend (`internal/wordlists/`):**
  - CRUD operations: `ListWordlists`, `CreateWordlist`, `UpdateWordlist`, `DeleteWordlist`, `ReadWordlistPage`, `SaveWordlistContent`, `ImportWordlistFromPath`, `LinkWordlistPath`, `RelocateWordlist`, `GetWordlistStats`, `ExportWordlist`.
  - Atomic file writing (temp file + `fsync` + rename) with `0700` dir / `0600` file permissions.
  - Path sanitization (`filepath.Clean`, `EvalSymlinks`, reject `/dev/*`, pipe, socket, directory).
  - Streaming line count and preview for large files (>50MB).
  - Concurrency lock preventing deletion or modification while an active scan is reading.
- **Frontend UI:**
  - Dedicated Wordlist Library & Manager view.
  - Built-in Virtualized Line-by-Line Wordlist Editor (supports 1M+ lines without UI freeze).
  - Tools: Deduplicate, Sort, Trim, Lowercase/Uppercase, Strip Empty, Add Prefix/Suffix.
  - Native file picker via `runtime.OpenFileDialog` supporting Link vs Import modes.
  - Integration with Scan Control Bar / Composer with real-time word count & request estimates.

### Step 8.4: UI/UX Polish & Design Token Harmonization
- Standardized 4px spacing scale, button height standards (24/28/32px), focus ring tokens.
- Shared component library: `Button`, `IconButton`, `Input`, `Select`, `Segmented`, `Tooltip`, `Dialog`, `Toast`, `Badge`, `EmptyState`, `Skeleton`, `Tabs`, `Menu`.
- Toast notification manager for background actions.
- Keyboard navigation (Tab order, Escape to dismiss, Command Palette / shortcuts).
- Minimum contrast ratio >= 4.5:1 text, >= 3:1 graphical elements across all 4 theme modes.
- `prefers-reduced-motion` compliance.

---

## 3. Open Risks Table

| Risk Description | Probability | Impact | Mitigation Action | Owner | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Large wordlist memory exhaustion in UI | Medium | Critical | Virtualized editor + paginated streaming reads (`ReadWordlistPage`), Go-side stats calculation. | Fullstack Lead | Open |
| ReDoS vulnerability in user-supplied regex filters | Low | High | Pre-compile with strict character length limits, timeout guards, and friendly UI validation errors. | Security Lead | Open |
| Atomic wordlist write corrupted on crash | Low | High | Temporary file write + fsync + atomic rename; lock guards against concurrent file writes. | Backend Lead | Open |
| Wordlist deleted during active scan | Medium | High | Concurrency check rejecting delete/edit for active scan wordlist with clear error message. | Backend Lead | Open |
| Query builder performance lag on 500k+ rows | Medium | Medium | Parameterized SQLite queries with indexed columns, debounced input, cancellable query contexts. | Backend Lead | Open |
| Inconsistency between SQLite and View filter logic | Medium | High | Differential property testing verifying identical output between SQLite query and in-memory filter. | QA / Test Lead | Open |

---

## 4. Exit Gate Criteria

1. `go vet`, `go test -race -count=1 ./...`, `goleak.VerifyTestMain(m)` all passing.
2. `npm run check`, `npm test` all passing.
3. Anti-emoji CI script passing; `docs/ICONS.md` completed; zero emojis in UI.
4. Comprehensive unit, integration, and differential tests for filters and wordlists.
5. All UI audit items resolved; 4 theme modes verified.
6. Tag `phase-8-done` committed cleanly.
