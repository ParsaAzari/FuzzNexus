<script lang="ts">
  import { scanStore } from '../stores/scan.svelte';
  import Icon from './Icon.svelte';
</script>

{#if scanStore.pendingResume && scanStore.state === 'Idle'}
  <div class="resume-banner" role="status">
    <div class="resume-info">
      <Icon name="alert-triangle" size={16} />
      <div class="resume-text">
        <strong class="resume-label">Interrupted scan checkpoint found:</strong>
        <span class="resume-method font-mono">{scanStore.pendingResume.method || 'GET'}</span>
        <span class="resume-url font-mono">{scanStore.pendingResume.target_url}</span>
        <span class="resume-meta">
          (Index #{scanStore.pendingResume.current_index}
          {#if scanStore.pendingResume.ip_hash}
            &middot; IP {scanStore.pendingResume.ip_hash}
          {/if})
        </span>
      </div>
    </div>

    <div class="resume-actions">
      <button
        type="button"
        disabled={scanStore.isResumingPending}
        onclick={() => scanStore.resumeSavedScan()}
        class="btn-resume"
      >
        {#if scanStore.isResumingPending}
          Resuming...
        {:else}
          Resume from Index #{scanStore.pendingResume.current_index}
        {/if}
      </button>
      <button
        type="button"
        disabled={scanStore.isResumingPending}
        onclick={() => scanStore.discardSavedScan()}
        class="btn-discard"
      >
        Discard
      </button>
    </div>
  </div>
{/if}

<style>
  .resume-banner {
    margin: 8px 14px 0;
    padding: 10px 14px;
    border-radius: var(--radius-sm);
    border: 1px solid rgba(245, 158, 11, 0.45);
    background-color: rgba(245, 158, 11, 0.12);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    font-size: 12px;
    color: var(--text);
  }

  .resume-info {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
    flex: 1;
    color: #fbbf24;
  }

  .resume-text {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
    min-width: 0;
    color: var(--text);
  }

  .resume-label {
    color: #fbbf24;
  }

  .resume-url {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 360px;
  }

  .resume-meta {
    color: var(--text-muted);
    font-size: 11px;
  }

  .resume-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .btn-resume {
    background-color: #f59e0b;
    border: none;
    border-radius: var(--radius-sm);
    color: #18181b;
    font-size: 11px;
    font-weight: 600;
    padding: 5px 10px;
    cursor: pointer;
  }

  .btn-resume:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-discard {
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    font-size: 11px;
    padding: 5px 10px;
    cursor: pointer;
  }

  .btn-discard:hover:not(:disabled) {
    color: var(--text);
    background-color: var(--surface-hover);
  }
</style>
