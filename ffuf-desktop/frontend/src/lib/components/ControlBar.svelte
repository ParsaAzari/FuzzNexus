<script lang="ts">
  import { scanStore } from '../stores/scan.svelte';
  import Icon from './Icon.svelte';

  let liveFilterType = $state<'status' | 'size' | 'words' | 'lines'>('status');
  let liveFilterValue = $state<string>('');

  function handleAddLiveFilter() {
    if (!liveFilterValue.trim()) return;
    scanStore.addFilter(liveFilterType, liveFilterValue.trim());
    liveFilterValue = '';
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
</script>

<div class="control-bar-wrapper select-none">
  <!-- Top Row: Configuration Inputs -->
  <div class="config-row">
    <!-- Method Selector -->
    <select bind:value={scanStore.method} disabled={scanStore.state === 'Running'} class="method-select font-mono">
      <option value="GET">GET</option>
      <option value="POST">POST</option>
      <option value="PUT">PUT</option>
      <option value="DELETE">DELETE</option>
      <option value="HEAD">HEAD</option>
      <option value="PATCH">PATCH</option>
    </select>

    <!-- Target URL Input -->
    <div class="input-group url-group">
      <span class="input-label">URL</span>
      <input
        type="text"
        placeholder="https://example.com/FUZZ"
        bind:value={scanStore.url}
        disabled={scanStore.state === 'Running'}
        class="text-input font-mono"
      />
    </div>

    <!-- Wordlist Path Input -->
    <div class="input-group wordlist-group">
      <span class="input-label">Wordlist</span>
      <input
        type="text"
        placeholder="/path/to/wordlist.txt"
        bind:value={scanStore.wordlist}
        disabled={scanStore.state === 'Running'}
        class="text-input font-mono"
      />
    </div>

    <!-- Threads Input -->
    <div class="input-group compact-group">
      <span class="input-label">Threads</span>
      <input
        type="number"
        min="1"
        max="100"
        bind:value={scanStore.threads}
        disabled={scanStore.state === 'Running'}
        class="text-input font-mono compact-input"
      />
    </div>

    <!-- Rate Limit Input -->
    <div class="input-group compact-group">
      <span class="input-label">Rate (req/s)</span>
      <input
        type="number"
        min="0"
        max="5000"
        bind:value={scanStore.rate}
        onchange={() => scanStore.changeRate(scanStore.rate)}
        class="text-input font-mono compact-input"
        title="0 = unlimited. Can be changed live during scan."
      />
    </div>
  </div>

  <!-- Bottom Row: Execution Controls & Live Telemetry -->
  <div class="actions-row">
    <div class="buttons-group">
      {#if scanStore.state !== 'Running' && scanStore.state !== 'Paused'}
        <button
          class="btn btn-primary"
          onclick={() => scanStore.start()}
          disabled={scanStore.isSubmitting || !scanStore.url || !scanStore.wordlist}
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

      <!-- Status Pill -->
      <div class="status-pill">
        <span class="status-dot" style="background-color: {getStatusDotColor(scanStore.state)};"></span>
        <span class="status-text">{scanStore.state}</span>
      </div>
    </div>

    <!-- Live Telemetry & Progress Bar -->
    <div class="telemetry-group">
      {#if scanStore.reqTotal > 0}
        <div class="progress-container">
          <div class="progress-bar-bg">
            <div class="progress-bar-fill" style="width: {scanStore.percentage.toFixed(1)}%;"></div>
          </div>
          <span class="progress-label font-mono">
            {scanStore.percentage.toFixed(1)}% ({scanStore.reqCount.toLocaleString()} / {scanStore.reqTotal.toLocaleString()})
          </span>
        </div>
      {/if}

      <div class="metrics-pill font-mono">
        <span class="metric-item">
          <strong>{scanStore.reqSec.toLocaleString()}</strong> req/s
        </span>
        {#if scanStore.errorCount > 0}
          <span class="metric-item metric-error">
            <strong>{scanStore.errorCount}</strong> errors
          </span>
        {/if}
      </div>
    </div>

    <!-- Live Filter Adder -->
    <div class="live-filter-group">
      <span class="filter-icon">
        <Icon name="zap" size={14} />
      </span>
      <select bind:value={liveFilterType} class="filter-type-select">
        <option value="status">Status</option>
        <option value="size">Size</option>
        <option value="words">Words</option>
        <option value="lines">Lines</option>
      </select>
      <input
        type="text"
        placeholder="Filter value (e.g. 404)"
        bind:value={liveFilterValue}
        onkeydown={(e) => e.key === 'Enter' && handleAddLiveFilter()}
        class="filter-input font-mono"
      />
      <button class="btn btn-subtle" onclick={handleAddLiveFilter}>
        + Filter
      </button>
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
    height: 28px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 0 8px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text);
    outline: none;
  }

  .input-group {
    display: flex;
    align-items: center;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 6px;
    height: 28px;
    padding: 0 8px;
    box-sizing: border-box;
    transition: border-color 0.15s ease;
  }

  .input-group:focus-within {
    border-color: var(--accent);
  }

  .url-group {
    flex: 2;
    min-width: 200px;
  }

  .wordlist-group {
    flex: 2;
    min-width: 180px;
  }

  .compact-group {
    width: 110px;
  }

  .input-label {
    font-size: 10px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    margin-right: 6px;
    user-select: none;
    white-space: nowrap;
  }

  .text-input {
    all: unset;
    flex: 1;
    font-size: 11px;
    color: var(--text);
    width: 100%;
  }

  .compact-input {
    text-align: right;
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
    height: 26px;
    padding: 0 12px;
    border-radius: 6px;
    font-size: 11px;
    font-weight: 600;
    display: flex;
    align-items: center;
    justify-content: center;
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
    filter: brightness(1.1);
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

  .btn-subtle {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    color: var(--text);
    padding: 0 8px;
  }

  .btn-subtle:hover {
    background-color: var(--surface-hover);
  }

  .status-pill {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 3px 8px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 12px;
    font-size: 11px;
    font-weight: 500;
  }

  .status-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
  }

  .telemetry-group {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    margin: 0 8px;
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
    font-size: 11px;
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .metrics-pill {
    display: flex;
    align-items: center;
    gap: 8px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    padding: 3px 8px;
    border-radius: 6px;
    font-size: 11px;
    color: var(--text-secondary);
  }

  .metric-item strong {
    color: var(--text);
  }

  .metric-error {
    color: #FF3B30 !important;
  }

  .metric-error strong {
    color: #FF3B30 !important;
  }

  .live-filter-group {
    display: flex;
    align-items: center;
    gap: 4px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 2px 4px;
  }

  .filter-icon {
    font-size: 12px;
    color: var(--text-secondary);
  }

  .filter-type-select {
    background: none;
    border: none;
    font-size: 11px;
    color: var(--text);
    outline: none;
    cursor: pointer;
  }

  .filter-input {
    width: 90px;
    background: none;
    border: none;
    font-size: 11px;
    color: var(--text);
    outline: none;
  }

  .error-banner {
    background-color: rgba(255, 59, 48, 0.1);
    color: #FF3B30;
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 11px;
    border: 1px solid rgba(255, 59, 48, 0.2);
  }
</style>
