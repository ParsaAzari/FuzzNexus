<script lang="ts">
  import { tick } from 'svelte';
  import { themeStore } from '../stores/theme.svelte';
  import { scanStore } from '../stores/scan.svelte';
  import { shortcutsStore } from '../stores/shortcuts.svelte';
  import { WindowToggleMaximise } from '../../../wailsjs/runtime/runtime';
  import Icon from './Icon.svelte';
  import ThemePicker from './ThemePicker.svelte';
  import Logo from './Logo.svelte';

  interface Props {
    activeView?: 'grid' | 'analytics' | 'wordlists';
    onViewChange?: (view: 'grid' | 'analytics' | 'wordlists') => void;
    onOpenAbout?: () => void;
  }

  let { activeView = 'grid', onViewChange, onOpenAbout }: Props = $props();

  let btnGrid = $state<HTMLButtonElement | null>(null);
  let btnAnalytics = $state<HTMLButtonElement | null>(null);
  let btnWordlists = $state<HTMLButtonElement | null>(null);

  let pillLeft = $state<number>(2);
  let pillWidth = $state<number>(0);
  let isPillReady = $state<boolean>(false);

  function updatePill(animate = true) {
    let targetBtn: HTMLButtonElement | null = null;
    if (activeView === 'grid') targetBtn = btnGrid;
    else if (activeView === 'analytics') targetBtn = btnAnalytics;
    else if (activeView === 'wordlists') targetBtn = btnWordlists;

    if (targetBtn) {
      pillLeft = targetBtn.offsetLeft;
      pillWidth = targetBtn.offsetWidth;
      isPillReady = true;
    }
  }

  $effect(() => {
    // Re-calculate when activeView changes or elements bind
    activeView;
    tick().then(() => updatePill());
  });

  function handleTitlebarDblClick(e: MouseEvent) {
    const target = e.target as HTMLElement | null;
    if (target && target.closest('button, input, select, a, [style*="no-drag"]')) {
      return;
    }
    try {
      WindowToggleMaximise();
    } catch (err) {
      console.warn('Native WindowToggleMaximise not supported in current environment:', err);
    }
  }
</script>

<header
  class="titlebar select-none"
  style="--wails-draggable: drag;"
  ondblclick={handleTitlebarDblClick}
  role="group"
>
  <div class="titlebar-left">
    <!-- macOS traffic lights space -->
    <div class="traffic-light-spacer"></div>
    <button
      type="button"
      class="app-branding"
      onclick={() => onOpenAbout?.()}
      title="About FuzzNexus v1.0"
      aria-label="About FuzzNexus"
      style="--wails-draggable: no-drag;"
    >
      <Logo size={22} />
      <span class="app-title">FuzzNexus</span>
      <span class="app-version">v1.0</span>
    </button>
    {#if scanStore.sessionID > 0}
      <div class="session-badge font-mono">
        Session #{scanStore.sessionID}
      </div>
    {/if}
  </div>

  <div class="titlebar-right" style="--wails-draggable: no-drag;">
    <!-- View Mode Switcher with Jelly Indicator -->
    <div class="view-picker">
      <div
        class="picker-pill"
        class:ready={isPillReady}
        style="transform: translateX({pillLeft}px); width: {pillWidth}px;"
      ></div>

      <button
        bind:this={btnGrid}
        class="picker-btn"
        class:active={activeView === 'grid'}
        onclick={() => onViewChange?.('grid')}
      >
        Table
      </button>
      <button
        bind:this={btnAnalytics}
        class="picker-btn"
        class:active={activeView === 'analytics'}
        onclick={() => onViewChange?.('analytics')}
      >
        Analytics
      </button>
      <button
        bind:this={btnWordlists}
        class="picker-btn"
        class:active={activeView === 'wordlists'}
        onclick={() => onViewChange?.('wordlists')}
      >
        Wordlists
      </button>
    </div>

    <!-- Session History Button -->
    <button
      type="button"
      class="mode-toggle-btn"
      onclick={() => (scanStore.isHistoryModalOpen = true)}
      title="View scan session history and load past results"
      aria-label="Open session history"
    >
      <Icon name="rotate-ccw" size={14} />
      <span>History</span>
    </button>

    <!-- Keyboard Shortcuts Button (Puzzle Icon) -->
    <button
      type="button"
      class="about-btn"
      onclick={() => (shortcutsStore.isModalOpen = true)}
      data-tooltip="Shortcuts"
      aria-label="Keyboard Shortcuts"
    >
      <Icon name="puzzle" size={14} />
    </button>

    <!-- Theme Palette Selector -->
    <ThemePicker />

    <!-- Dark/Light Mode Toggle -->
    <button
      class="mode-toggle-btn"
      onclick={() => themeStore.toggleMode()}
      title="Mode: {themeStore.mode} (click to toggle)"
      aria-label="Toggle theme mode"
    >
      {#if themeStore.mode === 'light'}
        <Icon name="sun" size={14} />
        <span>Light</span>
      {:else if themeStore.mode === 'dark'}
        <Icon name="moon" size={14} />
        <span>Dark</span>
      {:else}
        <Icon name="laptop" size={14} />
        <span>Auto</span>
      {/if}
    </button>

    <!-- About FuzzNexus Button -->
    <button
      type="button"
      class="about-btn"
      onclick={() => onOpenAbout?.()}
      title="About FuzzNexus (Developer: Parsa Azari)"
      aria-label="About FuzzNexus"
    >
      <Icon name="info" size={14} />
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
    font-size: var(--text-base);
    position: relative;
    z-index: 10005;
  }

  :global([data-platform="windows"]) .titlebar {
    padding-right: 138px; /* Safe space for Windows native min/max/close controls */
  }

  .titlebar-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .traffic-light-spacer {
    width: 72px; /* Safe inset for macOS native red/yellow/green window buttons */
    height: 100%;
  }

  :global([data-platform="windows"]) .traffic-light-spacer,
  :global([data-platform="linux"]) .traffic-light-spacer {
    display: none;
  }

  .app-branding {
    display: flex;
    align-items: center;
    gap: 7px;
    background: transparent;
    border: none;
    padding: 3px 6px;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: background 0.15s ease, transform 0.15s ease;
    color: inherit;
    text-align: left;
  }

  .app-branding:hover {
    background: rgba(255, 255, 255, 0.06);
    transform: translateY(-0.5px);
  }

  :global([data-mode="light"]) .app-branding:hover {
    background: rgba(0, 0, 0, 0.05);
  }

  .about-btn {
    height: 28px;
    width: 28px;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .about-btn:hover {
    background: var(--bg-surface);
    color: var(--text);
    border-color: var(--accent);
  }

  .app-title {
    font-weight: 700;
    font-size: var(--text-lg);
    color: var(--text);
    letter-spacing: -0.02em;
  }

  .app-version {
    font-size: var(--text-xs);
    font-weight: var(--weight-medium);
    color: var(--accent);
    background: rgba(0, 122, 255, 0.1);
    padding: 1px 5px;
    border-radius: var(--radius-xs);
  }

  .session-badge {
    font-size: var(--text-xs);
    color: var(--text-secondary);
    background-color: var(--surface);
    padding: 2px 8px;
    border-radius: var(--radius-xs);
    border: 1px solid var(--border);
  }

  .titlebar-right {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 28px;
  }

  .view-picker {
    position: relative;
    display: flex;
    align-items: center;
    height: 28px;
    box-sizing: border-box;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 2px;
    gap: 2px;
  }

  .picker-pill {
    position: absolute;
    top: 2px;
    left: 0;
    height: 22px;
    background-color: var(--accent);
    border-radius: var(--radius-xs);
    pointer-events: none;
    z-index: 1;
    opacity: 0;
    will-change: transform, width;
    /* Elastic / Spring jelly cubic-bezier curves for playful physical bounce */
    transition:
      transform 0.38s cubic-bezier(0.34, 1.56, 0.64, 1),
      width 0.3s cubic-bezier(0.25, 1, 0.5, 1),
      opacity 0.15s ease;
  }

  .picker-pill.ready {
    opacity: 1;
  }

  .picker-btn {
    position: relative;
    z-index: 2;
    height: 22px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    padding: 0 10px;
    border: none;
    background: transparent;
    border-radius: var(--radius-xs);
    cursor: pointer;
    color: var(--text-secondary);
    transition: color 0.2s ease, font-weight 0.15s ease;
    line-height: 1;
    user-select: none;
  }

  .picker-btn:hover {
    color: var(--text);
  }

  .picker-btn.active {
    color: var(--accent-contrast);
    font-weight: var(--weight-semibold);
  }

  .mode-toggle-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    height: 28px;
    box-sizing: border-box;
    padding: 0 10px;
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    background-color: var(--surface);
    cursor: pointer;
    color: var(--text);
    transition: background-color 0.12s ease, border-color 0.12s ease;
    line-height: 1;
    user-select: none;
  }

  .mode-toggle-btn:hover {
    background-color: var(--surface-hover);
  }

  .mode-toggle-btn :global(.inline-icon) {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .mode-toggle-btn span {
    display: inline-flex;
    align-items: center;
    line-height: 1;
  }
</style>
