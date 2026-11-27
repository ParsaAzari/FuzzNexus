<script lang="ts">
  import { onMount } from 'svelte';
  import { wordlistsStore } from '../stores/wordlists.svelte';
  import { scanStore } from '../stores/scan.svelte';
  import { toastStore } from '../stores/toast.svelte';
  import Icon from './Icon.svelte';
  import WordlistContextMenu from './WordlistContextMenu.svelte';
  import MergeAndScanModal from './MergeAndScanModal.svelte';
  import type { WordlistDTO, WordlistHealthDTO } from '../types';
  import { SelectWordlistDialog } from '../../../wailsjs/go/main/App';
  import { EventsOn } from '../../../wailsjs/runtime/runtime';

  interface Props {
    onSelectForScan?: (path: string) => void;
  }

  let { onSelectForScan }: Props = $props();

  // Modals state
  let newWordlistModalOpen = $state<boolean>(false);
  let newWordlistMode = $state<'text' | 'import' | 'link'>('text');
  let newWordlistFilePath = $state<string>('');
  let newWordlistName = $state<string>('');
  let initialLines = $state<string>('');
  let newWordlistCategory = $state<string>('general');
  let showInlineCatCreatorNew = $state<boolean>(false);
  let inlineCatNameNew = $state<string>('');
  let inlineCatColorNew = $state<string>('#10b981');
  let isDraggingFile = $state<boolean>(false);
  let modalFileHealth = $state<WordlistHealthDTO | null>(null);
  let isVerifyingFile = $state<boolean>(false);
  let autoDeduplicate = $state<boolean>(true);

  // Real-time duplicate & line analysis for Blank / Custom input
  let textAnalysis = $derived.by(() => {
    if (!initialLines) return { total: 0, duplicates: 0, empty: 0, unique: 0 };
    const lines = initialLines.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n');
    let empty = 0;
    let duplicates = 0;
    const seen = new Set<string>();
    for (const raw of lines) {
      const trimmed = raw.trim();
      if (!trimmed) {
        empty++;
        continue;
      }
      if (seen.has(trimmed)) {
        duplicates++;
      } else {
        seen.add(trimmed);
      }
    }
    return { total: lines.length, duplicates, empty, unique: seen.size };
  });

  let prefixSuffixModalOpen = $state<boolean>(false);
  let prefixInput = $state<string>('');
  let suffixInput = $state<string>('');

  let confirmDeleteModalOpen = $state<boolean>(false);
  let deleteTargets = $state<WordlistDTO[]>([]);

  let unsavedDiscardModalOpen = $state<boolean>(false);
  let pendingPageNavOffset = $state<number | null>(null);

  let shortcutsModalOpen = $state<boolean>(false);

  // Rename & Merge Modals state
  let renameModalOpen = $state<boolean>(false);
  let renameTarget = $state<WordlistDTO | null>(null);
  let renameInput = $state<string>('');
  let renameCategory = $state<string>('general');
  let showInlineCatCreatorRename = $state<boolean>(false);
  let inlineCatNameRename = $state<string>('');
  let inlineCatColorRename = $state<string>('#10b981');

  let mergeModalOpen = $state<boolean>(false);
  let mergeTargetName = $state<string>('');
  let mergeCategory = $state<string>('general');

  let mergeAndScanModalOpen = $state<boolean>(false);
  let mergeAndScanTargets = $state<WordlistDTO[]>([]);

  function openMergeAndScanModal(targets: WordlistDTO[]) {
    if (!targets || targets.length === 0) return;
    mergeAndScanTargets = targets;
    mergeAndScanModalOpen = true;
  }

  // Built-in wordlist factory reset state & actions
  let resetBuiltinModalOpen = $state<boolean>(false);
  let resetBuiltinTarget = $state<WordlistDTO | null>(null);

  function openResetBuiltinModal(w: WordlistDTO) {
    resetBuiltinTarget = w;
    resetBuiltinModalOpen = true;
  }

  async function handleConfirmResetBuiltin() {
    if (!resetBuiltinTarget) return;
    try {
      await wordlistsStore.resetBuiltinWordlist(resetBuiltinTarget.id);
      toastStore.success(`Restored "${resetBuiltinTarget.name}" to factory default`);
      resetBuiltinModalOpen = false;
      resetBuiltinTarget = null;
    } catch (err: any) {
      toastStore.error(err?.message || 'Failed to reset built-in wordlist');
    }
  }

  async function handleRestoreAllBuiltins() {
    try {
      const restored = await wordlistsStore.restoreAllBuiltinWordlists();
      toastStore.success(`Restored ${restored.length} built-in wordlists`);
    } catch (err: any) {
      toastStore.error(err?.message || 'Failed to restore built-in wordlists');
    }
  }

  // Resizable Sidebar state & actions
  let sidebarWidth = $state<number>(344);
  let isResizingSidebar = $state<boolean>(false);

  onMount(() => {
    try {
      const savedWidth = localStorage.getItem('fuzznexus_wordlists_sidebar_w');
      if (savedWidth) {
        const parsed = parseInt(savedWidth, 10);
        if (parsed >= 240 && parsed <= 480) {
          sidebarWidth = parsed;
        }
      }
    } catch {}
  });

  function startSidebarResize(e: MouseEvent) {
    e.preventDefault();
    isResizingSidebar = true;
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    function onMouseMove(ev: MouseEvent) {
      if (!isResizingSidebar) return;
      const delta = ev.clientX - startX;
      const newWidth = Math.min(480, Math.max(240, startWidth + delta));
      sidebarWidth = newWidth;
    }

    function onMouseUp() {
      isResizingSidebar = false;
      try {
        localStorage.setItem('fuzznexus_wordlists_sidebar_w', String(sidebarWidth));
      } catch {}
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    }

    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
  }

  function resetSidebarWidth() {
    sidebarWidth = 300;
    try {
      localStorage.setItem('fuzznexus_wordlists_sidebar_w', '300');
    } catch {}
  }

  // Manage Categories Modal state
  let manageCategoriesModalOpen = $state<boolean>(false);
  let editingCatId = $state<string | null>(null);
  let editCatName = $state<string>('');
  let editCatColor = $state<string>('#10b981');
  let newCatName = $state<string>('');
  let newCatColor = $state<string>('#10b981');
  let catActionError = $state<string>('');

  const paletteColors = [
    '#10b981', // emerald
    '#06b6d4', // cyan
    '#3b82f6', // blue
    '#8b5cf6', // purple
    '#ec4899', // pink
    '#f59e0b', // amber
    '#ef4444', // red
    '#64748b'  // gray
  ];

  // Context Menu state
  let contextMenu = $state<{
    open: boolean;
    x: number;
    y: number;
    wordlist: WordlistDTO | null;
  }>({
    open: false,
    x: 0,
    y: 0,
    wordlist: null
  });

  // Editor Lock / Read-Only mode (default: true for safe viewing)
  let isReadOnly = $state<boolean>(true);

  // Editor & navigation state
  let editorLines = $state<string[]>([]);
  let editorMode = $state<'raw' | 'lines'>('raw');
  let rawTextContent = $state<string>('');
  let rawTextareaRef = $state<HTMLTextAreaElement | null>(null);
  let rawGutterRef = $state<HTMLElement | null>(null);
  let lineInputRefs = $state<Record<number, HTMLInputElement>>({});
  let sortDropdownOpen = $state<boolean>(false);
  let transformDropdownOpen = $state<boolean>(false);
  let moreDropdownOpen = $state<boolean>(false);
  let sidebarCollapsed = $state<boolean>(false);
  let selectedLineIndices = $state<Set<number>>(new Set());
  let isGutterDragging = $state<boolean>(false);
  let dragAnchorIndex = $state<number | null>(null);
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

  const defaultCategoryFilters = [
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

  let dynamicCategories = $derived.by(() => {
    if (wordlistsStore.categories && wordlistsStore.categories.length > 0) {
      return wordlistsStore.categories.map((c) => ({
        id: c.id,
        label: c.name,
        color: c.color,
        isDefault: c.is_default
      }));
    }
    return defaultCategoryFilters.map((c) => ({ ...c, isDefault: true }));
  });

  let categoryFilters = $derived.by(() => {
    return [
      { id: 'all', label: 'All', color: '#64748b', isDefault: true },
      ...dynamicCategories
    ];
  });

  function getCategoryColor(cat: string): string {
    const found = categoryFilters.find((c) => c.id === (cat || 'general').toLowerCase());
    return found?.color || '#64748b';
  }

  function openContextMenu(e: MouseEvent, w: WordlistDTO) {
    e.preventDefault();
    e.stopPropagation();
    contextMenu = {
      open: true,
      x: e.clientX,
      y: e.clientY,
      wordlist: w
    };
  }

  function closeContextMenu() {
    contextMenu = {
      open: false,
      x: 0,
      y: 0,
      wordlist: null
    };
  }

  function openRenameModal(w: WordlistDTO) {
    renameTarget = w;
    renameInput = w.name;
    renameCategory = w.category || 'general';
    renameModalOpen = true;
  }

  async function handleSaveRename() {
    if (!renameTarget || !renameInput.trim()) return;
    try {
      await wordlistsStore.updateMetadata(renameTarget.id, renameInput.trim(), renameCategory);
      toastStore.success(`Updated "${renameInput.trim()}"`);
      renameModalOpen = false;
      renameTarget = null;
    } catch (e: any) {
      toastStore.error(`Rename failed: ${e?.message || e}`);
    }
  }

  function openMergeModal() {
    const count = wordlistsStore.checkedCount;
    if (count < 2) {
      toastStore.warning('Select at least 2 wordlists to merge');
      return;
    }
    mergeTargetName = `merged_${count}_wordlists.txt`;
    mergeCategory = 'general';
    mergeModalOpen = true;
  }

  async function handleConfirmMerge() {
    if (!mergeTargetName.trim()) return;
    try {
      const merged = await wordlistsStore.mergeWordlists(
        wordlistsStore.checkedIDs,
        mergeTargetName.trim(),
        mergeCategory
      );
      toastStore.success(`Merged ${wordlistsStore.checkedCount} wordlists into "${merged.name}" (${merged.line_count.toLocaleString()} unique lines)`);
      wordlistsStore.uncheckAll();
      mergeModalOpen = false;
    } catch (e: any) {
      toastStore.error(`Merge failed: ${e?.message || e}`);
    }
  }

  async function exportWordlistNative(w: WordlistDTO) {
    try {
      const ok = await wordlistsStore.exportNative(w.id, w.name);
      if (ok) {
        toastStore.success(`Exported "${w.name}" successfully`);
      }
    } catch (e: any) {
      toastStore.error(`Export failed: ${e?.message || e}`);
    }
  }

  onMount(() => {
    wordlistsStore.loadWordlists();

    const unbind = EventsOn('wails:file-drop', (paths: string[]) => {
      if (!newWordlistModalOpen) return;
      if (paths && paths.length > 0) {
        handleSelectFilePath(paths[0]);
      }
    });

    return () => {
      if (unbind) unbind();
    };
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

  function setEditorMode(mode: 'raw' | 'lines') {
    if (editorMode === mode) return;
    if (mode === 'raw') {
      rawTextContent = editorLines.join('\n');
      editorMode = 'raw';
    } else {
      editorLines = rawTextContent.split('\n');
      editorMode = 'lines';
      selectedLineIndices = new Set();
    }
  }

  function toggleEditorMode() {
    setEditorMode(editorMode === 'raw' ? 'lines' : 'raw');
  }

  function handleSelectAll() {
    if (editorMode === 'raw') {
      if (rawTextareaRef) {
        rawTextareaRef.focus();
        rawTextareaRef.select();
      }
    } else {
      // Line-by-line mode
      if (selectedLineIndices.size === editorLines.length) {
        selectedLineIndices = new Set();
      } else {
        selectedLineIndices = new Set(editorLines.map((_, i) => i));
      }
    }
  }

  function handleTextareaScroll(e: Event) {
    const target = e.currentTarget as HTMLTextAreaElement;
    if (rawGutterRef) {
      rawGutterRef.scrollTop = target.scrollTop;
    }
  }

  function getLineCharRange(targetIdx: number): { start: number; end: number } {
    let charPos = 0;
    for (let i = 0; i < editorLines.length; i++) {
      const lineLen = editorLines[i].length;
      if (i === targetIdx) {
        return {
          start: charPos,
          end: charPos + lineLen + (i < editorLines.length - 1 ? 1 : 0),
        };
      }
      charPos += lineLen + 1; // +1 for '\n'
    }
    return { start: 0, end: rawTextContent.length };
  }

  function selectLineRangeInTextarea(startLineIdx: number, endLineIdx: number) {
    if (!rawTextareaRef) return;
    const first = Math.min(startLineIdx, endLineIdx);
    const last = Math.max(startLineIdx, endLineIdx);
    const r1 = getLineCharRange(first);
    const r2 = getLineCharRange(last);
    rawTextareaRef.focus();
    rawTextareaRef.setSelectionRange(r1.start, r2.end);
  }

  function handleRawGutterMouseDown(idx: number, e: MouseEvent) {
    if (e.button !== 0) return;
    isGutterDragging = true;
    dragAnchorIndex = idx;
    selectLineRangeInTextarea(idx, idx);
  }

  function handleRawGutterMouseEnter(idx: number) {
    if (!isGutterDragging || dragAnchorIndex === null) return;
    selectLineRangeInTextarea(dragAnchorIndex, idx);
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

  function handleGutterMouseDown(idx: number, e: MouseEvent) {
    if (e.button !== 0) return; // Left mouse button only

    if (e.shiftKey && selectedLineIndices.size > 0) {
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

    isGutterDragging = true;
    dragAnchorIndex = idx;

    if (e.metaKey || e.ctrlKey) {
      const newSet = new Set(selectedLineIndices);
      if (newSet.has(idx)) {
        newSet.delete(idx);
      } else {
        newSet.add(idx);
      }
      selectedLineIndices = newSet;
    } else {
      selectedLineIndices = new Set([idx]);
    }
  }

  function handleGutterMouseEnter(idx: number) {
    if (!isGutterDragging || dragAnchorIndex === null) return;
    const start = Math.min(dragAnchorIndex, idx);
    const end = Math.max(dragAnchorIndex, idx);
    const newSet = new Set<number>();
    for (let i = start; i <= end; i++) {
      newSet.add(i);
    }
    selectedLineIndices = newSet;
  }

  function handleWindowMouseUp() {
    if (isGutterDragging) {
      isGutterDragging = false;
      dragAnchorIndex = null;
    }
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

    // Copy selected lines
    if (isMeta && e.key.toLowerCase() === 'c' && selectedLineIndices.size > 0) {
      e.preventDefault();
      copyAllLines();
      return;
    }

    // Escape clears line selection
    if (e.key === 'Escape' && selectedLineIndices.size > 0) {
      e.preventDefault();
      selectedLineIndices = new Set();
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

  async function handleSelectFilePath(path: string) {
    if (!path) return;
    newWordlistFilePath = path;
    if (!newWordlistName.trim()) {
      const baseName = path.split(/[/\\]/).pop() || '';
      newWordlistName = baseName;
    }
    modalFileHealth = null;
    isVerifyingFile = true;
    try {
      const health = await wordlistsStore.verifyPath(path);
      modalFileHealth = health;
    } catch {
      modalFileHealth = null;
    } finally {
      isVerifyingFile = false;
    }
  }

  function handleClearSelectedFile() {
    newWordlistFilePath = '';
    modalFileHealth = null;
  }

  async function handleCleanSelectedFile() {
    if (!newWordlistFilePath) return;
    try {
      const res = await wordlistsStore.cleanPath(newWordlistFilePath);
      await handleSelectFilePath(newWordlistFilePath);
      toastStore.success(`Cleaned file: removed ${res.duplicates_removed} duplicates, ${res.empty_lines_removed} empty lines`);
    } catch (err: any) {
      toastStore.error(`Clean failed: ${err?.message || err}`);
    }
  }

  function handleCleanTextDuplicates() {
    if (!initialLines) return;
    const lines = initialLines.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n');
    const clean: string[] = [];
    const seen = new Set<string>();
    let dups = 0;
    let empties = 0;
    for (const raw of lines) {
      const trimmed = raw.trim();
      if (!trimmed) {
        empties++;
        continue;
      }
      if (seen.has(trimmed)) {
        dups++;
        continue;
      }
      seen.add(trimmed);
      clean.push(trimmed);
    }
    initialLines = clean.join('\n');
    toastStore.success(`Cleaned ${dups} duplicates and ${empties} empty lines`);
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    if (e.dataTransfer) {
      e.dataTransfer.dropEffect = 'copy';
    }
    isDraggingFile = true;
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    if (e.currentTarget && (e.currentTarget as HTMLElement).contains(e.relatedTarget as Node)) {
      return;
    }
    isDraggingFile = false;
  }

  async function handleDrop(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    isDraggingFile = false;

    const files = e.dataTransfer?.files;
    if (files && files.length > 0) {
      const file = files[0];
      const nativePath = (file as any).path || '';
      if (nativePath) {
        await handleSelectFilePath(nativePath);
        return;
      }

      // If in text mode or dropped directly onto textarea
      if (newWordlistMode === 'text') {
        try {
          const text = await file.text();
          initialLines = text;
          if (!newWordlistName.trim()) {
            newWordlistName = file.name;
          }
          toastStore.success(`Attached "${file.name}" (${textAnalysis.total} lines)`);
        } catch (err: any) {
          toastStore.error(`Failed to read file: ${err?.message || err}`);
        }
      } else {
        if (!newWordlistName.trim()) {
          newWordlistName = file.name;
        }
      }
    }
  }

  async function handlePickFileForModal() {
    try {
      const title =
        newWordlistMode === 'link'
          ? 'Select Wordlist to Link (In-Place)'
          : 'Select Wordlist to Import (Copy)';
      const selected = await SelectWordlistDialog(title);
      if (!selected) return;
      await handleSelectFilePath(selected);
    } catch (err: any) {
      toastStore.error(`File selection failed: ${err?.message || err}`);
    }
  }

  async function handleCreateNew() {
    const name = newWordlistName.trim();
    if (!name) return;

    try {
      if (newWordlistMode === 'text') {
        let content = initialLines;
        if (autoDeduplicate) {
          const lines = initialLines.replace(/\r\n/g, '\n').replace(/\r/g, '\n').split('\n');
          const clean: string[] = [];
          const seen = new Set<string>();
          for (const raw of lines) {
            const trimmed = raw.trim();
            if (!trimmed || seen.has(trimmed)) continue;
            seen.add(trimmed);
            clean.push(trimmed);
          }
          content = clean.join('\n');
        }
        await wordlistsStore.createWordlist(name, content, newWordlistCategory);
        toastStore.success(`Created wordlist "${name}"`);
      } else if (newWordlistMode === 'import') {
        if (!newWordlistFilePath) {
          toastStore.error('Please select a file to import');
          return;
        }
        await wordlistsStore.importFromPath(newWordlistFilePath, name, newWordlistCategory);
        toastStore.success(`Imported wordlist "${name}" (Smart cleaned & deduplicated)`);
      } else if (newWordlistMode === 'link') {
        if (!newWordlistFilePath) {
          toastStore.error('Please select a file to link');
          return;
        }
        await wordlistsStore.linkFromPath(newWordlistFilePath, name, newWordlistCategory);
        if (autoDeduplicate && modalFileHealth && (modalFileHealth.duplicate_count > 0 || modalFileHealth.blank_lines_count > 0)) {
          try {
            const res = await wordlistsStore.cleanPath(newWordlistFilePath);
            toastStore.success(`Linked "${name}" & cleaned ${res.duplicates_removed} duplicates in-place`);
          } catch (err: any) {
            toastStore.info(`Linked "${name}" (in-place clean skipped: ${err?.message || err})`);
          }
        } else {
          toastStore.success(`Linked wordlist "${name}"`);
        }
      }

      newWordlistName = '';
      initialLines = '';
      newWordlistFilePath = '';
      modalFileHealth = null;
      newWordlistMode = 'text';
      newWordlistCategory = 'general';
      showInlineCatCreatorNew = false;
      newWordlistModalOpen = false;
    } catch (e: any) {
      toastStore.error(`Operation failed: ${e?.message || e}`);
    }
  }

  async function handleQuickCreateCategoryNew() {
    const name = inlineCatNameNew.trim();
    if (!name) return;
    try {
      const created = await wordlistsStore.createCategory(name, inlineCatColorNew);
      newWordlistCategory = created.id;
      inlineCatNameNew = '';
      showInlineCatCreatorNew = false;
      toastStore.success(`Category "${created.name}" created & assigned`);
    } catch (err: any) {
      toastStore.error(err?.message || 'Failed to create category');
    }
  }

  async function handleQuickCreateCategoryRename() {
    const name = inlineCatNameRename.trim();
    if (!name) return;
    try {
      const created = await wordlistsStore.createCategory(name, inlineCatColorRename);
      renameCategory = created.id;
      inlineCatNameRename = '';
      showInlineCatCreatorRename = false;
      toastStore.success(`Category "${created.name}" created & assigned`);
    } catch (err: any) {
      toastStore.error(err?.message || 'Failed to create category');
    }
  }

  async function handleAddNewCategory() {
    catActionError = '';
    const name = newCatName.trim();
    if (!name) return;
    try {
      await wordlistsStore.createCategory(name, newCatColor);
      newCatName = '';
      toastStore.success(`Added category "${name}"`);
    } catch (err: any) {
      catActionError = err?.message || 'Failed to create category';
    }
  }

  async function handleSaveEditCategory(id: string) {
    catActionError = '';
    const name = editCatName.trim();
    if (!name) return;
    try {
      await wordlistsStore.updateCategory(id, name, editCatColor);
      editingCatId = null;
      toastStore.success('Category updated successfully');
    } catch (err: any) {
      catActionError = err?.message || 'Failed to update category';
    }
  }

  async function handleDeleteCategory(id: string) {
    catActionError = '';
    try {
      await wordlistsStore.deleteCategory(id);
      toastStore.info('Category deleted, wordlists moved to General');
    } catch (err: any) {
      catActionError = err?.message || 'Failed to delete category';
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
      if (res.empty_lines_removed > 0 || res.duplicates_removed > 0 || res.lines_trimmed > 0) {
        toastStore.success(
          `Cleaned "${selected.name}": removed ${res.duplicates_removed.toLocaleString()} duplicates, ${res.empty_lines_removed.toLocaleString()} blank lines`
        );
      } else {
        toastStore.info(`Wordlist "${selected.name}" is verified clean (0 duplicates, 0 blank lines)`);
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
    scanStore.setWordlist(w.path, w.line_count);
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
    if (resetBuiltinModalOpen) {
      if (e.key === 'Escape') {
        e.preventDefault();
        resetBuiltinModalOpen = false;
        return;
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        handleConfirmResetBuiltin();
        return;
      }
      return;
    }

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

    if (renameModalOpen) {
      if (e.key === 'Escape') {
        e.preventDefault();
        renameModalOpen = false;
        return;
      }
      if (isMeta && e.key === 'Enter') {
        e.preventDefault();
        handleSaveRename();
        return;
      }
      return;
    }

    if (mergeModalOpen) {
      if (e.key === 'Escape') {
        e.preventDefault();
        mergeModalOpen = false;
        return;
      }
      if (isMeta && e.key === 'Enter') {
        e.preventDefault();
        handleConfirmMerge();
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
      newWordlistMode = 'text';
      newWordlistFilePath = '';
      newWordlistModalOpen = true;
      return;
    }

    if (isMeta && e.key.toLowerCase() === 'o') {
      e.preventDefault();
      newWordlistMode = 'import';
      newWordlistModalOpen = true;
      handlePickFileForModal();
      return;
    }

    if (isMeta && e.key.toLowerCase() === 'l') {
      e.preventDefault();
      newWordlistMode = 'link';
      newWordlistModalOpen = true;
      handlePickFileForModal();
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

      // If lines are selected in the editor, keyboard actions target those lines
      if (selectedLineIndices.size > 0) {
        if (e.key === 'Delete' || e.key === 'Backspace') {
          e.preventDefault();
          deleteSelectedLines();
          return;
        }
        if (isMeta && e.key.toLowerCase() === 'c') {
          e.preventDefault();
          copyAllLines();
          return;
        }
        if (e.key === 'Escape') {
          e.preventDefault();
          selectedLineIndices = new Set();
          return;
        }
      }

      if (isMeta && e.key.toLowerCase() === 'a') {
        e.preventDefault();
        if (selected) {
          handleSelectAll();
          return;
        }
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

<svelte:window onkeydown={handleKeyDown} onclick={handleClickOutside} onmouseup={handleWindowMouseUp} />

<div class="wordlist-manager select-none" class:sidebar-is-collapsed={sidebarCollapsed}>
  <!-- Left Column: Wordlist Library -->
  {#if !sidebarCollapsed}
    <aside class="sidebar-library" style="width: {sidebarWidth}px;">
      <div class="sidebar-header">
        <div class="sidebar-title-bar">
          <span class="sidebar-heading">Wordlists</span>
          <div class="sidebar-title-actions">
            <button
              type="button"
              class="btn-icon-toggle"
              onclick={handleRestoreAllBuiltins}
              title="Restore any missing built-in wordlists"
              aria-label="Restore Built-ins"
            >
              <Icon name="rotate-ccw" size={13} />
            </button>
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

      <!-- Category Filter Pills Strip -->
      <div class="category-filter-strip">
        {#each categoryFilters as cat (cat.id)}
          <button
            type="button"
            class="cat-chip"
            class:active={wordlistsStore.selectedCategory === cat.id}
            onclick={() => (wordlistsStore.selectedCategory = cat.id)}
          >
            {#if cat.id !== 'all'}
              <span class="cat-chip-dot" style="background-color: {cat.color};"></span>
            {/if}
            <span class="cat-chip-label">{cat.label}</span>
            <span class="cat-chip-count font-mono">{wordlistsStore.getCategoryCount(cat.id)}</span>
            {#if cat.id !== 'all' && wordlistsStore.getCheckedCountByCategory(cat.id) > 0}
              <span class="cat-chip-selected-badge font-mono" title="{wordlistsStore.getCheckedCountByCategory(cat.id)} selected in {cat.label}">
                [{wordlistsStore.getCheckedCountByCategory(cat.id)}]
              </span>
            {/if}
          </button>
        {/each}
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
            class="btn-batch-scan"
            onclick={() => {
              const targets = wordlistsStore.items.filter((w) => wordlistsStore.checkedIDs.includes(w.id));
              openMergeAndScanModal(targets);
            }}
            title="Sanitize, merge, and use selected wordlist(s) in active scan"
          >
            <Icon name="target" size={11} />
            <span>Scan</span>
          </button>
          {#if wordlistsStore.checkedCount > 1}
            <button
              class="btn-batch-merge"
              onclick={openMergeModal}
              title="Merge and deduplicate selected wordlists into a new list"
            >
              <Icon name="layers" size={11} />
              <span>Merge</span>
            </button>
          {/if}
          <button
            class="btn-batch-delete btn-icon-only"
            onclick={openBatchDeleteModal}
            title="Delete selected wordlists (Delete or Backspace)"
            aria-label="Delete selected wordlists"
          >
            <Icon name="trash" size={11} />
          </button>
          <button
            class="btn-batch-clear btn-icon-only"
            onclick={() => wordlistsStore.uncheckAll()}
            title="Clear all selections across all categories"
            aria-label="Clear all selections"
          >
            <Icon name="x" size={11} />
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
            class:is-primary-target={scanStore.wordlist === w.path}
            onclick={(e) => {
              focusedIndex = idx;
              if (e.metaKey || e.ctrlKey) {
                wordlistsStore.toggleCheck(w.id);
              } else {
                isReadOnly = true;
                wordlistsStore.selectWordlist(w.id);
              }
            }}
            oncontextmenu={(e) => openContextMenu(e, w)}
            onkeydown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                isReadOnly = true;
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
              <div class="item-header-row">
                <span class="cat-indicator-dot" style="background-color: {getCategoryColor(w.category)};" title="Category: {w.category || 'general'}"></span>
                <span class="item-name" title={w.name}>{w.name}</span>
                {#if w.is_builtin}
                  <span class="badge-builtin" title="Curated built-in wordlist">
                    <Icon name="shield-check" size={10} />
                    <span>BUILT-IN</span>
                  </span>
                {/if}
                {#if scanStore.wordlist === w.path}
                  <span class="badge-target" title="Active scan target in ControlBar">
                    <Icon name="target" size={10} />
                    <span>TARGET</span>
                  </span>
                {/if}
              </div>
              <div class="item-tags font-mono">
                <span class="tag-lines">{w.line_count.toLocaleString()} lines</span>
                <span class="tag-divider">·</span>
                <span class="tag-size">{(w.size_bytes / 1024).toFixed(1)} KB</span>
                {#if w.category && w.category.toLowerCase() !== 'general'}
                  <span class="tag-divider">·</span>
                  <span class="tag-category" style="color: {getCategoryColor(w.category)};">
                    {w.category}
                  </span>
                {/if}
                <span class="tag-badge" class:tag-linked={w.is_linked} class:tag-builtin={w.is_builtin}>
                  {w.is_linked ? 'Linked' : (w.is_builtin ? 'Built-in' : 'Local')}
                </span>
              </div>
            </div>
          </div>
        {/each}
      {/if}
    </div>

    <!-- Sticky Frosted-Glass Bottom Actions Section -->
    <div class="sidebar-footer">
      <button
        type="button"
        class="btn-footer-primary"
        onclick={() => {
          newWordlistMode = 'text';
          newWordlistFilePath = '';
          newWordlistModalOpen = true;
        }}
        title="Create or import a wordlist (⌘N)"
      >
        <Icon name="plus" size={14} />
        <span>Add Wordlist</span>
      </button>
      <button
        type="button"
        class="btn-footer-secondary"
        onclick={() => (manageCategoriesModalOpen = true)}
        title="Manage and customize categories"
      >
        <Icon name="tag" size={13} />
        <span>Manage Categories</span>
      </button>
    </div>
  </aside>

  <!-- Resizable Divider Handle -->
  <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
  <div
    class="sidebar-resizer"
    class:is-resizing={isResizingSidebar}
    onmousedown={startSidebarResize}
    ondblclick={resetSidebarWidth}
    role="separator"
    aria-orientation="vertical"
    aria-label="Resize library sidebar"
    tabindex="0"
    title="Drag to resize sidebar, double-click to reset"
  >
    <span class="resizer-handle"></span>
  </div>
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
            <span class="editor-title" title={selected.name}>{selected.name}</span>
            {#if selected.is_builtin}
              <span class="badge-builtin-pill" title="Curated built-in wordlist">
                <Icon name="shield-check" size={11} />
                <span>BUILT-IN</span>
              </span>
            {:else}
              <span class="type-pill" class:type-linked={selected.is_linked}>
                {selected.is_linked ? 'Linked' : 'Managed'}
              </span>
            {/if}
            {#if selected.category && selected.category.toLowerCase() !== 'general'}
              <span class="cat-pill" style="border-color: {getCategoryColor(selected.category)}; color: {getCategoryColor(selected.category)};">
                {selected.category}
              </span>
            {/if}
          </div>
        </div>

        <div class="header-right">
          {#if selected.is_builtin}
            <button
              type="button"
              class="btn-reset-default"
              onclick={() => openResetBuiltinModal(selected)}
              title="Reset this built-in wordlist back to original factory keywords"
            >
              <Icon name="rotate-ccw" size={13} />
              <span class="btn-text-full">Reset to Default</span>
              <span class="btn-text-short">Reset</span>
            </button>
          {/if}
          <button
            type="button"
            class="btn-lock-toggle"
            class:is-unlocked={!isReadOnly}
            onclick={() => {
              isReadOnly = !isReadOnly;
              toastStore.info(isReadOnly ? 'Wordlist locked (View Mode)' : 'Wordlist unlocked for editing');
            }}
            title={isReadOnly ? 'View Mode active. Click to unlock and edit lines directly.' : 'Edit Mode active. Click to lock against accidental edits.'}
          >
            <Icon name={isReadOnly ? 'lock' : 'unlock'} size={13} />
            <span class="btn-text-full">{isReadOnly ? 'View Mode' : 'Editing'}</span>
            <span class="btn-text-short">{isReadOnly ? 'View' : 'Edit'}</span>
          </button>
          <button
            class="btn-use-scan"
            class:is-active-target={scanStore.wordlist === selected.path}
            onclick={() => handleSelectForScan(selected)}
            title="Set this wordlist as target for FUZZ in Control Bar"
          >
            <Icon name="target" size={13} />
            <span class="btn-text-full">{scanStore.wordlist === selected.path ? 'Active Target' : 'Use in Scan'}</span>
            <span class="btn-text-short">{scanStore.wordlist === selected.path ? 'Active' : 'Scan'}</span>
          </button>
          <button
            class="btn-header-action"
            onclick={() => (shortcutsModalOpen = true)}
            title="Keyboard Shortcuts Cheatsheet (?)"
            aria-label="Keyboard Shortcuts"
          >
            <Icon name="info" size={13} />
          </button>
          <button
            class="btn-header-action"
            onclick={() => openRenameModal(selected)}
            title="Rename or change category (Cmd+R)"
            aria-label="Rename or change category"
          >
            <Icon name="edit" size={13} />
          </button>
          <button
            class="btn-header-action btn-header-danger"
            onclick={() => openDeleteModal([selected])}
            title="Delete wordlist (Delete)"
            aria-label="Delete wordlist"
          >
            <Icon name="trash" size={13} />
          </button>
        </div>
      </header>

      <!-- KPI Statistics Strip & Path Bar -->
      <div class="stats-strip font-mono">
        <div class="stats-left">
          <div class="stat-item">
            <span class="stat-label">Lines</span>
            <span class="stat-value">{selected.line_count.toLocaleString()}</span>
          </div>
          <span class="stat-sep">/</span>
          <div class="stat-item">
            <span class="stat-label">Size</span>
            <span class="stat-value">{(selected.size_bytes / 1024).toFixed(1)} KB</span>
          </div>
          {#if stats}
            <span class="stat-sep">/</span>
            <div class="stat-item">
              <span class="stat-label">Unique</span>
              <span class="stat-value">{stats.unique_lines.toLocaleString()}</span>
            </div>
            <span class="stat-sep">/</span>
            <div class="stat-item">
              <span class="stat-label">Avg</span>
              <span class="stat-value">{stats.avg_line_len.toFixed(1)}</span>
            </div>
            <span class="stat-sep">/</span>
            <div class="stat-item">
              <span class="stat-label">Max</span>
              <span class="stat-value">{stats.max_line_len}</span>
            </div>
          {/if}
        </div>

        <div class="path-display" title={selected.path}>
          <span class="path-text font-mono">{selected.path}</span>
          <button
            type="button"
            class="btn-copy-path"
            onclick={() => {
              navigator.clipboard?.writeText(selected.path);
              toastStore.success('Path copied to clipboard');
            }}
            title="Copy full path to clipboard"
            aria-label="Copy wordlist path"
          >
            <Icon name="copy" size={11} />
          </button>
        </div>
      </div>

      <!-- Health Warning Strip (when blank lines or duplicates detected) -->
      {#if health && !health.is_clean && (health.blank_lines_count > 0 || (health.duplicate_count && health.duplicate_count > 0))}
        <div class="health-warning-strip font-mono">
          <div class="health-warning-left">
            <Icon name="alert-triangle" size={13} color="#ff9500" />
            <span>
              Wordlist contains
              {#if health.duplicate_count && health.duplicate_count > 0}
                {health.duplicate_count.toLocaleString()} duplicate entries
              {/if}
              {#if health.duplicate_count && health.duplicate_count > 0 && health.blank_lines_count > 0}
                and
              {/if}
              {#if health.blank_lines_count > 0}
                {health.blank_lines_count.toLocaleString()} blank lines
              {/if}
              that may impair fuzzing efficiency
            </span>
          </div>
          <button class="btn-clean-pill" onclick={handleCleanWordlist} title="Remove duplicates, blank lines and trim whitespace (Shift+Cmd+C)">
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
              class:active={editorMode === 'raw'}
              onclick={() => setEditorMode('raw')}
              title="Full multiline text editor with drag-select & Cmd+A (Cmd+E)"
            >
              <Icon name="file-text" size={11} />
              <span>Editor</span>
            </button>
            <button
              type="button"
              class="segment-btn"
              class:active={editorMode === 'lines'}
              onclick={() => setEditorMode('lines')}
              title="Line-by-line structured grid (Cmd+E)"
            >
              <Icon name="layers" size={11} />
              <span>Rows</span>
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
                class="tool-btn"
                onclick={handleCleanWordlist}
                title="Verify and auto-clean blank lines (Shift+Cmd+C)"
              >
                <Icon name="filter" size={11} />
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
                onmousedown={(e) => handleGutterMouseDown(idx, e)}
                onmouseenter={() => handleGutterMouseEnter(idx)}
                onclick={(e) => toggleLineSelect(idx, e)}
                title="Click or drag to select lines (Shift+Click for range, Cmd+Click to toggle)"
              >
                {pageOffset + idx + 1}
              </button>
            {/each}
          </div>
          <div class="lines-content font-mono select-text">
            {#each editorLines as line, idx}
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div
                class="line-row"
                class:line-row-selected={selectedLineIndices.has(idx)}
                onmousedown={(e) => {
                  if ((e.target as HTMLElement)?.tagName !== 'INPUT') {
                    handleGutterMouseDown(idx, e);
                  }
                }}
                onmouseenter={() => {
                  if (isGutterDragging) {
                    handleGutterMouseEnter(idx);
                  }
                }}
              >
                <input
                  bind:this={lineInputRefs[idx]}
                  type="text"
                  class="line-input"
                  class:is-read-only={isReadOnly}
                  readonly={isReadOnly}
                  value={line}
                  oninput={(e) => !isReadOnly && handleLineChange(idx, (e.currentTarget as HTMLInputElement).value)}
                  onkeydown={(e) => !isReadOnly && handleLineInputKeyDown(e, idx)}
                  onpaste={(e) => !isReadOnly && handleLinePaste(e, idx)}
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
          <div bind:this={rawGutterRef} class="lines-gutter font-mono select-none">
            {#each editorLines as _, idx}
              <button
                type="button"
                class="gutter-number"
                onmousedown={(e) => handleRawGutterMouseDown(idx, e)}
                onmouseenter={() => handleRawGutterMouseEnter(idx)}
                onclick={() => selectLineRangeInTextarea(idx, idx)}
                title="Click or drag to select line range"
              >
                {pageOffset + idx + 1}
              </button>
            {/each}
          </div>
          <div class="raw-textarea-wrap">
            <textarea
              bind:this={rawTextareaRef}
              class="raw-textarea font-mono select-text"
              class:is-read-only={isReadOnly}
              readonly={isReadOnly}
              value={rawTextContent}
              oninput={(e) => !isReadOnly && handleRawTextChange((e.currentTarget as HTMLTextAreaElement).value)}
              onscroll={handleTextareaScroll}
              placeholder={isReadOnly ? "Wordlist is in View Mode. Click 'Unlock to Edit' to modify." : "Enter wordlist payloads, one per line..."}
              spellcheck="false"
            ></textarea>
          </div>
        </div>
      {/if}

      <!-- Editor Footer Action Bar -->
      <footer class="editor-footer">
        <div class="footer-left">
          {#if isReadOnly}
            <span class="viewmode-badge font-mono">
              <Icon name="lock" size={12} />
              <span>View Mode (Locked)</span>
            </span>
          {:else if isDirty}
            <span class="dirty-badge font-mono">Unsaved Changes (⌘S to save)</span>
          {/if}
        </div>
        <div class="footer-right">
          {#if isReadOnly}
            <button
              class="btn-footer-unlock"
              onclick={() => {
                isReadOnly = false;
                toastStore.info('Wordlist unlocked for editing');
              }}
              title="Unlock to edit wordlist lines"
            >
              <Icon name="unlock" size={14} />
              <span>Unlock to Edit</span>
            </button>
          {:else}
            <button
              class="btn-footer-save"
              onclick={handleSave}
              disabled={!isDirty || wordlistsStore.isSaving}
              title="Save changes (⌘S)"
            >
              <Icon name="save" size={14} />
              <span>{wordlistsStore.isSaving ? 'Saving...' : 'Save Wordlist (⌘S)'}</span>
            </button>
          {/if}
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
            Managed local files are permanently deleted from disk. Linked wordlists are removed from FuzzNexus without deleting the original source file.
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

<!-- Modal: Create New Wordlist (Unified: Custom, Import, Link) -->
{#if newWordlistModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) newWordlistModalOpen = false; }}
    onkeydown={(e) => {
      if (e.key === 'Escape') newWordlistModalOpen = false;
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') handleCreateNew();
    }}
  >
    <div class="modal-dialog" role="document">
      <div class="modal-head">
        <div class="modal-head-title">
          <Icon name={newWordlistMode === 'text' ? 'plus' : newWordlistMode === 'import' ? 'upload' : 'folder'} size={15} color="var(--accent)" />
          <span>{newWordlistMode === 'text' ? 'Create New Wordlist' : newWordlistMode === 'import' ? 'Import Wordlist File' : 'Link External Wordlist'}</span>
        </div>
        <button class="btn-close-modal" onclick={() => (newWordlistModalOpen = false)} aria-label="Close">
          <Icon name="x" size={16} />
        </button>
      </div>
      <div class="modal-body-content">
        <!-- Unified Source Switcher -->
        <div class="modal-mode-tabs">
          <button
            type="button"
            class="modal-mode-tab"
            class:active={newWordlistMode === 'text'}
            onclick={() => (newWordlistMode = 'text')}
          >
            <Icon name="file-text" size={13} />
            <span>Blank / Custom</span>
          </button>
          <button
            type="button"
            class="modal-mode-tab"
            class:active={newWordlistMode === 'import'}
            onclick={() => (newWordlistMode = 'import')}
          >
            <Icon name="upload" size={13} />
            <span>Import File</span>
          </button>
          <button
            type="button"
            class="modal-mode-tab"
            class:active={newWordlistMode === 'link'}
            onclick={() => (newWordlistMode = 'link')}
          >
            <Icon name="folder" size={13} />
            <span>Link File</span>
          </button>
        </div>

        {#if newWordlistMode === 'import' || newWordlistMode === 'link'}
          <div class="file-picker-section">
            <div class="file-picker-header">
              <span class="modal-label-text">
                {newWordlistMode === 'import' ? 'File to import (copied into app storage):' : 'File to link (read in-place):'}
              </span>
              {#if modalFileHealth}
                <div class="file-health-quick-badge">
                  {#if modalFileHealth.duplicate_count > 0}
                    <span class="badge-dup-warn font-mono">
                      <Icon name="alert-triangle" size={11} color="#f59e0b" />
                      <span>{modalFileHealth.duplicate_count.toLocaleString()} duplicates</span>
                    </span>
                  {:else}
                    <span class="badge-dup-clean font-mono">
                      <Icon name="check" size={11} color="#10b981" />
                      <span>0 duplicates</span>
                    </span>
                  {/if}
                </div>
              {/if}
            </div>

            {#if newWordlistFilePath}
              <div class="selected-file-card">
                <div class="file-card-info">
                  <div class="file-card-icon-wrap">
                    <Icon name={newWordlistMode === 'import' ? 'upload' : 'folder'} size={16} color="var(--accent)" />
                  </div>
                  <div class="file-card-details">
                    <span class="file-name font-mono">{newWordlistFilePath.split(/[/\\]/).pop()}</span>
                    <span class="file-path font-mono" title={newWordlistFilePath}>{newWordlistFilePath}</span>
                    {#if modalFileHealth}
                      <div class="file-card-meta font-mono">
                        <span class="meta-item">{modalFileHealth.total_lines.toLocaleString()} lines</span>
                        <span class="meta-sep">•</span>
                        {#if modalFileHealth.duplicate_count > 0}
                          <span class="meta-item text-warn">
                            <Icon name="alert-triangle" size={10} color="#f59e0b" />
                            <span>{modalFileHealth.duplicate_count.toLocaleString()} duplicate{modalFileHealth.duplicate_count > 1 ? 's' : ''}</span>
                          </span>
                        {:else}
                          <span class="meta-item text-success">
                            <Icon name="check" size={10} color="#10b981" />
                            <span>0 duplicates</span>
                          </span>
                        {/if}
                        {#if modalFileHealth.blank_lines_count > 0}
                          <span class="meta-sep">•</span>
                          <span class="meta-item text-muted">{modalFileHealth.blank_lines_count} empty</span>
                        {/if}
                      </div>
                    {:else if isVerifyingFile}
                      <div class="file-card-meta font-mono text-muted">
                        <span>Analyzing wordlist health...</span>
                      </div>
                    {/if}
                  </div>
                </div>
                <div class="file-card-actions">
                  {#if modalFileHealth && modalFileHealth.duplicate_count > 0 && newWordlistMode === 'link'}
                    <button
                      type="button"
                      class="btn-clean-file-now"
                      onclick={handleCleanSelectedFile}
                      title="Clean duplicate lines and blank entries in-place"
                    >
                      <Icon name="sliders" size={11} />
                      <span>Clean Now</span>
                    </button>
                  {/if}
                  <button type="button" class="btn-change-file" onclick={handlePickFileForModal} title="Choose different file">
                    Change...
                  </button>
                  <button type="button" class="btn-clear-file" onclick={handleClearSelectedFile} title="Remove file selection" aria-label="Clear selection">
                    <Icon name="x" size={12} />
                  </button>
                </div>
              </div>
            {:else}
              <div
                class="file-dropzone"
                class:dragging={isDraggingFile}
                role="button"
                tabindex="0"
                onclick={handlePickFileForModal}
                onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && handlePickFileForModal()}
                ondragover={handleDragOver}
                ondragenter={handleDragOver}
                ondragleave={handleDragLeave}
                ondrop={handleDrop}
              >
                <div class="dropzone-icon-circle" class:bounce={isDraggingFile}>
                  <Icon name={newWordlistMode === 'import' ? 'upload' : 'folder'} size={20} color="var(--accent)" />
                </div>
                <div class="dropzone-text-group">
                  <span class="dropzone-title">
                    {isDraggingFile ? 'Release to attach wordlist file' : 'Drop wordlist file here, or click to browse'}
                  </span>
                  <span class="dropzone-subtitle">
                    {newWordlistMode === 'import'
                      ? 'Copies .txt, .lst, .wordlist into managed app storage'
                      : 'Links file directly in-place without duplicating storage'}
                  </span>
                </div>
                <div class="dropzone-chips">
                  <span class="dropzone-tag">UTF-8 Text</span>
                  <span class="dropzone-tag">Case Preserved</span>
                  <span class="dropzone-tag">.txt, .lst, .wordlist</span>
                </div>
              </div>
            {/if}
          </div>
        {/if}

        <label class="modal-label">
          Wordlist Name:
          <input
            type="text"
            placeholder="e.g. custom_routes.txt"
            bind:value={newWordlistName}
            class="modal-input font-mono"
          />
        </label>

        <div class="modal-field-block">
          <div class="modal-field-header">
            <span class="modal-label-text">Category:</span>
            <button
              type="button"
              class="btn-text-action"
              onclick={() => (showInlineCatCreatorNew = !showInlineCatCreatorNew)}
            >
              <Icon name={showInlineCatCreatorNew ? 'x' : 'plus'} size={12} />
              <span>{showInlineCatCreatorNew ? 'Cancel' : 'New Category'}</span>
            </button>
          </div>

          {#if showInlineCatCreatorNew}
            <div class="inline-cat-creator">
              <input
                type="text"
                placeholder="Category name..."
                bind:value={inlineCatNameNew}
                class="modal-input font-mono inline-cat-input"
                onkeydown={(e) => e.key === 'Enter' && handleQuickCreateCategoryNew()}
              />
              <div class="color-swatches">
                {#each paletteColors as color}
                  <button
                    type="button"
                    class="swatch-btn"
                    class:selected={inlineCatColorNew === color}
                    style="background-color: {color};"
                    onclick={() => (inlineCatColorNew = color)}
                    title={color}
                    aria-label="Pick color {color}"
                  ></button>
                {/each}
              </div>
              <button
                type="button"
                class="btn-inline-add"
                disabled={!inlineCatNameNew.trim()}
                onclick={handleQuickCreateCategoryNew}
              >
                Add & Assign
              </button>
            </div>
          {:else}
            <select bind:value={newWordlistCategory} class="modal-select font-mono">
              {#each dynamicCategories as cat (cat.id)}
                <option value={cat.id}>{cat.label}</option>
              {/each}
            </select>
          {/if}
        </div>

        {#if newWordlistMode === 'text'}
          <div class="modal-field-block">
            <div class="modal-field-header">
              <span class="modal-label-text">Initial Lines (one per line):</span>
              <div class="text-stats-bar font-mono">
                <span class="stats-counter">{textAnalysis.total.toLocaleString()} lines</span>
                {#if textAnalysis.duplicates > 0}
                  <span class="dup-alert-pill">
                    <Icon name="alert-triangle" size={11} color="#f59e0b" />
                    <span>{textAnalysis.duplicates.toLocaleString()} duplicates</span>
                  </span>
                  <button
                    type="button"
                    class="btn-clean-text-dups"
                    onclick={handleCleanTextDuplicates}
                    title="Remove duplicates and empty lines while preserving line order and case"
                  >
                    Clean Duplicates
                  </button>
                {:else if textAnalysis.total > 0}
                  <span class="clean-check-pill">
                    <Icon name="check" size={11} color="#10b981" />
                    <span>0 duplicates</span>
                  </span>
                {/if}
              </div>
            </div>
            <textarea
              rows="6"
              placeholder="admin&#10;api&#10;login&#10;v1&#10;v2&#10;Tip: You can also drag & drop a text file here"
              bind:value={initialLines}
              class="modal-textarea font-mono"
              ondragover={handleDragOver}
              ondragenter={handleDragOver}
              ondragleave={handleDragLeave}
              ondrop={handleDrop}
            ></textarea>
          </div>
        {/if}

        <!-- Smart Deduplication Option (All 3 Modes) -->
        <label class="modal-checkbox-row select-none">
          <input
            type="checkbox"
            bind:checked={autoDeduplicate}
            class="checkbox-input"
          />
          <div class="checkbox-text-group">
            <div class="checkbox-title-row">
              <span class="checkbox-title">Remove duplicates & empty lines (Smart Deduplication)</span>
              <span class="checkbox-badge">Recommended</span>
            </div>
            <span class="checkbox-desc">
              Preserves exact case sensitivity (/admin ≠ /Admin) and payload syntax while eliminating repeated entries and empty lines.
            </span>
          </div>
        </label>
      </div>
      <div class="modal-foot">
        <span class="shortcut-tip font-mono">Cmd+Enter to submit</span>
        <button class="btn-cancel" onclick={() => (newWordlistModalOpen = false)}>Cancel</button>
        <button
          class="btn-confirm"
          onclick={handleCreateNew}
          disabled={!newWordlistName.trim() || ((newWordlistMode === 'import' || newWordlistMode === 'link') && !newWordlistFilePath)}
        >
          {newWordlistMode === 'text' ? 'Create Wordlist' : newWordlistMode === 'import' ? 'Import Wordlist' : 'Link Wordlist'}
        </button>
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

<!-- Modal: Rename Wordlist & Category -->
{#if renameModalOpen && renameTarget}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) renameModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (renameModalOpen = false)}
  >
    <div class="modal-dialog" role="document">
      <div class="modal-head">
        <span>Edit Wordlist Metadata</span>
        <button class="btn-close-modal" onclick={() => (renameModalOpen = false)} aria-label="Close">
          <Icon name="x" size={16} />
        </button>
      </div>
      <div class="modal-body-content">
        <label class="modal-label">
          Wordlist Name:
          <input
            type="text"
            bind:value={renameInput}
            class="modal-input font-mono"
            placeholder="e.g. routes.txt"
          />
        </label>
        <div class="modal-field-block">
          <div class="modal-field-header">
            <span class="modal-label-text">Category:</span>
            <button
              type="button"
              class="btn-text-action"
              onclick={() => (showInlineCatCreatorRename = !showInlineCatCreatorRename)}
            >
              <Icon name={showInlineCatCreatorRename ? 'x' : 'plus'} size={12} />
              <span>{showInlineCatCreatorRename ? 'Cancel' : 'New Category'}</span>
            </button>
          </div>

          {#if showInlineCatCreatorRename}
            <div class="inline-cat-creator">
              <input
                type="text"
                placeholder="Category name..."
                bind:value={inlineCatNameRename}
                class="modal-input font-mono inline-cat-input"
                onkeydown={(e) => e.key === 'Enter' && handleQuickCreateCategoryRename()}
              />
              <div class="color-swatches">
                {#each paletteColors as color}
                  <button
                    type="button"
                    class="swatch-btn"
                    class:selected={inlineCatColorRename === color}
                    style="background-color: {color};"
                    onclick={() => (inlineCatColorRename = color)}
                    title={color}
                    aria-label="Pick color {color}"
                  ></button>
                {/each}
              </div>
              <button
                type="button"
                class="btn-inline-add"
                disabled={!inlineCatNameRename.trim()}
                onclick={handleQuickCreateCategoryRename}
              >
                Add & Assign
              </button>
            </div>
          {:else}
            <select bind:value={renameCategory} class="modal-select font-mono">
              {#each dynamicCategories as cat (cat.id)}
                <option value={cat.id}>{cat.label}</option>
              {/each}
            </select>
          {/if}
        </div>
      </div>
      <div class="modal-foot">
        <span class="shortcut-tip font-mono">Cmd+Enter to save</span>
        <button class="btn-cancel" onclick={() => (renameModalOpen = false)}>Cancel</button>
        <button class="btn-confirm" onclick={handleSaveRename} disabled={!renameInput.trim()}>Save Changes</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Merge Wordlists -->
{#if mergeModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) mergeModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (mergeModalOpen = false)}
  >
    <div class="modal-dialog" role="document">
      <div class="modal-head">
        <div class="modal-head-title">
          <Icon name="layers" size={16} color="var(--accent)" />
          <span>Merge {wordlistsStore.checkedCount} Wordlists</span>
        </div>
        <button class="btn-close-modal" onclick={() => (mergeModalOpen = false)} aria-label="Close">
          <Icon name="x" size={16} />
        </button>
      </div>
      <div class="modal-body-content">
        <p class="dialog-desc">
          Combine all selected wordlists into a single, deduplicated wordlist file. Empty lines and carriage returns are automatically filtered.
        </p>
        <label class="modal-label">
          New Merged File Name:
          <input
            type="text"
            bind:value={mergeTargetName}
            class="modal-input font-mono"
            placeholder="e.g. merged_suite.txt"
          />
        </label>
        <label class="modal-label">
          Assign Category:
          <select bind:value={mergeCategory} class="modal-input font-mono">
            {#each categoryFilters.filter(c => c.id !== 'all') as cat (cat.id)}
              <option value={cat.id}>{cat.label}</option>
            {/each}
          </select>
        </label>
      </div>
      <div class="modal-foot">
        <span class="shortcut-tip font-mono">Cmd+Enter to merge</span>
        <button class="btn-cancel" onclick={() => (mergeModalOpen = false)}>Cancel</button>
        <button class="btn-confirm" onclick={handleConfirmMerge} disabled={!mergeTargetName.trim() || wordlistsStore.isSaving}>
          {wordlistsStore.isSaving ? 'Merging...' : 'Merge & Create'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Manage Categories -->
{#if manageCategoriesModalOpen}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) manageCategoriesModalOpen = false; }}
    onkeydown={(e) => e.key === 'Escape' && (manageCategoriesModalOpen = false)}
  >
    <div class="modal-dialog modal-dialog-wide" role="document">
      <div class="modal-head">
        <div class="modal-head-title">
          <Icon name="tag" size={16} color="var(--accent)" />
          <span>Manage Wordlist Categories</span>
        </div>
        <button class="btn-close-modal" onclick={() => (manageCategoriesModalOpen = false)} aria-label="Close">
          <Icon name="x" size={16} />
        </button>
      </div>

      <div class="modal-body-content">
        {#if catActionError}
          <div class="cat-error-alert">
            <Icon name="alert-triangle" size={14} />
            <span>{catActionError}</span>
          </div>
        {/if}

        <div class="cat-list-section">
          <div class="section-title">Existing Categories</div>
          <div class="cat-table-wrap">
            <table class="cat-table">
              <thead>
                <tr>
                  <th style="width: 40px;">Color</th>
                  <th>Category Name</th>
                  <th style="width: 80px;">Wordlists</th>
                  <th style="width: 90px;">Type</th>
                  <th style="width: 90px; text-align: right;">Actions</th>
                </tr>
              </thead>
              <tbody>
                {#each dynamicCategories as cat (cat.id)}
                  <tr>
                    <td>
                      {#if editingCatId === cat.id}
                        <div class="swatches-inline">
                          {#each paletteColors as color}
                            <button
                              type="button"
                              class="swatch-mini"
                              class:selected={editCatColor === color}
                              style="background-color: {color};"
                              onclick={() => (editCatColor = color)}
                              title={color}
                              aria-label="Color {color}"
                            ></button>
                          {/each}
                        </div>
                      {:else}
                        <span class="cat-dot-cell" style="background-color: {cat.color};"></span>
                      {/if}
                    </td>
                    <td>
                      {#if editingCatId === cat.id}
                        <input
                          type="text"
                          class="modal-input font-mono cat-edit-input"
                          bind:value={editCatName}
                          onkeydown={(e) => e.key === 'Enter' && handleSaveEditCategory(cat.id)}
                        />
                      {:else}
                        <span class="cat-name-cell">{cat.label}</span>
                      {/if}
                    </td>
                    <td class="font-mono">{wordlistsStore.getCategoryCount(cat.id)}</td>
                    <td>
                      {#if cat.isDefault}
                        <span class="cat-badge-default">Default</span>
                      {:else}
                        <span class="cat-badge-custom">Custom</span>
                      {/if}
                    </td>
                    <td style="text-align: right;">
                      {#if editingCatId === cat.id}
                        <div class="action-btn-group">
                          <button
                            type="button"
                            class="btn-icon-action save"
                            title="Save"
                            onclick={() => handleSaveEditCategory(cat.id)}
                          >
                            <Icon name="check" size={13} />
                          </button>
                          <button
                            type="button"
                            class="btn-icon-action cancel"
                            title="Cancel"
                            onclick={() => (editingCatId = null)}
                          >
                            <Icon name="x" size={13} />
                          </button>
                        </div>
                      {:else}
                        <div class="action-btn-group">
                          <button
                            type="button"
                            class="btn-icon-action edit"
                            title="Edit Category"
                            onclick={() => {
                              editingCatId = cat.id;
                              editCatName = cat.label;
                              editCatColor = cat.color;
                            }}
                          >
                            <Icon name="edit" size={13} />
                          </button>
                          {#if !cat.isDefault && cat.id !== 'general'}
                            <button
                              type="button"
                              class="btn-icon-action delete"
                              title="Delete Category (reassigns wordlists to General)"
                              onclick={() => handleDeleteCategory(cat.id)}
                            >
                              <Icon name="trash" size={13} />
                            </button>
                          {/if}
                        </div>
                      {/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>

        <!-- Add Category Block -->
        <div class="cat-create-section">
          <div class="section-title">Create New Category</div>
          <div class="cat-create-form">
            <div class="cat-create-inputs">
              <input
                type="text"
                placeholder="Category name (e.g. Subdomains, API v2)..."
                bind:value={newCatName}
                class="modal-input font-mono cat-new-input"
                onkeydown={(e) => e.key === 'Enter' && handleAddNewCategory()}
              />
              <div class="color-swatches-strip">
                {#each paletteColors as color}
                  <button
                    type="button"
                    class="swatch-btn"
                    class:selected={newCatColor === color}
                    style="background-color: {color};"
                    onclick={() => (newCatColor = color)}
                    title={color}
                    aria-label="Pick color {color}"
                  ></button>
                {/each}
              </div>
            </div>
            <button
              type="button"
              class="btn-confirm btn-add-cat"
              disabled={!newCatName.trim()}
              onclick={handleAddNewCategory}
            >
              <Icon name="plus" size={13} />
              <span>Add Category</span>
            </button>
          </div>
        </div>
      </div>

      <div class="modal-foot">
        <button class="btn-cancel" onclick={() => (manageCategoriesModalOpen = false)}>Close</button>
      </div>
    </div>
  </div>
{/if}

<!-- Modal: Confirm Reset Built-in Wordlist -->
{#if resetBuiltinModalOpen && resetBuiltinTarget}
  <div
    class="modal-backdrop select-none"
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) resetBuiltinModalOpen = false; }}
    onkeydown={(e) => {
      if (e.key === 'Escape') resetBuiltinModalOpen = false;
      if (e.key === 'Enter') handleConfirmResetBuiltin();
    }}
  >
    <div class="modal-dialog modal-dialog-delete" role="document">
      <div class="modal-head">
        <div class="modal-head-title">
          <Icon name="rotate-ccw" size={16} color="#6366f1" />
          <span>Reset to Factory Default</span>
        </div>
        <button class="btn-close-modal" onclick={() => (resetBuiltinModalOpen = false)} aria-label="Close">
          <Icon name="x" size={16} />
        </button>
      </div>

      <div class="modal-body-content">
        <div class="delete-warning-text">
          Are you sure you want to revert <strong class="reset-target-name font-mono">{resetBuiltinTarget.name}</strong> back to factory defaults?
        </div>

        <div class="delete-target-list font-mono">
          <div class="target-row">
            <Icon name="shield-check" size={13} color="#6366f1" />
            <span class="target-filename">{resetBuiltinTarget.name}</span>
            <span class="target-kind-pill" style="border-color: rgba(99, 102, 241, 0.4); color: #818cf8;">
              Built-in
            </span>
          </div>
        </div>

        <div class="delete-impact-notice">
          <Icon name="info" size={14} color="var(--text-secondary)" />
          <span>
            Any manual modifications or custom lines you added to this wordlist will be replaced with the original curated high-precision keyword set.
          </span>
        </div>
      </div>

      <div class="modal-foot">
        <span class="shortcut-tip font-mono">Esc to cancel · Enter to reset</span>
        <button class="btn-cancel" onclick={() => (resetBuiltinModalOpen = false)}>Cancel</button>
        <button class="btn-primary-confirm" onclick={handleConfirmResetBuiltin}>
          <Icon name="rotate-ccw" size={13} color="#ffffff" />
          <span>Reset Wordlist</span>
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Floating Context Menu -->
{#if contextMenu.open && contextMenu.wordlist}
  <WordlistContextMenu
    x={contextMenu.x}
    y={contextMenu.y}
    wordlist={contextMenu.wordlist}
    isPrimaryTarget={scanStore.wordlist === contextMenu.wordlist.path}
    {isReadOnly}
    categories={categoryFilters}
    onSetPrimary={(w) => {
      handleSelectForScan(w);
    }}
    onRename={(w) => {
      openRenameModal(w);
    }}
    onChangeCategory={async (w, cat) => {
      try {
        await wordlistsStore.updateMetadata(w.id, w.name, cat);
        toastStore.success(`Updated category to ${cat.toUpperCase()}`);
      } catch (err: any) {
        toastStore.error(`Failed to update category: ${err?.message || err}`);
      }
    }}
    onToggleReadOnly={() => {
      isReadOnly = !isReadOnly;
      toastStore.info(isReadOnly ? 'Wordlist locked (View Mode)' : 'Wordlist unlocked for editing');
    }}
    onExport={(w) => {
      exportWordlistNative(w);
    }}
    onDuplicate={async (w) => {
      try {
        await wordlistsStore.importFromPath(w.path);
        toastStore.success(`Duplicated "${w.name}"`);
      } catch (err: any) {
        toastStore.error(`Failed to duplicate: ${err?.message || err}`);
      }
    }}
    onResetBuiltin={(w) => {
      openResetBuiltinModal(w);
    }}
    onDelete={(w) => {
      openDeleteModal([w]);
    }}
    onClose={closeContextMenu}
  />
{/if}

<!-- Modal: Merge & Use in Scan (Sanitization & Sandbox) -->
<MergeAndScanModal
  isOpen={mergeAndScanModalOpen}
  selectedWordlists={mergeAndScanTargets}
  onClose={() => (mergeAndScanModalOpen = false)}
  onConfirm={(targetWordlist, autoStart) => {
    scanStore.setWordlist(targetWordlist.path, targetWordlist.line_count);
    onSelectForScan?.(targetWordlist.path);
    if (autoStart) {
      scanStore.start();
    }
  }}
/>

<style>
  .wordlist-manager {
    display: flex;
    flex: 1;
    height: 100%;
    min-height: 0;
    background-color: var(--surface);
  }

  .sidebar-library {
    min-width: 260px;
    max-width: 520px;
    background-color: var(--bg-subtle);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    position: relative;
  }

  .sidebar-resizer {
    width: 6px;
    margin-left: -3px;
    cursor: col-resize;
    position: relative;
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: center;
    user-select: none;
    -webkit-user-select: none;
    flex-shrink: 0;
    transition: background-color 0.15s ease;
  }

  .sidebar-resizer:hover,
  .sidebar-resizer.is-resizing {
    background-color: rgba(16, 185, 129, 0.2);
  }

  .resizer-handle {
    width: 2px;
    height: 100%;
    background-color: transparent;
    transition: background-color 0.15s ease;
  }

  .sidebar-resizer:hover .resizer-handle,
  .sidebar-resizer.is-resizing .resizer-handle {
    background-color: #10b981;
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
    padding: 12px 12px 2px 12px;
  }

  .sidebar-heading {
    font-size: 11px;
    font-weight: 700;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .sidebar-title-actions {
    display: flex;
    align-items: center;
    gap: 4px;
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
    padding: 0;
    border-bottom: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .search-wrap {
    position: relative;
    display: flex;
    align-items: center;
    padding: 0 10px;
  }

  :global(.search-icon) {
    position: absolute;
    left: 18px;
    color: var(--text-secondary);
    pointer-events: none;
  }

  .sidebar-search {
    width: 100%;
    height: 28px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 0 24px 0 28px;
    font-size: var(--text-xs);
    color: var(--text);
    box-sizing: border-box;
    transition: border-color 0.12s ease;
  }

  .sidebar-search:focus {
    outline: none;
    border-color: var(--accent);
  }

  .btn-clear-search {
    position: absolute;
    right: 16px;
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    padding: 0;
  }

  /* Category Filter Strip */
  .category-filter-strip {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 0 10px 6px;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    scroll-behavior: smooth;
    flex-shrink: 0;
  }

  .category-filter-strip::-webkit-scrollbar {
    height: 4px;
  }

  .category-filter-strip::-webkit-scrollbar-track {
    background: transparent;
  }

  .category-filter-strip::-webkit-scrollbar-thumb {
    background-color: rgba(128, 128, 128, 0.35);
    border-radius: 2px;
  }

  .category-filter-strip:hover::-webkit-scrollbar-thumb {
    background-color: rgba(128, 128, 128, 0.6);
  }

  .cat-chip {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    height: 23px;
    padding: 0 9px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 12px;
    font-size: 11px;
    font-weight: 500;
    color: var(--text-secondary);
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.14s ease;
    flex-shrink: 0;
    box-sizing: border-box;
  }

  .cat-chip:hover {
    background-color: var(--surface-hover);
    color: var(--text);
    border-color: var(--border-hover, rgba(128, 128, 128, 0.35));
  }

  .cat-chip.active {
    background-color: color-mix(in srgb, var(--accent) 14%, var(--surface));
    color: var(--accent);
    border-color: var(--accent);
    font-weight: 600;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
  }

  .cat-chip-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .cat-chip-count {
    font-size: 10px;
    opacity: 0.75;
    margin-left: 2px;
    font-weight: 500;
  }

  .cat-chip-selected-badge {
    font-size: 10px;
    font-weight: 700;
    color: #10b981;
    background: rgba(16, 185, 129, 0.15);
    padding: 0 4px;
    border-radius: 3px;
    margin-left: 2px;
  }

  /* Selection Bar & Master Checkbox */
  .selection-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 10px;
    background-color: rgba(255, 255, 255, 0.02);
    border-bottom: 1px solid var(--border);
    font-size: var(--text-xs);
    height: 32px;
    box-sizing: border-box;
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
    font-size: var(--text-xs);
    font-weight: 600;
  }

  .count-total {
    color: var(--text-secondary);
    font-size: var(--text-xs);
  }

  .batch-controls {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .btn-batch-scan {
    height: 22px;
    padding: 0 8px;
    background-color: #10b981;
    color: #0f172a;
    border: 1px solid #10b981;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
    line-height: 1;
    transition: all 0.1s ease;
  }

  .btn-batch-scan:hover {
    background-color: #059669;
    border-color: #059669;
  }

  .btn-batch-merge {
    height: 22px;
    padding: 0 7px;
    background-color: rgba(0, 122, 255, 0.15);
    border: 1px solid rgba(0, 122, 255, 0.3);
    color: #3894ff;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    cursor: pointer;
    white-space: nowrap;
    flex-shrink: 0;
    line-height: 1;
    transition: opacity 0.1s ease;
  }

  .btn-batch-merge:hover {
    opacity: 0.9;
  }

  .btn-icon-only {
    height: 22px;
    width: 22px;
    padding: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    border-radius: 4px;
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.1s ease;
  }

  .btn-batch-delete {
    background-color: rgba(255, 59, 48, 0.12);
    color: #ff453a;
    border: 1px solid rgba(255, 59, 48, 0.25);
  }

  .btn-batch-delete:hover {
    background-color: #ff3b30;
    color: #ffffff;
  }

  .btn-batch-clear {
    background-color: transparent;
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .btn-batch-clear:hover {
    background-color: var(--surface-hover);
    color: var(--text);
  }

  /* Wordlist Item List */
  .wordlists-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px 10px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .wordlist-item {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 8px 10px;
    background-color: rgba(255, 255, 255, 0.025);
    border: 1px solid rgba(255, 255, 255, 0.07);
    border-radius: 8px;
    cursor: pointer;
    transition: background-color 0.12s ease, border-color 0.12s ease, box-shadow 0.12s ease;
    position: relative;
    user-select: none;
    -webkit-user-select: none;
  }

  .wordlist-item:hover {
    background-color: rgba(255, 255, 255, 0.05);
    border-color: rgba(255, 255, 255, 0.12);
  }

  .wordlist-item.focused {
    outline: none;
  }

  .wordlist-item.active {
    background-color: rgba(0, 122, 255, 0.12);
    border-color: rgba(0, 122, 255, 0.45);
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.18);
  }

  .wordlist-item.checked {
    border-color: rgba(0, 122, 255, 0.35);
  }

  .wordlist-item.is-primary-target::before {
    content: '';
    position: absolute;
    left: 0;
    top: 6px;
    bottom: 6px;
    width: 3px;
    border-radius: 0 2px 2px 0;
    background-color: #10b981;
  }

  .item-checkbox-wrap {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-top: 1px;
    flex-shrink: 0;
  }

  .item-main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .item-header-row {
    display: flex;
    align-items: center;
    gap: 6px;
    line-height: 1.2;
    min-height: 18px;
    min-width: 0;
  }

  .cat-indicator-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .item-name {
    font-size: var(--text-sm, 13px);
    font-weight: 600;
    color: var(--text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
    flex: 1;
    line-height: 1.2;
  }

  .badge-target {
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.5px;
    background-color: rgba(16, 185, 129, 0.15);
    color: #10b981;
    border: 1px solid rgba(16, 185, 129, 0.35);
    border-radius: 4px;
    padding: 1px 5px;
    display: inline-flex;
    align-items: center;
    gap: 3px;
    margin-left: auto;
    flex-shrink: 0;
  }

  .badge-builtin {
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.5px;
    background-color: rgba(99, 102, 241, 0.15);
    color: #818cf8;
    border: 1px solid rgba(99, 102, 241, 0.35);
    border-radius: 4px;
    padding: 1px 5px;
    display: inline-flex;
    align-items: center;
    gap: 3px;
    flex-shrink: 0;
  }

  :global([data-mode="light"]) .badge-builtin {
    background-color: rgba(79, 70, 229, 0.12);
    color: #4338ca;
    border-color: rgba(79, 70, 229, 0.3);
  }

  .item-tags {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: var(--text-secondary);
    white-space: nowrap !important;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.2;
    margin-top: 2px;
  }

  .tag-lines,
  .tag-size {
    font-family: var(--font-mono);
    font-size: 11px;
    font-weight: 500;
    letter-spacing: -0.01em;
    white-space: nowrap !important;
    flex-shrink: 0;
  }

  .tag-category {
    font-weight: 600;
    text-transform: uppercase;
    font-size: 9.5px;
    letter-spacing: 0.5px;
    white-space: nowrap !important;
    flex-shrink: 0;
  }

  .tag-divider {
    opacity: 0.4;
    flex-shrink: 0;
  }

  .tag-badge {
    background-color: rgba(128, 128, 128, 0.18);
    border: 1px solid rgba(128, 128, 128, 0.2);
    padding: 1px 5px;
    border-radius: 4px;
    margin-left: auto;
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.02em;
    flex-shrink: 0;
  }

  .tag-linked {
    background-color: rgba(0, 122, 255, 0.15);
    color: var(--accent);
  }

  .tag-builtin {
    background-color: rgba(99, 102, 241, 0.14);
    color: #818cf8;
    border-color: rgba(99, 102, 241, 0.28);
  }

  :global([data-mode="light"]) .tag-builtin {
    background-color: rgba(79, 70, 229, 0.12);
    color: #4338ca;
    border-color: rgba(79, 70, 229, 0.25);
  }

  /* Sticky Frosted-Glass Sidebar Footer */
  .sidebar-footer {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 10px;
    background: color-mix(in srgb, var(--surface) 85%, transparent);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    border-top: 1px solid var(--border);
    flex-shrink: 0;
    z-index: 5;
  }

  .btn-footer-primary {
    width: 100%;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    background: linear-gradient(180deg, #0a84ff 0%, #0071e3 100%);
    border: 1px solid rgba(255, 255, 255, 0.18);
    border-radius: 7px;
    color: #ffffff;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.25), inset 0 1px 0 rgba(255, 255, 255, 0.2);
    transform-origin: center center;
    transition: transform 0.18s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.18s ease, filter 0.18s ease;
    will-change: transform;
    backface-visibility: hidden;
    -webkit-font-smoothing: antialiased;
  }

  .btn-footer-primary:hover {
    filter: brightness(1.06);
    box-shadow: 0 3px 10px rgba(0, 122, 255, 0.35), inset 0 1px 0 rgba(255, 255, 255, 0.3);
    transform: scale(1.015);
  }

  .btn-footer-primary:active {
    transform: scale(0.985);
    filter: brightness(0.96);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  }

  .btn-footer-secondary {
    width: 100%;
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 7px;
    color: var(--text-secondary);
    font-size: 11.5px;
    font-weight: 500;
    cursor: pointer;
    transform-origin: center center;
    transition: transform 0.18s cubic-bezier(0.16, 1, 0.3, 1), background-color 0.18s ease, color 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
    will-change: transform;
    backface-visibility: hidden;
    -webkit-font-smoothing: antialiased;
  }

  .btn-footer-secondary:hover {
    background-color: var(--surface-hover);
    color: var(--text);
    border-color: var(--border);
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
    transform: scale(1.015);
  }

  .btn-footer-secondary:active {
    transform: scale(0.985);
    background-color: var(--surface);
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
    user-select: text;
    -webkit-user-select: text;
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
    height: 48px;
    padding: 0 14px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    background-color: var(--bg-subtle);
    overflow-x: auto;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
    flex-shrink: 0;
  }

  .editor-header::-webkit-scrollbar {
    display: none;
  }

  .header-left {
    display: flex;
    align-items: center;
    min-width: 0;
    flex: 1 1 auto;
    overflow: hidden;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    max-width: 100%;
  }

  .editor-title {
    font-size: var(--text-md, 14.5px);
    font-weight: 600;
    color: var(--text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 60px;
    flex-shrink: 1;
  }

  .type-pill {
    font-size: 11px;
    padding: 2px 7px;
    border-radius: 10px;
    font-weight: 500;
    background-color: rgba(52, 199, 89, 0.12);
    color: #34c759;
    border: 1px solid rgba(52, 199, 89, 0.25);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .type-linked {
    background-color: rgba(0, 122, 255, 0.12);
    color: var(--accent);
    border-color: rgba(0, 122, 255, 0.25);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .badge-builtin-pill {
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.5px;
    background-color: rgba(99, 102, 241, 0.16);
    color: #818cf8;
    border: 1px solid rgba(99, 102, 241, 0.36);
    border-radius: 12px;
    padding: 2px 8px;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
    white-space: nowrap;
  }

  :global([data-mode="light"]) .badge-builtin-pill {
    background-color: rgba(79, 70, 229, 0.12);
    color: #4338ca;
    border-color: rgba(79, 70, 229, 0.3);
  }

  .cat-pill {
    font-size: 11px;
    padding: 2px 7px;
    border-radius: 10px;
    border: 1px solid;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.3px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
    white-space: nowrap;
  }

  .btn-text-full {
    display: inline;
    white-space: nowrap;
  }

  .btn-text-short {
    display: none;
    white-space: nowrap;
  }

  /* Responsive label switching via Container Query */
  @container editor (max-width: 760px) {
    .btn-text-full {
      display: none;
    }
    .btn-text-short {
      display: inline;
    }
    .header-right {
      gap: 5px;
    }
  }

  @container editor (max-width: 520px) {
    .btn-text-full,
    .btn-text-short {
      display: none;
    }
  }

  /* Fallback Media Query for window-level scaling */
  @media (max-width: 1080px) {
    .btn-text-full {
      display: none;
    }
    .btn-text-short {
      display: inline;
    }
  }

  @media (max-width: 820px) {
    .btn-text-full,
    .btn-text-short {
      display: none;
    }
  }

  .btn-reset-default {
    height: 28px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background-color: rgba(99, 102, 241, 0.12);
    border: 1px solid rgba(99, 102, 241, 0.3);
    border-radius: 6px;
    color: #818cf8;
    padding: 0 10px;
    font-size: var(--text-xs);
    font-weight: 600;
    cursor: pointer;
    transition: all 0.12s ease;
    user-select: none;
    -webkit-user-select: none;
    white-space: nowrap !important;
    flex-shrink: 0;
  }

  .btn-reset-default:hover {
    background-color: rgba(99, 102, 241, 0.22);
    border-color: rgba(99, 102, 241, 0.5);
    color: #a5b4fc;
  }

  :global([data-mode="light"]) .btn-reset-default {
    background-color: rgba(79, 70, 229, 0.1);
    color: #4338ca;
    border-color: rgba(79, 70, 229, 0.25);
  }

  :global([data-mode="light"]) .btn-reset-default:hover {
    background-color: rgba(79, 70, 229, 0.18);
    color: #3730a3;
  }

  .btn-lock-toggle {
    height: 28px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background-color: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: var(--text-secondary);
    padding: 0 10px;
    font-size: var(--text-xs);
    font-weight: 500;
    cursor: pointer;
    transition: all 0.12s ease;
    white-space: nowrap !important;
    flex-shrink: 0;
  }

  .btn-lock-toggle:hover {
    background-color: rgba(255, 255, 255, 0.08);
    color: var(--text);
    border-color: rgba(255, 255, 255, 0.18);
  }

  .btn-lock-toggle.is-unlocked {
    background-color: rgba(245, 158, 11, 0.12);
    color: #f59e0b;
    border-color: rgba(245, 158, 11, 0.35);
    font-weight: 600;
  }

  .btn-use-scan {
    height: 28px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background-color: rgba(0, 122, 255, 0.14);
    color: #3894ff;
    border: 1px solid rgba(0, 122, 255, 0.32);
    border-radius: 6px;
    padding: 0 11px;
    font-size: var(--text-xs);
    font-weight: 600;
    cursor: pointer;
    transition: all 0.12s ease;
    white-space: nowrap !important;
    flex-shrink: 0;
  }

  .btn-use-scan:hover {
    background-color: rgba(0, 122, 255, 0.24);
    border-color: rgba(0, 122, 255, 0.5);
  }

  .btn-use-scan.is-active-target {
    background-color: rgba(16, 185, 129, 0.14);
    color: #10b981;
    border-color: rgba(16, 185, 129, 0.35);
  }

  .btn-header-action {
    height: 28px;
    width: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background-color: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.12s ease;
    flex-shrink: 0;
  }

  .btn-header-action:hover {
    background-color: rgba(255, 255, 255, 0.08);
    color: var(--text);
    border-color: rgba(255, 255, 255, 0.18);
  }

  .btn-header-danger:hover {
    background-color: rgba(255, 59, 48, 0.12);
    color: #ff453a;
    border-color: rgba(255, 59, 48, 0.3);
  }

  .stats-strip {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 32px;
    padding: 0 16px;
    background-color: var(--surface);
    border-bottom: 1px solid var(--border);
    font-size: 11px;
    gap: 16px;
    box-sizing: border-box;
  }

  .stats-left {
    display: flex;
    align-items: center;
    gap: 6px;
    overflow: hidden;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .stat-item {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .stat-label {
    font-size: 10px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .stat-value {
    font-weight: 600;
    color: var(--text);
  }

  .stat-sep {
    color: var(--text-secondary);
    opacity: 0.35;
    font-size: 10px;
  }

  .path-display {
    display: flex;
    align-items: center;
    gap: 6px;
    max-width: 52%;
    min-width: 0;
    margin-left: auto;
  }

  .path-text {
    font-size: 11px;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: right;
  }

  .btn-copy-path {
    height: 20px;
    width: 20px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid transparent;
    border-radius: 4px;
    color: var(--text-secondary);
    cursor: pointer;
    flex-shrink: 0;
    transition: all 0.1s ease;
  }

  .btn-copy-path:hover {
    background-color: var(--surface-hover);
    border-color: var(--border);
    color: var(--text);
  }

  .health-warning-strip {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 16px;
    background-color: rgba(255, 149, 0, 0.12);
    border-bottom: 1px solid rgba(255, 149, 0, 0.28);
    font-size: var(--text-xs);
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
    font-size: var(--text-xs);
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
    min-height: 38px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 14px;
    gap: 8px;
    position: relative;
    z-index: 50;
    user-select: none;
    overflow: visible;
  }

  .tools-group {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
  }

  .tools-extended {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .tools-overflow,
  .dropdown-wrapper.tools-overflow {
    display: none !important;
  }

  .dropdown-section-title {
    font-size: var(--text-2xs);
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
    padding: 0 7px;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text-secondary);
    font-size: var(--text-xs);
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
    padding: 0 6px;
    font-size: var(--text-xs);
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
    min-width: 185px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.45), 0 0 0 1px rgba(255, 255, 255, 0.05);
    padding: 4px;
    z-index: 1000;
    display: flex;
    flex-direction: column;
    gap: 2px;
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
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
    font-size: var(--text-xs);
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
    gap: 4px;
    font-size: var(--text-xs);
    flex-shrink: 0;
  }

  .btn-page {
    height: 24px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 7px;
    font-size: var(--text-xs);
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
    font-size: var(--text-xs);
    white-space: nowrap;
    padding: 0 2px;
  }

  .page-indicator-compact {
    display: none;
  }

  /* Container Queries for Adaptive Toolbar */
  @container editor (max-width: 860px) {
    .btn-label-long {
      display: none !important;
    }
  }

  @container editor (max-width: 760px) {
    .tools-extended {
      display: none !important;
    }

    .tools-overflow,
    .dropdown-wrapper.tools-overflow {
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
    font-size: var(--text-xs);
    line-height: 24px;
    overflow-y: hidden;
    user-select: none;
    flex-shrink: 0;
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
    font-size: var(--text-xs);
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
    background-color: rgba(0, 122, 255, 0.28);
    color: var(--accent);
    font-weight: 700;
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
    background-color: var(--accent);
    border-left-color: var(--accent-contrast, #ffffff);
  }

  .line-row.line-row-selected .line-input {
    color: var(--accent-contrast, #ffffff) !important;
    background-color: transparent !important;
  }

  .line-input {
    width: 100%;
    height: 24px;
    padding: 0 8px;
    background: none;
    border: none;
    outline: none;
    font-size: var(--text-sm);
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
    user-select: text;
    -webkit-user-select: text;
  }

  .raw-textarea-wrap {
    flex: 1;
    display: flex;
    overflow: hidden;
    user-select: text;
    -webkit-user-select: text;
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
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    line-height: 24px;
    white-space: pre;
    overflow-y: auto;
    overflow-x: auto;
    box-sizing: border-box;
    user-select: text !important;
    -webkit-user-select: text !important;
  }

  .raw-textarea::selection {
    background-color: var(--accent);
    color: var(--accent-contrast, #ffffff);
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
    font-size: var(--text-xs);
    color: #ff9500;
    font-weight: 600;
  }

  .viewmode-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: var(--text-xs);
    color: var(--text-secondary);
    padding: 3px 8px;
    border-radius: 4px;
    background-color: var(--bg-hover, rgba(255, 255, 255, 0.05));
    border: 1px solid var(--border);
  }

  .btn-footer-unlock {
    height: 26px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background-color: rgba(0, 122, 255, 0.12);
    color: var(--accent);
    border: 1px solid var(--accent);
    border-radius: 4px;
    padding: 0 12px;
    font-size: var(--text-xs);
    font-weight: 600;
    cursor: pointer;
    transition: all 0.1s ease;
  }

  .btn-footer-unlock:hover {
    background-color: var(--accent);
    color: var(--accent-contrast, #ffffff);
  }

  .raw-textarea.is-read-only,
  .line-input.is-read-only {
    cursor: text;
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
    font-size: var(--text-xs);
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
    width: 480px;
    max-width: 95vw;
  }

  .modal-head {
    height: 38px;
    padding: 0 14px;
    background-color: var(--bg-subtle);
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: var(--text-sm);
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
    font-size: var(--text-base);
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
    font-size: var(--text-xs);
  }

  .target-filename {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text);
  }

  .target-kind-pill {
    font-size: var(--text-xs);
    padding: 2px 6px;
    border-radius: var(--radius-xs);
    background-color: rgba(128, 128, 128, 0.15);
    color: var(--text-secondary);
  }

  .target-kind-pill.pill-linked {
    background-color: rgba(0, 122, 255, 0.15);
    color: var(--accent);
  }

  .target-row-more {
    font-size: var(--text-xs);
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
    font-size: var(--text-xs);
    color: var(--text-secondary);
    line-height: 1.4;
  }

  .dialog-desc {
    font-size: var(--text-sm);
    color: var(--text);
    margin: 0;
    line-height: 1.4;
  }

  /* Shortcuts Cheatsheet Grid */
  .shortcuts-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
    font-size: var(--text-xs);
  }

  .shortcut-section {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .section-title {
    font-size: var(--text-sm);
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
    font-size: var(--text-2xs);
    color: var(--text);
    white-space: nowrap;
  }

  .shortcut-desc {
    color: var(--text-secondary);
    font-size: var(--text-xs);
    text-align: right;
  }

  .modal-label {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }

  .modal-input {
    height: 28px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0 8px;
    font-size: var(--text-sm);
    color: var(--text);
  }

  .modal-textarea {
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 8px;
    font-size: var(--text-sm);
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
    font-size: var(--text-2xs);
    color: var(--text-secondary);
    margin-right: auto;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .btn-cancel {
    height: 28px;
    padding: 0 12px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    font-size: var(--text-xs);
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
    font-size: var(--text-xs);
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
    font-size: var(--text-xs);
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
    font-size: var(--text-xs);
    font-weight: 600;
    cursor: pointer;
  }

  .reset-target-name {
    color: #818cf8;
    font-weight: 700;
  }

  .btn-primary-confirm {
    height: 28px;
    padding: 0 14px;
    background-color: #4f46e5 !important;
    color: #ffffff !important;
    border: none;
    border-radius: 4px;
    font-size: var(--text-xs);
    font-weight: 600;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
    box-shadow: 0 1px 3px rgba(79, 70, 229, 0.3);
    transition: background-color 0.15s ease, transform 0.1s ease;
  }

  .btn-primary-confirm:hover {
    background-color: #4338ca !important;
  }

  .btn-primary-confirm:active {
    background-color: #3730a3 !important;
  }

  /* Inline Category Creator & Field Layouts */
  .modal-field-block {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .modal-field-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .modal-label-text {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text);
  }

  .btn-text-action {
    all: unset;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    font-weight: 500;
    color: var(--accent);
    cursor: pointer;
    padding: 1px 4px;
    border-radius: 4px;
    transition: background-color 0.1s ease;
  }

  .btn-text-action:hover {
    background-color: rgba(0, 122, 255, 0.1);
  }

  .modal-select {
    width: 100%;
    height: 30px;
    padding: 0 8px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 4px;
    color: var(--text);
    font-size: var(--text-xs);
    box-sizing: border-box;
    cursor: pointer;
  }

  .modal-select:focus {
    outline: 2px solid var(--accent);
    border-color: transparent;
  }

  .inline-cat-creator {
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 8px 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .inline-cat-input {
    height: 28px !important;
    font-size: 11px !important;
  }

  .color-swatches {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
  }

  .swatch-btn {
    all: unset;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    cursor: pointer;
    box-sizing: border-box;
    transition: transform 0.12s ease;
    border: 2px solid transparent;
  }

  .swatch-btn:hover {
    transform: scale(1.18);
  }

  .swatch-btn.selected {
    transform: scale(1.25);
    border-color: #ffffff;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.4);
  }

  .btn-inline-add {
    all: unset;
    height: 24px;
    padding: 0 10px;
    background-color: var(--accent);
    color: var(--accent-contrast);
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    align-self: flex-end;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    transition: opacity 0.1s ease;
  }

  .btn-inline-add:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Manage Categories Modal Styles */
  .modal-dialog-wide {
    width: 620px !important;
    max-width: 95vw;
  }

  .cat-error-alert {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 10px;
    background-color: rgba(255, 59, 48, 0.12);
    border: 1px solid rgba(255, 59, 48, 0.3);
    border-radius: 6px;
    color: #ff3b30;
    font-size: 12px;
  }

  .cat-list-section {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .section-title {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary);
  }

  .cat-table-wrap {
    max-height: 220px;
    overflow-y: auto;
    border: 1px solid var(--border);
    border-radius: 6px;
    background-color: var(--bg-subtle);
  }

  .cat-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
  }

  .cat-table th {
    background-color: var(--surface);
    color: var(--text-secondary);
    padding: 6px 10px;
    font-weight: 600;
    font-size: 11px;
    text-align: left;
    border-bottom: 1px solid var(--border);
    position: sticky;
    top: 0;
    z-index: 1;
  }

  .cat-table td {
    padding: 6px 10px;
    border-bottom: 1px solid var(--border);
    color: var(--text);
    vertical-align: middle;
  }

  .cat-dot-cell {
    display: block;
    width: 12px;
    height: 12px;
    border-radius: 50%;
  }

  .cat-name-cell {
    font-weight: 500;
  }

  .cat-badge-default {
    display: inline-block;
    padding: 1px 6px;
    background-color: rgba(100, 116, 139, 0.15);
    border: 1px solid rgba(100, 116, 139, 0.3);
    border-radius: 4px;
    font-size: 10px;
    color: var(--text-secondary);
    font-weight: 600;
  }

  .cat-badge-custom {
    display: inline-block;
    padding: 1px 6px;
    background-color: rgba(0, 122, 255, 0.15);
    border: 1px solid rgba(0, 122, 255, 0.3);
    border-radius: 4px;
    font-size: 10px;
    color: var(--accent);
    font-weight: 600;
  }

  .action-btn-group {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 4px;
  }

  .btn-icon-action {
    all: unset;
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    color: var(--text-secondary);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    transition: all 0.1s ease;
  }

  .btn-icon-action:hover {
    color: var(--text);
    background-color: var(--surface-hover);
  }

  .btn-icon-action.edit:hover {
    color: var(--accent);
    background-color: rgba(0, 122, 255, 0.12);
  }

  .btn-icon-action.delete:hover {
    color: #ff3b30;
    background-color: rgba(255, 59, 48, 0.12);
  }

  .btn-icon-action.save:hover {
    color: #34c759;
    background-color: rgba(52, 199, 89, 0.1);
  }

  .swatches-inline {
    display: flex;
    align-items: center;
    gap: 3px;
  }

  .swatch-mini {
    all: unset;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    cursor: pointer;
    box-sizing: border-box;
    border: 1px solid transparent;
  }

  .swatch-mini.selected {
    border-color: #ffffff;
    box-shadow: 0 0 0 1px #000;
  }

  .cat-edit-input {
    height: 24px !important;
    font-size: 11px !important;
    padding: 2px 6px !important;
  }

  .cat-create-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 10px 12px;
  }

  .cat-create-form {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .cat-create-inputs {
    display: flex;
    flex: 1;
    align-items: center;
    gap: 8px;
  }

  .cat-new-input {
    flex: 1;
    height: 28px !important;
    font-size: 11px !important;
  }

  .color-swatches-strip {
    display: flex;
    align-items: center;
    gap: 5px;
  }

  .btn-add-cat {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    white-space: nowrap;
  }

  /* Unified Modal Source Switcher */
  .modal-mode-tabs {
    display: flex;
    gap: 4px;
    padding: 3px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    border-radius: 7px;
  }

  .modal-mode-tab {
    flex: 1;
    height: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    background: transparent;
    border: 1px solid transparent;
    border-radius: 5px;
    color: var(--text-secondary);
    font-size: 11.5px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.12s ease;
  }

  .modal-mode-tab:hover {
    color: var(--text);
    background-color: rgba(255, 255, 255, 0.04);
  }

  .modal-mode-tab.active {
    background-color: var(--surface);
    color: var(--accent);
    border-color: rgba(0, 122, 255, 0.3);
    font-weight: 600;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
  }

  /* File Picker & Dropzone in Modal */
  .file-picker-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .file-picker-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .file-health-quick-badge {
    display: inline-flex;
    align-items: center;
  }

  .badge-dup-warn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 1px 7px;
    border-radius: 10px;
    background-color: color-mix(in srgb, #f59e0b 16%, transparent);
    color: #f59e0b;
    font-size: 10px;
    font-weight: 500;
  }

  .badge-dup-clean {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 1px 7px;
    border-radius: 10px;
    background-color: color-mix(in srgb, #10b981 16%, transparent);
    color: #10b981;
    font-size: 10px;
  }

  .file-dropzone {
    width: 100%;
    box-sizing: border-box;
    padding: 16px 14px;
    border: 2px dashed color-mix(in srgb, var(--accent) 40%, var(--border));
    border-radius: 8px;
    background-color: color-mix(in srgb, var(--accent) 4%, var(--bg-subtle));
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    text-align: center;
    transition: all 0.18s cubic-bezier(0.16, 1, 0.3, 1);
    position: relative;
    overflow: hidden;
  }

  .file-dropzone:hover {
    border-color: var(--accent);
    background-color: color-mix(in srgb, var(--accent) 9%, var(--bg-subtle));
    color: var(--text);
  }

  .file-dropzone.dragging {
    border-color: var(--accent);
    background-color: color-mix(in srgb, var(--accent) 15%, var(--bg-subtle));
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 25%, transparent);
    transform: scale(1.01);
  }

  .dropzone-icon-circle {
    width: 38px;
    height: 38px;
    border-radius: 50%;
    background-color: color-mix(in srgb, var(--accent) 16%, var(--surface));
    border: 1px solid color-mix(in srgb, var(--accent) 28%, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), background-color 0.15s ease;
  }

  .file-dropzone:hover .dropzone-icon-circle,
  .dropzone-icon-circle.bounce {
    transform: translateY(-2px);
  }

  .dropzone-text-group {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 3px;
  }

  .dropzone-title {
    font-size: 12.5px;
    font-weight: 600;
    color: var(--text);
  }

  .dropzone-subtitle {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .dropzone-chips {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    margin-top: 4px;
    flex-wrap: wrap;
  }

  .dropzone-tag {
    font-size: 9.5px;
    font-family: var(--font-mono, monospace);
    padding: 2px 7px;
    border-radius: 10px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    color: var(--text-muted);
  }

  .selected-file-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 12px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  }

  .file-card-info {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
    flex: 1;
  }

  .file-card-icon-wrap {
    width: 32px;
    height: 32px;
    border-radius: 6px;
    background-color: color-mix(in srgb, var(--accent) 14%, var(--bg-subtle));
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .file-card-details {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
    flex: 1;
  }

  .file-card-details .file-name {
    font-size: 12px;
    font-weight: 600;
    color: var(--text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-card-details .file-path {
    font-size: 10.5px;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-card-meta {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 10px;
    margin-top: 2px;
  }

  .meta-sep {
    color: var(--border);
  }

  .text-warn {
    color: #f59e0b;
    display: inline-flex;
    align-items: center;
    gap: 3px;
  }

  .text-success {
    color: #10b981;
    display: inline-flex;
    align-items: center;
    gap: 3px;
  }

  .file-card-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .btn-clean-file-now {
    height: 24px;
    padding: 0 8px;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    border-radius: 4px;
    background-color: color-mix(in srgb, #f59e0b 16%, var(--surface));
    border: 1px solid color-mix(in srgb, #f59e0b 35%, transparent);
    color: #f59e0b;
    font-size: 10.5px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.12s ease;
  }

  .btn-clean-file-now:hover {
    background-color: color-mix(in srgb, #f59e0b 24%, var(--surface));
    border-color: #f59e0b;
  }

  .btn-change-file {
    padding: 3px 8px;
    height: 24px;
    background-color: var(--surface);
    border: 1px solid var(--border);
    border-radius: 4px;
    font-size: 10.5px;
    color: var(--text-secondary);
    cursor: pointer;
    flex-shrink: 0;
    transition: all 0.1s ease;
  }

  .btn-change-file:hover {
    color: var(--text);
    border-color: var(--accent);
  }

  .btn-clear-file {
    width: 24px;
    height: 24px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    border: 1px solid var(--border);
    background-color: var(--surface);
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.1s ease;
  }

  .btn-clear-file:hover {
    color: #ef4444;
    border-color: #ef4444;
    background-color: color-mix(in srgb, #ef4444 10%, var(--surface));
  }

  /* Smart Deduplication Checkbox Row */
  .modal-checkbox-row {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 12px;
    border-radius: 6px;
    background-color: var(--bg-subtle);
    border: 1px solid var(--border);
    cursor: pointer;
    margin-top: 4px;
    transition: background-color 0.12s ease, border-color 0.12s ease;
  }

  .modal-checkbox-row:hover {
    background-color: var(--surface-hover);
    border-color: color-mix(in srgb, var(--accent) 30%, var(--border));
  }

  .modal-checkbox-row .checkbox-input {
    margin-top: 2px;
    cursor: pointer;
    accent-color: var(--accent);
  }

  .checkbox-text-group {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
    flex: 1;
  }

  .checkbox-title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .checkbox-title {
    font-size: 11.5px;
    font-weight: 600;
    color: var(--text);
  }

  .checkbox-badge {
    font-size: 9px;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 8px;
    background-color: color-mix(in srgb, var(--accent) 15%, transparent);
    color: var(--accent);
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .checkbox-desc {
    font-size: 10.5px;
    color: var(--text-secondary);
    line-height: 1.4;
  }

  /* Text stats bar & duplicate action in Blank / Custom */
  .text-stats-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 10.5px;
  }

  .stats-counter {
    color: var(--text-muted);
  }

  .dup-alert-pill {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 1px 6px;
    border-radius: 10px;
    background-color: color-mix(in srgb, #f59e0b 16%, transparent);
    color: #f59e0b;
    font-size: 10px;
    font-weight: 500;
  }

  .clean-check-pill {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 1px 6px;
    border-radius: 10px;
    background-color: color-mix(in srgb, #10b981 16%, transparent);
    color: #10b981;
    font-size: 10px;
  }

  .btn-clean-text-dups {
    padding: 2px 7px;
    font-size: 10px;
    border-radius: 4px;
    background-color: color-mix(in srgb, #f59e0b 15%, var(--surface));
    border: 1px solid color-mix(in srgb, #f59e0b 35%, transparent);
    color: #f59e0b;
    cursor: pointer;
    transition: all 0.12s ease;
  }

  .btn-clean-text-dups:hover {
    background-color: color-mix(in srgb, #f59e0b 25%, var(--surface));
    border-color: #f59e0b;
  }

  :global(.spin) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
