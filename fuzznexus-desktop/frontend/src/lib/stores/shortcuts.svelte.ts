/**
 * Keyboard shortcut definitions, normalization helpers, and conflict-guard utilities.
 */

export type ShortcutCategory = 'Scan Engine' | 'Navigation' | 'Tools';

export interface ShortcutActionDef {
  id: string;
  label: string;
  category: ShortcutCategory;
  defaultCombo: string;
}

export const CRITICAL_SHORTCUTS: ShortcutActionDef[] = [
  { id: 'scan.startOrResume', label: 'Start / Resume Scan', category: 'Scan Engine', defaultCombo: 'Mod+Shift+Enter' },
  { id: 'scan.pause', label: 'Pause Scan', category: 'Scan Engine', defaultCombo: 'Mod+Shift+P' },
  { id: 'scan.stop', label: 'Stop Scan', category: 'Scan Engine', defaultCombo: 'Mod+Shift+X' },
  { id: 'scan.reset', label: 'Reset Session', category: 'Scan Engine', defaultCombo: 'Mod+Shift+Backspace' },
  { id: 'ui.viewGrid', label: 'Table View', category: 'Navigation', defaultCombo: 'Mod+1' },
  { id: 'ui.viewAnalytics', label: 'Analytics View', category: 'Navigation', defaultCombo: 'Mod+2' },
  { id: 'ui.viewWordlists', label: 'Wordlists View', category: 'Navigation', defaultCombo: 'Mod+3' },
  { id: 'ui.toggleOptions', label: 'Toggle Request Options', category: 'Tools', defaultCombo: 'Mod+Shift+O' },
  { id: 'ui.toggleHistory', label: 'Toggle Session History', category: 'Tools', defaultCombo: 'Mod+Shift+H' },
  { id: 'ui.toggleFilter', label: 'Toggle Advanced Filter', category: 'Tools', defaultCombo: 'Mod+K' },
  { id: 'ui.openShortcuts', label: 'Keyboard Shortcuts', category: 'Tools', defaultCombo: 'Mod+Shift+/' },
  { id: 'ui.toggleAbout', label: 'About FuzzNexus', category: 'Tools', defaultCombo: 'Mod+I' }
];

// Truly un-interceptable or destructive OS / Window-level shortcuts
const HARD_BLOCKED_COMBOS = new Set([
  'Mod+Space',
  'Mod+Q',
  'Mod+W',
  'Alt+Tab',
  'Alt+F4'
]);

// Standard text-editing or browser-chrome shortcuts: allowed, but trigger a soft warning
const SOFT_WARNING_COMBOS = new Set([
  'Mod+C',
  'Mod+V',
  'Mod+X',
  'Mod+A',
  'Mod+Z',
  'Mod+Shift+Z',
  'Mod+Y',
  'Mod+R',
  'Mod+T',
  'Mod+N',
  'Mod+P',
  'Mod+S',
  'Mod+F',
  'Mod+H',
  'Mod+M'
]);

/** Returns true when running on macOS. */
export function detectIsMac(): boolean {
  if (typeof navigator === 'undefined') return false;
  return /mac/i.test(navigator.platform || navigator.userAgent || '');
}

/** Normalizes a DOM KeyboardEvent.key into a canonical shortcut token. */
export function normalizeKey(key: string): string {
  if (!key) return '';
  if (key === ' ') return 'Space';
  if (key === '?') return '/';
  if (key.length === 1) return key.toUpperCase();
  return key;
}

/** Converts a KeyboardEvent into a canonical combo string (e.g. "Mod+Shift+Enter"). */
export function eventToCombo(
  e: Pick<KeyboardEvent, 'key' | 'metaKey' | 'ctrlKey' | 'altKey' | 'shiftKey'>,
  isMac: boolean = detectIsMac()
): string | null {
  const key = normalizeKey(e.key);
  if (!key || ['Meta', 'Control', 'Alt', 'Shift'].includes(key)) {
    return null;
  }

  const modPressed = isMac ? e.metaKey : e.ctrlKey;
  const parts: string[] = [];
  if (modPressed) parts.push('Mod');
  if (e.altKey) parts.push('Alt');
  if (e.shiftKey) parts.push('Shift');
  parts.push(key);
  return parts.join('+');
}

/** Returns true if the combo is reserved by the OS/window manager and cannot be bound. */
export function isHardBlockedShortcut(combo: string): boolean {
  return HARD_BLOCKED_COMBOS.has(combo);
}

/** Returns true if the combo overlaps with common text-editing or browser shortcuts. */
export function isSoftWarningShortcut(combo: string): boolean {
  return SOFT_WARNING_COMBOS.has(combo);
}

/** Returns true if the combo includes a primary modifier (Mod or Alt). */
export function hasModifier(combo: string): boolean {
  return combo.includes('Mod+') || combo.includes('Alt+');
}

/** Checks whether a KeyboardEvent matches the given canonical combo string. */
export function matchesCombo(
  e: Pick<KeyboardEvent, 'key' | 'metaKey' | 'ctrlKey' | 'altKey' | 'shiftKey'>,
  combo: string,
  isMac: boolean = detectIsMac()
): boolean {
  const pressed = eventToCombo(e, isMac);
  return pressed !== null && pressed === combo;
}

export function formatComboForDisplay(combo: string, isMac: boolean = detectIsMac()): string {
  if (!combo) return '';
  return combo
    .split('+')
    .map((part) => {
      if (part === 'Mod') return isMac ? 'Cmd' : 'Ctrl';
      if (part === 'Alt') return isMac ? 'Option' : 'Alt';
      return part;
    })
    .join('+');
}

const STORAGE_KEY = 'fuzznexus.shortcuts.v1';

class ShortcutsStore {
  bindings = $state<Record<string, string>>({});
  isModalOpen = $state<boolean>(false);

  constructor() {
    this.load();
  }

  private load() {
    const defaults: Record<string, string> = {};
    for (const def of CRITICAL_SHORTCUTS) {
      defaults[def.id] = def.defaultCombo;
    }
    if (typeof localStorage !== 'undefined') {
      try {
        const raw = localStorage.getItem(STORAGE_KEY);
        if (raw) {
          const parsed = JSON.parse(raw) as Record<string, string>;
          for (const [k, v] of Object.entries(parsed)) {
            if (defaults[k] && typeof v === 'string' && v.length > 0 && !isHardBlockedShortcut(v)) {
              defaults[k] = v;
            }
          }
        }
      } catch {
        // Ignore corrupt localStorage
      }
    }
    this.bindings = defaults;
  }

  private persist() {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(this.bindings));
    }
  }

  getCombo(id: string): string {
    return this.bindings[id] || CRITICAL_SHORTCUTS.find((s) => s.id === id)?.defaultCombo || '';
  }

  findConflict(combo: string, excludeId: string): ShortcutActionDef | null {
    for (const def of CRITICAL_SHORTCUTS) {
      if (def.id !== excludeId && this.getCombo(def.id) === combo) {
        return def;
      }
    }
    return null;
  }

  setBinding(id: string, combo: string): { ok: boolean; error?: string; warning?: string } {
    if (!hasModifier(combo)) {
      return { ok: false, error: 'Shortcut must include Cmd/Ctrl or Alt modifier' };
    }
    if (isHardBlockedShortcut(combo)) {
      return { ok: false, error: `${combo} is reserved by OS / Window Manager and cannot be overridden` };
    }
    const conflict = this.findConflict(combo, id);
    if (conflict) {
      return { ok: false, error: `Already assigned to "${conflict.label}"` };
    }
    this.bindings = { ...this.bindings, [id]: combo };
    this.persist();
    if (isSoftWarningShortcut(combo)) {
      return {
        ok: true,
        warning: `${combo} overlaps with standard text-editing or browser shortcuts`
      };
    }
    return { ok: true };
  }

  resetOne(id: string) {
    const def = CRITICAL_SHORTCUTS.find((s) => s.id === id);
    if (!def) return;
    this.bindings = { ...this.bindings, [id]: def.defaultCombo };
    this.persist();
  }

  resetAll() {
    const next: Record<string, string> = {};
    for (const def of CRITICAL_SHORTCUTS) {
      next[def.id] = def.defaultCombo;
    }
    this.bindings = next;
    this.persist();
  }
}

export const shortcutsStore = new ShortcutsStore();
