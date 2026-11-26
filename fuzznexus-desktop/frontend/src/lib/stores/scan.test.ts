import { describe, it, expect, beforeEach, vi } from 'vitest';

const { eventHandlers, appMocks } = vi.hoisted(() => {
  (globalThis as any).window = (globalThis as any).window || {};
  return {
    eventHandlers: new Map<string, (payload: any) => void>(),
    appMocks: {
      StartScan: vi.fn().mockResolvedValue(81),
      PauseScan: vi.fn().mockResolvedValue(undefined),
      ResumeScan: vi.fn().mockResolvedValue(undefined),
      StopScan: vi.fn().mockResolvedValue(undefined),
      UpdateRate: vi.fn().mockResolvedValue(undefined),
      AddLiveFilter: vi.fn().mockResolvedValue(undefined),
      GetRecentErrors: vi.fn().mockResolvedValue([]),
      GetResultsSince: vi.fn().mockResolvedValue({ results: [], has_more: false, last_id: 0 }),
      GetResultsFiltered: vi.fn().mockResolvedValue({ results: [], total_matches: 0 }),
      AnalyzeWordlistRisk: vi.fn().mockResolvedValue({
        path: '/wl/builtin_bypasses_heuristics.txt',
        total_lines: 323,
        traversal_count: 60,
        waf_signature_count: 0,
        sensitive_route_count: 10,
        risk_level: 'high',
        sample_risky_lines: ['.//admin', './/system'],
        summary: 'Contains 60 traversal/bypass patterns.'
      }),
      CreateSanitizedSandboxWordlist: vi.fn().mockResolvedValue({
        id: 0,
        name: 'Sanitized (builtin_bypasses_heuristics.txt)',
        path: '/sandboxes/wordlists/sb_sanitized_123.tmp',
        line_count: 263,
        size_bytes: 3100,
        is_linked: false,
        category: 'sandbox',
        created_at: '',
        updated_at: ''
      }),
      GetPendingResumeState: vi.fn().mockResolvedValue({
        session_id: 55,
        target_url: 'https://parsaazari.com/FUZZ',
        method: 'GET',
        wordlist: '/wl/params.txt',
        browser_profile: 'chrome-macos',
        tls_profile: 'chrome',
        current_index: 42,
        ip_hash: 'aa11bb22',
        timestamp: '2026-11-24T12:00:00Z'
      }),
      DiscardPendingResumeState: vi.fn().mockResolvedValue(undefined),
      ResumePendingScan: vi.fn().mockResolvedValue(99)
    }
  };
});

vi.mock('../../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn((event: string, cb: (payload: any) => void) => {
    eventHandlers.set(event, cb);
    return () => {};
  })
}));

vi.mock('../../../wailsjs/go/main/App', () => appMocks);

import { scanStore } from './scan.svelte';
import { filtersStore } from './filters.svelte';

describe('ScanStore Dynamic Advisory & Passive Risk Actions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    scanStore.state = 'Idle';
    scanStore.reset();
    scanStore.setWordlist('/wl/builtin_bypasses_heuristics.txt', 323);
  });

  it('analyzes wordlist risk passively', async () => {
    const report = await scanStore.analyzeWordlistRisk('/wl/builtin_bypasses_heuristics.txt');
    expect(appMocks.AnalyzeWordlistRisk).toHaveBeenCalledWith('/wl/builtin_bypasses_heuristics.txt');
    expect(report?.risk_level).toBe('high');
    expect(scanStore.wordlistRisk?.traversal_count).toBe(60);

    await scanStore.analyzeWordlistRisk('');
    expect(scanStore.wordlistRisk).toBeNull();
  });

  it('captures scan:advisory_alert and respects session dismiss', () => {
    const handler = eventHandlers.get('scan:advisory_alert');
    expect(handler).toBeDefined();

    handler!({
      session_id: 80,
      verdict: 'WORDLIST_WAF_TRIGGER',
      confidence: 99,
      title: 'Wordlist Payloads Triggering WAF',
      summary: 'Your IP is healthy.',
      evidence_header: 'x-blocked-by: bootstrap-security',
      fixed_body_length: 13,
      sample_payloads: ['.//system', './/manager'],
      canary: {
        root_url: 'https://parsaazari.com/',
        root_status: 200,
        benign_path_status: 404,
        checked_at: ''
      }
    });

    expect(scanStore.advisoryAlert?.verdict).toBe('WORDLIST_WAF_TRIGGER');
    expect(scanStore.advisoryAlert?.evidence_header).toBe('x-blocked-by: bootstrap-security');

    scanStore.dismissAdvisory();
    expect(scanStore.advisoryAlert).toBeNull();
    expect(scanStore.advisoryDismissed).toBe(true);

    // Re-emitting inside same dismissed session should keep banner suppressed
    handler!({
      session_id: 80,
      verdict: 'WORDLIST_WAF_TRIGGER',
      confidence: 95,
      title: 'Duplicate',
      summary: 'Duplicate',
      canary: { root_url: 'https://parsaazari.com/', root_status: 200, checked_at: '' }
    });
    expect(scanStore.advisoryAlert).toBeNull();
  });

  it('creates sanitized sandbox copy and restarts scan in one click', async () => {
    scanStore.url = 'https://parsaazari.com/FUZZ';
    scanStore.state = 'Running';

    const sanitized = await scanStore.sanitizeActiveWordlist(true);
    expect(appMocks.StopScan).toHaveBeenCalledTimes(1);
    expect(appMocks.CreateSanitizedSandboxWordlist).toHaveBeenCalledWith('/wl/builtin_bypasses_heuristics.txt');
    expect(sanitized.path).toBe('/sandboxes/wordlists/sb_sanitized_123.tmp');
    expect(scanStore.wordlist).toBe('/sandboxes/wordlists/sb_sanitized_123.tmp');
    expect(scanStore.wordlistLineCount).toBe(263);
    expect(appMocks.StartScan).toHaveBeenCalledTimes(1);
  });

  it('passes headers, body, proxyURL, autoCalibration, followRedirects, and tlsProfile to StartScan', async () => {
    scanStore.url = 'https://api.example.com/v1/login';
    scanStore.method = 'POST';
    scanStore.setHeader('Authorization', 'Bearer jwt123');
    scanStore.setHeader('X-Custom', 'val');
    scanStore.removeHeader('X-Custom');
    scanStore.data = '{"user":"FUZZ"}';
    scanStore.proxyURL = 'http://127.0.0.1:8080';
    scanStore.followRedirects = true;
    scanStore.autoCalibration = true;
    scanStore.tlsProfile = 'chrome';

    await scanStore.start();

    expect(appMocks.StartScan).toHaveBeenCalledWith(
      expect.objectContaining({
        url: 'https://api.example.com/v1/login',
        method: 'POST',
        headers: { Authorization: 'Bearer jwt123' },
        data: '{"user":"FUZZ"}',
        proxy_url: 'http://127.0.0.1:8080',
        follow_redirects: true,
        auto_calibration: true,
        tls_profile: 'chrome'
      })
    );
  });

  it('loads historical session results and clones non-redacted config', () => {
    scanStore.loadHistoricalSession({
      id: 77,
      started_at: '2026-11-24T10:00:00Z',
      state: 'Completed',
      config_json: '{}',
      config_redacted_json: JSON.stringify({
        url: 'https://target.local/FUZZ',
        method: 'POST',
        wordlist: '/wl/api.txt',
        proxy_url: 'http://127.0.0.1:8080',
        tls_profile: 'firefox',
        headers: { Authorization: '***', 'X-Env': 'prod' }
      }),
      total_requests: 120
    } as any);

    expect(scanStore.sessionID).toBe(77);
    expect(scanStore.url).toBe('https://target.local/FUZZ');
    expect(scanStore.method).toBe('POST');
    expect(scanStore.proxyURL).toBe('http://127.0.0.1:8080');
    expect(scanStore.tlsProfile).toBe('firefox');
    expect(scanStore.headers).toEqual({ 'X-Env': 'prod' });
    expect(appMocks.GetResultsSince).toHaveBeenCalledWith(77, 0, 1000);
  });

  it('checks, resumes, and discards pending resume checkpoints', async () => {
    await scanStore.checkPendingResume();
    expect(scanStore.pendingResume?.current_index).toBe(42);

    await scanStore.resumeSavedScan();
    expect(appMocks.ResumePendingScan).toHaveBeenCalledTimes(1);
    expect(scanStore.sessionID).toBe(99);
    expect(scanStore.pendingResume).toBeNull();

    await scanStore.checkPendingResume();
    expect(scanStore.pendingResume).not.toBeNull();
    await scanStore.discardSavedScan();
    expect(appMocks.DiscardPendingResumeState).toHaveBeenCalledTimes(1);
    expect(scanStore.pendingResume).toBeNull();
  });

  it('forwards pre-scan status != exclusion rules from filtersStore into StartScan filters', async () => {
    filtersStore.clearFilters();
    filtersStore.addRule({
      field: 'status',
      operator: '!=',
      value: '404',
      enabled: true,
      layer: 'both'
    });
    scanStore.url = 'https://api.example.com/FUZZ';
    await scanStore.start();

    expect(appMocks.StartScan).toHaveBeenCalledWith(
      expect.objectContaining({
        filters: '404'
      })
    );
    filtersStore.clearFilters();
  });
});
