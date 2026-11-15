<script lang="ts">
  import { themeStore } from '../stores/theme.svelte';
  import { scanStore } from '../stores/scan.svelte';
  import { resultsStore } from '../stores/results.svelte';

  let filteredCount = $derived(resultsStore.filteredItems.length);
  let totalMatches = $derived(resultsStore.items.length);
</script>

<header class="titlebar select-none" style="--wails-draggable: drag;">
  <div class="titlebar-left">
    <!-- macOS traffic lights space -->
    <div class="traffic-light-spacer"></div>
    <div class="app-branding">
      <span class="app-title">ffuf</span>
      <span class="app-version">v2</span>
    </div>
    {#if scanStore.sessionID > 0}
      <div class="session-badge font-mono">
        Session #{scanStore.sessionID}
      </div>
    {/if}
  </div>

  <div class="titlebar-center" style="--wails-draggable: no-drag;">
    <div class="search-box">
      <span class="search-icon">⚲</span>
      <input
        type="text"
        placeholder="Filter results (URL, status, payload)..."
        bind:value={resultsStore.quickFilter}
        class="search-input font-mono"
      />
      {#if resultsStore.quickFilter}
        <button class="clear-btn" onclick={() => (resultsStore.quickFilter = '')}>✕</button>
      {/if}
    </div>
    <div class="match-counter font-mono">
      {#if resultsStore.quickFilter}
        {filteredCount} / {totalMatches}
      {:else}
        {totalMatches} matches
      {/if}
    </div>
  </div>

  <div class="titlebar-right" style="--wails-draggable: no-drag;">
    <!-- Theme Palette Selector -->
    <div class="theme-picker">
      <button
        class="theme-btn"
        class:active={themeStore.palette === 'neutral'}
        onclick={() => themeStore.setPalette('neutral')}
        title="Apple Neutral Theme"
      >
        Neutral
      </button>
      <button
        class="theme-btn"
        class:active={themeStore.palette === 'ocean'}
        onclick={() => themeStore.setPalette('ocean')}
        title="Apple Ocean Theme"
      >
        Ocean
      </button>
    </div>

    <!-- Dark/Light Mode Toggle -->
    <button
      class="mode-toggle-btn"
      onclick={() => themeStore.toggleMode()}
      title="Mode: {themeStore.mode} (click to toggle)"
    >
      {#if themeStore.mode === 'light'}
        ☀️ Light
      {:else if themeStore.mode === 'dark'}
        🌙 Dark
      {:else}
        ⚙️ Auto
      {/if}
    </button>
  </div>
</header>

<style>
  .titlebar {
    height: 44px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px;
    box-sizing: border-box;
    font-size: 13px;
  }

  .titlebar-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .traffic-light-spacer {
    width: 68px; /* Safe inset for macOS native red/yellow/green window buttons */
    height: 100%;
  }

  .app-branding {
    display: flex;
    align-items: baseline;
    gap: 4px;
  }

  .app-title {
    font-weight: 700;
    font-size: 15px;
    color: var(--text);
    letter-spacing: -0.02em;
  }

  .app-version {
    font-size: 10px;
    font-weight: 600;
    color: var(--accent);
    background: rgba(0, 122, 255, 0.1);
    padding: 1px 4px;
    border-radius: 4px;
  }

  .session-badge {
    font-size: 11px;
    color: var(--text-secondary);
    background-color: var(--surface);
    padding: 2px 8px;
    border-radius: 4px;
    border: 1px solid var(--border);
  }

  .titlebar-center {
    display: flex;
    align-items: center;
    gap: 10px;
    flex: 1;
    max-width: 480px;
    margin: 0 16px;
  }

  .search-box {
    position: relative;
    display: flex;
    align-items: center;
    flex: 1;
  }

  .search-icon {
    position: absolute;
    left: 8px;
    font-size: 12px;
    color: var(--text-secondary);
    pointer-events: none;
  }

  .search-input {
    width: 100%;
    height: 26px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 0 24px 0 24px;
    font-size: 11px;
    color: var(--text);
    box-sizing: border-box;
    outline: none;
    transition: border-color 0.15s ease, box-shadow 0.15s ease;
  }

  .search-input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px var(--focus-ring);
  }

  .clear-btn {
    position: absolute;
    right: 6px;
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 10px;
    padding: 2px;
  }

  .match-counter {
    font-size: 11px;
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .titlebar-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .theme-picker {
    display: flex;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 2px;
  }

  .theme-btn {
    all: unset;
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 4px;
    cursor: pointer;
    color: var(--text-secondary);
    transition: all 0.12s ease;
  }

  .theme-btn.active {
    background-color: var(--bg-subtle);
    color: var(--text);
    font-weight: 600;
  }

  .mode-toggle-btn {
    all: unset;
    font-size: 11px;
    padding: 3px 8px;
    border-radius: 6px;
    border: 1px solid var(--border);
    background-color: var(--surface);
    cursor: pointer;
    color: var(--text);
    transition: background 0.12s ease;
  }

  .mode-toggle-btn:hover {
    background-color: var(--surface-hover);
  }
</style>
