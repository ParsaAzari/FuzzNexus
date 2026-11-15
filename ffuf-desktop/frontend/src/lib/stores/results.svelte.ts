import { GetResultsSince } from '../../../wailsjs/go/main/App';
import type { ResultDTO, CommittedEvent, SortField, SortDirection } from '../types';

class ResultsStore {
  items = $state<ResultDTO[]>([]);
  sessionID = $state<number>(0);
  lastFetchedID = $state<number>(0);
  isPulling = $state<boolean>(false);
  sortField = $state<SortField>('id');
  sortDirection = $state<SortDirection>('asc');
  quickFilter = $state<string>('');
  selectedID = $state<number | null>(null);

  selectedItem = $derived(
    this.selectedID === null ? null : (this.items.find((r) => r.id === this.selectedID) ?? null)
  );

  filteredItems = $derived.by(() => {
    let list = this.items;
    const q = this.quickFilter.trim().toLowerCase();
    if (q) {
      list = list.filter((r) => {
        if (r.url.toLowerCase().includes(q)) return true;
        if (String(r.status).includes(q)) return true;
        if (r.inputs) {
          for (const val of Object.values(r.inputs)) {
            if (val.toLowerCase().includes(q)) return true;
          }
        }
        return false;
      });
    }

    const field = this.sortField;
    const asc = this.sortDirection === 'asc' ? 1 : -1;

    return [...list].sort((a, b) => {
      let av: any = a.id;
      let bv: any = b.id;

      if (field === 'status') {
        av = a.status;
        bv = b.status;
      } else if (field === 'length') {
        av = a.length;
        bv = b.length;
      } else if (field === 'words') {
        av = a.words;
        bv = b.words;
      } else if (field === 'lines') {
        av = a.lines;
        bv = b.lines;
      } else if (field === 'duration') {
        av = a.duration_ms;
        bv = b.duration_ms;
      } else if (field === 'url') {
        av = a.url;
        bv = b.url;
      }

      if (av < bv) return -1 * asc;
      if (av > bv) return 1 * asc;
      return (a.id - b.id) * asc;
    });
  });

  initSession(sid: number) {
    this.sessionID = sid;
    this.items = [];
    this.lastFetchedID = 0;
    this.selectedID = null;
    this.pull();
  }

  onCommitted(event: CommittedEvent) {
    const latest = event.latest_id ?? (event as any).latestID;
    if (latest && latest > this.lastFetchedID) {
      this.pull();
    }
  }

  async pull() {
    if (this.isPulling || this.sessionID <= 0) return;
    this.isPulling = true;

    try {
      let hasMore = true;
      while (hasMore) {
        const page = await GetResultsSince(this.sessionID, this.lastFetchedID, 1000);
        if (page && page.results && page.results.length > 0) {
          this.items = this.items.concat(page.results);
          this.lastFetchedID = page.last_id;
          hasMore = page.has_more;
        } else {
          hasMore = false;
        }
      }
    } catch (err) {
      console.error('Failed to pull results:', err);
    } finally {
      this.isPulling = false;
    }
  }

  setSort(field: SortField) {
    if (this.sortField === field) {
      this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc';
    } else {
      this.sortField = field;
      this.sortDirection = 'asc';
    }
  }

  select(id: number | null) {
    this.selectedID = id;
  }

  clear() {
    this.items = [];
    this.sessionID = 0;
    this.lastFetchedID = 0;
    this.selectedID = null;
    this.quickFilter = '';
  }
}

export const resultsStore = new ResultsStore();
