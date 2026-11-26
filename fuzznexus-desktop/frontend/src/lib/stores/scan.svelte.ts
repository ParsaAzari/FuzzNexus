import {
  StartScan,
  PauseScan,
  ResumeScan,
  StopScan,
  UpdateRate,
  AddLiveFilter,
  GetRecentErrors,
  AnalyzeWordlistRisk,
  CreateSanitizedSandboxWordlist,
  SelectAuxiliaryFileDialog,
  SelectDirectoryDialog,
  ParseRawHTTPRequest,
  GetAvailableEncoders,
  ImportTOMLConfig,
  SearchFNHASH,
  GetPendingResumeState,
  DiscardPendingResumeState,
  ResumePendingScan
} from '../../../wailsjs/go/main/App';
import { toastStore } from './toast.svelte';
import { EventsOn } from '../../../wailsjs/runtime/runtime';
import { resultsStore } from './results.svelte';
import { filtersStore } from './filters.svelte';
import type {
  ScanState,
  ProgressData,
  StateEvent,
  CommittedEvent,
  ScanRequestDTO,
  PendingResumeDTO,
  PreflightConfig,
  SafeStopPrompt,
  SafeStopTestResult,
  NetworkErrorDTO,
  AdvisoryAlert,
  WordlistRiskReportDTO,
  WordlistDTO,
  SessionDTO
} from '../types';

export interface AdditionalWordlistEntry {
  path: string;
  keyword: string;
  encoders: string;
}

class ScanStore {
  url = $state<string>('http://127.0.0.1:8080/FUZZ');
  wordlist = $state<string>('');
  wordlistLineCount = $state<number | null>(null);
  primaryKeyword = $state<string>('FUZZ');
  primaryEncoders = $state<string>('');
  additionalWordlists = $state<AdditionalWordlistEntry[]>([]);
  inputMode = $state<'clusterbomb' | 'pitchfork' | 'sniper'>('clusterbomb');

  method = $state<string>('GET');
  threads = $state<number>(10);
  rate = $state<number>(0);
  timeout = $state<number>(10);
  delay = $state<string>('');
  maxTime = $state<number>(0);
  maxTimeJob = $state<number>(0);

  matchers = $state<string>('');
  filters = $state<string>('');
  matcherMode = $state<'or' | 'and'>('or');
  matcherSize = $state<string>('');
  matcherWords = $state<string>('');
  matcherLines = $state<string>('');
  matcherRegexp = $state<string>('');
  matcherTime = $state<string>('');

  filterMode = $state<'or' | 'and'>('or');
  filterSize = $state<string>('');
  filterWords = $state<string>('');
  filterLines = $state<string>('');
  filterRegexp = $state<string>('');
  filterTime = $state<string>('');

  data = $state<string>('');
  headers = $state<Record<string, string>>({});
  followRedirects = $state<boolean>(false);
  rawURI = $state<boolean>(false);
  http2 = $state<boolean>(false);
  ignoreBody = $state<boolean>(false);

  autoCalibration = $state<boolean>(true);
  autoCalibrationPerHost = $state<boolean>(false);
  autoCalibrationKeyword = $state<string>('FUZZ');
  autoCalibrationStrategies = $state<string[]>(['basic']);
  autoCalibrationStrings = $state<string>('');

  proxyURL = $state<string>('');
  replayProxyURL = $state<string>('');
  tlsProfile = $state<string>('');
  sni = $state<string>('');
  clientCert = $state<string>('');
  clientKey = $state<string>('');

  browserProfile = $state<string>('auto');
  rotateOnBlock = $state<boolean>(true);
  closeConnOnThrottle = $state<boolean>(false);
  detectSoftBlock = $state<boolean>(true);
  stopOn403 = $state<boolean>(false);
  stopOnErrors = $state<boolean>(false);
  stopOnAll = $state<boolean>(false);

  extensions = $state<string>('');
  dirSearchCompat = $state<boolean>(false);
  ignoreWordlistComments = $state<boolean>(false);

  recursion = $state<boolean>(false);
  recursionDepth = $state<number>(0);
  recursionStrategy = $state<'default' | 'greedy'>('default');

  requestFile = $state<string>('');
  requestProto = $state<'https' | 'http'>('https');

  inputCommand = $state<string>('');
  inputNum = $state<number>(100);
  inputShell = $state<string>('');

  scrapers = $state<string>('all');
  scraperFile = $state<string>('');
  auditLog = $state<string>('');
  debugLog = $state<string>('');
  outputDirectory = $state<string>('');

  preflights = $state<PreflightConfig[]>([]);
  postflights = $state<PreflightConfig[]>([]);
  preflightMode = $state<'per-request' | 'per-thread'>('per-request');
  preflightError = $state<'abort' | 'ignore'>('abort');
  preflightAnyHost = $state<boolean>(false);

  availableEncoders = $state<string[]>([]);

  isRequestOptionsOpen = $state<boolean>(false);
  isHistoryModalOpen = $state<boolean>(false);
  pendingResume = $state<PendingResumeDTO | null>(null);
  isResumingPending = $state<boolean>(false);

  sessionID = $state<number>(0);
  state = $state<ScanState>('Idle');
  lastError = $state<string>('');

  reqCount = $state<number>(0);
  reqTotal = $state<number>(0);
  reqSec = $state<number>(0);
  smoothedReqSec = $state<number>(0);
  bytesSent = $state<number>(0);
  bytesRecv = $state<number>(0);
  queuePos = $state<number>(0);
  queueTotal = $state<number>(0);
  errorCount = $state<number>(0);
  percentage = $state<number>(0);

  isSubmitting = $state<boolean>(false);

  safeStopPrompt = $state<SafeStopPrompt | null>(null);
  isTestingConnection = $state<boolean>(false);
  testConnectionResult = $state<SafeStopTestResult | null>(null);

  advisoryAlert = $state<AdvisoryAlert | null>(null);
  advisoryDismissed = $state<boolean>(false);
  wordlistRisk = $state<WordlistRiskReportDTO | null>(null);
  isSanitizingWordlist = $state<boolean>(false);

  recentErrors = $state<NetworkErrorDTO[]>([]);
  showErrorDiagnostics = $state<boolean>(false);
  private nextErrorIndex = 0;
  private unreachableNotified = false;
  circuitState = $state<string>('Closed');
  circuitReason = $state<string>('');

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
      this.smoothedReqSec =
        this.smoothedReqSec === 0
          ? p.req_sec
          : Math.round(this.smoothedReqSec * 0.65 + p.req_sec * 0.35);
      if (typeof p.bytes_sent === 'number') {
        this.bytesSent = p.bytes_sent;
      }
      if (typeof p.bytes_recv === 'number') {
        this.bytesRecv = p.bytes_recv;
      }
      this.queuePos = p.queue_pos;
      this.queueTotal = p.queue_total;
      this.errorCount = p.error_count;
      this.percentage = p.percentage;
      if (p.circuit_state) {
        this.circuitState = p.circuit_state;
      }
      if (p.circuit_reason) {
        this.circuitReason = p.circuit_reason;
      }
    });

    EventsOn('scan:circuit_state', (ev: { state?: string; reason?: string }) => {
      if (ev?.state) {
        this.circuitState = ev.state;
      }
      if (ev?.reason) {
        this.circuitReason = ev.reason;
      }
    });

    EventsOn('scan:error', (msg: string) => {
      this.nextErrorIndex += 1;
      this.recentErrors.push({
        index: this.nextErrorIndex,
        message: msg,
        timestamp: new Date().toISOString()
      });
      if (this.recentErrors.length > 100) {
        this.recentErrors.shift();
      }
      if (!this.unreachableNotified && this.recentErrors.length >= 10 && this.reqCount === 0) {
        this.unreachableNotified = true;
        toastStore.warning('Target unreachable — check VPN, Proxy, or target host');
      }
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

    EventsOn('scan:safe_stop_prompt', (prompt: SafeStopPrompt) => {
      this.safeStopPrompt = prompt;
      this.testConnectionResult = null;
      this.state = 'Paused';
    });

    EventsOn('scan:advisory_alert', (alert: AdvisoryAlert) => {
      if (!this.advisoryDismissed && alert && alert.verdict && alert.verdict !== 'NONE') {
        this.advisoryAlert = alert;
      }
    });
  }

  async start() {
    if (this.state === 'Running' || this.isSubmitting) return;
    this.isSubmitting = true;
    this.lastError = '';
    this.circuitState = 'Closed';
    this.circuitReason = '';
    this.advisoryAlert = null;
    this.advisoryDismissed = false;

    try {
      // Intelligently normalize target URL before dispatching (Postel's Law)
      let targetUrl = (this.url || '').trim().replace(/^["']|["']$/g, '');
      if (targetUrl) {
        if (!/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(targetUrl)) {
          const lower = targetUrl.toLowerCase();
          if (
            lower.startsWith('localhost') ||
            lower.startsWith('127.0.0.1') ||
            lower.startsWith('[::1]') ||
            lower.includes(':8000') ||
            lower.includes(':8080') ||
            lower.includes(':3000') ||
            lower.includes(':5000')
          ) {
            targetUrl = `http://${targetUrl}`;
          } else {
            targetUrl = `https://${targetUrl}`;
          }
        }

        // Check if any casing of FUZZ (fuzz, Fuzz, fUzz, etc.) exists in headers or data
        const activeKeywords = [
          this.primaryKeyword.trim() || 'FUZZ',
          ...this.additionalWordlists.map((w) => w.keyword.trim()).filter(Boolean)
        ];
        const hasKeywordOrSniper = (str: string) =>
          (this.inputMode === 'sniper' && str.includes('§')) ||
          activeKeywords.some((kw) => str.includes(kw)) ||
          /fuzz/i.test(str);

        const hasFuzzInHeaders = Object.entries(this.headers).some(
          ([k, v]) => hasKeywordOrSniper(k) || hasKeywordOrSniper(v)
        );
        const hasFuzzInData = hasKeywordOrSniper(this.data);
        const hasFuzzInTarget = hasKeywordOrSniper(targetUrl);

        if (/fuzz/i.test(targetUrl) && !activeKeywords.some((kw) => kw !== 'FUZZ' && targetUrl.includes(kw))) {
          // Normalize any casing variant of 'fuzz' in the target URL to uppercase 'FUZZ'
          targetUrl = targetUrl.replace(/fuzz/gi, 'FUZZ');
        } else if (!this.requestFile.trim() && !hasFuzzInTarget && !hasFuzzInHeaders && !hasFuzzInData && this.inputMode !== 'sniper') {
          // Only auto-append /FUZZ if neither URL, Headers nor Data have any fuzz keyword
          if (targetUrl.includes('?')) {
            const parts = targetUrl.split('?');
            const query = parts[1];
            if (query.endsWith('=') || query.endsWith('&') || query === '') {
              targetUrl = `${targetUrl}FUZZ`;
            } else {
              if (parts[0].endsWith('/')) {
                targetUrl = `${parts[0]}FUZZ?${query}`;
              } else {
                targetUrl = `${parts[0]}/FUZZ?${query}`;
              }
            }
          } else {
            if (targetUrl.endsWith('/')) {
              targetUrl = `${targetUrl}FUZZ`;
            } else {
              targetUrl = `${targetUrl}/FUZZ`;
            }
          }
        }
        this.url = targetUrl;
      }

      const preScanStatusFilters = filtersStore.activeRules
        .filter((r) => r.enabled && r.field === 'status' && r.operator === '!=' && String(r.value).trim() !== '')
        .map((r) => String(r.value).trim());
      const mergedFilters = [this.filters.trim(), ...preScanStatusFilters].filter(Boolean).join(',');

      // Build wordlist & encoder slices for single/multi-wordlist modes
      const primaryKw = this.primaryKeyword.trim() || 'FUZZ';
      const wordlistsPayload: string[] = [];
      const encodersPayload: string[] = [];

      if (!this.inputCommand.trim()) {
        if (this.wordlist.trim()) {
          if (this.inputMode !== 'sniper' && primaryKw !== 'FUZZ') {
            wordlistsPayload.push(`${this.wordlist.trim()}:${primaryKw}`);
          } else {
            wordlistsPayload.push(this.wordlist.trim());
          }
          if (this.primaryEncoders.trim()) {
            encodersPayload.push(`${this.inputMode === 'sniper' ? 'FUZZ' : primaryKw}:${this.primaryEncoders.trim()}`);
          }
        }
        if (this.inputMode !== 'sniper') {
          for (const extra of this.additionalWordlists) {
            if (!extra.path.trim()) continue;
            const kw = extra.keyword.trim() || 'FUZZ';
            wordlistsPayload.push(`${extra.path.trim()}:${kw}`);
            if (extra.encoders.trim()) {
              encodersPayload.push(`${kw}:${extra.encoders.trim()}`);
            }
          }
        }
      }

      const acStrings = this.autoCalibrationStrings
        .split(/\r?\n/)
        .map((s) => s.trim())
        .filter(Boolean);

      const req: ScanRequestDTO = {
        url: this.url,
        method: this.method,
        headers: this.headers,
        data: this.data,
        wordlist: this.wordlist,
        wordlists: wordlistsPayload.length > 0 ? wordlistsPayload : undefined,
        input_mode: this.inputMode,
        threads: this.threads,
        rate: this.rate,
        matchers: this.matchers.trim(),
        filters: mergedFilters,
        timeout: this.timeout > 0 ? this.timeout : 10,
        follow_redirects: this.followRedirects,
        auto_calibration: this.autoCalibration,
        proxy_url: this.proxyURL.trim(),
        tls_profile: this.tlsProfile,
        browser_profile: this.browserProfile,
        rotate_on_block: this.rotateOnBlock,

        // Phase 1
        replay_proxy_url: this.replayProxyURL.trim(),
        raw: this.rawURI,
        http2: this.http2,
        ignore_body: this.ignoreBody,
        close_conn_on_throttle: this.closeConnOnThrottle,
        detect_soft_block: this.detectSoftBlock,
        delay: this.delay.trim(),
        max_time: this.maxTime > 0 ? this.maxTime : 0,
        max_time_job: this.maxTimeJob > 0 ? this.maxTimeJob : 0,
        stop_on_403: this.stopOn403,
        stop_on_errors: this.stopOnErrors,
        stop_on_all: this.stopOnAll,
        extensions: this.extensions.trim(),
        dirsearch_compat: this.dirSearchCompat,
        ignore_wordlist_comments: this.ignoreWordlistComments,
        encoders: encodersPayload.length > 0 ? encodersPayload : undefined,
        matcher_mode: this.matcherMode,
        matcher_status: this.matchers.trim(),
        matcher_size: this.matcherSize.trim(),
        matcher_words: this.matcherWords.trim(),
        matcher_lines: this.matcherLines.trim(),
        matcher_regexp: this.matcherRegexp.trim(),
        matcher_time: this.matcherTime.trim(),
        filter_mode: this.filterMode,
        filter_status: mergedFilters,
        filter_size: this.filterSize.trim(),
        filter_words: this.filterWords.trim(),
        filter_lines: this.filterLines.trim(),
        filter_regexp: this.filterRegexp.trim(),
        filter_time: this.filterTime.trim(),
        auto_calibration_per_host: this.autoCalibrationPerHost,
        auto_calibration_keyword: this.autoCalibrationKeyword.trim() || 'FUZZ',
        auto_calibration_strategies: this.autoCalibrationStrategies.length > 0 ? this.autoCalibrationStrategies : ['basic'],
        auto_calibration_strings: acStrings.length > 0 ? acStrings : undefined,

        // Phase 2
        recursion: this.recursion,
        recursion_depth: this.recursionDepth > 0 ? this.recursionDepth : 0,
        recursion_strategy: this.recursionStrategy,
        request_file: this.requestFile.trim(),
        request_proto: this.requestProto,

        // Phase 3
        sni: this.sni.trim(),
        client_cert: this.clientCert.trim(),
        client_key: this.clientKey.trim(),
        input_commands: this.inputCommand.trim() ? [this.inputCommand.trim()] : undefined,
        input_num: this.inputCommand.trim() ? (this.inputNum > 0 ? this.inputNum : 100) : undefined,
        input_shell: this.inputCommand.trim() && this.inputShell.trim() ? this.inputShell.trim() : undefined,
        scrapers: this.scrapers.trim() || 'all',
        scraper_file: this.scraperFile.trim(),
        audit_log: this.auditLog.trim(),
        debug_log: this.debugLog.trim(),
        output_directory: this.outputDirectory.trim(),
        preflights: this.preflights.length > 0 ? this.preflights : undefined,
        postflights: this.postflights.length > 0 ? this.postflights : undefined,
        preflight_mode: this.preflightMode,
        preflight_error: this.preflightError,
        preflight_anyhost: this.preflightAnyHost
      } as ScanRequestDTO;

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

  setWordlist(path: string, lineCount?: number | null) {
    this.wordlist = path;
    if (lineCount !== undefined) {
      this.wordlistLineCount = lineCount;
    }
  }

  async testConnection() {
    this.isTestingConnection = true;
    this.testConnectionResult = null;
    try {
      const goApp = (window as unknown as { go?: { main?: { App?: { TestConnection?: () => Promise<SafeStopTestResult> } } } })?.go?.main?.App;
      if (goApp?.TestConnection) {
        const res = await goApp.TestConnection();
        this.testConnectionResult = res;
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      this.testConnectionResult = {
        success: false,
        new_ip_hash: '',
        error: msg
      };
    } finally {
      this.isTestingConnection = false;
    }
  }

  async resumeSafeStop() {
    try {
      const goApp = (window as unknown as { go?: { main?: { App?: { ResumeSafeStop?: () => Promise<void> } } } })?.go?.main?.App;
      if (goApp?.ResumeSafeStop) {
        await goApp.ResumeSafeStop();
      }
      this.safeStopPrompt = null;
      this.testConnectionResult = null;
      this.state = 'Running';
    } catch (err: unknown) {
      this.lastError = err instanceof Error ? err.message : String(err);
    }
  }

  async resumeAnyway() {
    try {
      const goApp = (window as unknown as { go?: { main?: { App?: { ResumeAnyway?: () => Promise<void> } } } })?.go?.main?.App;
      if (goApp?.ResumeAnyway) {
        await goApp.ResumeAnyway();
      }
      this.safeStopPrompt = null;
      this.testConnectionResult = null;
      this.state = 'Running';
    } catch (err: unknown) {
      this.lastError = err instanceof Error ? err.message : String(err);
    }
  }

  async cancelSafeStop() {
    try {
      const goApp = (window as unknown as { go?: { main?: { App?: { CancelSafeStop?: () => Promise<void> } } } })?.go?.main?.App;
      if (goApp?.CancelSafeStop) {
        await goApp.CancelSafeStop();
      }
      this.safeStopPrompt = null;
      this.testConnectionResult = null;
      this.state = 'Stopped';
    } catch (err: unknown) {
      this.lastError = err instanceof Error ? err.message : String(err);
    }
  }

  dismissSafeStopModal() {
    this.safeStopPrompt = null;
    this.testConnectionResult = null;
  }

  dismissAdvisory() {
    this.advisoryAlert = null;
    this.advisoryDismissed = true;
  }

  async analyzeWordlistRisk(path: string): Promise<WordlistRiskReportDTO | null> {
    const trimmed = (path || '').trim();
    if (!trimmed) {
      this.wordlistRisk = null;
      return null;
    }
    try {
      const report = await AnalyzeWordlistRisk(trimmed);
      if (this.wordlist?.trim() === trimmed) {
        this.wordlistRisk = report;
      }
      return report;
    } catch {
      if (this.wordlist?.trim() === trimmed) {
        this.wordlistRisk = null;
      }
      return null;
    }
  }

  async sanitizeActiveWordlist(rescanAfter = true): Promise<WordlistDTO> {
    const currentPath = (this.wordlist || '').trim();
    if (!currentPath) {
      throw new Error('No wordlist selected');
    }

    this.isSanitizingWordlist = true;
    try {
      if (this.state === 'Running' || this.state === 'Paused') {
        await StopScan();
        this.state = 'Stopped';
      }

      const sanitized = await CreateSanitizedSandboxWordlist(currentPath);
      this.setWordlist(sanitized.path, sanitized.line_count);
      this.wordlistRisk = null;
      this.advisoryAlert = null;
      this.advisoryDismissed = false;

      if (rescanAfter) {
        this.state = 'Idle';
        await this.start();
      }
      return sanitized;
    } finally {
      this.isSanitizingWordlist = false;
    }
  }

  async loadRecentErrors() {
    try {
      const errs = await GetRecentErrors();
      if (errs && Array.isArray(errs)) {
        this.recentErrors = errs;
        for (const item of errs) {
          if (item.index > this.nextErrorIndex) {
            this.nextErrorIndex = item.index;
          }
        }
      }
    } catch (err: unknown) {
      console.warn('Failed to load recent scan errors:', err);
    }
  }

  openErrorDiagnostics() {
    this.showErrorDiagnostics = true;
    this.loadRecentErrors();
  }

  toggleErrorDiagnostics() {
    this.showErrorDiagnostics = !this.showErrorDiagnostics;
    if (this.showErrorDiagnostics) {
      this.loadRecentErrors();
    }
  }

  closeErrorDiagnostics() {
    this.showErrorDiagnostics = false;
  }

  reset() {
    if (this.state === 'Running') return;
    this.sessionID = 0;
    this.state = 'Idle';
    this.lastError = '';
    this.reqCount = 0;
    this.reqTotal = 0;
    this.reqSec = 0;
    this.smoothedReqSec = 0;
    this.bytesSent = 0;
    this.bytesRecv = 0;
    this.queuePos = 0;
    this.queueTotal = 0;
    this.errorCount = 0;
    this.percentage = 0;
    this.circuitState = 'Closed';
    this.circuitReason = '';
    this.advisoryAlert = null;
    this.advisoryDismissed = false;
    this.recentErrors = [];
    this.nextErrorIndex = 0;
    this.unreachableNotified = false;
    this.showErrorDiagnostics = false;
  }

  setHeader(key: string, value: string) {
    const cleanKey = key.trim();
    if (!cleanKey) return;
    this.headers = { ...this.headers, [cleanKey]: value };
  }

  removeHeader(key: string) {
    const next = { ...this.headers };
    delete next[key];
    this.headers = next;
  }

  async checkPendingResume() {
    try {
      const pending = await GetPendingResumeState();
      this.pendingResume = pending ?? null;
    } catch {
      this.pendingResume = null;
    }
  }

  async resumeSavedScan() {
    if (this.isResumingPending || this.state === 'Running') return;
    this.isResumingPending = true;
    this.lastError = '';
    try {
      const sid = await ResumePendingScan();
      if (this.pendingResume) {
        this.url = this.pendingResume.target_url || this.url;
        this.method = this.pendingResume.method || this.method;
        this.wordlist = this.pendingResume.wordlist || this.wordlist;
        this.browserProfile = this.pendingResume.browser_profile || this.browserProfile;
        this.tlsProfile = this.pendingResume.tls_profile || this.tlsProfile;
      }
      this.pendingResume = null;
      this.sessionID = sid;
      this.state = 'Running';
      resultsStore.initSession(sid);
    } catch (err: unknown) {
      this.lastError = err instanceof Error ? err.message : String(err);
    } finally {
      this.isResumingPending = false;
    }
  }

  async discardSavedScan() {
    try {
      await DiscardPendingResumeState();
    } catch {
      // Ignore cleanup errors
    } finally {
      this.pendingResume = null;
    }
  }

  loadHistoricalSession(session: SessionDTO) {
    if (this.state === 'Running' || this.state === 'Paused') return;
    this.sessionID = session.id;
    this.state = (session.state as ScanState) || 'Completed';
    this.lastError = session.last_error || '';
    this.reqCount = session.total_requests || 0;
    this.reqTotal = session.total_requests || 0;
    this.reqSec = 0;
    this.smoothedReqSec = 0;
    this.bytesSent = 0;
    this.bytesRecv = 0;
    this.errorCount = 0;
    this.percentage = session.state === 'Completed' ? 100 : 0;
    this.cloneSessionConfig(session);
    resultsStore.initSession(session.id);
    this.isHistoryModalOpen = false;
  }

  async loadAvailableEncoders() {
    if (this.availableEncoders.length > 0) return this.availableEncoders;
    try {
      const list = await GetAvailableEncoders();
      if (Array.isArray(list)) {
        this.availableEncoders = list;
      }
    } catch {
      this.availableEncoders = ['urlencode', 'b64encode', 'htmlencode', 'hexencode', 'md5', 'sha1', 'sha256'];
    }
    return this.availableEncoders;
  }

  async parseAndApplyRawRequest(rawText: string, proto: 'https' | 'http' = 'https') {
    const parsed = await ParseRawHTTPRequest(rawText, proto);
    if (!parsed) return;
    if (parsed.method) this.method = parsed.method;
    if (parsed.url) this.url = parsed.url;
    this.headers = parsed.headers || {};
    this.data = parsed.data || '';
    this.requestFile = '';
  }

  async pickAuxiliaryFile(title: string, filterName: string, filterPattern: string): Promise<string> {
    return await SelectAuxiliaryFileDialog(title, filterName, filterPattern);
  }

  async pickDirectory(title: string): Promise<string> {
    return await SelectDirectoryDialog(title);
  }

  async importTOMLConfig(path: string) {
    const dto = await ImportTOMLConfig(path);
    if (dto) {
      this.applyDTOConfig(dto);
    }
  }

  async searchFNHASH(hash: string): Promise<string> {
    return await SearchFNHASH(hash);
  }

  cloneSessionConfig(session: SessionDTO) {
    if (!session.config_redacted_json) return;
    try {
      const parsed = JSON.parse(session.config_redacted_json) as Partial<ScanRequestDTO>;
      this.applyDTOConfig(parsed);
    } catch {
      // Ignore malformed historical config JSON
    }
  }

  applyDTOConfig(parsed: Partial<ScanRequestDTO>) {
      if (parsed.url !== undefined) this.url = parsed.url;
      if (parsed.method) this.method = parsed.method;
      if (parsed.wordlist) this.wordlist = parsed.wordlist;
      if (Array.isArray(parsed.wordlists) && parsed.wordlists.length > 0) {
        const splitSpec = (spec: string) => {
          const idx = spec.lastIndexOf(':');
          if (idx > 1) return { path: spec.slice(0, idx), kw: spec.slice(idx + 1) };
          return { path: spec, kw: 'FUZZ' };
        };
        const encByKw: Record<string, string> = {};
        if (Array.isArray(parsed.encoders)) {
          for (const encSpec of parsed.encoders) {
            const cIdx = encSpec.indexOf(':');
            if (cIdx > 0) {
              encByKw[encSpec.slice(0, cIdx).trim()] = encSpec.slice(cIdx + 1).trim();
            }
          }
        }
        const first = splitSpec(parsed.wordlists[0]);
        this.wordlist = first.path;
        this.primaryKeyword = first.kw;
        this.primaryEncoders = encByKw[first.kw] || encByKw['FUZZ'] || '';
        this.additionalWordlists = parsed.wordlists.slice(1).map((s) => {
          const p = splitSpec(s);
          return { path: p.path, keyword: p.kw, encoders: encByKw[p.kw] || '' };
        });
      } else if (Array.isArray(parsed.encoders) && parsed.encoders.length > 0) {
        const firstEnc = parsed.encoders[0];
        const cIdx = firstEnc.indexOf(':');
        this.primaryEncoders = cIdx > 0 ? firstEnc.slice(cIdx + 1).trim() : firstEnc;
      }
      if (parsed.threads && parsed.threads > 0) this.threads = parsed.threads;
      if (typeof parsed.rate === 'number') this.rate = parsed.rate;
      if (typeof parsed.timeout === 'number' && parsed.timeout > 0) this.timeout = parsed.timeout;
      if (typeof parsed.data === 'string') this.data = parsed.data;
      if (typeof parsed.follow_redirects === 'boolean') this.followRedirects = parsed.follow_redirects;
      if (typeof parsed.auto_calibration === 'boolean') this.autoCalibration = parsed.auto_calibration;
      if (typeof parsed.proxy_url === 'string') this.proxyURL = parsed.proxy_url;
      if (typeof parsed.replay_proxy_url === 'string') this.replayProxyURL = parsed.replay_proxy_url;
      if (typeof parsed.tls_profile === 'string') this.tlsProfile = parsed.tls_profile;
      if (typeof parsed.browser_profile === 'string') this.browserProfile = parsed.browser_profile;
      if (typeof parsed.raw === 'boolean') this.rawURI = parsed.raw;
      if (typeof parsed.http2 === 'boolean') this.http2 = parsed.http2;
      if (typeof parsed.ignore_body === 'boolean') this.ignoreBody = parsed.ignore_body;
      if (typeof parsed.delay === 'string') this.delay = parsed.delay;
      if (typeof parsed.max_time === 'number') this.maxTime = parsed.max_time;
      if (typeof parsed.max_time_job === 'number') this.maxTimeJob = parsed.max_time_job;
      if (typeof parsed.stop_on_403 === 'boolean') this.stopOn403 = parsed.stop_on_403;
      if (typeof parsed.stop_on_errors === 'boolean') this.stopOnErrors = parsed.stop_on_errors;
      if (typeof parsed.stop_on_all === 'boolean') this.stopOnAll = parsed.stop_on_all;
      if (typeof parsed.close_conn_on_throttle === 'boolean') this.closeConnOnThrottle = parsed.close_conn_on_throttle;
      if (typeof parsed.detect_soft_block === 'boolean') this.detectSoftBlock = parsed.detect_soft_block;
      if (typeof parsed.extensions === 'string') this.extensions = parsed.extensions;
      if (typeof parsed.dirsearch_compat === 'boolean') this.dirSearchCompat = parsed.dirsearch_compat;
      if (typeof parsed.ignore_wordlist_comments === 'boolean') this.ignoreWordlistComments = parsed.ignore_wordlist_comments;
      if (parsed.input_mode === 'clusterbomb' || parsed.input_mode === 'pitchfork' || parsed.input_mode === 'sniper') {
        this.inputMode = parsed.input_mode;
      }
      if (typeof parsed.matcher_status === 'string') this.matchers = parsed.matcher_status;
      else if (typeof parsed.matchers === 'string') this.matchers = parsed.matchers;
      if (parsed.matcher_mode === 'and' || parsed.matcher_mode === 'or') this.matcherMode = parsed.matcher_mode;
      if (typeof parsed.matcher_size === 'string') this.matcherSize = parsed.matcher_size;
      if (typeof parsed.matcher_words === 'string') this.matcherWords = parsed.matcher_words;
      if (typeof parsed.matcher_lines === 'string') this.matcherLines = parsed.matcher_lines;
      if (typeof parsed.matcher_regexp === 'string') this.matcherRegexp = parsed.matcher_regexp;
      if (typeof parsed.matcher_time === 'string') this.matcherTime = parsed.matcher_time;

      if (typeof parsed.filter_status === 'string') this.filters = parsed.filter_status;
      else if (typeof parsed.filters === 'string') this.filters = parsed.filters;
      if (parsed.filter_mode === 'and' || parsed.filter_mode === 'or') this.filterMode = parsed.filter_mode;
      if (typeof parsed.filter_size === 'string') this.filterSize = parsed.filter_size;
      if (typeof parsed.filter_words === 'string') this.filterWords = parsed.filter_words;
      if (typeof parsed.filter_lines === 'string') this.filterLines = parsed.filter_lines;
      if (typeof parsed.filter_regexp === 'string') this.filterRegexp = parsed.filter_regexp;
      if (typeof parsed.filter_time === 'string') this.filterTime = parsed.filter_time;

      if (typeof parsed.auto_calibration_per_host === 'boolean') this.autoCalibrationPerHost = parsed.auto_calibration_per_host;
      if (typeof parsed.auto_calibration_keyword === 'string') this.autoCalibrationKeyword = parsed.auto_calibration_keyword;
      if (Array.isArray(parsed.auto_calibration_strategies)) this.autoCalibrationStrategies = parsed.auto_calibration_strategies;
      if (Array.isArray(parsed.auto_calibration_strings)) this.autoCalibrationStrings = parsed.auto_calibration_strings.join('\n');

      if (typeof parsed.recursion === 'boolean') this.recursion = parsed.recursion;
      if (typeof parsed.recursion_depth === 'number') this.recursionDepth = parsed.recursion_depth;
      if (parsed.recursion_strategy === 'default' || parsed.recursion_strategy === 'greedy') this.recursionStrategy = parsed.recursion_strategy;
      if (typeof parsed.request_file === 'string') this.requestFile = parsed.request_file;
      if (parsed.request_proto === 'http' || parsed.request_proto === 'https') this.requestProto = parsed.request_proto;

      if (typeof parsed.sni === 'string') this.sni = parsed.sni;
      if (typeof parsed.client_cert === 'string') this.clientCert = parsed.client_cert;
      if (typeof parsed.client_key === 'string') this.clientKey = parsed.client_key;
      if (Array.isArray(parsed.input_commands)) this.inputCommand = parsed.input_commands[0] || '';
      if (typeof parsed.input_num === 'number') this.inputNum = parsed.input_num;
      if (typeof parsed.input_shell === 'string') this.inputShell = parsed.input_shell;
      if (typeof parsed.scrapers === 'string') this.scrapers = parsed.scrapers;
      if (typeof parsed.scraper_file === 'string') this.scraperFile = parsed.scraper_file;
      if (typeof parsed.audit_log === 'string') this.auditLog = parsed.audit_log;
      if (typeof parsed.debug_log === 'string') this.debugLog = parsed.debug_log;
      if (typeof parsed.output_directory === 'string') this.outputDirectory = parsed.output_directory;
      if (Array.isArray(parsed.preflights)) this.preflights = parsed.preflights;
      if (Array.isArray(parsed.postflights)) this.postflights = parsed.postflights;
      if (parsed.preflight_mode === 'per-request' || parsed.preflight_mode === 'per-thread') this.preflightMode = parsed.preflight_mode;
      if (parsed.preflight_error === 'abort' || parsed.preflight_error === 'ignore') this.preflightError = parsed.preflight_error;
      if (typeof parsed.preflight_anyhost === 'boolean') this.preflightAnyHost = parsed.preflight_anyhost;

      if (parsed.headers && typeof parsed.headers === 'object') {
        const safeHeaders: Record<string, string> = {};
        for (const [k, v] of Object.entries(parsed.headers)) {
          if (v !== '***') {
            safeHeaders[k] = v;
          }
        }
        this.headers = safeHeaders;
      }
  }
}

export const scanStore = new ScanStore();
