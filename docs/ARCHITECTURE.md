# FuzzNexus Architecture

FuzzNexus is a cross-platform desktop web fuzzer built on **Go + Wails v2 + Svelte 5 (Runes) + SQLite**, sharing the core fuzzing engine (`pkg/*`) with the CLI.

---

## Directory Structure

```text
.
├── go.mod                      # Root Go module (github.com/parsaazari/fuzznexus)
├── main.go / help.go           # CLI entrypoint & segmented help output
├── pkg/                        # Shared core fuzzing engine
│   ├── fuzznexus/              # Config, ConfigOptions, CLI flags (cli:"..."), browser profiles
│   ├── engine/                 # Job runner, RateThrottle, CircuitBreaker, atomic TX/RX counters
│   ├── runner/                 # HTTP/1.1, HTTP/2, uTLS, mTLS, proxy, replay-proxy
│   ├── filter/                 # Matchers & filters (MatcherManager)
│   ├── input/                  # Wordlists, Clusterbomb/Pitchfork/Sniper, pencode encoders
│   ├── output/                 # CLI & AuditLogger
│   ├── scraper/                # Response data extractors
│   └── assembly/               # Job assembly
├── testlab/                    # Docker WAF / 502 / netem integration tests
└── fuzznexus-desktop/          # Wails v2 Desktop Application (independent go.mod)
    ├── main.go / app.go        # Wails app lifecycle & native dialogs
    ├── wails.json              # Build configuration
    ├── internal/
    │   ├── bridge/             # Service, ScanRequestDTO, StartScan, ExportResults, ImportTOMLConfig
    │   ├── scan/               # Session lifecycle, NormalizeTargetWithOptions, ProgressDTO
    │   ├── store/              # SQLite persistence (fuzznexus.db), sessions, filter presets
    │   ├── wordlists/          # Wordlist library, sanitizer, risk analyzer
    │   ├── output/             # GUIOutputProvider (live events + -od matched file saving)
    │   ├── blobs/              # Request/response payload blob storage
    │   └── dock/               # macOS dock badge integration
    └── frontend/               # Svelte 5 + TypeScript + Vite
        └── src/
            ├── App.svelte      # Shell, footer telemetry (TX/RX/Rate, Proxy pill, Net Log), global shortcuts
            └── lib/
                ├── types.ts    # TypeScript DTOs aligned with Wails bindings
                ├── stores/     # Reactive stores (scan, results, filters, wordlists, shortcuts, theme, toast)
                └── components/ # UI components (ControlBar, VirtualGrid, Inspector, RequestOptionsDrawer, ShortcutsModal, ...)
```

---

## Key Architectural Guarantees
- **Single Engine:** Both CLI and GUI execute through `pkg/engine.Job`, `pkg/runner`, and `pkg/filter`.
- **Non-Blocking Telemetry:** `TX`/`RX` byte counters in `pkg/engine/job.go` use `atomic.Int64` and are smoothed in `scan.svelte.ts` via EMA to prevent UI jitter or freezes.
- **WKWebView-Safe UI:** Table column reordering in `VirtualGrid.svelte` uses Pointer Events (`onpointerdown`/`pointermove`/`pointerup`) instead of HTML5 `draggable="true"` to avoid macOS WKWebView drag crashes.
- **Cross-Platform Shortcuts:** `shortcuts.svelte.ts` maps `Mod` to `Cmd` (macOS) or `Ctrl` (Windows/Linux) with a 2-tier guard (hard-blocking OS-reserved combos, soft-warning on text-editing/browser overlaps).
