<script lang="ts">
  import { resultsStore } from '../stores/results.svelte';
  import { filtersStore } from '../stores/filters.svelte';
  import type { ResultDTO, SortField } from '../types';
  import Icon from './Icon.svelte';

  const ROW_HEIGHT = 32;
  const OVERSCAN = 15;

  let containerRef = $state<HTMLDivElement | null>(null);
  let scrollTop = $state<number>(0);
  let containerHeight = $state<number>(600);
  let contextMenu = $state<{ x: number; y: number; row: ResultDTO } | null>(null);

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

  function handleContextMenu(e: MouseEvent, row: ResultDTO) {
    e.preventDefault();
    contextMenu = { x: e.clientX, y: e.clientY, row };
  }

  function closeContextMenu() {
    contextMenu = null;
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
  <div class="grid-header" role="row">
    <button class="col col-id header-btn" onclick={() => resultsStore.setSort('id')} aria-label="Sort by ID">
      <span>#</span>
      {#if resultsStore.sortField === 'id'}
        <Icon name={resultsStore.sortDirection === 'asc' ? 'arrow-up' : 'arrow-down'} size={12} />
      {/if}
    </button>
    <button class="col col-status header-btn" onclick={() => resultsStore.setSort('status')} aria-label="Sort by Status">
      <span>Status</span>
      {#if resultsStore.sortField === 'status'}
        <Icon name={resultsStore.sortDirection === 'asc' ? 'arrow-up' : 'arrow-down'} size={12} />
      {/if}
    </button>
    <button class="col col-payload header-btn">
      <span>Payload (FUZZ)</span>
    </button>
    <button class="col col-url header-btn" onclick={() => resultsStore.setSort('url')} aria-label="Sort by Target URL">
      <span>Target URL</span>
      {#if resultsStore.sortField === 'url'}
        <Icon name={resultsStore.sortDirection === 'asc' ? 'arrow-up' : 'arrow-down'} size={12} />
      {/if}
    </button>
    <button class="col col-length header-btn" onclick={() => resultsStore.setSort('length')} aria-label="Sort by Size">
      <span>Size</span>
      {#if resultsStore.sortField === 'length'}
        <Icon name={resultsStore.sortDirection === 'asc' ? 'arrow-up' : 'arrow-down'} size={12} />
      {/if}
    </button>
    <button class="col col-words header-btn" onclick={() => resultsStore.setSort('words')} aria-label="Sort by Words">
      <span>Words</span>
      {#if resultsStore.sortField === 'words'}
        <Icon name={resultsStore.sortDirection === 'asc' ? 'arrow-up' : 'arrow-down'} size={12} />
      {/if}
    </button>
    <button class="col col-lines header-btn" onclick={() => resultsStore.setSort('lines')} aria-label="Sort by Lines">
      <span>Lines</span>
      {#if resultsStore.sortField === 'lines'}
        <Icon name={resultsStore.sortDirection === 'asc' ? 'arrow-up' : 'arrow-down'} size={12} />
      {/if}
    </button>
    <button class="col col-duration header-btn" onclick={() => resultsStore.setSort('duration')} aria-label="Sort by Duration">
      <span>Duration</span>
      {#if resultsStore.sortField === 'duration'}
        <Icon name={resultsStore.sortDirection === 'asc' ? 'arrow-up' : 'arrow-down'} size={12} />
      {/if}
    </button>
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
            <div class="col col-id font-mono">{row.id}</div>
            <div class="col col-status">
              <span class="status-badge font-mono {getStatusBadgeClass(row.status)}">
                <Icon name={getStatusIconName(row.status)} size={11} />
                <span>{row.status}</span>
              </span>
            </div>
            <div class="col col-payload font-mono payload-text" title={getFirstInputValue(row.inputs)}>
              {getFirstInputValue(row.inputs)}
            </div>
            <div class="col col-url font-mono url-text" title={row.url}>
              {row.url}
            </div>
            <div class="col col-length font-mono">{row.length.toLocaleString()}</div>
            <div class="col col-words font-mono">{row.words.toLocaleString()}</div>
            <div class="col col-lines font-mono">{row.lines.toLocaleString()}</div>
            <div class="col col-duration font-mono">{row.duration_ms}ms</div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  {#if contextMenu}
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
          onclick={() => {
            filtersStore.addQuickFilter('status', '=', String(contextMenu?.row.status));
            closeContextMenu();
          }}
        >
          <Icon name="filter" size={13} />
          <span>Only Status {contextMenu.row.status}</span>
        </button>
        <button
          class="menu-item"
          onclick={() => {
            filtersStore.addQuickFilter('status', '!=', String(contextMenu?.row.status));
            closeContextMenu();
          }}
        >
          <Icon name="x" size={13} />
          <span>Hide Status {contextMenu.row.status}</span>
        </button>
        <div class="menu-divider"></div>
        <button
          class="menu-item"
          onclick={() => {
            filtersStore.addQuickFilter('length', '=', String(contextMenu?.row.length));
            closeContextMenu();
          }}
        >
          <Icon name="filter" size={13} />
          <span>Only Size {contextMenu.row.length}</span>
        </button>
        <button
          class="menu-item"
          onclick={() => {
            filtersStore.addQuickFilter('length', '!=', String(contextMenu?.row.length));
            closeContextMenu();
          }}
        >
          <Icon name="x" size={13} />
          <span>Hide Size {contextMenu.row.length}</span>
        </button>
        {#if contextMenu.row.host}
          <div class="menu-divider"></div>
          <button
            class="menu-item"
            onclick={() => {
              filtersStore.addQuickFilter('host', '=', contextMenu?.row.host || '');
              closeContextMenu();
            }}
          >
            <Icon name="globe" size={13} />
            <span>Only Host {contextMenu.row.host}</span>
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
    height: 30px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    font-size: 11px;
    font-weight: 600;
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
    font-size: 12px;
    cursor: pointer;
    transition: background 0.08s ease;
  }

  .grid-row:hover {
    background-color: var(--surface-hover);
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
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
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
    font-size: 10px;
    color: var(--text-secondary);
    border-bottom: 1px solid var(--border);
    margin-bottom: 2px;
  }

  .menu-item {
    width: 100%;
    height: 26px;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 10px;
    background: none;
    border: none;
    font-size: 11px;
    color: var(--text);
    cursor: pointer;
    text-align: left;
  }

  .menu-item:hover {
    background-color: var(--accent);
    color: var(--accent-contrast);
  }

  .menu-divider {
    height: 1px;
    background-color: var(--border);
    margin: 3px 0;
  }
</style>
