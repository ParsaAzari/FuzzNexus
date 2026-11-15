<script lang="ts">
  import { resultsStore } from '../stores/results.svelte';
  import { scanStore } from '../stores/scan.svelte';
  import { GetResultPayload } from '../../../wailsjs/go/main/App';
  import type { PayloadDTO } from '../types';
  import Icon from './Icon.svelte';

  type TabType = 'response' | 'request' | 'metadata';
  type ViewMode = 'raw' | 'preview';

  let activeTab = $state<TabType>('response');
  let viewMode = $state<ViewMode>('raw');
  let payload = $state<PayloadDTO | null>(null);
  let isLoading = $state<boolean>(false);
  let loadError = $state<string>('');
  let copied = $state<boolean>(false);

  let selectedItem = $derived(resultsStore.selectedItem);

  $effect(() => {
    const item = selectedItem;
    if (item && scanStore.sessionID > 0) {
      loadPayload(scanStore.sessionID, item.id);
    } else {
      payload = null;
      loadError = '';
    }
  });

  async function loadPayload(sessionID: number, resultID: number) {
    isLoading = true;
    loadError = '';
    try {
      const data = await GetResultPayload(sessionID, resultID, 500 * 1024);
      payload = data;
    } catch (err: any) {
      loadError = err?.message || String(err);
      payload = null;
    } finally {
      isLoading = false;
    }
  }

  function copyToClipboard(text: string) {
    if (!text) return;
    navigator.clipboard.writeText(text).then(() => {
      copied = true;
      setTimeout(() => (copied = false), 2000);
    });
  }

  // Construct secure srcdoc for sandboxed iframe
  let previewSrcdoc = $derived.by(() => {
    if (!payload || !payload.response_text) return '';
    const body = payload.response_text;
    return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline';">
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      font-size: 13px;
      line-height: 1.5;
      padding: 14px;
      margin: 0;
      color: #1d1d1f;
      background-color: #ffffff;
      word-break: break-word;
    }
  </style>
</head>
<body>
${body}
</body>
</html>`;
  });
</script>

<div class="inspector-wrapper select-none">
  {#if !selectedItem}
    <div class="empty-inspector">
      <div class="empty-icon">
        <Icon name="file-text" size={36} />
      </div>
      <div class="empty-text">Select a row from the grid to inspect HTTP request & response details</div>
    </div>
  {:else}
    <!-- Inspector Header Bar -->
    <div class="inspector-header">
      <div class="header-left">
        <span class="inspector-title">Inspector</span>
        <span class="item-badge font-mono">#{selectedItem.id}</span>
        <span class="url-badge font-mono" title={selectedItem.url}>{selectedItem.url}</span>
      </div>

      <!-- Tab Buttons -->
      <div class="tab-group">
        <button
          class="tab-btn"
          class:active={activeTab === 'response'}
          onclick={() => (activeTab = 'response')}
        >
          Response
        </button>
        <button
          class="tab-btn"
          class:active={activeTab === 'request'}
          onclick={() => (activeTab = 'request')}
        >
          Request
        </button>
        <button
          class="tab-btn"
          class:active={activeTab === 'metadata'}
          onclick={() => (activeTab = 'metadata')}
        >
          Metadata
        </button>
      </div>

      <div class="header-right">
        {#if activeTab === 'response'}
          <div class="view-mode-toggle">
            <button
              class="toggle-btn"
              class:active={viewMode === 'raw'}
              onclick={() => (viewMode = 'raw')}
            >
              Raw
            </button>
            <button
              class="toggle-btn"
              class:active={viewMode === 'preview'}
              onclick={() => (viewMode = 'preview')}
              title="Render in sandboxed iframe"
            >
              Preview
            </button>
          </div>
        {/if}

        <button
          class="btn-copy"
          onclick={() => {
            if (activeTab === 'response') copyToClipboard(payload?.response_text || '');
            else if (activeTab === 'request') copyToClipboard(payload?.request_text || '');
            else copyToClipboard(JSON.stringify(selectedItem, null, 2));
          }}
          aria-label="Copy content"
        >
          {#if copied}
            <Icon name="check" size={14} />
            <span>Copied</span>
          {:else}
            <Icon name="copy" size={14} />
            <span>Copy</span>
          {/if}
        </button>

        <button
          class="btn-close"
          onclick={() => resultsStore.select(null)}
          title="Close Inspector"
          aria-label="Close Inspector"
        >
          <Icon name="x" size={14} />
        </button>
      </div>
    </div>

    <!-- Inspector Content Area -->
    <div class="inspector-content">
      {#if isLoading}
        <div class="loading-state">
          <div class="spinner"></div>
          <span>Loading payload...</span>
        </div>
      {:else if loadError}
        <div class="error-state">
          <span>Failed to load payload: {loadError}</span>
        </div>
      {:else if activeTab === 'response'}
        {#if payload?.truncated}
          <div class="truncation-alert">
            <Icon name="alert-triangle" size={16} />
            <span>Response truncated to 500KB (Original size: {(payload.full_size_bytes / 1024).toFixed(1)} KB)</span>
          </div>
        {/if}

        {#if viewMode === 'raw'}
          <pre class="code-viewer font-mono select-text">{payload?.response_text || '(Empty Response Body)'}</pre>
        {:else}
          <div class="preview-container">
            <iframe
              sandbox=""
              title="Sandboxed HTML Preview"
              srcdoc={previewSrcdoc}
              class="preview-iframe"
            ></iframe>
          </div>
        {/if}
      {:else if activeTab === 'request'}
        <pre class="code-viewer font-mono select-text">{payload?.request_text || '(No Raw Request Recorded)'}</pre>
      {:else if activeTab === 'metadata'}
        <div class="metadata-view select-text">
          <table class="meta-table font-mono">
            <tbody>
              <tr>
                <td class="meta-key">Result ID</td>
                <td class="meta-val">{selectedItem.id}</td>
              </tr>
              <tr>
                <td class="meta-key">Status Code</td>
                <td class="meta-val">{selectedItem.status}</td>
              </tr>
              <tr>
                <td class="meta-key">Content Length</td>
                <td class="meta-val">{selectedItem.length} bytes</td>
              </tr>
              <tr>
                <td class="meta-key">Words / Lines</td>
                <td class="meta-val">{selectedItem.words} words <span class="meta-sep">&bull;</span> {selectedItem.lines} lines</td>
              </tr>
              <tr>
                <td class="meta-key">Duration</td>
                <td class="meta-val">{selectedItem.duration_ms} ms</td>
              </tr>
              <tr>
                <td class="meta-key">Content-Type</td>
                <td class="meta-val">{payload?.content_type || selectedItem.content_type || 'Unknown'}</td>
              </tr>
              <tr>
                <td class="meta-key">Redirect Target</td>
                <td class="meta-val">{selectedItem.redirect_location || '(None)'}</td>
              </tr>
              <tr>
                <td class="meta-key">FFUF Hash</td>
                <td class="meta-val">{selectedItem.hash || '(None)'}</td>
              </tr>
              {#if selectedItem.inputs}
                {#each Object.entries(selectedItem.inputs) as [k, v]}
                  <tr>
                    <td class="meta-key">Input: {k}</td>
                    <td class="meta-val">{v}</td>
                  </tr>
                {/each}
              {/if}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .inspector-wrapper {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
    background-color: var(--surface);
    overflow: hidden;
  }

  .empty-inspector {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
    font-size: 12px;
    gap: 8px;
  }

  .empty-icon {
    font-size: 32px;
    opacity: 0.5;
  }

  .inspector-header {
    height: 36px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 10px;
    box-sizing: border-box;
    font-size: 11px;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 8px;
    overflow: hidden;
  }

  .inspector-title {
    font-weight: 700;
    color: var(--text);
  }

  .item-badge {
    background-color: var(--surface);
    border: 1px solid var(--border);
    padding: 1px 6px;
    border-radius: 4px;
    color: var(--accent);
    font-weight: 600;
  }

  .url-badge {
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 250px;
  }

  .tab-group {
    display: flex;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 2px;
  }

  .tab-btn {
    all: unset;
    padding: 2px 8px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 11px;
    color: var(--text-secondary);
    transition: all 0.12s ease;
  }

  .tab-btn.active {
    background-color: var(--bg-subtle);
    color: var(--text);
    font-weight: 600;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .view-mode-toggle {
    display: flex;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 2px;
  }

  .toggle-btn {
    all: unset;
    padding: 1px 6px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 10px;
    color: var(--text-secondary);
  }

  .toggle-btn.active {
    background-color: var(--bg-subtle);
    color: var(--text);
    font-weight: 600;
  }

  .btn-copy {
    all: unset;
    font-size: 10px;
    padding: 2px 8px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    cursor: pointer;
    color: var(--text);
  }

  .btn-copy:hover {
    background-color: var(--surface-hover);
  }

  .btn-close {
    all: unset;
    font-size: 11px;
    padding: 2px 6px;
    color: var(--text-secondary);
    cursor: pointer;
  }

  .btn-close:hover {
    color: var(--text);
  }

  .inspector-content {
    flex: 1;
    overflow-y: auto;
    position: relative;
    display: flex;
    flex-direction: column;
  }

  .truncation-alert {
    background-color: rgba(255, 149, 0, 0.15);
    color: #c97700;
    padding: 6px 12px;
    font-size: 11px;
    font-weight: 500;
    border-bottom: 1px solid rgba(255, 149, 0, 0.3);
  }

  .code-viewer {
    margin: 0;
    padding: 12px;
    font-size: 11px;
    line-height: 1.5;
    color: var(--text);
    white-space: pre-wrap;
    word-break: break-all;
    overflow-y: auto;
    flex: 1;
  }

  .preview-container {
    flex: 1;
    width: 100%;
    height: 100%;
    position: relative;
  }

  .preview-iframe {
    width: 100%;
    height: 100%;
    border: none;
    background: #ffffff;
  }

  .metadata-view {
    padding: 12px;
    overflow-y: auto;
  }

  .meta-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 11px;
  }

  .meta-table tr {
    border-bottom: 1px solid var(--border);
  }

  .meta-key {
    padding: 6px 8px;
    width: 160px;
    color: var(--text-secondary);
    font-weight: 600;
  }

  .meta-val {
    padding: 6px 8px;
    color: var(--text);
  }

  .loading-state,
  .error-state {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 100%;
    color: var(--text-secondary);
    font-size: 12px;
  }

  .error-state {
    color: #FF3B30;
  }

  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
