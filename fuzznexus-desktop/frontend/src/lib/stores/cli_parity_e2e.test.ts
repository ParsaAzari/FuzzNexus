import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockStartScan = vi.fn().mockResolvedValue(42);
const mockAddLiveFilter = vi.fn().mockResolvedValue(undefined);
const mockParseRaw = vi.fn().mockResolvedValue({
  method: 'POST',
  url: 'https://parsaazari.com/api/v1?id=FUZZ',
  headers: { 'Content-Type': 'application/json', Authorization: 'Bearer tok' },
  data: '{"item":"FUZZ"}'
});
const mockImportTOML = vi.fn().mockResolvedValue({
  url: 'https://parsaazari.com/FUZZ',
  method: 'GET',
  headers: { 'X-Test': '1' },
  wordlist: '/tmp/wl.txt',
  threads: 25,
  rate: 50,
  timeout: 15,
  follow_redirects: true,
  auto_calibration: true,
  auto_calibration_per_host: true,
  auto_calibration_keyword: 'FUZZ',
  auto_calibration_strategies: ['basic', 'advanced'],
  extensions: '.php,.json',
  dirsearch_compat: true,
  ignore_wordlist_comments: true,
  raw: true,
  http2: true,
  ignore_body: false,
  replay_proxy_url: 'http://127.0.0.1:8080',
  delay: '0.1-0.4',
  max_time: 120,
  stop_on_403: true,
  recursion: true,
  recursion_depth: 2,
  recursion_strategy: 'greedy',
  encoders: ['FUZZ:urlencode b64encode'],
  wordlists: ['/tmp/wl.txt:FUZZ'],
  sni: 'origin.parsaazari.com',
  audit_log: '/tmp/audit.log',
  debug_log: '/tmp/debug.log',
  output_directory: '/tmp/od'
});

vi.mock('../../../wailsjs/go/main/App', () => ({
  StartScan: (...args: any[]) => mockStartScan(...args),
  PauseScan: vi.fn().mockResolvedValue(undefined),
  ResumeScan: vi.fn().mockResolvedValue(undefined),
  StopScan: vi.fn().mockResolvedValue(undefined),
  UpdateRate: vi.fn().mockResolvedValue(undefined),
  AddLiveFilter: (...args: any[]) => mockAddLiveFilter(...args),
  GetRecentErrors: vi.fn().mockResolvedValue([]),
  AnalyzeWordlistRisk: vi.fn().mockResolvedValue(null),
  CreateSanitizedSandboxWordlist: vi.fn(),
  SelectAuxiliaryFileDialog: vi.fn().mockResolvedValue('/tmp/picked.txt'),
  SelectDirectoryDialog: vi.fn().mockResolvedValue('/tmp/picked_dir'),
  ParseRawHTTPRequest: (...args: any[]) => mockParseRaw(...args),
  GetAvailableEncoders: vi.fn().mockResolvedValue(['urlencode', 'b64encode', 'htmlencode']),
  ImportTOMLConfig: (...args: any[]) => mockImportTOML(...args),
  SearchFNHASH: vi.fn().mockResolvedValue('GET /admin HTTP/1.1'),
  GetPendingResumeState: vi.fn().mockResolvedValue(null),
  DiscardPendingResumeState: vi.fn().mockResolvedValue(undefined),
  ResumePendingScan: vi.fn().mockResolvedValue(1),
  GetResultsSince: vi.fn().mockResolvedValue({ results: [], has_more: false, last_id: 0 }),
  GetFilterPresets: vi.fn().mockResolvedValue([]),
  SaveFilterPreset: vi.fn().mockResolvedValue(undefined),
  DeleteFilterPreset: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn()
}));

import { scanStore } from './scan.svelte';
import { filtersStore } from './filters.svelte';
import { resultsStore } from './results.svelte';

describe('Full CLI-to-GUI E2E Store & Options Parity', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    scanStore.state = 'Idle';
    scanStore.additionalWordlists = [];
    scanStore.primaryKeyword = 'FUZZ';
    scanStore.primaryEncoders = '';
    scanStore.inputMode = 'clusterbomb';
    scanStore.inputCommand = '';
    scanStore.requestFile = '';
    filtersStore.reset();
    resultsStore.clear();
  });

  it('forwards singular word/line filter names to live Go engine', async () => {
    scanStore.state = 'Running';
    filtersStore.addQuickFilter('words', '!=', '42', 'engine');
    filtersStore.addQuickFilter('lines', '!=', '15', 'both');
    filtersStore.addQuickFilter('length', '!=', '1024', 'engine');
    filtersStore.addQuickFilter('status', '!=', '404', 'engine');

    expect(mockAddLiveFilter).toHaveBeenCalledWith('word', '42');
    expect(mockAddLiveFilter).toHaveBeenCalledWith('line', '15');
    expect(mockAddLiveFilter).toHaveBeenCalledWith('size', '1024');
    expect(mockAddLiveFilter).toHaveBeenCalledWith('status', '404');
  });

  it('dispatches all Phase 1-3 CLI options in StartScan DTO accurately', async () => {
    scanStore.url = 'https://parsaazari.com/FUZZ';
    scanStore.wordlist = '/tmp/users.txt';
    scanStore.primaryKeyword = 'USER';
    scanStore.primaryEncoders = 'urlencode b64encode';
    scanStore.additionalWordlists = [
      { path: '/tmp/pass.txt', keyword: 'PASS', encoders: 'md5' }
    ];
    scanStore.inputMode = 'pitchfork';
    scanStore.threads = 15;
    scanStore.rate = 30;
    scanStore.timeout = 20;
    scanStore.delay = '0.2-0.8';
    scanStore.maxTime = 300;
    scanStore.maxTimeJob = 60;
    scanStore.rawURI = true;
    scanStore.http2 = true;
    scanStore.ignoreBody = true;
    scanStore.replayProxyURL = 'http://127.0.0.1:8080';
    scanStore.extensions = '.php,.json';
    scanStore.dirSearchCompat = true;
    scanStore.ignoreWordlistComments = true;
    scanStore.matchers = 'all';
    scanStore.matcherMode = 'and';
    scanStore.matcherSize = '100-500';
    scanStore.matcherRegexp = 'admin';
    scanStore.matcherTime = '>50';
    scanStore.filters = '404';
    scanStore.filterMode = 'and';
    scanStore.filterWords = '10';
    scanStore.filterLines = '5';
    scanStore.autoCalibrationPerHost = true;
    scanStore.autoCalibrationKeyword = 'USER';
    scanStore.autoCalibrationStrategies = ['basic', 'advanced'];
    scanStore.autoCalibrationStrings = 'probe1\nprobe2';
    scanStore.stopOn403 = true;
    scanStore.stopOnErrors = true;
    scanStore.stopOnAll = true;
    scanStore.closeConnOnThrottle = true;
    scanStore.detectSoftBlock = false;
    scanStore.recursion = true;
    scanStore.recursionDepth = 3;
    scanStore.recursionStrategy = 'greedy';
    scanStore.sni = 'cdn.parsaazari.com';
    scanStore.clientCert = '/tmp/c.crt';
    scanStore.clientKey = '/tmp/c.key';
    scanStore.scrapers = 'secrets';
    scanStore.scraperFile = '/tmp/s.json';
    scanStore.auditLog = '/tmp/audit.log';
    scanStore.debugLog = '/tmp/debug.log';
    scanStore.outputDirectory = '/tmp/out';
    scanStore.preflights = [{ request_file: '/tmp/pre.req', vars: [{ name: 'CSRF', regex: '', source: 'auto', key: '_token' }] } as any];
    scanStore.preflightMode = 'per-thread';
    scanStore.preflightError = 'ignore';
    scanStore.preflightAnyHost = true;

    await scanStore.start();

    expect(mockStartScan).toHaveBeenCalledTimes(1);
    const dto = mockStartScan.mock.calls[0][0];
    expect(dto.wordlists).toEqual(['/tmp/users.txt:USER', '/tmp/pass.txt:PASS']);
    expect(dto.encoders).toEqual(['USER:urlencode b64encode', 'PASS:md5']);
    expect(dto.input_mode).toBe('pitchfork');
    expect(dto.timeout).toBe(20);
    expect(dto.delay).toBe('0.2-0.8');
    expect(dto.raw).toBe(true);
    expect(dto.http2).toBe(true);
    expect(dto.ignore_body).toBe(true);
    expect(dto.replay_proxy_url).toBe('http://127.0.0.1:8080');
    expect(dto.extensions).toBe('.php,.json');
    expect(dto.dirsearch_compat).toBe(true);
    expect(dto.ignore_wordlist_comments).toBe(true);
    expect(dto.matcher_status).toBe('all');
    expect(dto.matcher_mode).toBe('and');
    expect(dto.filter_words).toBe('10');
    expect(dto.auto_calibration_strings).toEqual(['probe1', 'probe2']);
    expect(dto.recursion).toBe(true);
    expect(dto.recursion_strategy).toBe('greedy');
    expect(dto.sni).toBe('cdn.parsaazari.com');
    expect(dto.audit_log).toBe('/tmp/audit.log');
    expect(dto.debug_log).toBe('/tmp/debug.log');
    expect(dto.output_directory).toBe('/tmp/out');
    expect(dto.preflight_mode).toBe('per-thread');
    expect(dto.preflights).toHaveLength(1);
  });

  it('preserves sniper §marker§ syntax without appending /FUZZ', async () => {
    scanStore.inputMode = 'sniper';
    scanStore.url = 'https://parsaazari.com/api/§item§';
    scanStore.wordlist = '/tmp/items.txt';

    await scanStore.start();
    const dto = mockStartScan.mock.calls[0][0];
    expect(dto.url).toBe('https://parsaazari.com/api/§item§');
    expect(dto.input_mode).toBe('sniper');
  });

  it('imports .fuzznexusrc TOML config and parses raw HTTP requests into UI state', async () => {
    await scanStore.importTOMLConfig('/tmp/.fuzznexusrc');
    expect(scanStore.threads).toBe(25);
    expect(scanStore.replayProxyURL).toBe('http://127.0.0.1:8080');
    expect(scanStore.recursionStrategy).toBe('greedy');
    expect(scanStore.primaryEncoders).toBe('urlencode b64encode');
    expect(scanStore.debugLog).toBe('/tmp/debug.log');

    await scanStore.parseAndApplyRawRequest('POST /api/v1 HTTP/1.1\nHost: parsaazari.com', 'https');
    expect(scanStore.method).toBe('POST');
    expect(scanStore.url).toBe('https://parsaazari.com/api/v1?id=FUZZ');
    expect(scanStore.data).toBe('{"item":"FUZZ"}');
  });

  it('filters results by FNHASH and scraper_data in resultsStore.quickFilter', () => {
    resultsStore.items = [
      {
        id: 1,
        pos: 1,
        inputs: { FUZZ: 'robots.txt' },
        status: 200,
        length: 120,
        words: 15,
        lines: 4,
        duration_ms: 25,
        url: 'https://parsaazari.com/robots.txt',
        host: 'parsaazari.com',
        hash: 'abc9988',
        scraper_data: { token: 'secret_apikey_123' },
        cancelled: false
      } as any
    ];

    resultsStore.quickFilter = 'abc9988';
    expect(resultsStore.filteredItems).toHaveLength(1);

    resultsStore.quickFilter = 'secret_apikey_123';
    expect(resultsStore.filteredItems).toHaveLength(1);
  });
});
