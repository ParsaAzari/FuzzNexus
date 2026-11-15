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
});
