<script lang="ts">
  import { GetSessions, DeleteSessions, ClearSessions } from '../../../wailsjs/go/main/App';
  import { scanStore } from '../stores/scan.svelte';
  import { toastStore } from '../stores/toast.svelte';
  import { resultsStore } from '../stores/results.svelte';
  import type { SessionDTO } from '../types';
  import Icon from './Icon.svelte';

  let sessions = $state<SessionDTO[]>([]);
  let isLoading = $state<boolean>(false);
  let isDeleting = $state<boolean>(false);
  let loadError = $state<string>('');
  let selectedIDs = $state<number[]>([]);
  let confirmClearAll = $state<boolean>(false);

  let allSelected = $derived(sessions.length > 0 && selectedIDs.length === sessions.length);

  $effect(() => {
    if (scanStore.isHistoryModalOpen) {
      confirmClearAll = false;
      selectedIDs = [];
      fetchSessions();
    }
  });

  async function fetchSessions() {
    isLoading = true;
    loadError = '';
    try {
      const list = await GetSessions(100);
      sessions = Array.isArray(list) ? list : [];
    } catch (err: unknown) {
      loadError = err instanceof Error ? err.message : String(err);
    } finally {
      isLoading = false;
    }
  }

  function toggleSelect(id: number) {
    if (selectedIDs.includes(id)) {
      selectedIDs = selectedIDs.filter((x) => x !== id);
    } else {
      selectedIDs = [...selectedIDs, id];
    }
  }

  function toggleSelectAll() {
    if (allSelected) {
      selectedIDs = [];
    } else {
      selectedIDs = sessions.map((s) => s.id);
    }
  }

  async function deleteSelected() {
    if (selectedIDs.length === 0 || isDeleting) return;
    isDeleting = true;
    loadError = '';
    try {
      const deletedSet = new Set(selectedIDs);
      await DeleteSessions(selectedIDs);
      if (deletedSet.has(scanStore.sessionID) && scanStore.state !== 'Running') {
        scanStore.reset();
        resultsStore.items = [];
      }
      selectedIDs = [];
      await fetchSessions();
    } catch (err: unknown) {
      loadError = err instanceof Error ? err.message : String(err);
    } finally {
      isDeleting = false;
    }
  }

  async function clearAllHistory() {
    if (isDeleting) return;
    isDeleting = true;
    loadError = '';
    try {
      await ClearSessions();
      if (scanStore.state !== 'Running') {
        scanStore.reset();
        resultsStore.items = [];
      }
      selectedIDs = [];
      confirmClearAll = false;
      await fetchSessions();
    } catch (err: unknown) {
      loadError = err instanceof Error ? err.message : String(err);
    } finally {
      isDeleting = false;
    }
  }

  function parseTarget(configRedacted: string): { url: string; method: string } {
    try {
      const parsed = JSON.parse(configRedacted) as { url?: string; method?: string };
      return {
        url: parsed.url || 'Unknown target',
        method: parsed.method || 'GET'
      };
    } catch {
      return { url: 'Unknown target', method: 'GET' };
    }
  }

  function close() {
    confirmClearAll = false;
    scanStore.isHistoryModalOpen = false;
  }
</script>

{#if scanStore.isHistoryModalOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="history-backdrop"
    onclick={(e) => e.target === e.currentTarget && close()}
    role="dialog"
    aria-modal="true"
    aria-label="Scan Session History"
    tabindex="-1"
  >
    <div class="history-modal">
      <div class="history-header">
        <div class="history-title-group">
          <Icon name="rotate-ccw" size={16} />
          <h2>Scan Session History</h2>
        </div>
        <div class="history-header-actions">
          <button type="button" onclick={fetchSessions} class="btn-secondary">
            Refresh
          </button>
          <button type="button" onclick={close} class="btn-icon-close" aria-label="Close history">
            <Icon name="x" size={16} />
          </button>
        </div>
      </div>

      {#if sessions.length > 0 && !isLoading}
        <div class="history-bulk-bar">
          <label class="select-all-label">
            <input
              type="checkbox"
              checked={allSelected}
              onchange={toggleSelectAll}
              class="native-checkbox"
            />
            <span>Select All ({sessions.length})</span>
          </label>
          <div class="bulk-actions">
            {#if selectedIDs.length > 0}
              <button
                type="button"
                disabled={isDeleting}
                onclick={deleteSelected}
                class="btn-danger-outline"
              >
                Delete Selected ({selectedIDs.length})
              </button>
            {/if}
            {#if !confirmClearAll}
              <button
                type="button"
                disabled={isDeleting}
                onclick={() => (confirmClearAll = true)}
                class="btn-danger-outline"
              >
                Clear All
              </button>
            {:else}
              <button
                type="button"
                disabled={isDeleting}
                onclick={clearAllHistory}
                class="btn-danger-solid"
              >
                Confirm Clear All
              </button>
              <button
                type="button"
                disabled={isDeleting}
                onclick={() => (confirmClearAll = false)}
                class="btn-secondary"
              >
                Cancel
              </button>
            {/if}
          </div>
        </div>
      {/if}

      <div class="history-body">
        {#if isLoading}
          <div class="history-empty">Loading saved sessions...</div>
        {:else if loadError}
          <div class="history-error">{loadError}</div>
        {:else if sessions.length === 0}
          <div class="history-empty">No historical scan sessions recorded yet.</div>
        {:else}
          <div class="sessions-list">
            {#each sessions as sess (sess.id)}
              {@const meta = parseTarget(sess.config_redacted_json)}
              <div class="session-row">
                <input
                  type="checkbox"
                  checked={selectedIDs.includes(sess.id)}
                  onchange={() => toggleSelect(sess.id)}
                  aria-label="Select session #{sess.id}"
                  class="native-checkbox"
                />
                <div class="session-meta">
                  <div class="session-top-line">
                    <span class="id-pill font-mono">#{sess.id}</span>
                    <span class="method-pill font-mono">{meta.method}</span>
                    <span class="url-text font-mono">{meta.url}</span>
                  </div>
                  <div class="session-sub-line">
                    <span>State: <strong>{sess.state}</strong></span>
                    <span>Requests: <strong class="req-count">{sess.total_requests}</strong></span>
                    <span>Started: {sess.started_at}</span>
                  </div>
                </div>
                <div class="session-actions">
                  <button
                    type="button"
                    onclick={() => {
                      scanStore.cloneSessionConfig(sess);
                      toastStore.success(`Cloned config from Session #${sess.id}`);
                      close();
                    }}
                    title="Copy this session's target URL, method, and headers into ControlBar without clearing current table"
                    class="btn-secondary"
                  >
                    Clone Config
                  </button>
                  <button
                    type="button"
                    disabled={scanStore.state === 'Running' || scanStore.state === 'Paused'}
                    onclick={() => {
                      scanStore.loadHistoricalSession(sess);
                      toastStore.info(`Loaded results from Session #${sess.id}`);
                    }}
                    title="Load this session's saved results from SQLite into Table and Analytics"
                    class="btn-primary"
                  >
                    Load Results
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .history-backdrop {
    position: fixed;
    inset: 0;
    z-index: 10020;
    background: rgba(0, 0, 0, 0.58);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
  }

  .history-modal {
    width: 100%;
    max-width: 780px;
    max-height: 82vh;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    color: var(--text);
    display: flex;
    flex-direction: column;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.45);
    overflow: hidden;
  }

  .history-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 18px;
    border-bottom: 1px solid var(--border);
  }

  .history-title-group {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--accent);
  }

  .history-title-group h2 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--text);
  }

  .history-header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .history-bulk-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 8px 18px;
    border-bottom: 1px solid var(--border);
    background-color: var(--bg-subtle);
  }

  .select-all-label {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-size: 11px;
    font-weight: 500;
    color: var(--text-muted);
    cursor: pointer;
  }

  .bulk-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .native-checkbox {
    width: 14px;
    height: 14px;
    accent-color: var(--accent);
    cursor: pointer;
    flex-shrink: 0;
  }

  .btn-icon-close {
    background: transparent;
    border: none;
    color: var(--text-muted);
    padding: 4px;
    border-radius: var(--radius-sm);
    cursor: pointer;
    display: inline-flex;
  }

  .btn-icon-close:hover {
    background-color: var(--surface-hover);
    color: var(--text);
  }

  .history-body {
    flex: 1;
    overflow-y: auto;
    padding: 14px 18px;
  }

  .history-empty {
    padding: 36px 0;
    text-align: center;
    font-size: 12px;
    color: var(--text-muted);
  }

  .history-error {
    padding: 12px;
    border-radius: var(--radius-sm);
    border: 1px solid rgba(244, 63, 94, 0.35);
    background-color: rgba(244, 63, 94, 0.1);
    color: #fda4af;
    font-size: 12px;
  }

  .sessions-list {
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background-color: var(--bg-subtle);
    overflow: hidden;
  }

  .session-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 14px;
    border-bottom: 1px solid var(--border);
  }

  .session-row:last-child {
    border-bottom: none;
  }

  .session-row:hover {
    background-color: var(--surface-hover);
  }

  .session-meta {
    min-width: 0;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .session-top-line {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .id-pill {
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 1px 6px;
    font-size: 10px;
    color: var(--text-muted);
  }

  .method-pill {
    background-color: rgba(56, 189, 248, 0.15);
    border-radius: 4px;
    padding: 1px 6px;
    font-size: 10px;
    font-weight: 600;
    color: #38bdf8;
  }

  .url-text {
    font-size: 12px;
    color: var(--text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .session-sub-line {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    font-size: 11px;
    color: var(--text-muted);
  }

  .req-count {
    color: #34d399;
  }

  .session-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .btn-secondary {
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text);
    font-size: 11px;
    font-weight: 500;
    padding: 5px 10px;
    cursor: pointer;
  }

  .btn-secondary:hover {
    background-color: var(--surface-hover);
  }

  .btn-primary {
    background-color: var(--accent);
    border: none;
    border-radius: var(--radius-sm);
    color: var(--accent-contrast, #ffffff);
    font-size: 11px;
    font-weight: 500;
    padding: 5px 10px;
    cursor: pointer;
  }

  .btn-primary:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .btn-danger-outline {
    background-color: transparent;
    border: 1px solid rgba(244, 63, 94, 0.45);
    border-radius: var(--radius-sm);
    color: #fb7185;
    font-size: 11px;
    font-weight: 500;
    padding: 4px 10px;
    cursor: pointer;
  }

  .btn-danger-outline:hover:not(:disabled) {
    background-color: rgba(244, 63, 94, 0.12);
  }

  .btn-danger-solid {
    background-color: #e11d48;
    border: none;
    border-radius: var(--radius-sm);
    color: #ffffff;
    font-size: 11px;
    font-weight: 600;
    padding: 5px 10px;
    cursor: pointer;
  }

  .btn-danger-outline:disabled,
  .btn-danger-solid:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
</style>
