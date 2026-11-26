<script lang="ts">
  import TitleBar from './lib/components/TitleBar.svelte';
  import SplashScreen from './lib/components/SplashScreen.svelte';
  import ControlBar from './lib/components/ControlBar.svelte';
  import FilterBar from './lib/components/FilterBar.svelte';
  import AdvancedFilterModal from './lib/components/AdvancedFilterModal.svelte';
  import AboutModal from './lib/components/AboutModal.svelte';
  import SafeStopModal from './lib/components/SafeStopModal.svelte';
  import ScanAdvisoryBanner from './lib/components/ScanAdvisoryBanner.svelte';
  import ResumeRecoveryBanner from './lib/components/ResumeRecoveryBanner.svelte';
  import RequestOptionsDrawer from './lib/components/RequestOptionsDrawer.svelte';
  import SessionHistoryModal from './lib/components/SessionHistoryModal.svelte';
  import ErrorDiagnosticsModal from './lib/components/ErrorDiagnosticsModal.svelte';
  import VirtualGrid from './lib/components/VirtualGrid.svelte';
  import Inspector from './lib/components/Inspector.svelte';
  import SplitPane from './lib/components/SplitPane.svelte';
  import AnalyticsDashboard from './lib/components/AnalyticsDashboard.svelte';
  import WordlistManager from './lib/components/WordlistManager.svelte';
  import ToastContainer from './lib/components/ToastContainer.svelte';
  import GlobalTooltip from './lib/components/GlobalTooltip.svelte';
  import Icon from './lib/components/Icon.svelte';
  import { scanStore } from './lib/stores/scan.svelte';
  import { resultsStore } from './lib/stores/results.svelte';
  import { filtersStore } from './lib/stores/filters.svelte';
  import { toastStore } from './lib/stores/toast.svelte';
  import { shortcutsStore, matchesCombo } from './lib/stores/shortcuts.svelte';
  import ShortcutsModal from './lib/components/ShortcutsModal.svelte';
  import { ExportResults, SaveExportDialog } from '../wailsjs/go/main/App';

  type ViewMode = 'grid' | 'analytics' | 'wordlists';
  let activeView = $state<ViewMode>('grid');

  let selectedItem = $derived(resultsStore.selectedItem);
  let isExporting = $state<boolean>(false);
  let isAboutOpen = $state<boolean>(false);
  let showSplash = $state<boolean>(true);

  $effect(() => {
    scanStore.checkPendingResume();

    if (typeof window !== 'undefined') {
      const w = window as unknown as { runtime?: { EventsOn?: (event: string, callback: () => void) => void } };
      if (w.runtime?.EventsOn) {
        w.runtime.EventsOn('open:about-modal', () => {
          isAboutOpen = true;
        });
      }
    }

    function handleDragOver(e: DragEvent) {
      e.preventDefault();
    }
    function handleDrop(e: DragEvent) {
      e.preventDefault();
    }
    window.addEventListener('dragover', handleDragOver);
    window.addEventListener('drop', handleDrop);
    return () => {
      window.removeEventListener('dragover', handleDragOver);
      window.removeEventListener('drop', handleDrop);
    };
  });

  function handleGlobalShortcuts(e: KeyboardEvent) {
    const isMeta = e.metaKey || e.ctrlKey;

    if (isMeta && e.key.toLowerCase() === 'a') {
      const target = e.target as HTMLElement | null;
      const isInput =
        target instanceof HTMLInputElement ||
        target instanceof HTMLTextAreaElement ||
        target?.isContentEditable === true;
      if (!isInput) {
        e.preventDefault();
        return;
      }
    }

    if (e.key === 'Escape') {
      if (isAboutOpen) {
        isAboutOpen = false;
        return;
      }
      if (exportMenuOpen) {
        exportMenuOpen = false;
        return;
      }
      if (filtersStore.isModalOpen) {
        filtersStore.isModalOpen = false;
        return;
      }
      if (resultsStore.selectedID !== null) {
        resultsStore.select(null);
        return;
      }
    }

    if (shortcutsStore.isModalOpen) return;

    if (matchesCombo(e, shortcutsStore.getCombo('ui.openShortcuts'))) {
      e.preventDefault();
      shortcutsStore.isModalOpen = !shortcutsStore.isModalOpen;
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('ui.toggleAbout'))) {
      e.preventDefault();
      isAboutOpen = !isAboutOpen;
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('ui.viewGrid'))) {
      e.preventDefault();
      activeView = 'grid';
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('ui.viewAnalytics'))) {
      e.preventDefault();
      activeView = 'analytics';
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('ui.viewWordlists'))) {
      e.preventDefault();
      activeView = 'wordlists';
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('ui.toggleFilter'))) {
      e.preventDefault();
      filtersStore.isModalOpen = !filtersStore.isModalOpen;
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('ui.toggleHistory'))) {
      e.preventDefault();
      scanStore.isHistoryModalOpen = !scanStore.isHistoryModalOpen;
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('ui.toggleOptions'))) {
      e.preventDefault();
      document.querySelector<HTMLButtonElement>('[aria-label="Request Options"]')?.click();
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('scan.startOrResume'))) {
      e.preventDefault();
      if (scanStore.state === 'Paused') {
        void scanStore.resume();
      } else if (scanStore.state !== 'Running') {
        void scanStore.start();
      }
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('scan.pause'))) {
      e.preventDefault();
      if (scanStore.state === 'Running') {
        void scanStore.pause();
      }
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('scan.stop'))) {
      e.preventDefault();
      if (scanStore.state === 'Running' || scanStore.state === 'Paused') {
        void scanStore.stop();
      }
      return;
    }
    if (matchesCombo(e, shortcutsStore.getCombo('scan.reset'))) {
      e.preventDefault();
      if (scanStore.state !== 'Running' && scanStore.state !== 'Paused') {
        scanStore.reset();
        resultsStore.clear();
        toastStore.info('Session reset via shortcut');
      }
      return;
    }

    if (isMeta && e.key.toLowerCase() === 'z') {
      e.preventDefault();
      if (e.shiftKey) {
        filtersStore.redo();
      } else {
        filtersStore.undo();
      }
    }
  }

  let exportMenuOpen = $state<boolean>(false);

  async function handleExport(format: 'json' | 'csv' | 'txt' | 'html' | 'md' | 'ejson' | 'ecsv') {
    exportMenuOpen = false;
    if (resultsStore.items.length === 0) {
      toastStore.warning('No scan results available to export');
      return;
    }
    if (scanStore.sessionID <= 0) {
      toastStore.warning('No active scan session found');
      return;
    }
    isExporting = true;
    try {
      const defaultFileName = `fuzznexus_session_${scanStore.sessionID}_results.${format}`;
      const targetPath = await SaveExportDialog(defaultFileName, format);
      if (!targetPath) {
        // User cancelled native Save Dialog
        return;
      }
      await ExportResults(scanStore.sessionID, format, targetPath);
      const baseName = targetPath.split(/[/\\]/).pop() || targetPath;
      toastStore.success(`Exported ${resultsStore.items.length.toLocaleString()} items to ${baseName}`);
    } catch (err: any) {
      toastStore.error(`Export failed: ${err?.message || err}`);
    } finally {
      isExporting = false;
    }
  }

  function handleDocumentClick(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (!target.closest('.export-dropdown-wrapper')) {
      exportMenuOpen = false;
    }
  }

  function getStatusDotColor(state: string): string {
    switch (state) {
      case 'Running':
        return '#34c759';
      case 'Paused':
        return '#ff9500';
      case 'Completed':
        return '#007aff';
      default:
        return '#8e8e93';
    }
  }

  function getStatusBadgeClass(status: number): string {
    if (status >= 200 && status < 300) return 'badge-2xx';
    if (status >= 300 && status < 400) return 'badge-3xx';
    if (status >= 400 && status < 500) return 'badge-4xx';
    if (status >= 500) return 'badge-5xx';
    return 'badge-err';
  }

  function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  }
</script>

<svelte:window onkeydown={handleGlobalShortcuts} onclick={handleDocumentClick} />

{#if showSplash}
  <SplashScreen onComplete={() => (showSplash = false)} />
{:else}
  <div class="app-layout">
  <!-- Top Navigation & Window Title -->
  <TitleBar
    {activeView}
    onViewChange={(v) => (activeView = v)}
    onOpenAbout={() => (isAboutOpen = true)}
  />

  <!-- Scan Execution & Parameters Bar -->
  <ControlBar />

  <!-- Dual-Layer Advanced Filter Bar (Results Grid Only) -->
  {#if activeView === 'grid'}
    <FilterBar />
  {/if}

  <!-- Dynamic WAF / Wordlist Heuristic Advisory Banner -->
  <ResumeRecoveryBanner />
  <ScanAdvisoryBanner onOpenWordlists={() => (activeView = 'wordlists')} />

  <!-- Main View Area -->
  <main class="content-container">
    {#if activeView === 'grid'}
      {#if selectedItem}
        <SplitPane orientation="vertical" initialSplit={55} minSplit={25} maxSplit={75}>
          {#snippet first()}
            <VirtualGrid />
          {/snippet}
          {#snippet second()}
            <Inspector />
          {/snippet}
        </SplitPane>
      {:else}
        <VirtualGrid />
      {/if}
    {:else if activeView === 'analytics'}
      <AnalyticsDashboard />
    {:else if activeView === 'wordlists'}
      <WordlistManager onSelectForScan={() => (activeView = 'grid')} />
    {/if}
  </main>

  <!-- Bottom Telemetry & Status Bar -->
  <footer class="status-bar select-none">
    <div class="status-bar-left">
      <div class="status-indicator">
        <span class="status-dot" style="background-color: {getStatusDotColor(scanStore.state)};"></span>
        <span class="status-label">Engine: <strong>{scanStore.state}</strong></span>
      </div>

      <span class="v-divider"></span>
      <div class="net-telemetry font-mono" data-tooltip="Live network throughput (Upload TX, Download RX, Smoothed Req/s)">
        <span class="net-metric tx">
          <Icon name="arrow-up" size={10} />
          <span>{formatBytes(scanStore.bytesSent)}</span>
        </span>
        <span class="net-metric rx">
          <Icon name="arrow-down" size={10} />
          <span>{formatBytes(scanStore.bytesRecv)}</span>
        </span>
        <span class="net-metric rate">
          <Icon name="activity" size={10} />
          <span>{scanStore.smoothedReqSec}/s</span>
        </span>
      </div>

      {#if scanStore.proxyURL || scanStore.replayProxyURL}
        <span class="v-divider"></span>
        <button
          type="button"
          class="proxy-pill font-mono"
          onclick={() => (document.querySelector<HTMLButtonElement>('[aria-label="Request Options"]')?.click())}
          data-tooltip="Burp / Caido proxy active — click to configure"
        >
          <span class="proxy-dot"></span>
          <span>
            {scanStore.proxyURL
              ? `Proxy: ${scanStore.proxyURL.replace(/^https?:\/\//, '')}`
              : `Replay: ${scanStore.replayProxyURL.replace(/^https?:\/\//, '')}`}
          </span>
        </button>
      {/if}

      {#if selectedItem}
        <span class="v-divider"></span>
        <div class="selection-pill font-mono">
          <span class="pill-id">#{selectedItem.id}</span>
          <span class="status-badge {getStatusBadgeClass(selectedItem.status)}">{selectedItem.status}</span>
          <span class="pill-sep">&bull;</span>
          <span class="pill-metric">{formatBytes(selectedItem.length)}</span>
          <span class="pill-sep">&bull;</span>
          <span class="pill-metric">{selectedItem.duration_ms}ms</span>
        </div>
      {/if}
    </div>

    <div class="status-bar-right">
      <button
        type="button"
        class="netlog-btn font-mono"
        class:has-errors={scanStore.recentErrors.length > 0}
        onclick={() => (scanStore.showErrorDiagnostics = true)}
        data-tooltip="View live network diagnostics & error log"
      >
        <Icon name="terminal" size={10} />
        <span>Net Log</span>
        {#if scanStore.recentErrors.length > 0}
          <span class="netlog-badge">{scanStore.recentErrors.length}</span>
        {/if}
      </button>

      <div class="export-dropdown-wrapper">
        <button
          type="button"
          class="btn-export"
          class:active={exportMenuOpen}
          onclick={(e) => {
            e.stopPropagation();
            exportMenuOpen = !exportMenuOpen;
          }}
          disabled={isExporting}
          data-tooltip="Export scan results (JSON, CSV, TXT, HTML, Markdown)"
          aria-haspopup="true"
          aria-expanded={exportMenuOpen}
        >
          <Icon name="download" size={11} />
          <span>Export</span>
          <Icon name="chevron-down" size={9} />
        </button>

        {#if exportMenuOpen}
          <div class="export-menu" role="menu">
            <button
              type="button"
              class="export-menu-item"
              role="menuitem"
              onclick={() => handleExport('json')}
            >
              <div class="export-item-left">
                <span class="export-format-badge json">JSON</span>
                <span class="export-format-title">Structured Data</span>
              </div>
              <span class="export-format-ext">.json</span>
            </button>
            <button
              type="button"
              class="export-menu-item"
              role="menuitem"
              onclick={() => handleExport('csv')}
            >
              <div class="export-item-left">
                <span class="export-format-badge csv">CSV</span>
                <span class="export-format-title">Spreadsheet Table</span>
              </div>
              <span class="export-format-ext">.csv</span>
            </button>
            <button
              type="button"
              class="export-menu-item"
              role="menuitem"
              onclick={() => handleExport('txt')}
            >
              <div class="export-item-left">
                <span class="export-format-badge txt">TXT</span>
                <span class="export-format-title">Plain Text Lines</span>
              </div>
              <span class="export-format-ext">.txt</span>
            </button>
            <button
              type="button"
              class="export-menu-item"
              role="menuitem"
              onclick={() => handleExport('html')}
            >
              <div class="export-item-left">
                <span class="export-format-badge html">HTML</span>
                <span class="export-format-title">Interactive Report</span>
              </div>
              <span class="export-format-ext">.html</span>
            </button>
            <button
              type="button"
              class="export-menu-item"
              role="menuitem"
              onclick={() => handleExport('md')}
            >
              <div class="export-item-left">
                <span class="export-format-badge md">MD</span>
                <span class="export-format-title">Markdown Table</span>
              </div>
              <span class="export-format-ext">.md</span>
            </button>
            <button
              type="button"
              class="export-menu-item"
              role="menuitem"
              onclick={() => handleExport('ejson')}
            >
              <div class="export-item-left">
                <span class="export-format-badge json">EJSON</span>
                <span class="export-format-title">Base64 JSON</span>
              </div>
              <span class="export-format-ext">.ejson</span>
            </button>
            <button
              type="button"
              class="export-menu-item"
              role="menuitem"
              onclick={() => handleExport('ecsv')}
            >
              <div class="export-item-left">
                <span class="export-format-badge csv">ECSV</span>
                <span class="export-format-title">Base64 CSV</span>
              </div>
              <span class="export-format-ext">.ecsv</span>
            </button>
          </div>
        {/if}
      </div>

      <span class="v-divider"></span>

      <span class="font-mono total-label">
        Total: <strong>{resultsStore.items.length.toLocaleString()}</strong> items
      </span>
    </div>
  </footer>

  <!-- Advanced Query Builder Modal -->
  <AdvancedFilterModal />

  <!-- About FuzzNexus Modal -->
  <AboutModal isOpen={isAboutOpen} onClose={() => (isAboutOpen = false)} />

  <!-- Phase 5 Safe Stop IP Block Warning Modal -->
  <SafeStopModal />
  <RequestOptionsDrawer />
  <SessionHistoryModal />

  <!-- Network Error Diagnostics Modal -->
  <ErrorDiagnosticsModal
    isOpen={scanStore.showErrorDiagnostics}
    onClose={() => scanStore.closeErrorDiagnostics()}
  />

  <!-- Global Toast Notifications Overlay -->
  <ToastContainer />

  <!-- Global Keyboard Shortcuts Modal -->
  <ShortcutsModal />

  <!-- Global Viewport-Aware Tooltip Portal -->
  <GlobalTooltip />
</div>
{/if}

<style>
  .app-layout {
    display: flex;
    flex-direction: column;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    background-color: var(--bg);
    color: var(--text);
  }

  .content-container {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    position: relative;
    overflow: hidden;
  }

  .status-bar {
    height: 32px;
    background-color: var(--bg-subtle);
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 14px;
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-secondary);
    box-sizing: border-box;
    position: relative;
    z-index: 50;
  }

  .status-bar-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .status-indicator {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-xs);
  }

  .status-indicator .status-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    transition: background-color 0.2s ease;
  }

  .status-label strong {
    color: var(--text);
  }

  .v-divider {
    width: 1px;
    height: 14px;
    background-color: var(--border);
    display: inline-block;
  }

  .selection-pill {
    background-color: var(--surface);
    border: 1px solid var(--border);
    padding: 1px 7px;
    border-radius: var(--radius-xs);
    font-size: var(--text-xs);
    color: var(--text);
    display: inline-flex;
    align-items: center;
    gap: 5px;
    line-height: 14px;
  }

  .pill-id {
    color: var(--accent);
    font-weight: var(--weight-bold);
  }

  .pill-sep {
    color: var(--text-secondary);
    opacity: 0.6;
  }

  .pill-metric {
    color: var(--text);
  }

  .status-badge {
    padding: 1px 5px;
    border-radius: 3px;
    font-size: 10px;
    font-weight: var(--weight-semibold);
    line-height: 12px;
  }

  .badge-2xx {
    background-color: rgba(52, 199, 89, 0.15);
    color: var(--status-2xx);
    border: 1px solid rgba(52, 199, 89, 0.3);
  }

  .badge-3xx {
    background-color: rgba(0, 122, 255, 0.15);
    color: var(--status-3xx);
    border: 1px solid rgba(0, 122, 255, 0.3);
  }

  .badge-4xx {
    background-color: rgba(255, 149, 0, 0.15);
    color: var(--status-4xx);
    border: 1px solid rgba(255, 149, 0, 0.3);
  }

  .badge-5xx {
    background-color: rgba(255, 59, 48, 0.15);
    color: var(--status-5xx);
    border: 1px solid rgba(255, 59, 48, 0.3);
  }

  .badge-err {
    background-color: rgba(142, 142, 147, 0.15);
    color: var(--status-err);
    border: 1px solid rgba(142, 142, 147, 0.3);
  }

  .status-bar-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .net-telemetry {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    font-size: var(--text-xs);
    font-variant-numeric: tabular-nums;
    color: var(--text-muted);
  }

  .net-metric {
    display: inline-flex;
    align-items: center;
    gap: 3px;
  }

  .net-metric.tx {
    color: var(--accent, #3b82f6);
  }

  .net-metric.rx {
    color: var(--success, #10b981);
  }

  .proxy-pill {
    all: unset;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: var(--text-xs);
    padding: 0 7px;
    height: 20px;
    border-radius: var(--radius-sm);
    background-color: rgba(245, 158, 11, 0.12);
    border: 1px solid rgba(245, 158, 11, 0.35);
    color: var(--warning, #f59e0b);
    cursor: pointer;
    transition: background-color 0.12s ease;
  }

  .proxy-pill:hover {
    background-color: rgba(245, 158, 11, 0.22);
  }

  .proxy-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background-color: currentColor;
  }

  .netlog-btn {
    all: unset;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: var(--text-xs);
    padding: 0 7px;
    height: 22px;
    border-radius: var(--radius-sm);
    background-color: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-muted);
    cursor: pointer;
  }

  .netlog-btn:hover {
    color: var(--text);
    background-color: var(--surface-hover);
  }

  .netlog-btn.has-errors {
    border-color: rgba(239, 68, 68, 0.45);
    color: var(--danger, #ef4444);
  }

  .netlog-badge {
    font-size: 10px;
    font-weight: 700;
    padding: 0 4px;
    border-radius: 999px;
    background-color: rgba(239, 68, 68, 0.16);
  }

  .export-dropdown-wrapper {
    position: relative;
    display: inline-flex;
    align-items: center;
  }

  .btn-export {
    all: unset;
    font-size: var(--text-xs);
    font-weight: 500;
    padding: 0 8px;
    height: 22px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-xs);
    cursor: pointer;
    color: var(--text);
    display: inline-flex;
    align-items: center;
    gap: 5px;
    box-sizing: border-box;
    transition: all 0.12s cubic-bezier(0.16, 1, 0.3, 1);
    user-select: none;
  }

  .btn-export:hover:not(:disabled) {
    background-color: var(--surface-hover);
    border-color: var(--border-hover, var(--border));
    color: var(--text);
  }

  .btn-export.active {
    background-color: var(--surface-hover);
    border-color: var(--accent);
    color: var(--accent);
  }

  .btn-export:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .export-menu {
    position: absolute;
    bottom: calc(100% + 6px);
    right: 0;
    min-width: 215px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    box-shadow: 0 10px 28px rgba(0, 0, 0, 0.28), 0 2px 8px rgba(0, 0, 0, 0.12);
    padding: 4px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    z-index: 100;
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    animation: menu-slide-up 0.12s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes menu-slide-up {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .export-menu-item {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 8px;
    border-radius: var(--radius-xs);
    cursor: pointer;
    user-select: none;
    box-sizing: border-box;
    transition: background-color 0.12s ease;
  }

  .export-menu-item:hover {
    background-color: var(--surface-hover);
  }

  .export-item-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .export-format-badge {
    font-size: 10px;
    font-weight: 700;
    padding: 1px 5px;
    border-radius: 3px;
    font-family: var(--font-mono);
    line-height: 12px;
    letter-spacing: 0.3px;
  }

  .export-format-badge.json {
    background-color: rgba(0, 122, 255, 0.15);
    color: #0a84ff;
    border: 1px solid rgba(0, 122, 255, 0.3);
  }

  .export-format-badge.csv {
    background-color: rgba(52, 199, 89, 0.15);
    color: #30d158;
    border: 1px solid rgba(52, 199, 89, 0.3);
  }

  .export-format-badge.txt {
    background-color: rgba(255, 159, 10, 0.15);
    color: #ff9f0a;
    border: 1px solid rgba(255, 159, 10, 0.3);
  }

  .export-format-badge.html {
    background-color: rgba(168, 85, 247, 0.15);
    color: #c084fc;
    border: 1px solid rgba(168, 85, 247, 0.3);
  }

  .export-format-badge.md {
    background-color: rgba(56, 189, 248, 0.15);
    color: #38bdf8;
    border: 1px solid rgba(56, 189, 248, 0.3);
  }

  .export-format-title {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text);
  }

  .export-format-ext {
    font-size: 11px;
    color: var(--text-secondary);
    font-family: var(--font-mono);
  }

  .total-label {
    font-size: var(--text-xs);
    font-variant-numeric: tabular-nums;
  }

  .total-label strong {
    color: var(--text);
  }
</style>
