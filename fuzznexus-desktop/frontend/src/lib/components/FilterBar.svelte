<script lang="ts">
  import { onMount } from 'svelte';
  import { filtersStore } from '../stores/filters.svelte';
  import { resultsStore } from '../stores/results.svelte';
  import { scanStore } from '../stores/scan.svelte';
  import { toastStore } from '../stores/toast.svelte';
  import Icon from './Icon.svelte';
  import type { FilterRule } from '../types';

  let showPresetsDropdown = $state<boolean>(false);
  let presetNameInput = $state<string>('');
  let isSavingPreset = $state<boolean>(false);
  let quickField = $state<'status' | 'size' | 'words' | 'lines'>('status');
  let quickValue = $state<string>('');

  let filteredCount = $derived(resultsStore.filteredItems.length);
  let totalMatches = $derived(resultsStore.items.length);

  // Dynamic autocomplete suggestions for Quick Exclude based on live scan results
  let quickExcludeSuggestions = $derived.by(() => {
    const items = resultsStore.items;
    const freq = new Map<string, number>();
    for (const item of items) {
      let key = '';
      if (quickField === 'status') key = String(item.status);
      else if (quickField === 'size') key = String(item.length);
      else if (quickField === 'words') key = String(item.words);
      else if (quickField === 'lines') key = String(item.lines);
      if (key) freq.set(key, (freq.get(key) || 0) + 1);
    }
    const topObserved = Array.from(freq.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 5)
      .map(([val, count]) => ({ value: val, label: `${val} (${count} hits)` }));

    if (quickField === 'status') {
      const defaults = ['404', '403', '400', '429', '500', '502', '503'];
      const seen = new Set(topObserved.map((s) => s.value));
      for (const d of defaults) {
        if (!seen.has(d)) topObserved.push({ value: d, label: `${d} (HTTP status)` });
      }
    }
    return topObserved;
  });

  async function handleQuickExclude() {
    const val = quickValue.trim();
    if (!val) return;
    const isScanActive = scanStore.state === 'Running' || scanStore.state === 'Paused';
    filtersStore.addRule({
      field: quickField,
      operator: '!=',
      value: val,
      enabled: true,
      layer: isScanActive || quickField === 'status' ? 'both' : 'view'
    });
    if (isScanActive) {
      try {
        const engineField = quickField === 'words' ? 'word' : quickField === 'lines' ? 'line' : quickField;
        await scanStore.addFilter(engineField, val);
      } catch (err: unknown) {
        console.error('Failed to forward live filter to engine:', err);
      }
    }
    toastStore.success(`Added exclusion filter: ${quickField} != ${val}`);
    quickValue = '';
  }

  onMount(() => {
    filtersStore.loadPresets();
  });

  function handleDocumentClick(e: MouseEvent) {
    if (!showPresetsDropdown) return;
    const target = e.target as HTMLElement;
    if (!target.closest('.presets-menu-wrapper')) {
      showPresetsDropdown = false;
      isSavingPreset = false;
    }
  }

  async function handleSavePreset() {
    const name = presetNameInput.trim();
    if (!name) return;
    await filtersStore.savePreset(name);
    toastStore.success(`Saved filter preset: "${name}"`);
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

<svelte:window onclick={handleDocumentClick} />

<div class="filter-bar select-none">
  <!-- Left Side: Live Search & Active Filter Chips -->
  <div class="chips-container">
    <!-- Live Search / Quick Filter -->
    <div class="search-box">
      <span class="search-icon">
        <Icon name="search" size={13} />
      </span>
      <input
        type="text"
        placeholder="Filter results (URL, status, payload)..."
        bind:value={resultsStore.quickFilter}
        class="search-input font-mono"
      />
      {#if resultsStore.quickFilter}
        <button class="clear-btn" onclick={() => (resultsStore.quickFilter = '')} aria-label="Clear quick filter">
          <Icon name="x" size={11} />
        </button>
      {/if}
    </div>

    <div class="match-counter font-mono">
      {#if resultsStore.quickFilter}
        {filteredCount} / {totalMatches}
      {:else}
        {totalMatches} matches
      {/if}
    </div>

    <span class="filter-sep"></span>

    <button
      class="btn-query-builder"
      onclick={() => (filtersStore.isModalOpen = true)}
      title="Open Visual Query Builder: combine AND/OR rules, ranges (between), regex, and save presets"
    >
      <Icon name="filter" size={14} />
      <span>Filters</span>
      {#if filtersStore.activeRules.length > 0}
        <span class="badge-count font-mono">{filtersStore.activeRules.length}</span>
      {/if}
    </button>

    <div
      class="quick-exclude-group"
      title="Quick Exclude: hide noisy/repetitive responses from table and live scan. Example: Status != 404 hides Not Found, Size != 65015 hides identical Soft-404 pages."
    >
      <select bind:value={quickField} class="quick-field-select font-mono" aria-label="Quick Exclude Field">
        <option value="status">Status !=</option>
        <option value="size">Size !=</option>
        <option value="words">Words !=</option>
        <option value="lines">Lines !=</option>
      </select>
      <input
        type="text"
        list="quick-exclude-suggestions"
        placeholder={quickField === 'status' ? '404, 500...' : quickField === 'size' ? '0, 1024...' : 'value...'}
        bind:value={quickValue}
        onkeydown={(e) => e.key === 'Enter' && handleQuickExclude()}
        class="quick-exclude-input font-mono"
        aria-label="Quick Exclude Value"
      />
      <datalist id="quick-exclude-suggestions">
        {#each quickExcludeSuggestions as sug (sug.value)}
          <option value={sug.value}>{sug.label}</option>
        {/each}
      </datalist>
      <button
        type="button"
        class="btn-quick-exclude"
        disabled={!quickValue.trim()}
        onclick={handleQuickExclude}
        title="Apply exclusion rule (Enter)"
      >
        <Icon name="plus" size={11} />
        <span>Exclude</span>
      </button>
    </div>

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
          <div class="dropdown-header">
            <span>Filter Presets</span>
            <span class="preset-count">{filtersStore.presets.length}</span>
          </div>
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
                      toastStore.info(`Applied filter preset: "${p.name}"`);
                    }}
                    title={`Apply ${p.name}`}
                  >
                    {p.name}
                  </button>
                  <button
                    class="btn-preset-del"
                    onclick={async () => {
                      await filtersStore.deletePreset(p.name);
                      toastStore.info(`Deleted preset: "${p.name}"`);
                    }}
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
                class:is-dimmed={filtersStore.activeRules.length === 0}
                onclick={() => {
                  if (filtersStore.activeRules.length === 0) {
                    toastStore.warning('Add at least one active filter before saving a preset');
                    return;
                  }
                  isSavingPreset = true;
                }}
                title={filtersStore.activeRules.length === 0 ? 'Add at least one active filter to save as preset' : 'Save current active filters as a preset'}
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
    height: 36px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px;
    gap: 8px;
    position: relative;
    z-index: 60;
    overflow: visible;
  }

  .quick-exclude-group {
    display: inline-flex;
    align-items: center;
    height: 24px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 5px;
    overflow: hidden;
    flex-shrink: 0;
  }

  .quick-exclude-group:focus-within {
    border-color: var(--accent);
  }

  .quick-field-select {
    background: transparent;
    border: none;
    border-right: 1px solid var(--border);
    color: var(--text-muted);
    font-size: 11px;
    padding: 0 6px;
    height: 100%;
    outline: none;
    cursor: pointer;
  }

  .quick-exclude-input {
    background: transparent;
    border: none;
    color: var(--text);
    font-size: 11px;
    padding: 0 7px;
    width: 88px;
    height: 100%;
    outline: none;
  }

  .btn-quick-exclude {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    height: 100%;
    padding: 0 8px;
    border: none;
    border-left: 1px solid var(--border);
    background-color: transparent;
    color: var(--text-muted);
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
  }

  .btn-quick-exclude:hover:not(:disabled) {
    background-color: var(--surface-hover);
    color: var(--accent);
  }

  .btn-quick-exclude:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .chips-container {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: 1;
    min-width: 0;
    overflow-x: auto;
  }

  .search-box {
    position: relative;
    display: flex;
    align-items: center;
    width: 240px;
    min-width: 170px;
    height: 24px;
    flex-shrink: 0;
  }

  .search-icon {
    position: absolute;
    left: 7px;
    color: var(--text-secondary);
    pointer-events: none;
    display: flex;
    align-items: center;
  }

  .search-input {
    width: 100%;
    height: 24px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-xs);
    padding: 0 22px 0 24px;
    font-size: 11px;
    color: var(--text);
    box-sizing: border-box;
    outline: none;
    transition: border-color 0.15s ease, box-shadow 0.15s ease;
  }

  .search-input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px var(--focus-ring);
  }

  .clear-btn {
    position: absolute;
    right: 5px;
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: var(--text-xs);
    display: flex;
    align-items: center;
    padding: 2px;
  }

  .clear-btn:hover {
    color: var(--text);
  }

  .match-counter {
    font-size: 11px;
    color: var(--text-secondary);
    white-space: nowrap;
    display: flex;
    align-items: center;
    line-height: 1;
    flex-shrink: 0;
  }

  .filter-sep {
    width: 1px;
    height: 16px;
    background-color: var(--border);
    flex-shrink: 0;
    margin: 0 2px;
  }

  .btn-query-builder {
    height: 26px;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-xs);
    padding: 0 10px;
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
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
    font-size: var(--text-xs);
    padding: 1px 6px;
    border-radius: 8px;
    line-height: 1;
  }

  .filter-chip {
    height: 24px;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 0 8px 0 6px;
    font-size: var(--text-sm);
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
    font-size: var(--text-sm);
  }

  .chip-layer-badge {
    color: #ff9500;
    display: inline-flex;
  }

  .btn-clear-all {
    background: none;
    border: none;
    font-size: var(--text-xs);
    color: var(--text-secondary);
    cursor: pointer;
    padding: 2px 6px;
    border-radius: var(--radius-xs);
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
    z-index: 70;
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
    top: calc(100% + 5px);
    width: 240px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-md, 8px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.35), 0 2px 6px rgba(0, 0, 0, 0.15);
    z-index: 99999;
    overflow: hidden;
  }

  .dropdown-header {
    padding: 8px 10px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .preset-count {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    padding: 1px 6px;
    border-radius: 10px;
    font-size: 10px;
    font-weight: 600;
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

  .btn-save-toggle:hover {
    background-color: var(--surface-hover);
  }

  .btn-save-toggle.is-dimmed {
    opacity: 0.65;
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
