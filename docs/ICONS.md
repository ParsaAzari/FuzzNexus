# Unified Icon System Specification (`docs/ICONS.md`)

- **Library:** `@lucide/svelte`
- **Wrapper Component:** `src/lib/components/Icon.svelte`
- **Standard Stroke Width:** `2px`
- **Color Mode:** `currentColor` (inherits surrounding CSS text/foreground color)
- **Standard Sizes:**
  - `14px` (micro/inline badges, table sort indicators)
  - `16px` (standard button labels, inputs, toolbar icons)
  - `20px` (prominent action buttons, tab headers)
  - `32px` - `48px` (empty state placeholders)

---

## 1. Icon Mapping Registry

| Identifier (`name`) | Lucide Component | Standard Size | Primary Use Case |
| :--- | :--- | :--- | :--- |
| `play` | `Play` | 16px | Start / Resume engine scan |
| `pause` | `Pause` | 16px | Pause engine scan |
| `stop` / `square` | `Square` | 16px | Terminate scan execution |
| `zap` | `Zap` | 14px / 16px | Live engine filter indicator |
| `check` | `Check` | 16px | Copy feedback, completed confirmation |
| `check-circle` | `CheckCircle2` | 14px | Success states, valid tokens |
| `status-2xx` | `CheckCircle2` | 14px | HTTP 2xx (Success) response badge |
| `status-3xx` | `CornerDownRight` | 14px | HTTP 3xx (Redirection) response badge |
| `status-4xx` | `ShieldAlert` | 14px | HTTP 4xx (Client Error) response badge |
| `status-5xx` | `AlertOctagon` | 14px | HTTP 5xx (Server Error) response badge |
| `status-err` | `Ban` | 14px | Network / Protocol / Socket error |
| `alert-triangle` | `AlertTriangle` | 16px | Warning alerts, non-fatal engine error banner |
| `alert-circle` | `AlertCircle` | 16px | Payload truncation notice, validation error |
| `alert-octagon` | `AlertOctagon` | 16px | Critical / fatal error alert |
| `info` | `Info` | 16px | Informational tooltips & hints |
| `search` | `Search` | 16px | Title bar quick search, query builder |
| `x` / `close` | `X` | 14px / 16px | Clear search input, dismiss modals / panels |
| `copy` | `Copy` | 14px | Copy response / request / URL to clipboard |
| `filter` | `Filter` | 16px | Advanced filter builder toggle, active filter chip |
| `sliders` | `SlidersHorizontal` | 16px | Scan rate limits, engine tuning |
| `arrow-up` | `ArrowUp` | 14px | Table column ascending sort |
| `arrow-down` | `ArrowDown` | 14px | Table column descending sort |
| `arrow-up-down` | `ArrowUpDown` | 14px | Table unsorted indicator |
| `chevron-down` | `ChevronDown` | 14px | Dropdown selector caret |
| `chevron-right` | `ChevronRight` | 14px | Tree / collapsible disclosure indicator |
| `corner-down-right` | `CornerDownRight` | 14px | Redirect indicator |
| `sun` | `Sun` | 16px | Light theme mode option |
| `moon` | `Moon` | 16px | Dark theme mode option |
| `laptop` / `monitor` | `Laptop` | 16px | System / Auto theme option |
| `file-text` | `FileText` | 16px / 32px | Raw HTTP request / response view, wordlist editor |
| `layers` | `Layers` | 16px / 32px | Inspector preview, parsed headers |
| `plus` | `Plus` | 14px / 16px | Add filter rule, create wordlist |
| `trash` | `Trash2` | 14px / 16px | Delete filter rule, remove wordlist |
| `edit` | `Edit3` | 14px / 16px | Edit filter rule, rename wordlist |
| `save` | `Save` | 16px | Save filter preset, save wordlist content |
| `download` | `Download` | 16px | Export results / wordlist / presets |
| `upload` | `Upload` | 16px | Import wordlist / load preset |
| `refresh` | `RefreshCw` | 16px | Reload wordlists, reset filters |
| `undo` | `Undo2` | 14px | Filter undo stack |
| `redo` | `Redo2` | 14px | Filter redo stack |
| `folder` | `FolderOpen` | 16px | Browse local wordlist file dialog |
| `eye` | `Eye` | 14px | Enable filter rule |
| `eye-off` | `EyeOff` | 14px | Disable filter rule |
| `terminal` | `Terminal` | 16px | CLI command preview generator |
| `database` | `Database` | 16px | SQLite results storage |
| `globe` | `Globe` | 16px | Target URL / domain |

---

## 2. Anti-Emoji Verification Policy

- **Rule:** Zero Unicode emojis (range `\u1F300-\u1F9FF`), legacy symbols (`\u2600-\u26FF`, `\u2700-\u27BF`), or raw geometric arrows are allowed in source code (.svelte, .ts, .html).
- **Enforcement:** Automated script `scripts/check-no-emojis.py` runs during `npm test` and CI workflow.
