# UI/UX Audit & Prioritized Findings (Phase 8 Baseline)

- **Date:** 2026-11-15
- **Branch:** `main`
- **Scope:** Complete frontend audit of `ffuf-desktop/frontend/src` across all themes (Neutral Light/Dark, Ocean Light/Dark).
- **Standards:** Apple Human Interface Guidelines (HIG), WCAG 2.1 AA Contrast (>= 4.5:1 text, >= 3:1 graphical elements), Strict Offline Bundle, Air-Gapped SVG Iconography.

---

## 1. Inventory of Emojis and Raw Unicode Symbols

Automated scans detected 26 raw Unicode/emoji characters embedded in frontend components:

| Component | Line(s) | Character | Usage Context | Recommended SVG Icon (`lucide-svelte`) |
| :--- | :--- | :--- | :--- | :--- |
| `ControlBar.svelte` | 103, 111 | `▶` | Start / Resume Scan button | `Play` |
| `ControlBar.svelte` | 107 | `⏸` | Pause Scan button | `Pause` |
| `ControlBar.svelte` | 117 | `⏹` | Stop Scan button | `Square` |
| `ControlBar.svelte` | 155 | `⚡` | Live filter indicator | `Zap` |
| `ControlBar.svelte` | 177 | `⚠️` | Scan error alert banner | `AlertTriangle` |
| `VirtualGrid.svelte` | 85-106 | `↑`, `↓` | Column header sort direction | `ArrowUp`, `ArrowDown` |
| `VirtualGrid.svelte` | 120 | `🔍` | Empty results placeholder | `Search` |
| `TitleBar.svelte` | 34 | `⚲` | Search input prefix | `Search` |
| `TitleBar.svelte` | 42 | `✕` | Search clear button | `X` |
| `TitleBar.svelte` | 100 | `☀️` | Light mode select option | `Sun` |
| `TitleBar.svelte` | 102 | `🌙` | Dark mode select option | `Moon` |
| `TitleBar.svelte` | 104 | `⚙️` | Auto mode select option | `Laptop` / `Monitor` |
| `Inspector.svelte` | 83 | `📋` | Empty inspector placeholder | `FileText` / `Layers` |
| `Inspector.svelte` | 149 | `✓` | Copied to clipboard feedback | `Check` |
| `Inspector.svelte` | 157 | `✕` | Close inspector pane button | `X` |
| `Inspector.svelte` | 176 | `⚠️` | 500KB truncation alert banner | `AlertCircle` |
| `Inspector.svelte` | 212 | `•` | Meta value separator | Styled CSS dot or `span.separator` |
| `App.svelte` | 76 | `•` | Status bar selection separator | Styled CSS dot or `span.separator` |

---

## 2. Color Contrast & Theme Verification (WCAG AA)

| Token / Element | Neutral Light | Neutral Dark | Ocean Light | Ocean Dark | Compliance (>= 4.5:1) | Remedy |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Primary Text on Surface | `#1D1D1F` on `#FFFFFF` (16.1:1) | `#F5F5F7` on `#2C2C2E` (11.5:1) | `#0B1B3F` on `#FFFFFF` (16.8:1) | `#E8EEFF` on `#16203A` (13.2:1) | **Pass** | None |
| Secondary Text on Surface | `#6E6E73` on `#FFFFFF` (4.58:1) | `#98989D` on `#2C2C2E` (4.62:1) | `#4D628F` on `#FFFFFF` (5.1:1) | `#93A4CE` on `#16203A` (5.7:1) | **Pass (Borderline)** | Ensure minimum 12px font size |
| Status 2xx (`--status-2xx`) | `#34C759` on `#FFFFFF` (2.8:1) | `#34C759` on `#2C2C2E` (3.9:1) | `#34C759` on `#FFFFFF` (2.8:1) | `#34C759` on `#16203A` (4.4:1) | **Fail (as text)** | Use high-contrast badge styles: light mode `#1F7A38`, dark mode `#34C759` with subtle background tint |
| Status 3xx (`--status-3xx`) | `#007AFF` on `#FFFFFF` (4.0:1) | `#0A84FF` on `#2C2C2E` (4.8:1) | `#1F6FEB` on `#FFFFFF` (4.9:1) | `#4C8DFF` on `#16203A` (5.6:1) | **Fail (in Neutral Light)** | Adjust light mode 3xx to `#0062CC` (5.2:1) |
| Status 4xx (`--status-4xx`) | `#FF9500` on `#FFFFFF` (2.1:1) | `#FF9F0A` on `#2C2C2E` (3.0:1) | `#FF9500` on `#FFFFFF` (2.1:1) | `#FF9F0A` on `#16203A` (3.4:1) | **Fail (as text)** | Adjust light mode 4xx text to `#B25900` (5.3:1) or use badge background fill |
| Status 5xx (`--status-5xx`) | `#FF3B30` on `#FFFFFF` (4.0:1) | `#FF453A` on `#2C2C2E` (4.2:1) | `#FF3B30` on `#FFFFFF` (4.0:1) | `#FF453A` on `#16203A` (4.8:1) | **Fail (in Light Modes)** | Adjust light mode 5xx text to `#D70015` (4.9:1) |

---

## 3. Spacing, Typography & Focus Rings

1. **Spacing System:** Current paddings and margins mix `5px`, `6px`, `10px`, `12px`, and `14px`. Standardize on a strict **4px scale**:
   - Spacing: `4px` (`var(--space-1)`), `8px` (`var(--space-2)`), `12px` (`var(--space-3)`), `16px` (`var(--space-4)`), `24px` (`var(--space-6)`).
   - Component Heights: `24px` (micro/badge), `28px` (compact/filter chips), `32px` (standard input/buttons), `38px` (navigation/control bar).
2. **Focus Visibility:** Several buttons and icon triggers currently lack explicit `:focus-visible` styling. Every interactive element must display:
   ```css
   outline: 2px solid var(--accent);
   outline-offset: 2px;
   ```
3. **Empty States:** The current empty states in `VirtualGrid` and `Inspector` are basic placeholders. Replace with structured empty states with custom SVG illustrations, title, subtitle, and hotkey hints.

---

## 4. Prioritized Action Plan

| Priority | ID | Description | Target Step |
| :--- | :--- | :--- | :--- |
| **P0** | **AUD-01** | Replace all emojis and raw Unicode symbols with `@lucide/svelte` unified icon wrapper `Icon.svelte`. | Step 8.1 |
| **P0** | **AUD-02** | Create `docs/ICONS.md` mapping application actions to icon identifiers. | Step 8.1 |
| **P0** | **AUD-03** | Implement distinct HTTP status class icons (2xx Check, 3xx CornerDownRight, 4xx ShieldAlert, 5xx AlertOctagon). | Step 8.1 |
| **P0** | **AUD-04** | Add automated CI anti-emoji script (`scripts/check-no-emojis.sh`) and npm test task. | Step 8.1 |
| **P1** | **AUD-05** | Implement Advanced Dual-Layer Filters (`FilterRule`, `FilterGroup`, in-memory View filter & SQLite parameterized queries). | Step 8.2 |
| **P1** | **AUD-06** | Visual Query Builder modal + Filter chips with right-click context menu ("Hide status", "Only host"). | Step 8.2 |
| **P1** | **AUD-07** | Filter presets (save/load/export/import JSON) and 20-level undo/redo stack. | Step 8.2 |
| **P1** | **AUD-08** | Implement Wordlist Manager in Go (`internal/wordlists/`) with atomic writes, sanitization, concurrency locks, and streaming pagination. | Step 8.3 |
| **P1** | **AUD-09** | Implement frontend Wordlist Manager view with virtualized line editor and wordlist transformation tools. | Step 8.3 |
| **P2** | **AUD-10** | Standardize design tokens in `style.css` (4px spacing scale, button height standards 24/28/32px). | Step 8.4 |
| **P2** | **AUD-11** | Shared UI component library (`Button`, `IconButton`, `Input`, `Select`, `Badge`, `Toast`, `Tooltip`, `Dialog`). | Step 8.4 |
| **P2** | **AUD-12** | Fix status code text contrast with theme-aware high-contrast values or pill backgrounds (WCAG AA >= 4.5:1). | Step 8.4 |
| **P2** | **AUD-13** | Add `prefers-reduced-motion` media queries and keyboard shortcut bindings. | Step 8.4 |
