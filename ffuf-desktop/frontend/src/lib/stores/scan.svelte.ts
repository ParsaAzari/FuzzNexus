import {
  StartScan,
  PauseScan,
  ResumeScan,
  StopScan,
  UpdateRate,
  AddLiveFilter
} from '../../../wailsjs/go/main/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';
import { resultsStore } from './results.svelte';
import type { ScanState, ProgressData, StateEvent, CommittedEvent, ScanRequestDTO } from '../types';

class ScanStore {
  url = $state<string>('http://127.0.0.1:8080/FUZZ');
  wordlist = $state<string>('');
  method = $state<string>('GET');
  threads = $state<number>(10);
  rate = $state<number>(0);
  matchers = $state<string>('');
  filters = $state<string>('');
  data = $state<string>('');
  headers = $state<Record<string, string>>({});

  sessionID = $state<number>(0);
  state = $state<ScanState>('Idle');
  lastError = $state<string>('');

  reqCount = $state<number>(0);
  reqTotal = $state<number>(0);
  reqSec = $state<number>(0);
  queuePos = $state<number>(0);
  queueTotal = $state<number>(0);
  errorCount = $state<number>(0);
  percentage = $state<number>(0);

  isSubmitting = $state<boolean>(false);

  constructor() {
    if (typeof window !== 'undefined') {
      this.initEventListeners();
    }
  }

  private initEventListeners() {
    EventsOn('scan:progress', (p: ProgressData) => {
      this.reqCount = p.req_count;
      this.reqTotal = p.req_total;
      this.reqSec = p.req_sec;
      this.queuePos = p.queue_pos;
      this.queueTotal = p.queue_total;
      this.errorCount = p.error_count;
      this.percentage = p.percentage;
    });

    EventsOn('scan:state', (event: StateEvent) => {
      this.state = event.state;
      if (event.error) {
        this.lastError = event.error;
      }
      if (event.state === 'Completed' || event.state === 'Stopped' || event.state === 'Failed') {
        // Trigger a final pull to guarantee all results in SQLite are displayed
        resultsStore.pull();
      }
    });

    EventsOn('results:committed', (event: CommittedEvent) => {
      resultsStore.onCommitted(event);
    });
  }

  async start() {
    if (this.state === 'Running' || this.isSubmitting) return;
    this.isSubmitting = true;
    this.lastError = '';

    try {
      const req: ScanRequestDTO = {
        url: this.url,
        method: this.method,
        headers: this.headers,
        data: this.data,
        wordlist: this.wordlist,
        threads: this.threads,
        rate: this.rate,
        matchers: this.matchers,
        filters: this.filters,
        timeout: 10,
        follow_redirects: false
      };

      const sid = await StartScan(req);
      this.sessionID = sid;
      this.state = 'Running';
      resultsStore.initSession(sid);
    } catch (err: any) {
      this.lastError = err?.message || String(err);
      this.state = 'Failed';
    } finally {
      this.isSubmitting = false;
    }
  }

  async pause() {
    if (this.state !== 'Running') return;
    try {
      await PauseScan();
      this.state = 'Paused';
    } catch (err: any) {
      this.lastError = err?.message || String(err);
    }
  }

  async resume() {
    if (this.state !== 'Paused') return;
    try {
      await ResumeScan();
      this.state = 'Running';
    } catch (err: any) {
      this.lastError = err?.message || String(err);
    }
  }

  async stop() {
    if (this.state === 'Idle' || this.state === 'Completed' || this.state === 'Stopped') return;
    this.state = 'Stopping';
    try {
      await StopScan();
    } catch (err: any) {
      this.lastError = err?.message || String(err);
    }
  }

  async changeRate(newRate: number) {
    this.rate = newRate;
    if (this.state === 'Running' || this.state === 'Paused') {
      try {
        await UpdateRate(newRate);
      } catch (err: any) {
        console.error('Failed to update rate:', err);
      }
    }
  }

  async addFilter(name: string, value: string) {
    try {
      await AddLiveFilter(name, value);
    } catch (err: any) {
      console.error('Failed to add live filter:', err);
    }
  }
}

export const scanStore = new ScanStore();
