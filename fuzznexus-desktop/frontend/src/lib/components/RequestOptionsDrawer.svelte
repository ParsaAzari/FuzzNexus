<script lang="ts">
  import { scanStore } from '../stores/scan.svelte';
  import { wordlistsStore } from '../stores/wordlists.svelte';
  import { toastStore } from '../stores/toast.svelte';
  import Icon from './Icon.svelte';
  import NumberStepper from './NumberStepper.svelte';
  import type { PreflightConfig, VarExtract } from '../types';

  type DrawerTab = 'general' | 'http' | 'inputs' | 'filters' | 'advanced';
  let activeTab = $state<DrawerTab>('general');

  let headerKey = $state<string>('');
  let headerVal = $state<string>('');
  let bodyError = $state<string>('');
  let rawPasteOpen = $state<boolean>(false);
  let rawPasteText = $state<string>('');
  let rawPasteError = $state<string>('');

  // Preflight/Postflight editor form state
  let newFlightKind = $state<'preflight' | 'postflight'>('preflight');
  let newFlightFile = $state<string>('');
  let newVarName = $state<string>('');
  let newVarSource = $state<'regex' | 'auto' | 'form' | 'meta' | 'cookie' | 'header'>('auto');
  let newVarPattern = $state<string>('');

  // FNHASH Search state (-search)
  let searchHashInput = $state<string>('');
  let searchHashOutput = $state<string>('');
  let searchHashError = $state<string>('');

  let headerEntries = $derived(Object.entries(scanStore.headers));
  let isLocked = $derived(scanStore.state === 'Running' || scanStore.state === 'Paused');

  let headerValueSuggestions = $derived.by(() => {
    const k = headerKey.trim().toLowerCase();
    if (k === 'content-type') {
      return ['application/json', 'application/x-www-form-urlencoded', 'multipart/form-data', 'text/plain'];
    }
    if (k === 'authorization') {
      return ['Bearer ', 'Basic '];
    }
    if (k === 'x-forwarded-for' || k === 'x-real-ip') {
      return ['127.0.0.1', 'localhost', '10.0.0.1'];
    }
    if (k === 'accept') {
      return ['application/json, text/plain, */*', '*/*'];
    }
    return [];
  });

  // Tab badge counts so active settings are visible at a glance
  let httpBadge = $derived(
    headerEntries.length +
      (scanStore.data.trim() ? 1 : 0) +
      (scanStore.proxyURL.trim() ? 1 : 0) +
      (scanStore.replayProxyURL.trim() ? 1 : 0) +
      (scanStore.tlsProfile ? 1 : 0) +
      (scanStore.followRedirects ? 1 : 0) +
      (scanStore.rawURI ? 1 : 0) +
      (scanStore.http2 ? 1 : 0) +
      (scanStore.ignoreBody ? 1 : 0) +
      (scanStore.sni.trim() || scanStore.clientCert.trim() ? 1 : 0) +
      (scanStore.requestFile.trim() ? 1 : 0)
  );

  let inputsBadge = $derived(
    scanStore.additionalWordlists.length +
      (scanStore.inputMode !== 'clusterbomb' ? 1 : 0) +
      (scanStore.extensions.trim() ? 1 : 0) +
      (scanStore.primaryEncoders.trim() ? 1 : 0) +
      (scanStore.dirSearchCompat ? 1 : 0) +
      (scanStore.ignoreWordlistComments ? 1 : 0) +
      (scanStore.inputCommand.trim() ? 1 : 0)
  );

  let filtersBadge = $derived(
    (scanStore.matchers.trim() ? 1 : 0) +
      (scanStore.matcherSize.trim() ? 1 : 0) +
      (scanStore.matcherWords.trim() ? 1 : 0) +
      (scanStore.matcherLines.trim() ? 1 : 0) +
      (scanStore.matcherRegexp.trim() ? 1 : 0) +
      (scanStore.matcherTime.trim() ? 1 : 0) +
      (scanStore.filters.trim() ? 1 : 0) +
      (scanStore.filterSize.trim() ? 1 : 0) +
      (scanStore.filterWords.trim() ? 1 : 0) +
      (scanStore.filterLines.trim() ? 1 : 0) +
      (scanStore.filterRegexp.trim() ? 1 : 0) +
      (scanStore.filterTime.trim() ? 1 : 0)
  );

  let advancedBadge = $derived(
    (scanStore.recursion ? 1 : 0) +
      (scanStore.autoCalibrationPerHost ? 1 : 0) +
      (scanStore.stopOn403 || scanStore.stopOnErrors || scanStore.stopOnAll ? 1 : 0) +
      (scanStore.scraperFile.trim() ? 1 : 0) +
      (scanStore.auditLog.trim() || scanStore.outputDirectory.trim() ? 1 : 0) +
      scanStore.preflights.length +
      scanStore.postflights.length
  );

  let ignoreBodyConflict = $derived(
    scanStore.ignoreBody &&
      Boolean(
        scanStore.matcherSize.trim() ||
          scanStore.matcherWords.trim() ||
          scanStore.matcherLines.trim() ||
          scanStore.filterSize.trim() ||
          scanStore.filterWords.trim() ||
          scanStore.filterLines.trim()
      )
  );

  let recursionUrlWarning = $derived(
    scanStore.recursion && scanStore.url.trim() !== '' && !scanStore.url.trim().endsWith('FUZZ')
  );

  $effect(() => {
    if (scanStore.isRequestOptionsOpen) {
      void scanStore.loadAvailableEncoders();
    }
  });

  function addHeader() {
    if (!headerKey.trim()) return;
    scanStore.setHeader(headerKey.trim(), headerVal);
    headerKey = '';
    headerVal = '';
  }

  function applyHeaderPreset(key: string, defaultValue: string) {
    headerKey = key;
    headerVal = defaultValue;
  }

  function formatJSONBody() {
    bodyError = '';
    if (!scanStore.data.trim()) return;
    try {
      const parsed = JSON.parse(scanStore.data);
      scanStore.data = JSON.stringify(parsed, null, 2);
    } catch {
      bodyError = 'Body is not strict JSON (tip: if using unquoted FUZZ, format manually).';
    }
  }

  async function handleParseRawRequest() {
    rawPasteError = '';
    try {
      await scanStore.parseAndApplyRawRequest(rawPasteText, scanStore.requestProto);
      rawPasteOpen = false;
      rawPasteText = '';
      toastStore.success('Parsed raw HTTP request into Target, Method, Headers & Body');
    } catch (err: unknown) {
      rawPasteError = err instanceof Error ? err.message : String(err);
    }
  }

  function appendEncoderToken(token: string) {
    const cur = scanStore.primaryEncoders.trim();
    scanStore.primaryEncoders = cur ? `${cur} ${token}` : token;
  }

  async function addAdditionalWordlist() {
    if (isLocked) return;
    const selected = await wordlistsStore.openNativeFilePicker('link');
    if (!selected) return;
    const nextIdx = scanStore.additionalWordlists.length + 2;
    scanStore.additionalWordlists = [
      ...scanStore.additionalWordlists,
      { path: selected, keyword: `W${nextIdx}`, encoders: '' }
    ];
  }

  function removeAdditionalWordlist(index: number) {
    scanStore.additionalWordlists = scanStore.additionalWordlists.filter((_, i) => i !== index);
  }

  function toggleCalibrationStrategy(strategy: string) {
    const has = scanStore.autoCalibrationStrategies.includes(strategy);
    if (has) {
      const next = scanStore.autoCalibrationStrategies.filter((s) => s !== strategy);
      scanStore.autoCalibrationStrategies = next.length > 0 ? next : ['basic'];
    } else {
      scanStore.autoCalibrationStrategies = [...scanStore.autoCalibrationStrategies, strategy];
    }
  }

  function addFlightRule() {
    if (!newFlightFile.trim()) {
      toastStore.warning('Select a request file for the flight step');
      return;
    }
    const vars: VarExtract[] = [];
    if (newVarName.trim() && newVarPattern.trim()) {
      if (newVarSource === 'regex') {
        vars.push({ name: newVarName.trim(), regex: newVarPattern.trim() });
      } else {
        vars.push({
          name: newVarName.trim(),
          regex: '',
          source: newVarSource,
          key: newVarPattern.trim()
        });
      }
    }
    const entry = {
      request_file: newFlightFile.trim(),
      vars
    } as PreflightConfig;
    if (newFlightKind === 'preflight') {
      scanStore.preflights = [...scanStore.preflights, entry];
    } else {
      scanStore.postflights = [...scanStore.postflights, entry];
    }
    newFlightFile = '';
    newVarName = '';
    newVarPattern = '';
  }

  function applyQuickPreset(preset: 'safe' | 'waf' | 'proxy' | 'json-api') {
    if (isLocked) return;
    if (preset === 'safe') {
      scanStore.autoCalibration = true;
      scanStore.followRedirects = false;
      scanStore.tlsProfile = '';
      scanStore.proxyURL = '';
      scanStore.browserProfile = 'auto';
    } else if (preset === 'waf') {
      scanStore.autoCalibration = true;
      scanStore.followRedirects = false;
      scanStore.tlsProfile = 'chrome';
      scanStore.browserProfile = 'chrome-macos';
    } else if (preset === 'proxy') {
      scanStore.autoCalibration = true;
      scanStore.replayProxyURL = 'http://127.0.0.1:8080';
    } else if (preset === 'json-api') {
      scanStore.method = 'POST';
      scanStore.autoCalibration = true;
      scanStore.setHeader('Content-Type', 'application/json');
      if (!scanStore.data.trim()) {
        scanStore.data = '{\n  "param": "FUZZ"\n}';
      }
    }
  }

  function resetMatchersAndFilters() {
    if (isLocked) return;
    scanStore.matchers = '';
    scanStore.matcherSize = '';
    scanStore.matcherWords = '';
    scanStore.matcherLines = '';
    scanStore.matcherRegexp = '';
    scanStore.matcherTime = '';
    scanStore.matcherMode = 'or';
    scanStore.filters = '';
    scanStore.filterSize = '';
    scanStore.filterWords = '';
    scanStore.filterLines = '';
    scanStore.filterRegexp = '';
    scanStore.filterTime = '';
    scanStore.filterMode = 'or';
  }

  function close() {
    scanStore.isRequestOptionsOpen = false;
  }
</script>

{#if scanStore.isRequestOptionsOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="drawer-backdrop"
    onclick={(e) => e.target === e.currentTarget && close()}
    role="dialog"
    aria-modal="true"
    aria-label="Request and Engine Options"
    tabindex="-1"
  >
    <div class="drawer-panel">
      <div class="drawer-header">
        <div class="drawer-title-group">
          <Icon name="sliders" size={16} />
          <h2>Request &amp; Engine Options</h2>
        </div>
        <button
          type="button"
          onclick={close}
          class="drawer-close-btn"
          aria-label="Close options"
        >
          <Icon name="x" size={16} />
        </button>
      </div>

      <!-- Drawer Navigation Tabs -->
      <div class="drawer-tabs select-none" role="tablist">
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'general'}
          class="drawer-tab"
          class:active={activeTab === 'general'}
          onclick={() => (activeTab = 'general')}
        >
          <span>Engine</span>
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'http'}
          class="drawer-tab"
          class:active={activeTab === 'http'}
          onclick={() => (activeTab = 'http')}
        >
          <span>HTTP &amp; TLS</span>
          {#if httpBadge > 0}<span class="tab-pill font-mono">{httpBadge}</span>{/if}
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'inputs'}
          class="drawer-tab"
          class:active={activeTab === 'inputs'}
          onclick={() => (activeTab = 'inputs')}
        >
          <span>Inputs</span>
          {#if inputsBadge > 0}<span class="tab-pill font-mono">{inputsBadge}</span>{/if}
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'filters'}
          class="drawer-tab"
          class:active={activeTab === 'filters'}
          onclick={() => (activeTab = 'filters')}
        >
          <span>Match / Filter</span>
          {#if filtersBadge > 0}<span class="tab-pill font-mono">{filtersBadge}</span>{/if}
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'advanced'}
          class="drawer-tab"
          class:active={activeTab === 'advanced'}
          onclick={() => (activeTab = 'advanced')}
        >
          <span>Pro &amp; Logs</span>
          {#if advancedBadge > 0}<span class="tab-pill font-mono">{advancedBadge}</span>{/if}
        </button>
      </div>

      <div class="drawer-body">
        {#if isLocked}
          <div class="locked-notice">
            Scan is currently active. Stop the scan to modify static request, input, or engine flags. Rate limit can still be changed live.
          </div>
        {/if}

        <!-- TAB 1: ENGINE & TIMING -->
        {#if activeTab === 'general'}
          <section class="drawer-section">
            <div class="field-label-row">
              <h3 class="section-heading">Quick Presets (1-Click Setup)</h3>
              <button
                type="button"
                class="preset-chip"
                disabled={isLocked}
                onclick={async () => {
                  try {
                    const p = await scanStore.pickAuxiliaryFile('Import FuzzNexus TOML Config (.fuzznexusrc)', 'Config Files (*.toml, *.fuzznexusrc, *.conf)', '*.toml;*.fuzznexusrc;*.conf');
                    if (!p) return;
                    await scanStore.importTOMLConfig(p);
                    toastStore.success('Imported CLI .fuzznexusrc configuration');
                  } catch (err: unknown) {
                    toastStore.error(err instanceof Error ? err.message : String(err));
                  }
                }}
              >
                Import .fuzznexusrc (-config)
              </button>
            </div>
            <div class="quick-presets-grid">
              <button
                type="button"
                disabled={isLocked}
                onclick={() => applyQuickPreset('safe')}
                class="quick-preset-card"
              >
                <span class="qp-title">Auto Safe</span>
                <span class="qp-sub">Soft-404 ON, Auto-Rotate</span>
              </button>
              <button
                type="button"
                disabled={isLocked}
                onclick={() => applyQuickPreset('waf')}
                class="quick-preset-card"
              >
                <span class="qp-title">Strict WAF / SPA</span>
                <span class="qp-sub">Chrome uTLS + Soft-404</span>
              </button>
              <button
                type="button"
                disabled={isLocked}
                onclick={() => applyQuickPreset('proxy')}
                class="quick-preset-card"
              >
                <span class="qp-title">Replay Matches :8080</span>
                <span class="qp-sub">Send hits only to Burp/Caido</span>
              </button>
              <button
                type="button"
                disabled={isLocked}
                onclick={() => applyQuickPreset('json-api')}
                class="quick-preset-card"
              >
                <span class="qp-title">JSON POST API</span>
                <span class="qp-sub">POST + JSON body FUZZ</span>
              </button>
            </div>
          </section>

          <section class="drawer-section">
            <h3 class="section-heading">Concurrency &amp; Rate</h3>
            <div class="card-group">
              <div class="concurrency-steppers-row">
                <div class="stepper-item">
                  <div class="field-label-sub">Worker Threads (-t)</div>
                  <NumberStepper
                    label="Threads"
                    bind:value={scanStore.threads}
                    min={1}
                    max={200}
                    step={1}
                    shiftStep={5}
                    disabled={isLocked}
                    width="100%"
                    title={isLocked
                      ? "Concurrent worker threads (1-200). Stop scan to modify."
                      : "Concurrent worker threads (1-200)."}
                    ariaLabel="Scan Threads"
                  />
                </div>

                <div class="stepper-item">
                  <div class="field-label-sub">Rate Limit (-rate req/s)</div>
                  <NumberStepper
                    label="Rate (req/s)"
                    bind:value={scanStore.rate}
                    min={0}
                    max={5000}
                    step={10}
                    shiftStep={50}
                    width="100%"
                    onchange={(val) => scanStore.changeRate(val)}
                    title="0 = unlimited. Can be adjusted live during scan."
                    ariaLabel="Rate Limit"
                  />
                </div>
              </div>
            </div>
          </section>

          <section class="drawer-section">
            <h3 class="section-heading">Timeouts, Jitter Delay &amp; Time Limits</h3>
            <div class="card-group">
              <div class="two-col-grid">
                <div class="field-block">
                  <label for="timeout-input" class="field-label-sub">HTTP Timeout (-timeout sec)</label>
                  <input
                    id="timeout-input"
                    type="number"
                    min={1}
                    max={300}
                    bind:value={scanStore.timeout}
                    disabled={isLocked}
                    class="native-input font-mono"
                  />
                </div>
                <div class="field-block">
                  <label for="delay-input" class="field-label-sub">Request Delay / Jitter (-p)</label>
                  <input
                    id="delay-input"
                    type="text"
                    bind:value={scanStore.delay}
                    disabled={isLocked}
                    placeholder="e.g. 0.1 or 0.2-1.5"
                    data-tooltip="Fixed delay in seconds or random range min-max between requests"
                    class="native-input font-mono"
                  />
                </div>
                <div class="field-block">
                  <label for="maxtime-input" class="field-label-sub">Max Scan Time (-maxtime sec)</label>
                  <input
                    id="maxtime-input"
                    type="number"
                    min={0}
                    bind:value={scanStore.maxTime}
                    disabled={isLocked}
                    placeholder="0 = unlimited"
                    class="native-input font-mono"
                  />
                </div>
                <div class="field-block">
                  <label for="maxtime-job-input" class="field-label-sub">Max Job Time (-maxtime-job sec)</label>
                  <input
                    id="maxtime-job-input"
                    type="number"
                    min={0}
                    bind:value={scanStore.maxTimeJob}
                    disabled={isLocked}
                    placeholder="0 = unlimited"
                    class="native-input font-mono"
                  />
                </div>
              </div>
            </div>
          </section>
        {/if}

        <!-- TAB 2: HTTP, PROXY & TLS -->
        {#if activeTab === 'http'}
          <section class="drawer-section">
            <h3 class="section-heading">HTTP Transport &amp; Protocol</h3>
            <div class="card-group">
              <label class="toggle-row">
                <div class="toggle-info">
                  <div class="toggle-title">Follow HTTP Redirects (-r)</div>
                  <div class="toggle-desc">Automatically follow 301/302/307/308 redirects during fuzzing.</div>
                </div>
                <input type="checkbox" bind:checked={scanStore.followRedirects} disabled={isLocked} class="native-checkbox" />
              </label>

              <label class="toggle-row with-divider">
                <div class="toggle-info">
                  <div class="toggle-title">Raw URI / Do Not Encode (-raw)</div>
                  <div class="toggle-desc">Preserves raw payloads (slashes, ../, special chars) without URL encoding.</div>
                </div>
                <input type="checkbox" bind:checked={scanStore.rawURI} disabled={isLocked} class="native-checkbox" />
              </label>

              <label class="toggle-row with-divider">
                <div class="toggle-info">
                  <div class="toggle-title">Force HTTP/2 (-http2)</div>
                  <div class="toggle-desc">Prefer HTTP/2 transport when using standard Go TLS client.</div>
                </div>
                <input type="checkbox" bind:checked={scanStore.http2} disabled={isLocked} class="native-checkbox" />
              </label>

              <label class="toggle-row with-divider">
                <div class="toggle-info">
                  <div class="toggle-title">Ignore Response Body (-ignore-body)</div>
                  <div class="toggle-desc">Skips downloading response bodies for maximum speed when filtering by status only.</div>
                </div>
                <input type="checkbox" bind:checked={scanStore.ignoreBody} disabled={isLocked} class="native-checkbox" />
              </label>

              {#if ignoreBodyConflict}
                <div class="warn-banner">
                  <Icon name="alert-triangle" size={13} />
                  <span>Warning: -ignore-body is active alongside size/word/line matchers or filters.</span>
                </div>
              {/if}
            </div>
          </section>

          <section class="drawer-section">
            <h3 class="section-heading">Upstream &amp; Replay Proxies</h3>
            <div class="card-group">
              <div class="field-block">
                <div class="field-label-row">
                  <label for="proxy-url-input" class="toggle-title">Upstream Proxy (-x)</label>
                  <div class="preset-btns">
                    <button type="button" disabled={isLocked} onclick={() => (scanStore.proxyURL = 'http://127.0.0.1:8080')} class="preset-chip">Burp/Caido :8080</button>
                    <button type="button" disabled={isLocked} onclick={() => (scanStore.proxyURL = 'socks5://127.0.0.1:9050')} class="preset-chip">Tor :9050</button>
                    {#if scanStore.proxyURL}
                      <button type="button" disabled={isLocked} onclick={() => (scanStore.proxyURL = '')} class="preset-chip danger">Clear</button>
                    {/if}
                  </div>
                </div>
                <input
                  id="proxy-url-input"
                  type="text"
                  bind:value={scanStore.proxyURL}
                  disabled={isLocked}
                  placeholder="http://127.0.0.1:8080 or socks5://127.0.0.1:9050"
                  class="native-input font-mono"
                />
              </div>

              <div class="field-block with-divider">
                <div class="field-label-row">
                  <label for="replay-proxy-input" class="toggle-title">Replay Matched Requests Proxy (-replay-proxy)</label>
                  <div class="preset-btns">
                    <button type="button" disabled={isLocked} onclick={() => (scanStore.replayProxyURL = 'http://127.0.0.1:8080')} class="preset-chip">Burp/Caido :8080</button>
                    {#if scanStore.replayProxyURL}
                      <button type="button" disabled={isLocked} onclick={() => (scanStore.replayProxyURL = '')} class="preset-chip danger">Clear</button>
                    {/if}
                  </div>
                </div>
                <input
                  id="replay-proxy-input"
                  type="text"
                  bind:value={scanStore.replayProxyURL}
                  disabled={isLocked}
                  placeholder="http://127.0.0.1:8080 (forwards matched hits only)"
                  class="native-input font-mono"
                />
              </div>
            </div>
          </section>

          <section class="drawer-section">
            <h3 class="section-heading">TLS Fingerprinting, SNI &amp; Client Auth (mTLS)</h3>
            <div class="card-group">
              <div class="field-block">
                <label for="tls-profile-select" class="toggle-title">TLS Fingerprint (uTLS ClientHello)</label>
                <select id="tls-profile-select" bind:value={scanStore.tlsProfile} disabled={isLocked} class="native-input">
                  <option value="">Default (Standard Go TLS)</option>
                  <option value="chrome">Chrome 120+ (uTLS Browser JA3/JA4 + HTTP/2)</option>
                  <option value="firefox">Firefox 120+ (uTLS Browser JA3/JA4 + HTTP/2)</option>
                </select>
              </div>

              <div class="field-block with-divider">
                <label for="sni-input" class="toggle-title">Custom TLS SNI (-sni)</label>
                <input
                  id="sni-input"
                  type="text"
                  bind:value={scanStore.sni}
                  disabled={isLocked}
                  placeholder="e.g. cdn-origin.example.com"
                  class="native-input font-mono"
                />
              </div>

              <div class="two-col-grid with-divider">
                <div class="field-block">
                  <label for="cc-input" class="field-label-sub">Client Certificate (-cc)</label>
                  <div class="path-picker-row">
                    <input id="cc-input" type="text" bind:value={scanStore.clientCert} disabled={isLocked} placeholder="/path/client.crt" class="native-input font-mono" />
                    <button
                      type="button"
                      class="preset-chip"
                      disabled={isLocked}
                      onclick={async () => {
                        const p = await scanStore.pickAuxiliaryFile('Select Client Certificate (-cc)', 'Certificates (*.crt, *.pem)', '*.crt;*.pem;*.cer');
                        if (p) scanStore.clientCert = p;
                      }}
                    >Browse</button>
                  </div>
                </div>
                <div class="field-block">
                  <label for="ck-input" class="field-label-sub">Client Private Key (-ck)</label>
                  <div class="path-picker-row">
                    <input id="ck-input" type="text" bind:value={scanStore.clientKey} disabled={isLocked} placeholder="/path/client.key" class="native-input font-mono" />
                    <button
                      type="button"
                      class="preset-chip"
                      disabled={isLocked}
                      onclick={async () => {
                        const p = await scanStore.pickAuxiliaryFile('Select Client Private Key (-ck)', 'Key Files (*.key, *.pem)', '*.key;*.pem');
                        if (p) scanStore.clientKey = p;
                      }}
                    >Browse</button>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <section class="drawer-section">
            <div class="field-label-row">
              <h3 class="section-heading">Custom Headers &amp; Cookies ({headerEntries.length})</h3>
              <div class="preset-btns">
                <button type="button" disabled={isLocked} onclick={() => applyHeaderPreset('Authorization', 'Bearer ')} class="preset-chip">+ Bearer</button>
                <button type="button" disabled={isLocked} onclick={() => applyHeaderPreset('Cookie', 'session=')} class="preset-chip">+ Cookie</button>
                <button type="button" disabled={isLocked} onclick={() => applyHeaderPreset('Content-Type', 'application/json')} class="preset-chip">+ JSON</button>
                <button type="button" disabled={isLocked} onclick={() => applyHeaderPreset('X-Forwarded-For', '127.0.0.1')} class="preset-chip">+ XFF</button>
              </div>
            </div>
            <div class="card-group">
              <div class="header-add-row">
                <input
                  type="text"
                  list="header-name-suggestions"
                  bind:value={headerKey}
                  disabled={isLocked}
                  placeholder="Header (e.g. Authorization)"
                  class="native-input key-input font-mono"
                />
                <datalist id="header-name-suggestions">
                  <option value="Authorization"></option>
                  <option value="Cookie"></option>
                  <option value="Content-Type"></option>
                  <option value="X-Forwarded-For"></option>
                  <option value="X-Real-IP"></option>
                  <option value="X-Api-Key"></option>
                  <option value="Origin"></option>
                  <option value="Referer"></option>
                  <option value="User-Agent"></option>
                  <option value="X-Requested-With"></option>
                </datalist>
                <input
                  type="text"
                  list="header-value-suggestions"
                  bind:value={headerVal}
                  disabled={isLocked}
                  placeholder="Value (supports FUZZ)"
                  onkeydown={(e) => e.key === 'Enter' && addHeader()}
                  class="native-input val-input font-mono"
                />
                <datalist id="header-value-suggestions">
                  {#each headerValueSuggestions as val (val)}
                    <option value={val}></option>
                  {/each}
                </datalist>
                <button type="button" disabled={isLocked || !headerKey.trim()} onclick={addHeader} class="btn-primary-sm">Add</button>
              </div>

              {#if headerEntries.length > 0}
                <div class="headers-list">
                  {#each headerEntries as [k, v] (k)}
                    <div class="header-item">
                      <div class="header-item-text font-mono">
                        <span class="header-key">{k}</span>
                        <span class="header-sep">: </span>
                        <span class="header-val">{v}</span>
                      </div>
                      <button type="button" disabled={isLocked} onclick={() => scanStore.removeHeader(k)} class="btn-remove">Remove</button>
                    </div>
                  {/each}
                </div>
              {:else}
                <p class="empty-hint">No custom headers configured.</p>
              {/if}
            </div>
          </section>

          <section class="drawer-section">
            <div class="field-label-row">
              <h3 class="section-heading">Request Body &amp; Raw HTTP Import (-d / -request)</h3>
              <div class="preset-btns">
                <button type="button" disabled={isLocked} onclick={() => (rawPasteOpen = !rawPasteOpen)} class="preset-chip">
                  {rawPasteOpen ? 'Hide Raw Paste' : 'Paste Raw HTTP'}
                </button>
                <button type="button" disabled={isLocked || !scanStore.data.trim()} onclick={formatJSONBody} class="preset-chip">Format JSON</button>
              </div>
            </div>

            {#if rawPasteOpen}
              <div class="card-group">
                <div class="field-label-row">
                  <span class="field-label-sub">Paste Burp / Caido Raw HTTP Request</span>
                  <select bind:value={scanStore.requestProto} disabled={isLocked} class="mini-select font-mono">
                    <option value="https">HTTPS</option>
                    <option value="http">HTTP</option>
                  </select>
                </div>
                <textarea
                  rows={6}
                  bind:value={rawPasteText}
                  disabled={isLocked}
                  placeholder={"POST /api/login HTTP/1.1\nHost: target.com\nContent-Type: application/json\n\n{\"user\":\"FUZZ\"}"}
                  class="native-textarea font-mono"
                ></textarea>
                {#if rawPasteError}<p class="warn-hint">{rawPasteError}</p>{/if}
                <div class="field-label-row">
                  <span class="empty-hint">Populates Method, URL, Headers &amp; Body</span>
                  <button type="button" disabled={isLocked || !rawPasteText.trim()} onclick={handleParseRawRequest} class="btn-primary-sm">Parse &amp; Apply</button>
                </div>
              </div>
            {/if}

            <textarea
              rows={4}
              bind:value={scanStore.data}
              disabled={isLocked}
              placeholder={'{"username":"admin","param":"FUZZ"}'}
              class="native-textarea font-mono"
            ></textarea>
            {#if bodyError}<p class="warn-hint">{bodyError}</p>{/if}

            <div class="card-group">
              <div class="field-label-row">
                <span class="field-label-sub">Raw Request File (-request / -request-proto)</span>
                <select bind:value={scanStore.requestProto} disabled={isLocked} class="mini-select font-mono">
                  <option value="https">https</option>
                  <option value="http">http</option>
                </select>
              </div>
              <div class="path-picker-row">
                <input type="text" bind:value={scanStore.requestFile} disabled={isLocked} placeholder="Optional .req file containing raw HTTP request" class="native-input font-mono" />
                <button
                  type="button"
                  class="preset-chip"
                  disabled={isLocked}
                  onclick={async () => {
                    const p = await scanStore.pickAuxiliaryFile('Select Raw HTTP Request File (-request)', 'Request Files (*.req, *.txt)', '*.req;*.txt');
                    if (p) scanStore.requestFile = p;
                  }}
                >Browse</button>
                {#if scanStore.requestFile}
                  <button type="button" class="preset-chip danger" disabled={isLocked} onclick={() => (scanStore.requestFile = '')}>Clear</button>
                {/if}
              </div>
            </div>
          </section>
        {/if}

        <!-- TAB 3: INPUTS, MODES & ENCODERS -->
        {#if activeTab === 'inputs'}
          <section class="drawer-section">
            <h3 class="section-heading">Attack Mode &amp; Multi-Wordlist (-mode / -w)</h3>
            <div class="card-group">
              <div class="field-block">
                <label for="input-mode-select" class="toggle-title">Operation Mode (-mode)</label>
                <select id="input-mode-select" bind:value={scanStore.inputMode} disabled={isLocked} class="native-input font-mono">
                  <option value="clusterbomb">Clusterbomb (All combinations across wordlists)</option>
                  <option value="pitchfork">Pitchfork (Line-by-line lockstep zip across wordlists)</option>
                  <option value="sniper">Sniper (Single wordlist across §marker§ positions)</option>
                </select>
                {#if scanStore.inputMode === 'sniper'}
                  <p class="empty-hint">Sniper mode replaces paired <code class="inline-code">§token§</code> markers one at a time using the primary wordlist (do not use FUZZ).</p>
                {/if}
              </div>

              <div class="two-col-grid with-divider">
                <div class="field-block">
                  <label for="primary-kw-input" class="field-label-sub">Primary Wordlist Keyword</label>
                  <input
                    id="primary-kw-input"
                    type="text"
                    bind:value={scanStore.primaryKeyword}
                    disabled={isLocked || scanStore.inputMode === 'sniper'}
                    placeholder="FUZZ"
                    class="native-input font-mono"
                  />
                </div>
                <div class="field-block">
                  <label for="primary-enc-input" class="field-label-sub">Encoder Chain (-enc)</label>
                  <input
                    id="primary-enc-input"
                    type="text"
                    bind:value={scanStore.primaryEncoders}
                    disabled={isLocked}
                    placeholder="e.g. urlencode b64encode"
                    class="native-input font-mono"
                  />
                </div>
              </div>

              {#if scanStore.availableEncoders.length > 0}
                <div class="encoder-chips-wrap">
                  <span class="empty-hint">Click to append encoder:</span>
                  <div class="preset-btns">
                    {#each scanStore.availableEncoders.slice(0, 12) as enc}
                      <button type="button" disabled={isLocked} onclick={() => appendEncoderToken(enc)} class="preset-chip font-mono">+{enc}</button>
                    {/each}
                    {#if scanStore.primaryEncoders}
                      <button type="button" disabled={isLocked} onclick={() => (scanStore.primaryEncoders = '')} class="preset-chip danger">Clear</button>
                    {/if}
                  </div>
                </div>
              {/if}

              {#if scanStore.inputMode !== 'sniper'}
                <div class="field-block with-divider">
                  <div class="field-label-row">
                    <span class="field-label-sub">Additional Wordlists ({scanStore.additionalWordlists.length})</span>
                    <button type="button" disabled={isLocked} onclick={addAdditionalWordlist} class="preset-chip">+ Add Wordlist</button>
                  </div>
                  {#each scanStore.additionalWordlists as wl, idx (idx)}
                    <div class="extra-wl-row">
                      <input type="text" bind:value={wl.keyword} disabled={isLocked} placeholder="KEYWORD" class="native-input font-mono wl-kw-input" />
                      <input type="text" bind:value={wl.path} disabled={isLocked} placeholder="/path/to/wordlist.txt" class="native-input font-mono wl-path-input" />
                      <input type="text" bind:value={wl.encoders} disabled={isLocked} placeholder="encoders..." class="native-input font-mono wl-enc-input" />
                      <button type="button" disabled={isLocked} onclick={() => removeAdditionalWordlist(idx)} class="btn-remove">x</button>
                    </div>
                  {:else}
                    <p class="empty-hint">Single wordlist active. Add extra wordlists for Clusterbomb or Pitchfork fuzzing.</p>
                  {/each}
                </div>
              {/if}
            </div>
          </section>

          <section class="drawer-section">
            <div class="field-label-row">
              <h3 class="section-heading">Extensions &amp; Wordlist Behavior</h3>
              <div class="preset-btns">
                <button type="button" disabled={isLocked} onclick={() => (scanStore.extensions = '.php,.html,.txt,.bak')} class="preset-chip">+ Web</button>
                <button type="button" disabled={isLocked} onclick={() => (scanStore.extensions = '.json,.yaml,.xml,.env')} class="preset-chip">+ Config/API</button>
                {#if scanStore.extensions}
                  <button type="button" disabled={isLocked} onclick={() => (scanStore.extensions = '')} class="preset-chip danger">Clear</button>
                {/if}
              </div>
            </div>
            <div class="card-group">
              <div class="field-block">
                <label for="ext-input" class="toggle-title">File Extensions (-e, comma separated)</label>
                <input
                  id="ext-input"
                  type="text"
                  bind:value={scanStore.extensions}
                  disabled={isLocked}
                  placeholder=".php,.json,.bak,.old"
                  class="native-input font-mono"
                />
              </div>

              <label class="toggle-row with-divider">
                <div class="toggle-info">
                  <div class="toggle-title">DirSearch Compatibility Mode (-D)</div>
                  <div class="toggle-desc">Replaces <code class="inline-code">%EXT%</code> placeholders inside wordlists with configured extensions.</div>
                </div>
                <input type="checkbox" bind:checked={scanStore.dirSearchCompat} disabled={isLocked} class="native-checkbox" />
              </label>

              <label class="toggle-row with-divider">
                <div class="toggle-info">
                  <div class="toggle-title">Ignore Wordlist Comments (-ic)</div>
                  <div class="toggle-desc">Skips lines starting with <code class="inline-code">#</code> in raw wordlist files.</div>
                </div>
                <input type="checkbox" bind:checked={scanStore.ignoreWordlistComments} disabled={isLocked} class="native-checkbox" />
              </label>
            </div>
          </section>

          <section class="drawer-section">
            <h3 class="section-heading">Command Input Generator (--input-cmd)</h3>
            <div class="card-group">
              <div class="field-block">
                <label for="input-cmd-input" class="field-label-sub">Shell Command (overrides -w when non-empty)</label>
                <input
                  id="input-cmd-input"
                  type="text"
                  bind:value={scanStore.inputCommand}
                  disabled={isLocked}
                  placeholder="e.g. seq -w 1 100"
                  class="native-input font-mono"
                />
              </div>
              <div class="two-col-grid">
                <div class="field-block">
                  <label for="input-num-input" class="field-label-sub">Input Count (--input-num)</label>
                  <input id="input-num-input" type="number" min={1} bind:value={scanStore.inputNum} disabled={isLocked} class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="input-shell-input" class="field-label-sub">Custom Shell (--input-shell)</label>
                  <input id="input-shell-input" type="text" bind:value={scanStore.inputShell} disabled={isLocked} placeholder="Default (/bin/sh or cmd)" class="native-input font-mono" />
                </div>
              </div>
            </div>
          </section>
        {/if}

        <!-- TAB 4: ENGINE MATCHERS & PRE-SCAN FILTERS -->
        {#if activeTab === 'filters'}
          <section class="drawer-section">
            <div class="field-label-row">
              <h3 class="section-heading">Engine Matchers (Accept Criteria)</h3>
              <div class="preset-btns">
                <button type="button" disabled={isLocked} onclick={() => (scanStore.matchers = 'all')} class="preset-chip">Match All (-mc all)</button>
                <button type="button" disabled={isLocked} onclick={resetMatchersAndFilters} class="preset-chip danger">Reset All</button>
              </div>
            </div>
            <div class="card-group">
              <div class="field-label-row">
                <span class="field-label-sub">Matcher Set Operator (-mmode)</span>
                <select bind:value={scanStore.matcherMode} disabled={isLocked} class="mini-select font-mono">
                  <option value="or">OR (Match any)</option>
                  <option value="and">AND (Match all)</option>
                </select>
              </div>
              <div class="two-col-grid">
                <div class="field-block">
                  <label for="mc-input" class="field-label-sub">Status Codes (-mc)</label>
                  <input id="mc-input" type="text" bind:value={scanStore.matchers} disabled={isLocked} placeholder="200-299,301,302,401,403,500" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="ms-input" class="field-label-sub">Response Size (-ms)</label>
                  <input id="ms-input" type="text" bind:value={scanStore.matcherSize} disabled={isLocked} placeholder="e.g. 100-500,1024" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="mw-input" class="field-label-sub">Word Count (-mw)</label>
                  <input id="mw-input" type="text" bind:value={scanStore.matcherWords} disabled={isLocked} placeholder="e.g. 42,100-200" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="ml-input" class="field-label-sub">Line Count (-ml)</label>
                  <input id="ml-input" type="text" bind:value={scanStore.matcherLines} disabled={isLocked} placeholder="e.g. 10-50" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="mr-input" class="field-label-sub">Body/Header Regexp (-mr)</label>
                  <input id="mr-input" type="text" bind:value={scanStore.matcherRegexp} disabled={isLocked} placeholder="e.g. Welcome|Admin" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="mt-input" class="field-label-sub">TTFB Time ms (-mt)</label>
                  <input id="mt-input" type="text" bind:value={scanStore.matcherTime} disabled={isLocked} placeholder="e.g. >500 or <50" class="native-input font-mono" />
                </div>
              </div>
            </div>
          </section>

          <section class="drawer-section">
            <h3 class="section-heading">Engine Pre-Scan Filters (Drop Before Saving)</h3>
            <div class="card-group">
              <div class="field-label-row">
                <span class="field-label-sub">Filter Set Operator (-fmode)</span>
                <select bind:value={scanStore.filterMode} disabled={isLocked} class="mini-select font-mono">
                  <option value="or">OR (Filter any)</option>
                  <option value="and">AND (Filter all)</option>
                </select>
              </div>
              <div class="two-col-grid">
                <div class="field-block">
                  <label for="fc-input" class="field-label-sub">Filter Status (-fc)</label>
                  <input id="fc-input" type="text" bind:value={scanStore.filters} disabled={isLocked} placeholder="e.g. 404,403" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="fs-input" class="field-label-sub">Filter Size (-fs)</label>
                  <input id="fs-input" type="text" bind:value={scanStore.filterSize} disabled={isLocked} placeholder="e.g. 0,1245" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="fw-input" class="field-label-sub">Filter Words (-fw)</label>
                  <input id="fw-input" type="text" bind:value={scanStore.filterWords} disabled={isLocked} placeholder="e.g. 12,150-160" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="fl-input" class="field-label-sub">Filter Lines (-fl)</label>
                  <input id="fl-input" type="text" bind:value={scanStore.filterLines} disabled={isLocked} placeholder="e.g. 25" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="fr-input" class="field-label-sub">Filter Regexp (-fr)</label>
                  <input id="fr-input" type="text" bind:value={scanStore.filterRegexp} disabled={isLocked} placeholder="e.g. Page Not Found" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="ft-input" class="field-label-sub">Filter Time ms (-ft)</label>
                  <input id="ft-input" type="text" bind:value={scanStore.filterTime} disabled={isLocked} placeholder="e.g. >2000" class="native-input font-mono" />
                </div>
              </div>
            </div>
          </section>
        {/if}

        <!-- TAB 5: RECURSION, SAFETY, CALIBRATION, PREFLIGHTS & LOGS -->
        {#if activeTab === 'advanced'}
          <section class="drawer-section">
            <h3 class="section-heading">Recursive Discovery (-recursion)</h3>
            <div class="card-group">
              <label class="toggle-row">
                <div class="toggle-info">
                  <div class="toggle-title">Enable Recursive Scanning (-recursion)</div>
                  <div class="toggle-desc">Automatically queues deeper paths when directories are discovered. Target URL must end with <code class="inline-code">FUZZ</code>.</div>
                </div>
                <input type="checkbox" bind:checked={scanStore.recursion} disabled={isLocked} class="native-checkbox" />
              </label>

              {#if recursionUrlWarning}
                <div class="warn-banner">
                  <Icon name="alert-triangle" size={13} />
                  <span>Target URL must end with FUZZ for recursion.</span>
                  <button
                    type="button"
                    class="preset-chip"
                    onclick={() => {
                      const base = scanStore.url.trim().replace(/\/+$/, '');
                      scanStore.url = `${base}/FUZZ`;
                    }}
                  >Fix URL</button>
                </div>
              {/if}

              <div class="two-col-grid with-divider">
                <div class="field-block">
                  <label for="rec-depth" class="field-label-sub">Max Depth (-recursion-depth)</label>
                  <input id="rec-depth" type="number" min={0} max={20} bind:value={scanStore.recursionDepth} disabled={isLocked || !scanStore.recursion} placeholder="0 = unlimited" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="rec-strat" class="field-label-sub">Strategy (-recursion-strategy)</label>
                  <select id="rec-strat" bind:value={scanStore.recursionStrategy} disabled={isLocked || !scanStore.recursion} class="native-input font-mono">
                    <option value="default">default (Redirect-based directories)</option>
                    <option value="greedy">greedy (Recurse on all matches)</option>
                  </select>
                </div>
              </div>
            </div>
          </section>

          <section class="drawer-section">
            <h3 class="section-heading">Auto-Calibration &amp; Engine Stop Guards</h3>
            <div class="card-group">
              <div class="two-col-grid">
                <div class="field-block">
                  <label for="ack-input" class="field-label-sub">Calibration Keyword (-ack)</label>
                  <input id="ack-input" type="text" bind:value={scanStore.autoCalibrationKeyword} disabled={isLocked} placeholder="FUZZ" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <span class="field-label-sub">Strategies (-acs)</span>
                  <div class="preset-btns" style="margin-top: 4px;">
                    <label class="inline-check">
                      <input type="checkbox" checked={scanStore.autoCalibrationStrategies.includes('basic')} onchange={() => toggleCalibrationStrategy('basic')} disabled={isLocked} />
                      <span>basic</span>
                    </label>
                    <label class="inline-check">
                      <input type="checkbox" checked={scanStore.autoCalibrationStrategies.includes('advanced')} onchange={() => toggleCalibrationStrategy('advanced')} disabled={isLocked} />
                      <span>advanced</span>
                    </label>
                  </div>
                </div>
              </div>

              <label class="toggle-row with-divider">
                <div class="toggle-info">
                  <div class="toggle-title">Per-Host Auto-Calibration (-ach)</div>
                  <div class="toggle-desc">Calibrates separate baseline filters for each virtual host / subdomain.</div>
                </div>
                <input type="checkbox" bind:checked={scanStore.autoCalibrationPerHost} disabled={isLocked} class="native-checkbox" />
              </label>

              <div class="field-block with-divider">
                <label for="acc-textarea" class="field-label-sub">Custom Calibration Probes (-acc, one per line)</label>
                <textarea id="acc-textarea" rows={2} bind:value={scanStore.autoCalibrationStrings} disabled={isLocked} placeholder="Optional custom random probe strings..." class="native-textarea font-mono"></textarea>
              </div>

              <div class="guard-toggles with-divider">
                <label class="inline-check"><input type="checkbox" bind:checked={scanStore.stopOn403} disabled={isLocked} /><span>Stop on &gt;95% 403 (-sf)</span></label>
                <label class="inline-check"><input type="checkbox" bind:checked={scanStore.stopOnErrors} disabled={isLocked} /><span>Stop on Errors (-se)</span></label>
                <label class="inline-check"><input type="checkbox" bind:checked={scanStore.stopOnAll} disabled={isLocked} /><span>Stop on All (-sa)</span></label>
                <label class="inline-check"><input type="checkbox" bind:checked={scanStore.closeConnOnThrottle} disabled={isLocked} /><span>Close Conn on Throttle</span></label>
                <label class="inline-check"><input type="checkbox" bind:checked={scanStore.detectSoftBlock} disabled={isLocked} /><span>Detect Soft-Block Challenges</span></label>
              </div>
            </div>
          </section>

          <section class="drawer-section">
            <h3 class="section-heading">Scrapers, Forensics &amp; Preflight Workflows</h3>
            <div class="card-group">
              <div class="two-col-grid">
                <div class="field-block">
                  <label for="scrapers-input" class="field-label-sub">Scraper Groups (-scrapers)</label>
                  <input id="scrapers-input" type="text" bind:value={scanStore.scrapers} disabled={isLocked} placeholder="all" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="scraper-file-input" class="field-label-sub">Custom Scraper JSON (-scraperfile)</label>
                  <div class="path-picker-row">
                    <input id="scraper-file-input" type="text" bind:value={scanStore.scraperFile} disabled={isLocked} placeholder="/path/scrapers.json" class="native-input font-mono" />
                    <button type="button" class="preset-chip" disabled={isLocked} onclick={async () => {
                      const p = await scanStore.pickAuxiliaryFile('Select Custom Scraper JSON', 'JSON Files (*.json)', '*.json');
                      if (p) scanStore.scraperFile = p;
                    }}>Browse</button>
                  </div>
                </div>
              </div>

              <div class="two-col-grid with-divider">
                <div class="field-block">
                  <label for="audit-log-input" class="field-label-sub">Audit Log File (-audit-log)</label>
                  <input id="audit-log-input" type="text" bind:value={scanStore.auditLog} disabled={isLocked} placeholder="/path/to/audit.log" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="debug-log-input" class="field-label-sub">Debug Log File (-debug-log)</label>
                  <input id="debug-log-input" type="text" bind:value={scanStore.debugLog} disabled={isLocked} placeholder="/path/to/debug.log" class="native-input font-mono" />
                </div>
                <div class="field-block">
                  <label for="od-input" class="field-label-sub">Matched Bodies Dir (-od)</label>
                  <div class="path-picker-row">
                    <input id="od-input" type="text" bind:value={scanStore.outputDirectory} disabled={isLocked} placeholder="/path/to/output_dir" class="native-input font-mono" />
                    <button type="button" class="preset-chip" disabled={isLocked} onclick={async () => {
                      const p = await scanStore.pickDirectory('Select Output Directory (-od)');
                      if (p) scanStore.outputDirectory = p;
                    }}>Browse</button>
                  </div>
                </div>
                <div class="field-block">
                  <label for="search-hash-input" class="field-label-sub">Reproduce FNHASH (-search)</label>
                  <div class="path-picker-row">
                    <input id="search-hash-input" type="text" bind:value={searchHashInput} placeholder="Enter FNHASH..." class="native-input font-mono" />
                    <button
                      type="button"
                      class="preset-chip"
                      disabled={!searchHashInput.trim()}
                      onclick={async () => {
                        searchHashError = '';
                        searchHashOutput = '';
                        try {
                          searchHashOutput = await scanStore.searchFNHASH(searchHashInput);
                        } catch (err: unknown) {
                          searchHashError = err instanceof Error ? err.message : String(err);
                        }
                      }}
                    >Lookup</button>
                  </div>
                </div>
              </div>

              {#if searchHashError}
                <p class="warn-hint">{searchHashError}</p>
              {/if}
              {#if searchHashOutput}
                <textarea rows={4} readonly value={searchHashOutput} class="native-textarea font-mono"></textarea>
              {/if}

              <div class="field-block with-divider">
                <div class="field-label-row">
                  <span class="field-label-sub">Preflight / Postflight Token Extraction</span>
                  <div class="preset-btns">
                    <select bind:value={scanStore.preflightMode} disabled={isLocked} class="mini-select font-mono">
                      <option value="per-request">per-request</option>
                      <option value="per-thread">per-thread</option>
                    </select>
                    <select bind:value={scanStore.preflightError} disabled={isLocked} class="mini-select font-mono">
                      <option value="abort">abort</option>
                      <option value="ignore">ignore</option>
                    </select>
                    <label class="inline-check">
                      <input type="checkbox" bind:checked={scanStore.preflightAnyHost} disabled={isLocked} />
                      <span>anyhost</span>
                    </label>
                  </div>
                </div>

                <div class="flight-add-grid">
                  <select bind:value={newFlightKind} disabled={isLocked} class="mini-select font-mono">
                    <option value="preflight">Preflight</option>
                    <option value="postflight">Postflight</option>
                  </select>
                  <div class="path-picker-row">
                    <input type="text" bind:value={newFlightFile} disabled={isLocked} placeholder="Flight raw .req file" class="native-input font-mono" />
                    <button type="button" class="preset-chip" disabled={isLocked} onclick={async () => {
                      const p = await scanStore.pickAuxiliaryFile('Select Flight Request File', 'Request Files (*.req, *.txt)', '*.req;*.txt');
                      if (p) newFlightFile = p;
                    }}>Browse</button>
                  </div>
                  <input type="text" bind:value={newVarName} disabled={isLocked} placeholder="Var Keyword (e.g. CSRF)" class="native-input font-mono" />
                  <div class="path-picker-row">
                    <select bind:value={newVarSource} disabled={isLocked} class="mini-select font-mono">
                      <option value="auto">auto</option>
                      <option value="form">form</option>
                      <option value="meta">meta</option>
                      <option value="cookie">cookie</option>
                      <option value="header">header</option>
                      <option value="regex">regex</option>
                    </select>
                    <input type="text" bind:value={newVarPattern} disabled={isLocked} placeholder={newVarSource === 'regex' ? 'Capture Regex' : 'Key / Field name'} class="native-input font-mono" />
                    <button type="button" disabled={isLocked || !newFlightFile.trim()} onclick={addFlightRule} class="btn-primary-sm">Add</button>
                  </div>
                </div>

                {#each scanStore.preflights as pf, i}
                  <div class="header-item font-mono">
                    <span>[PRE] {pf.request_file.split(/[/\\]/).pop()} {pf.vars?.[0] ? `-> ${pf.vars[0].name}` : ''}</span>
                    <button type="button" disabled={isLocked} onclick={() => (scanStore.preflights = scanStore.preflights.filter((_, idx) => idx !== i))} class="btn-remove">Remove</button>
                  </div>
                {/each}
                {#each scanStore.postflights as pf, i}
                  <div class="header-item font-mono">
                    <span>[POST] {pf.request_file.split(/[/\\]/).pop()} {pf.vars?.[0] ? `-> ${pf.vars[0].name}` : ''}</span>
                    <button type="button" disabled={isLocked} onclick={() => (scanStore.postflights = scanStore.postflights.filter((_, idx) => idx !== i))} class="btn-remove">Remove</button>
                  </div>
                {/each}
              </div>
            </div>
          </section>
        {/if}
      </div>

      <div class="drawer-footer">
        <button type="button" onclick={close} class="btn-done">
          Done
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .drawer-backdrop {
    position: fixed;
    inset: 0;
    z-index: 10020;
    background: rgba(0, 0, 0, 0.55);
    backdrop-filter: blur(4px);
    display: flex;
    justify-content: flex-end;
  }

  .drawer-panel {
    width: 100%;
    max-width: 610px;
    height: 100%;
    background-color: var(--surface);
    border-left: 1px solid var(--border);
    color: var(--text);
    display: flex;
    flex-direction: column;
    box-shadow: -12px 0 36px rgba(0, 0, 0, 0.35);
  }

  .drawer-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 13px 18px;
    border-bottom: 1px solid var(--border);
  }

  .drawer-title-group {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--accent);
  }

  .drawer-title-group h2 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--text);
  }

  .drawer-close-btn {
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    padding: 4px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
  }

  .drawer-close-btn:hover {
    background-color: var(--surface-hover);
    color: var(--text);
  }

  .drawer-tabs {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 8px 18px;
    border-bottom: 1px solid var(--border);
    background-color: var(--bg-subtle);
    overflow-x: auto;
  }

  .drawer-tab {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 5px 10px;
    border-radius: 6px;
    border: 1px solid transparent;
    background: transparent;
    color: var(--text-muted);
    font-size: 11.5px;
    font-weight: 500;
    cursor: pointer;
    white-space: nowrap;
    transition: background-color 0.14s, color 0.14s, border-color 0.14s;
  }

  .drawer-tab:hover {
    color: var(--text);
    background-color: var(--surface-hover);
  }

  .drawer-tab.active {
    background-color: var(--surface);
    color: var(--text);
    border-color: var(--border);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  }

  .tab-pill {
    font-size: 10px;
    font-weight: 600;
    padding: 0 5px;
    border-radius: 999px;
    background-color: rgba(14, 165, 233, 0.15);
    color: var(--accent);
  }

  .drawer-body {
    flex: 1;
    overflow-y: auto;
    padding: 16px 18px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .locked-notice {
    padding: 8px 12px;
    border-radius: var(--radius-sm);
    border: 1px solid rgba(245, 158, 11, 0.35);
    background-color: rgba(245, 158, 11, 0.1);
    color: #fbbf24;
    font-size: 11.5px;
  }

  .warn-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 7px 10px;
    border-radius: var(--radius-sm);
    border: 1px solid rgba(245, 158, 11, 0.35);
    background-color: rgba(245, 158, 11, 0.1);
    color: #fbbf24;
    font-size: 11px;
  }

  .drawer-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .section-heading {
    margin: 0;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
  }

  .card-group {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .two-col-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .concurrency-steppers-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .stepper-item {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .field-label-sub {
    font-size: 11px;
    font-weight: 600;
    color: var(--text);
  }

  .toggle-row {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    cursor: pointer;
  }

  .with-divider {
    border-top: 1px solid var(--border);
    padding-top: 10px;
  }

  .toggle-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .toggle-title {
    font-size: 12px;
    font-weight: 500;
    color: var(--text);
  }

  .toggle-desc {
    font-size: 11px;
    color: var(--text-muted);
    line-height: 1.4;
  }

  .native-checkbox {
    width: 16px;
    height: 16px;
    accent-color: var(--accent);
    cursor: pointer;
    flex-shrink: 0;
    margin-top: 2px;
  }

  .field-block {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .field-label-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    flex-wrap: wrap;
  }

  .preset-btns {
    display: flex;
    align-items: center;
    gap: 5px;
    flex-wrap: wrap;
  }

  .preset-chip {
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    color: var(--text-muted);
    font-size: 10px;
    font-weight: 500;
    padding: 3px 8px;
    cursor: pointer;
    transition: all 0.12s;
    flex-shrink: 0;
  }

  .preset-chip:hover:not(:disabled) {
    border-color: var(--accent);
    color: var(--text);
  }

  .preset-chip.danger {
    color: #f87171;
  }

  .preset-chip:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .native-input,
  .native-textarea {
    width: 100%;
    box-sizing: border-box;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text);
    font-size: 11.5px;
    padding: 6px 9px;
    outline: none;
  }

  .native-input {
    height: 30px;
    line-height: 16px;
  }

  .mini-select {
    height: 26px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    color: var(--text);
    font-size: 11px;
    padding: 0 6px;
    cursor: pointer;
  }

  select.native-input {
    padding: 0 8px;
    cursor: pointer;
  }

  .path-picker-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .extra-wl-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .wl-kw-input {
    width: 82px;
    flex-shrink: 0;
  }

  .wl-path-input {
    flex: 1.4;
  }

  .wl-enc-input {
    flex: 1;
  }

  .encoder-chips-wrap {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .inline-check {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 11px;
    color: var(--text);
    cursor: pointer;
  }

  .guard-toggles {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .flight-add-grid {
    display: grid;
    grid-template-columns: 110px 1fr;
    gap: 6px;
  }

  .quick-presets-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .quick-preset-card {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    padding: 8px 10px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    cursor: pointer;
    text-align: left;
    transition: border-color 0.12s, background-color 0.12s;
  }

  .quick-preset-card:hover:not(:disabled) {
    border-color: var(--accent);
    background-color: var(--surface-hover);
  }

  .quick-preset-card:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .qp-title {
    font-size: 11px;
    font-weight: 600;
    color: var(--text);
  }

  .qp-sub {
    font-size: 10px;
    color: var(--text-muted);
  }

  .native-input:focus,
  .native-textarea:focus {
    border-color: var(--accent);
  }

  .native-input:disabled,
  .native-textarea:disabled {
    opacity: 0.55;
  }

  .header-add-row {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .key-input {
    width: 42%;
  }

  .val-input {
    flex: 1;
  }

  .btn-primary-sm {
    background-color: var(--accent);
    border: none;
    border-radius: var(--radius-sm);
    color: var(--accent-contrast, #ffffff);
    font-size: 11.5px;
    font-weight: 500;
    padding: 6px 11px;
    cursor: pointer;
    flex-shrink: 0;
  }

  .btn-primary-sm:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .headers-list {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background-color: var(--surface);
    overflow: hidden;
  }

  .header-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 6px 10px;
    border-bottom: 1px solid var(--border);
    font-size: 11px;
  }

  .header-item:last-child {
    border-bottom: none;
  }

  .header-item-text {
    min-width: 0;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .header-key {
    color: var(--accent);
    font-weight: 600;
  }

  .header-sep {
    color: var(--text-muted);
  }

  .header-val {
    color: var(--text);
  }

  .btn-remove {
    background: transparent;
    border: none;
    color: #f87171;
    font-size: 11px;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .btn-remove:hover:not(:disabled) {
    background-color: rgba(248, 113, 113, 0.12);
  }

  .empty-hint {
    margin: 0;
    font-size: 11px;
    color: var(--text-muted);
  }

  .warn-hint {
    margin: 0;
    font-size: 11px;
    color: #fbbf24;
  }

  .inline-code {
    color: var(--accent);
    font-family: 'JetBrains Mono', monospace;
  }

  .drawer-footer {
    display: flex;
    justify-content: flex-end;
    padding: 12px 18px;
    border-top: 1px solid var(--border);
  }

  .btn-done {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text);
    font-size: 12px;
    font-weight: 500;
    padding: 6px 16px;
    cursor: pointer;
  }

  .btn-done:hover {
    background-color: var(--surface-hover);
  }
</style>
