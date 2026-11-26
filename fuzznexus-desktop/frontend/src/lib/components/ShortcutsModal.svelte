<script lang="ts">
  import {
    CRITICAL_SHORTCUTS,
    detectIsMac,
    eventToCombo,
    formatComboForDisplay,
    shortcutsStore,
    type ShortcutCategory
  } from '../stores/shortcuts.svelte';
  import { toastStore } from '../stores/toast.svelte';
  import Icon from './Icon.svelte';

  const isMac = detectIsMac();
  const categories: ShortcutCategory[] = ['Scan Engine', 'Navigation', 'Tools'];

  let recordingId = $state<string | null>(null);
  let recordError = $state<string | null>(null);
  let recordWarning = $state<string | null>(null);

  function close() {
    recordingId = null;
    recordError = null;
    recordWarning = null;
    shortcutsStore.isModalOpen = false;
  }

  function handleWindowKeyDown(e: KeyboardEvent) {
    if (!shortcutsStore.isModalOpen) return;

    if (recordingId) {
      e.preventDefault();
      e.stopPropagation();
      if (e.key === 'Escape') {
        recordingId = null;
        recordError = null;
        return;
      }
      const combo = eventToCombo(e, isMac);
      if (!combo) return;
      const res = shortcutsStore.setBinding(recordingId, combo);
      if (!res.ok) {
        recordError = res.error || 'Invalid shortcut';
        recordWarning = null;
        return;
      }
      if (res.warning) {
        recordWarning = res.warning;
        toastStore.warning(`Saved with notice: ${res.warning}`);
      } else {
        recordWarning = null;
        toastStore.success(`Shortcut updated to ${formatComboForDisplay(combo, isMac)}`);
      }
      recordingId = null;
      recordError = null;
      return;
    }

    if (e.key === 'Escape') {
      e.preventDefault();
      close();
    }
  }
</script>

<svelte:window onkeydown={handleWindowKeyDown} />

{#if shortcutsStore.isModalOpen}
  <div
    class="modal-backdrop"
    role="presentation"
    onclick={(e) => e.target === e.currentTarget && close()}
  >
    <div class="modal-card" role="dialog" aria-modal="true" aria-label="Keyboard Shortcuts">
      <div class="modal-header">
        <div class="modal-title-group">
          <Icon name="puzzle" size={16} />
          <h3>Keyboard Shortcuts</h3>
          <span class="platform-pill font-mono">{isMac ? 'macOS' : 'Win / Linux'}</span>
        </div>
        <div class="header-actions">
          <button
            type="button"
            class="btn-reset-all"
            onclick={() => {
              shortcutsStore.resetAll();
              recordError = null;
              toastStore.info('All shortcuts reset to safe defaults');
            }}
          >
            Reset Defaults
          </button>
          <button type="button" class="btn-close" onclick={close} aria-label="Close">
            <Icon name="x" size={15} />
          </button>
        </div>
      </div>

      {#if recordError}
        <div class="error-banner" role="alert">
          <Icon name="alert-triangle" size={13} />
          <span>{recordError}</span>
        </div>
      {:else if recordWarning}
        <div class="warning-banner" role="status">
          <Icon name="info" size={13} />
          <span>{recordWarning}</span>
        </div>
      {/if}

      <div class="modal-body">
        {#each categories as cat}
          <div class="cat-section">
            <div class="cat-title">{cat}</div>
            <div class="shortcut-list">
              {#each CRITICAL_SHORTCUTS.filter((s) => s.category === cat) as item (item.id)}
                {@const isRec = recordingId === item.id}
                {@const combo = shortcutsStore.getCombo(item.id)}
                <div class="shortcut-row" class:recording={isRec}>
                  <span class="shortcut-label">{item.label}</span>
                  <div class="shortcut-controls">
                    <kbd class="kbd-badge font-mono">
                      {isRec ? 'Press keys...' : formatComboForDisplay(combo, isMac)}
                    </kbd>
                    <button
                      type="button"
                      class="btn-record"
                      class:active={isRec}
                      onclick={() => {
                        recordError = null;
                        recordingId = isRec ? null : item.id;
                      }}
                    >
                      {isRec ? 'Cancel' : 'Edit'}
                    </button>
                    {#if combo !== item.defaultCombo}
                      <button
                        type="button"
                        class="btn-reset-one"
                        onclick={() => {
                          shortcutsStore.resetOne(item.id);
                          toastStore.info(`Reset "${item.label}"`);
                        }}
                        title="Reset to default"
                      >
                        <Icon name="rotate-ccw" size={12} />
                      </button>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 2000;
    background: rgba(0, 0, 0, 0.55);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
  }

  .modal-card {
    width: 100%;
    max-width: 560px;
    max-height: 82vh;
    background: var(--surface-elevated, var(--surface));
    border: 1px solid var(--border);
    border-radius: var(--radius-lg, 12px);
    box-shadow: 0 24px 48px rgba(0, 0, 0, 0.35);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 18px;
    border-bottom: 1px solid var(--border);
  }

  .modal-title-group {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .modal-title-group h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
  }

  .platform-pill {
    font-size: 10px;
    padding: 2px 6px;
    border-radius: 999px;
    background: var(--bg-subtle);
    color: var(--text-muted);
    border: 1px solid var(--border);
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .btn-reset-all {
    all: unset;
    font-size: 11px;
    font-weight: 500;
    padding: 4px 10px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    color: var(--text-muted);
    cursor: pointer;
  }

  .btn-reset-all:hover {
    color: var(--text);
    background: var(--surface-hover);
  }

  .btn-close {
    all: unset;
    width: 26px;
    height: 26px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    cursor: pointer;
  }

  .btn-close:hover {
    background: var(--surface-hover);
    color: var(--text);
  }

  .error-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 18px;
    background: rgba(239, 68, 68, 0.12);
    border-bottom: 1px solid rgba(239, 68, 68, 0.35);
    color: var(--danger, #ef4444);
    font-size: 12px;
  }

  .warning-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 18px;
    background: rgba(245, 158, 11, 0.12);
    border-bottom: 1px solid rgba(245, 158, 11, 0.35);
    color: var(--warning, #f59e0b);
    font-size: 12px;
  }

  .modal-body {
    padding: 14px 18px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .cat-title {
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
    margin-bottom: 8px;
  }

  .shortcut-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .shortcut-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 7px 10px;
    border-radius: var(--radius-sm);
    background: var(--bg-subtle);
    border: 1px solid transparent;
  }

  .shortcut-row.recording {
    border-color: var(--accent);
    background: rgba(59, 130, 246, 0.08);
  }

  .shortcut-label {
    font-size: 12.5px;
    color: var(--text);
  }

  .shortcut-controls {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .kbd-badge {
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 5px;
    background: var(--surface);
    border: 1px solid var(--border);
    color: var(--text);
    min-width: 72px;
    text-align: center;
  }

  .btn-record {
    all: unset;
    font-size: 11px;
    font-weight: 500;
    padding: 3px 9px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    color: var(--text-muted);
    cursor: pointer;
  }

  .btn-record:hover,
  .btn-record.active {
    color: var(--accent);
    border-color: var(--accent);
  }

  .btn-reset-one {
    all: unset;
    width: 22px;
    height: 22px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    cursor: pointer;
  }

  .btn-reset-one:hover {
    color: var(--text);
    background: var(--surface-hover);
  }
</style>
