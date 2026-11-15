import { describe, it, expect, beforeEach, vi } from 'vitest';
import { wordlistsStore } from './wordlists.svelte';
import type { WordlistDTO } from '../types';

vi.mock('../../../wailsjs/go/main/App', () => ({
  ListWordlists: vi.fn().mockResolvedValue([]),
  CreateWordlist: vi.fn(),
  UpdateWordlist: vi.fn(),
  DeleteWordlist: vi.fn(),
  DeleteWordlistsBatch: vi.fn().mockResolvedValue(2),
  ReadWordlistPage: vi.fn().mockResolvedValue({ lines: [], total_lines: 0, offset: 0, limit: 100 }),
  SaveWordlistContent: vi.fn(),
  ImportWordlistFromPath: vi.fn(),
  LinkWordlistPath: vi.fn(),
  GetWordlistStats: vi.fn(),
  ExportWordlist: vi.fn(),
  SelectWordlistDialog: vi.fn()
}));

describe('WordlistsStore Multi-selection & Batch Actions', () => {
  const sampleWordlists: WordlistDTO[] = [
    { id: 1, name: 'common.txt', path: '/app/common.txt', line_count: 1000, size_bytes: 8192, is_linked: false, created_at: '', updated_at: '' },
    { id: 2, name: 'routes.txt', path: '/app/routes.txt', line_count: 500, size_bytes: 4096, is_linked: false, created_at: '', updated_at: '' },
    { id: 3, name: 'api_endpoints.txt', path: '/ext/api.txt', line_count: 300, size_bytes: 2048, is_linked: true, created_at: '', updated_at: '' }
  ] as unknown as WordlistDTO[];

  beforeEach(() => {
    wordlistsStore.items = [...sampleWordlists];
    wordlistsStore.searchFilter = '';
    wordlistsStore.uncheckAll();
    wordlistsStore.selectedID = 1;
  });

  it('correctly toggles individual item selection and computes checkedCount', () => {
    expect(wordlistsStore.checkedCount).toBe(0);
    expect(wordlistsStore.isAllChecked).toBe(false);
    expect(wordlistsStore.isSomeChecked).toBe(false);

    wordlistsStore.toggleCheck(1);
    expect(wordlistsStore.checkedCount).toBe(1);
    expect(wordlistsStore.checkedIDs).toContain(1);
    expect(wordlistsStore.isSomeChecked).toBe(true);
    expect(wordlistsStore.isAllChecked).toBe(false);

    wordlistsStore.toggleCheck(2);
    expect(wordlistsStore.checkedCount).toBe(2);
    expect(wordlistsStore.isSomeChecked).toBe(true);
    expect(wordlistsStore.isAllChecked).toBe(false);

    // Toggle off item 1
    wordlistsStore.toggleCheck(1);
    expect(wordlistsStore.checkedCount).toBe(1);
    expect(wordlistsStore.checkedIDs).not.toContain(1);
    expect(wordlistsStore.checkedIDs).toContain(2);
  });

  it('handles checkAll, uncheckAll, and toggleCheckAll properly', () => {
    wordlistsStore.checkAll();
    expect(wordlistsStore.checkedCount).toBe(3);
    expect(wordlistsStore.isAllChecked).toBe(true);
    expect(wordlistsStore.isSomeChecked).toBe(false);

    wordlistsStore.toggleCheckAll();
    expect(wordlistsStore.checkedCount).toBe(0);
    expect(wordlistsStore.isAllChecked).toBe(false);
    expect(wordlistsStore.isSomeChecked).toBe(false);

    wordlistsStore.toggleCheck(1);
    expect(wordlistsStore.isSomeChecked).toBe(true);
    // When some are checked, toggleCheckAll should select all
    wordlistsStore.toggleCheckAll();
    expect(wordlistsStore.checkedCount).toBe(3);
    expect(wordlistsStore.isAllChecked).toBe(true);
  });

  it('filters wordlists and limits checkAll to filtered subset', () => {
    wordlistsStore.searchFilter = 'api';
    expect(wordlistsStore.filteredWordlists.length).toBe(1);
    expect(wordlistsStore.filteredWordlists[0].id).toBe(3);

    wordlistsStore.checkAll();
    expect(wordlistsStore.checkedIDs).toEqual([3]);
    expect(wordlistsStore.isAllChecked).toBe(true);
  });

  it('handles deleteBatch and cleans up checkedIDs and selectedID', async () => {
    const { ListWordlists } = await import('../../../wailsjs/go/main/App');
    vi.mocked(ListWordlists).mockResolvedValueOnce([sampleWordlists[2]]);

    wordlistsStore.checkAll();
    expect(wordlistsStore.checkedCount).toBe(3);

    const deleted = await wordlistsStore.deleteBatch([1, 2]);
    expect(deleted).toBe(2);

    // 1 and 2 should be removed from checkedIDs, and 3 should remain
    expect(wordlistsStore.checkedIDs).toEqual([3]);

    // Since selectedID was 1 (deleted), it automatically selects the first remaining wordlist (3)
    expect(wordlistsStore.selectedID).toBe(3);
  });
});
