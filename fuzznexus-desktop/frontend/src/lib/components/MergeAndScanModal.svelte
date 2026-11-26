<script lang="ts">
  import Icon from './Icon.svelte';
  import { wordlistsStore } from '../stores/wordlists.svelte';
  import { toastStore } from '../stores/toast.svelte';
  import type { WordlistDTO, SanitizeOptions, MergeReportDTO } from '../types';

  interface Props {
    isOpen: boolean;
    selectedWordlists: WordlistDTO[];
    onClose: () => void;
    onConfirm: (target: WordlistDTO, autoStart: boolean) => void;
  }

  let { isOpen, selectedWordlists, onClose, onConfirm }: Props = $props();

  let sanitizeOptions = $state<SanitizeOptions>({
    trim_whitespace: true,
    strip_bom: true,
    remove_empty: true,
    strip_comments: true,
    strip_leading_slash: false,
    strip_traversal_payloads: false,
    case_insensitive: false,
    max_line_len: 2048
  });

  let targetMode = $state<'sandbox' | 'persistent'>('sandbox');
  let persistentName = $state<string>('');
  let persistentCategory = $state<string>('general');

  let isMerging = $state<boolean>(false);
  let isPreviewLoading = $state<boolean>(false);
  let previewReport = $state<MergeReportDTO | null>(null);

  let selectedIDs = $derived(selectedWordlists.map((w) => w.id));
  let totalSourceLinesRaw = $derived(
    selectedWordlists.reduce((acc, curr) => acc + curr.line_count, 0)
  );

  // Update default name when wordlists change
  $effect(() => {
    if (isOpen && selectedWordlists.length > 0) {
      persistentName = `merged_${selectedWordlists.length}_lists.txt`;
      persistentCategory = 'general';
      fetchPreview();
    }
  });

  async function fetchPreview() {
    if (selectedIDs.length === 0) return;
    isPreviewLoading = true;
    try {
      previewReport = await wordlistsStore.previewMerge(selectedIDs, sanitizeOptions);
    } catch (err: any) {
      console.warn('Failed to load merge preview:', err);
    } finally {
      isPreviewLoading = false;
    }
  }

  function handleOptionChange() {
    fetchPreview();
  }

  async function handleExecute(autoStart: boolean) {
    if (selectedIDs.length === 0) return;
    isMerging = true;

    try {
      const isEphemeral = targetMode === 'sandbox';
      const targetName = isEphemeral
        ? `sandbox_merged_${selectedWordlists.length}_lists.txt`
        : (persistentName.trim() || 'merged_wordlist.txt');

      const targetWordlist = await wordlistsStore.mergeToSandbox(
        selectedIDs,
        sanitizeOptions,
        isEphemeral,
        targetName,
        persistentCategory
      );

      toastStore.success(
        `Merged ${selectedWordlists.length} lists into ${isEphemeral ? 'Sandbox target' : `"${targetWordlist.name}"`} (${targetWordlist.line_count.toLocaleString()} clean lines)`
      );

      onConfirm(targetWordlist, autoStart);
      onClose();
    } catch (err: any) {
      toastStore.error(`Merge failed: ${err?.message || err}`);
    } finally {
      isMerging = false;
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !isMerging) {
      onClose();
    } else if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && !isMerging) {
      handleExecute(false);
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen}
  <div
    class="modal-backdrop"
    onclick={(e) => {
      if (e.target === e.currentTarget && !isMerging) onClose();
    }}
    role="presentation"
  >
    <div class="modal-dialog" role="dialog" aria-modal="true" aria-labelledby="merge-modal-title">
      <!-- Modal Header with Native Window Dragging -->
      <div class="modal-header" style="--wails-draggable: drag;">
        <div class="header-title-wrap" style="--wails-draggable: drag;">
          <Icon name="layers" size={18} class="header-icon" />
          <h2 id="merge-modal-title" class="modal-title">
            Merge & Use in Scan ({selectedWordlists.length} Wordlists)
          </h2>
        </div>
        <button
          type="button"
          class="btn-close-modal"
          onclick={onClose}
          disabled={isMerging}
          aria-label="Close"
          style="--wails-draggable: no-drag;"
        >
          <Icon name="close" size={16} />
        </button>
      </div>

      <!-- Modal Body -->
      <div class="modal-body">
        <!-- Section: Selected Wordlists Summary -->
        <div class="section-group">
          <div class="section-header">
            <span class="section-label">Selected Source Wordlists</span>
            <span class="section-badge font-mono">{totalSourceLinesRaw.toLocaleString()} raw lines</span>
          </div>
          <div class="sources-pill-box">
            {#each selectedWordlists as w (w.id)}
              <div class="source-item">
                <span class="source-name" title={w.name}>{w.name}</span>
                <span class="source-cat badge-{w.category || 'general'}">{w.category || 'general'}</span>
                <span class="source-lines font-mono">{w.line_count.toLocaleString()}L</span>
              </div>
            {/each}
          </div>
        </div>

        <!-- Section: Live Sanitization & Deduplication Stats -->
        <div class="section-group stats-preview-box">
          <div class="stats-header">
            <Icon name="sliders" size={14} />
            <span>High-Precision Deduplication Preview</span>
            {#if isPreviewLoading}
              <Icon name="refresh" size={12} class="spin ml-auto" />
            {/if}
          </div>
          <div class="stats-grid">
            <div class="stat-card">
              <span class="stat-val font-mono">{previewReport ? previewReport.total_source_lines.toLocaleString() : totalSourceLinesRaw.toLocaleString()}</span>
              <span class="stat-lbl">Raw Lines</span>
            </div>
            <div class="stat-card stat-accent">
              <span class="stat-val font-mono">{previewReport ? previewReport.deduplicated_lines.toLocaleString() : '-'}</span>
              <span class="stat-lbl">Clean Target Lines</span>
            </div>
            <div class="stat-card stat-danger">
              <span class="stat-val font-mono">{previewReport ? previewReport.duplicates_removed.toLocaleString() : '-'}</span>
              <span class="stat-lbl">Duplicates Stripped</span>
            </div>
            <div class="stat-card">
              <span class="stat-val font-mono">{previewReport ? (previewReport.empty_lines_removed + previewReport.comments_removed).toLocaleString() : '-'}</span>
              <span class="stat-lbl">Filtered / Empty</span>
            </div>
          </div>
        </div>

        <!-- Section: Sanitization Filters & Controls -->
        <div class="section-group">
          <div class="section-header">
            <span class="section-label">Sanitization & Cleaning Rules</span>
          </div>
          <div class="options-grid">
            <label class="option-label" title="Strip trailing/leading whitespace and Unicode zero-width/BOM chars">
              <input
                type="checkbox"
                class="custom-checkbox"
                bind:checked={sanitizeOptions.trim_whitespace}
                onchange={handleOptionChange}
                disabled={isMerging}
              />
              <span>Trim Whitespace & Invisible Runes</span>
            </label>

            <label class="option-label" title="Discard blank or whitespace-only lines">
              <input
                type="checkbox"
                class="custom-checkbox"
                bind:checked={sanitizeOptions.remove_empty}
                onchange={handleOptionChange}
                disabled={isMerging}
              />
              <span>Remove Blank Lines</span>
            </label>

            <label class="option-label" title="Filter lines beginning with '#' or '//'">
              <input
                type="checkbox"
                class="custom-checkbox"
                bind:checked={sanitizeOptions.strip_comments}
                onchange={handleOptionChange}
                disabled={isMerging}
              />
              <span>Strip Comment Lines (#, //)</span>
            </label>

            <label class="option-label" title="Remove leading '/' for seamless URL FUZZ path appending">
              <input
                type="checkbox"
                class="custom-checkbox"
                bind:checked={sanitizeOptions.strip_leading_slash}
                onchange={handleOptionChange}
                disabled={isMerging}
              />
              <span>Strip Leading Slashes (/)</span>
            </label>

            <label class="option-label" title="Deduplicate regardless of case (admin == Admin)">
              <input
                type="checkbox"
                class="custom-checkbox"
                bind:checked={sanitizeOptions.case_insensitive}
                onchange={handleOptionChange}
                disabled={isMerging}
              />
              <span>Case-Insensitive Deduplication</span>
            </label>

            <label class="option-label" title="Strip UTF-8 Byte Order Mark header">
              <input
                type="checkbox"
                class="custom-checkbox"
                bind:checked={sanitizeOptions.strip_bom}
                onchange={handleOptionChange}
                disabled={isMerging}
              />
              <span>Strip UTF-8 BOM Header</span>
            </label>
          </div>
        </div>

        <!-- Section: Destination Target Mode -->
        <div class="section-group">
          <div class="section-header">
            <span class="section-label">Target Storage Mode</span>
          </div>
          <div class="mode-selector">
            <button
              type="button"
              class="mode-btn"
              class:active={targetMode === 'sandbox'}
              onclick={() => (targetMode = 'sandbox')}
              disabled={isMerging}
            >
              <Icon name="target" size={14} />
              <div class="mode-text">
                <span class="mode-title">Isolated Sandbox (Recommended)</span>
                <span class="mode-sub">Created in private temp directory, locked during scan, cleaned up automatically.</span>
              </div>
            </button>

            <button
              type="button"
              class="mode-btn"
              class:active={targetMode === 'persistent'}
              onclick={() => (targetMode = 'persistent')}
              disabled={isMerging}
            >
              <Icon name="save" size={14} />
              <div class="mode-text">
                <span class="mode-title">Save to Wordlist Library</span>
                <span class="mode-sub">Registers a persistent copy in your Wordlist Manager and SQLite database.</span>
              </div>
            </button>
          </div>

          {#if targetMode === 'persistent'}
            <div class="persistent-config-box">
              <div class="config-row">
                <label for="merge-target-filename" class="config-label">File Name:</label>
                <input
                  id="merge-target-filename"
                  type="text"
                  bind:value={persistentName}
                  placeholder="merged_wordlist.txt"
                  class="config-input font-mono"
                  disabled={isMerging}
                />
              </div>
              <div class="config-row">
                <label for="merge-target-category" class="config-label">Category:</label>
                <select id="merge-target-category" bind:value={persistentCategory} class="config-input font-mono" disabled={isMerging}>
                  {#each wordlistsStore.categories as cat (cat.id)}
                    <option value={cat.id}>{cat.name}</option>
                  {/each}
                </select>
              </div>
            </div>
          {/if}
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="modal-footer">
        <span class="shortcut-tip">Cmd+Enter to set target</span>
        <button
          type="button"
          class="btn-modal btn-cancel"
          onclick={onClose}
          disabled={isMerging}
        >
          Cancel
        </button>
        <button
          type="button"
          class="btn-modal btn-target"
          onclick={() => handleExecute(false)}
          disabled={isMerging}
        >
          {#if isMerging}
            <Icon name="refresh" size={14} class="spin" />
            <span>Merging...</span>
          {:else}
            <Icon name="target" size={14} />
            <span>Merge & Set as Target</span>
          {/if}
        </button>
        <button
          type="button"
          class="btn-modal btn-primary"
          onclick={() => handleExecute(true)}
          disabled={isMerging}
        >
          {#if isMerging}
            <Icon name="refresh" size={14} class="spin" />
            <span>Processing...</span>
          {:else}
            <Icon name="play" size={14} />
            <span>Merge & Start Scan</span>
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(4px);
    z-index: 9999;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
    animation: fadeIn 0.15s ease-out;
  }

  .modal-dialog {
    background: var(--bg-surface, #1e1e24);
    border: 1px solid var(--border-color, #2f333d);
    border-radius: 10px;
    width: 100%;
    max-width: 680px;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 16px 36px rgba(0, 0, 0, 0.45);
    overflow: hidden;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 20px;
    border-bottom: 1px solid var(--border-color, #2f333d);
    background: var(--bg-surface-elevated, #242630);
  }

  .header-title-wrap {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .header-icon {
    color: var(--accent-color, #10b981);
  }

  .modal-title {
    font-size: 15px;
    font-weight: 600;
    margin: 0;
    color: var(--text-primary, #f8fafc);
    letter-spacing: -0.01em;
  }

  .btn-close-modal {
    background: transparent;
    border: none;
    color: var(--text-muted, #94a3b8);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    display: flex;
    align-items: center;
  }

  .btn-close-modal:hover:not(:disabled) {
    background: var(--bg-hover, #333846);
    color: var(--text-primary, #ffffff);
  }

  .modal-body {
    padding: 16px 20px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .section-label {
    font-size: 11.5px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted, #94a3b8);
    font-weight: 600;
  }

  .section-badge {
    font-size: 11.5px;
    font-weight: 600;
    color: var(--accent-color, #10b981);
  }

  .sources-pill-box {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    max-height: 85px;
    overflow-y: auto;
    padding: 6px;
    background: var(--bg-surface-elevated, #17181e);
    border: 1px solid var(--border-color, #2a2d36);
    border-radius: 6px;
  }

  .source-item {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 9px;
    background: var(--bg-surface, #21242d);
    border: 1px solid var(--border-color, #333742);
    border-radius: 5px;
    font-size: 12px;
    font-weight: 500;
  }

  .source-name {
    color: var(--text-primary, #f1f5f9);
    max-width: 150px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-weight: 500;
  }

  .source-cat {
    font-size: 9.5px;
    font-weight: 600;
    padding: 1px 5px;
    border-radius: 3px;
    background: var(--border-color, #3a3f4d);
    color: #e2e8f0;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .source-lines {
    color: var(--text-secondary, #cbd5e1);
    font-size: 11px;
    font-weight: 500;
  }

  .stats-preview-box {
    background: var(--bg-surface-elevated, #161820);
    border: 1px solid var(--border-color, #2f333d);
    border-radius: 8px;
    padding: 10px 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .stats-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12.5px;
    font-weight: 600;
    color: var(--text-secondary, #e2e8f0);
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 8px;
  }

  .stat-card {
    background: var(--bg-surface, #1e2029);
    border: 1px solid var(--border-color, #2e3340);
    border-radius: 6px;
    padding: 8px;
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
  }

  .stat-card.stat-accent {
    border-color: rgba(16, 185, 129, 0.4);
    background: rgba(16, 185, 129, 0.05);
  }

  .stat-card.stat-danger {
    border-color: rgba(244, 63, 94, 0.4);
    background: rgba(244, 63, 94, 0.05);
  }

  .stat-val {
    font-size: 16px;
    font-weight: 700;
    color: #ffffff;
    letter-spacing: -0.01em;
  }

  .stat-card.stat-accent .stat-val {
    color: #10b981;
  }

  .stat-card.stat-danger .stat-val {
    color: #f43f5e;
  }

  .stat-lbl {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #cbd5e1;
    margin-top: 3px;
  }

  .options-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }

  .option-label {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 9px 12px;
    background: var(--bg-surface-elevated, #1b1d24);
    border: 1px solid var(--border-color, #2b2e38);
    border-radius: 6px;
    font-size: 12.5px;
    font-weight: 500;
    color: #f1f5f9;
    cursor: pointer;
    user-select: none;
    transition: background 0.15s, border-color 0.15s;
  }

  .option-label:hover {
    background: var(--bg-hover, #262933);
    border-color: #3b4050;
  }

  .custom-checkbox {
    accent-color: var(--accent-color, #10b981);
    cursor: pointer;
  }

  .mode-selector {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }

  .mode-btn {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 12px;
    background: var(--bg-surface-elevated, #1a1c22);
    border: 1px solid var(--border-color, #2d303b);
    border-radius: 8px;
    text-align: left;
    cursor: pointer;
    color: var(--text-secondary, #cbd5e1);
    transition: all 0.15s;
  }

  .mode-btn:hover:not(:disabled) {
    background: var(--bg-hover, #252832);
    border-color: var(--border-hover, #424654);
  }

  .mode-btn.active {
    background: rgba(16, 185, 129, 0.08);
    border-color: var(--accent-color, #10b981);
    color: var(--text-primary, #ffffff);
  }

  .mode-text {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .mode-title {
    font-size: 13px;
    font-weight: 600;
    color: #f8fafc;
  }

  .mode-sub {
    font-size: 11.5px;
    font-weight: 500;
    color: #cbd5e1;
    line-height: 1.4;
  }

  .persistent-config-box {
    margin-top: 6px;
    padding: 10px 12px;
    background: var(--bg-surface-elevated, #16181f);
    border: 1px solid var(--border-color, #2b2e38);
    border-radius: 6px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .config-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .config-label {
    width: 80px;
    font-size: 12px;
    font-weight: 500;
    color: #cbd5e1;
  }

  .config-input {
    flex: 1;
    background: var(--bg-surface, #20222a);
    border: 1px solid var(--border-color, #333742);
    border-radius: 4px;
    padding: 6px 10px;
    color: var(--text-primary, #ffffff);
    font-size: 12px;
    font-weight: 500;
  }

  .config-input:focus {
    border-color: var(--accent-color, #10b981);
    outline: none;
  }

  .modal-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    padding: 12px 20px;
    border-top: 1px solid var(--border-color, #2f333d);
    background: var(--bg-surface-elevated, #242630);
  }

  .shortcut-tip {
    font-size: 11.5px;
    font-weight: 500;
    color: #cbd5e1;
    margin-right: auto;
  }

  .btn-modal {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 7px 12px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    border: none;
    transition: opacity 0.15s;
  }

  .btn-modal:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-cancel {
    background: var(--bg-hover, #333742);
    color: var(--text-secondary, #cbd5e1);
  }

  .btn-cancel:hover:not(:disabled) {
    background: #3e4452;
  }

  .btn-target {
    background: #2563eb;
    color: #ffffff;
  }

  .btn-target:hover:not(:disabled) {
    background: #1d4ed8;
  }

  .btn-primary {
    background: #10b981;
    color: #0f172a;
  }

  .btn-primary:hover:not(:disabled) {
    background: #059669;
  }

  :global(.spin) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }
</style>
