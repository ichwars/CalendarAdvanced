import { useEffect, useRef, useState, type CSSProperties } from 'react';
import { api } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { ConfirmDialog } from '../../shared/components/ConfirmDialog';
import { FormField } from '../../shared/components/FormField';
import { Icon } from '../../shared/components/Icon';
import { SelectField } from '../../shared/components/SelectField';
import { TimePicker } from '../../shared/components/TimePicker';
import { getHolidayHighlightRegionOptions, getHolidayRegionOptions } from '../../shared/holidays';
import { getAvailableTimezones, getGeneralPreferences, normalizeGeneralPreferences, setGeneralPreferences, type CalendarDensityPreference, type CalendarViewPreference, type DateFormatPreference, type DefaultEventDurationPreference, type GeneralPreferences, type HolidayHighlightRegionPreference, type HolidayRegionPreference, type StartPagePreference, type TimeGridPreference, type WeekStartPreference } from '../../shared/preferences';
import { getTheme, setTheme, type ThemeName } from '../../shared/theme';
import { useI18n } from '../../shared/i18n';
import type { Locale } from '../../shared/i18nTranslations';
import type { Calendar } from '../../shared/types';

const calendarColors = ['#ffcd00', '#6d8cff', '#22c55e', '#f97316', '#ec4899', '#a855f7', '#1f2933', '#8a5a2b'];
type SettingsTab = 'general' | 'calendars';

export function SettingsPage() {
  const { t, locale, setLocale } = useI18n();
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const [themeChoice, setThemeChoice] = useState<ThemeName>(getTheme());
  const [preferences, setPreferencesState] = useState<GeneralPreferences>(() => getGeneralPreferences());
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [calendarName, setCalendarName] = useState('');
  const [calendarColor, setCalendarColor] = useState(calendarColors[0]);
  const [calendarReminderEnabled, setCalendarReminderEnabled] = useState(false);
  const [calendarReminderDaysBefore, setCalendarReminderDaysBefore] = useState('1');
  const [calendarReminderTime, setCalendarReminderTime] = useState('09:00');
  const [calendarSameDayReminderTime, setCalendarSameDayReminderTime] = useState('09:00');
  const [editingCalendarId, setEditingCalendarId] = useState<number | null>(null);
  const [deleteCalendarTarget, setDeleteCalendarTarget] = useState<Calendar | null>(null);
  const [deletingCalendar, setDeletingCalendar] = useState(false);
  const [calendarSaveState, setCalendarSaveState] = useState<'idle' | 'error'>('idle');
  const [saveState, setSaveState] = useState<'idle' | 'saved' | 'error'>('idle');
  const saveRequestRef = useRef(0);
  const saveToastTimerRef = useRef(0);
  const timezones = getAvailableTimezones();
  const themeOptions = [
    { value: 'dark', label: t('settings.dark') },
    { value: 'light', label: t('settings.light') },
    { value: 'system', label: t('settings.system') }
  ] satisfies { value: ThemeName; label: string }[];
  const localeOptions = [
    { value: 'de', label: t('language.de') },
    { value: 'en', label: t('language.en') }
  ] satisfies { value: Locale; label: string }[];
  const calendarViewOptions = [
    { value: 'day', label: t('calendar.view.day') },
    { value: 'week', label: t('calendar.view.week') },
    { value: 'month', label: t('calendar.view.month') },
    { value: 'agenda', label: t('calendar.view.agenda') }
  ] satisfies { value: CalendarViewPreference; label: string }[];
  const calendarDensityOptions = [
    { value: 'comfortable', label: t('settings.calendarDensityComfortable') },
    { value: 'compact', label: t('settings.calendarDensityCompact') },
    { value: 'dense', label: t('settings.calendarDensityDense') }
  ] satisfies { value: CalendarDensityPreference; label: string }[];
  const weekStartOptions = [
    { value: 'monday', label: t('settings.weekStartMonday') },
    { value: 'sunday', label: t('settings.weekStartSunday') }
  ] satisfies { value: WeekStartPreference; label: string }[];
  const defaultEventDurationOptions = [
    { value: '30', label: t('settings.defaultEventDuration30') },
    { value: '45', label: t('settings.defaultEventDuration45') },
    { value: '60', label: t('settings.defaultEventDuration60') },
    { value: '90', label: t('settings.defaultEventDuration90') }
  ] satisfies { value: DefaultEventDurationPreference; label: string }[];
  const timeGridOptions = [
    { value: '5', label: t('settings.timeGrid5') },
    { value: '10', label: t('settings.timeGrid10') },
    { value: '15', label: t('settings.timeGrid15') },
    { value: '30', label: t('settings.timeGrid30') }
  ] satisfies { value: TimeGridPreference; label: string }[];
  const dateFormatOptions = [
    { value: 'de', label: t('settings.dateFormatDe') },
    { value: 'iso', label: t('settings.dateFormatIso') }
  ] satisfies { value: DateFormatPreference; label: string }[];
  const startPageOptions = [
    { value: 'overview', label: t('app.overview') },
    { value: 'calendar', label: t('nav.calendar') },
    { value: 'events', label: t('nav.events') }
  ] satisfies { value: StartPagePreference; label: string }[];
  const holidayRegionOptions = getHolidayRegionOptions() satisfies { value: HolidayRegionPreference; label: string }[];
  const holidayHighlightRegionOptions = getHolidayHighlightRegionOptions(t('settings.noHolidayHighlight')) satisfies { value: HolidayHighlightRegionPreference; label: string }[];
  const timezoneOptions = timezones.map((timezone) => ({ value: timezone, label: timezone }));
  const activeCalendarCount = calendars.filter((calendar) => calendar.visible).length;
  const settingsSummary = [
    { key: 'theme', label: t('settings.theme'), value: themeOptions.find((option) => option.value === themeChoice)?.label ?? themeChoice },
    { key: 'language', label: t('settings.language'), value: localeOptions.find((option) => option.value === locale)?.label ?? locale },
    { key: 'timezone', label: t('common.timezone'), value: preferences.timezone },
    { key: 'calendars', label: t('settings.manageCalendars'), value: t('settings.calendarCount').replace('{count}', String(activeCalendarCount)) }
  ];

  function updateTheme(next: ThemeName) {
    setThemeChoice(next);
    setTheme(next);
    updatePreferences({ theme: next });
  }

  function updateLocale(next: Locale) {
    setLocale(next);
    updatePreferences({ locale: next });
  }

  function updatePreferences(next: Partial<GeneralPreferences>) {
    const updated = { ...preferences, ...next };
    setPreferencesState(updated);
    setGeneralPreferences(updated);
    void savePreferences(updated);
  }

  async function savePreferences(updated: GeneralPreferences, silent = false) {
    const requestId = saveRequestRef.current + 1;
    saveRequestRef.current = requestId;
    window.clearTimeout(saveToastTimerRef.current);
    try {
      const result = await api.savePreferences(updated);
      if (requestId !== saveRequestRef.current) return;
      const normalized = normalizeGeneralPreferences(result.preferences);
      setPreferencesState(normalized);
      setGeneralPreferences(normalized);
      if (!silent) {
        setSaveState('saved');
      }
    } catch {
      if (requestId !== saveRequestRef.current) return;
      if (!silent) {
        setSaveState('error');
      }
    }
    if (!silent) {
      saveToastTimerRef.current = window.setTimeout(() => setSaveState('idle'), 3600);
    }
  }

  async function loadCalendars() {
    const response = await api.calendars();
    setCalendars(response.items);
  }

  function resetCalendarForm() {
    setCalendarName('');
    setCalendarColor(calendarColors[0]);
    setCalendarReminderEnabled(false);
    setCalendarReminderDaysBefore('1');
    setCalendarReminderTime('09:00');
    setCalendarSameDayReminderTime('09:00');
    setEditingCalendarId(null);
    setCalendarSaveState('idle');
  }

  function editCalendar(calendar: Calendar) {
    setActiveTab('calendars');
    setCalendarName(calendar.name);
    setCalendarColor(calendar.color);
    setCalendarReminderEnabled(calendar.reminderEnabled);
    setCalendarReminderDaysBefore(String(calendar.reminderDaysBefore || 1));
    setCalendarReminderTime(calendar.reminderTime || '09:00');
    setCalendarSameDayReminderTime(calendar.sameDayReminderTime || '09:00');
    setEditingCalendarId(calendar.id);
    setCalendarSaveState('idle');
  }

  async function saveCalendar() {
    const name = calendarName.trim();
    if (!name) {
      setCalendarSaveState('error');
      return;
    }
    const body = {
      color: calendarColor,
      name,
      reminderDaysBefore: Math.max(1, Number(calendarReminderDaysBefore) || 1),
      reminderEnabled: calendarReminderEnabled,
      reminderTime: calendarReminderTime,
      sameDayReminderTime: calendarSameDayReminderTime,
      timezone: preferences.timezone,
      visible: true
    };
    try {
      if (editingCalendarId) {
        await api.updateCalendar(editingCalendarId, body);
      } else {
        await api.createCalendar(body);
      }
      await loadCalendars();
      resetCalendarForm();
    } catch {
      setCalendarSaveState('error');
    }
  }

  async function confirmDeleteCalendar() {
    if (!deleteCalendarTarget) {
      return;
    }
    setDeletingCalendar(true);
    try {
      await api.deleteCalendar(deleteCalendarTarget.id);
      await loadCalendars();
      if (editingCalendarId === deleteCalendarTarget.id) {
        resetCalendarForm();
      }
      setDeleteCalendarTarget(null);
    } catch {
      setCalendarSaveState('error');
    } finally {
      setDeletingCalendar(false);
    }
  }

  useEffect(() => {
    let mounted = true;
    api.preferences().then(async (result) => {
      if (!mounted) return;
      if (result.persisted) {
        const normalized = normalizeGeneralPreferences(result.preferences);
        setPreferencesState(normalized);
        setGeneralPreferences(normalized);
        setThemeChoice(normalized.theme);
        setTheme(normalized.theme);
        setLocale(normalized.locale);
        return;
      }
      await savePreferences(getGeneralPreferences(), true);
    }).catch(() => undefined);
    return () => {
      mounted = false;
      window.clearTimeout(saveToastTimerRef.current);
    };
  }, []);

  useEffect(() => {
    void loadCalendars().catch(() => undefined);
  }, []);

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>{t('settings.title')}</h1>
          <p>{t('settings.subtitle')}</p>
        </div>
      </header>
      <div className="settings-summary">
        {settingsSummary.map((item) => (
          <span key={item.key}>
            <strong>{item.value}</strong>
            {item.label}
          </span>
        ))}
      </div>
      <div className="settings-tabs" role="tablist" aria-label={t('settings.tabs')}>
        <button aria-selected={activeTab === 'general'} className="settings-tab" onClick={() => setActiveTab('general')} role="tab" type="button">{t('settings.general')}</button>
        <button aria-selected={activeTab === 'calendars'} className="settings-tab" onClick={() => setActiveTab('calendars')} role="tab" type="button">{t('settings.manageCalendars')} <span>{activeCalendarCount}</span></button>
      </div>

      {activeTab === 'general' ? (
      <div className="settings-grid settings-grid--tiles" role="tabpanel">
          <section className="settings-section settings-tile">
            <header>
              <h2>{t('settings.display')}</h2>
              <p>{t('settings.displayDescription')}</p>
            </header>
            <div className="grid-form">
              <SelectField label={t('settings.theme')} value={themeChoice} onChange={updateTheme} options={themeOptions} />
              <SelectField label={t('settings.language')} value={locale} onChange={updateLocale} options={localeOptions} />
              <label className="check check--switch"><input type="checkbox" checked={preferences.compactMode} onChange={(event) => updatePreferences({ compactMode: event.currentTarget.checked })} />{t('settings.compactMode')}</label>
            </div>
          </section>

          <section className="settings-section settings-tile">
            <header>
              <h2>{t('settings.holidays')}</h2>
              <p>{t('settings.holidaysDescription')}</p>
            </header>
            <div className="holiday-settings-grid">
              <SelectField label={t('settings.holidayRegion')} value={preferences.holidayRegion} onChange={(value) => updatePreferences({ holidayRegion: value })} options={holidayRegionOptions} />
              <SelectField label={t('settings.highlightedHolidayRegion')} value={preferences.highlightedHolidayRegion} onChange={(value) => updatePreferences({ highlightedHolidayRegion: value })} options={holidayHighlightRegionOptions} />
              <label className="check check--switch holiday-settings-grid__toggle"><input type="checkbox" checked={preferences.showHolidays} onChange={(event) => updatePreferences({ showHolidays: event.currentTarget.checked })} />{t('settings.showHolidays')}</label>
            </div>
          </section>

          <section className="settings-section settings-tile">
            <header>
              <h2>{t('settings.dateTime')}</h2>
              <p>{t('settings.dateTimeDescription')}</p>
            </header>
            <div className="grid-form">
              <SelectField label={t('settings.dateFormat')} value={preferences.dateFormat} onChange={(value) => updatePreferences({ dateFormat: value })} options={dateFormatOptions} />
              <label className="check check--switch"><input type="checkbox" checked={preferences.timeFormat24h} onChange={(event) => updatePreferences({ timeFormat24h: event.currentTarget.checked })} />{t('settings.timeFormat24h')}</label>
            </div>
          </section>

          <section className="settings-section settings-tile">
            <header>
              <h2>{t('settings.startBehavior')}</h2>
              <p>{t('settings.startBehaviorDescription')}</p>
            </header>
            <div className="grid-form">
              <SelectField label={t('settings.startPage')} value={preferences.startPage} onChange={(value) => updatePreferences({ startPage: value })} options={startPageOptions} />
              <label className="check check--switch"><input type="checkbox" checked={preferences.rememberLastRoute} onChange={(event) => updatePreferences({ rememberLastRoute: event.currentTarget.checked })} />{t('settings.rememberLastRoute')}</label>
            </div>
          </section>
      </div>
      ) : (
      <div className="settings-grid settings-grid--tiles" role="tabpanel">
          <section className="settings-section settings-tile settings-tile--wide">
            <header>
              <h2>{t('settings.manageCalendars')}</h2>
              <p>{t('settings.manageCalendarsDescription')}</p>
            </header>
            <div className="calendar-manage-form">
              <FormField label={t('common.name')} value={calendarName} onChange={(event) => setCalendarName(event.currentTarget.value)} />
              <div className="field">
                <span>{t('common.color')}</span>
                <div className="color-swatch-row" role="radiogroup" aria-label={t('common.color')}>
                  {calendarColors.map((color) => (
                    <button
                      aria-checked={calendarColor === color}
                      aria-label={color}
                      className={calendarColor === color ? 'color-swatch active' : 'color-swatch'}
                      key={color}
                      onClick={() => setCalendarColor(color)}
                      role="radio"
                      style={{ '--swatch-color': color } as CSSProperties}
                      type="button"
                    />
                  ))}
                </div>
              </div>
              <label className="check check--switch calendar-manage-form__toggle"><input checked={calendarReminderEnabled} onChange={(event) => setCalendarReminderEnabled(event.currentTarget.checked)} type="checkbox" />{t('settings.calendarReminderEnabled')}</label>
              <div className="calendar-manage-form__reminder">
                <FormField disabled={!calendarReminderEnabled} label={t('settings.calendarReminderDaysBefore')} min="1" onChange={(event) => setCalendarReminderDaysBefore(event.currentTarget.value)} type="number" value={calendarReminderDaysBefore} />
                <label className="field">
                  <span>{t('settings.calendarReminderTime')}</span>
                  <TimePicker ariaLabel={t('settings.calendarReminderTime')} disabled={!calendarReminderEnabled} minuteStep={5} onChange={setCalendarReminderTime} value={calendarReminderTime} />
                </label>
                <label className="field">
                  <span>{t('settings.calendarSameDayReminderTime')}</span>
                  <TimePicker ariaLabel={t('settings.calendarSameDayReminderTime')} disabled={!calendarReminderEnabled} minuteStep={5} onChange={setCalendarSameDayReminderTime} value={calendarSameDayReminderTime} />
                </label>
              </div>
              <div className="button-row calendar-manage-form__actions">
                <Button type="button" onClick={() => void saveCalendar()}>{editingCalendarId ? t('settings.calendarSave') : t('settings.calendarAdd')}</Button>
                {editingCalendarId && <Button type="button" variant="ghost" onClick={resetCalendarForm}>{t('common.cancel')}</Button>}
              </div>
            </div>
            {calendarSaveState === 'error' && <p className="error">{t('settings.calendarSaveFailed')}</p>}
            <div className="calendar-list">
              {calendars.length === 0 ? (
                <div className="calendar-list__empty">
                  <strong>{t('settings.noCalendars')}</strong>
                  <p>{t('settings.noCalendarsDescription')}</p>
                </div>
              ) : calendars.map((calendar) => (
                <article className="calendar-list__item" key={calendar.id}>
                  <span className="event-dot" style={{ background: calendar.color }} />
                  <div>
                    <strong>{calendar.name}</strong>
                    <p>{calendar.reminderEnabled ? t('settings.calendarReminderSummary').replace('{days}', String(calendar.reminderDaysBefore || 1)).replace('{time}', calendar.reminderTime || '09:00').replace('{sameDayTime}', calendar.sameDayReminderTime || '09:00') : t('settings.calendarReminderOff')}</p>
                  </div>
                  <button className="icon-button" type="button" onClick={() => editCalendar(calendar)} aria-label={t('common.edit')} title={t('common.edit')}><Icon name="pencil" /></button>
                  <button className="icon-button icon-button--danger" type="button" onClick={() => setDeleteCalendarTarget(calendar)} aria-label={t('common.delete')} title={t('common.delete')}><Icon name="trash" /></button>
                </article>
              ))}
            </div>
          </section>

          <section className="settings-section settings-tile">
            <header>
              <h2>{t('settings.calendarViews')}</h2>
              <p>{t('settings.calendarViewsDescription')}</p>
            </header>
            <div className="grid-form">
              <SelectField label={t('settings.defaultCalendarView')} value={preferences.defaultCalendarView} onChange={(value) => updatePreferences({ defaultCalendarView: value })} options={calendarViewOptions} />
              <SelectField label={t('settings.calendarDensity')} value={preferences.calendarDensity} onChange={(value) => updatePreferences({ calendarDensity: value })} options={calendarDensityOptions} />
              <SelectField label={t('settings.weekStart')} value={preferences.weekStart} onChange={(value) => updatePreferences({ weekStart: value })} options={weekStartOptions} />
              <SelectField label={t('common.timezone')} value={preferences.timezone} onChange={(value) => updatePreferences({ timezone: value })} options={timezoneOptions} />
              <label className="check check--switch"><input type="checkbox" checked={preferences.showWeekends} onChange={(event) => updatePreferences({ showWeekends: event.currentTarget.checked })} />{t('settings.showWeekends')}</label>
            </div>
          </section>

          <section className="settings-section settings-tile">
            <header>
              <h2>{t('settings.eventDefaults')}</h2>
              <p>{t('settings.eventDefaultsDescription')}</p>
            </header>
            <div className="grid-form">
              <SelectField label={t('settings.defaultEventDuration')} value={preferences.defaultEventDuration} onChange={(value) => updatePreferences({ defaultEventDuration: value })} options={defaultEventDurationOptions} />
              <SelectField label={t('settings.timeGrid')} value={preferences.timeGrid} onChange={(value) => updatePreferences({ timeGrid: value })} options={timeGridOptions} />
            </div>
          </section>

          <section className="settings-section settings-tile">
            <header>
              <h2>{t('settings.workingHours')}</h2>
              <p>{t('settings.workingHoursDescription')}</p>
            </header>
            <div className="time-range-field">
              <div className="time-range-field__controls">
                <label className="field">
                  <span>{t('settings.workingHoursStart')}</span>
                  <TimePicker ariaLabel={t('settings.workingHoursStart')} name="workingHoursStart" value={preferences.workingHoursStart} onChange={(value) => updatePreferences({ workingHoursStart: value })} />
                </label>
                <label className="field">
                  <span>{t('settings.workingHoursEnd')}</span>
                  <TimePicker ariaLabel={t('settings.workingHoursEnd')} name="workingHoursEnd" value={preferences.workingHoursEnd} onChange={(value) => updatePreferences({ workingHoursEnd: value })} />
                </label>
              </div>
            </div>
          </section>
      </div>
      )}
      {saveState !== 'idle' && (
        <div className={saveState === 'saved' ? 'settings-save-toast' : 'settings-save-toast settings-save-toast--error'} role="status">
          {saveState === 'saved' ? t('settings.saved') : t('settings.saveFailed')}
        </div>
      )}
      {deleteCalendarTarget && (
        <ConfirmDialog
          busy={deletingCalendar}
          message={t('settings.calendarDeleteConfirm').replace('{name}', deleteCalendarTarget.name)}
          onCancel={() => setDeleteCalendarTarget(null)}
          onConfirm={() => void confirmDeleteCalendar()}
          title={t('settings.calendarDeleteTitle')}
        />
      )}
    </div>
  );
}
