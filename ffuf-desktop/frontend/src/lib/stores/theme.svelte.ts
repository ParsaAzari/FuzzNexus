export type ThemePalette = 'neutral' | 'ocean';
export type ThemeMode = 'system' | 'light' | 'dark';

class ThemeStore {
  palette = $state<ThemePalette>('neutral');
  mode = $state<ThemeMode>('system');
  actualDark = $state<boolean>(false);

  constructor() {
    if (typeof window !== 'undefined') {
      const savedPalette = localStorage.getItem('ffuf-theme-palette') as ThemePalette;
      if (savedPalette === 'neutral' || savedPalette === 'ocean') {
        this.palette = savedPalette;
      }

      const savedMode = localStorage.getItem('ffuf-theme-mode') as ThemeMode;
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

      this.apply();
    }
  }

  setPalette(p: ThemePalette) {
    this.palette = p;
    localStorage.setItem('ffuf-theme-palette', p);
    this.apply();
  }

  setMode(m: ThemeMode) {
    this.mode = m;
    localStorage.setItem('ffuf-theme-mode', m);
    if (m === 'system') {
      this.actualDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
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
  }
}

export const themeStore = new ThemeStore();
