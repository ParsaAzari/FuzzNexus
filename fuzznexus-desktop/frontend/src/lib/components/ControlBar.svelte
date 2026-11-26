<script lang="ts">
  import { scanStore } from '../stores/scan.svelte';
  import { resultsStore } from '../stores/results.svelte';
  import { wordlistsStore } from '../stores/wordlists.svelte';
  import { toastStore } from '../stores/toast.svelte';
  import Icon from './Icon.svelte';
  import NumberStepper from './NumberStepper.svelte';
  import { VerifyWordlistPath } from '../../../wailsjs/go/main/App';

  let activeOptionsCount = $derived(
    Object.keys(scanStore.headers).length +
      (scanStore.data.trim() ? 1 : 0) +
      (scanStore.proxyURL.trim() ? 1 : 0) +
      (scanStore.replayProxyURL.trim() ? 1 : 0) +
      (scanStore.followRedirects ? 1 : 0) +
      (scanStore.rawURI ? 1 : 0) +
      (scanStore.http2 ? 1 : 0) +
      (scanStore.ignoreBody ? 1 : 0) +
      (scanStore.tlsProfile ? 1 : 0) +
      (scanStore.extensions.trim() ? 1 : 0) +
      (scanStore.recursion ? 1 : 0) +
      (scanStore.delay.trim() ? 1 : 0) +
      (scanStore.additionalWordlists.length > 0 ? scanStore.additionalWordlists.length : 0) +
      (scanStore.inputMode !== 'clusterbomb' ? 1 : 0) +
      (scanStore.matchers.trim() || scanStore.filters.trim() || scanStore.matcherSize.trim() || scanStore.filterSize.trim() ? 1 : 0) +
      (scanStore.sni.trim() || scanStore.clientCert.trim() || scanStore.auditLog.trim() || scanStore.outputDirectory.trim() ? 1 : 0) +
      (scanStore.preflights.length + scanStore.postflights.length)
  );

  let isSandbox = $derived(
    !!scanStore.wordlist &&
      (scanStore.wordlist.includes('sandboxes') ||
        scanStore.wordlist.includes('sb_wordlist') ||
        scanStore.wordlist.includes('sb_merged') ||
        scanStore.wordlist.includes('sb_sanitized'))
  );

  let activeWordlistLines = $derived.by(() => {
    if (!scanStore.wordlist) return null;
    if (scanStore.wordlistLineCount != null) {
      return scanStore.wordlistLineCount;
    }
    const item = wordlistsStore.items.find((w) => w.path === scanStore.wordlist);
    if (item && item.line_count > 0) {
      return item.line_count;
    }
    return null;
  });

  $effect(() => {
    const path = scanStore.wordlist?.trim();
    if (!path) {
      scanStore.wordlistLineCount = null;
      scanStore.wordlistRisk = null;
      return;
    }
    void scanStore.analyzeWordlistRisk(path);

    const item = wordlistsStore.items.find((w) => w.path === path);
    if (item && item.line_count > 0) {
      if (scanStore.wordlistLineCount !== item.line_count) {
        scanStore.wordlistLineCount = item.line_count;
      }
      return;
    }
    if (scanStore.wordlistLineCount != null) return;

    let active = true;
    VerifyWordlistPath(path)
      .then((health) => {
        if (active && health && typeof health.total_lines === 'number' && health.total_lines >= 0) {
          scanStore.wordlistLineCount = health.total_lines;
        }
      })
      .catch(() => {
        // Path might not exist yet or still being entered
      });

    return () => {
      active = false;
    };
  });

  function handleClearAll() {
    if (scanStore.state === 'Running') return;
    resultsStore.clear();
    scanStore.reset();
    toastStore.info('Session reset: table, progress, and inspector cleared');
  }

  function getStatusDotColor(state: string): string {
    switch (state) {
      case 'Running':
        return '#34C759'; // Apple Green
      case 'Paused':
        return '#FF9500'; // Apple Orange
      case 'Completed':
        return '#007AFF'; // Apple Blue
      case 'Failed':
        return '#FF3B30'; // Apple Red
      default:
        return '#8E8E93'; // Apple Gray
    }
  }

  // Target URL & FUZZ Intelligent UX State
  let urlHelpOpen = $state<boolean>(false);
  let urlGroupRef = $state<HTMLElement | null>(null);
  let urlInputRef = $state<HTMLInputElement | null>(null);
  let selectedText = $state<string>('');

  function onUrlInputSelect() {
    if (!urlInputRef) return;
    const start = urlInputRef.selectionStart ?? 0;
    const end = urlInputRef.selectionEnd ?? 0;
    if (end > start) {
      selectedText = urlInputRef.value.substring(start, end);
    } else {
      selectedText = '';
    }
  }

  function setSelectionAsFuzz() {
    if (scanStore.state === 'Running' || !urlInputRef) return;
    const start = urlInputRef.selectionStart ?? 0;
    const end = urlInputRef.selectionEnd ?? 0;
    const val = scanStore.url;
    if (end > start) {
      scanStore.url = val.substring(0, start) + 'FUZZ' + val.substring(end);
      selectedText = '';
      toastStore.success('Replaced selection with FUZZ');
      // Set cursor right after FUZZ
      setTimeout(() => {
        urlInputRef?.focus();
        urlInputRef?.setSelectionRange(start + 4, start + 4);
      }, 10);
    } else {
      // Insert at cursor
      scanStore.url = val.substring(0, start) + 'FUZZ' + val.substring(start);
      toastStore.info('Inserted FUZZ at cursor');
      setTimeout(() => {
        urlInputRef?.focus();
        urlInputRef?.setSelectionRange(start + 4, start + 4);
      }, 10);
    }
  }

  let urlAnalysis = $derived.by(() => {
    const raw = (scanStore.url || '').trim().replace(/^["']|["']$/g, '');
    const headers = scanStore.headers || {};
    const body = scanStore.data || '';

    // Check headers and body for fuzz (case-insensitive)
    let hasFuzzInHeaders = false;
    let headerKeyMatch = '';
    for (const [k, v] of Object.entries(headers)) {
      if (k.toLowerCase().includes('fuzz') || v.toLowerCase().includes('fuzz')) {
        hasFuzzInHeaders = true;
        headerKeyMatch = k;
        break;
      }
    }
    const hasFuzzInBody = body.toLowerCase().includes('fuzz');

    if (!raw) {
      return {
        scheme: 'https',
        hasFuzz: hasFuzzInHeaders || hasFuzzInBody,
        fuzzLocation: hasFuzzInHeaders ? 'headers' : hasFuzzInBody ? 'body' : 'none',
        fuzzScope: hasFuzzInHeaders ? `Header (${headerKeyMatch})` : hasFuzzInBody ? 'Body (POST/JSON)' : 'none',
        autoAppended: false,
        normalizedPreview: '',
        hasFuzzInHeaders,
        hasFuzzInBody
      };
    }

    let scheme = 'https';
    const schemeMatch = raw.match(/^([a-zA-Z][a-zA-Z0-9+.-]*):\/\//);
    if (schemeMatch) {
      scheme = schemeMatch[1].toLowerCase();
    } else {
      const lower = raw.toLowerCase();
      if (
        lower.startsWith('localhost') ||
        lower.startsWith('127.0.0.1') ||
        lower.startsWith('[::1]') ||
        lower.includes(':8000') ||
        lower.includes(':8080') ||
        lower.includes(':3000') ||
        lower.includes(':5000')
      ) {
        scheme = 'http';
      }
    }

    // Check FUZZ in URL (case-insensitive: matches fuzz, FUZZ, Fuzz, etc.)
    const hasFuzzInURL = /fuzz/i.test(raw);
    const hasAnyFuzz = hasFuzzInURL || hasFuzzInHeaders || hasFuzzInBody;

    let fuzzLocation = 'none';
    let fuzzScope = 'none';

    if (hasFuzzInURL) {
      if (/:\/\/.*fuzz.*\./i.test(raw) || /\..*fuzz.*\./i.test(raw) || /:\/\/[^/]*fuzz[^/]*\//i.test(raw)) {
        fuzzLocation = 'subdomain';
        fuzzScope = 'Subdomain / Host';
      } else if (raw.includes('?') && /fuzz/i.test(raw.split('?')[1])) {
        const queryPart = raw.split('?')[1];
        if (/fuzz=/i.test(queryPart)) {
          fuzzLocation = 'query';
          fuzzScope = 'Query Param Name';
        } else {
          fuzzLocation = 'query';
          fuzzScope = 'Query Param Value';
        }
      } else if (/\.[a-zA-Z0-9_-]*fuzz[a-zA-Z0-9_-]*($|[/?#])/i.test(raw)) {
        fuzzLocation = 'extension';
        fuzzScope = 'File Extension';
      } else {
        fuzzLocation = 'path';
        fuzzScope = 'URL Path / Endpoint';
      }
    } else if (hasFuzzInHeaders) {
      fuzzLocation = 'headers';
      fuzzScope = `Header (${headerKeyMatch})`;
    } else if (hasFuzzInBody) {
      fuzzLocation = 'body';
      fuzzScope = 'Body (POST/JSON)';
    }

    // Build normalized preview URL
    let target = raw;
    if (!schemeMatch) {
      target = `${scheme}://${target}`;
    }
    // Normalize any casing of fuzz to uppercase FUZZ
    target = target.replace(/fuzz/gi, 'FUZZ');

    if (!hasAnyFuzz) {
      if (target.includes('?')) {
        const parts = target.split('?');
        const query = parts[1];
        if (query.endsWith('=') || query.endsWith('&') || query === '') {
          target = `${target}FUZZ`;
        } else {
          if (parts[0].endsWith('/')) {
            target = `${parts[0]}FUZZ?${query}`;
          } else {
            target = `${parts[0]}/FUZZ?${query}`;
          }
        }
      } else {
        if (target.endsWith('/')) {
          target = `${target}FUZZ`;
        } else {
          target = `${target}/FUZZ`;
        }
      }
    }

    return {
      scheme,
      hasFuzz: hasAnyFuzz,
      hasFuzzInURL,
      hasFuzzInHeaders,
      hasFuzzInBody,
      fuzzLocation,
      fuzzScope,
      autoAppended: !hasAnyFuzz,
      normalizedPreview: target
    };
  });

  function toggleScheme() {
    if (scanStore.state === 'Running') return;
    const current = (scanStore.url || '').trim();
    if (!current) {
      scanStore.url = urlAnalysis.scheme === 'https' ? 'http://' : 'https://';
      return;
    }
    const schemeMatch = current.match(/^([a-zA-Z][a-zA-Z0-9+.-]*):\/\/(.*)$/);
    if (schemeMatch) {
      const curScheme = schemeMatch[1].toLowerCase();
      const rest = schemeMatch[2];
      const newScheme = curScheme === 'https' ? 'http' : 'https';
      scanStore.url = `${newScheme}://${rest}`;
    } else {
      const newScheme = urlAnalysis.scheme === 'https' ? 'http' : 'https';
      scanStore.url = `${newScheme}://${current}`;
    }
  }

  function insertDefaultFuzz() {
    if (scanStore.state === 'Running') return;
    if (urlAnalysis.normalizedPreview) {
      scanStore.url = urlAnalysis.normalizedPreview;
      toastStore.info('Target URL normalized with /FUZZ');
    }
  }

  let dynamicTemplates = $derived.by(() => {
    const raw = (scanStore.url || '').trim();
    // Default base domain if none provided
    let base = 'https://example.com';
    let baseHost = 'example.com';
    let basePath = '';

    if (raw) {
      try {
        const withScheme = /^https?:\/\//i.test(raw) ? raw : `https://${raw}`;
        const u = new URL(withScheme.replace(/fuzz/gi, 'FUZZ_TEMP'));
        base = `${u.protocol}//${u.host}`;
        baseHost = u.host.replace(/FUZZ_TEMP/g, 'example');
        basePath = u.pathname.replace(/\/+$/, '').replace(/FUZZ_TEMP/g, 'api');
      } catch {
        base = raw.replace(/\/?fuzz.*$/i, '').replace(/\/+$/, '') || 'https://example.com';
      }
    }

    const currentOrigin = basePath ? `${base}${basePath}` : base;

    return [
      {
        name: 'Path / Directory',
        desc: 'Fuzz endpoints, hidden admin panels & API paths',
        category: 'URL Path',
        preview: `${currentOrigin}/FUZZ`,
        template: `${currentOrigin}/FUZZ`
      },
      {
        name: 'File Extension',
        desc: 'Discover backup, config, or source files (.php, .bak, .env, .json)',
        category: 'Extension',
        preview: `${currentOrigin}/index.FUZZ`,
        template: `${currentOrigin}/index.FUZZ`
      },
      {
        name: 'Query Parameter Value',
        desc: 'Fuzz parameter inputs, resource IDs & SQLi/XSS targets',
        category: 'Query Param',
        preview: `${currentOrigin}?id=FUZZ`,
        template: `${currentOrigin}?id=FUZZ`
      },
      {
        name: 'Query Parameter Name',
        desc: 'Discover unadvertised debug / test parameters (?FUZZ=1)',
        category: 'Param Name',
        preview: `${currentOrigin}?FUZZ=1`,
        template: `${currentOrigin}?FUZZ=1`
      },
      {
        name: 'Subdomain / Host Discovery',
        desc: 'Discover hidden subdomains & microservices (FUZZ.domain.com)',
        category: 'Subdomain',
        preview: `https://FUZZ.${baseHost.replace(/^www\./i, '')}/`,
        template: `https://FUZZ.${baseHost.replace(/^www\./i, '')}/`
      }
    ];
  });

  function applyTemplate(templateStr: string) {
    if (scanStore.state === 'Running') return;
    scanStore.url = templateStr;
    urlHelpOpen = false;
    toastStore.info(`Applied template: ${templateStr}`);
  }

  $effect(() => {
    if (!urlHelpOpen) return;

    function onDocClick(e: MouseEvent) {
      if (urlGroupRef && !urlGroupRef.contains(e.target as Node)) {
        urlHelpOpen = false;
      }
    }

    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        urlHelpOpen = false;
      }
    }

    window.addEventListener('mousedown', onDocClick);
    window.addEventListener('keydown', onKeyDown);

    return () => {
      window.removeEventListener('mousedown', onDocClick);
      window.removeEventListener('keydown', onKeyDown);
    };
  });
</script>

<div class="control-bar-wrapper select-none">
  <!-- Top Row: Configuration Inputs -->
  <div class="config-row">
    <!-- Method Selector -->
    <select
      bind:value={scanStore.method}
      disabled={scanStore.state === 'Running'}
      class="method-select font-mono"
      data-tooltip="HTTP Method: Use POST, PUT, or PATCH when fuzzing JSON or body data"
      data-tooltip-pos="down"
    >
      <option value="GET">GET</option>
      <option value="POST">POST</option>
      <option value="PUT">PUT</option>
      <option value="DELETE">DELETE</option>
      <option value="HEAD">HEAD</option>
      <option value="PATCH">PATCH</option>
      <option value="OPTIONS">OPTIONS</option>
    </select>

    <!-- Target URL Input -->
    <div
      class="input-group url-group"
      class:has-fuzz={urlAnalysis.hasFuzz}
      bind:this={urlGroupRef}
    >
      <div class="url-left-controls">
        <button
          type="button"
          class="scheme-toggle-pill font-mono"
          class:is-https={urlAnalysis.scheme === 'https'}
          onclick={toggleScheme}
          disabled={scanStore.state === 'Running'}
          data-tooltip="Protocol: Switch between secure HTTPS and HTTP"
          data-tooltip-pos="down"
        >
          {urlAnalysis.scheme.toUpperCase()}
        </button>
        <span class="input-label">URL</span>
      </div>

      <input
        bind:this={urlInputRef}
        type="text"
        placeholder="https://example.com/FUZZ"
        bind:value={scanStore.url}
        onselect={onUrlInputSelect}
        onkeyup={onUrlInputSelect}
        onclick={onUrlInputSelect}
        onblur={() => {
          // Instantly approve and normalize any variant of fuzz (fuzz, Fuzz, fUzz) to standard uppercase FUZZ
          if (scanStore.url && /fuzz/i.test(scanStore.url)) {
            scanStore.url = scanStore.url.replace(/fuzz/gi, 'FUZZ');
          }
        }}
        disabled={scanStore.state === 'Running'}
        class="text-input font-mono"
        spellcheck="false"
      />

      <div class="url-right-controls">
        {#if selectedText.trim() && scanStore.state !== 'Running'}
          <button
            type="button"
            class="badge-set-fuzz font-mono"
            onclick={setSelectionAsFuzz}
            data-tooltip="Set Selection: Replace '{selectedText}' with FUZZ keyword"
          >
            <Icon name="zap" size={10} />
            <span>Set FUZZ</span>
          </button>
        {:else if urlAnalysis.hasFuzz}
          <span
            class="badge-fuzz-loc font-mono"
            class:badge-subdomain={urlAnalysis.fuzzLocation === 'subdomain'}
            class:badge-query={urlAnalysis.fuzzLocation === 'query'}
            class:badge-header={urlAnalysis.fuzzLocation === 'headers'}
            class:badge-body={urlAnalysis.fuzzLocation === 'body'}
            data-tooltip="Injection Active: Payload replaces {urlAnalysis.fuzzScope}"
          >
            <Icon name="zap" size={10} />
            <span class="badge-fuzz-text">{urlAnalysis.fuzzScope}</span>
          </span>
        {:else if (scanStore.url || '').trim()}
          <button
            type="button"
            class="badge-auto-fuzz font-mono"
            onclick={insertDefaultFuzz}
            disabled={scanStore.state === 'Running'}
            data-tooltip="Auto-Append: /FUZZ will be appended at scan start"
          >
            <Icon name="plus" size={10} />
            <span>Auto /FUZZ</span>
          </button>
        {/if}

        <button
          type="button"
          class="btn-url-help"
          class:is-active={urlHelpOpen}
          onclick={() => (urlHelpOpen = !urlHelpOpen)}
          data-tooltip="Target & Payload Guide: Syntax examples and 1-click presets"
          data-tooltip-pos="down"
          aria-label="URL and FUZZ Help"
        >
          <Icon name="info" size={13} />
        </button>
      </div>

      {#if urlHelpOpen}
        <div class="url-help-popover select-none" role="dialog" aria-modal="true">
          <!-- Popover Header -->
          <div class="popover-head">
            <div class="popover-head-title">
              <Icon name="zap" size={13} />
              <span>Target & Payload Injection Guide</span>
            </div>
            <button
              type="button"
              class="btn-popover-close"
              onclick={() => (urlHelpOpen = false)}
              aria-label="Close guide"
            >
              <Icon name="x" size={12} />
            </button>
          </div>

          <!-- Live Attack Surface Status HUD -->
          <div class="attack-surface-hud">
            <div class="hud-label-row">
              <span class="hud-title font-mono">DETECTED INJECTION SURFACE</span>
              <span class="hud-status-badge {urlAnalysis.hasFuzz ? 'active' : 'auto'}">
                {urlAnalysis.hasFuzz ? 'ACTIVE' : 'AUTO-PATH'}
              </span>
            </div>

            <div class="hud-detail-box">
              <div class="hud-metric">
                <span class="hud-metric-key">Current Target:</span>
                <span class="hud-metric-val font-mono">{urlAnalysis.fuzzScope}</span>
              </div>
              <div class="hud-metric">
                <span class="hud-metric-key">Casing Support:</span>
                <span class="hud-metric-val font-mono text-green">Case-Insensitive (FUZZ / fuzz / Fuzz)</span>
              </div>
            </div>

            {#if !urlAnalysis.hasFuzz}
              <div class="hud-tip font-mono">
                <Icon name="info" size={12} /> <span>Tip: Type <code class="fuzz-pill font-mono">FUZZ</code> (or lowercase <code class="fuzz-pill font-mono">fuzz</code>) anywhere in the URL, Header, or Request Body to inject wordlists at that exact position.</span>
              </div>
            {/if}
          </div>

          <!-- Quick 1-Click Templates on user domain -->
          <div class="popover-section-label font-mono">
            <span>PRESETS ON YOUR TARGET</span>
            <span class="label-dim">({scanStore.url.trim() ? 'Adapted to current URL' : 'Default'})</span>
          </div>
          <div class="template-list">
            {#each dynamicTemplates as t}
              <button
                type="button"
                class="template-btn"
                onclick={() => applyTemplate(t.template)}
              >
                <div class="template-top">
                  <span class="template-name">{t.name}</span>
                  <span class="template-cat font-mono">{t.category}</span>
                </div>
                <div class="template-desc">{t.desc}</div>
                <code class="template-code font-mono">{t.preview}</code>
              </button>
            {/each}
          </div>

          <!-- Extended Injection Scenarios (Headers & Body) -->
          <div class="popover-section-label font-mono">HEADER & BODY FUZZING</div>
          <div class="extra-guides">
            <div class="extra-guide-item">
              <span class="guide-badge font-mono">HEADERS</span>
              <span class="guide-text">Set <code class="code-chip">Authorization: Bearer FUZZ</code> or <code class="code-chip">Host: FUZZ.site.com</code> in Options drawer.</span>
            </div>
            <div class="extra-guide-item">
              <span class="guide-badge font-mono">POST BODY</span>
              <span class="guide-text">Include <code class="code-chip font-mono">FUZZ</code> in JSON/Forms (<code class="code-chip font-mono">&#123;"user": "FUZZ"&#125;</code>) for credential fuzzing.</span>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <!-- Wordlist Selector Pill (Compact & Informative) -->
    <div class="wordlist-pill-wrapper">
      {#if isSandbox}
        <div
          class="wordlist-chip sandbox font-mono"
          data-tooltip="Sandbox Wordlist: {activeWordlistLines != null ? activeWordlistLines.toLocaleString() + ' lines' : 'Active'}"
          data-tooltip-pos="down"
        >
          <Icon name="target" size={12} />
          <span class="chip-title">SANDBOX</span>
          {#if activeWordlistLines != null}
            <span class="chip-lines">{activeWordlistLines.toLocaleString()} l</span>
          {/if}
          <button
            type="button"
            class="btn-chip-action"
            data-tooltip="Discard sandbox copy"
            disabled={scanStore.state === 'Running'}
            onclick={async (e) => {
              e.stopPropagation();
              const path = scanStore.wordlist;
              scanStore.setWordlist('', null);
              if (path) await wordlistsStore.cleanupSandbox(path);
              toastStore.info('Sandbox wordlist discarded');
            }}
          >
            <Icon name="x" size={10} />
          </button>
        </div>
      {:else if scanStore.wordlist}
        {@const baseName = scanStore.wordlist.split(/[/\\]/).pop() || scanStore.wordlist}
        <button
          type="button"
          class="wordlist-chip active font-mono"
          disabled={scanStore.state === 'Running'}
          onclick={async () => {
            const selected = await wordlistsStore.openNativeFilePicker('link');
            if (selected) scanStore.setWordlist(selected, null);
          }}
          data-tooltip="Active Wordlist: {baseName} ({activeWordlistLines != null ? activeWordlistLines.toLocaleString() + ' lines' : 'Click to change'})"
          data-tooltip-pos="down"
        >
          <Icon name="file-text" size={12} />
          <span class="chip-name">{baseName}</span>
          {#if activeWordlistLines != null}
            <span class="chip-lines">{activeWordlistLines.toLocaleString()} l</span>
          {/if}
          <Icon name="chevron-down" size={10} />
        </button>
      {:else}
        <button
          type="button"
          class="wordlist-chip empty font-mono"
          disabled={scanStore.state === 'Running'}
          onclick={async () => {
            const selected = await wordlistsStore.openNativeFilePicker('link');
            if (selected) scanStore.setWordlist(selected, null);
          }}
          data-tooltip="Active Wordlist: Click to select file (or browse in Wordlists tab)"
          data-tooltip-pos="down"
        >
          <Icon name="folder" size={12} />
          <span>Select Wordlist...</span>
        </button>
      {/if}

      {#if !isSandbox && scanStore.wordlistRisk && scanStore.wordlistRisk.risk_level === 'high'}
        <button
          type="button"
          class="wordlist-chip risk font-mono"
          disabled={scanStore.state === 'Running' || scanStore.isSanitizingWordlist}
          onclick={async () => {
            try {
              const dto = await scanStore.sanitizeActiveWordlist(false);
              toastStore.success(`Created sanitized sandbox copy (${dto.line_count.toLocaleString()} lines)`);
            } catch (err: unknown) {
              toastStore.error(err instanceof Error ? err.message : String(err));
            }
          }}
          data-tooltip="WAF Risk Detected: {scanStore.wordlistRisk.traversal_count + scanStore.wordlistRisk.waf_signature_count} trigger lines. Click to auto-clean."
          data-tooltip-pos="down"
        >
          <Icon name="alert-triangle" size={10} />
          <span>WAF Risk</span>
        </button>
      {/if}
    </div>

    <!-- Request & Engine Options Trigger in Config Row -->
    <button
      type="button"
      onclick={() => (scanStore.isRequestOptionsOpen = true)}
      data-tooltip="Request & Engine Options: Threads, Matchers, Extensions, Proxy, Recursion, uTLS & Logs"
      data-tooltip-pos="down"
      class="options-trigger-btn top-row font-mono"
      class:has-active={activeOptionsCount > 0}
    >
      <Icon name="sliders" size={13} />
      <span>Options</span>
      {#if activeOptionsCount > 0}
        <span class="options-badge font-mono">{activeOptionsCount}</span>
      {/if}
    </button>
  </div>

  <!-- Bottom Row: Execution Controls & Live Telemetry -->
  <div class="actions-row">
    <div class="buttons-group">
      {#if scanStore.state !== 'Running' && scanStore.state !== 'Paused'}
        <button
          class="btn btn-primary"
          onclick={() => scanStore.start()}
          disabled={scanStore.isSubmitting || (!scanStore.url.trim() && !scanStore.requestFile.trim()) || (!scanStore.wordlist.trim() && !scanStore.inputCommand.trim())}
        >
          <Icon name="play" size={14} />
          <span>Start Scan</span>
        </button>
      {:else if scanStore.state === 'Running'}
        <button class="btn btn-warning" onclick={() => scanStore.pause()}>
          <Icon name="pause" size={14} />
          <span>Pause</span>
        </button>
      {:else if scanStore.state === 'Paused'}
        <button class="btn btn-success" onclick={() => scanStore.resume()}>
          <Icon name="play" size={14} />
          <span>Resume</span>
        </button>
      {/if}

      {#if scanStore.state === 'Running' || scanStore.state === 'Paused'}
        <button class="btn btn-danger" onclick={() => scanStore.stop()}>
          <Icon name="stop" size={14} />
          <span>Stop</span>
        </button>
      {/if}

      <button
        type="button"
        class="btn btn-secondary btn-clear"
        onclick={handleClearAll}
        disabled={scanStore.state === 'Running' || (resultsStore.items.length === 0 && scanStore.sessionID === 0 && scanStore.reqTotal === 0 && scanStore.state === 'Idle')}
        data-tooltip="Reset Session: Clear table results, telemetry, and inspector"
        aria-label="Clear scan session and results"
      >
        <Icon name="trash" size={13} />
        <span>Clear</span>
      </button>

      <!-- Status Pill -->
      <div class="status-pill">
        <span class="status-dot" style="background-color: {getStatusDotColor(scanStore.state)};"></span>
        <span class="status-text">{scanStore.state}</span>
      </div>

      <!-- Live Rate / Metrics Pill -->
      <div class="metrics-pill font-mono">
        <span class="metric-item">
          <strong>{scanStore.reqSec.toLocaleString()}</strong> req/s
        </span>
        {#if scanStore.circuitState === 'Half-Open (Throttled)'}
          <span
            class="metric-item badge-throttled"
            data-tooltip="WAF Throttling Active: Auto-backoff delay applied to protect IP"
          >
            <Icon name="shield-alert" size={11} />
            <span>Throttled</span>
          </span>
        {/if}
        <button
          type="button"
          class="metric-item btn-error-diagnostics font-mono"
          class:has-errors={scanStore.errorCount > 0 || scanStore.recentErrors.length > 0}
          onclick={() => scanStore.toggleErrorDiagnostics()}
          data-tooltip={scanStore.errorCount > 0 || scanStore.recentErrors.length > 0
            ? `Network Diagnostics: ${Math.max(scanStore.errorCount, scanStore.recentErrors.length)} errors (click to inspect log)`
            : "Network Diagnostics: 0 errors (all requests healthy)"}
        >
          {#if scanStore.errorCount > 0 || scanStore.recentErrors.length > 0}
            <Icon name="alert-triangle" size={11} />
            <strong>{Math.max(scanStore.errorCount, scanStore.recentErrors.length)}</strong> errors
          {:else}
            <Icon name="activity" size={11} />
            <span>0 err</span>
          {/if}
        </button>
      </div>

      <!-- Realistic Browser Persona Selector -->
      <div
        class="browser-persona-pill"
        data-tooltip="Browser Persona: Sends authentic Client Hints & Sec-CH-UA to bypass WAFs"
      >
        <Icon name="globe" size={12} color="#38bdf8" />
        <select
          bind:value={scanStore.browserProfile}
          disabled={scanStore.state === 'Running'}
          class="browser-select font-mono"
          aria-label="Browser Persona"
        >
          <option value="auto">Auto-Rotate</option>
          <option value="chrome-macos">Chrome (macOS)</option>
          <option value="chrome-windows">Chrome (Win)</option>
          <option value="safari-macos">Safari (macOS)</option>
          <option value="firefox-windows">Firefox</option>
          <option value="edge-windows">Edge (Win)</option>
        </select>
      </div>

      <!-- Auto-Calibrate Soft-404 (-ac) Clean & Professional Switcher -->
      <button
        type="button"
        role="switch"
        aria-checked={scanStore.autoCalibration}
        disabled={scanStore.state === 'Running'}
        onclick={() => (scanStore.autoCalibration = !scanStore.autoCalibration)}
        data-tooltip="Auto-Calibrate (-ac): Filters wildcard responses and Soft-404 false positives"
        class="ac-switch-btn font-mono"
        class:is-active={scanStore.autoCalibration}
        aria-label="Auto-Calibrate Soft-404 (-ac)"
      >
        <Icon name="sliders-horizontal" size={11} />
        <span class="ac-switch-label">Auto-Calibrate</span>
        <div class="ac-track">
          <div class="ac-thumb"></div>
        </div>
      </button>
    </div>

    <!-- Live Telemetry & Progress Bar -->
    <div class="telemetry-group">
      {#if scanStore.reqTotal > 0}
        <div class="progress-container">
          {#if scanStore.queueTotal > 1}
            <span class="queue-badge font-mono" data-tooltip="Active Recursion / Sniper Queue Job">
              Job {scanStore.queuePos}/{scanStore.queueTotal}
            </span>
          {/if}
          <div class="progress-bar-bg">
            <div class="progress-bar-fill" style="width: {scanStore.percentage.toFixed(1)}%;"></div>
          </div>
          <span class="progress-label font-mono">
            {scanStore.percentage.toFixed(1)}% ({scanStore.reqCount.toLocaleString()} / {scanStore.reqTotal.toLocaleString()})
          </span>
        </div>
      {/if}
    </div>
  </div>

  {#if scanStore.lastError}
    <div class="error-banner">
      <Icon name="alert-triangle" size={16} />
      <span>{scanStore.lastError}</span>
    </div>
  {/if}
</div>

<style>
  .control-bar-wrapper {
    background-color: var(--surface);
    border-bottom: 1px solid var(--border);
    padding: 10px 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    font-size: 12px;
  }

  .config-row {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: nowrap;
  }

  .method-select {
    height: var(--control-h-base);
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 0 8px;
    font-size: var(--text-base);
    font-weight: var(--weight-medium);
    color: var(--text);
    outline: none;
  }

  .input-group {
    display: flex;
    align-items: center;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    height: var(--control-h-base);
    padding: 0 8px;
    box-sizing: border-box;
    transition: border-color 0.15s ease;
  }

  .input-group:focus-within {
    border-color: var(--accent);
  }

  .url-group {
    flex: 1;
    min-width: 280px;
    position: relative;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .wordlist-pill-wrapper {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .wordlist-chip {
    height: var(--control-h-base);
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 0 10px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text);
    font-size: 11.5px;
    cursor: pointer;
    transition: all 0.15s ease;
    user-select: none;
    max-width: 220px;
  }

  .wordlist-chip:hover:not(:disabled) {
    background-color: var(--surface-hover);
    border-color: var(--accent);
  }

  .wordlist-chip.empty {
    color: var(--text-muted);
    border-style: dashed;
  }

  .wordlist-chip.empty:hover:not(:disabled) {
    color: var(--accent);
    border-style: solid;
  }

  .wordlist-chip.sandbox {
    background-color: rgba(16, 185, 129, 0.12);
    border-color: rgba(16, 185, 129, 0.4);
    color: #10b981;
    cursor: default;
  }

  .wordlist-chip.risk {
    background-color: rgba(245, 158, 11, 0.12);
    border-color: rgba(245, 158, 11, 0.4);
    color: #f59e0b;
  }

  .chip-name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-weight: 500;
  }

  .chip-lines {
    font-size: 10px;
    opacity: 0.7;
    background-color: var(--bg-subtle);
    padding: 1px 4px;
    border-radius: 3px;
    flex-shrink: 0;
  }

  .btn-chip-action {
    background: transparent;
    border: none;
    color: inherit;
    opacity: 0.6;
    cursor: pointer;
    padding: 1px;
    display: inline-flex;
    align-items: center;
  }

  .btn-chip-action:hover {
    opacity: 1;
  }

  .options-trigger-btn.top-row {
    height: var(--control-h-base);
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 0 12px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    cursor: pointer;
    transition: all 0.15s ease;
    flex-shrink: 0;
  }

  .options-trigger-btn.top-row:hover {
    background-color: var(--surface-hover);
    border-color: var(--accent);
  }

  .options-trigger-btn.top-row.has-active {
    border-color: var(--accent);
    background-color: rgba(0, 122, 255, 0.08);
  }

  .url-left-controls {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .scheme-toggle-pill {
    height: 19px;
    padding: 0 5px;
    border-radius: var(--radius-xs, 4px);
    font-size: 9.5px;
    font-weight: 700;
    line-height: 1;
    letter-spacing: 0.04em;
    cursor: pointer;
    transition: all 0.15s ease;
    border: 1px solid rgba(245, 158, 11, 0.35);
    background-color: rgba(245, 158, 11, 0.12);
    color: #f59e0b;
    user-select: none;
    -webkit-user-select: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  .scheme-toggle-pill:hover:not(:disabled) {
    background-color: rgba(245, 158, 11, 0.22);
    border-color: rgba(245, 158, 11, 0.5);
  }

  .scheme-toggle-pill.is-https {
    border-color: rgba(16, 185, 129, 0.35);
    background-color: rgba(16, 185, 129, 0.12);
    color: #10b981;
  }

  .scheme-toggle-pill.is-https:hover:not(:disabled) {
    background-color: rgba(16, 185, 129, 0.22);
    border-color: rgba(16, 185, 129, 0.5);
  }

  :root[data-mode="light"] .scheme-toggle-pill,
  :root:not([data-mode]) .scheme-toggle-pill {
    color: #b45309;
    border-color: rgba(217, 119, 6, 0.35);
  }

  :root[data-mode="light"] .scheme-toggle-pill.is-https,
  :root:not([data-mode]) .scheme-toggle-pill.is-https {
    color: #047857;
    border-color: rgba(5, 150, 105, 0.35);
  }

  .url-right-controls {
    display: flex;
    align-items: center;
    gap: 5px;
    flex-shrink: 0;
  }

  .badge-fuzz-loc {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    height: 20px;
    padding: 0 6px;
    border-radius: var(--radius-xs, 4px);
    font-size: 10px;
    font-weight: 700;
    line-height: 1;
    letter-spacing: 0.03em;
    white-space: nowrap;
    background-color: rgba(99, 102, 241, 0.15);
    border: 1px solid rgba(99, 102, 241, 0.35);
    color: #818cf8;
    user-select: none;
    -webkit-user-select: none;
  }

  :root[data-mode="light"] .badge-fuzz-loc,
  :root:not([data-mode]) .badge-fuzz-loc {
    background-color: rgba(79, 70, 229, 0.1);
    border-color: rgba(79, 70, 229, 0.3);
    color: #4338ca;
  }

  .badge-auto-fuzz {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    height: 20px;
    padding: 0 6px;
    border-radius: var(--radius-xs, 4px);
    font-size: 10px;
    font-weight: 600;
    line-height: 1;
    white-space: nowrap;
    background-color: rgba(0, 122, 255, 0.12);
    border: 1px solid rgba(0, 122, 255, 0.3);
    color: #007aff;
    cursor: pointer;
    user-select: none;
    -webkit-user-select: none;
    transition: all 0.15s ease;
  }

  .badge-auto-fuzz:hover:not(:disabled) {
    background-color: rgba(0, 122, 255, 0.22);
    border-color: rgba(0, 122, 255, 0.5);
  }

  :root[data-mode="light"] .badge-auto-fuzz,
  :root:not([data-mode]) .badge-auto-fuzz {
    background-color: rgba(0, 122, 255, 0.08);
    border-color: rgba(0, 122, 255, 0.28);
    color: #0056b3;
  }

  .btn-url-help {
    width: 22px;
    height: 22px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--radius-xs, 4px);
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.15s ease;
    padding: 0;
  }

  .btn-url-help:hover,
  .btn-url-help.is-active {
    background-color: var(--bg-hover, rgba(255, 255, 255, 0.08));
    color: var(--text);
    border-color: var(--border);
  }

  .badge-set-fuzz {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    height: 20px;
    padding: 0 7px;
    background-color: var(--accent);
    color: var(--accent-contrast, #ffffff);
    border: none;
    border-radius: var(--radius-xs, 4px);
    font-size: 10px;
    font-weight: 700;
    cursor: pointer;
    box-shadow: 0 0 6px rgba(0, 122, 255, 0.4);
    animation: pulse-glow 1.5s infinite alternate;
  }

  @keyframes pulse-glow {
    0% { transform: scale(0.98); }
    100% { transform: scale(1.02); }
  }

  .badge-fuzz-loc.badge-subdomain {
    background-color: rgba(168, 85, 247, 0.14);
    border-color: rgba(168, 85, 247, 0.4);
    color: #c084fc;
  }

  .badge-fuzz-loc.badge-query {
    background-color: rgba(234, 179, 8, 0.14);
    border-color: rgba(234, 179, 8, 0.4);
    color: #facc15;
  }

  .badge-fuzz-loc.badge-header {
    background-color: rgba(6, 182, 212, 0.14);
    border-color: rgba(6, 182, 212, 0.4);
    color: #22d3ee;
  }

  .badge-fuzz-loc.badge-body {
    background-color: rgba(249, 115, 22, 0.14);
    border-color: rgba(249, 115, 22, 0.4);
    color: #fb923c;
  }

  .attack-surface-hud {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 8px 10px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .hud-label-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .hud-title {
    font-size: 9px;
    font-weight: 700;
    color: var(--text-secondary);
    letter-spacing: 0.05em;
  }

  .hud-status-badge {
    font-size: 9px;
    font-weight: 700;
    padding: 1px 6px;
    border-radius: 3px;
    letter-spacing: 0.03em;
  }

  .hud-status-badge.active {
    background-color: rgba(52, 199, 89, 0.15);
    color: #34C759;
    border: 1px solid rgba(52, 199, 89, 0.3);
  }

  .hud-status-badge.auto {
    background-color: rgba(255, 149, 0, 0.15);
    color: #FF9500;
    border: 1px solid rgba(255, 149, 0, 0.3);
  }

  .hud-detail-box {
    display: flex;
    flex-direction: column;
    gap: 3px;
    font-size: 11px;
  }

  .hud-metric {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .hud-metric-key {
    color: var(--text-secondary);
  }

  .hud-metric-val {
    font-weight: 600;
    color: var(--text);
  }

  .hud-metric-val.text-green {
    color: #34C759;
  }

  .hud-tip {
    font-size: 10px;
    color: var(--text-secondary);
    line-height: 1.4;
    padding-top: 4px;
    border-top: 1px dashed var(--border);
  }

  .fuzz-pill {
    background-color: rgba(245, 158, 11, 0.16);
    color: #f59e0b;
    padding: 1px 4px;
    border-radius: 3px;
    font-weight: 700;
  }

  .label-dim {
    font-weight: 400;
    opacity: 0.65;
    font-size: 9px;
  }

  .template-cat {
    font-size: 9px;
    padding: 1px 4px;
    border-radius: 3px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
  }

  .extra-guides {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .extra-guide-item {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 10px;
    color: var(--text-secondary);
    background-color: var(--bg-subtle);
    padding: 5px 8px;
    border-radius: 4px;
    border: 1px solid var(--border);
  }

  .guide-badge {
    font-size: 9px;
    font-weight: 700;
    padding: 1px 4px;
    border-radius: 3px;
    background-color: var(--surface);
    color: var(--text);
    border: 1px solid var(--border);
    flex-shrink: 0;
  }

  .code-chip {
    padding: 1px 3px;
    background-color: var(--surface);
    border-radius: 2px;
    color: var(--accent);
  }

  /* Popover styles */
  .url-help-popover {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    width: 390px;
    max-width: min(420px, 92vw);
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-md, 8px);
    box-shadow: 0 12px 28px rgba(0, 0, 0, 0.35), 0 4px 10px rgba(0, 0, 0, 0.2);
    z-index: 1000;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
  }

  .popover-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-bottom: 6px;
    border-bottom: 1px solid var(--border);
  }

  .popover-head-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 600;
    color: var(--text);
  }

  .btn-popover-close {
    width: 20px;
    height: 20px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    border-radius: 3px;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.12s ease;
    padding: 0;
  }

  .btn-popover-close:hover {
    background-color: var(--bg-hover, rgba(255, 255, 255, 0.08));
    color: var(--text);
  }

  .popover-section-label {
    font-size: 9.5px;
    font-weight: 700;
    letter-spacing: 0.05em;
    color: var(--text-secondary);
    opacity: 0.8;
  }

  .template-list {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .template-btn {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    padding: 6px 8px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 5px;
    text-align: left;
    cursor: pointer;
    transition: all 0.12s ease;
    width: 100%;
  }

  .template-btn:hover {
    background-color: rgba(0, 122, 255, 0.08);
    border-color: rgba(0, 122, 255, 0.4);
  }

  .template-top {
    display: flex;
    align-items: baseline;
    gap: 6px;
    width: 100%;
  }

  .template-name {
    font-size: 11px;
    font-weight: 600;
    color: var(--text);
  }

  .template-desc {
    font-size: 10px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .template-code {
    font-size: 10px;
    color: var(--accent);
    opacity: 0.95;
    background: transparent;
    padding: 0;
  }

  .btn-error-diagnostics {
    background: transparent;
    border: none;
    cursor: pointer;
    font-size: inherit;
    font-family: inherit;
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.1rem 0.4rem;
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    transition: all 0.15s ease;
  }

  .btn-error-diagnostics:hover {
    background-color: var(--hover-bg);
    color: var(--text);
  }

  .btn-error-diagnostics.has-errors {
    color: var(--danger, #ef4444);
    background-color: rgba(239, 68, 68, 0.12);
  }

  .btn-error-diagnostics.has-errors:hover {
    background-color: rgba(239, 68, 68, 0.22);
    transform: translateY(-1px);
  }

  .badge-throttled {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.1rem 0.45rem;
    border-radius: var(--radius-sm);
    background-color: rgba(245, 158, 11, 0.14);
    border: 1px solid rgba(245, 158, 11, 0.35);
    color: #f59e0b;
    font-weight: 600;
    font-size: 0.72rem;
    animation: throttlePulse 2s ease-in-out infinite;
  }

  @keyframes throttlePulse {
    0%, 100% {
      opacity: 1;
      transform: scale(1);
    }
    50% {
      opacity: 0.75;
      transform: scale(0.98);
    }
  }

  .input-label {
    font-size: var(--text-xs);
    font-weight: var(--weight-semibold);
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin-right: 6px;
    user-select: none;
    white-space: nowrap;
  }

  .text-input {
    all: unset;
    flex: 1;
    font-size: var(--text-base);
    color: var(--text);
    width: 100%;
  }

  .actions-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .buttons-group {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .btn {
    all: unset;
    box-sizing: border-box;
    height: var(--control-h-base);
    padding: 0 14px;
    border-radius: var(--radius-sm);
    font-size: var(--text-base);
    font-weight: var(--weight-medium);
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    cursor: pointer;
    transition: all 0.12s ease;
    user-select: none;
  }

  .btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .btn-primary {
    background-color: var(--accent);
    color: var(--accent-contrast);
  }

  .btn-primary:hover:not(:disabled) {
    filter: brightness(1.08);
  }

  .btn-warning {
    background-color: #FF9500;
    color: #ffffff;
  }

  .btn-success {
    background-color: #34C759;
    color: #ffffff;
  }

  .btn-danger {
    background-color: #FF3B30;
    color: #ffffff;
  }

  .btn-secondary {
    background-color: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-secondary);
  }

  .btn-secondary:hover:not(:disabled) {
    background-color: var(--surface-hover);
    color: var(--text);
    border-color: var(--border-strong, var(--border));
  }

  .btn-clear {
    padding: 0 11px;
    gap: 6px;
  }

  .status-pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: var(--control-h-base);
    box-sizing: border-box;
    padding: 0 11px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    font-size: var(--text-base);
    font-weight: var(--weight-medium);
    color: var(--text);
    user-select: none;
    flex-shrink: 0;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .browser-persona-pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: var(--control-h-base);
    box-sizing: border-box;
    padding: 0 6px 0 9px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    flex-shrink: 0;
    transition: border-color var(--duration-fast);
  }

  .browser-persona-pill:focus-within {
    border-color: var(--accent);
  }

  .browser-select {
    background: transparent;
    border: none;
    color: var(--text);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    cursor: pointer;
    outline: none;
    padding: 0 2px;
  }

  .browser-select:focus,
  .browser-select:focus-visible {
    outline: none;
    box-shadow: none;
  }

  .browser-select:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .ac-switch-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: var(--control-h-base);
    box-sizing: border-box;
    padding: 0 9px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    cursor: pointer;
    flex-shrink: 0;
    user-select: none;
    transition: border-color 0.15s ease, background-color 0.15s ease;
  }

  .ac-switch-btn:hover:not(:disabled) {
    background-color: var(--surface-hover);
    border-color: var(--border-hover, var(--text-muted));
  }

  .ac-switch-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .ac-switch-label {
    font-size: 11px;
    font-weight: 500;
    letter-spacing: -0.01em;
    color: var(--text-secondary);
    transition: color 0.15s ease;
  }

  .ac-switch-btn.is-active .ac-switch-label {
    color: var(--text);
  }

  .ac-track {
    width: 26px;
    height: 15px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 3px;
    padding: 1px;
    box-sizing: border-box;
    display: flex;
    align-items: center;
    position: relative;
    transition: background-color 0.2s ease, border-color 0.2s ease;
  }

  .ac-switch-btn.is-active .ac-track {
    background-color: var(--accent);
    border-color: var(--accent);
  }

  .ac-thumb {
    width: 11px;
    height: 11px;
    background-color: var(--text-muted);
    border-radius: 2px;
    transform: translateX(0);
    will-change: transform, background-color;
    /* انیمیشن دقیق و بدون لرزش یا پرش بیش از حد */
    transition:
      transform 0.25s cubic-bezier(0.25, 1, 0.5, 1),
      background-color 0.2s ease;
  }

  .ac-switch-btn.is-active .ac-thumb {
    transform: translateX(11px);
    background-color: var(--accent-contrast, #ffffff);
  }

  .options-trigger-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: var(--control-h-base);
    box-sizing: border-box;
    padding: 0 10px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    cursor: pointer;
    flex-shrink: 0;
    transition: all var(--duration-fast);
  }

  .options-trigger-btn:hover {
    background-color: var(--surface-hover);
    color: var(--text);
    border-color: var(--accent);
  }

  .options-trigger-btn.has-active {
    border-color: var(--accent);
  }

  .options-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 16px;
    height: 16px;
    padding: 0 5px;
    border-radius: 999px;
    background-color: var(--accent);
    color: var(--accent-contrast, #ffffff);
    font-size: 10px;
    font-weight: 600;
  }

  .telemetry-group {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    margin: 0 10px;
    min-width: 0;
  }

  .progress-container {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
  }

  .progress-bar-bg {
    flex: 1;
    height: 6px;
    background-color: var(--bg-subtle);
    border-radius: 3px;
    overflow: hidden;
  }

  .progress-bar-fill {
    height: 100%;
    background-color: var(--accent);
    border-radius: 3px;
    transition: width 0.2s ease;
  }

  .progress-label {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .queue-badge {
    font-size: 10.5px;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 4px;
    background-color: rgba(14, 165, 233, 0.14);
    border: 1px solid rgba(14, 165, 233, 0.3);
    color: var(--accent);
    white-space: nowrap;
  }

  .metrics-pill {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    height: var(--control-h-base);
    box-sizing: border-box;
    padding: 0 11px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    font-size: var(--text-base);
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
    font-weight: 500;
    color: var(--text-secondary);
    user-select: none;
    flex-shrink: 0;
  }

  .metric-item strong {
    color: var(--text);
  }

  .btn-error-diagnostics.has-errors strong {
    color: #FF3B30 !important;
  }

  .error-banner {
    background-color: rgba(255, 59, 48, 0.1);
    color: #FF3B30;
    padding: 4px 10px;
    border-radius: var(--radius-xs);
    font-size: var(--text-sm);
    border: 1px solid rgba(255, 59, 48, 0.2);
  }
</style>
