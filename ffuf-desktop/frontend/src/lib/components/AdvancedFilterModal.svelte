<script lang="ts">
  import { filtersStore } from '../stores/filters.svelte';
  import { resultsStore } from '../stores/results.svelte';
  import Icon from './Icon.svelte';
  import type { FilterField, FilterOperator, FilterLayer, FilterRule, FilterGroup } from '../types';

  let totalItems = $derived(resultsStore.items.length);
  let matchedItems = $derived(resultsStore.filteredItems.length);
  let hiddenItems = $derived(Math.max(0, totalItems - matchedItems));
  let hiddenPercent = $derived(totalItems > 0 ? ((hiddenItems / totalItems) * 100).toFixed(1) : '0.0');

  const fields: { value: FilterField; label: string }[] = [
    { value: 'status', label: 'Status Code' },
    { value: 'length', label: 'Content Length' },
    { value: 'words', label: 'Word Count' },
    { value: 'lines', label: 'Line Count' },
    { value: 'duration', label: 'Duration (ms)' },
    { value: 'url', label: 'Target URL' },
    { value: 'host', label: 'Host Header' },
    { value: 'payload', label: 'Payload (FUZZ)' }
  ];

  const numericOperators: { value: FilterOperator; label: string }[] = [
    { value: '=', label: '= (Equal)' },
    { value: '!=', label: '!= (Not Equal)' },
    { value: '<', label: '< (Less than)' },
    { value: '<=', label: '<= (Less or equal)' },
    { value: '>', label: '> (Greater than)' },
    { value: '>=', label: '>= (Greater or equal)' },
    { value: 'between', label: 'Between (Range)' },
    { value: 'in', label: 'IN (Comma separated)' }
  ];

  const textOperators: { value: FilterOperator; label: string }[] = [
    { value: 'contains', label: 'Contains (Substring)' },
    { value: 'not_contains', label: 'Not Contains' },
    { value: '=', label: '= (Exact Match)' },
    { value: '!=', label: '!= (Not Exact Match)' },
    { value: 'regex', label: 'Regular Expression (ReDoS Safe)' }
  ];

  function getOperatorsForField(field: FilterField) {
    if (['status', 'length', 'words', 'lines', 'duration'].includes(field)) {
      return numericOperators;
    }
    return textOperators;
  }

  function handleFieldChange(rule: FilterRule, newField: FilterField) {
    rule.field = newField;
    const ops = getOperatorsForField(newField);
    if (!ops.some((o) => o.value === rule.operator)) {
      rule.operator = ops[0].value;
    }
    filtersStore.updateRule(rule.id, { field: rule.field, operator: rule.operator });
  }

  function validateRegex(pattern: string): string | null {
    if (!pattern) return null;
    if (pattern.length > 256) {
      return 'Regex length exceeds 256 character safety limit (ReDoS protection).';
    }
    try {
      new RegExp(pattern);
      return null;
    } catch (e: any) {
      return `Invalid regular expression syntax: ${e.message}`;
    }
  }
</script>

{#if filtersStore.isModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) filtersStore.isModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (filtersStore.isModalOpen = false)}
  >
    <div class="modal-window" role="document">
      <!-- Modal Header -->
      <div class="modal-header">
        <div class="header-title">
          <Icon name="filter" size={18} />
          <span>Advanced Query Builder</span>
        </div>
        <div class="header-actions">
          <button
            class="btn-icon"
            disabled={!filtersStore.canUndo}
            onclick={() => filtersStore.undo()}
            title="Undo (Cmd+Z)"
          >
            <Icon name="undo" size={14} />
          </button>
          <button
            class="btn-icon"
            disabled={!filtersStore.canRedo}
            onclick={() => filtersStore.redo()}
            title="Redo (Cmd+Shift+Z)"
          >
            <Icon name="redo" size={14} />
          </button>
          <button
            class="btn-close"
            onclick={() => (filtersStore.isModalOpen = false)}
            title="Close Query Builder"
          >
            <Icon name="x" size={16} />
          </button>
        </div>
      </div>

      <!-- Live Match Preview Banner -->
      <div class="match-banner font-mono">
        <Icon name="search" size={14} />
        <span>
          Matches <strong>{matchedItems.toLocaleString()}</strong> of {totalItems.toLocaleString()} items
          {#if hiddenItems > 0}
            (Hides {hiddenItems.toLocaleString()} &bull; {hiddenPercent}% filtered out)
          {/if}
        </span>
      </div>

      <!-- Query Builder Rules Tree -->
      <div class="modal-body">
        <div class="group-container root-group">
          <!-- Root Conjunction Bar -->
          <div class="group-header">
            <span class="group-label">Match</span>
            <div class="conjunction-picker">
              <button
                class="btn-conj"
                class:active={filtersStore.rootGroup.conjunction === 'AND'}
                onclick={() => (filtersStore.rootGroup.conjunction = 'AND')}
              >
                ALL (AND)
              </button>
              <button
                class="btn-conj"
                class:active={filtersStore.rootGroup.conjunction === 'OR'}
                onclick={() => (filtersStore.rootGroup.conjunction = 'OR')}
              >
                ANY (OR)
              </button>
            </div>
            <span class="group-label">of the following conditions:</span>
          </div>

          <!-- Root Rules List -->
          <div class="rules-list">
            {#each filtersStore.rootGroup.rules as rule (rule.id)}
              <div class="rule-row" class:rule-disabled={!rule.enabled}>
                <!-- Toggle enabled -->
                <button
                  class="btn-rule-toggle"
                  onclick={() => filtersStore.toggleRule(rule.id)}
                  title={rule.enabled ? 'Disable condition' : 'Enable condition'}
                >
                  <Icon name={rule.enabled ? 'eye' : 'eye-off'} size={14} />
                </button>

                <!-- Field Selector -->
                <select
                  class="rule-select field-select"
                  value={rule.field}
                  onchange={(e) => handleFieldChange(rule, (e.currentTarget as HTMLSelectElement).value as FilterField)}
                >
                  {#each fields as f}
                    <option value={f.value}>{f.label}</option>
                  {/each}
                </select>

                <!-- Operator Selector -->
                <select
                  class="rule-select op-select"
                  bind:value={rule.operator}
                  onchange={() => filtersStore.updateRule(rule.id, { operator: rule.operator })}
                >
                  {#each getOperatorsForField(rule.field as FilterField) as op}
                    <option value={op.value}>{op.label}</option>
                  {/each}
                </select>

                <!-- Value Input -->
                <input
                  type="text"
                  class="rule-input font-mono"
                  placeholder="Value..."
                  bind:value={rule.value}
                  oninput={() => filtersStore.updateRule(rule.id, { value: rule.value })}
                />

                <!-- Value2 Input (for Between) -->
                {#if rule.operator === 'between'}
                  <span class="sep-text">and</span>
                  <input
                    type="text"
                    class="rule-input input-narrow font-mono"
                    placeholder="Max value..."
                    bind:value={rule.value2}
                    oninput={() => filtersStore.updateRule(rule.id, { value2: rule.value2 })}
                  />
                {/if}

                <!-- Layer Selector -->
                <select
                  class="rule-select layer-select"
                  bind:value={rule.layer}
                  onchange={() => filtersStore.updateRule(rule.id, { layer: rule.layer })}
                  title="Filter execution layer (View client vs Live Engine)"
                >
                  <option value="view">View Only</option>
                  <option value="engine">Engine Only</option>
                  <option value="both">View + Engine</option>
                </select>

                <!-- Delete Rule -->
                <button
                  class="btn-rule-del"
                  onclick={() => filtersStore.removeRule(rule.id)}
                  title="Delete condition"
                >
                  <Icon name="trash" size={14} />
                </button>
              </div>

              {#if rule.operator === 'regex' && validateRegex(rule.value)}
                <div class="regex-warning font-mono">
                  <Icon name="alert-triangle" size={12} />
                  <span>{validateRegex(rule.value)}</span>
                </div>
              {/if}
            {/each}
          </div>

          <!-- Nested Groups (up to 2 levels) -->
          {#each filtersStore.rootGroup.groups || [] as subGroup (subGroup.id)}
            <div class="nested-group">
              <div class="nested-header">
                <span class="nested-title">Match</span>
                <div class="conjunction-picker">
                  <button
                    class="btn-conj"
                    class:active={subGroup.conjunction === 'AND'}
                    onclick={() => (subGroup.conjunction = 'AND')}
                  >
                    ALL (AND)
                  </button>
                  <button
                    class="btn-conj"
                    class:active={subGroup.conjunction === 'OR'}
                    onclick={() => (subGroup.conjunction = 'OR')}
                  >
                    ANY (OR)
                  </button>
                </div>
                <button
                  class="btn-remove-subgroup"
                  onclick={() => filtersStore.removeGroup(subGroup.id)}
                  title="Remove group"
                >
                  <Icon name="x" size={12} /> Remove Group
                </button>
              </div>

              <div class="nested-rules">
                {#each subGroup.rules as rule (rule.id)}
                  <div class="rule-row" class:rule-disabled={!rule.enabled}>
                    <button
                      class="btn-rule-toggle"
                      onclick={() => filtersStore.toggleRule(rule.id)}
                    >
                      <Icon name={rule.enabled ? 'eye' : 'eye-off'} size={14} />
                    </button>

                    <select
                      class="rule-select field-select"
                      value={rule.field}
                      onchange={(e) => handleFieldChange(rule, (e.currentTarget as HTMLSelectElement).value as FilterField)}
                    >
                      {#each fields as f}
                        <option value={f.value}>{f.label}</option>
                      {/each}
                    </select>

                    <select
                      class="rule-select op-select"
                      bind:value={rule.operator}
                      onchange={() => filtersStore.updateRule(rule.id, { operator: rule.operator })}
                    >
                      {#each getOperatorsForField(rule.field as FilterField) as op}
                        <option value={op.value}>{op.label}</option>
                      {/each}
                    </select>

                    <input
                      type="text"
                      class="rule-input font-mono"
                      placeholder="Value..."
                      bind:value={rule.value}
                      oninput={() => filtersStore.updateRule(rule.id, { value: rule.value })}
                    />

                    {#if rule.operator === 'between'}
                      <span class="sep-text">and</span>
                      <input
                        type="text"
                        class="rule-input input-narrow font-mono"
                        placeholder="Max..."
                        bind:value={rule.value2}
                        oninput={() => filtersStore.updateRule(rule.id, { value2: rule.value2 })}
                      />
                    {/if}

                    <button
                      class="btn-rule-del"
                      onclick={() => filtersStore.removeRule(rule.id)}
                    >
                      <Icon name="trash" size={14} />
                    </button>
                  </div>
                {/each}

                <button
                  class="btn-add-condition-nested"
                  onclick={() => filtersStore.addRule({}, subGroup.id)}
                >
                  <Icon name="plus" size={12} /> Add Condition in Group
                </button>
              </div>
            </div>
          {/each}

          <!-- Group Action Triggers -->
          <div class="group-actions">
            <button class="btn-action" onclick={() => filtersStore.addRule({})}>
              <Icon name="plus" size={13} />
              <span>Add Condition</span>
            </button>
            <button class="btn-action" onclick={() => filtersStore.addGroup()}>
              <Icon name="plus" size={13} />
              <span>Add Nested Group</span>
            </button>
            <button
              class="btn-action btn-danger-action"
              onclick={() => filtersStore.clearFilters()}
              disabled={filtersStore.rootGroup.rules.length === 0}
            >
              <Icon name="trash" size={13} />
              <span>Reset All</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="modal-footer">
        <button class="btn-secondary" onclick={() => (filtersStore.isModalOpen = false)}>
          Close
        </button>
        <button class="btn-primary" onclick={() => (filtersStore.isModalOpen = false)}>
          Apply Filters
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
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

  .modal-window {
    width: 780px;
    max-width: 95vw;
    max-height: 85vh;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.3);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .modal-header {
    height: 44px;
    padding: 0 16px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .header-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    font-weight: 600;
    color: var(--text);
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .btn-icon,
  .btn-close {
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .btn-icon:hover:not(:disabled),
  .btn-close:hover {
    color: var(--text);
    background-color: var(--surface-hover);
  }

  .btn-icon:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .match-banner {
    padding: 8px 16px;
    background-color: rgba(0, 122, 255, 0.08);
    border-bottom: 1px solid var(--border);
    font-size: 12px;
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--accent);
  }

  .modal-body {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
  }

  .group-container {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .group-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
  }

  .group-label {
    color: var(--text-secondary);
  }

  .conjunction-picker {
    display: flex;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 4px;
    overflow: hidden;
  }

  .btn-conj {
    height: 24px;
    padding: 0 8px;
    border: none;
    background: none;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    cursor: pointer;
  }

  .btn-conj.active {
    background-color: var(--accent);
    color: var(--accent-contrast);
  }

  .rules-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .rule-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 6px;
  }

  .rule-disabled {
    opacity: 0.5;
  }

  .btn-rule-toggle,
  .btn-rule-del {
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .btn-rule-del:hover {
    color: #ff3b30;
  }

  .rule-select {
    height: 28px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    font-size: 12px;
    color: var(--text);
    padding: 0 6px;
  }

  .field-select {
    width: 140px;
  }

  .op-select {
    width: 130px;
  }

  .layer-select {
    width: 110px;
    font-size: 11px;
    color: var(--text-secondary);
  }

  .rule-input {
    flex: 1;
    height: 28px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    font-size: 12px;
    color: var(--text);
    padding: 0 8px;
    box-sizing: border-box;
  }

  .input-narrow {
    width: 100px;
    flex: none;
  }

  .sep-text {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .regex-warning {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    background-color: rgba(255, 59, 48, 0.1);
    color: #ff3b30;
    font-size: 11px;
    border-radius: 4px;
    margin-top: -4px;
    margin-left: 36px;
  }

  .nested-group {
    background-color: var(--surface-hover);
    border: 1px dashed var(--border);
    border-radius: 6px;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-left: 20px;
  }

  .nested-header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .nested-title {
    font-size: 11px;
    font-weight: 600;
  }

  .btn-remove-subgroup {
    background: none;
    border: none;
    color: var(--text-secondary);
    font-size: 11px;
    cursor: pointer;
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .btn-remove-subgroup:hover {
    color: #ff3b30;
  }

  .nested-rules {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .btn-add-condition-nested {
    align-self: flex-start;
    height: 24px;
    background: none;
    border: 1px dashed var(--border);
    border-radius: 4px;
    font-size: 11px;
    color: var(--accent);
    cursor: pointer;
    padding: 0 8px;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .group-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 6px;
  }

  .btn-action {
    height: 28px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 10px;
    font-size: 11px;
    color: var(--text);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .btn-action:hover:not(:disabled) {
    background-color: var(--surface-hover);
    border-color: var(--accent);
  }

  .btn-danger-action {
    margin-left: auto;
    color: var(--text-secondary);
  }

  .btn-danger-action:hover:not(:disabled) {
    color: #ff3b30;
    border-color: #ff3b30;
  }

  .modal-footer {
    height: 50px;
    padding: 0 16px;
    background-color: var(--bg-subtle);
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
  }

  .btn-secondary {
    height: 32px;
    padding: 0 14px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    font-size: 12px;
    color: var(--text);
    cursor: pointer;
  }

  .btn-primary {
    height: 32px;
    padding: 0 16px;
    background-color: var(--accent);
    color: var(--accent-contrast);
    border: none;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }
</style>
