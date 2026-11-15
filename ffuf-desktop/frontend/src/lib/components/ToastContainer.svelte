<script lang="ts">
  import { toastStore } from '../stores/toast.svelte';
  import Icon from './Icon.svelte';

  function getToastIcon(type: string): string {
    switch (type) {
      case 'success':
        return 'check-circle';
      case 'warning':
        return 'alert-triangle';
      case 'error':
        return 'alert-circle';
      default:
        return 'info';
    }
  }
</script>

{#if toastStore.toasts.length > 0}
  <aside class="toast-container select-none" aria-live="polite">
    {#each toastStore.toasts as toast (toast.id)}
      <div class="toast-item {toast.type}">
        <span class="toast-icon">
          <Icon name={getToastIcon(toast.type)} size={16} />
        </span>
        <span class="toast-message">{toast.message}</span>
        <button
          class="btn-toast-close"
          onclick={() => toastStore.removeToast(toast.id)}
          title="Dismiss notice"
          aria-label="Dismiss notice"
        >
          <Icon name="x" size={13} />
        </button>
      </div>
    {/each}
  </aside>
{/if}

<style>
  .toast-container {
    position: fixed;
    bottom: 38px;
    right: 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    z-index: 10001;
    max-width: 380px;
    pointer-events: none;
  }

  .toast-item {
    pointer-events: auto;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    background-color: var(--surface);
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 8px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
    font-size: 12px;
    line-height: 1.4;
    transition: all 0.15s ease;
  }

  .toast-item.success {
    border-left: 4px solid #34c759;
  }

  .toast-item.success .toast-icon {
    color: #34c759;
  }

  .toast-item.error {
    border-left: 4px solid #ff3b30;
  }

  .toast-item.error .toast-icon {
    color: #ff3b30;
  }

  .toast-item.warning {
    border-left: 4px solid #ff9500;
  }

  .toast-item.warning .toast-icon {
    color: #ff9500;
  }

  .toast-item.info {
    border-left: 4px solid var(--accent);
  }

  .toast-item.info .toast-icon {
    color: var(--accent);
  }

  .toast-message {
    flex: 1;
  }

  .btn-toast-close {
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 2px;
    display: flex;
    align-items: center;
    border-radius: 3px;
  }

  .btn-toast-close:hover {
    color: var(--text);
    background-color: var(--surface-hover);
  }
</style>
