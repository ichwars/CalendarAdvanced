export type CalendarDensityPreference = 'comfortable' | 'compact' | 'dense';
export type CalendarViewPreference = 'day' | 'week' | 'month' | 'agenda';
export type DateFormatPreference = 'de' | 'iso';
export type DefaultEventDurationPreference = '30' | '45' | '60' | '90';
export type HolidayRegionPreference =
  | 'DE'
  | 'DE-BB'
  | 'DE-BE'
  | 'DE-BW'
  | 'DE-BY'
  | 'DE-HB'
  | 'DE-HE'
  | 'DE-HH'
  | 'DE-MV'
  | 'DE-NI'
  | 'DE-NW'
  | 'DE-RP'
  | 'DE-SH'
  | 'DE-SL'
  | 'DE-SN'
  | 'DE-ST'
  | 'DE-TH';
export type HolidayHighlightRegionPreference = '' | Exclude<HolidayRegionPreference, 'DE'>;
export type LocalePreference = 'de' | 'en';
export type StartPagePreference = 'overview' | 'calendar' | 'events';
export type ThemePreference = 'dark' | 'light' | 'system';
export type TimeGridPreference = '5' | '10' | '15' | '30';
export type WeekStartPreference = 'monday' | 'sunday';

export interface GeneralPreferences {
  calendarDensity: CalendarDensityPreference;
  compactMode: boolean;
  dateFormat: DateFormatPreference;
  defaultCalendarView: CalendarViewPreference;
  defaultEventDuration: DefaultEventDurationPreference;
  highlightedHolidayRegion: HolidayHighlightRegionPreference;
  holidayRegion: HolidayRegionPreference;
  locale: LocalePreference;
  rememberLastRoute: boolean;
  showHolidays: boolean;
  showWeekends: boolean;
  startPage: StartPagePreference;
  timeFormat24h: boolean;
  timeGrid: TimeGridPreference;
  theme: ThemePreference;
  timezone: string;
  weekStart: WeekStartPreference;
  workingHoursEnd: string;
  workingHoursStart: string;
}

const storageKey = 'calendaradvanced.generalPreferences';

export const defaultGeneralPreferences: GeneralPreferences = {
  calendarDensity: 'comfortable',
  compactMode: false,
  dateFormat: 'de',
  defaultCalendarView: 'week',
  defaultEventDuration: '60',
  highlightedHolidayRegion: '',
  holidayRegion: 'DE',
  locale: navigator.language.toLowerCase().startsWith('de') ? 'de' : 'en',
  rememberLastRoute: true,
  showHolidays: true,
  showWeekends: true,
  startPage: 'overview',
  timeFormat24h: true,
  timeGrid: '15',
  theme: 'dark',
  timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Europe/Berlin',
  weekStart: 'monday',
  workingHoursEnd: '17:00',
  workingHoursStart: '08:00'
};

export function getGeneralPreferences(): GeneralPreferences {
  const stored = localStorage.getItem(storageKey);
  if (!stored) {
    return defaultGeneralPreferences;
  }
  try {
    return normalizeGeneralPreferences(JSON.parse(stored) as Partial<GeneralPreferences>);
  } catch {
    return defaultGeneralPreferences;
  }
}

export function setGeneralPreferences(next: GeneralPreferences): void {
  const normalized = normalizeGeneralPreferences(next);
  localStorage.setItem(storageKey, JSON.stringify(normalized));
  applyGeneralPreferences(normalized);
}

export function normalizeGeneralPreferences(input: Partial<GeneralPreferences>): GeneralPreferences {
  return { ...defaultGeneralPreferences, ...input };
}

export function applyGeneralPreferences(preferences = getGeneralPreferences()): void {
  document.documentElement.dataset.compact = preferences.compactMode ? 'true' : 'false';
}

export function getLastRoute(): string | null {
  return localStorage.getItem('calendaradvanced.lastRoute');
}

export function setLastRoute(route: string): void {
  localStorage.setItem('calendaradvanced.lastRoute', route);
}

export function getAvailableTimezones(): string[] {
  const intlWithValues = Intl as typeof Intl & { supportedValuesOf?: (input: 'timeZone') => string[] };
  return intlWithValues.supportedValuesOf?.('timeZone') ?? [
    'Europe/Berlin',
    'Europe/London',
    'UTC',
    'America/New_York',
    'America/Los_Angeles'
  ];
}
