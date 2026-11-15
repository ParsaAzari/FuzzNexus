<script lang="ts">
  import { resultsStore } from '../stores/results.svelte';
  import { scanStore } from '../stores/scan.svelte';

  let items = $derived(resultsStore.items);
  let totalItems = $derived(items.length);

  // Status code breakdown
  let statusStats = $derived.by(() => {
    let s2xx = 0;
    let s3xx = 0;
    let s4xx = 0;
    let s5xx = 0;
    let other = 0;

    for (const item of items) {
      const code = item.status;
      if (code >= 200 && code < 300) s2xx++;
      else if (code >= 300 && code < 400) s3xx++;
      else if (code >= 400 && code < 500) s4xx++;
      else if (code >= 500) s5xx++;
      else other++;
    }

    return {
      s2xx,
      s3xx,
      s4xx,
      s5xx,
      other,
      p2xx: totalItems ? (s2xx / totalItems) * 100 : 0,
      p3xx: totalItems ? (s3xx / totalItems) * 100 : 0,
      p4xx: totalItems ? (s4xx / totalItems) * 100 : 0,
      p5xx: totalItems ? (s5xx / totalItems) * 100 : 0
    };
  });

  // Size distribution buckets
  let sizeBuckets = $derived.by(() => {
    const buckets = [
      { label: '< 500 B', count: 0 },
      { label: '500 B - 2 KB', count: 0 },
      { label: '2 KB - 10 KB', count: 0 },
      { label: '10 KB - 50 KB', count: 0 },
      { label: '> 50 KB', count: 0 }
    ];

    for (const item of items) {
      const len = item.length;
      if (len < 500) buckets[0].count++;
      else if (len < 2048) buckets[1].count++;
      else if (len < 10240) buckets[2].count++;
      else if (len < 51200) buckets[3].count++;
      else buckets[4].count++;
    }

    const maxCount = Math.max(1, ...buckets.map((b) => b.count));
    return buckets.map((b) => ({
      ...b,
      percentage: totalItems ? (b.count / totalItems) * 100 : 0,
      barHeight: (b.count / maxCount) * 100
    }));
  });

  // Average duration
  let avgDuration = $derived.by(() => {
    if (totalItems === 0) return 0;
    const sum = items.reduce((acc, curr) => acc + curr.duration_ms, 0);
    return Math.round(sum / totalItems);
  });

  // Hit rate
  let hitRate = $derived.by(() => {
    if (scanStore.reqCount === 0) return '0.00%';
    return ((totalItems / scanStore.reqCount) * 100).toFixed(2) + '%';
  });

  // Top word count clusters
  let wordClusters = $derived.by(() => {
    const counts = new Map<number, number>();
    for (const item of items) {
      counts.set(item.words, (counts.get(item.words) || 0) + 1);
    }
    const sorted = Array.from(counts.entries())
      .map(([words, count]) => ({ words, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 5);
    return sorted;
  });
</script>

<div class="analytics-wrapper select-none">
  <!-- Top KPI Cards -->
  <div class="kpi-grid">
    <div class="kpi-card">
      <div class="kpi-title">Total Matches</div>
      <div class="kpi-value font-mono">{totalItems.toLocaleString()}</div>
      <div class="kpi-sub">Filtered responses found</div>
    </div>

    <div class="kpi-card">
      <div class="kpi-title">Total Scanned</div>
      <div class="kpi-value font-mono">{scanStore.reqCount.toLocaleString()}</div>
      <div class="kpi-sub">
        {#if scanStore.reqTotal > 0}
          of {scanStore.reqTotal.toLocaleString()} requests
        {:else}
          Requests executed
        {/if}
      </div>
    </div>

    <div class="kpi-card">
      <div class="kpi-title">Hit Rate</div>
      <div class="kpi-value font-mono accent-text">{hitRate}</div>
      <div class="kpi-sub">Matches / Total scanned</div>
    </div>

    <div class="kpi-card">
      <div class="kpi-title">Average Latency</div>
      <div class="kpi-value font-mono">{avgDuration} ms</div>
      <div class="kpi-sub">Round-trip response time</div>
    </div>

    <div class="kpi-card">
      <div class="kpi-title">Current Speed</div>
      <div class="kpi-value font-mono">{scanStore.reqSec.toLocaleString()} req/s</div>
      <div class="kpi-sub">{scanStore.errorCount} total errors</div>
    </div>
  </div>

  <!-- Charts Grid -->
  <div class="charts-grid">
    <!-- Status Code Proportional Bar -->
    <div class="chart-card">
      <div class="card-header">
        <span class="card-title">Status Code Distribution</span>
        <span class="card-caption font-mono">{totalItems} results</span>
      </div>

      <!-- Segmented Bar -->
      <div class="segmented-bar">
        {#if statusStats.p2xx > 0}
          <div class="bar-segment bar-2xx" style="width: {statusStats.p2xx}%;" title="2xx: {statusStats.s2xx}"></div>
        {/if}
        {#if statusStats.p3xx > 0}
          <div class="bar-segment bar-3xx" style="width: {statusStats.p3xx}%;" title="3xx: {statusStats.s3xx}"></div>
        {/if}
        {#if statusStats.p4xx > 0}
          <div class="bar-segment bar-4xx" style="width: {statusStats.p4xx}%;" title="4xx: {statusStats.s4xx}"></div>
        {/if}
        {#if statusStats.p5xx > 0}
          <div class="bar-segment bar-5xx" style="width: {statusStats.p5xx}%;" title="5xx: {statusStats.s5xx}"></div>
        {/if}
      </div>

      <!-- Legend -->
      <div class="legend-table">
        <div class="legend-row">
          <span class="legend-dot dot-2xx"></span>
          <span class="legend-label">2xx Success</span>
          <span class="legend-count font-mono">{statusStats.s2xx}</span>
          <span class="legend-pct font-mono">{statusStats.p2xx.toFixed(1)}%</span>
        </div>
        <div class="legend-row">
          <span class="legend-dot dot-3xx"></span>
          <span class="legend-label">3xx Redirection</span>
          <span class="legend-count font-mono">{statusStats.s3xx}</span>
          <span class="legend-pct font-mono">{statusStats.p3xx.toFixed(1)}%</span>
        </div>
        <div class="legend-row">
          <span class="legend-dot dot-4xx"></span>
          <span class="legend-label">4xx Client Error</span>
          <span class="legend-count font-mono">{statusStats.s4xx}</span>
          <span class="legend-pct font-mono">{statusStats.p4xx.toFixed(1)}%</span>
        </div>
        <div class="legend-row">
          <span class="legend-dot dot-5xx"></span>
          <span class="legend-label">5xx Server Error</span>
          <span class="legend-count font-mono">{statusStats.s5xx}</span>
          <span class="legend-pct font-mono">{statusStats.p5xx.toFixed(1)}%</span>
        </div>
      </div>
    </div>

    <!-- Response Size Histogram -->
    <div class="chart-card">
      <div class="card-header">
        <span class="card-title">Response Size Histogram</span>
        <span class="card-caption">Content-Length Clusters</span>
      </div>

      <div class="histogram-container">
        {#each sizeBuckets as bucket}
          <div class="hist-bar-col">
            <span class="hist-count font-mono">{bucket.count}</span>
            <div class="hist-track">
              <div
                class="hist-fill"
                style="height: {Math.max(4, bucket.barHeight)}%;"
                title="{bucket.label}: {bucket.count} items ({bucket.percentage.toFixed(1)}%)"
              ></div>
            </div>
            <span class="hist-label">{bucket.label}</span>
          </div>
        {/each}
      </div>
    </div>

    <!-- Common Word Clusters -->
    <div class="chart-card">
      <div class="card-header">
        <span class="card-title">Recurring Word Clusters</span>
        <span class="card-caption">Useful for Calibrating -fw Filter</span>
      </div>

      {#if wordClusters.length === 0}
        <div class="empty-cluster">No word clusters available yet</div>
      {:else}
        <div class="cluster-list">
          {#each wordClusters as cluster}
            <div class="cluster-row">
              <div class="cluster-info">
                <span class="cluster-badge font-mono">{cluster.words} words</span>
                <span class="cluster-count font-mono">{cluster.count} responses</span>
              </div>
              <button
                class="btn-quick-filter"
                onclick={() => scanStore.addFilter('words', String(cluster.words))}
                title="Add filter -fw {cluster.words} to live scan"
              >
                + Filter -fw {cluster.words}
              </button>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .analytics-wrapper {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    overflow-y: auto;
    height: 100%;
    box-sizing: border-box;
    background-color: var(--bg);
  }

  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 12px;
  }

  .kpi-card {
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 14px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  }

  .kpi-title {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .kpi-value {
    font-size: 24px;
    font-weight: 700;
    color: var(--text);
    margin: 6px 0 2px 0;
  }

  .accent-text {
    color: var(--accent);
  }

  .kpi-sub {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .charts-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 16px;
  }

  .chart-card {
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  }

  .card-header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    margin-bottom: 14px;
  }

  .card-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--text);
  }

  .card-caption {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .segmented-bar {
    display: flex;
    height: 14px;
    border-radius: 7px;
    overflow: hidden;
    background-color: var(--bg-subtle);
    margin-bottom: 14px;
  }

  .bar-segment {
    height: 100%;
    transition: width 0.3s ease;
  }

  .bar-2xx {
    background-color: #34C759;
  }

  .bar-3xx {
    background-color: #007AFF;
  }

  .bar-4xx {
    background-color: #FF9500;
  }

  .bar-5xx {
    background-color: #FF3B30;
  }

  .legend-table {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .legend-row {
    display: flex;
    align-items: center;
    font-size: 11px;
  }

  .legend-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    margin-right: 8px;
  }

  .dot-2xx {
    background-color: #34C759;
  }

  .dot-3xx {
    background-color: #007AFF;
  }

  .dot-4xx {
    background-color: #FF9500;
  }

  .dot-5xx {
    background-color: #FF3B30;
  }

  .legend-label {
    flex: 1;
    color: var(--text);
  }

  .legend-count {
    width: 60px;
    text-align: right;
    color: var(--text-secondary);
  }

  .legend-pct {
    width: 50px;
    text-align: right;
    font-weight: 600;
    color: var(--text);
  }

  .histogram-container {
    display: flex;
    align-items: flex-end;
    justify-content: space-around;
    height: 140px;
    padding-top: 10px;
    gap: 8px;
  }

  .hist-bar-col {
    display: flex;
    flex-direction: column;
    align-items: center;
    flex: 1;
    height: 100%;
  }

  .hist-count {
    font-size: 10px;
    color: var(--text-secondary);
    margin-bottom: 4px;
  }

  .hist-track {
    flex: 1;
    width: 28px;
    background-color: var(--bg-subtle);
    border-radius: 4px 4px 0 0;
    display: flex;
    align-items: flex-end;
    overflow: hidden;
  }

  .hist-fill {
    width: 100%;
    background: linear-gradient(180deg, var(--accent) 0%, rgba(0, 122, 255, 0.6) 100%);
    border-radius: 4px 4px 0 0;
    transition: height 0.3s ease;
  }

  .hist-label {
    font-size: 10px;
    color: var(--text-secondary);
    margin-top: 6px;
    text-align: center;
    white-space: nowrap;
  }

  .cluster-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .cluster-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 10px;
    background-color: var(--bg-subtle);
    border-radius: 6px;
  }

  .cluster-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .cluster-badge {
    background-color: var(--surface);
    border: 1px solid var(--border);
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text);
  }

  .cluster-count {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .btn-quick-filter {
    all: unset;
    font-size: 10px;
    padding: 3px 8px;
    border-radius: 4px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    cursor: pointer;
    color: var(--accent);
    font-weight: 600;
  }

  .btn-quick-filter:hover {
    background-color: var(--accent);
    color: var(--accent-contrast);
  }

  .empty-cluster {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 120px;
    color: var(--text-secondary);
    font-size: 12px;
  }
</style>
