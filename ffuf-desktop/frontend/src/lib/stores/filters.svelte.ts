import {
  GetFilterPresets,
  SaveFilterPreset,
  DeleteFilterPreset
} from '../../../wailsjs/go/main/App';
import type {
  FilterGroup,
  FilterRule,
  FilterPresetDTO,
  FilterField,
  FilterOperator,
  FilterLayer,
  ResultDTO
} from '../types';
import { scanStore } from './scan.svelte';

function createDefaultRoot(): FilterGroup {
  return {
    id: 'root',
    conjunction: 'AND',
    rules: [],
    groups: [],
    enabled: true
  };
}

function cloneGroup(g: FilterGroup): FilterGroup {
  return JSON.parse(JSON.stringify(g));
}

class FiltersStore {
  rootGroup = $state<FilterGroup>(createDefaultRoot());
  presets = $state<FilterPresetDTO[]>([]);
  isModalOpen = $state<boolean>(false);

  private history = $state<FilterGroup[]>([]);
  private future = $state<FilterGroup[]>([]);

  canUndo = $derived(this.history.length > 0);
  canRedo = $derived(this.future.length > 0);

  // Flattened active rules for quick chips
  activeRules = $derived.by(() => {
    const list: FilterRule[] = [];
    const collect = (g: FilterGroup) => {
      if (!g.enabled) return;
      for (const r of g.rules) {
        list.push(r);
      }
      for (const sub of g.groups || []) {
        collect(sub);
      }
    };
    collect(this.rootGroup);
    return list;
  });

  private pushHistory() {
    this.history = [...this.history.slice(-24), cloneGroup(this.rootGroup)];
    this.future = [];
  }

  undo() {
    if (this.history.length === 0) return;
    const prev = this.history[this.history.length - 1];
    this.history = this.history.slice(0, -1);
    this.future = [cloneGroup(this.rootGroup), ...this.future.slice(0, 24)];
    this.rootGroup = cloneGroup(prev);
  }

  redo() {
    if (this.future.length === 0) return;
    const next = this.future[0];
    this.future = this.future.slice(1);
    this.history = [...this.history.slice(-24), cloneGroup(this.rootGroup)];
    this.rootGroup = cloneGroup(next);
  }

  addQuickFilter(field: FilterField, operator: FilterOperator, value: string, layer: FilterLayer = 'view') {
    this.pushHistory();
    const newRule: FilterRule = {
      id: `rule_${Date.now()}_${Math.random().toString(36).substring(2, 6)}`,
      field,
      operator,
      value: String(value),
      enabled: true,
      layer
    };
    this.rootGroup.rules = [...this.rootGroup.rules, newRule];

    // If layer includes engine and scan is running, forward to engine live filter
    if (layer === 'engine' || layer === 'both') {
      this.forwardToEngine(field, String(value));
    }
  }

  addRule(rulePartial: Partial<FilterRule>, parentGroupId: string = 'root') {
    this.pushHistory();
    const newRule: FilterRule = {
      id: rulePartial.id || `rule_${Date.now()}_${Math.random().toString(36).substring(2, 6)}`,
      field: rulePartial.field || 'status',
      operator: rulePartial.operator || '=',
      value: rulePartial.value ?? '',
      value2: rulePartial.value2,
      enabled: rulePartial.enabled ?? true,
      layer: rulePartial.layer || 'view'
    };

    if (parentGroupId === 'root') {
      this.rootGroup.rules = [...this.rootGroup.rules, newRule];
    } else {
      const g = this.findGroup(this.rootGroup, parentGroupId);
      if (g) {
        g.rules = [...g.rules, newRule];
      }
    }

    if (newRule.layer === 'engine' || newRule.layer === 'both') {
      this.forwardToEngine(newRule.field as FilterField, newRule.value);
    }
  }

  updateRule(ruleId: string, updates: Partial<FilterRule>) {
    this.pushHistory();
    const updateInGroup = (g: FilterGroup) => {
      g.rules = g.rules.map((r) => {
        if (r.id === ruleId) {
          const updated = { ...r, ...updates };
          if ((updated.layer === 'engine' || updated.layer === 'both') && updates.value) {
            this.forwardToEngine(updated.field as FilterField, updated.value);
          }
          return updated;
        }
        return r;
      });
      for (const sub of g.groups || []) {
        updateInGroup(sub);
      }
    };
    updateInGroup(this.rootGroup);
  }

  removeRule(ruleId: string) {
    this.pushHistory();
    const removeFromGroup = (g: FilterGroup) => {
      g.rules = g.rules.filter((r) => r.id !== ruleId);
      for (const sub of g.groups || []) {
        removeFromGroup(sub);
      }
    };
    removeFromGroup(this.rootGroup);
  }

  toggleRule(ruleId: string) {
    this.pushHistory();
    const toggleInGroup = (g: FilterGroup) => {
      g.rules = g.rules.map((r) => (r.id === ruleId ? { ...r, enabled: !r.enabled } : r));
      for (const sub of g.groups || []) {
        toggleInGroup(sub);
      }
    };
    toggleInGroup(this.rootGroup);
  }

  addGroup(conjunction: 'AND' | 'OR' = 'AND') {
    this.pushHistory();
    const newGroup: FilterGroup = {
      id: `group_${Date.now()}_${Math.random().toString(36).substring(2, 6)}`,
      conjunction,
      rules: [],
      groups: [],
      enabled: true
    };
    this.rootGroup.groups = [...(this.rootGroup.groups || []), newGroup];
  }

  removeGroup(groupId: string) {
    this.pushHistory();
    this.rootGroup.groups = (this.rootGroup.groups || []).filter((g) => g.id !== groupId);
  }

  clearFilters() {
    if (this.rootGroup.rules.length === 0 && (this.rootGroup.groups?.length || 0) === 0) return;
    this.pushHistory();
    this.rootGroup = createDefaultRoot();
  }

  reset() {
    this.rootGroup = createDefaultRoot();
    this.history = [];
    this.future = [];
  }

  private findGroup(parent: FilterGroup, id: string): FilterGroup | null {
    if (parent.id === id) return parent;
    for (const sub of parent.groups || []) {
      const found = this.findGroup(sub, id);
      if (found) return found;
    }
    return null;
  }

  private forwardToEngine(field: FilterField, value: string) {
    if (scanStore.state !== 'Running' && scanStore.state !== 'Paused') return;
    // Engine live filter supports status (fc), size (fs), words (fw), lines (fl)
    switch (field) {
      case 'status':
        scanStore.addFilter('status', value);
        break;
      case 'length':
        scanStore.addFilter('size', value);
        break;
      case 'words':
        scanStore.addFilter('words', value);
        break;
      case 'lines':
        scanStore.addFilter('lines', value);
        break;
    }
  }

  // In-Memory Evaluation for instant client-side filtering
  evaluateItem(item: ResultDTO): boolean {
    if (!this.rootGroup.enabled) return true;
    return this.evaluateGroup(this.rootGroup, item);
  }

  private evaluateGroup(g: FilterGroup, item: ResultDTO): boolean {
    if (!g.enabled) return true;
    const isOR = g.conjunction?.toUpperCase() === 'OR';

    let hasRules = false;

    for (const rule of g.rules) {
      if (!rule.enabled) continue;
      hasRules = true;
      const matched = this.evaluateRule(rule, item);
      if (isOR && matched) return true;
      if (!isOR && !matched) return false;
    }

    for (const sub of g.groups || []) {
      if (!sub.enabled) continue;
      hasRules = true;
      const matched = this.evaluateGroup(sub, item);
      if (isOR && matched) return true;
      if (!isOR && !matched) return false;
    }

    if (!hasRules) return true;
    return !isOR;
  }

  private evaluateRule(rule: FilterRule, item: ResultDTO): boolean {
    const field = rule.field?.toLowerCase();
    const op = rule.operator?.toLowerCase();
    const val = (rule.value || '').trim();

    // Numeric fields
    if (['status', 'length', 'words', 'lines', 'duration'].includes(field)) {
      let itemVal = 0;
      switch (field) {
        case 'status':
          itemVal = item.status;
          break;
        case 'length':
          itemVal = item.length;
          break;
        case 'words':
          itemVal = item.words;
          break;
        case 'lines':
          itemVal = item.lines;
          break;
        case 'duration':
          itemVal = item.duration_ms;
          break;
      }

      if (op === 'in') {
        const parts = val.split(',').map((p) => parseInt(p.trim(), 10)).filter((n) => !isNaN(n));
        return parts.includes(itemVal);
      }

      const numVal = parseInt(val, 10);
      if (isNaN(numVal)) return true;

      switch (op) {
        case '=':
          return itemVal === numVal;
        case '!=':
          return itemVal !== numVal;
        case '<':
          return itemVal < numVal;
        case '<=':
          return itemVal <= numVal;
        case '>':
          return itemVal > numVal;
        case '>=':
          return itemVal >= numVal;
        case 'between': {
          const numVal2 = parseInt((rule.value2 || '').trim(), 10);
          if (isNaN(numVal2)) return true;
          return itemVal >= numVal && itemVal <= numVal2;
        }
        default:
          return true;
      }
    }

    // Text fields
    let itemText = '';
    switch (field) {
      case 'url':
        itemText = item.url || '';
        break;
      case 'host':
        itemText = item.host || '';
        break;
      case 'payload':
        if (item.inputs) {
          itemText = item.inputs['FUZZ'] || Object.values(item.inputs)[0] || '';
        }
        break;
    }

    switch (op) {
      case '=':
        return itemText.toLowerCase() === val.toLowerCase();
      case '!=':
        return itemText.toLowerCase() !== val.toLowerCase();
      case 'contains':
        return itemText.toLowerCase().includes(val.toLowerCase());
      case 'not_contains':
        return !itemText.toLowerCase().includes(val.toLowerCase());
      case 'regex':
        if (!val || val.length > 256) return true;
        try {
          const re = new RegExp(val, 'i');
          return re.test(itemText);
        } catch {
          return true; // Don't crash on invalid regex during live typing
        }
      default:
        return true;
    }
  }

  // Presets API integration
  async loadPresets() {
    try {
      this.presets = await GetFilterPresets();
    } catch (err) {
      console.error('Failed to load filter presets:', err);
    }
  }

  async savePreset(name: string) {
    if (!name.trim()) return;
    try {
      await SaveFilterPreset(name.trim(), JSON.stringify(this.rootGroup));
      await this.loadPresets();
    } catch (err) {
      console.error('Failed to save filter preset:', err);
    }
  }

  applyPreset(preset: FilterPresetDTO) {
    try {
      const parsed = JSON.parse(preset.filter_json);
      this.pushHistory();
      this.rootGroup = parsed;
    } catch (err) {
      console.error('Failed to apply filter preset:', err);
    }
  }

  async deletePreset(name: string) {
    try {
      await DeleteFilterPreset(name);
      await this.loadPresets();
    } catch (err) {
      console.error('Failed to delete filter preset:', err);
    }
  }

  exportPresetsJSON(): string {
    return JSON.stringify(this.presets, null, 2);
  }

  async importPresetsJSON(jsonStr: string) {
    try {
      const parsed = JSON.parse(jsonStr);
      if (Array.isArray(parsed)) {
        for (const p of parsed) {
          if (p.name && p.filter_json) {
            await SaveFilterPreset(p.name, p.filter_json);
          }
        }
        await this.loadPresets();
      }
    } catch (err) {
      console.error('Failed to import presets:', err);
      throw err;
    }
  }
}

export const filtersStore = new FiltersStore();
