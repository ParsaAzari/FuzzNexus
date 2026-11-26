import { describe, it, expect, beforeEach } from 'vitest';
import { filtersStore } from './filters.svelte';
import type { ResultDTO } from '../types';

describe('FiltersStore', () => {
  beforeEach(() => {
    filtersStore.reset();
  });

  const sampleResults: ResultDTO[] = [
    {
      id: 1,
      pos: 1,
      status: 200,
      length: 1250,
      words: 150,
      lines: 25,
      duration_ms: 120,
      url: 'http://localhost/admin/login',
      host: 'localhost',
      inputs: { FUZZ: 'admin/login' },
      redirect_location: '',
      content_type: 'text/html',
      hash: '',
      cancelled: false
    },
    {
      id: 2,
      pos: 2,
      status: 404,
      length: 220,
      words: 15,
      lines: 4,
      duration_ms: 45,
      url: 'http://localhost/not-found',
      host: 'localhost',
      inputs: { FUZZ: 'not-found' },
      redirect_location: '',
      content_type: 'text/html',
      hash: '',
      cancelled: false
    },
    {
      id: 3,
      pos: 3,
      status: 301,
      length: 0,
      words: 0,
      lines: 0,
      duration_ms: 80,
      url: 'http://localhost/redirect',
      host: 'localhost',
      inputs: { FUZZ: 'redirect' },
      redirect_location: '/redirected',
      content_type: '',
      hash: '',
      cancelled: false
    }
  ];

  it('filters results with numeric operators correctly', () => {
    // Status != 404
    filtersStore.addQuickFilter('status', '!=', '404');
    expect(filtersStore.evaluateItem(sampleResults[0])).toBe(true);
    expect(filtersStore.evaluateItem(sampleResults[1])).toBe(false);
    expect(filtersStore.evaluateItem(sampleResults[2])).toBe(true);

    // Add length > 500 (AND conjunction)
    filtersStore.addQuickFilter('length', '>', '500');
    expect(filtersStore.evaluateItem(sampleResults[0])).toBe(true);
    expect(filtersStore.evaluateItem(sampleResults[1])).toBe(false);
    expect(filtersStore.evaluateItem(sampleResults[2])).toBe(false);
  });

  it('filters results with string operators (contains, regex)', () => {
    filtersStore.addQuickFilter('url', 'contains', 'admin');
    expect(filtersStore.evaluateItem(sampleResults[0])).toBe(true);
    expect(filtersStore.evaluateItem(sampleResults[1])).toBe(false);

    filtersStore.clearFilters();
    filtersStore.addQuickFilter('payload', 'regex', '^admin/.*');
    expect(filtersStore.evaluateItem(sampleResults[0])).toBe(true);
    expect(filtersStore.evaluateItem(sampleResults[1])).toBe(false);
  });

  it('supports undo and redo stack up to 20+ operations', () => {
    expect(filtersStore.canUndo).toBe(false);

    // Push 5 filters
    for (let i = 1; i <= 5; i++) {
      filtersStore.addQuickFilter('status', '=', String(200 + i));
    }

    expect(filtersStore.activeRules.length).toBe(5);
    expect(filtersStore.canUndo).toBe(true);

    // Undo 3 times
    filtersStore.undo();
    filtersStore.undo();
    filtersStore.undo();

    expect(filtersStore.activeRules.length).toBe(2);
    expect(filtersStore.canRedo).toBe(true);

    // Redo 2 times
    filtersStore.redo();
    filtersStore.redo();

    expect(filtersStore.activeRules.length).toBe(4);
  });

  it('protects against ReDoS and invalid regex patterns without crashing', () => {
    // Oversized regex (> 256 chars) is safely ignored
    const longPattern = 'a'.repeat(300);
    filtersStore.addQuickFilter('url', 'regex', longPattern);
    expect(() => filtersStore.evaluateItem(sampleResults[0])).not.toThrow();

    // Invalid regex syntax is caught safely
    filtersStore.clearFilters();
    filtersStore.addQuickFilter('url', 'regex', '[a-z(');
    expect(() => filtersStore.evaluateItem(sampleResults[0])).not.toThrow();
  });

  it('correctly applies a preset group to the store and updates active rules', () => {
    const presetDTO = {
      name: 'Hide 404 (Not Found)',
      filter_json: JSON.stringify({
        id: 'root',
        conjunction: 'AND',
        enabled: true,
        rules: [
          { id: 'def-hide-404', field: 'status', operator: '!=', value: '404', enabled: true, layer: 'view' }
        ],
        groups: []
      }),
      created_at: '2026-11-15T00:00:00Z'
    };

    filtersStore.applyPreset(presetDTO);
    expect(filtersStore.activeRules.length).toBe(1);
    expect(filtersStore.activeRules[0].field).toBe('status');
    expect(filtersStore.activeRules[0].operator).toBe('!=');
    expect(filtersStore.activeRules[0].value).toBe('404');
    expect(filtersStore.evaluateItem(sampleResults[0])).toBe(true); // 200 OK
    expect(filtersStore.evaluateItem(sampleResults[1])).toBe(false); // 404 Not Found
  });

  it('prevents adding duplicate quick filters and warns user', () => {
    filtersStore.clearFilters();
    expect(filtersStore.hasFilter('status', '=', '200')).toBe(false);

    const firstAdd = filtersStore.addQuickFilter('status', '=', '200');
    expect(firstAdd).toBe(true);
    expect(filtersStore.activeRules.length).toBe(1);
    expect(filtersStore.hasFilter('status', '=', '200')).toBe(true);

    // Second add with identical parameters must be rejected
    const secondAdd = filtersStore.addQuickFilter('status', '=', '200');
    expect(secondAdd).toBe(false);
    expect(filtersStore.activeRules.length).toBe(1);
  });

  it('prevents adding conflicting quick filters (e.g. status = 200 and status != 200)', () => {
    filtersStore.clearFilters();
    expect(filtersStore.addQuickFilter('status', '=', '200')).toBe(true);
    expect(filtersStore.activeRules.length).toBe(1);

    // Conflicting filter (!= 200) must be rejected
    expect(filtersStore.hasConflictingFilter('status', '!=', '200')).toBe('status = 200');
    const conflictAdd = filtersStore.addQuickFilter('status', '!=', '200');
    expect(conflictAdd).toBe(false);
    expect(filtersStore.activeRules.length).toBe(1);
  });

  it('enforces maximum quick filter rules limit (MAX_QUICK_FILTERS = 20)', () => {
    filtersStore.clearFilters();
    for (let i = 1; i <= 20; i++) {
      expect(filtersStore.addQuickFilter('status', '=', String(200 + i))).toBe(true);
    }
    expect(filtersStore.activeRules.length).toBe(20);

    // 21st rule must be rejected due to limit
    const overLimit = filtersStore.addQuickFilter('status', '=', '999');
    expect(overLimit).toBe(false);
    expect(filtersStore.activeRules.length).toBe(20);
  });

  it('toggles quick filters on and off (checking and unchecking)', () => {
    filtersStore.clearFilters();
    expect(filtersStore.hasFilter('status', '=', '200')).toBe(false);

    // Toggle ON
    const toggledOn = filtersStore.toggleQuickFilter('status', '=', '200');
    expect(toggledOn).toBe(true);
    expect(filtersStore.hasFilter('status', '=', '200')).toBe(true);
    expect(filtersStore.activeRules.length).toBe(1);

    // Toggle OFF (uncheck)
    const toggledOff = filtersStore.toggleQuickFilter('status', '=', '200');
    expect(toggledOff).toBe(false);
    expect(filtersStore.hasFilter('status', '=', '200')).toBe(false);
    expect(filtersStore.activeRules.length).toBe(0);

    // Toggle ON again
    const toggledOnAgain = filtersStore.toggleQuickFilter('status', '=', '200');
    expect(toggledOnAgain).toBe(true);
    expect(filtersStore.hasFilter('status', '=', '200')).toBe(true);
    expect(filtersStore.activeRules.length).toBe(1);
  });

  it('seamlessly switches between opposing filters (Only Status vs Hide Status, Only Size vs Hide Size)', () => {
    filtersStore.clearFilters();

    // 1. Activate "Only Status 200"
    expect(filtersStore.toggleQuickFilter('status', '=', '200')).toBe(true);
    expect(filtersStore.hasFilter('status', '=', '200')).toBe(true);
    expect(filtersStore.hasFilter('status', '!=', '200')).toBe(false);
    expect(filtersStore.activeRules.length).toBe(1);

    // 2. Seamlessly switch to "Hide Status 200" without manual deactivation
    expect(filtersStore.toggleQuickFilter('status', '!=', '200')).toBe(true);
    expect(filtersStore.hasFilter('status', '=', '200')).toBe(false);
    expect(filtersStore.hasFilter('status', '!=', '200')).toBe(true);
    expect(filtersStore.activeRules.length).toBe(1);

    // 3. Seamlessly switch back to "Only Status 200"
    expect(filtersStore.toggleQuickFilter('status', '=', '200')).toBe(true);
    expect(filtersStore.hasFilter('status', '=', '200')).toBe(true);
    expect(filtersStore.hasFilter('status', '!=', '200')).toBe(false);
    expect(filtersStore.activeRules.length).toBe(1);

    // 4. Activate "Only Size 466009"
    expect(filtersStore.toggleQuickFilter('length', '=', '466009')).toBe(true);
    expect(filtersStore.hasFilter('length', '=', '466009')).toBe(true);
    expect(filtersStore.hasFilter('length', '!=', '466009')).toBe(false);
    expect(filtersStore.activeRules.length).toBe(2);

    // 5. Seamlessly switch to "Hide Size 466009"
    expect(filtersStore.toggleQuickFilter('length', '!=', '466009')).toBe(true);
    expect(filtersStore.hasFilter('length', '=', '466009')).toBe(false);
    expect(filtersStore.hasFilter('length', '!=', '466009')).toBe(true);
    expect(filtersStore.activeRules.length).toBe(2);

    // 6. Clicking active "Hide Size 466009" toggles it OFF
    expect(filtersStore.toggleQuickFilter('length', '!=', '466009')).toBe(false);
    expect(filtersStore.hasFilter('length', '!=', '466009')).toBe(false);
    expect(filtersStore.activeRules.length).toBe(1); // Only status = 200 remains
  });

  it('allows switching opposing filter even when at maximum filter limit', () => {
    filtersStore.clearFilters();

    // Fill to 20 filters, with status = 200 as one of them
    expect(filtersStore.addQuickFilter('status', '=', '200')).toBe(true);
    for (let i = 1; i < 20; i++) {
      expect(filtersStore.addQuickFilter('status', '=', String(300 + i))).toBe(true);
    }
    expect(filtersStore.activeRules.length).toBe(20);

    // Attempting to add an unrelated 21st rule is blocked
    expect(filtersStore.addQuickFilter('status', '=', '999')).toBe(false);

    // Switching opposing filter (from status = 200 to status != 200) succeeds without exceeding limit
    expect(filtersStore.toggleQuickFilter('status', '!=', '200')).toBe(true);
    expect(filtersStore.hasFilter('status', '=', '200')).toBe(false);
    expect(filtersStore.hasFilter('status', '!=', '200')).toBe(true);
    expect(filtersStore.activeRules.length).toBe(20);
  });
});
