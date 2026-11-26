import { describe, it, expect, beforeEach } from 'vitest';
import {
  CRITICAL_SHORTCUTS,
  eventToCombo,
  formatComboForDisplay,
  isHardBlockedShortcut,
  isSoftWarningShortcut,
  matchesCombo,
  shortcutsStore
} from './shortcuts.svelte';

describe('Cross-Platform Shortcuts Store', () => {
  beforeEach(() => {
    shortcutsStore.resetAll();
  });

  it('matches Mod+Shift+Enter on macOS (metaKey) and Windows/Linux (ctrlKey)', () => {
    const macEvent = { key: 'Enter', metaKey: true, ctrlKey: false, altKey: false, shiftKey: true };
    const winEvent = { key: 'Enter', metaKey: false, ctrlKey: true, altKey: false, shiftKey: true };

    expect(matchesCombo(macEvent, 'Mod+Shift+Enter', true)).toBe(true);
    expect(matchesCombo(winEvent, 'Mod+Shift+Enter', false)).toBe(true);
    expect(matchesCombo(macEvent, 'Mod+Shift+Enter', false)).toBe(false);
  });

  it('ignores bare modifier keys during recording', () => {
    expect(eventToCombo({ key: 'Meta', metaKey: true, ctrlKey: false, altKey: false, shiftKey: false }, true)).toBeNull();
    expect(eventToCombo({ key: 'Control', metaKey: false, ctrlKey: true, altKey: false, shiftKey: false }, false)).toBeNull();
  });

  it('hard-blocks OS/Window shortcuts (Cmd/Ctrl+Q, W, Space, Alt+Tab, Alt+F4)', () => {
    for (const blocked of ['Mod+Q', 'Mod+W', 'Mod+Space', 'Alt+Tab', 'Alt+F4']) {
      expect(isHardBlockedShortcut(blocked)).toBe(true);
      const res = shortcutsStore.setBinding('scan.pause', blocked);
      expect(res.ok).toBe(false);
      expect(res.error).toContain('reserved');
    }
  });

  it('allows standard text-editing/browser shortcuts with a soft warning', () => {
    expect(isSoftWarningShortcut('Mod+C')).toBe(true);
    const res = shortcutsStore.setBinding('scan.pause', 'Mod+C');
    expect(res.ok).toBe(true);
    expect(res.warning).toContain('overlaps');
    expect(shortcutsStore.getCombo('scan.pause')).toBe('Mod+C');
  });

  it('prevents duplicate bindings across different actions', () => {
    const res = shortcutsStore.setBinding('scan.pause', 'Mod+Shift+Enter');
    expect(res.ok).toBe(false);
    expect(res.error).toContain('Start / Resume Scan');
  });

  it('allows valid custom binding and resets cleanly', () => {
    const res = shortcutsStore.setBinding('scan.pause', 'Mod+Alt+P');
    expect(res.ok).toBe(true);
    expect(shortcutsStore.getCombo('scan.pause')).toBe('Mod+Alt+P');

    shortcutsStore.resetAll();
    const def = CRITICAL_SHORTCUTS.find((s) => s.id === 'scan.pause')!;
    expect(shortcutsStore.getCombo('scan.pause')).toBe(def.defaultCombo);
  });

  it('formats shortcuts natively for macOS vs Windows/Linux', () => {
    expect(formatComboForDisplay('Mod+Shift+O', true)).toBe('Cmd+Shift+O');
    expect(formatComboForDisplay('Mod+Shift+O', false)).toBe('Ctrl+Shift+O');
  });
});
