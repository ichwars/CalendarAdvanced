import { useCallback, useEffect, useState } from 'react';
import { translations, type Locale, type TranslationKey } from './i18nTranslations';

const storageKey = 'calendaradvanced.locale';
let activeLocale: Locale = readLocale();
const listeners = new Set<() => void>();

function readLocale(): Locale {
  const stored = localStorage.getItem(storageKey);
  if (stored === 'de' || stored === 'en') {
    return stored;
  }
  return navigator.language.toLowerCase().startsWith('de') ? 'de' : 'en';
}

export function t(key: TranslationKey): string {
  return translations[activeLocale][key] ?? translations.en[key] ?? key;
}

export function setActiveLocale(next: Locale): void {
  activeLocale = next;
  localStorage.setItem(storageKey, next);
  document.documentElement.lang = next;
  listeners.forEach((listener) => listener());
}

export function useI18n() {
  const [locale, setLocaleState] = useState<Locale>(activeLocale);
  useEffect(() => {
    const listener = () => setLocaleState(activeLocale);
    listeners.add(listener);
    return () => { listeners.delete(listener); };
  }, []);
  const setLocale = useCallback(setActiveLocale, []);
  return { locale, setLocale, t };
}
