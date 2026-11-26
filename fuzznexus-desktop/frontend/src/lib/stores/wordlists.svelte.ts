import {
  ListWordlists,
  CreateWordlist,
  CreateWordlistWithCategory,
  GetWordlistCategories,
  CreateWordlistCategory,
  UpdateWordlistCategory,
  DeleteWordlistCategory,
  UpdateWordlist,
  UpdateWordlistMetadata,
  MergeWordlists,
  DeleteWordlist,
  DeleteWordlistsBatch,
  ReadWordlistPage,
  SaveWordlistContent,
  ImportWordlistFromPath,
  LinkWordlistPath,
  GetWordlistStats,
  ExportWordlist,
  SelectWordlistDialog,
  SaveWordlistDialog,
  VerifyWordlist,
  CleanWordlist,
  VerifyWordlistPath,
  CleanWordlistPath,
  MergeWordlistsToSandbox,
  PreviewMergeStats,
  CleanupSandboxWordlist,
  ResetBuiltinWordlist,
  RestoreAllBuiltinWordlists
} from '../../../wailsjs/go/main/App';
import type { WordlistDTO, CategoryDTO, WordlistPageDTO, WordlistStatsDTO, CleanResultDTO, WordlistHealthDTO, SanitizeOptions, MergeReportDTO } from '../types';

class WordlistsStore {
  items = $state<WordlistDTO[]>([]);
  categories = $state<CategoryDTO[]>([]);
  selectedID = $state<number | null>(null);
  checkedIDs = $state<number[]>([]);
  selectedStats = $state<WordlistStatsDTO | null>(null);
  selectedHealth = $state<WordlistHealthDTO | null>(null);
  currentPage = $state<WordlistPageDTO>({ lines: [], total_lines: 0, offset: 0, limit: 100 });
  isLoading = $state<boolean>(false);
  isSaving = $state<boolean>(false);
  searchFilter = $state<string>('');
  selectedCategory = $state<string>('all');
  error = $state<string>('');

  selectedWordlist = $derived(
    this.selectedID === null ? null : (this.items.find((w) => w.id === this.selectedID) ?? null)
  );

  filteredWordlists = $derived.by(() => {
    let list = this.items;
    if (this.selectedCategory !== 'all') {
      list = list.filter((w) => (w.category || 'general').toLowerCase() === this.selectedCategory.toLowerCase());
    }
    const q = this.searchFilter.trim().toLowerCase();
    if (!q) return list;
    return list.filter((w) => w.name.toLowerCase().includes(q) || w.path.toLowerCase().includes(q));
  });

  isAllChecked = $derived(
    this.filteredWordlists.length > 0 &&
      this.filteredWordlists.every((w) => this.checkedIDs.includes(w.id))
  );

  isSomeChecked = $derived(
    this.checkedIDs.length > 0 && !this.isAllChecked
  );

  checkedCount = $derived(this.checkedIDs.length);

  getCategoryCount(category: string): number {
    if (category === 'all') return this.items.length;
    return this.items.filter((w) => (w.category || 'general').toLowerCase() === category.toLowerCase()).length;
  }

  getCheckedCountByCategory(category: string): number {
    if (category === 'all') return this.checkedIDs.length;
    return this.items.filter(
      (w) => (w.category || 'general').toLowerCase() === category.toLowerCase() && this.checkedIDs.includes(w.id)
    ).length;
  }

  toggleCheck(id: number) {
    if (this.checkedIDs.includes(id)) {
      this.checkedIDs = this.checkedIDs.filter((i) => i !== id);
    } else {
      this.checkedIDs = [...this.checkedIDs, id];
    }
  }

  checkAll() {
    const visibleIDs = this.filteredWordlists.map((w) => w.id);
    const set = new Set([...this.checkedIDs, ...visibleIDs]);
    this.checkedIDs = Array.from(set);
  }

  uncheckAllVisible() {
    const visibleSet = new Set(this.filteredWordlists.map((w) => w.id));
    this.checkedIDs = this.checkedIDs.filter((id) => !visibleSet.has(id));
  }

  uncheckAll() {
    this.checkedIDs = [];
  }

  toggleCheckAll() {
    if (this.isAllChecked) {
      this.uncheckAllVisible();
    } else {
      this.checkAll();
    }
  }

  async loadCategories() {
    try {
      this.categories = (await GetWordlistCategories()) || [];
    } catch (err: any) {
      console.error('Failed to load wordlist categories:', err);
    }
  }

  async createCategory(name: string, color: string = '#64748b'): Promise<CategoryDTO> {
    try {
      const created = await CreateWordlistCategory('', name, color);
      await this.loadCategories();
      return created;
    } catch (err: any) {
      this.error = `Failed to create category: ${err?.message || err}`;
      throw err;
    }
  }

  async updateCategory(id: string, name: string, color: string) {
    try {
      await UpdateWordlistCategory(id, name, color);
      await this.loadCategories();
    } catch (err: any) {
      this.error = `Failed to update category: ${err?.message || err}`;
      throw err;
    }
  }

  async deleteCategory(id: string) {
    try {
      await DeleteWordlistCategory(id);
      if (this.selectedCategory === id) {
        this.selectedCategory = 'all';
      }
      await Promise.all([this.loadCategories(), this.loadWordlists()]);
    } catch (err: any) {
      this.error = `Failed to delete category: ${err?.message || err}`;
      throw err;
    }
  }

  async loadWordlists() {
    this.isLoading = true;
    this.error = '';
    try {
      const [wls, cats] = await Promise.all([
        ListWordlists(),
        GetWordlistCategories().catch(() => [])
      ]);
      this.items = wls || [];
      if (cats && cats.length > 0) {
        this.categories = cats;
      }
      // Clean up any checked IDs that are no longer in items
      this.checkedIDs = this.checkedIDs.filter((id) => this.items.some((w) => w.id === id));
      if (this.items.length > 0 && this.selectedID === null) {
        this.selectWordlist(this.items[0].id);
      }
    } catch (err: any) {
      this.error = `Failed to load wordlists: ${err?.message || err}`;
    } finally {
      this.isLoading = false;
    }
  }

  async selectWordlist(id: number) {
    this.selectedID = id;
    this.selectedStats = null;
    this.selectedHealth = null;
    await Promise.all([this.loadPage(0, 100), this.loadStats(id), this.loadHealth(id)]);
  }

  async loadPage(offset: number = 0, limit: number = 100) {
    if (this.selectedID === null) return;
    try {
      this.currentPage = await ReadWordlistPage(this.selectedID, offset, limit);
    } catch (err: any) {
      this.error = `Failed to read lines: ${err?.message || err}`;
    }
  }

  async loadStats(id: number) {
    try {
      this.selectedStats = await GetWordlistStats(id);
    } catch {
      // Non-fatal if stats calculation fails
      this.selectedStats = null;
    }
  }

  async loadHealth(id: number) {
    try {
      this.selectedHealth = await VerifyWordlist(id);
    } catch {
      // Non-fatal if health check fails
      this.selectedHealth = null;
    }
  }

  async cleanWordlist(id: number): Promise<CleanResultDTO> {
    this.isSaving = true;
    this.error = '';
    try {
      const res = await CleanWordlist(id);
      await Promise.all([
        this.loadWordlists(),
        this.loadPage(this.currentPage.offset, this.currentPage.limit),
        this.loadStats(id),
        this.loadHealth(id)
      ]);
      return res;
    } catch (err: any) {
      this.error = `Failed to clean wordlist: ${err?.message || err}`;
      throw err;
    } finally {
      this.isSaving = false;
    }
  }

  async verifyPath(filePath: string): Promise<WordlistHealthDTO | null> {
    try {
      return await VerifyWordlistPath(filePath);
    } catch {
      return null;
    }
  }

  async cleanPath(filePath: string): Promise<CleanResultDTO> {
    try {
      return await CleanWordlistPath(filePath);
    } catch (err: any) {
      this.error = `Failed to clean path: ${err?.message || err}`;
      throw err;
    }
  }

  async createWordlist(name: string, content: string = '', category: string = 'general') {
    this.isSaving = true;
    this.error = '';
    try {
      const created = await CreateWordlistWithCategory(name, content, category);
      await this.loadWordlists();
      if (created) {
        this.selectWordlist(created.id);
      }
      return created;
    } catch (err: any) {
      this.error = `Failed to create wordlist: ${err?.message || err}`;
      throw err;
    } finally {
      this.isSaving = false;
    }
  }

  async importFromPath(filePath: string, customName?: string, category?: string): Promise<WordlistDTO | null> {
    this.isLoading = true;
    this.error = '';
    try {
      const imported = await ImportWordlistFromPath(filePath);
      if (imported && (customName || category)) {
        const finalName = customName?.trim() || imported.name;
        const finalCategory = category || 'general';
        await UpdateWordlistMetadata(imported.id, finalName, finalCategory);
      }
      await this.loadWordlists();
      if (imported) {
        this.selectWordlist(imported.id);
      }
      return imported;
    } catch (err: any) {
      this.error = `Failed to import wordlist: ${err?.message || err}`;
      throw err;
    } finally {
      this.isLoading = false;
    }
  }

  async linkFromPath(filePath: string, customName?: string, category?: string): Promise<WordlistDTO | null> {
    this.isLoading = true;
    this.error = '';
    try {
      const linked = await LinkWordlistPath(filePath);
      if (linked && (customName || category)) {
        const finalName = customName?.trim() || linked.name;
        const finalCategory = category || 'general';
        await UpdateWordlistMetadata(linked.id, finalName, finalCategory);
      }
      await this.loadWordlists();
      if (linked) {
        this.selectWordlist(linked.id);
      }
      return linked;
    } catch (err: any) {
      this.error = `Failed to link wordlist: ${err?.message || err}`;
      throw err;
    } finally {
      this.isLoading = false;
    }
  }

  async openNativeFilePicker(mode: 'import' | 'link'): Promise<string | null> {
    try {
      const selectedPath = await SelectWordlistDialog(
        mode === 'import' ? 'Select Wordlist to Import (Copy)' : 'Select Wordlist to Link (In-Place)'
      );
      if (!selectedPath) return null;
      if (mode === 'import') {
        await this.importFromPath(selectedPath);
      } else {
        await this.linkFromPath(selectedPath);
      }
      return selectedPath;
    } catch (err: any) {
      this.error = `File selection failed: ${err?.message || err}`;
      return null;
    }
  }

  async updateName(id: number, newName: string) {
    try {
      await UpdateWordlist(id, newName);
      await this.loadWordlists();
    } catch (err: any) {
      this.error = `Failed to rename: ${err?.message || err}`;
    }
  }

  async updateMetadata(id: number, newName: string, category: string) {
    try {
      await UpdateWordlistMetadata(id, newName, category);
      await this.loadWordlists();
    } catch (err: any) {
      this.error = `Failed to update wordlist metadata: ${err?.message || err}`;
      throw err;
    }
  }

  async mergeWordlists(ids: number[], targetName: string, category: string = 'general'): Promise<WordlistDTO> {
    this.isSaving = true;
    this.error = '';
    try {
      const merged = await MergeWordlists(ids, targetName, category);
      await this.loadWordlists();
      if (merged) {
        this.selectWordlist(merged.id);
      }
      return merged;
    } catch (err: any) {
      this.error = `Failed to merge wordlists: ${err?.message || err}`;
      throw err;
    } finally {
      this.isSaving = false;
    }
  }

  async mergeToSandbox(
    ids: number[],
    opts: SanitizeOptions,
    isEphemeral: boolean = true,
    targetName: string = 'merged_wordlist.txt',
    category: string = 'general'
  ): Promise<WordlistDTO> {
    this.isSaving = true;
    this.error = '';
    try {
      const merged = await MergeWordlistsToSandbox(ids, opts, isEphemeral, targetName, category);
      if (!isEphemeral) {
        await this.loadWordlists();
        if (merged) {
          this.selectWordlist(merged.id);
        }
      }
      return merged;
    } catch (err: any) {
      this.error = `Failed to merge wordlists: ${err?.message || err}`;
      throw err;
    } finally {
      this.isSaving = false;
    }
  }

  async previewMerge(ids: number[], opts: SanitizeOptions): Promise<MergeReportDTO> {
    try {
      return await PreviewMergeStats(ids, opts);
    } catch (err: any) {
      this.error = `Failed to preview merge: ${err?.message || err}`;
      throw err;
    }
  }

  async cleanupSandbox(path: string): Promise<void> {
    try {
      await CleanupSandboxWordlist(path);
    } catch (err: any) {
      console.warn('Sandbox cleanup error:', err);
    }
  }

  async deleteWordlist(id: number) {
    try {
      await DeleteWordlist(id);
      this.checkedIDs = this.checkedIDs.filter((i) => i !== id);
      if (this.selectedID === id) {
        this.selectedID = null;
        this.currentPage = { lines: [], total_lines: 0, offset: 0, limit: 100 };
      }
      await this.loadWordlists();
    } catch (err: any) {
      this.error = `Failed to delete wordlist: ${err?.message || err}`;
      throw err;
    }
  }

  async deleteBatch(ids: number[]): Promise<number> {
    if (!ids || ids.length === 0) return 0;
    try {
      const deletedCount = await DeleteWordlistsBatch(ids);
      this.checkedIDs = this.checkedIDs.filter((i) => !ids.includes(i));
      if (this.selectedID && ids.includes(this.selectedID)) {
        this.selectedID = null;
        this.currentPage = { lines: [], total_lines: 0, offset: 0, limit: 100 };
      }
      await this.loadWordlists();
      return deletedCount;
    } catch (err: any) {
      this.error = `Failed to delete wordlists: ${err?.message || err}`;
      throw err;
    }
  }

  async saveContent(id: number, content: string) {
    this.isSaving = true;
    this.error = '';
    try {
      await SaveWordlistContent(id, content);
      await Promise.all([
        this.loadWordlists(),
        this.loadPage(this.currentPage.offset, this.currentPage.limit),
        this.loadStats(id),
        this.loadHealth(id)
      ]);
    } catch (err: any) {
      this.error = `Failed to save content: ${err?.message || err}`;
      throw err;
    } finally {
      this.isSaving = false;
    }
  }

  async exportToFile(id: number, destinationPath: string) {
    try {
      await ExportWordlist(id, destinationPath);
    } catch (err: any) {
      this.error = `Failed to export wordlist: ${err?.message || err}`;
      throw err;
    }
  }

  async exportNative(id: number, defaultName: string): Promise<boolean> {
    try {
      const dest = await SaveWordlistDialog(defaultName);
      if (!dest) return false;
      await this.exportToFile(id, dest);
      return true;
    } catch (err: any) {
      this.error = `Export failed: ${err?.message || err}`;
      throw err;
    }
  }

  async resetBuiltinWordlist(id: number): Promise<WordlistDTO> {
    this.isSaving = true;
    this.error = '';
    try {
      const restored = await ResetBuiltinWordlist(id);
      await Promise.all([
        this.loadWordlists(),
        this.loadPage(0, this.currentPage.limit),
        this.loadStats(id),
        this.loadHealth(id)
      ]);
      return restored;
    } catch (err: any) {
      this.error = `Failed to reset built-in wordlist: ${err?.message || err}`;
      throw err;
    } finally {
      this.isSaving = false;
    }
  }

  async restoreAllBuiltinWordlists(): Promise<WordlistDTO[]> {
    this.isLoading = true;
    this.error = '';
    try {
      const restored = await RestoreAllBuiltinWordlists();
      await this.loadWordlists();
      return restored;
    } catch (err: any) {
      this.error = `Failed to restore built-in wordlists: ${err?.message || err}`;
      throw err;
    } finally {
      this.isLoading = false;
    }
  }
}

export const wordlistsStore = new WordlistsStore();
