import { describe, it, expect, beforeEach } from 'vitest';
import { themeStore, THEMES, type ThemePalette } from './theme.svelte';

describe('ThemeStore & Palette System', () => {
  const store = new Map<string, string>();
  const domAttrs = new Map<string, string>();

  beforeEach(() => {
    store.clear();
    domAttrs.clear();

    const localStorageMock = {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => store.set(key, String(value)),
      removeItem: (key: string) => store.delete(key),
      clear: () => store.clear()
    };

    const documentMock = {
      documentElement: {
        setAttribute: (key: string, value: string) => domAttrs.set(key, value),
        getAttribute: (key: string) => domAttrs.get(key) ?? null,
        removeAttribute: (key: string) => domAttrs.delete(key)
      }
    };

    // Polyfill globals for Node environment test
    (globalThis as any).localStorage = localStorageMock;
    (globalThis as any).document = documentMock;

    themeStore.setPalette('neutral');
    themeStore.setMode('dark');
  });

  it('includes all 8 curated palettes with proper metadata and previews', () => {
    expect(THEMES.length).toBe(8);

    const ids = THEMES.map((t) => t.id);
    expect(ids).toContain('neutral');
    expect(ids).toContain('ocean');
    expect(ids).toContain('dracula');
    expect(ids).toContain('one-dark');
    expect(ids).toContain('catppuccin');
    expect(ids).toContain('nord');
    expect(ids).toContain('tokyo-night');
    expect(ids).toContain('gruvbox');

    for (const theme of THEMES) {
      expect(theme.name).toBeDefined();
      expect(theme.preview.bg).toMatch(/^#[0-9a-fA-F]{6}$/);
      expect(theme.preview.surface).toMatch(/^#[0-9a-fA-F]{6}$/);
      expect(theme.preview.accent).toMatch(/^#[0-9a-fA-F]{6}$/);
    }
  });

  it('switches palettes cleanly and updates DOM data-theme attribute', () => {
    const targetThemes: ThemePalette[] = [
      'dracula',
      'one-dark',
      'catppuccin',
      'nord',
      'tokyo-night',
      'gruvbox',
      'ocean',
      'neutral'
    ];

    for (const p of targetThemes) {
      themeStore.setPalette(p);
      expect(themeStore.palette).toBe(p);
      expect(themeStore.currentTheme.id).toBe(p);
      expect(domAttrs.get('data-theme')).toBe(p);
      expect(store.get('fuzznexus-theme-palette')).toBe(p);
    }
  });

  it('ignores invalid palette identifiers', () => {
    themeStore.setPalette('dracula');
    // @ts-expect-error test invalid value
    themeStore.setPalette('invalid-palette');
    expect(themeStore.palette).toBe('dracula');
  });

  it('updates mode and toggles properly across light, dark, and system', () => {
    themeStore.setMode('light');
    expect(themeStore.mode).toBe('light');
    expect(domAttrs.get('data-mode')).toBe('light');
    expect(store.get('fuzznexus-theme-mode')).toBe('light');

    themeStore.toggleMode(); // light -> dark
    expect(themeStore.mode).toBe('dark');
    expect(domAttrs.get('data-mode')).toBe('dark');

    themeStore.toggleMode(); // dark -> system
    expect(themeStore.mode).toBe('system');

    themeStore.toggleMode(); // system -> light
    expect(themeStore.mode).toBe('light');
  });
});
