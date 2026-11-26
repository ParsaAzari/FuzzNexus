export type ThemePalette =
  | 'neutral'
  | 'ocean'
  | 'dracula'
  | 'one-dark'
  | 'catppuccin'
  | 'nord'
  | 'tokyo-night'
  | 'gruvbox';

export type ThemeMode = 'system' | 'light' | 'dark';

export interface ThemeDefinition {
  id: ThemePalette;
  name: string;
  category: 'core' | 'community';
  preview: {
    bg: string;
    surface: string;
    accent: string;
  };
  description?: string;
}

export const THEMES: ThemeDefinition[] = [
  {
    id: 'neutral',
    name: 'Neutral',
    category: 'core',
    preview: {
      bg: '#1C1C1E',
      surface: '#2C2C2E',
      accent: '#0A84FF'
    },
    description: 'Apple HIG macOS Native'
  },
  {
    id: 'ocean',
    name: 'Ocean',
    category: 'core',
    preview: {
      bg: '#0F1729',
      surface: '#16203A',
      accent: '#4C8DFF'
    },
    description: 'Deep Oceanic Blue'
  },
  {
    id: 'dracula',
    name: 'Dracula',
    category: 'community',
    preview: {
      bg: '#282a36',
      surface: '#343746',
      accent: '#bd93f9'
    },
    description: 'Official Dracula Theme'
  },
  {
    id: 'one-dark',
    name: 'Atom One Dark',
    category: 'community',
    preview: {
      bg: '#21252b',
      surface: '#282c34',
      accent: '#61afef'
    },
    description: 'Atom / One Dark Standard'
  },
  {
    id: 'catppuccin',
    name: 'Catppuccin Mocha',
    category: 'community',
    preview: {
      bg: '#181825',
      surface: '#1e1e2e',
      accent: '#cba6f7'
    },
    description: 'Warm Community Dark Palette'
  },
  {
    id: 'nord',
    name: 'Nord',
    category: 'community',
    preview: {
      bg: '#242933',
      surface: '#3b4252',
      accent: '#88c0d0'
    },
    description: 'Arctic Ice Studio North-bluish'
  },
  {
    id: 'tokyo-night',
    name: 'Tokyo Night',
    category: 'community',
    preview: {
      bg: '#16161e',
      surface: '#24283b',
      accent: '#7aa2f7'
    },
    description: 'Modern Cyber Dark'
  },
  {
    id: 'gruvbox',
    name: 'Gruvbox Dark',
    category: 'community',
    preview: {
      bg: '#1d2021',
      surface: '#32302f',
      accent: '#fe8019'
    },
    description: 'Warm Retro Groove'
  }
];

class ThemeStore {
  palette = $state<ThemePalette>('neutral');
  mode = $state<ThemeMode>('system');
  actualDark = $state<boolean>(false);

  constructor() {
    if (typeof window !== 'undefined') {
      const savedPalette = localStorage.getItem('fuzznexus-theme-palette') as ThemePalette;
      if (THEMES.some((t) => t.id === savedPalette)) {
        this.palette = savedPalette;
      }

      const savedMode = localStorage.getItem('fuzznexus-theme-mode') as ThemeMode;
      if (savedMode === 'system' || savedMode === 'light' || savedMode === 'dark') {
        this.mode = savedMode;
      }

      const mql = window.matchMedia('(prefers-color-scheme: dark)');
      this.actualDark = this.mode === 'dark' || (this.mode === 'system' && mql.matches);

      mql.addEventListener('change', (e) => {
        if (this.mode === 'system') {
          this.actualDark = e.matches;
          this.apply();
        }
      });

      try {
        const w = window as unknown as { runtime?: { EventsOn?: (event: string, callback: (isDark: boolean) => void) => void } };
        if (w.runtime?.EventsOn) {
          w.runtime.EventsOn('macos:theme-changed', (isDark: boolean) => {
            if (this.mode === 'system') {
              this.actualDark = isDark;
              this.apply();
            }
          });
        }
      } catch {
        // Ignore in testing environments
      }

      this.apply();
    }
  }

  get currentTheme(): ThemeDefinition {
    return THEMES.find((t) => t.id === this.palette) || THEMES[0];
  }

  setPalette(p: ThemePalette) {
    if (!THEMES.some((t) => t.id === p)) return;
    this.palette = p;
    localStorage.setItem('fuzznexus-theme-palette', p);
    this.apply();
  }

  setMode(m: ThemeMode) {
    this.mode = m;
    localStorage.setItem('fuzznexus-theme-mode', m);
    if (m === 'system') {
      this.actualDark =
        typeof window !== 'undefined' && typeof window.matchMedia === 'function'
          ? window.matchMedia('(prefers-color-scheme: dark)').matches
          : false;
    } else {
      this.actualDark = m === 'dark';
    }
    this.apply();
  }

  toggleMode() {
    if (this.mode === 'light') this.setMode('dark');
    else if (this.mode === 'dark') this.setMode('system');
    else this.setMode('light');
  }

  apply() {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    root.setAttribute('data-theme', this.palette);
    root.setAttribute('data-mode', this.actualDark ? 'dark' : 'light');

    if (typeof window !== 'undefined') {
      try {
        const goApp = (window as unknown as { go?: { main?: { App?: { SetAppDockTheme?: (isDark: boolean) => void } } } })?.go?.main?.App;
        if (goApp?.SetAppDockTheme) {
          goApp.SetAppDockTheme(this.actualDark);
        }
      } catch {
        // Safe fallback in test or headless environments
      }
    }
  }
}

export const themeStore = new ThemeStore();
