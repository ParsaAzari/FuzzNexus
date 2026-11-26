<script lang="ts">
  import Icon from './Icon.svelte';
  import { scanStore } from '../stores/scan.svelte';
  import { toastStore } from '../stores/toast.svelte';

  interface Props {
    isOpen: boolean;
    onClose: () => void;
  }

  let { isOpen, onClose }: Props = $props();

  let isCopied = $state<boolean>(false);

  async function handleCopy() {
    if (scanStore.recentErrors.length === 0) return;
    const text = scanStore.recentErrors
      .map((e) => `[${new Date(e.timestamp).toLocaleTimeString()}] #${e.index}: ${e.message}`)
      .join('\n');
    try {
      await navigator.clipboard.writeText(text);
      isCopied = true;
      toastStore.success('Error diagnostics copied to clipboard');
      setTimeout(() => {
        isCopied = false;
      }, 2000);
    } catch {
      toastStore.error('Failed to copy to clipboard');
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape' && isOpen) {
      onClose();
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen}
  <div
    class="modal-backdrop"
    onclick={(e) => {
      if (e.target === e.currentTarget) onClose();
    }}
    role="presentation"
  >
    <div class="modal-dialog" role="dialog" aria-modal="true" aria-labelledby="diag-modal-title">
      <!-- Modal Header with Native Window Dragging -->
      <div class="modal-header" style="--wails-draggable: drag;">
        <div class="header-title-wrap" style="--wails-draggable: drag;">
          <Icon name="alert-triangle" size={18} class="header-icon" />
          <h2 id="diag-modal-title" class="modal-title">
            Network Error Diagnostics
          </h2>
          {#if scanStore.errorCount > 0}
            <span class="badge-err-count font-mono">{scanStore.errorCount} total errors</span>
          {/if}
        </div>
        <button
          type="button"
          class="btn-close-modal"
          onclick={onClose}
          aria-label="Close"
          style="--wails-draggable: no-drag;"
        >
          <Icon name="close" size={16} />
        </button>
      </div>

      <!-- Modal Body -->
      <div class="modal-body">
        <p class="diag-desc">
          Live stream of recent low-level socket, timeout, DNS, and connection-level failures captured by the engine.
        </p>

        {#if scanStore.recentErrors.length === 0}
          <div class="empty-state">
            <Icon name="check-circle" size={28} class="empty-icon" />
            <span class="empty-title">No Recent Network Errors</span>
            <span class="empty-sub">
              {scanStore.errorCount > 0
                ? 'Errors occurred before this session or have cleared.'
                : 'All requests dispatched by the engine are completing normally.'}
            </span>
          </div>
        {:else}
          <div class="errors-list-container">
            <table class="errors-table font-mono">
              <thead>
                <tr>
                  <th style="width: 50px;">#</th>
                  <th style="width: 90px;">Time</th>
                  <th>Diagnostic Message / Reason</th>
                </tr>
              </thead>
              <tbody>
                {#each scanStore.recentErrors as err, i (`${err.index}-${i}`)}
                  <tr>
                    <td class="col-idx">{err.index}</td>
                    <td class="col-time">{new Date(err.timestamp).toLocaleTimeString()}</td>
                    <td class="col-msg">{err.message}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>

      <!-- Modal Footer -->
      <div class="modal-footer">
        <div class="footer-left">
          <span class="footer-tip">Displaying up to 100 recent occurrences</span>
        </div>
        <div class="footer-actions">
          <button
            type="button"
            class="btn-modal btn-copy"
            onclick={handleCopy}
            disabled={scanStore.recentErrors.length === 0}
          >
            <Icon name={isCopied ? 'check' : 'copy'} size={14} />
            <span>{isCopied ? 'Copied' : 'Copy All'}</span>
          </button>
          <button
            type="button"
            class="btn-modal btn-primary"
            onclick={onClose}
          >
            Close
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(4px);
    z-index: 10050;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
    animation: fadeIn 0.15s ease-out;
  }

  .modal-dialog {
    width: 100%;
    max-width: 680px;
    max-height: 85vh;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: 0 16px 36px rgba(0, 0, 0, 0.4);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    animation: scaleIn 0.15s ease-out;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.85rem 1.25rem;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    user-select: none;
  }

  .header-title-wrap {
    display: flex;
    align-items: center;
    gap: 0.6rem;
  }

  .header-title-wrap :global(.header-icon) {
    color: var(--danger);
  }

  .modal-title {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text);
    margin: 0;
  }

  .badge-err-count {
    font-size: 0.72rem;
    padding: 0.15rem 0.5rem;
    border-radius: 9999px;
    background-color: rgba(239, 68, 68, 0.15);
    color: var(--danger);
    border: 1px solid rgba(239, 68, 68, 0.3);
  }

  .btn-close-modal {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 0.3rem;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s ease;
  }

  .btn-close-modal:hover {
    color: var(--text);
    background-color: var(--hover-bg);
  }

  .modal-body {
    padding: 1.2rem;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    flex: 1;
    min-height: 240px;
  }

  .diag-desc {
    font-size: 0.82rem;
    color: var(--text-muted);
    margin: 0;
    line-height: 1.4;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 3rem 1rem;
    gap: 0.5rem;
    text-align: center;
    color: var(--text-muted);
  }

  .empty-state :global(.empty-icon) {
    color: var(--accent);
    margin-bottom: 0.25rem;
  }

  .empty-title {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text);
  }

  .empty-sub {
    font-size: 0.8rem;
    max-width: 360px;
  }

  .errors-list-container {
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    background-color: var(--bg);
    overflow: auto;
    max-height: 380px;
  }

  .errors-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.78rem;
    text-align: left;
  }

  .errors-table thead {
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    position: sticky;
    top: 0;
    z-index: 1;
  }

  .errors-table th {
    padding: 0.5rem 0.75rem;
    font-weight: 600;
    color: var(--text-muted);
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .errors-table td {
    padding: 0.45rem 0.75rem;
    border-bottom: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.05));
    vertical-align: top;
  }

  .errors-table tbody tr:hover {
    background-color: var(--hover-bg);
  }

  .col-idx {
    color: var(--text-muted);
  }

  .col-time {
    color: var(--text-muted);
    white-space: nowrap;
  }

  .col-msg {
    color: #f87171;
    word-break: break-all;
    line-height: 1.4;
  }

  .modal-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.85rem 1.25rem;
    background-color: var(--bg-subtle);
    border-top: 1px solid var(--border);
  }

  .footer-tip {
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .footer-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .btn-modal {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.45rem 0.9rem;
    font-size: 0.82rem;
    font-weight: 500;
    border-radius: var(--radius-md);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-copy {
    background-color: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
  }

  .btn-copy:hover:not(:disabled) {
    background-color: var(--hover-bg);
    border-color: var(--border-hover);
  }

  .btn-copy:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .btn-primary {
    background-color: var(--accent);
    border: 1px solid var(--accent);
    color: var(--accent-contrast, #fff);
  }

  .btn-primary:hover {
    filter: brightness(1.1);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes scaleIn {
    from {
      opacity: 0;
      transform: scale(0.96);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }
</style>
