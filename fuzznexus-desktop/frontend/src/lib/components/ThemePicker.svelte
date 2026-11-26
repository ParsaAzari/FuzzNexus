<script lang="ts">
  import { themeStore, THEMES, type ThemePalette } from '../stores/theme.svelte';
  import Icon from './Icon.svelte';

  let isOpen = $state(false);
  let pickerRef = $state<HTMLDivElement | null>(null);

  function toggleDropdown(e: MouseEvent) {
    e.stopPropagation();
    isOpen = !isOpen;
  }

  function selectTheme(palette: ThemePalette) {
    themeStore.setPalette(palette);
    isOpen = false;
  }

  function handleWindowPointerDown(e: PointerEvent) {
    if (!isOpen || !pickerRef) return;
    const target = e.target as Node | null;
    if (target && !pickerRef.contains(target)) {
      isOpen = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && isOpen) {
      isOpen = false;
      e.stopPropagation();
    }
  }

  const currentTheme = $derived(themeStore.currentTheme);
</script>

<svelte:window onpointerdown={handleWindowPointerDown} onkeydown={handleKeydown} />

<div class="theme-picker-wrapper" bind:this={pickerRef}>
  <button
    type="button"
    class="theme-picker-btn"
    class:active={isOpen}
    onclick={toggleDropdown}
    title="Theme: {currentTheme.name} (Click to change)"
    aria-haspopup="true"
    aria-expanded={isOpen}
    aria-label="Select color theme"
  >
    <Icon name="palette" size={14} color="var(--accent)" />
    <span class="theme-label">{currentTheme.name}</span>
    <span class="chevron" class:rotated={isOpen}>
      <Icon name="chevron-down" size={10} />
    </span>
  </button>

  {#if isOpen}
    <div class="theme-dropdown-menu" role="menu" aria-label="Color themes">
      <div class="menu-section-header">Apple HIG</div>
      {#each THEMES.filter((t) => t.category === 'core') as t}
        <button
          type="button"
          class="menu-item"
          class:selected={themeStore.palette === t.id}
          onclick={() => selectTheme(t.id)}
          role="menuitem"
        >
          <div class="item-left">
            <span
              class="swatch-pill"
              style="background-color: {t.preview.bg}; border-color: {t.preview.surface};"
            >
              <span class="swatch-dot" style="background-color: {t.preview.accent};"></span>
            </span>
            <span class="item-name">{t.name}</span>
          </div>
          {#if themeStore.palette === t.id}
            <span class="item-check">
              <Icon name="check" size={12} />
            </span>
          {/if}
        </button>
      {/each}

      <div class="menu-divider"></div>
      <div class="menu-section-header">Dark Favorites</div>
      {#each THEMES.filter((t) => t.category === 'community') as t}
        <button
          type="button"
          class="menu-item"
          class:selected={themeStore.palette === t.id}
          onclick={() => selectTheme(t.id)}
          role="menuitem"
        >
          <div class="item-left">
            <span
              class="swatch-pill"
              style="background-color: {t.preview.bg}; border-color: {t.preview.surface};"
            >
              <span class="swatch-dot" style="background-color: {t.preview.accent};"></span>
            </span>
            <span class="item-name">{t.name}</span>
          </div>
          {#if themeStore.palette === t.id}
            <span class="item-check">
              <Icon name="check" size={12} />
            </span>
          {/if}
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .theme-picker-wrapper {
    position: relative;
    display: inline-flex;
    align-items: center;
  }

  .theme-picker-btn {
    all: unset;
    height: 28px;
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    gap: 7px;
    padding: 0 9px;
    font-size: var(--text-sm);
    font-weight: var(--weight-medium);
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
    background-color: var(--surface);
    cursor: pointer;
    color: var(--text);
    transition: background-color 0.12s ease, border-color 0.12s ease, box-shadow 0.12s ease;
    user-select: none;
    -webkit-user-select: none;
  }

  .theme-picker-btn:hover {
    background-color: var(--surface-hover);
    border-color: var(--border-hover, var(--border));
  }

  .theme-picker-btn.active {
    background-color: var(--surface-hover);
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent);
  }

  .theme-label {
    line-height: 1;
    white-space: nowrap;
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .chevron {
    display: inline-flex;
    align-items: center;
    color: var(--text-secondary);
    transition: transform 0.15s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .chevron.rotated {
    transform: rotate(180deg);
  }

  .theme-dropdown-menu {
    position: absolute;
    top: calc(100% + 5px);
    right: 0;
    min-width: 195px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    box-shadow: 0 10px 28px rgba(0, 0, 0, 0.28), 0 2px 8px rgba(0, 0, 0, 0.12);
    padding: 5px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    z-index: 120;
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    animation: menu-slide 0.12s cubic-bezier(0.16, 1, 0.3, 1);
    box-sizing: border-box;
  }

  @keyframes menu-slide {
    from {
      opacity: 0;
      transform: translateY(-4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .menu-section-header {
    font-size: 10px;
    font-weight: var(--weight-bold);
    text-transform: uppercase;
    letter-spacing: 0.6px;
    color: var(--text-secondary);
    padding: 5px 8px 3px 8px;
    user-select: none;
    -webkit-user-select: none;
  }

  .menu-divider {
    height: 1px;
    background-color: var(--border);
    margin: 4px 4px;
    opacity: 0.6;
  }

  .menu-item {
    all: unset;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 9px;
    border-radius: var(--radius-xs);
    cursor: pointer;
    box-sizing: border-box;
    transition: background-color 0.1s ease;
    user-select: none;
    -webkit-user-select: none;
  }

  .menu-item:hover {
    background-color: var(--surface-hover);
  }

  .menu-item.selected {
    background-color: var(--bg-subtle);
  }

  .menu-item.selected .item-name {
    font-weight: var(--weight-semibold);
    color: var(--text);
  }

  .item-left {
    display: flex;
    align-items: center;
    gap: 9px;
  }

  .swatch-pill {
    width: 18px;
    height: 18px;
    border-radius: 4px;
    border: 1px solid var(--border);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    flex-shrink: 0;
  }

  .swatch-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    box-sizing: border-box;
  }

  .item-name {
    font-size: var(--text-sm);
    color: var(--text);
    white-space: nowrap;
    line-height: 1.2;
  }

  .item-check {
    color: var(--accent);
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }
</style>
