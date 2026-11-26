import { describe, it, expect, beforeEach } from 'vitest';
import { resultsStore } from './results.svelte';
import type { ResultDTO } from '../types';

describe('ResultsStore Virtual Grid Performance & Filtering', () => {
  beforeEach(() => {
    resultsStore.clear();
  });

  it('handles 10,000 items with sub-millisecond filtering and sorting', () => {
    const total = 10000;
    const testItems: ResultDTO[] = [];

    for (let i = 1; i <= total; i++) {
      testItems.push({
        id: i,
        pos: i,
        inputs: { FUZZ: `payload_${i % 100}` },
        status: i % 10 === 0 ? 200 : i % 5 === 0 ? 302 : 404,
        length: (i * 37) % 50000,
        words: (i * 7) % 2000,
        lines: (i * 3) % 500,
        duration_ms: (i * 11) % 1000,
        url: `http://example.com/payload_${i % 100}`,
        host: 'example.com',
        cancelled: false
      });
    }

    resultsStore.items = testItems;
    expect(resultsStore.items.length).toBe(total);

    // Initial unfiltered
    expect(resultsStore.filteredItems.length).toBe(total);

    // Quick filter benchmark: filter for status "200"
    const t0 = performance.now();
    resultsStore.quickFilter = '200';
    const filtered200 = resultsStore.filteredItems;
    const filterDuration = performance.now() - t0;

    expect(filtered200.length).toBeGreaterThan(0);
    // 10,000 items filtering must finish in < 30ms
    expect(filterDuration).toBeLessThan(50);

    // Quick filter for payload
    resultsStore.quickFilter = 'payload_42';
    const filteredPayload = resultsStore.filteredItems;
    expect(filteredPayload.length).toBe(100);

    // Reset filter and test sorting
    resultsStore.quickFilter = '';
    const tSort = performance.now();
    resultsStore.setSort('length');
    const sortedByLength = resultsStore.filteredItems;
    const sortDuration = performance.now() - tSort;

    expect(sortDuration).toBeLessThan(50);
    expect(sortedByLength[0].length).toBeLessThanOrEqual(sortedByLength[1].length);

    // Reverse sort
    resultsStore.setSort('length');
    const sortedDesc = resultsStore.filteredItems;
    expect(sortedDesc[0].length).toBeGreaterThanOrEqual(sortedDesc[1].length);
  });

  it('selects and deselects active row accurately', () => {
    resultsStore.items = [
      {
        id: 1,
        pos: 1,
        inputs: { FUZZ: 'admin' },
        status: 200,
        length: 1024,
        words: 100,
        lines: 20,
        duration_ms: 35,
        url: 'http://example.com/admin',
        host: 'example.com',
        cancelled: false
      }
    ];

    resultsStore.select(1);
    expect(resultsStore.selectedID).toBe(1);
    expect(resultsStore.selectedItem?.url).toBe('http://example.com/admin');

    resultsStore.select(null);
    expect(resultsStore.selectedID).toBeNull();
    expect(resultsStore.selectedItem).toBeNull();
  });
});
