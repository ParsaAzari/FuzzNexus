import { describe, it, expect, beforeEach, vi } from 'vitest';
import { wordlistsStore } from './wordlists.svelte';
import type { WordlistDTO } from '../types';

vi.mock('../../../wailsjs/go/main/App', () => ({
  ListWordlists: vi.fn().mockResolvedValue([]),
  CreateWordlist: vi.fn(),
  CreateWordlistWithCategory: vi.fn(),
  GetWordlistCategories: vi.fn().mockResolvedValue([]),
  CreateWordlistCategory: vi.fn(),
  UpdateWordlistCategory: vi.fn(),
  DeleteWordlistCategory: vi.fn(),
  UpdateWordlist: vi.fn(),
  DeleteWordlist: vi.fn(),
  DeleteWordlistsBatch: vi.fn().mockResolvedValue(2),
  ReadWordlistPage: vi.fn().mockResolvedValue({ lines: [], total_lines: 0, offset: 0, limit: 100 }),
  SaveWordlistContent: vi.fn(),
  ImportWordlistFromPath: vi.fn(),
  LinkWordlistPath: vi.fn(),
  GetWordlistStats: vi.fn(),
  ExportWordlist: vi.fn(),
  SelectWordlistDialog: vi.fn(),
  VerifyWordlist: vi.fn().mockResolvedValue({
    id: 1,
    total_lines: 100,
    blank_lines_count: 5,
    has_whitespace: true,
    has_crlf: false,
    is_clean: false
  }),
  CleanWordlist: vi.fn().mockResolvedValue({
    id: 1,
    lines_before: 100,
    lines_after: 95,
    empty_lines_removed: 5,
    lines_trimmed: 10
  }),
  MergeWordlistsToSandbox: vi.fn().mockImplementation((ids, opts, isEphemeral, name, cat) =>
    Promise.resolve({
      id: isEphemeral ? 0 : 99,
      name,
      path: isEphemeral ? '/sandboxes/sb_wordlist_123.tmp' : `/wordlists/${name}`,
      line_count: 1400,
      size_bytes: 12000,
      is_linked: false,
      category: isEphemeral ? 'sandbox' : cat,
      created_at: '',
      updated_at: ''
    })
  ),
  PreviewMergeStats: vi.fn().mockResolvedValue({
    total_source_lines: 1800,
    deduplicated_lines: 1400,
    duplicates_removed: 300,
    empty_lines_removed: 100,
    comments_removed: 0,
    lines_trimmed: 50,
    size_bytes: 12000
  }),
  CleanupSandboxWordlist: vi.fn().mockResolvedValue(undefined),
  ResetBuiltinWordlist: vi.fn().mockImplementation((id: number) =>
    Promise.resolve({
      id,
      name: 'Admin & Portals',
      path: '/app/wordlists/builtin_admin.txt',
      line_count: 20096,
      size_bytes: 350000,
      is_linked: false,
      category: 'admin',
      is_builtin: true,
      builtin_id: 'admin_portals',
      created_at: '',
      updated_at: ''
    })
  ),
  RestoreAllBuiltinWordlists: vi.fn().mockResolvedValue([
    {
      id: 10,
      name: 'Admin & Portals',
      path: '/app/wordlists/builtin_admin.txt',
      line_count: 20096,
      size_bytes: 350000,
      is_linked: false,
      category: 'admin',
      is_builtin: true,
      builtin_id: 'admin_portals',
      created_at: '',
      updated_at: ''
    }
  ])
}));

describe('WordlistsStore Multi-selection & Batch Actions', () => {
  const sampleWordlists: WordlistDTO[] = [
    { id: 1, name: 'common.txt', path: '/app/common.txt', line_count: 1000, size_bytes: 8192, is_linked: false, category: 'discovery', created_at: '', updated_at: '' },
    { id: 2, name: 'routes.txt', path: '/app/routes.txt', line_count: 500, size_bytes: 4096, is_linked: false, category: 'discovery', created_at: '', updated_at: '' },
    { id: 3, name: 'api_endpoints.txt', path: '/ext/api.txt', line_count: 300, size_bytes: 2048, is_linked: true, category: 'api', created_at: '', updated_at: '' }
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

  it('verifies wordlist health and automatically updates state on cleanWordlist', async () => {
    await wordlistsStore.loadHealth(1);
    expect(wordlistsStore.selectedHealth).not.toBeNull();
    expect(wordlistsStore.selectedHealth?.is_clean).toBe(false);
    expect(wordlistsStore.selectedHealth?.blank_lines_count).toBe(5);

    const cleanRes = await wordlistsStore.cleanWordlist(1);
    expect(cleanRes.empty_lines_removed).toBe(5);
    expect(cleanRes.lines_trimmed).toBe(10);
  });

  it('tracks selections across categories with getCheckedCountByCategory', () => {
    wordlistsStore.toggleCheck(1); // discovery
    wordlistsStore.toggleCheck(3); // api

    expect(wordlistsStore.checkedCount).toBe(2);
    expect(wordlistsStore.getCheckedCountByCategory('discovery')).toBe(1);
    expect(wordlistsStore.getCheckedCountByCategory('api')).toBe(1);
    expect(wordlistsStore.getCheckedCountByCategory('ai')).toBe(0);
    expect(wordlistsStore.getCheckedCountByCategory('fuzzing')).toBe(0);
    expect(wordlistsStore.getCheckedCountByCategory('all')).toBe(2);
  });

  it('performs previewMerge and mergeToSandbox correctly', async () => {
    const opts = {
      trim_whitespace: true,
      strip_bom: true,
      remove_empty: true,
      strip_comments: true,
      strip_leading_slash: true,
      strip_traversal_payloads: false,
      case_insensitive: false,
      max_line_len: 2048
    };

    const preview = await wordlistsStore.previewMerge([1, 2], opts);
    expect(preview.deduplicated_lines).toBe(1400);
    expect(preview.duplicates_removed).toBe(300);

    // Ephemeral Sandbox merge
    const sb = await wordlistsStore.mergeToSandbox([1, 2], opts, true, 'sb_test.txt', 'sandbox');
    expect(sb.id).toBe(0);
    expect(sb.category).toBe('sandbox');
    expect(sb.path).toContain('sandboxes');

    // Persistent merge
    const pers = await wordlistsStore.mergeToSandbox([1, 2], opts, false, 'pers_test.txt', 'fuzzing');
    expect(pers.id).toBe(99);
    expect(pers.category).toBe('fuzzing');
  });

  it('correctly resets a built-in wordlist and restores all built-in packs', async () => {
    const reset = await wordlistsStore.resetBuiltinWordlist(10);
    expect(reset.is_builtin).toBe(true);
    expect(reset.builtin_id).toBe('admin_portals');
    expect(reset.line_count).toBe(20096);

    const allRestored = await wordlistsStore.restoreAllBuiltinWordlists();
    expect(allRestored.length).toBe(1);
    expect(allRestored[0].builtin_id).toBe('admin_portals');
  });
});
