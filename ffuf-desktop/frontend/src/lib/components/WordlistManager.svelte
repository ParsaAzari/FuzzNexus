<script lang="ts">
  import { onMount } from 'svelte';
  import { wordlistsStore } from '../stores/wordlists.svelte';
  import { scanStore } from '../stores/scan.svelte';
  import Icon from './Icon.svelte';
  import type { WordlistDTO } from '../types';

  interface Props {
    onSelectForScan?: (path: string) => void;
  }

  let { onSelectForScan }: Props = $props();

  let newWordlistModalOpen = $state<boolean>(false);
  let newWordlistName = $state<string>('');
  let initialLines = $state<string>('');

  let prefixSuffixModalOpen = $state<boolean>(false);
  let prefixInput = $state<string>('');
  let suffixInput = $state<string>('');

  let editorLines = $state<string[]>([]);
  let isDirty = $state<boolean>(false);
  let pageOffset = $state<number>(0);
  const pageSize = 100;

  let selected = $derived(wordlistsStore.selectedWordlist);
  let stats = $derived(wordlistsStore.selectedStats);
  let totalLines = $derived(wordlistsStore.currentPage.total_lines || 0);

  onMount(() => {
    wordlistsStore.loadWordlists();
  });

  $effect(() => {
    if (wordlistsStore.currentPage.lines) {
      editorLines = [...wordlistsStore.currentPage.lines];
      isDirty = false;
    }
  });

  function handleLineChange(index: number, value: string) {
    editorLines[index] = value;
    isDirty = true;
  }

  async function handleSave() {
    if (!selected) return;
    try {
      await wordlistsStore.saveContent(selected.id, editorLines.join('\n'));
      isDirty = false;
    } catch (e: any) {
      alert(`Save failed: ${e?.message || e}`);
    }
  }

  async function handleCreateNew() {
    if (!newWordlistName.trim()) return;
    try {
      await wordlistsStore.createWordlist(newWordlistName.trim(), initialLines);
      newWordlistName = '';
      initialLines = '';
      newWordlistModalOpen = false;
    } catch (e: any) {
      alert(`Create failed: ${e?.message || e}`);
    }
  }

  // Wordlist transformation tools
  function applyDeduplicate() {
    const seen = new Set<string>();
    editorLines = editorLines.filter((l) => {
      if (seen.has(l)) return false;
      seen.add(l);
      return true;
    });
    isDirty = true;
  }

  function applySort(asc: boolean) {
    editorLines = [...editorLines].sort((a, b) => (asc ? a.localeCompare(b) : b.localeCompare(a)));
    isDirty = true;
  }

  function applyTrim() {
    editorLines = editorLines.map((l) => l.trim());
    isDirty = true;
  }

  function applyCase(lower: boolean) {
    editorLines = editorLines.map((l) => (lower ? l.toLowerCase() : l.toUpperCase()));
    isDirty = true;
  }

  function applyStripEmpty() {
    editorLines = editorLines.filter((l) => l.trim().length > 0);
    isDirty = true;
  }

  function applyPrefixSuffix() {
    const p = prefixInput;
    const s = suffixInput;
    editorLines = editorLines.map((l) => `${p}${l}${s}`);
    isDirty = true;
    prefixSuffixModalOpen = false;
    prefixInput = '';
    suffixInput = '';
  }

  function handlePageNav(newOffset: number) {
    if (isDirty) {
      if (!confirm('You have unsaved changes in the current page. Discard changes and navigate?')) {
        return;
      }
    }
    pageOffset = Math.max(0, Math.min(newOffset, Math.max(0, totalLines - pageSize)));
    wordlistsStore.loadPage(pageOffset, pageSize);
  }

  function handleSelectForScan(w: WordlistDTO) {
    scanStore.wordlist = w.path;
    onSelectForScan?.(w.path);
  }

  async function handleDelete(w: WordlistDTO) {
    if (!confirm(`Are you sure you want to delete wordlist "${w.name}"?`)) return;
    try {
      await wordlistsStore.deleteWordlist(w.id);
    } catch (e: any) {
      alert(`Delete rejected: ${e?.message || e}`);
    }
  }
</script>

<div class="wordlist-manager select-none">
  <!-- Left Column: Wordlist Library -->
  <aside class="sidebar-library">
    <div class="sidebar-header">
      <div class="search-wrap">
        <Icon name="search" size={13} class="search-icon" />
        <input
          type="text"
          placeholder="Filter wordlists..."
          bind:value={wordlistsStore.searchFilter}
          class="sidebar-search font-mono"
        />
        {#if wordlistsStore.searchFilter}
          <button class="btn-clear-search" onclick={() => (wordlistsStore.searchFilter = '')}>
            <Icon name="x" size={12} />
          </button>
        {/if}
      </div>

      <div class="sidebar-actions">
        <button
          class="btn-action-primary"
          onclick={() => (newWordlistModalOpen = true)}
          title="Create a new empty or custom wordlist"
        >
          <Icon name="plus" size={13} />
          <span>New</span>
        </button>
        <button
          class="btn-action-subtle"
          onclick={() => wordlistsStore.openNativeFilePicker('import')}
          title="Import an existing wordlist file into application storage"
        >
          <Icon name="upload" size={13} />
          <span>Import</span>
        </button>
        <button
          class="btn-action-subtle"
          onclick={() => wordlistsStore.openNativeFilePicker('link')}
          title="Link an external wordlist file in place without copying"
        >
          <Icon name="folder" size={13} />
          <span>Link</span>
        </button>
      </div>
    </div>

    <!-- Wordlist Records List -->
    <div class="wordlists-list">
      {#if wordlistsStore.isLoading && wordlistsStore.items.length === 0}
        <div class="loading-state">
          <Icon name="refresh" size={18} class="spin" />
          <span>Loading wordlists...</span>
        </div>
      {:else if wordlistsStore.filteredWordlists.length === 0}
        <div class="empty-list">
          <Icon name="file-text" size={32} />
          <div class="empty-title">No Wordlists Found</div>
          <div class="empty-sub">Import, link, or create a wordlist to get started.</div>
        </div>
      {:else}
        {#each wordlistsStore.filteredWordlists as w (w.id)}
          <div
            class="wordlist-item"
            class:active={wordlistsStore.selectedID === w.id}
            onclick={() => wordlistsStore.selectWordlist(w.id)}
            role="button"
            tabindex="0"
            onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && wordlistsStore.selectWordlist(w.id)}
          >
            <div class="item-main">
              <span class="item-name">{w.name}</span>
              <div class="item-tags font-mono">
                <span class="tag-lines">{w.line_count.toLocaleString()} lines</span>
                <span class="tag-size">{(w.size_bytes / 1024).toFixed(1)} KB</span>
                <span class="tag-badge" class:tag-linked={w.is_linked}>
                  {w.is_linked ? 'Linked' : 'Local'}
                </span>
              </div>
            </div>
          </div>
        {/each}
      {/if}
    </div>
  </aside>

  <!-- Right Column: Virtualized Editor & Operations -->
  <main class="editor-pane">
    {#if !selected}
      <div class="empty-editor">
        <Icon name="file-text" size={48} />
        <div class="empty-title">No Wordlist Selected</div>
        <div class="empty-sub">Select a wordlist from the library on the left to edit and inspect lines.</div>
      </div>
    {:else}
      <!-- Editor Header -->
      <header class="editor-header">
        <div class="header-left">
          <div class="title-row">
            <span class="editor-title">{selected.name}</span>
            <span class="type-pill font-mono" class:type-linked={selected.is_linked}>
              {selected.is_linked ? 'Linked File' : 'Managed Wordlist'}
            </span>
          </div>
          <div class="path-sub font-mono" title={selected.path}>{selected.path}</div>
        </div>

        <div class="header-right">
          <button
            class="btn-use-scan"
            onclick={() => handleSelectForScan(selected)}
            title="Set this wordlist as target for FUZZ in Control Bar"
          >
            <Icon name="play" size={13} />
            <span>Use in Scan</span>
          </button>
          <button
            class="btn-delete-wl"
            onclick={() => handleDelete(selected)}
            title="Delete or unlink this wordlist"
          >
            <Icon name="trash" size={14} />
          </button>
        </div>
      </header>

      <!-- KPI Statistics Strip -->
      <div class="stats-strip font-mono">
        <div class="stat-box">
          <span class="stat-label">Total Lines</span>
          <span class="stat-value">{selected.line_count.toLocaleString()}</span>
        </div>
        <div class="stat-box">
          <span class="stat-label">File Size</span>
          <span class="stat-value">{(selected.size_bytes / 1024).toFixed(1)} KB</span>
        </div>
        {#if stats}
          <div class="stat-box">
            <span class="stat-label">Unique Items</span>
            <span class="stat-value">{stats.unique_lines.toLocaleString()}</span>
          </div>
          <div class="stat-box">
            <span class="stat-label">Avg Length</span>
            <span class="stat-value">{stats.avg_line_len.toFixed(1)} chars</span>
          </div>
          <div class="stat-box">
            <span class="stat-label">Max Length</span>
            <span class="stat-value">{stats.max_line_len} chars</span>
          </div>
        {/if}
      </div>

      <!-- Editor Toolbar (Transformations) -->
      <div class="editor-toolbar">
        <div class="tools-group">
          <button class="tool-btn" onclick={applyDeduplicate} title="Remove duplicate lines">
            <Icon name="sliders" size={12} />
            <span>Dedupe</span>
          </button>
          <button class="tool-btn" onclick={() => applySort(true)} title="Sort lines alphabetically ascending">
            <Icon name="arrow-up" size={12} />
            <span>Sort A-Z</span>
          </button>
          <button class="tool-btn" onclick={() => applySort(false)} title="Sort lines alphabetically descending">
            <Icon name="arrow-down" size={12} />
            <span>Sort Z-A</span>
          </button>
          <button class="tool-btn" onclick={applyTrim} title="Trim whitespace on all lines">
            <span>Trim</span>
          </button>
          <button class="tool-btn" onclick={() => applyCase(true)} title="Convert all lines to lowercase">
            <span>lower</span>
          </button>
          <button class="tool-btn" onclick={() => applyCase(false)} title="Convert all lines to uppercase">
            <span>UPPER</span>
          </button>
          <button class="tool-btn" onclick={applyStripEmpty} title="Remove empty lines">
            <span>Strip Empty</span>
          </button>
          <button class="tool-btn" onclick={() => (prefixSuffixModalOpen = true)} title="Add prefix and/or suffix to all lines">
            <span>+ Prefix/Suffix</span>
          </button>
        </div>

        <!-- Pagination Controls -->
        <div class="pagination-group font-mono">
          <button
            class="btn-page"
            disabled={pageOffset === 0}
            onclick={() => handlePageNav(pageOffset - pageSize)}
            title="Previous page"
          >
            Prev
          </button>
          <span class="page-indicator">
            {pageOffset + 1} - {Math.min(pageOffset + pageSize, totalLines)} of {totalLines.toLocaleString()}
          </span>
          <button
            class="btn-page"
            disabled={pageOffset + pageSize >= totalLines}
            onclick={() => handlePageNav(pageOffset + pageSize)}
            title="Next page"
          >
            Next
          </button>
        </div>
      </div>

      <!-- Virtualized Line-by-Line Editor -->
      <div class="editor-container">
        <div class="lines-gutter font-mono select-none">
          {#each editorLines as _, idx}
            <div class="gutter-number">{pageOffset + idx + 1}</div>
          {/each}
        </div>
        <div class="lines-content font-mono select-text">
          {#each editorLines as line, idx}
            <input
              type="text"
              class="line-input"
              value={line}
              oninput={(e) => handleLineChange(idx, (e.currentTarget as HTMLInputElement).value)}
            />
          {/each}
        </div>
      </div>

      <!-- Editor Footer Action Bar -->
      <footer class="editor-footer">
        <div class="footer-left">
          {#if isDirty}
            <span class="dirty-badge">Unsaved Changes</span>
          {/if}
        </div>
        <div class="footer-right">
          <button
            class="btn-footer-save"
            onclick={handleSave}
            disabled={!isDirty || wordlistsStore.isSaving}
          >
            <Icon name="save" size={14} />
            <span>{wordlistsStore.isSaving ? 'Saving...' : 'Save Wordlist'}</span>
          </button>
        </div>
      </footer>
    {/if}
  </main>
</div>

<!-- Modal: Create New Wordlist -->
{#if newWordlistModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) newWordlistModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (newWordlistModalOpen = false)}
  >
    <div class="modal-dialog" role="document">
      <div class="modal-head">
        <span>Create New Wordlist</span>
        <button class="btn-close-modal" onclick={() => (newWordlistModalOpen = false)}>
          <Icon name="x" size={16} />
        </button>
      </div>
      <div class="modal-body-content">
        <label class="modal-label">
          Wordlist Name:
          <input
            type="text"
            placeholder="e.g. custom_routes.txt"
            bind:value={newWordlistName}
            class="modal-input font-mono"
          />
        </label>
        <label class="modal-label">
          Initial Lines (one per line):
          <textarea
            rows="8"
            placeholder="admin&#10;api&#10;login&#10;v1&#10;v2"
            bind:value={initialLines}
            class="modal-textarea font-mono"
          ></textarea>
        </label>
      </div>
      <div class="modal-foot">
        <button class="btn-cancel" onclick={() => (newWordlistModalOpen = false)}>Cancel</button>
        <button class="btn-confirm" onclick={handleCreateNew} disabled={!newWordlistName.trim()}>Create</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Prefix / Suffix -->
{#if prefixSuffixModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) prefixSuffixModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (prefixSuffixModalOpen = false)}
  >
    <div class="modal-dialog" role="document">
      <div class="modal-head">
        <span>Add Prefix & Suffix to Lines</span>
        <button class="btn-close-modal" onclick={() => (prefixSuffixModalOpen = false)}>
          <Icon name="x" size={16} />
        </button>
      </div>
      <div class="modal-body-content">
        <label class="modal-label">
          Prefix (prepend to all lines):
          <input type="text" placeholder="e.g. /api/v1/" bind:value={prefixInput} class="modal-input font-mono" />
        </label>
        <label class="modal-label">
          Suffix (append to all lines):
          <input type="text" placeholder="e.g. .php or .json" bind:value={suffixInput} class="modal-input font-mono" />
        </label>
      </div>
      <div class="modal-foot">
        <button class="btn-cancel" onclick={() => (prefixSuffixModalOpen = false)}>Cancel</button>
        <button class="btn-confirm" onclick={applyPrefixSuffix}>Apply to Lines</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .wordlist-manager {
    display: flex;
    flex: 1;
    height: 100%;
    min-height: 0;
    background-color: var(--surface);
  }

  .sidebar-library {
    width: 280px;
    background-color: var(--bg-subtle);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
  }

  .sidebar-header {
    padding: 10px;
    border-bottom: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .search-wrap {
    position: relative;
    display: flex;
    align-items: center;
  }

  :global(.search-icon) {
    position: absolute;
    left: 8px;
    color: var(--text-secondary);
  }

  .sidebar-search {
    width: 100%;
    height: 26px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 24px 0 26px;
    font-size: 11px;
    color: var(--text);
    box-sizing: border-box;
  }

  .btn-clear-search {
    position: absolute;
    right: 4px;
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
  }

  .sidebar-actions {
    display: flex;
    gap: 4px;
  }

  .btn-action-primary,
  .btn-action-subtle {
    height: 24px;
    flex: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    border-radius: 4px;
    font-size: 11px;
    cursor: pointer;
  }

  .btn-action-primary {
    background-color: var(--accent);
    color: var(--accent-contrast);
    border: none;
    font-weight: 500;
  }

  .btn-action-subtle {
    background-color: var(--surface);
    color: var(--text);
    border: 1px solid var(--border);
  }

  .wordlists-list {
    flex: 1;
    overflow-y: auto;
    padding: 6px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .wordlist-item {
    padding: 8px 10px;
    background-color: var(--surface);
    border: 1px solid transparent;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.08s ease;
  }

  .wordlist-item:hover {
    background-color: var(--surface-hover);
  }

  .wordlist-item.active {
    background-color: rgba(0, 122, 255, 0.1);
    border-color: var(--accent);
  }

  .item-name {
    font-size: 12px;
    font-weight: 600;
    color: var(--text);
    display: block;
    margin-bottom: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .item-tags {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 10px;
    color: var(--text-secondary);
  }

  .tag-badge {
    background-color: rgba(128, 128, 128, 0.15);
    padding: 1px 4px;
    border-radius: 3px;
  }

  .tag-linked {
    background-color: rgba(0, 122, 255, 0.15);
    color: var(--accent);
  }

  .editor-pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    background-color: var(--surface);
  }

  .empty-editor,
  .empty-list {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
    gap: 8px;
    padding: 24px;
    text-align: center;
  }

  .editor-header {
    height: 52px;
    padding: 0 16px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    background-color: var(--bg-subtle);
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .editor-title {
    font-size: 14px;
    font-weight: 700;
    color: var(--text);
  }

  .type-pill {
    font-size: 10px;
    padding: 1px 6px;
    border-radius: 10px;
    background-color: rgba(52, 199, 89, 0.15);
    color: #248a3d;
  }

  .type-linked {
    background-color: rgba(0, 122, 255, 0.15);
    color: var(--accent);
  }

  .path-sub {
    font-size: 11px;
    color: var(--text-secondary);
    margin-top: 2px;
    max-width: 480px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .btn-use-scan {
    height: 28px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background-color: var(--accent);
    color: var(--accent-contrast);
    border: none;
    border-radius: 4px;
    padding: 0 10px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
  }

  .btn-delete-wl {
    height: 28px;
    width: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: 1px solid var(--border);
    border-radius: 4px;
    color: var(--text-secondary);
    cursor: pointer;
  }

  .btn-delete-wl:hover {
    color: #ff3b30;
    border-color: #ff3b30;
  }

  .stats-strip {
    display: flex;
    gap: 16px;
    padding: 8px 16px;
    background-color: var(--surface);
    border-bottom: 1px solid var(--border);
    font-size: 11px;
  }

  .stat-box {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .stat-label {
    font-size: 9.5px;
    color: var(--text-secondary);
    text-transform: uppercase;
  }

  .stat-value {
    font-weight: 600;
    color: var(--text);
  }

  .editor-toolbar {
    height: 32px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px;
    gap: 8px;
    overflow-x: auto;
  }

  .tools-group {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .tool-btn {
    height: 22px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 3px;
    padding: 0 6px;
    font-size: 10.5px;
    color: var(--text);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .tool-btn:hover {
    background-color: var(--surface-hover);
  }

  .pagination-group {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
  }

  .btn-page {
    height: 22px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 3px;
    padding: 0 6px;
    font-size: 10px;
    color: var(--text);
    cursor: pointer;
  }

  .btn-page:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .page-indicator {
    color: var(--text-secondary);
    font-size: 10.5px;
  }

  .editor-container {
    flex: 1;
    overflow-y: auto;
    display: flex;
    background-color: var(--surface);
  }

  .lines-gutter {
    width: 48px;
    background-color: var(--bg-subtle);
    border-right: 1px solid var(--border);
    padding: 6px 0;
    text-align: right;
    color: var(--text-secondary);
    font-size: 11px;
    line-height: 24px;
  }

  .gutter-number {
    padding-right: 8px;
    height: 24px;
  }

  .lines-content {
    flex: 1;
    padding: 6px 0;
  }

  .line-input {
    width: 100%;
    height: 24px;
    padding: 0 8px;
    background: none;
    border: none;
    outline: none;
    font-size: 12px;
    color: var(--text);
    box-sizing: border-box;
  }

  .line-input:focus {
    background-color: rgba(0, 122, 255, 0.08);
  }

  .editor-footer {
    height: 40px;
    background-color: var(--bg-subtle);
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
  }

  .dirty-badge {
    font-size: 11px;
    color: #ff9500;
    font-weight: 600;
  }

  .btn-footer-save {
    height: 26px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background-color: var(--accent);
    color: var(--accent-contrast);
    border: none;
    border-radius: 4px;
    padding: 0 12px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
  }

  .btn-footer-save:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Modals */
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background-color: rgba(0, 0, 0, 0.45);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
    backdrop-filter: blur(2px);
  }

  .modal-dialog {
    width: 480px;
    max-width: 90vw;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    box-shadow: 0 8px 30px rgba(0, 0, 0, 0.25);
    overflow: hidden;
  }

  .modal-head {
    height: 38px;
    padding: 0 14px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 12px;
    font-weight: 600;
  }

  .btn-close-modal {
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
  }

  .modal-body-content {
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .modal-label {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 11px;
    color: var(--text-secondary);
  }

  .modal-input {
    height: 28px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 8px;
    font-size: 12px;
    color: var(--text);
  }

  .modal-textarea {
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 8px;
    font-size: 12px;
    color: var(--text);
    resize: vertical;
  }

  .modal-foot {
    height: 44px;
    padding: 0 14px;
    background-color: var(--bg-subtle);
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
  }

  .btn-cancel {
    height: 28px;
    padding: 0 12px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    font-size: 11px;
    cursor: pointer;
    color: var(--text);
  }

  .btn-confirm {
    height: 28px;
    padding: 0 14px;
    background-color: var(--accent);
    color: var(--accent-contrast);
    border: none;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
  }

  :global(.spin) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
