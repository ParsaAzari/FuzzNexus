<script lang="ts">
  import { onMount } from 'svelte';
  import { wordlistsStore } from '../stores/wordlists.svelte';
  import { scanStore } from '../stores/scan.svelte';
  import { toastStore } from '../stores/toast.svelte';
  import Icon from './Icon.svelte';
  import type { WordlistDTO } from '../types';

  interface Props {
    onSelectForScan?: (path: string) => void;
  }

  let { onSelectForScan }: Props = $props();

  // Modals state
  let newWordlistModalOpen = $state<boolean>(false);
  let newWordlistName = $state<string>('');
  let initialLines = $state<string>('');

  let prefixSuffixModalOpen = $state<boolean>(false);
  let prefixInput = $state<string>('');
  let suffixInput = $state<string>('');

  let confirmDeleteModalOpen = $state<boolean>(false);
  let deleteTargets = $state<WordlistDTO[]>([]);

  let unsavedDiscardModalOpen = $state<boolean>(false);
  let pendingPageNavOffset = $state<number | null>(null);

  let shortcutsModalOpen = $state<boolean>(false);

  // Editor & navigation state
  let editorLines = $state<string[]>([]);
  let editorMode = $state<'lines' | 'raw'>('lines');
  let rawTextContent = $state<string>('');
  let rawTextareaRef = $state<HTMLTextAreaElement | null>(null);
  let lineInputRefs = $state<Record<number, HTMLInputElement>>({});
  let sortDropdownOpen = $state<boolean>(false);
  let transformDropdownOpen = $state<boolean>(false);
  let moreDropdownOpen = $state<boolean>(false);
  let sidebarCollapsed = $state<boolean>(false);
  let selectedLineIndices = $state<Set<number>>(new Set());
  let isDirty = $state<boolean>(false);
  let pageOffset = $state<number>(0);
  const pageSize = 100;

  let focusedIndex = $state<number>(0);
  let searchInputRef = $state<HTMLInputElement | null>(null);
  let masterCheckboxRef = $state<HTMLInputElement | null>(null);
  let itemRefs = $state<Record<number, HTMLElement>>({});

  let selected = $derived(wordlistsStore.selectedWordlist);
  let stats = $derived(wordlistsStore.selectedStats);
  let health = $derived(wordlistsStore.selectedHealth);
  let totalLines = $derived(wordlistsStore.currentPage.total_lines || 0);

  onMount(() => {
    wordlistsStore.loadWordlists();
  });

  $effect(() => {
    if (wordlistsStore.currentPage.lines) {
      editorLines = [...wordlistsStore.currentPage.lines];
      rawTextContent = editorLines.join('\n');
      selectedLineIndices = new Set();
      isDirty = false;
    }
  });

  // Sync indeterminate property on master checkbox
  $effect(() => {
    if (masterCheckboxRef) {
      masterCheckboxRef.indeterminate = wordlistsStore.isSomeChecked;
    }
  });

  function handleClickOutside(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (!target.closest('.dropdown-wrapper')) {
      sortDropdownOpen = false;
      transformDropdownOpen = false;
      moreDropdownOpen = false;
    }
  }

  function handleLineChange(index: number, value: string) {
    editorLines[index] = value;
    rawTextContent = editorLines.join('\n');
    isDirty = true;
  }

  function handleRawTextChange(val: string) {
    rawTextContent = val;
    editorLines = val.split('\n');
    isDirty = true;
  }

  function setEditorMode(mode: 'lines' | 'raw') {
    if (editorMode === mode) return;
    if (mode === 'raw') {
      rawTextContent = editorLines.join('\n');
      editorMode = 'raw';
      toastStore.info('Raw View: Cmd+A selects all lines');
    } else {
      editorLines = rawTextContent.split('\n');
      editorMode = 'lines';
      selectedLineIndices = new Set();
      toastStore.info('Lines View');
    }
  }

  function toggleEditorMode() {
    setEditorMode(editorMode === 'lines' ? 'raw' : 'lines');
  }

  function handleSelectAll() {
    if (editorMode === 'raw') {
      if (rawTextareaRef) {
        rawTextareaRef.focus();
        rawTextareaRef.select();
      }
      toastStore.info(`Selected all ${editorLines.length} lines`);
    } else {
      // Line-by-line mode
      if (selectedLineIndices.size === editorLines.length) {
        selectedLineIndices = new Set();
      } else {
        selectedLineIndices = new Set(editorLines.map((_, i) => i));
        toastStore.info(`Selected all ${editorLines.length} lines (Cmd+C to copy, Delete to remove)`);
      }
    }
  }

  function toggleLineSelect(idx: number, e?: MouseEvent) {
    if (e && e.shiftKey && selectedLineIndices.size > 0) {
      const arr = Array.from(selectedLineIndices);
      const last = arr[arr.length - 1];
      const start = Math.min(last, idx);
      const end = Math.max(last, idx);
      const newSet = new Set(selectedLineIndices);
      for (let i = start; i <= end; i++) {
        newSet.add(i);
      }
      selectedLineIndices = newSet;
      return;
    }

    const newSet = new Set(selectedLineIndices);
    if (newSet.has(idx)) {
      newSet.delete(idx);
    } else {
      newSet.add(idx);
    }
    selectedLineIndices = newSet;
  }

  function deleteSelectedLines() {
    if (selectedLineIndices.size === 0) return;
    const count = selectedLineIndices.size;
    editorLines = editorLines.filter((_, i) => !selectedLineIndices.has(i));
    if (editorLines.length === 0) {
      editorLines = [''];
    }
    rawTextContent = editorLines.join('\n');
    selectedLineIndices = new Set();
    isDirty = true;
    toastStore.success(`Deleted ${count} lines`);
  }

  async function copyAllLines() {
    let textToCopy = '';
    if (editorMode === 'raw') {
      textToCopy = rawTextContent;
    } else if (selectedLineIndices.size > 0) {
      textToCopy = editorLines.filter((_, i) => selectedLineIndices.has(i)).join('\n');
    } else {
      textToCopy = editorLines.join('\n');
    }

    try {
      await navigator.clipboard.writeText(textToCopy);
      const count = textToCopy.split('\n').length;
      toastStore.success(`Copied ${count} lines to clipboard`);
    } catch {
      toastStore.error('Failed to copy to clipboard');
    }
  }

  function handleLineInputKeyDown(e: KeyboardEvent, idx: number) {
    const isMeta = e.metaKey || e.ctrlKey;

    if (e.key === 'Enter') {
      e.preventDefault();
      editorLines.splice(idx + 1, 0, '');
      rawTextContent = editorLines.join('\n');
      isDirty = true;
      setTimeout(() => {
        lineInputRefs[idx + 1]?.focus();
      }, 0);
      return;
    }

    if (e.key === 'Backspace' && editorLines[idx] === '' && editorLines.length > 1) {
      e.preventDefault();
      editorLines.splice(idx, 1);
      rawTextContent = editorLines.join('\n');
      isDirty = true;
      setTimeout(() => {
        lineInputRefs[Math.max(0, idx - 1)]?.focus();
      }, 0);
      return;
    }

    // Delete or Backspace when multiple lines are selected in Lines mode
    if ((e.key === 'Delete' || e.key === 'Backspace') && selectedLineIndices.size > 0) {
      e.preventDefault();
      deleteSelectedLines();
      return;
    }

    if (e.key === 'ArrowDown') {
      if (idx < editorLines.length - 1) {
        e.preventDefault();
        lineInputRefs[idx + 1]?.focus();
      }
      return;
    }

    if (e.key === 'ArrowUp') {
      if (idx > 0) {
        e.preventDefault();
        lineInputRefs[idx - 1]?.focus();
      }
      return;
    }

    // Cmd+A inside line input: If already fully selected, promote to Select All Lines!
    if (isMeta && e.key.toLowerCase() === 'a') {
      const input = e.currentTarget as HTMLInputElement;
      if (input.selectionStart === 0 && input.selectionEnd === input.value.length) {
        e.preventDefault();
        handleSelectAll();
        return;
      }
    }

    // Shift+Cmd+A: Select all lines directly
    if (isMeta && e.shiftKey && e.key.toLowerCase() === 'a') {
      e.preventDefault();
      handleSelectAll();
      return;
    }
  }

  function handleLinePaste(e: ClipboardEvent, idx: number) {
    const text = e.clipboardData?.getData('text');
    if (!text || (!text.includes('\n') && !text.includes('\r'))) {
      return; // Single-line paste handled normally
    }
    e.preventDefault();
    const pastedLines = text.split(/\r?\n/).map((l) => l.trim()).filter((l) => l.length > 0);
    if (!pastedLines.length) return;

    editorLines.splice(idx, 1, ...pastedLines);
    rawTextContent = editorLines.join('\n');
    isDirty = true;
    toastStore.success(`Pasted ${pastedLines.length} lines`);
  }

  async function handleSave() {
    if (!selected) return;
    try {
      await wordlistsStore.saveContent(selected.id, editorLines.join('\n'));
      isDirty = false;
      toastStore.success(`Saved changes to "${selected.name}"`);
    } catch (e: any) {
      toastStore.error(`Save failed: ${e?.message || e}`);
    }
  }

  async function handleCreateNew() {
    const name = newWordlistName.trim();
    if (!name) return;
    try {
      await wordlistsStore.createWordlist(name, initialLines);
      toastStore.success(`Created wordlist "${name}"`);
      newWordlistName = '';
      initialLines = '';
      newWordlistModalOpen = false;
    } catch (e: any) {
      toastStore.error(`Create failed: ${e?.message || e}`);
    }
  }

  // Wordlist transformation tools
  function applyDeduplicate() {
    const seen = new Set<string>();
    editorLines = editorLines.filter((l) => {
      if (seen.has(l)) return false;
      seen.add(l);
      return true;
    });
    isDirty = true;
    toastStore.info('Deduplicated current page');
  }

  function applySort(asc: boolean) {
    editorLines = [...editorLines].sort((a, b) => (asc ? a.localeCompare(b) : b.localeCompare(a)));
    isDirty = true;
    toastStore.info(asc ? 'Sorted current page A-Z' : 'Sorted current page Z-A');
  }

  function applyTrim() {
    editorLines = editorLines.map((l) => l.trim());
    isDirty = true;
    toastStore.info('Trimmed whitespace on current page');
  }

  function applyCase(lower: boolean) {
    editorLines = editorLines.map((l) => (lower ? l.toLowerCase() : l.toUpperCase()));
    isDirty = true;
  }

  function applyStripEmpty() {
    editorLines = editorLines.filter((l) => l.trim().length > 0);
    isDirty = true;
    toastStore.info('Stripped empty lines from current page');
  }

  async function handleCleanWordlist() {
    if (!selected) return;
    try {
      const res = await wordlistsStore.cleanWordlist(selected.id);
      if (res.empty_lines_removed > 0 || res.lines_trimmed > 0) {
        toastStore.success(
          `Cleaned "${selected.name}": removed ${res.empty_lines_removed.toLocaleString()} blank lines, trimmed ${res.lines_trimmed.toLocaleString()} lines`
        );
      } else {
        toastStore.info(`Wordlist "${selected.name}" is verified clean (0 blank lines)`);
      }
    } catch (e: any) {
      toastStore.error(`Clean failed: ${e?.message || e}`);
    }
  }

  function applyPrefixSuffix() {
    const p = prefixInput;
    const s = suffixInput;
    editorLines = editorLines.map((l) => `${p}${l}${s}`);
    isDirty = true;
    prefixSuffixModalOpen = false;
    prefixInput = '';
    suffixInput = '';
    toastStore.info('Applied prefix/suffix to current page');
  }

  function handlePageNav(newOffset: number) {
    if (isDirty) {
      pendingPageNavOffset = newOffset;
      unsavedDiscardModalOpen = true;
      return;
    }
    navigatePage(newOffset);
  }

  function navigatePage(newOffset: number) {
    pageOffset = Math.max(0, Math.min(newOffset, Math.max(0, totalLines - pageSize)));
    wordlistsStore.loadPage(pageOffset, pageSize);
  }

  function confirmDiscardNav() {
    if (pendingPageNavOffset !== null) {
      navigatePage(pendingPageNavOffset);
    }
    isDirty = false;
    unsavedDiscardModalOpen = false;
    pendingPageNavOffset = null;
  }

  function handleSelectForScan(w: WordlistDTO) {
    scanStore.wordlist = w.path;
    toastStore.info(`Selected "${w.name}" for scan`);
    onSelectForScan?.(w.path);
  }

  // Delete handling with modal
  function openDeleteModal(targets: WordlistDTO[]) {
    if (!targets || targets.length === 0) return;
    deleteTargets = targets;
    confirmDeleteModalOpen = true;
  }

  function openBatchDeleteModal() {
    const targets = wordlistsStore.filteredWordlists.filter((w) =>
      wordlistsStore.checkedIDs.includes(w.id)
    );
    if (targets.length > 0) {
      openDeleteModal(targets);
    } else if (selected) {
      openDeleteModal([selected]);
    }
  }

  async function confirmDelete() {
    if (!deleteTargets.length) return;
    const count = deleteTargets.length;
    const names = count === 1 ? `"${deleteTargets[0].name}"` : `${count} wordlists`;
    try {
      if (count === 1) {
        await wordlistsStore.deleteWordlist(deleteTargets[0].id);
      } else {
        await wordlistsStore.deleteBatch(deleteTargets.map((w) => w.id));
      }
      toastStore.success(`Successfully deleted ${names}`);
      confirmDeleteModalOpen = false;
      deleteTargets = [];
    } catch (e: any) {
      toastStore.error(`Delete failed: ${e?.message || e}`);
    }
  }

  // Keyboard navigation helpers
  function moveFocus(delta: number) {
    const list = wordlistsStore.filteredWordlists;
    if (!list.length) return;
    focusedIndex = Math.max(0, Math.min(list.length - 1, focusedIndex + delta));
    const target = list[focusedIndex];
    if (target && itemRefs[target.id]) {
      itemRefs[target.id].scrollIntoView({ block: 'nearest' });
    }
  }

  function toggleFocusedCheck() {
    const list = wordlistsStore.filteredWordlists;
    if (!list.length || focusedIndex < 0 || focusedIndex >= list.length) return;
    const item = list[focusedIndex];
    wordlistsStore.toggleCheck(item.id);
  }

  function selectFocusedWordlist() {
    const list = wordlistsStore.filteredWordlists;
    if (!list.length || focusedIndex < 0 || focusedIndex >= list.length) return;
    const item = list[focusedIndex];
    wordlistsStore.selectWordlist(item.id);
  }

  function handleKeyDown(e: KeyboardEvent) {
    const activeEl = document.activeElement;
    const activeTag = (activeEl?.tagName || '').toLowerCase();
    const isEditingText =
      activeTag === 'input' ||
      activeTag === 'textarea' ||
      (activeEl as HTMLElement)?.isContentEditable;

    const isMeta = e.metaKey || e.ctrlKey;

    // Handle open modals
    if (confirmDeleteModalOpen) {
      if (e.key === 'Escape') {
        e.preventDefault();
        confirmDeleteModalOpen = false;
        return;
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        confirmDelete();
        return;
      }
      return;
    }

    if (unsavedDiscardModalOpen) {
      if (e.key === 'Escape') {
        e.preventDefault();
        unsavedDiscardModalOpen = false;
        return;
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        confirmDiscardNav();
        return;
      }
      return;
    }

    if (shortcutsModalOpen) {
      if (e.key === 'Escape' || e.key === 'Enter') {
        e.preventDefault();
        shortcutsModalOpen = false;
        return;
      }
      return;
    }

    if (newWordlistModalOpen) {
      if (e.key === 'Escape') {
        e.preventDefault();
        newWordlistModalOpen = false;
        return;
      }
      if (isMeta && e.key === 'Enter') {
        e.preventDefault();
        handleCreateNew();
        return;
      }
      return;
    }

    if (prefixSuffixModalOpen) {
      if (e.key === 'Escape') {
        e.preventDefault();
        prefixSuffixModalOpen = false;
        return;
      }
      if (isMeta && e.key === 'Enter') {
        e.preventDefault();
        applyPrefixSuffix();
        return;
      }
      return;
    }

    // Global Wordlist Shortcuts
    if (isMeta && e.key.toLowerCase() === 's') {
      e.preventDefault();
      handleSave();
      return;
    }

    if (isMeta && e.key.toLowerCase() === 'n') {
      e.preventDefault();
      newWordlistModalOpen = true;
      return;
    }

    if (isMeta && e.key.toLowerCase() === 'o') {
      e.preventDefault();
      wordlistsStore.openNativeFilePicker('import');
      return;
    }

    if (isMeta && e.key.toLowerCase() === 'l') {
      e.preventDefault();
      wordlistsStore.openNativeFilePicker('link');
      return;
    }

    if (isMeta && e.key.toLowerCase() === 'e') {
      e.preventDefault();
      toggleEditorMode();
      return;
    }

    if (isMeta && e.shiftKey && e.key.toLowerCase() === 'd') {
      e.preventDefault();
      applyDeduplicate();
      return;
    }

    if (isMeta && e.shiftKey && e.key.toLowerCase() === 's') {
      e.preventDefault();
      applySort(true);
      return;
    }

    if (isMeta && e.shiftKey && e.key.toLowerCase() === 't') {
      e.preventDefault();
      applyTrim();
      return;
    }

    if (isMeta && e.shiftKey && e.key.toLowerCase() === 'c') {
      e.preventDefault();
      handleCleanWordlist();
      return;
    }

    if (e.altKey && e.key === 'ArrowLeft') {
      e.preventDefault();
      handlePageNav(pageOffset - pageSize);
      return;
    }

    if (e.key === 'Escape') {
      if (sortDropdownOpen || transformDropdownOpen || moreDropdownOpen) {
        sortDropdownOpen = false;
        transformDropdownOpen = false;
        moreDropdownOpen = false;
        return;
      }
    }

    if (isMeta && (e.key === '\\' || e.key.toLowerCase() === 'b')) {
      e.preventDefault();
      sidebarCollapsed = !sidebarCollapsed;
      return;
    }

    if (e.altKey && e.key === 'ArrowRight') {
      e.preventDefault();
      handlePageNav(pageOffset + pageSize);
      return;
    }

    // Shortcuts active when NOT typing in an input
    if (!isEditingText) {
      if (e.key === '?' || (isMeta && e.key === '/')) {
        e.preventDefault();
        shortcutsModalOpen = true;
        return;
      }

      if (e.key === '/' || (isMeta && e.key.toLowerCase() === 'f')) {
        e.preventDefault();
        searchInputRef?.focus();
        searchInputRef?.select();
        return;
      }

      if (isMeta && e.key.toLowerCase() === 'a') {
        e.preventDefault();
        wordlistsStore.toggleCheckAll();
        return;
      }

      if (e.key === 'Delete' || e.key === 'Backspace') {
        e.preventDefault();
        openBatchDeleteModal();
        return;
      }

      if (e.key === 'ArrowDown' || e.key === 'j') {
        e.preventDefault();
        moveFocus(1);
        return;
      }

      if (e.key === 'ArrowUp' || e.key === 'k') {
        e.preventDefault();
        moveFocus(-1);
        return;
      }

      if (e.key === ' ') {
        e.preventDefault();
        toggleFocusedCheck();
        return;
      }

      if (e.key === 'Enter') {
        e.preventDefault();
        selectFocusedWordlist();
        return;
      }
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} onclick={handleClickOutside} />

<div class="wordlist-manager select-none" class:sidebar-is-collapsed={sidebarCollapsed}>
  <!-- Left Column: Wordlist Library -->
  {#if !sidebarCollapsed}
    <aside class="sidebar-library">
      <div class="sidebar-header">
        <div class="sidebar-title-bar">
          <span class="sidebar-heading">Wordlists</span>
          <button
            type="button"
            class="btn-icon-toggle"
            onclick={() => (sidebarCollapsed = true)}
            title="Collapse Library (Cmd+\)"
            aria-label="Collapse Library"
          >
            <Icon name="panel-left-close" size={14} />
          </button>
        </div>
        <div class="search-wrap">
        <Icon name="search" size={13} class="search-icon" />
        <input
          bind:this={searchInputRef}
          type="text"
          placeholder="Filter wordlists... (/ or ⌘F)"
          bind:value={wordlistsStore.searchFilter}
          class="sidebar-search font-mono"
        />
        {#if wordlistsStore.searchFilter}
          <button class="btn-clear-search" onclick={() => (wordlistsStore.searchFilter = '')} aria-label="Clear search">
            <Icon name="x" size={12} />
          </button>
        {/if}
      </div>

      <div class="sidebar-actions">
        <button
          class="btn-action-primary"
          onclick={() => (newWordlistModalOpen = true)}
          title="Create a new custom wordlist (⌘N)"
        >
          <Icon name="plus" size={13} />
          <span>New</span>
        </button>
        <button
          class="btn-action-subtle"
          onclick={() => wordlistsStore.openNativeFilePicker('import')}
          title="Import wordlist file into app storage (⌘O)"
        >
          <Icon name="upload" size={13} />
          <span>Import</span>
        </button>
        <button
          class="btn-action-subtle"
          onclick={() => wordlistsStore.openNativeFilePicker('link')}
          title="Link external wordlist without copying (⌘L)"
        >
          <Icon name="folder" size={13} />
          <span>Link</span>
        </button>
        <button
          class="btn-action-icon"
          onclick={() => (shortcutsModalOpen = true)}
          title="Keyboard Shortcuts Cheatsheet (?)"
          aria-label="Keyboard Shortcuts"
        >
          <Icon name="info" size={13} />
        </button>
      </div>
    </div>

    <!-- Multi-selection & Batch Action Bar -->
    <div class="selection-bar font-mono">
      <label class="master-checkbox-label" title={wordlistsStore.isAllChecked ? 'Deselect All (⌘A)' : 'Select All (⌘A)'}>
        <input
          bind:this={masterCheckboxRef}
          type="checkbox"
          class="custom-checkbox"
          checked={wordlistsStore.isAllChecked}
          onchange={() => wordlistsStore.toggleCheckAll()}
          disabled={wordlistsStore.filteredWordlists.length === 0}
        />
        <span class="selection-counter">
          {#if wordlistsStore.checkedCount > 0}
            <span class="count-badge">{wordlistsStore.checkedCount} selected</span>
          {:else}
            <span class="count-total">{wordlistsStore.filteredWordlists.length} items</span>
          {/if}
        </span>
      </label>

      {#if wordlistsStore.checkedCount > 0}
        <div class="batch-controls">
          <button
            class="btn-batch-delete"
            onclick={openBatchDeleteModal}
            title="Delete selected wordlists (Delete or Backspace)"
          >
            <Icon name="trash" size={12} />
            <span>Delete ({wordlistsStore.checkedCount})</span>
          </button>
        </div>
      {/if}
    </div>

    <!-- Wordlist Records List -->
    <div class="wordlists-list">
      {#if wordlistsStore.isLoading && wordlistsStore.items.length === 0}
        <div class="loading-state">
          <Icon name="refresh" size={18} class="spin" />
          <span>Loading wordlists...</span>
        </div>
      {:else if wordlistsStore.filteredWordlists.length === 0}
        <div class="empty-list">
          <Icon name="file-text" size={32} />
          <div class="empty-title">No Wordlists Found</div>
          <div class="empty-sub">Import, link, or create a wordlist to get started.</div>
        </div>
      {:else}
        {#each wordlistsStore.filteredWordlists as w, idx (w.id)}
          <div
            bind:this={itemRefs[w.id]}
            class="wordlist-item"
            class:active={wordlistsStore.selectedID === w.id}
            class:focused={focusedIndex === idx}
            class:checked={wordlistsStore.checkedIDs.includes(w.id)}
            onclick={() => {
              focusedIndex = idx;
              wordlistsStore.selectWordlist(w.id);
            }}
            onkeydown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                wordlistsStore.selectWordlist(w.id);
              }
            }}
            role="button"
            tabindex="0"
          >
            <span
              class="item-checkbox-wrap"
              onclick={(e) => e.stopPropagation()}
              role="presentation"
            >
              <input
                type="checkbox"
                class="custom-checkbox"
                checked={wordlistsStore.checkedIDs.includes(w.id)}
                onchange={() => wordlistsStore.toggleCheck(w.id)}
                aria-label="Select {w.name}"
              />
            </span>
            <div class="item-main">
              <span class="item-name" title={w.name}>{w.name}</span>
              <div class="item-tags font-mono">
                <span class="tag-lines">{w.line_count.toLocaleString()} lines</span>
                <span class="tag-size">{(w.size_bytes / 1024).toFixed(1)} KB</span>
                <span class="tag-badge" class:tag-linked={w.is_linked}>
                  {w.is_linked ? 'Linked' : 'Local'}
                </span>
              </div>
            </div>
          </div>
        {/each}
      {/if}
    </div>
  </aside>
  {:else}
    <div class="sidebar-collapsed-strip">
      <button
        type="button"
        class="btn-icon-toggle"
        onclick={() => (sidebarCollapsed = false)}
        title="Expand Library (Cmd+\)"
        aria-label="Expand Library"
      >
        <Icon name="panel-left-open" size={15} />
      </button>
    </div>
  {/if}

  <!-- Right Column: Virtualized Editor & Operations -->
  <main class="editor-pane">
    {#if !selected}
      <div class="empty-editor">
        <Icon name="file-text" size={48} />
        <div class="empty-title">No Wordlist Selected</div>
        <div class="empty-sub">Select a wordlist from the library on the left to edit and inspect lines.</div>
      </div>
    {:else}
      <!-- Editor Header -->
      <header class="editor-header">
        <div class="header-left">
          <div class="title-row">
            <span class="editor-title">{selected.name}</span>
            <span class="type-pill font-mono" class:type-linked={selected.is_linked}>
              {selected.is_linked ? 'Linked File' : 'Managed Wordlist'}
            </span>
          </div>
          <div class="path-sub font-mono" title={selected.path}>{selected.path}</div>
        </div>

        <div class="header-right">
          <button
            class="btn-use-scan"
            onclick={() => handleSelectForScan(selected)}
            title="Set this wordlist as target for FUZZ in Control Bar"
          >
            <Icon name="play" size={13} />
            <span>Use in Scan</span>
          </button>
          <button
            class="btn-delete-wl"
            onclick={() => openDeleteModal([selected])}
            title="Delete or unlink this wordlist (Delete)"
          >
            <Icon name="trash" size={14} />
          </button>
        </div>
      </header>

      <!-- KPI Statistics Strip -->
      <div class="stats-strip font-mono">
        <div class="stat-box">
          <span class="stat-label">Total Lines</span>
          <span class="stat-value">{selected.line_count.toLocaleString()}</span>
        </div>
        <div class="stat-box">
          <span class="stat-label">File Size</span>
          <span class="stat-value">{(selected.size_bytes / 1024).toFixed(1)} KB</span>
        </div>
        {#if stats}
          <div class="stat-box">
            <span class="stat-label">Unique Items</span>
            <span class="stat-value">{stats.unique_lines.toLocaleString()}</span>
          </div>
          <div class="stat-box">
            <span class="stat-label">Avg Length</span>
            <span class="stat-value">{stats.avg_line_len.toFixed(1)} chars</span>
          </div>
          <div class="stat-box">
            <span class="stat-label">Max Length</span>
            <span class="stat-value">{stats.max_line_len} chars</span>
          </div>
        {/if}
      </div>

      <!-- Health Warning Strip (when blank lines or defects detected) -->
      {#if health && !health.is_clean && health.blank_lines_count > 0}
        <div class="health-warning-strip font-mono">
          <div class="health-warning-left">
            <Icon name="alert-triangle" size={13} color="#ff9500" />
            <span>Wordlist contains {health.blank_lines_count.toLocaleString()} blank lines that may corrupt fuzzing payloads</span>
          </div>
          <button class="btn-clean-pill" onclick={handleCleanWordlist} title="Remove all blank lines and trim whitespace (Shift+Cmd+C)">
            <Icon name="check" size={12} />
            <span>Auto-Clean Now</span>
          </button>
        </div>
      {/if}

      <!-- Editor Toolbar (Transformations & Pagination) -->
      <div class="editor-toolbar">
        <div class="tools-group">
          <!-- View Segmented Control (Pinned) -->
          <div class="segmented-control" role="group" aria-label="Editor View Mode">
            <button
              type="button"
              class="segment-btn"
              class:active={editorMode === 'lines'}
              onclick={() => setEditorMode('lines')}
              title="Line-by-line structured editor (Cmd+E)"
            >
              <Icon name="layers" size={11} />
              <span>Lines</span>
            </button>
            <button
              type="button"
              class="segment-btn"
              class:active={editorMode === 'raw'}
              onclick={() => setEditorMode('raw')}
              title="Raw multiline view - Cmd+A selects all lines (Cmd+E)"
            >
              <Icon name="file-text" size={11} />
              <span>Raw</span>
            </button>
          </div>

          <!-- Selection & Copy Group (Pinned) -->
          <div class="btn-group">
            <button
              type="button"
              class="tool-btn"
              class:active={selectedLineIndices.size > 0 && selectedLineIndices.size === editorLines.length}
              onclick={handleSelectAll}
              title={editorMode === 'raw' ? 'Select all text (Cmd+A)' : selectedLineIndices.size === editorLines.length ? 'Deselect all lines' : 'Select all lines (Shift+Cmd+A)'}
            >
              <Icon name="check" size={11} />
              <span>{selectedLineIndices.size > 0 && editorMode === 'lines' ? `All (${selectedLineIndices.size})` : 'Select All'}</span>
            </button>
            <button
              type="button"
              class="tool-btn"
              onclick={copyAllLines}
              title={selectedLineIndices.size > 0 ? `Copy ${selectedLineIndices.size} selected lines to clipboard` : 'Copy lines to clipboard'}
            >
              <Icon name="copy" size={11} />
              <span>Copy</span>
            </button>
            {#if selectedLineIndices.size > 0 && editorMode === 'lines'}
              <button
                type="button"
                class="tool-btn btn-tool-danger"
                onclick={deleteSelectedLines}
                title="Delete selected lines (Backspace or Delete)"
              >
                <Icon name="trash" size={11} />
                <span>Delete</span>
              </button>
            {/if}
          </div>

          <!-- Extended Tools (Hidden in Compact Mode via @container query) -->
          <div class="tools-extended">
            <!-- Hygiene Group -->
            <div class="btn-group">
              <button
                type="button"
                class="tool-btn btn-tool-clean"
                onclick={handleCleanWordlist}
                title="Verify and auto-clean blank lines (Shift+Cmd+C)"
              >
                <Icon name="check" size={11} />
                <span>Clean<span class="btn-label-long"> Blanks</span></span>
              </button>
              <button
                type="button"
                class="tool-btn"
                onclick={applyDeduplicate}
                title="Remove duplicate lines (Shift+Cmd+D)"
              >
                <Icon name="sliders" size={11} />
                <span>Dedupe</span>
              </button>
            </div>

            <!-- Sort Dropdown -->
            <div class="dropdown-wrapper">
              <button
                type="button"
                class="tool-btn dropdown-toggle"
                class:dropdown-active={sortDropdownOpen}
                onclick={(e) => {
                  e.stopPropagation();
                  sortDropdownOpen = !sortDropdownOpen;
                  transformDropdownOpen = false;
                  moreDropdownOpen = false;
                }}
                title="Sort lines alphabetically"
              >
                <Icon name="arrow-up-down" size={11} />
                <span>Sort</span>
                <Icon name="chevron-down" size={10} />
              </button>
              {#if sortDropdownOpen}
                <div class="dropdown-menu">
                  <button
                    type="button"
                    class="dropdown-item"
                    onclick={() => {
                      applySort(true);
                      sortDropdownOpen = false;
                    }}
                  >
                    <Icon name="arrow-up" size={11} />
                    <span>Sort Ascending (A-Z)</span>
                  </button>
                  <button
                    type="button"
                    class="dropdown-item"
                    onclick={() => {
                      applySort(false);
                      sortDropdownOpen = false;
                    }}
                  >
                    <Icon name="arrow-down" size={11} />
                    <span>Sort Descending (Z-A)</span>
                  </button>
                </div>
              {/if}
            </div>

            <!-- Transform Dropdown -->
            <div class="dropdown-wrapper">
              <button
                type="button"
                class="tool-btn dropdown-toggle"
                class:dropdown-active={transformDropdownOpen}
                onclick={(e) => {
                  e.stopPropagation();
                  transformDropdownOpen = !transformDropdownOpen;
                  sortDropdownOpen = false;
                  moreDropdownOpen = false;
                }}
                title="Transform line text"
              >
                <Icon name="edit" size={11} />
                <span>Transform</span>
                <Icon name="chevron-down" size={10} />
              </button>
              {#if transformDropdownOpen}
                <div class="dropdown-menu">
                  <button
                    type="button"
                    class="dropdown-item"
                    onclick={() => {
                      applyTrim();
                      transformDropdownOpen = false;
                    }}
                  >
                    <span>Trim Whitespace</span>
                  </button>
                  <button
                    type="button"
                    class="dropdown-item"
                    onclick={() => {
                      applyCase(true);
                      transformDropdownOpen = false;
                    }}
                  >
                    <span>To Lowercase (lower)</span>
                  </button>
                  <button
                    type="button"
                    class="dropdown-item"
                    onclick={() => {
                      applyCase(false);
                      transformDropdownOpen = false;
                    }}
                  >
                    <span>To Uppercase (UPPER)</span>
                  </button>
                  <button
                    type="button"
                    class="dropdown-item"
                    onclick={() => {
                      applyStripEmpty();
                      transformDropdownOpen = false;
                    }}
                  >
                    <span>Strip Empty Lines</span>
                  </button>
                  <div class="dropdown-divider"></div>
                  <button
                    type="button"
                    class="dropdown-item"
                    onclick={() => {
                      prefixSuffixModalOpen = true;
                      transformDropdownOpen = false;
                    }}
                  >
                    <span>Add Prefix / Suffix...</span>
                  </button>
                </div>
              {/if}
            </div>
          </div>

          <!-- Apple HIG Overflow Menu (Visible in Compact Mode via @container query) -->
          <div class="dropdown-wrapper tools-overflow">
            <button
              type="button"
              class="tool-btn dropdown-toggle"
              class:dropdown-active={moreDropdownOpen}
              onclick={(e) => {
                e.stopPropagation();
                moreDropdownOpen = !moreDropdownOpen;
                sortDropdownOpen = false;
                transformDropdownOpen = false;
              }}
              title="More Actions and Transformations"
            >
              <Icon name="ellipsis" size={12} />
              <span>More</span>
              <Icon name="chevron-down" size={10} />
            </button>
            {#if moreDropdownOpen}
              <div class="dropdown-menu">
                <div class="dropdown-section-title">Hygiene</div>
                <button
                  type="button"
                  class="dropdown-item"
                  onclick={() => {
                    handleCleanWordlist();
                    moreDropdownOpen = false;
                  }}
                >
                  <Icon name="check" size={11} />
                  <span>Clean Blank Lines</span>
                </button>
                <button
                  type="button"
                  class="dropdown-item"
                  onclick={() => {
                    applyDeduplicate();
                    moreDropdownOpen = false;
                  }}
                >
                  <Icon name="sliders" size={11} />
                  <span>Deduplicate Lines</span>
                </button>
                <div class="dropdown-divider"></div>
                <div class="dropdown-section-title">Sort</div>
                <button
                  type="button"
                  class="dropdown-item"
                  onclick={() => {
                    applySort(true);
                    moreDropdownOpen = false;
                  }}
                >
                  <Icon name="arrow-up" size={11} />
                  <span>Sort Ascending (A-Z)</span>
                </button>
                <button
                  type="button"
                  class="dropdown-item"
                  onclick={() => {
                    applySort(false);
                    moreDropdownOpen = false;
                  }}
                >
                  <Icon name="arrow-down" size={11} />
                  <span>Sort Descending (Z-A)</span>
                </button>
                <div class="dropdown-divider"></div>
                <div class="dropdown-section-title">Transform</div>
                <button
                  type="button"
                  class="dropdown-item"
                  onclick={() => {
                    applyTrim();
                    moreDropdownOpen = false;
                  }}
                >
                  <span>Trim Whitespace</span>
                </button>
                <button
                  type="button"
                  class="dropdown-item"
                  onclick={() => {
                    applyCase(true);
                    moreDropdownOpen = false;
                  }}
                >
                  <span>To Lowercase (lower)</span>
                </button>
                <button
                  type="button"
                  class="dropdown-item"
                  onclick={() => {
                    applyCase(false);
                    moreDropdownOpen = false;
                  }}
                >
                  <span>To Uppercase (UPPER)</span>
                </button>
                <button
                  type="button"
                  class="dropdown-item"
                  onclick={() => {
                    applyStripEmpty();
                    moreDropdownOpen = false;
                  }}
                >
                  <span>Strip Empty Lines</span>
                </button>
                <div class="dropdown-divider"></div>
                <button
                  type="button"
                  class="dropdown-item"
                  onclick={() => {
                    prefixSuffixModalOpen = true;
                    moreDropdownOpen = false;
                  }}
                >
                  <span>Add Prefix / Suffix...</span>
                </button>
              </div>
            {/if}
          </div>
        </div>

        <!-- Pagination Controls -->
        <div class="pagination-group font-mono">
          <button
            type="button"
            class="btn-page"
            disabled={pageOffset === 0}
            onclick={() => handlePageNav(pageOffset - pageSize)}
            title="Previous page (Alt+Left)"
          >
            Prev
          </button>
          <span class="page-indicator">
            <span class="page-indicator-full">
              {totalLines > 0 ? pageOffset + 1 : 0} - {Math.min(pageOffset + pageSize, totalLines)} of {totalLines.toLocaleString()}
            </span>
            <span class="page-indicator-compact">
              {totalLines > 0 ? pageOffset + 1 : 0}-{Math.min(pageOffset + pageSize, totalLines)} / {totalLines >= 1000 ? `${(totalLines / 1000).toFixed(1)}k` : totalLines}
            </span>
          </span>
          <button
            type="button"
            class="btn-page"
            disabled={pageOffset + pageSize >= totalLines}
            onclick={() => handlePageNav(pageOffset + pageSize)}
            title="Next page (Alt+Right)"
          >
            Next
          </button>
        </div>
      </div>

      {#if editorMode === 'lines'}
        <!-- Virtualized Line-by-Line Editor -->
        <div class="editor-container">
          <div class="lines-gutter font-mono select-none">
            {#each editorLines as _, idx}
              <button
                type="button"
                class="gutter-number"
                class:gutter-selected={selectedLineIndices.has(idx)}
                onclick={(e) => toggleLineSelect(idx, e)}
                title="Click to select line (Shift+Click for range)"
              >
                {pageOffset + idx + 1}
              </button>
            {/each}
          </div>
          <div class="lines-content font-mono select-text">
            {#each editorLines as line, idx}
              <div class="line-row" class:line-row-selected={selectedLineIndices.has(idx)}>
                <input
                  bind:this={lineInputRefs[idx]}
                  type="text"
                  class="line-input"
                  value={line}
                  oninput={(e) => handleLineChange(idx, (e.currentTarget as HTMLInputElement).value)}
                  onkeydown={(e) => handleLineInputKeyDown(e, idx)}
                  onpaste={(e) => handleLinePaste(e, idx)}
                  onfocus={() => {
                    if (selectedLineIndices.size > 0 && !selectedLineIndices.has(idx)) {
                      selectedLineIndices = new Set();
                    }
                  }}
                />
              </div>
            {/each}
          </div>
        </div>
      {:else}
        <!-- Raw Multiline / Bulk Editor (Cmd+A selects all lines) -->
        <div class="editor-container raw-editor-container">
          <div class="lines-gutter font-mono select-none">
            {#each editorLines as _, idx}
              <div class="gutter-number">{pageOffset + idx + 1}</div>
            {/each}
          </div>
          <div class="raw-textarea-wrap">
            <textarea
              bind:this={rawTextareaRef}
              class="raw-textarea font-mono select-text"
              value={rawTextContent}
              oninput={(e) => handleRawTextChange((e.currentTarget as HTMLTextAreaElement).value)}
              placeholder="Enter wordlist payloads, one per line..."
              spellcheck="false"
            ></textarea>
          </div>
        </div>
      {/if}

      <!-- Editor Footer Action Bar -->
      <footer class="editor-footer">
        <div class="footer-left">
          {#if isDirty}
            <span class="dirty-badge font-mono">Unsaved Changes (⌘S to save)</span>
          {/if}
        </div>
        <div class="footer-right">
          <button
            class="btn-footer-save"
            onclick={handleSave}
            disabled={!isDirty || wordlistsStore.isSaving}
            title="Save changes (⌘S)"
          >
            <Icon name="save" size={14} />
            <span>{wordlistsStore.isSaving ? 'Saving...' : 'Save Wordlist (⌘S)'}</span>
          </button>
        </div>
      </footer>
    {/if}
  </main>
</div>

<!-- Modal: Confirm Delete Wordlist(s) -->
{#if confirmDeleteModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) confirmDeleteModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (confirmDeleteModalOpen = false)}
  >
    <div class="modal-dialog modal-dialog-delete" role="document">
      <div class="modal-head">
        <div class="modal-head-title">
          <Icon name="alert-triangle" size={16} color="#ff3b30" />
          <span>{deleteTargets.length === 1 ? 'Delete Wordlist' : `Delete ${deleteTargets.length} Wordlists`}</span>
        </div>
        <button class="btn-close-modal" onclick={() => (confirmDeleteModalOpen = false)} aria-label="Close">
          <Icon name="x" size={16} />
        </button>
      </div>

      <div class="modal-body-content">
        <div class="delete-warning-text">
          {#if deleteTargets.length === 1}
            Are you sure you want to delete <strong class="target-name font-mono">{deleteTargets[0].name}</strong>?
          {:else}
            Are you sure you want to delete <strong class="target-name font-mono">{deleteTargets.length} wordlists</strong>?
          {/if}
        </div>

        <div class="delete-target-list font-mono">
          {#each deleteTargets.slice(0, 5) as t (t.id)}
            <div class="target-row">
              <Icon name={t.is_linked ? 'folder' : 'file-text'} size={13} color="var(--text-secondary)" />
              <span class="target-filename">{t.name}</span>
              <span class="target-kind-pill" class:pill-linked={t.is_linked}>
                {t.is_linked ? 'Linked' : 'Local File'}
              </span>
            </div>
          {/each}
          {#if deleteTargets.length > 5}
            <div class="target-row-more">
              ... and {deleteTargets.length - 5} more wordlists
            </div>
          {/if}
        </div>

        <div class="delete-impact-notice">
          <Icon name="info" size={14} color="var(--text-secondary)" />
          <span>
            Managed local files are permanently deleted from disk. Linked wordlists are removed from ffuf without deleting the original source file.
          </span>
        </div>
      </div>

      <div class="modal-foot">
        <span class="shortcut-tip font-mono">Esc to cancel · Enter to delete</span>
        <button class="btn-cancel" onclick={() => (confirmDeleteModalOpen = false)}>Cancel</button>
        <button class="btn-danger-confirm" onclick={confirmDelete}>
          <Icon name="trash" size={13} />
          <span>Delete {deleteTargets.length > 1 ? `(${deleteTargets.length})` : ''}</span>
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Discard Unsaved Changes Confirmation -->
{#if unsavedDiscardModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) unsavedDiscardModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (unsavedDiscardModalOpen = false)}
  >
    <div class="modal-dialog" role="document">
      <div class="modal-head">
        <div class="modal-head-title">
          <Icon name="alert-circle" size={16} color="#ff9500" />
          <span>Unsaved Changes</span>
        </div>
        <button class="btn-close-modal" onclick={() => (unsavedDiscardModalOpen = false)} aria-label="Close">
          <Icon name="x" size={16} />
        </button>
      </div>
      <div class="modal-body-content">
        <p class="dialog-desc">
          You have unsaved edits in the current page of lines. Navigating to another page will discard these changes.
        </p>
      </div>
      <div class="modal-foot">
        <span class="shortcut-tip font-mono">Esc to keep · Enter to discard</span>
        <button class="btn-cancel" onclick={() => (unsavedDiscardModalOpen = false)}>Keep Editing</button>
        <button class="btn-warning-confirm" onclick={confirmDiscardNav}>Discard & Navigate</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Keyboard Shortcuts Cheatsheet -->
{#if shortcutsModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) shortcutsModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (shortcutsModalOpen = false)}
  >
    <div class="modal-dialog modal-dialog-shortcuts" role="document">
      <div class="modal-head">
        <div class="modal-head-title">
          <Icon name="info" size={16} color="var(--accent)" />
          <span>Wordlist Manager Keyboard Shortcuts</span>
        </div>
        <button class="btn-close-modal" onclick={() => (shortcutsModalOpen = false)} aria-label="Close">
          <Icon name="x" size={16} />
        </button>
      </div>

      <div class="modal-body-content shortcuts-grid font-mono">
        <div class="shortcut-section">
          <div class="section-title">Library & Selection</div>
          <div class="shortcut-row"><span class="shortcut-key">Up / Down</span><span class="shortcut-desc">Navigate library list</span></div>
          <div class="shortcut-row"><span class="shortcut-key">j / k</span><span class="shortcut-desc">Vim down / up navigation</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Space</span><span class="shortcut-desc">Toggle selection checkbox</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Enter</span><span class="shortcut-desc">Open wordlist in editor</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Cmd+A</span><span class="shortcut-desc">Select all / Deselect all</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Del / Backspace</span><span class="shortcut-desc">Delete selected wordlist(s)</span></div>
          <div class="shortcut-row"><span class="shortcut-key">/ or Cmd+F</span><span class="shortcut-desc">Focus search filter</span></div>
        </div>

        <div class="shortcut-section">
          <div class="section-title">Editor & Tools</div>
          <div class="shortcut-row"><span class="shortcut-key">Cmd+S</span><span class="shortcut-desc">Save wordlist changes</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Cmd+N</span><span class="shortcut-desc">Create new wordlist</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Cmd+O</span><span class="shortcut-desc">Import wordlist file</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Cmd+L</span><span class="shortcut-desc">Link external wordlist</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Alt+Left / Right</span><span class="shortcut-desc">Previous / Next page</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Shift+Cmd+D</span><span class="shortcut-desc">Deduplicate lines</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Shift+Cmd+S</span><span class="shortcut-desc">Sort lines alphabetically</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Shift+Cmd+T</span><span class="shortcut-desc">Trim whitespace</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Cmd+E</span><span class="shortcut-desc">Toggle Lines / Raw Mode</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Shift+Cmd+A</span><span class="shortcut-desc">Select all lines (Raw)</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Enter / Backspace</span><span class="shortcut-desc">Add / remove line</span></div>
          <div class="shortcut-row"><span class="shortcut-key">Shift+Cmd+C</span><span class="shortcut-desc">Auto-clean blank lines</span></div>
          <div class="shortcut-row"><span class="shortcut-key">?</span><span class="shortcut-desc">Toggle this shortcuts list</span></div>
        </div>
      </div>

      <div class="modal-foot">
        <button class="btn-confirm" onclick={() => (shortcutsModalOpen = false)}>Close (Esc)</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Create New Wordlist -->
{#if newWordlistModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) newWordlistModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (newWordlistModalOpen = false)}
  >
    <div class="modal-dialog" role="document">
      <div class="modal-head">
        <span>Create New Wordlist</span>
        <button class="btn-close-modal" onclick={() => (newWordlistModalOpen = false)} aria-label="Close">
          <Icon name="x" size={16} />
        </button>
      </div>
      <div class="modal-body-content">
        <label class="modal-label">
          Wordlist Name:
          <input
            type="text"
            placeholder="e.g. custom_routes.txt"
            bind:value={newWordlistName}
            class="modal-input font-mono"
          />
        </label>
        <label class="modal-label">
          Initial Lines (one per line):
          <textarea
            rows="8"
            placeholder="admin&#10;api&#10;login&#10;v1&#10;v2"
            bind:value={initialLines}
            class="modal-textarea font-mono"
          ></textarea>
        </label>
      </div>
      <div class="modal-foot">
        <span class="shortcut-tip font-mono">Cmd+Enter to submit</span>
        <button class="btn-cancel" onclick={() => (newWordlistModalOpen = false)}>Cancel</button>
        <button class="btn-confirm" onclick={handleCreateNew} disabled={!newWordlistName.trim()}>Create</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Prefix / Suffix -->
{#if prefixSuffixModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) prefixSuffixModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (prefixSuffixModalOpen = false)}
  >
    <div class="modal-dialog" role="document">
      <div class="modal-head">
        <span>Add Prefix & Suffix to Lines</span>
        <button class="btn-close-modal" onclick={() => (prefixSuffixModalOpen = false)} aria-label="Close">
          <Icon name="x" size={16} />
        </button>
      </div>
      <div class="modal-body-content">
        <label class="modal-label">
          Prefix (prepend to all lines):
          <input type="text" placeholder="e.g. /api/v1/" bind:value={prefixInput} class="modal-input font-mono" />
        </label>
        <label class="modal-label">
          Suffix (append to all lines):
          <input type="text" placeholder="e.g. .php or .json" bind:value={suffixInput} class="modal-input font-mono" />
        </label>
      </div>
      <div class="modal-foot">
        <span class="shortcut-tip font-mono">Cmd+Enter to apply</span>
        <button class="btn-cancel" onclick={() => (prefixSuffixModalOpen = false)}>Cancel</button>
        <button class="btn-confirm" onclick={applyPrefixSuffix}>Apply to Lines</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .wordlist-manager {
    display: flex;
    flex: 1;
    height: 100%;
    min-height: 0;
    background-color: var(--surface);
  }

  .sidebar-library {
    width: 270px;
    background-color: var(--bg-subtle);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
  }

  .sidebar-collapsed-strip {
    width: 36px;
    background-color: var(--bg-subtle);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    align-items: center;
    padding-top: 8px;
    flex-shrink: 0;
  }

  .sidebar-title-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 2px 2px 2px;
  }

  .sidebar-heading {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .btn-icon-toggle {
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: 1px solid transparent;
    border-radius: 4px;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.1s ease;
  }

  .btn-icon-toggle:hover {
    background-color: var(--surface-hover);
    color: var(--text);
    border-color: var(--border);
  }

  .sidebar-header {
    padding: 10px;
    border-bottom: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .search-wrap {
    position: relative;
    display: flex;
    align-items: center;
  }

  :global(.search-icon) {
    position: absolute;
    left: 8px;
    color: var(--text-secondary);
  }

  .sidebar-search {
    width: 100%;
    height: 26px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 24px 0 26px;
    font-size: 11px;
    color: var(--text);
    box-sizing: border-box;
  }

  .btn-clear-search {
    position: absolute;
    right: 4px;
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
  }

  .sidebar-actions {
    display: flex;
    gap: 4px;
  }

  .btn-action-primary,
  .btn-action-subtle,
  .btn-action-icon {
    height: 24px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    border-radius: 4px;
    font-size: 11px;
    cursor: pointer;
  }

  .btn-action-primary {
    flex: 1;
    background-color: var(--accent);
    color: var(--accent-contrast);
    border: none;
    font-weight: 500;
  }

  .btn-action-subtle {
    flex: 1;
    background-color: var(--surface);
    color: var(--text);
    border: 1px solid var(--border);
  }

  .btn-action-icon {
    width: 24px;
    background-color: var(--surface);
    color: var(--text-secondary);
    border: 1px solid var(--border);
    padding: 0;
  }

  .btn-action-icon:hover {
    color: var(--text);
  }

  /* Selection Bar & Master Checkbox */
  .selection-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 10px;
    background-color: var(--surface);
    border-bottom: 1px solid var(--border);
    font-size: 11px;
    min-height: 28px;
  }

  .master-checkbox-label {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    user-select: none;
  }

  .custom-checkbox {
    width: 14px;
    height: 14px;
    margin: 0;
    cursor: pointer;
    accent-color: var(--accent);
  }

  .selection-counter {
    display: inline-flex;
    align-items: center;
  }

  .count-badge {
    background-color: rgba(0, 122, 255, 0.15);
    color: var(--accent);
    padding: 1px 6px;
    border-radius: 10px;
    font-size: 10.5px;
    font-weight: 600;
  }

  .count-total {
    color: var(--text-secondary);
    font-size: 10.5px;
  }

  .batch-controls {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .btn-batch-delete {
    height: 20px;
    padding: 0 6px;
    background-color: rgba(255, 59, 48, 0.12);
    color: #ff3b30;
    border: 1px solid rgba(255, 59, 48, 0.3);
    border-radius: 3px;
    font-size: 10.5px;
    font-weight: 600;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    cursor: pointer;
    transition: all 0.1s ease;
  }

  .btn-batch-delete:hover {
    background-color: #ff3b30;
    color: #ffffff;
  }

  /* Wordlist Item List */
  .wordlists-list {
    flex: 1;
    overflow-y: auto;
    padding: 6px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .wordlist-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    background-color: var(--surface);
    border: 1px solid transparent;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.08s ease;
  }

  .wordlist-item:hover {
    background-color: var(--surface-hover);
  }

  .wordlist-item.focused {
    outline: 2px solid var(--accent);
    outline-offset: -1px;
  }

  .wordlist-item.active {
    background-color: rgba(0, 122, 255, 0.1);
    border-color: var(--accent);
  }

  .wordlist-item.checked {
    border-color: rgba(0, 122, 255, 0.4);
  }

  .item-checkbox-wrap {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .item-main {
    flex: 1;
    min-width: 0;
  }

  .item-name {
    font-size: 12px;
    font-weight: 600;
    color: var(--text);
    display: block;
    margin-bottom: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .item-tags {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 10px;
    color: var(--text-secondary);
  }

  .tag-badge {
    background-color: rgba(128, 128, 128, 0.15);
    padding: 1px 4px;
    border-radius: 3px;
  }

  .tag-linked {
    background-color: rgba(0, 122, 255, 0.15);
    color: var(--accent);
  }

  /* Editor Pane */
  .editor-pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    background-color: var(--surface);
    container-type: inline-size;
    container-name: editor;
  }

  .empty-editor,
  .empty-list {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
    gap: 8px;
    padding: 24px;
    text-align: center;
  }

  .editor-header {
    height: 52px;
    padding: 0 16px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    background-color: var(--bg-subtle);
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .editor-title {
    font-size: 14px;
    font-weight: 700;
    color: var(--text);
  }

  .type-pill {
    font-size: 10px;
    padding: 1px 6px;
    border-radius: 10px;
    background-color: rgba(52, 199, 89, 0.15);
    color: #248a3d;
  }

  .type-linked {
    background-color: rgba(0, 122, 255, 0.15);
    color: var(--accent);
  }

  .path-sub {
    font-size: 11px;
    color: var(--text-secondary);
    margin-top: 2px;
    max-width: 480px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .btn-use-scan {
    height: 28px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background-color: var(--accent);
    color: var(--accent-contrast);
    border: none;
    border-radius: 4px;
    padding: 0 10px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
  }

  .btn-delete-wl {
    height: 28px;
    width: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: 1px solid var(--border);
    border-radius: 4px;
    color: var(--text-secondary);
    cursor: pointer;
  }

  .btn-delete-wl:hover {
    color: #ff3b30;
    border-color: #ff3b30;
  }

  .stats-strip {
    display: flex;
    gap: 16px;
    padding: 8px 16px;
    background-color: var(--surface);
    border-bottom: 1px solid var(--border);
    font-size: 11px;
  }

  .stat-box {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .stat-label {
    font-size: 9.5px;
    color: var(--text-secondary);
    text-transform: uppercase;
  }

  .stat-value {
    font-weight: 600;
    color: var(--text);
  }

  .health-warning-strip {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 16px;
    background-color: rgba(255, 149, 0, 0.12);
    border-bottom: 1px solid rgba(255, 149, 0, 0.28);
    font-size: 11px;
    color: #ff9500;
  }

  .health-warning-left {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 500;
  }

  .btn-clean-pill {
    height: 22px;
    padding: 0 10px;
    background-color: #ff9500;
    color: #ffffff;
    border: none;
    border-radius: 12px;
    font-size: 10.5px;
    font-weight: 600;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    transition: background-color 0.1s ease;
  }

  .btn-clean-pill:hover {
    background-color: #e08500;
  }

  .editor-toolbar {
    height: 38px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px;
    gap: 8px;
    position: relative;
    z-index: 20;
    white-space: nowrap;
    user-select: none;
    overflow-x: auto;
    scrollbar-width: none;
    -ms-overflow-style: none;
  }

  .editor-toolbar::-webkit-scrollbar {
    display: none;
  }

  .tools-group {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .tools-extended {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  .tools-overflow {
    display: none;
  }

  .dropdown-section-title {
    font-size: 10px;
    font-weight: 600;
    color: var(--text-secondary);
    padding: 4px 8px 2px 8px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .segmented-control {
    display: inline-flex;
    align-items: center;
    background-color: var(--bg-hover);
    border-radius: 6px;
    padding: 2px;
    gap: 2px;
  }

  .segment-btn {
    height: 24px;
    padding: 0 8px;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text-secondary);
    font-size: 11px;
    font-weight: 500;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.12s ease;
  }

  .segment-btn:hover {
    color: var(--text);
  }

  .segment-btn.active {
    background-color: var(--surface);
    color: var(--text);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
    font-weight: 600;
  }

  .btn-group {
    display: inline-flex;
    align-items: center;
    border-radius: 5px;
    overflow: hidden;
    border: 1px solid var(--border);
    background-color: var(--surface);
  }

  .btn-group .tool-btn {
    border: none;
    border-right: 1px solid var(--border);
    border-radius: 0;
  }

  .btn-group .tool-btn:last-child {
    border-right: none;
  }

  .tool-btn {
    height: 24px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 5px;
    padding: 0 8px;
    font-size: 11px;
    font-weight: 500;
    color: var(--text);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    white-space: nowrap;
    flex-shrink: 0;
    line-height: 1;
    transition: background-color 0.1s ease, color 0.1s ease;
  }

  .tool-btn:hover {
    background-color: var(--surface-hover);
  }

  .tool-btn.active {
    background-color: rgba(0, 122, 255, 0.12);
    color: var(--accent);
  }

  .btn-tool-clean {
    color: var(--accent);
  }

  .btn-tool-danger {
    color: var(--danger, #ff453a);
  }

  .btn-tool-danger:hover {
    background-color: rgba(255, 69, 58, 0.1);
  }

  .dropdown-wrapper {
    position: relative;
    display: inline-flex;
  }

  .dropdown-toggle.dropdown-active {
    background-color: var(--surface-hover);
    border-color: var(--accent);
  }

  .dropdown-menu {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    min-width: 175px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.28);
    padding: 4px;
    z-index: 100;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .dropdown-item {
    height: 26px;
    display: flex;
    align-items: center;
    gap: 7px;
    padding: 0 8px;
    background: transparent;
    border: none;
    border-radius: 4px;
    color: var(--text);
    font-size: 11px;
    cursor: pointer;
    white-space: nowrap;
    text-align: left;
    width: 100%;
    transition: background-color 0.1s ease;
  }

  .dropdown-item:hover {
    background-color: var(--surface-hover);
    color: var(--text);
  }

  .dropdown-divider {
    height: 1px;
    background-color: var(--border);
    margin: 3px 0;
  }

  .pagination-group {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    flex-shrink: 0;
  }

  .btn-page {
    height: 24px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 8px;
    font-size: 10.5px;
    color: var(--text);
    cursor: pointer;
    white-space: nowrap;
  }

  .btn-page:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .page-indicator {
    color: var(--text-secondary);
    font-size: 10.5px;
    white-space: nowrap;
  }

  .page-indicator-compact {
    display: none;
  }

  /* Container Queries for Adaptive Toolbar */
  @container editor (max-width: 760px) {
    .btn-label-long {
      display: none !important;
    }
  }

  @container editor (max-width: 620px) {
    .tools-extended {
      display: none !important;
    }

    .tools-overflow {
      display: inline-flex !important;
    }

    .btn-label-long {
      display: none !important;
    }

    .page-indicator-full {
      display: none !important;
    }

    .page-indicator-compact {
      display: inline !important;
    }
  }

  .editor-container {
    flex: 1;
    overflow-y: auto;
    display: flex;
    background-color: var(--surface);
  }

  .lines-gutter {
    width: 52px;
    background-color: var(--bg-subtle);
    border-right: 1px solid var(--border);
    padding: 6px 0;
    text-align: right;
    color: var(--text-secondary);
    font-size: 11px;
    line-height: 24px;
  }

  .gutter-number {
    display: block;
    width: 100%;
    height: 24px;
    padding: 0 8px 0 0;
    background: none;
    border: none;
    color: var(--text-secondary);
    font-family: inherit;
    font-size: 11px;
    line-height: 24px;
    text-align: right;
    cursor: pointer;
    user-select: none;
    transition: background-color 0.1s ease, color 0.1s ease;
  }

  .gutter-number:hover {
    background-color: var(--bg-hover);
    color: var(--text);
  }

  .gutter-number.gutter-selected {
    background-color: rgba(0, 122, 255, 0.22);
    color: var(--accent);
    font-weight: 600;
  }

  .lines-content {
    flex: 1;
    padding: 6px 0;
  }

  .line-row {
    display: flex;
    align-items: center;
    height: 24px;
    border-left: 2px solid transparent;
    transition: background-color 0.1s ease;
  }

  .line-row.line-row-selected {
    background-color: rgba(0, 122, 255, 0.12);
    border-left-color: var(--accent);
  }

  .line-input {
    width: 100%;
    height: 24px;
    padding: 0 8px;
    background: none;
    border: none;
    outline: none;
    font-size: 12px;
    color: var(--text);
    box-sizing: border-box;
  }

  .line-input:focus {
    background-color: rgba(0, 122, 255, 0.08);
  }

  .raw-editor-container {
    position: relative;
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  .raw-textarea-wrap {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .raw-textarea {
    flex: 1;
    width: 100%;
    height: 100%;
    padding: 6px 12px;
    background-color: var(--surface);
    color: var(--text);
    border: none;
    outline: none;
    resize: none;
    font-size: 12px;
    line-height: 24px;
    white-space: pre;
    overflow-y: auto;
    overflow-x: auto;
    box-sizing: border-box;
  }

  .raw-textarea:focus {
    background-color: rgba(0, 122, 255, 0.03);
  }

  .editor-footer {
    height: 40px;
    background-color: var(--bg-subtle);
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
  }

  .dirty-badge {
    font-size: 11px;
    color: #ff9500;
    font-weight: 600;
  }

  .btn-footer-save {
    height: 26px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background-color: var(--accent);
    color: var(--accent-contrast);
    border: none;
    border-radius: 4px;
    padding: 0 12px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
  }

  .btn-footer-save:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Modals */
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background-color: rgba(0, 0, 0, 0.45);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
    backdrop-filter: blur(2px);
  }

  .modal-dialog {
    width: 480px;
    max-width: 90vw;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    box-shadow: 0 8px 30px rgba(0, 0, 0, 0.25);
    overflow: hidden;
  }

  .modal-dialog-shortcuts {
    width: 580px;
  }

  .modal-dialog-delete {
    width: 460px;
  }

  .modal-head {
    height: 38px;
    padding: 0 14px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 12px;
    font-weight: 600;
  }

  .modal-head-title {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .btn-close-modal {
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
  }

  .modal-body-content {
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .delete-warning-text {
    font-size: 13px;
    color: var(--text);
    line-height: 1.5;
  }

  .target-name {
    color: #ff3b30;
  }

  .delete-target-list {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 6px 10px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-height: 140px;
    overflow-y: auto;
  }

  .target-row {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
  }

  .target-filename {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text);
  }

  .target-kind-pill {
    font-size: 9.5px;
    padding: 1px 5px;
    border-radius: 3px;
    background-color: rgba(128, 128, 128, 0.15);
    color: var(--text-secondary);
  }

  .target-kind-pill.pill-linked {
    background-color: rgba(0, 122, 255, 0.15);
    color: var(--accent);
  }

  .target-row-more {
    font-size: 10px;
    color: var(--text-secondary);
    text-align: center;
    padding-top: 2px;
  }

  .delete-impact-notice {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    background-color: rgba(128, 128, 128, 0.08);
    border-radius: 6px;
    padding: 8px 10px;
    font-size: 11px;
    color: var(--text-secondary);
    line-height: 1.4;
  }

  .dialog-desc {
    font-size: 12px;
    color: var(--text);
    margin: 0;
    line-height: 1.4;
  }

  /* Shortcuts Cheatsheet Grid */
  .shortcuts-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
    font-size: 11px;
  }

  .shortcut-section {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .section-title {
    font-size: 11.5px;
    font-weight: 700;
    color: var(--text);
    border-bottom: 1px solid var(--border);
    padding-bottom: 4px;
    margin-bottom: 2px;
  }

  .shortcut-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .shortcut-key {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 3px;
    padding: 1px 5px;
    font-size: 10px;
    color: var(--text);
    white-space: nowrap;
  }

  .shortcut-desc {
    color: var(--text-secondary);
    font-size: 10.5px;
    text-align: right;
  }

  .modal-label {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 11px;
    color: var(--text-secondary);
  }

  .modal-input {
    height: 28px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 8px;
    font-size: 12px;
    color: var(--text);
  }

  .modal-textarea {
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 8px;
    font-size: 12px;
    color: var(--text);
    resize: vertical;
  }

  .modal-foot {
    height: 44px;
    padding: 0 14px;
    background-color: var(--bg-subtle);
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
  }

  .shortcut-tip {
    font-size: 10px;
    color: var(--text-secondary);
    margin-right: auto;
  }

  .btn-cancel {
    height: 28px;
    padding: 0 12px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    font-size: 11px;
    cursor: pointer;
    color: var(--text);
  }

  .btn-confirm {
    height: 28px;
    padding: 0 14px;
    background-color: var(--accent);
    color: var(--accent-contrast);
    border: none;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
  }

  .btn-danger-confirm {
    height: 28px;
    padding: 0 14px;
    background-color: #ff3b30;
    color: #ffffff;
    border: none;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
  }

  .btn-danger-confirm:hover {
    background-color: #d70015;
  }

  .btn-warning-confirm {
    height: 28px;
    padding: 0 14px;
    background-color: #ff9500;
    color: #ffffff;
    border: none;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
  }

  :global(.spin) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>