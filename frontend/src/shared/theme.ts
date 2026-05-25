export type ThemeName = 'dark' | 'light' | 'system';

const storageKey = 'calendaradvanced.theme';

export function applyInitialTheme(): void {
  const stored = (localStorage.getItem(storageKey) as ThemeName | null) ?? 'dark';
  setTheme(stored);
}

export function setTheme(theme: ThemeName): void {
  localStorage.setItem(storageKey, theme);
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  const resolved = theme === 'system' ? (prefersDark ? 'dark' : 'light') : theme;
  document.documentElement.dataset.theme = resolved;
  document.documentElement.dataset.themeChoice = theme;
}

export function getTheme(): ThemeName {
  return (localStorage.getItem(storageKey) as ThemeName | null) ?? 'dark';
}
