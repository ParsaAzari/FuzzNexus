<script lang="ts">
  import Icon from './Icon.svelte';

  interface Props {
    isOpen: boolean;
    onCancel: () => void;
    onConfirm: () => Promise<void> | void;
  }

  let { isOpen, onCancel, onConfirm }: Props = $props();
  let isQuitting = $state<boolean>(false);

  $effect(() => {
    if (!isOpen) {
      isQuitting = false;
    }
  });

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape' && isOpen && !isQuitting) {
      onCancel();
    }
  }

  async function handleConfirm() {
    if (isQuitting) return;
    isQuitting = true;
    try {
      await onConfirm();
    } catch {
      isQuitting = false;
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen}
  <div
    class="confirm-backdrop"
    onclick={(e) => {
      if (e.target === e.currentTarget && !isQuitting) onCancel();
    }}
    role="presentation"
  >
    <div
      class="confirm-dialog"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="confirm-close-title"
      aria-describedby="confirm-close-desc"
    >
      <div class="confirm-header">
        <div class="confirm-icon-wrap">
          <Icon name="alert-triangle" size={20} />
        </div>
        <div class="confirm-header-text">
          <h2 id="confirm-close-title" class="confirm-title">Scan in Progress</h2>
          <p id="confirm-close-desc" class="confirm-desc">
            A fuzzing scan is still active. Stopping now will halt in-flight requests and save current results before closing FuzzNexus.
          </p>
        </div>
      </div>

      <div class="confirm-footer">
        <button
          type="button"
          class="btn-cancel"
          onclick={onCancel}
          disabled={isQuitting}
        >
          Keep Scanning
        </button>
        <button
          type="button"
          class="btn-quit"
          onclick={handleConfirm}
          disabled={isQuitting}
        >
          {isQuitting ? 'Stopping...' : 'Stop & Quit'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .confirm-backdrop {
    position: fixed;
    inset: 0;
    z-index: 10060;
    background: rgba(0, 0, 0, 0.62);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    transform: translateZ(0);
    display: flex;
    align-items: safe center;
    justify-content: center;
    padding: 16px;
    overflow-y: auto;
    animation: fadeIn 0.14s ease-out;
  }

  .confirm-dialog {
    width: 100%;
    max-width: 430px;
    margin: auto;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg, 12px);
    box-shadow: 0 20px 48px rgba(0, 0, 0, 0.45);
    overflow: hidden;
    animation: scaleIn 0.14s ease-out;
  }

  .confirm-header {
    display: flex;
    align-items: flex-start;
    gap: 14px;
    padding: 20px 20px 16px;
  }

  .confirm-icon-wrap {
    width: 38px;
    height: 38px;
    border-radius: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    background-color: rgba(245, 158, 11, 0.14);
    color: #f59e0b;
    border: 1px solid rgba(245, 158, 11, 0.3);
  }

  .confirm-header-text {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .confirm-title {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    color: var(--text);
    letter-spacing: -0.01em;
  }

  .confirm-desc {
    margin: 0;
    font-size: 12.5px;
    line-height: 1.5;
    color: var(--text-secondary);
  }

  .confirm-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    padding: 12px 20px;
    background-color: var(--bg-subtle);
    border-top: 1px solid var(--border);
  }

  .btn-cancel,
  .btn-quit {
    height: 32px;
    padding: 0 14px;
    border-radius: var(--radius-sm, 6px);
    font-size: 12.5px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.14s ease;
  }

  .btn-cancel {
    background-color: var(--surface);
    color: var(--text);
    border: 1px solid var(--border);
  }

  .btn-cancel:hover:not(:disabled) {
    background-color: var(--surface-hover);
  }

  .btn-quit {
    background-color: #ef4444;
    color: #ffffff;
    border: 1px solid #dc2626;
  }

  .btn-quit:hover:not(:disabled) {
    background-color: #dc2626;
  }

  .btn-cancel:disabled,
  .btn-quit:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes scaleIn {
    from { opacity: 0; transform: scale(0.96); }
    to { opacity: 1; transform: scale(1); }
  }
</style>
