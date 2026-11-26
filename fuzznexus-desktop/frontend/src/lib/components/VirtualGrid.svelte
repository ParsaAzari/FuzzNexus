<script lang="ts">
  import { resultsStore } from '../stores/results.svelte';
  import { filtersStore } from '../stores/filters.svelte';
  import type { ResultDTO, SortField, FilterField, FilterOperator } from '../types';
  import Icon from './Icon.svelte';

  const ROW_HEIGHT = 32;
  const OVERSCAN = 15;

  let containerRef = $state<HTMLDivElement | null>(null);
  let scrollTop = $state<number>(0);
  let containerHeight = $state<number>(600);
  let contextMenu = $state<{ x: number; y: number; row: ResultDTO } | null>(null);

  type ColumnKey = 'id' | 'status' | 'payload' | 'url' | 'length' | 'words' | 'lines' | 'duration';
  interface ColumnDef {
    key: ColumnKey;
    label: string;
    width: number;
    minWidth: number;
    flex?: boolean;
    sortField?: SortField;
  }

  let columns = $state<ColumnDef[]>([
    { key: 'id', label: '#', width: 56, minWidth: 44, sortField: 'id' },
    { key: 'status', label: 'Status', width: 90, minWidth: 72, sortField: 'status' },
    { key: 'payload', label: 'Payload', width: 170, minWidth: 90 },
    { key: 'url', label: 'Target URL', width: 320, minWidth: 160, flex: true, sortField: 'url' },
    { key: 'length', label: 'Size', width: 88, minWidth: 64, sortField: 'length' },
    { key: 'words', label: 'Words', width: 78, minWidth: 56, sortField: 'words' },
    { key: 'lines', label: 'Lines', width: 78, minWidth: 56, sortField: 'lines' },
    { key: 'duration', label: 'Time', width: 78, minWidth: 56, sortField: 'duration' }
  ]);

  let draggedColKey = $state<ColumnKey | null>(null);
  let dragOverColKey = $state<ColumnKey | null>(null);
  let dragGhost = $state<{ label: string; x: number; y: number } | null>(null);
  let headerRowRef = $state<HTMLDivElement | null>(null);
  let suppressNextSortClick = false;

  function startColDrag(e: PointerEvent, key: ColumnKey) {
    if (e.button !== 0) return;
    const target = e.target as HTMLElement | null;
    if (target?.closest('.col-resizer')) return;

    const startX = e.clientX;
    let dragging = false;
    const colLabel = columns.find((c) => c.key === key)?.label ?? '';

    function onMove(ev: PointerEvent) {
      if (!dragging && Math.abs(ev.clientX - startX) > 6) {
        dragging = true;
        draggedColKey = key;
        suppressNextSortClick = true;
      }
      if (!dragging || !headerRowRef) return;
      dragGhost = { label: colLabel, x: ev.clientX, y: ev.clientY };

      const cells = headerRowRef.querySelectorAll<HTMLElement>('[data-col-key]');
      let hovered: ColumnKey | null = null;
      for (const cell of cells) {
        const rect = cell.getBoundingClientRect();
        if (ev.clientX >= rect.left && ev.clientX <= rect.right) {
          const k = cell.dataset.colKey as ColumnKey | undefined;
          if (k && k !== key) {
            hovered = k;
          }
          break;
        }
      }
      dragOverColKey = hovered;
    }

    function onUp() {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
      if (dragging && draggedColKey && dragOverColKey && draggedColKey !== dragOverColKey) {
        const fromIdx = columns.findIndex((c) => c.key === draggedColKey);
        const toIdx = columns.findIndex((c) => c.key === dragOverColKey);
        if (fromIdx !== -1 && toIdx !== -1) {
          const next = [...columns];
          const [moved] = next.splice(fromIdx, 1);
          next.splice(toIdx, 0, moved);
          columns = next;
        }
      }
      draggedColKey = null;
      dragOverColKey = null;
      dragGhost = null;
      if (dragging) {
        setTimeout(() => {
          suppressNextSortClick = false;
        }, 0);
      }
    }

    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
  }

  function startColResize(e: PointerEvent, key: ColumnKey) {
    e.stopPropagation();
    e.preventDefault();
    const col = columns.find((c) => c.key === key);
    if (!col) return;
    const startX = e.clientX;
    const startW = col.width;

    function onMove(ev: PointerEvent) {
      const nextW = Math.max(col!.minWidth, startW + (ev.clientX - startX));
      columns = columns.map((c) => (c.key === key ? { ...c, width: nextW, flex: false } : c));
    }

    function onUp() {
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    }

    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
  }

  function getColStyle(col: ColumnDef): string {
    return col.flex ? `flex: 1 1 ${col.width}px; min-width: ${col.minWidth}px;` : `width: ${col.width}px; min-width: ${col.minWidth}px; flex-shrink: 0;`;
  }

  let items = $derived(resultsStore.filteredItems);
  let totalCount = $derived(items.length);
  let totalHeight = $derived(totalCount * ROW_HEIGHT);

  let startIndex = $derived(Math.max(0, Math.floor(scrollTop / ROW_HEIGHT) - OVERSCAN));
  let endIndex = $derived(Math.min(totalCount, Math.ceil((scrollTop + containerHeight) / ROW_HEIGHT) + OVERSCAN));
  let visibleRows = $derived(items.slice(startIndex, endIndex));

  let topPadding = $derived(startIndex * ROW_HEIGHT);
  let bottomPadding = $derived(Math.max(0, (totalCount - endIndex) * ROW_HEIGHT));

  function handleScroll(e: Event) {
    const target = e.currentTarget as HTMLDivElement;
    scrollTop = target.scrollTop;
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (totalCount === 0) return;
    const currentID = resultsStore.selectedID;
    const currentIndex = currentID ? items.findIndex((r) => r.id === currentID) : -1;

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      const nextIndex = Math.min(totalCount - 1, currentIndex + 1);
      resultsStore.select(items[nextIndex].id);
      scrollRowIntoView(nextIndex);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      const prevIndex = Math.max(0, currentIndex - 1);
      resultsStore.select(items[prevIndex].id);
      scrollRowIntoView(prevIndex);
    }
  }

  function scrollRowIntoView(index: number) {
    if (!containerRef) return;
    const rowTop = index * ROW_HEIGHT;
    const rowBottom = rowTop + ROW_HEIGHT;
    if (rowTop < scrollTop) {
      containerRef.scrollTop = rowTop;
    } else if (rowBottom > scrollTop + containerHeight) {
      containerRef.scrollTop = rowBottom - containerHeight;
    }
  }

  function getStatusBadgeClass(status: number): string {
    if (status >= 200 && status < 300) return 'badge-2xx';
    if (status >= 300 && status < 400) return 'badge-3xx';
    if (status >= 400 && status < 500) return 'badge-4xx';
    if (status >= 500) return 'badge-5xx';
    return 'badge-err';
  }

  function getStatusIconName(status: number): string {
    if (status >= 200 && status < 300) return 'status-2xx';
    if (status >= 300 && status < 400) return 'status-3xx';
    if (status >= 400 && status < 500) return 'status-4xx';
    if (status >= 500) return 'status-5xx';
    return 'status-err';
  }

  function getFirstInputValue(inputs?: Record<string, string>): string {
    if (!inputs) return '';
    const keys = Object.keys(inputs);
    if (keys.length === 0) return '';
    if (inputs['FUZZ']) return inputs['FUZZ'];
    return inputs[keys[0]];
  }

  function formatRowPayloads(inputs?: Record<string, string>): string {
    if (!inputs) return '';
    const entries = Object.entries(inputs);
    if (entries.length === 0) return '';
    if (entries.length === 1) return entries[0][1];
    return entries.map(([k, v]) => `${k}=${v}`).join(' · ');
  }

  function handleContextMenu(e: MouseEvent, row: ResultDTO) {
    e.preventDefault();
    contextMenu = { x: e.clientX, y: e.clientY, row };
  }

  function closeContextMenu() {
    contextMenu = null;
  }

  function handleQuickFilter(field: FilterField, operator: FilterOperator, value: string) {
    filtersStore.toggleQuickFilter(field, operator, value);
    closeContextMenu();
  }
</script>

<div
  class="virtual-grid-wrapper select-none"
  role="grid"
  aria-label="Fuzzing Results Grid"
  aria-rowcount={totalCount}
  tabindex="0"
  onkeydown={handleKeyDown}
>
  <!-- Table Header -->
  {#if dragGhost}
    <div
      class="col-drag-ghost font-mono"
      style="transform: translate3d({dragGhost.x + 12}px, {dragGhost.y - 14}px, 0);"
    >
      {dragGhost.label}
    </div>
  {/if}
  <div class="grid-header" role="row" bind:this={headerRowRef}>
    {#each columns as col (col.key)}
      <div
        class="col col-{col.key} header-cell"
        class:drag-over={dragOverColKey === col.key}
        class:dragging={draggedColKey === col.key}
        style={getColStyle(col)}
        data-col-key={col.key}
        onpointerdown={(e) => startColDrag(e, col.key)}
        role="columnheader"
        tabindex="0"
      >
        {#if col.sortField}
          <button
            type="button"
            class="header-btn"
            onclick={() => {
              if (!suppressNextSortClick) resultsStore.setSort(col.sortField!);
            }}
            aria-label="Sort by {col.label}"
          >
            <span>{col.label}</span>
            {#if resultsStore.sortField === col.sortField}
              <Icon name={resultsStore.sortDirection === 'asc' ? 'arrow-up' : 'arrow-down'} size={12} />
            {/if}
          </button>
        {:else}
          <span class="header-btn">{col.label}</span>
        {/if}
        <div
          class="col-resizer"
          onpointerdown={(e) => startColResize(e, col.key)}
          role="separator"
          aria-orientation="vertical"
          title="Drag to resize column"
        ></div>
      </div>
    {/each}
  </div>

  <!-- Scroll Container -->
  <div
    class="grid-body-scroll"
    bind:this={containerRef}
    bind:clientHeight={containerHeight}
    onscroll={handleScroll}
    role="presentation"
  >
    {#if totalCount === 0}
      <div class="empty-state">
        <div class="empty-icon">
          <Icon name="search" size={40} />
        </div>
        <div class="empty-title">No Scan Results</div>
        <div class="empty-sub">Start a fuzzing scan or adjust filters above to view real-time matches.</div>
      </div>
    {:else}
      <div
        class="virtual-spacer"
        style="padding-top: {topPadding}px; padding-bottom: {bottomPadding}px;"
      >
        {#each visibleRows as row, i (row.id)}
          <div
            class="grid-row select-text"
            class:zebra-odd={(startIndex + i) % 2 === 1}
            class:selected={resultsStore.selectedID === row.id}
            style="height: {ROW_HEIGHT}px;"
            role="row"
            tabindex="0"
            onclick={() => resultsStore.select(row.id)}
            oncontextmenu={(e) => handleContextMenu(e, row)}
            onkeydown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                resultsStore.select(row.id);
              }
            }}
          >
            {#each columns as col (col.key)}
              {#if col.key === 'id'}
                <div class="col col-id font-mono" style={getColStyle(col)}>{row.id}</div>
              {:else if col.key === 'status'}
                <div class="col col-status" style={getColStyle(col)}>
                  <span class="status-badge font-mono {getStatusBadgeClass(row.status)}">
                    <Icon name={getStatusIconName(row.status)} size={11} />
                    <span>{row.status}</span>
                  </span>
                </div>
              {:else if col.key === 'payload'}
                <div class="col col-payload font-mono payload-text" style={getColStyle(col)} title={formatRowPayloads(row.inputs)}>
                  {formatRowPayloads(row.inputs)}
                </div>
              {:else if col.key === 'url'}
                <div class="col col-url font-mono url-text" style={getColStyle(col)} title={row.url}>
                  {row.url}
                </div>
              {:else if col.key === 'length'}
                <div class="col col-length font-mono" style={getColStyle(col)}>{row.length.toLocaleString()}</div>
              {:else if col.key === 'words'}
                <div class="col col-words font-mono" style={getColStyle(col)}>{row.words.toLocaleString()}</div>
              {:else if col.key === 'lines'}
                <div class="col col-lines font-mono" style={getColStyle(col)}>{row.lines.toLocaleString()}</div>
              {:else if col.key === 'duration'}
                <div class="col col-duration font-mono" style={getColStyle(col)}>{row.duration_ms}ms</div>
              {/if}
            {/each}
          </div>
        {/each}
      </div>
    {/if}
  </div>

  {#if contextMenu}
    {@const isOnlyStatusActive = filtersStore.hasFilter('status', '=', String(contextMenu.row.status))}
    {@const isHideStatusActive = filtersStore.hasFilter('status', '!=', String(contextMenu.row.status))}
    {@const isOnlySizeActive = filtersStore.hasFilter('length', '=', String(contextMenu.row.length))}
    {@const isHideSizeActive = filtersStore.hasFilter('length', '!=', String(contextMenu.row.length))}
    {@const isOnlyHostActive = contextMenu.row.host ? filtersStore.hasFilter('host', '=', contextMenu.row.host) : false}
    <div
      class="context-menu-backdrop"
      role="menu"
      tabindex="-1"
      onclick={(e) => { if (e.target === e.currentTarget) closeContextMenu(); }}
      onkeydown={(e) => e.key === 'Escape' && closeContextMenu()}
    >
      <div
        class="context-menu"
        role="presentation"
        style="top: {contextMenu.y}px; left: {contextMenu.x}px;"
      >
        <div class="menu-header font-mono">Row #{contextMenu.row.id}</div>

        <button
          class="menu-item"
          class:item-active={isOnlyStatusActive}
          onclick={() => handleQuickFilter('status', '=', String(contextMenu?.row.status))}
        >
          <Icon name={isOnlyStatusActive ? "check" : "filter"} size={13} />
          <span>Only Status {contextMenu.row.status}</span>
          {#if isOnlyStatusActive}
            <span class="menu-item-badge">Active</span>
          {/if}
        </button>
        <button
          class="menu-item"
          class:item-active={isHideStatusActive}
          onclick={() => handleQuickFilter('status', '!=', String(contextMenu?.row.status))}
        >
          <Icon name={isHideStatusActive ? "check" : "x"} size={13} />
          <span>Hide Status {contextMenu.row.status}</span>
          {#if isHideStatusActive}
            <span class="menu-item-badge">Active</span>
          {/if}
        </button>
        <div class="menu-divider"></div>
        <button
          class="menu-item"
          class:item-active={isOnlySizeActive}
          onclick={() => handleQuickFilter('length', '=', String(contextMenu?.row.length))}
        >
          <Icon name={isOnlySizeActive ? "check" : "filter"} size={13} />
          <span>Only Size {contextMenu.row.length}</span>
          {#if isOnlySizeActive}
            <span class="menu-item-badge">Active</span>
          {/if}
        </button>
        <button
          class="menu-item"
          class:item-active={isHideSizeActive}
          onclick={() => handleQuickFilter('length', '!=', String(contextMenu?.row.length))}
        >
          <Icon name={isHideSizeActive ? "check" : "x"} size={13} />
          <span>Hide Size {contextMenu.row.length}</span>
          {#if isHideSizeActive}
            <span class="menu-item-badge">Active</span>
          {/if}
        </button>
        {#if contextMenu.row.host}
          <div class="menu-divider"></div>
          <button
            class="menu-item"
            class:item-active={isOnlyHostActive}
            onclick={() => handleQuickFilter('host', '=', contextMenu?.row.host || '')}
          >
            <Icon name={isOnlyHostActive ? "check" : "globe"} size={13} />
            <span>Only Host {contextMenu.row.host}</span>
            {#if isOnlyHostActive}
              <span class="menu-item-badge">Active</span>
            {/if}
          </button>
        {/if}
        <div class="menu-divider"></div>
        <button
          class="menu-item"
          onclick={() => {
            navigator.clipboard.writeText(contextMenu?.row.url || '');
            closeContextMenu();
          }}
        >
          <Icon name="copy" size={13} />
          <span>Copy Target URL</span>
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .virtual-grid-wrapper {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    background-color: var(--surface);
    border-top: 1px solid var(--border);
    outline: none;
  }

  .grid-header {
    display: flex;
    height: 32px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    user-select: none;
  }

  .header-btn {
    all: unset;
    cursor: pointer;
    display: flex;
    align-items: center;
    box-sizing: border-box;
    padding: 0 8px;
    transition: background 0.15s ease;
  }

  .header-btn:hover {
    background-color: var(--surface-hover);
    color: var(--text);
  }

  .grid-body-scroll {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    position: relative;
    outline: none;
  }

  .grid-row {
    display: flex;
    align-items: center;
    box-sizing: border-box;
    border-bottom: 1px solid rgba(128, 128, 128, 0.1);
    font-size: var(--text-base);
    cursor: pointer;
    transition: background 0.08s ease;
  }

  .grid-row.zebra-odd {
    background-color: rgba(128, 128, 128, 0.045);
  }

  .grid-row:hover {
    background-color: var(--surface-hover);
  }

  .header-cell {
    position: relative;
    display: flex;
    align-items: center;
    cursor: grab;
  }

  .header-cell.dragging {
    opacity: 0.38;
    cursor: grabbing;
  }

  .header-cell.drag-over {
    box-shadow: inset 2px 0 0 var(--accent);
    background-color: rgba(59, 130, 246, 0.08);
  }

  .col-drag-ghost {
    position: fixed;
    top: 0;
    left: 0;
    z-index: 9999;
    pointer-events: none;
    padding: 4px 10px;
    border-radius: var(--radius-sm);
    background-color: var(--surface);
    border: 1px solid var(--accent);
    color: var(--text);
    font-size: var(--text-xs);
    font-weight: 600;
    box-shadow: 0 8px 20px rgba(0, 0, 0, 0.22);
    opacity: 0.92;
  }

  .col-resizer {
    position: absolute;
    right: 0;
    top: 4px;
    bottom: 4px;
    width: 6px;
    cursor: col-resize;
    border-right: 1px solid transparent;
  }

  .col-resizer:hover {
    border-right-color: var(--accent);
  }

  .grid-row.selected {
    background-color: var(--accent) !important;
    color: var(--accent-contrast) !important;
  }

  .grid-row.selected .status-badge {
    background: rgba(255, 255, 255, 0.25) !important;
    color: #ffffff !important;
    border-color: transparent !important;
  }

  .grid-row.selected .col-payload,
  .grid-row.selected .col-url,
  .grid-row.selected .font-mono {
    color: var(--accent-contrast) !important;
  }

  .col {
    padding: 0 8px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    box-sizing: border-box;
  }

  .col-id {
    width: 65px;
    text-align: right;
    color: var(--text-secondary);
  }

  .col-status {
    width: 85px;
    display: flex;
    justify-content: center;
  }

  .col-payload {
    width: 140px;
    font-weight: 600;
  }

  .col-url {
    flex: 1;
    min-width: 200px;
  }

  .col-length {
    width: 90px;
    text-align: right;
  }

  .col-words {
    width: 75px;
    text-align: right;
  }

  .col-lines {
    width: 75px;
    text-align: right;
  }

  .col-duration {
    width: 85px;
    text-align: right;
  }

  .status-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 1px 6px;
    border-radius: var(--radius-xs);
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    line-height: 16px;
  }

  .badge-2xx {
    background-color: rgba(52, 199, 89, 0.15);
    color: #248a3d;
    border: 1px solid rgba(52, 199, 89, 0.3);
  }

  .badge-3xx {
    background-color: rgba(0, 122, 255, 0.15);
    color: #007aff;
    border: 1px solid rgba(0, 122, 255, 0.3);
  }

  .badge-4xx {
    background-color: rgba(255, 149, 0, 0.15);
    color: #c97700;
    border: 1px solid rgba(255, 149, 0, 0.3);
  }

  .badge-5xx {
    background-color: rgba(255, 59, 48, 0.15);
    color: #d7261d;
    border: 1px solid rgba(255, 59, 48, 0.3);
  }

  .badge-err {
    background-color: rgba(142, 142, 147, 0.15);
    color: #636366;
    border: 1px solid rgba(142, 142, 147, 0.3);
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    min-height: 300px;
    color: var(--text-secondary);
    user-select: none;
  }

  .empty-icon {
    font-size: 42px;
    margin-bottom: 12px;
    opacity: 0.6;
  }

  .empty-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--text);
    margin-bottom: 4px;
  }

  .empty-sub {
    font-size: 12px;
    max-width: 320px;
    text-align: center;
    line-height: 1.5;
  }

  .context-menu-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    z-index: 9998;
  }

  .context-menu {
    position: fixed;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
    min-width: 180px;
    padding: 4px 0;
    z-index: 9999;
  }

  .menu-header {
    padding: 4px 10px;
    font-size: var(--text-xs);
    color: var(--text-secondary);
    border-bottom: 1px solid var(--border);
    margin-bottom: 2px;
  }

  .menu-item {
    width: 100%;
    height: 28px;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 10px;
    background: none;
    border: none;
    font-size: var(--text-sm);
    color: var(--text);
    cursor: pointer;
    text-align: left;
  }

  .menu-item:hover,
  .menu-item.item-active:hover {
    background-color: var(--accent) !important;
    color: #ffffff !important;
  }

  .menu-item:hover span,
  .menu-item.item-active:hover span {
    color: #ffffff !important;
  }

  .menu-item:hover :global(svg),
  .menu-item.item-active:hover :global(svg) {
    color: #ffffff !important;
    stroke: #ffffff !important;
  }

  .menu-item.item-active {
    color: var(--accent);
    font-weight: 500;
  }

  .menu-item:hover .menu-item-badge,
  .menu-item.item-active:hover .menu-item-badge {
    background-color: rgba(255, 255, 255, 0.28) !important;
    color: #ffffff !important;
    border: 1px solid rgba(255, 255, 255, 0.45);
  }

  .menu-item-badge {
    margin-left: auto;
    font-size: 10px;
    font-family: var(--font-mono);
    padding: 1px 5px;
    border-radius: 3px;
    background-color: rgba(0, 122, 255, 0.12);
    color: var(--accent);
    font-weight: 600;
  }

  .menu-divider {
    height: 1px;
    background-color: var(--border);
    margin: 3px 0;
  }
</style>
