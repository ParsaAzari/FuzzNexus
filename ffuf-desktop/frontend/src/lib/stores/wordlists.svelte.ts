import {
  ListWordlists,
  CreateWordlist,
  UpdateWordlist,
  DeleteWordlist,
  DeleteWordlistsBatch,
  ReadWordlistPage,
  SaveWordlistContent,
  ImportWordlistFromPath,
  LinkWordlistPath,
  GetWordlistStats,
  ExportWordlist,
  SelectWordlistDialog
} from '../../../wailsjs/go/main/App';
import type { WordlistDTO, WordlistPageDTO, WordlistStatsDTO } from '../types';

class WordlistsStore {
  items = $state<WordlistDTO[]>([]);
  selectedID = $state<number | null>(null);
  checkedIDs = $state<number[]>([]);
  selectedStats = $state<WordlistStatsDTO | null>(null);
  currentPage = $state<WordlistPageDTO>({ lines: [], total_lines: 0, offset: 0, limit: 100 });
  isLoading = $state<boolean>(false);
  isSaving = $state<boolean>(false);
  searchFilter = $state<string>('');
  error = $state<string>('');

  selectedWordlist = $derived(
    this.selectedID === null ? null : (this.items.find((w) => w.id === this.selectedID) ?? null)
  );

  filteredWordlists = $derived.by(() => {
    const q = this.searchFilter.trim().toLowerCase();
    if (!q) return this.items;
    return this.items.filter((w) => w.name.toLowerCase().includes(q) || w.path.toLowerCase().includes(q));
  });

  isAllChecked = $derived(
    this.filteredWordlists.length > 0 &&
      this.filteredWordlists.every((w) => this.checkedIDs.includes(w.id))
  );

  isSomeChecked = $derived(
    this.checkedIDs.length > 0 && !this.isAllChecked
  );

  checkedCount = $derived(this.checkedIDs.length);

  toggleCheck(id: number) {
    if (this.checkedIDs.includes(id)) {
      this.checkedIDs = this.checkedIDs.filter((i) => i !== id);
    } else {
      this.checkedIDs = [...this.checkedIDs, id];
    }
  }

  checkAll() {
    this.checkedIDs = this.filteredWordlists.map((w) => w.id);
  }

  uncheckAll() {
    this.checkedIDs = [];
  }

  toggleCheckAll() {
    if (this.isAllChecked) {
      this.uncheckAll();
    } else {
      this.checkAll();
    }
  }

  async loadWordlists() {
    this.isLoading = true;
    this.error = '';
    try {
      this.items = (await ListWordlists()) || [];
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
    await Promise.all([this.loadPage(0, 100), this.loadStats(id)]);
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

  async createWordlist(name: string, content: string = '') {
    this.isSaving = true;
    this.error = '';
    try {
      const created = await CreateWordlist(name, content);
      await this.loadWordlists();
      if (created) {
        this.selectWordlist(created.id);
      }
    } catch (err: any) {
      this.error = `Failed to create wordlist: ${err?.message || err}`;
      throw err;
    } finally {
      this.isSaving = false;
    }
  }

  async importFromPath(filePath: string) {
    this.isLoading = true;
    this.error = '';
    try {
      const imported = await ImportWordlistFromPath(filePath);
      await this.loadWordlists();
      if (imported) {
        this.selectWordlist(imported.id);
      }
    } catch (err: any) {
      this.error = `Failed to import wordlist: ${err?.message || err}`;
      throw err;
    } finally {
      this.isLoading = false;
    }
  }

  async linkFromPath(filePath: string) {
    this.isLoading = true;
    this.error = '';
    try {
      const linked = await LinkWordlistPath(filePath);
      await this.loadWordlists();
      if (linked) {
        this.selectWordlist(linked.id);
      }
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
      await Promise.all([this.loadWordlists(), this.loadPage(this.currentPage.offset, this.currentPage.limit), this.loadStats(id)]);
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
}

export const wordlistsStore = new WordlistsStore();
