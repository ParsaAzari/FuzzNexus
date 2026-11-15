<script lang="ts">
  import TitleBar from './lib/components/TitleBar.svelte';
  import ControlBar from './lib/components/ControlBar.svelte';
  import VirtualGrid from './lib/components/VirtualGrid.svelte';
  import Inspector from './lib/components/Inspector.svelte';
  import SplitPane from './lib/components/SplitPane.svelte';
  import AnalyticsDashboard from './lib/components/AnalyticsDashboard.svelte';
  import { scanStore } from './lib/stores/scan.svelte';
  import { resultsStore } from './lib/stores/results.svelte';
  import { ExportResults } from '../wailsjs/go/main/App';

  type ViewMode = 'grid' | 'analytics';
  let activeView = $state<ViewMode>('grid');

  let selectedItem = $derived(resultsStore.selectedItem);
  let isExporting = $state<boolean>(false);
  let exportMessage = $state<string>('');

  async function handleExport(format: 'json' | 'csv') {
    if (scanStore.sessionID <= 0 || resultsStore.items.length === 0) return;
    isExporting = true;
    exportMessage = '';
    try {
      const fileName = `ffuf_session_${scanStore.sessionID}_results.${format}`;
      const targetPath = `${fileName}`;
      await ExportResults(scanStore.sessionID, format, targetPath);
      exportMessage = `Exported ${resultsStore.items.length} items to ${targetPath}`;
      setTimeout(() => (exportMessage = ''), 4000);
    } catch (err: any) {
      exportMessage = `Export failed: ${err?.message || err}`;
    } finally {
      isExporting = false;
    }
  }
</script>

<div class="app-layout">
  <!-- Top Navigation & Window Title -->
  <TitleBar
    {activeView}
    onViewChange={(v) => (activeView = v)}
  />

  <!-- Scan Execution & Parameters Bar -->
  <ControlBar />

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
    {:else}
      <AnalyticsDashboard />
    {/if}
  </main>

  <!-- Bottom Telemetry & Status Bar -->
  <footer class="status-bar select-none">
    <div class="status-bar-left">
      <span class="status-indicator">
        <span class="dot" class:active={scanStore.state === 'Running'}></span>
        Engine: <strong>{scanStore.state}</strong>
      </span>
      {#if selectedItem}
        <span class="selection-pill font-mono">
          Selected #{selectedItem.id} ({selectedItem.status} • {selectedItem.length} bytes • {selectedItem.duration_ms}ms)
        </span>
      {/if}
      {#if exportMessage}
        <span class="export-msg">{exportMessage}</span>
      {/if}
    </div>

    <div class="status-bar-right">
      <div class="export-group">
        <button
          class="btn-export"
          onclick={() => handleExport('json')}
          disabled={isExporting || resultsStore.items.length === 0}
          title="Export current scan results to JSON file"
        >
          Export JSON
        </button>
        <button
          class="btn-export"
          onclick={() => handleExport('csv')}
          disabled={isExporting || resultsStore.items.length === 0}
          title="Export current scan results to CSV file"
        >
          Export CSV
        </button>
      </div>
      <span class="font-mono total-label">
        Total: {resultsStore.items.length.toLocaleString()} items
      </span>
    </div>
  </footer>
</div>

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
    height: 24px;
    background-color: var(--bg-subtle);
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px;
    font-size: 11px;
    color: var(--text-secondary);
  }

  .status-bar-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .status-indicator {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .status-indicator .dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background-color: #8E8E93;
  }

  .status-indicator .dot.active {
    background-color: #34C759;
    box-shadow: 0 0 6px #34C759;
  }

  .status-indicator strong {
    color: var(--text);
  }

  .selection-pill {
    background-color: var(--surface);
    border: 1px solid var(--border);
    padding: 1px 6px;
    border-radius: 4px;
    font-size: 10px;
    color: var(--text);
  }

  .export-msg {
    color: var(--accent);
    font-weight: 500;
  }

  .status-bar-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .export-group {
    display: flex;
    gap: 4px;
  }

  .btn-export {
    all: unset;
    font-size: 10px;
    padding: 1px 6px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    cursor: pointer;
    color: var(--text);
    transition: background 0.1s ease;
  }

  .btn-export:hover:not(:disabled) {
    background-color: var(--surface-hover);
  }

  .btn-export:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .total-label {
    font-size: 10px;
  }
</style>
