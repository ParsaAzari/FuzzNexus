<script lang="ts">
  import { onMount } from 'svelte';
  import { filtersStore } from '../stores/filters.svelte';
  import { resultsStore } from '../stores/results.svelte';
  import Icon from './Icon.svelte';
  import type { FilterRule } from '../types';

  let showPresetsDropdown = $state<boolean>(false);
  let presetNameInput = $state<string>('');
  let isSavingPreset = $state<boolean>(false);

  onMount(() => {
    filtersStore.loadPresets();
  });

  function handleSavePreset() {
    if (!presetNameInput.trim()) return;
    filtersStore.savePreset(presetNameInput.trim());
    presetNameInput = '';
    isSavingPreset = false;
  }

  function formatRuleChip(rule: FilterRule): string {
    if (rule.operator === 'between') {
      return `${rule.field} ${rule.operator} ${rule.value}..${rule.value2 || ''}`;
    }
    return `${rule.field} ${rule.operator} ${rule.value}`;
  }
</script>

<div class="filter-bar select-none">
  <!-- Left Side: Active Filter Chips -->
  <div class="chips-container">
    <button
      class="btn-query-builder"
      onclick={() => (filtersStore.isModalOpen = true)}
      title="Open Visual Query Builder"
    >
      <Icon name="filter" size={14} />
      <span>Filters</span>
      {#if filtersStore.activeRules.length > 0}
        <span class="badge-count font-mono">{filtersStore.activeRules.length}</span>
      {/if}
    </button>

    {#each filtersStore.activeRules as rule (rule.id)}
      <div class="filter-chip" class:disabled={!rule.enabled}>
        <button
          class="chip-toggle"
          onclick={() => filtersStore.toggleRule(rule.id)}
          title={rule.enabled ? 'Disable filter' : 'Enable filter'}
          aria-label="Toggle rule"
        >
          <Icon name={rule.enabled ? 'eye' : 'eye-off'} size={12} />
        </button>

        <button
          type="button"
          class="chip-label font-mono"
          onclick={() => (filtersStore.isModalOpen = true)}
          title="Click to edit filter in Query Builder"
        >
          {formatRuleChip(rule)}
        </button>

        {#if rule.layer === 'engine' || rule.layer === 'both'}
          <span class="chip-layer-badge" title="Engine live filter active">
            <Icon name="zap" size={10} />
          </span>
        {/if}

        <button
          class="chip-remove"
          onclick={() => filtersStore.removeRule(rule.id)}
          title="Remove filter rule"
          aria-label="Remove rule"
        >
          <Icon name="x" size={11} />
        </button>
      </div>
    {/each}

    {#if filtersStore.activeRules.length > 0}
      <button
        class="btn-clear-all"
        onclick={() => filtersStore.clearFilters()}
        title="Clear all active filters"
      >
        Clear
      </button>
    {/if}
  </div>

  <!-- Right Side: Undo/Redo & Presets -->
  <div class="filter-controls">
    <!-- Undo / Redo Group -->
    <div class="history-group">
      <button
        class="btn-icon"
        disabled={!filtersStore.canUndo}
        onclick={() => filtersStore.undo()}
        title="Undo filter change (Cmd+Z)"
        aria-label="Undo filter"
      >
        <Icon name="undo" size={13} />
      </button>
      <button
        class="btn-icon"
        disabled={!filtersStore.canRedo}
        onclick={() => filtersStore.redo()}
        title="Redo filter change (Cmd+Shift+Z)"
        aria-label="Redo filter"
      >
        <Icon name="redo" size={13} />
      </button>
    </div>

    <!-- Presets Dropdown Trigger -->
    <div class="presets-menu-wrapper">
      <button
        class="btn-presets"
        onclick={() => (showPresetsDropdown = !showPresetsDropdown)}
        title="Filter Presets"
      >
        <Icon name="save" size={13} />
        <span>Presets</span>
        <Icon name="chevron-down" size={11} />
      </button>

      {#if showPresetsDropdown}
        <div class="presets-dropdown">
          <div class="dropdown-header">Filter Presets</div>
          {#if filtersStore.presets.length === 0}
            <div class="dropdown-empty">No saved presets</div>
          {:else}
            <div class="presets-list">
              {#each filtersStore.presets as p (p.name)}
                <div class="preset-item">
                  <button
                    type="button"
                    class="preset-name"
                    onclick={() => {
                      filtersStore.applyPreset(p);
                      showPresetsDropdown = false;
                    }}
                  >
                    {p.name}
                  </button>
                  <button
                    class="btn-preset-del"
                    onclick={() => filtersStore.deletePreset(p.name)}
                    title="Delete preset"
                  >
                    <Icon name="trash" size={12} />
                  </button>
                </div>
              {/each}
            </div>
          {/if}

          <div class="dropdown-footer">
            {#if !isSavingPreset}
              <button
                class="btn-save-toggle"
                onclick={() => (isSavingPreset = true)}
                disabled={filtersStore.activeRules.length === 0}
              >
                + Save Current as Preset
              </button>
            {:else}
              <div class="preset-save-form">
                <input
                  type="text"
                  placeholder="Preset name..."
                  bind:value={presetNameInput}
                  onkeydown={(e) => e.key === 'Enter' && handleSavePreset()}
                  class="preset-input"
                />
                <button class="btn-save-confirm" onclick={handleSavePreset}>Save</button>
                <button class="btn-save-cancel" onclick={() => (isSavingPreset = false)}>
                  <Icon name="x" size={12} />
                </button>
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .filter-bar {
    height: 34px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px;
    gap: 8px;
    overflow-x: auto;
    overflow-y: hidden;
  }

  .chips-container {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: 1;
    min-width: 0;
    overflow-x: auto;
  }

  .btn-query-builder {
    height: 24px;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 8px;
    font-size: 11px;
    font-weight: 500;
    color: var(--text);
    cursor: pointer;
    flex-shrink: 0;
    transition: all 0.1s ease;
  }

  .btn-query-builder:hover {
    background-color: var(--surface-hover);
    border-color: var(--accent);
  }

  .badge-count {
    background-color: var(--accent);
    color: var(--accent-contrast);
    font-size: 10px;
    padding: 1px 5px;
    border-radius: 8px;
    line-height: 1;
  }

  .filter-chip {
    height: 22px;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 11px;
    padding: 0 6px 0 4px;
    font-size: 11px;
    color: var(--text);
    flex-shrink: 0;
    transition: all 0.1s ease;
  }

  .filter-chip.disabled {
    opacity: 0.5;
    background-color: var(--bg-subtle);
    border-style: dashed;
  }

  .chip-toggle,
  .chip-remove {
    background: none;
    border: none;
    padding: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: 3px;
  }

  .chip-toggle:hover,
  .chip-remove:hover {
    color: var(--text);
  }

  .chip-label {
    background: none;
    border: none;
    padding: 0;
    color: inherit;
    cursor: pointer;
    font-size: 10.5px;
  }

  .chip-layer-badge {
    color: #ff9500;
    display: inline-flex;
  }

  .btn-clear-all {
    background: none;
    border: none;
    font-size: 11px;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .btn-clear-all:hover {
    color: var(--text);
    text-decoration: underline;
  }

  .filter-controls {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .history-group {
    display: flex;
    align-items: center;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    overflow: hidden;
  }

  .btn-icon {
    width: 24px;
    height: 22px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    color: var(--text);
    cursor: pointer;
  }

  .btn-icon:hover:not(:disabled) {
    background-color: var(--surface-hover);
  }

  .btn-icon:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .presets-menu-wrapper {
    position: relative;
  }

  .btn-presets {
    height: 24px;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 8px;
    font-size: 11px;
    color: var(--text);
    cursor: pointer;
  }

  .btn-presets:hover {
    background-color: var(--surface-hover);
  }

  .presets-dropdown {
    position: absolute;
    right: 0;
    top: 28px;
    width: 220px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
    z-index: 1000;
    overflow: hidden;
  }

  .dropdown-header {
    padding: 8px 10px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    border-bottom: 1px solid var(--border);
  }

  .dropdown-empty {
    padding: 12px 10px;
    font-size: 11px;
    color: var(--text-secondary);
    text-align: center;
  }

  .presets-list {
    max-height: 180px;
    overflow-y: auto;
  }

  .preset-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 10px;
    font-size: 11px;
    border-bottom: 1px solid rgba(128, 128, 128, 0.1);
  }

  .preset-name {
    flex: 1;
    background: none;
    border: none;
    padding: 0;
    color: inherit;
    font-size: 11px;
    text-align: left;
    cursor: pointer;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .preset-name:hover {
    color: var(--accent);
  }

  .btn-preset-del {
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 2px;
  }

  .btn-preset-del:hover {
    color: #ff3b30;
  }

  .dropdown-footer {
    padding: 8px;
    background-color: var(--bg-subtle);
    border-top: 1px solid var(--border);
  }

  .btn-save-toggle {
    width: 100%;
    height: 24px;
    background: none;
    border: 1px dashed var(--border);
    border-radius: 4px;
    font-size: 11px;
    color: var(--accent);
    cursor: pointer;
  }

  .btn-save-toggle:hover:not(:disabled) {
    background-color: var(--surface-hover);
  }

  .btn-save-toggle:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .preset-save-form {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .preset-input {
    flex: 1;
    height: 22px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 3px;
    padding: 0 6px;
    font-size: 11px;
    color: var(--text);
  }

  .btn-save-confirm {
    height: 22px;
    background-color: var(--accent);
    color: var(--accent-contrast);
    border: none;
    border-radius: 3px;
    font-size: 10px;
    font-weight: 600;
    padding: 0 6px;
    cursor: pointer;
  }

  .btn-save-cancel {
    height: 22px;
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
  }
</style>
