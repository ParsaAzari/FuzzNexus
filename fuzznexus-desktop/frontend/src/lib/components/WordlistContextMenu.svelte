<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from './Icon.svelte';
  import type { WordlistDTO } from '../types';

  interface Props {
    x: number;
    y: number;
    wordlist: WordlistDTO;
    isPrimaryTarget: boolean;
    isReadOnly: boolean;
    categories?: { id: string; label: string; color: string }[];
    onSetPrimary: (w: WordlistDTO) => void;
    onRename: (w: WordlistDTO) => void;
    onChangeCategory: (w: WordlistDTO, cat: string) => void;
    onToggleReadOnly: () => void;
    onExport: (w: WordlistDTO) => void;
    onDuplicate: (w: WordlistDTO) => void;
    onResetBuiltin?: (w: WordlistDTO) => void;
    onDelete: (w: WordlistDTO) => void;
    onClose: () => void;
  }

  let {
    x,
    y,
    wordlist,
    isPrimaryTarget,
    isReadOnly,
    categories: propCategories,
    onSetPrimary,
    onRename,
    onChangeCategory,
    onToggleReadOnly,
    onExport,
    onDuplicate,
    onResetBuiltin,
    onDelete,
    onClose
  }: Props = $props();

  let menuRef = $state<HTMLDivElement | null>(null);
  let adjustedX = $state<number>(0);
  let adjustedY = $state<number>(0);
  let categorySubmenuOpen = $state<boolean>(false);

  const defaultCategories = [
    { id: 'admin', label: 'Admin & Portals', color: '#f43f5e' },
    { id: 'api', label: 'Modern APIs', color: '#06b6d4' },
    { id: 'ai', label: 'AI & LLMs', color: '#a855f7' },
    { id: 'frameworks', label: 'Frameworks', color: '#3b82f6' },
    { id: 'cms', label: 'CMS Platforms', color: '#8b5cf6' },
    { id: 'cloud', label: 'Cloud & DevOps', color: '#10b981' },
    { id: 'auth', label: 'Auth & SSO', color: '#ec4899' },
    { id: 'leaks', label: 'Leaks & Dumps', color: '#f59e0b' },
    { id: 'bypasses', label: '403 Bypasses', color: '#6366f1' },
    { id: 'general', label: 'General', color: '#64748b' }
  ];

  let categories = $derived(
    propCategories && propCategories.length > 0
      ? propCategories.filter((c) => c.id !== 'all')
      : defaultCategories
  );

  $effect(() => {
    adjustedX = x;
    adjustedY = y;
    if (menuRef) {
      const rect = menuRef.getBoundingClientRect();
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      if (x + rect.width > viewportWidth - 8) {
        adjustedX = Math.max(8, viewportWidth - rect.width - 8);
      }

      if (y + rect.height > viewportHeight - 8) {
        adjustedY = Math.max(8, viewportHeight - rect.height - 8);
      }
    }
  });

  onMount(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        onClose();
      }
    }

    function handleWindowClick(e: MouseEvent) {
      if (menuRef && !menuRef.contains(e.target as Node)) {
        onClose();
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('mousedown', handleWindowClick);

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('mousedown', handleWindowClick);
    };
  });
</script>

<div
  bind:this={menuRef}
  class="context-menu"
  style="top: {adjustedY}px; left: {adjustedX}px;"
  role="menu"
  tabindex="-1"
>
  <div class="menu-header">
    <span class="menu-filename" title={wordlist.name}>{wordlist.name}</span>
    <span class="menu-lines">{wordlist.line_count.toLocaleString()} lines</span>
  </div>

  <div class="menu-divider"></div>

  <button
    type="button"
    class="menu-item"
    onclick={() => { onSetPrimary(wordlist); onClose(); }}
  >
    <Icon name="target" size={14} class="menu-icon" />
    <span class="menu-label">{isPrimaryTarget ? 'Active Primary Target' : 'Set as Primary Scan Target'}</span>
    {#if isPrimaryTarget}
      <span class="menu-badge-active">Active</span>
    {/if}
  </button>

  <button
    type="button"
    class="menu-item"
    onclick={() => { onToggleReadOnly(); onClose(); }}
  >
    <Icon name={isReadOnly ? 'unlock' : 'lock'} size={14} class="menu-icon" />
    <span class="menu-label">{isReadOnly ? 'Unlock to Edit' : 'Lock as Read-Only'}</span>
    <span class="menu-hint">{isReadOnly ? 'View Mode' : 'Editable'}</span>
  </button>

  <div class="menu-divider"></div>

  <button
    type="button"
    class="menu-item"
    onclick={() => { onRename(wordlist); onClose(); }}
  >
    <Icon name="edit" size={14} class="menu-icon" />
    <span class="menu-label">Rename Wordlist</span>
  </button>

  <div
    class="menu-item has-submenu"
    onmouseenter={() => { categorySubmenuOpen = true; }}
    onmouseleave={() => { categorySubmenuOpen = false; }}
    role="none"
  >
    <Icon name="tag" size={14} class="menu-icon" />
    <span class="menu-label">Category</span>
    <span class="menu-current-cat">{(wordlist.category || 'general').toUpperCase()}</span>
    <Icon name="chevron-right" size={12} class="submenu-arrow" />

    {#if categorySubmenuOpen}
      <div class="submenu" role="menu">
        {#each categories as cat (cat.id)}
          <button
            type="button"
            class="submenu-item"
            class:active={(wordlist.category || 'general').toLowerCase() === cat.id}
            onclick={() => { onChangeCategory(wordlist, cat.id); onClose(); }}
          >
            <span class="cat-dot" style="background-color: {cat.color};"></span>
            <span class="submenu-label">{cat.label}</span>
            {#if (wordlist.category || 'general').toLowerCase() === cat.id}
              <Icon name="check" size={12} class="check-icon" />
            {/if}
          </button>
        {/each}
      </div>
    {/if}
  </div>

  <div class="menu-divider"></div>

  <button
    type="button"
    class="menu-item"
    onclick={() => { onDuplicate(wordlist); onClose(); }}
  >
    <Icon name="copy" size={14} class="menu-icon" />
    <span class="menu-label">Duplicate Wordlist</span>
  </button>

  <button
    type="button"
    class="menu-item"
    onclick={() => { onExport(wordlist); onClose(); }}
  >
    <Icon name="download" size={14} class="menu-icon" />
    <span class="menu-label">Export to File...</span>
  </button>

  {#if wordlist.is_builtin && onResetBuiltin}
    <button
      type="button"
      class="menu-item"
      onclick={() => { onResetBuiltin(wordlist); onClose(); }}
    >
      <Icon name="rotate-ccw" size={14} class="menu-icon" />
      <span class="menu-label">Reset to Factory Default</span>
    </button>
  {/if}

  <div class="menu-divider"></div>

  <button
    type="button"
    class="menu-item danger"
    onclick={() => { onDelete(wordlist); onClose(); }}
  >
    <Icon name="trash" size={14} class="menu-icon" />
    <span class="menu-label">Delete Wordlist</span>
  </button>
</div>

<style>
  .context-menu {
    position: fixed;
    z-index: 9999;
    min-width: 220px;
    background: var(--bg-surface, #1e2024);
    border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
    border-radius: var(--radius-md, 8px);
    padding: 4px;
    box-shadow: 0 10px 28px rgba(0, 0, 0, 0.45), 0 0 0 1px rgba(255, 255, 255, 0.05);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    font-family: inherit;
    user-select: none;
    -webkit-user-select: none;
    animation: menuFadeIn 0.12s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes menuFadeIn {
    from {
      opacity: 0;
      transform: scale(0.97);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }

  .menu-header {
    display: flex;
    flex-direction: column;
    padding: 6px 10px 4px;
  }

  .menu-filename {
    font-size: var(--text-xs, 12px);
    font-weight: 600;
    color: var(--text-primary, #ffffff);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 190px;
  }

  .menu-lines {
    font-size: 10px;
    color: var(--text-muted, #8e8e93);
    font-family: var(--font-mono, monospace);
  }

  .menu-divider {
    height: 1px;
    background: var(--border-color, rgba(255, 255, 255, 0.08));
    margin: 4px 0;
  }

  .menu-item {
    display: flex;
    align-items: center;
    width: 100%;
    padding: 6px 10px;
    gap: 8px;
    background: transparent;
    border: none;
    border-radius: var(--radius-sm, 6px);
    color: var(--text-primary, #e2e8f0);
    font-size: var(--text-xs, 12px);
    text-align: left;
    cursor: pointer;
    transition: background 0.1s ease, color 0.1s ease;
    position: relative;
  }

  .menu-item:hover,
  .menu-item:focus-visible {
    background: var(--accent, #3b82f6);
    color: #ffffff;
    outline: none;
  }

  .menu-item.danger {
    color: var(--error, #ef4444);
  }

  .menu-item.danger:hover {
    background: var(--error, #ef4444);
    color: #ffffff;
  }

  .menu-label {
    flex: 1;
    font-weight: 450;
  }

  .menu-hint {
    font-size: 10px;
    opacity: 0.7;
    margin-left: auto;
  }

  .menu-badge-active {
    font-size: 9px;
    font-weight: 600;
    text-transform: uppercase;
    background: rgba(16, 185, 129, 0.2);
    color: #10b981;
    border-radius: 4px;
    padding: 1px 4px;
    margin-left: auto;
  }

  .menu-item:hover .menu-badge-active {
    background: rgba(255, 255, 255, 0.25);
    color: #ffffff;
  }

  .menu-current-cat {
    font-size: 9px;
    font-weight: 600;
    letter-spacing: 0.5px;
    opacity: 0.75;
    background: var(--bg-card, rgba(255, 255, 255, 0.06));
    padding: 1px 5px;
    border-radius: 4px;
    margin-left: auto;
  }

  .has-submenu {
    cursor: default;
  }

  :global(.submenu-arrow) {
    opacity: 0.6;
    margin-left: 4px;
  }

  .submenu {
    position: absolute;
    left: calc(100% - 2px);
    top: -4px;
    min-width: 170px;
    background: var(--bg-surface, #1e2024);
    border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
    border-radius: var(--radius-md, 8px);
    padding: 4px;
    box-shadow: 0 10px 24px rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    animation: menuFadeIn 0.12s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .submenu-item {
    display: flex;
    align-items: center;
    width: 100%;
    padding: 5px 8px;
    gap: 8px;
    background: transparent;
    border: none;
    border-radius: var(--radius-sm, 6px);
    color: var(--text-primary, #e2e8f0);
    font-size: var(--text-xs, 12px);
    text-align: left;
    cursor: pointer;
    transition: background 0.1s ease, color 0.1s ease;
  }

  .submenu-item:hover {
    background: var(--accent, #3b82f6);
    color: #ffffff;
  }

  .submenu-item.active {
    font-weight: 600;
  }

  .cat-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .submenu-label {
    flex: 1;
  }

  :global(.check-icon) {
    margin-left: auto;
  }
</style>
