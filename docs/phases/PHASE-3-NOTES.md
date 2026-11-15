# Phase 3 Handoff Note: Dual Inspector, Sandboxed HTML Preview & Analytics Charts

- **Current Position:** Phase 3 Pre-Start (Planning & Architecture Handoff)
- **Completed Phases:**
  - Phase 0: Wails v2 Desktop Spike & Infrastructure (`phase-0-done`, commit `2f30b28`)
  - Phase 1: Core Engine, Storage, State Machine & Wails Bridge (`phase-1-done`, commit `132793c`)
  - Phase 2: High-Performance Data Grid, Svelte 5 Stores & Apple HIG Foundations (`phase-2-done`, commit `df47e4d`)
- **Date:** 2026-11-15
- **Toolchain:** Go 1.27.3 darwin/arm64, Node v26.8.2, Wails v2.17.1, Vite v6.4.6, Svelte 5.55.7, Vitest v5.1.0

---

## 1. Summary of Phase 2 Achievements (Commit `df47e4d`)

- **Svelte 5 Runes Stores:**
  - `theme.svelte.ts`: Apple Neutral and Ocean color palettes, System/Light/Dark mode syncing.
  - `results.svelte.ts`: Pull-based synchronization on `results:committed`, `$derived.by()` reactive filtering and sorting.
  - `scan.svelte.ts`: Full lifecycle execution control, live telemetry listener (`scan:progress` and `scan:state`).
- **Apple HIG Virtual Grid (`VirtualGrid.svelte`):**
  - Fixed 32px row height virtualizer with 15-row overscan.
  - Benchmarked with 10,000 items: 10ms filter/sort duration.
  - Monospace tabular numbers, semantic status pills, full keyboard navigation (ArrowUp, ArrowDown, Enter).
- **Window Chrome & Controls (`TitleBar.svelte`, `ControlBar.svelte`, `App.svelte`):**
  - macOS 68px traffic light padding inset, draggable titlebar.
  - Method selector, URL with `FUZZ`, wordlist input, thread and rate controls.
  - JSON & CSV export functionality.
- **Verification:**
  - `svelte-check`: 0 errors, 0 warnings.
  - Vitest: 2 tests passed (11ms).
  - Go build & test with race detector: all packages green.

---

## 2. Phase 3 Architecture & Requirements

1. **Dual Inspector Component (`Inspector.svelte`):**
   - Fetches payload dynamically on selection via `GetResultPayload(sessionID, resultID, 500*1024)`.
   - Tabs:
     - `Headers`: Formatted key-value table.
     - `Raw Request`: Raw HTTP text with copy button.
     - `Raw Response`: Raw HTTP text with copy button and 500KB truncation alert.
     - `Rendered Preview`: Rendered inside `<iframe sandbox="">` with `srcdoc`. Hardened internal CSP: `default-src 'none'; form-action 'none'; base-uri 'none'; frame-src 'none'; style-src 'unsafe-inline'`. Absolutely no `allow-scripts` or `allow-same-origin`.
2. **Analytics Dashboard (`AnalyticsDashboard.svelte`):**
   - Lightweight pure SVG charts matching Apple HIG design:
     - Status code distribution donut / bar chart (2xx green, 3xx blue, 4xx orange, 5xx red).
     - Response length distribution histogram.
     - Live throughput chart (Req/sec over time).
3. **Draggable Resizable Split Pane (`SplitPane.svelte`):**
   - Adjustable split between the data grid and bottom/side inspector.
   - Double-click divider to reset split (50/50 or collapse).

---

## 3. Open Risks Table

| Risk Description | Probability | Impact | Mitigation Action | Owner | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| XSS through malicious HTTP response preview | Low | Critical | Strict `<iframe sandbox="">` with `srcdoc` + inner CSP `default-src 'none'`. No script execution allowed. | Security Lead | Open (To be verified in Phase 3) |
| UI freeze on massive response payload in Inspector | Low | Medium | Go-side 500KB cap (`GetResultPayload`) on UTF-8 boundary. UI only receives pre-truncated text. | Frontend Lead | Open (To be verified in Phase 3) |
| Layout thrashing during drag resize of split pane | Low | Low | Resize handler throttled via `requestAnimationFrame`. | Frontend Lead | Open (To be verified in Phase 3) |

---

## 4. Next Steps (Phase 3 Detailed Plan)

1. **Step 3.1: Create SplitPane Component (`src/lib/components/SplitPane.svelte`):**
   - Flexible horizontal/vertical resizable container with persistent split ratio.
2. **Step 3.2: Create Hardened Dual Inspector (`src/lib/components/Inspector.svelte`):**
   - Sandboxed iframe with CSP, raw viewer, truncation warning, header table.
3. **Step 3.3: Create Analytics Dashboard (`src/lib/components/AnalyticsDashboard.svelte`):**
   - Pure SVG status code breakdown, size histogram, and metrics cards.
4. **Step 3.4: Integrate with `App.svelte`:**
   - Tabs to switch between Table + Inspector view and Analytical Dashboard view.
5. **Step 3.5: Run Full Verification & Testing:**
   - Vitest component tests, `svelte-check`, production build, and `phase-3-done` tag.
