import { type CSSProperties, type Dispatch, type FormEvent, type SetStateAction, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { api } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { Card } from '../../shared/components/Card';
import { ConfirmDialog } from '../../shared/components/ConfirmDialog';
import { DatePicker, formatDateValue, parseDateValue } from '../../shared/components/DatePicker';
import { DateTimePicker } from '../../shared/components/DateTimePicker';
import { EmptyState } from '../../shared/components/EmptyState';
import { FormField } from '../../shared/components/FormField';
import { Icon, type IconName } from '../../shared/components/Icon';
import { RichTextField } from '../../shared/components/RichTextField';
import { SelectField } from '../../shared/components/SelectField';
import { addDaysToKey, dateFromKey, dateKeyInTimeZone, formatDateKeyLabel } from '../../shared/dates';
import { formatDateKey, getCalendarHolidaysInRange, getHolidayRegionLabel, type Holiday } from '../../shared/holidays';
import { useI18n } from '../../shared/i18n';
import { getGeneralPreferences, type CalendarDensityPreference, type GeneralPreferences, type WeekStartPreference } from '../../shared/preferences';
import { isoLocal, toDateInputValue } from '../../shared/validation/forms';
import type { Calendar, EventItem, Recurrence, TaskItem } from '../../shared/types';

type CalendarView = 'day' | 'week' | 'month' | 'agenda';
const calendarViews: CalendarView[] = ['day', 'week', 'month', 'agenda'];

interface EventDraft {
  date: Date;
  startsAt?: Date;
  endsAt?: Date;
  allDay?: boolean;
}

export function CalendarPage() {
  const { locale, t } = useI18n();
  const generalPreferences = useMemo(() => getGeneralPreferences(), []);
  const [view, setView] = useState<CalendarView>(() => getGeneralPreferences().defaultCalendarView);
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [tasks, setTasks] = useState<TaskItem[]>([]);
  const [date, setDate] = useState(new Date());
  const [viewMenuOpen, setViewMenuOpen] = useState(false);
  const [eventDraft, setEventDraft] = useState<EventDraft | null>(null);
  const [detailEvent, setDetailEvent] = useState<EventItem | null>(null);
  const [editingEvent, setEditingEvent] = useState<EventItem | null>(null);
  const viewMenuRef = useRef<HTMLDivElement>(null);
  const range = useMemo(() => buildRange(date, view, generalPreferences.weekStart), [date, view, generalPreferences.weekStart]);

  async function load() {
    const [calendarResponse, eventResponse, taskResponse] = await Promise.all([
      api.calendars(),
      api.events(new URLSearchParams({ from: range.from.toISOString(), to: range.to.toISOString(), expand: 'true' })),
      api.tasks(new URLSearchParams({ limit: '500' }))
    ]);
    setCalendars(calendarResponse.items);
    setEvents(eventResponse.items);
    setTasks(taskResponse.items);
  }

  useEffect(() => {
    void load();
  }, [range.from.toISOString(), range.to.toISOString()]);

  useEffect(() => {
    function closeOnOutsideInteraction(event: PointerEvent) {
      if (!viewMenuRef.current?.contains(event.target as Node)) {
        setViewMenuOpen(false);
      }
    }

    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setViewMenuOpen(false);
      }
    }

    document.addEventListener('pointerdown', closeOnOutsideInteraction);
    document.addEventListener('keydown', closeOnEscape);
    return () => {
      document.removeEventListener('pointerdown', closeOnOutsideInteraction);
      document.removeEventListener('keydown', closeOnEscape);
    };
  }, []);

  async function moveEvent(event: EventItem, targetDate: Date, keepOriginalTime = true) {
    const start = new Date(event.startsAt);
    const end = new Date(event.endsAt);
    const duration = end.getTime() - start.getTime();
    const nextStart = new Date(targetDate);
    if (keepOriginalTime) {
      nextStart.setHours(start.getHours(), start.getMinutes(), 0, 0);
    } else {
      nextStart.setSeconds(0, 0);
    }
    const nextEnd = new Date(nextStart.getTime() + duration);
    await api.updateEvent(event.id, { ...event, startsAt: nextStart.toISOString(), endsAt: nextEnd.toISOString() });
    await load();
  }

  const today = useMemo(() => startOfDay(new Date()), []);
  const rows = useMemo(() => buildCalendarRows(range.from, view, generalPreferences.showWeekends), [range.from.toISOString(), view, generalPreferences.showWeekends]);
  const timeGridDays = useMemo(
    () => buildVisibleDays(range.from, view, generalPreferences.showWeekends),
    [range.from.toISOString(), view, generalPreferences.showWeekends]
  );
  const holidays = useMemo(
    () => generalPreferences.showHolidays
      ? getCalendarHolidaysInRange(range.from, range.to, generalPreferences.holidayRegion, generalPreferences.highlightedHolidayRegion)
      : [],
    [
      range.from.toISOString(),
      range.to.toISOString(),
      generalPreferences.highlightedHolidayRegion,
      generalPreferences.holidayRegion,
      generalPreferences.showHolidays
    ]
  );
  const rangeTitle = formatRangeTitle(range, date, view, locale);
  const calendarTasks = useMemo(
    () => tasks.filter((task) => task.showInCalendar && task.dueAt && isWithinRange(new Date(task.dueAt), range.from, range.to)),
    [range.from.toISOString(), range.to.toISOString(), tasks]
  );

  function shiftDate(direction: -1 | 1) {
    const next = new Date(date);
    if (view === 'month') {
      next.setMonth(next.getMonth() + direction);
    } else if (view === 'day') {
      next.setDate(next.getDate() + direction);
    } else {
      next.setDate(next.getDate() + direction * 7);
    }
    setDate(next);
  }

  async function createEvent(body: Partial<EventItem>) {
    await api.createEvent(body);
    setEventDraft(null);
    await load();
  }

  async function updateEvent(event: EventItem, body: Partial<EventItem>) {
    const updated = await api.updateEvent(event.id, { ...event, ...body });
    setEditingEvent(null);
    setDetailEvent(updated);
    await load();
  }

  async function deleteEvent(event: EventItem) {
    await api.deleteEvent(event.id);
    setDetailEvent(null);
    setEditingEvent(null);
    await load();
  }

  return (
    <div className="page">
      <header className="page-header calendar-page-header">
        <div className="calendar-page-header__title">
          <h1>{t('calendar.title')}</h1>
          <p>{t('calendar.dropHint')}</p>
        </div>
        <div className="calendar-page-header__actions">
          <Button disabled={!calendars.length} onClick={() => setEventDraft({ date })} type="button">
            + {t('events.add')}
          </Button>
          <div className="view-menu" ref={viewMenuRef}>
            <button
              className="view-menu__trigger"
              type="button"
              aria-expanded={viewMenuOpen}
              aria-haspopup="menu"
              aria-label={t('calendar.viewMenu')}
              onClick={() => setViewMenuOpen((current) => !current)}
            >
              <Icon name={viewIcon(view)} />
              <span>{t(`calendar.view.${view}` as const)}</span>
              <Icon className="view-menu__chevron" name="chevron-down" />
            </button>
            {viewMenuOpen && (
              <div className="view-menu__panel" role="menu" aria-label={t('calendar.viewMenu')}>
                {calendarViews.map((item) => (
                  <button
                    aria-checked={view === item}
                    className={view === item ? 'view-menu__item active' : 'view-menu__item'}
                    key={item}
                    onClick={() => {
                      setView(item);
                      setViewMenuOpen(false);
                    }}
                    role="menuitemradio"
                    type="button"
                  >
                    <Icon name={viewIcon(item)} />
                    <span>{t(`calendar.view.${item}` as const)}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      </header>

      <div className="calendar-toolbar">
        <button className="calendar-toolbar__button" type="button" onClick={() => setDate(new Date())}>{t('calendar.today')}</button>
        <button className="calendar-toolbar__icon" type="button" onClick={() => shiftDate(-1)} aria-label={t('calendar.previous')} title={t('calendar.previous')}><Icon name="chevron-left" /></button>
        <button className="calendar-toolbar__icon" type="button" onClick={() => shiftDate(1)} aria-label={t('calendar.next')} title={t('calendar.next')}><Icon name="chevron-right" /></button>
        <h2 className="calendar-toolbar__title">{rangeTitle}</h2>
        <DatePicker ariaLabel={t('calendar.jumpToDate')} value={formatDateValue(date)} onChange={(value) => setDate(parseDateValue(value) ?? date)} />
      </div>

      {view === 'agenda' ? (
        <Agenda calendars={calendars} events={events} locale={locale} onOpenEvent={setDetailEvent} tasks={calendarTasks} />
      ) : view === 'day' || view === 'week' ? (
        <TimeGridCalendar
          calendars={calendars}
          days={timeGridDays}
          events={events}
          holidays={holidays}
          locale={locale}
          onMoveEvent={moveEvent}
          onCreateEvent={setEventDraft}
          onOpenEvent={setDetailEvent}
          preferences={generalPreferences}
          tasks={calendarTasks}
          today={today}
          view={view}
        />
      ) : (
        <CalendarTable
          calendars={calendars}
          events={events}
          holidays={holidays}
          locale={locale}
          tasks={calendarTasks}
          density={generalPreferences.calendarDensity}
          month={date.getMonth()}
          onMoveEvent={moveEvent}
          onCreateEvent={setEventDraft}
          onOpenEvent={setDetailEvent}
          rows={rows}
          today={today}
          view={view}
        />
      )}
      {eventDraft && (
        <EventDialog
          calendars={calendars}
          date={eventDraft.date}
          initialAllDay={eventDraft.allDay}
          initialEndsAt={eventDraft.endsAt}
          initialStartsAt={eventDraft.startsAt}
          onClose={() => setEventDraft(null)}
          onSave={createEvent}
          preferences={generalPreferences}
        />
      )}
      {detailEvent && !editingEvent && (
        <EventDetailDialog
          calendars={calendars}
          event={detailEvent}
          locale={locale}
          onClose={() => setDetailEvent(null)}
          onDelete={() => deleteEvent(detailEvent)}
          onEdit={() => setEditingEvent(detailEvent)}
        />
      )}
      {editingEvent && (
        <EventDialog
          calendars={calendars}
          date={new Date(editingEvent.startsAt)}
          event={editingEvent}
          onClose={() => setEditingEvent(null)}
          onSave={(body) => updateEvent(editingEvent, body)}
          preferences={generalPreferences}
        />
      )}
    </div>
  );
}

export function EventDialog({
  calendars,
  date,
  event,
  initialAllDay,
  initialEndsAt,
  initialStartsAt,
  onClose,
  onSave,
  preferences
}: {
  calendars: Calendar[];
  date: Date;
  event?: EventItem;
  initialAllDay?: boolean;
  initialEndsAt?: Date;
  initialStartsAt?: Date;
  onClose: () => void;
  onSave: (body: Partial<EventItem>) => Promise<void>;
  preferences: GeneralPreferences;
}) {
  const { t } = useI18n();
  const initialIsAllDay = event?.allDay ?? initialAllDay ?? false;
  const initialStartDate = event ? new Date(event.startsAt) : initialStartsAt ?? defaultEventStart(date, preferences);
  const initialEndDate = event
    ? new Date(event.endsAt)
    : initialEndsAt ?? (initialStartsAt ? addMinutes(initialStartsAt, Number(preferences.defaultEventDuration)) : defaultEventEnd(date, preferences));
  const [title, setTitle] = useState(event?.title ?? '');
  const [calendarId, setCalendarId] = useState(() => String(event?.calendarId ?? calendars[0]?.id ?? ''));
  const [location, setLocation] = useState(event?.location ?? '');
  const [description, setDescription] = useState(event?.description ?? '');
  const [birthdayYear, setBirthdayYear] = useState(event?.birthdayYear ? String(event.birthdayYear) : '');
  const [startsAt, setStartsAt] = useState(() => initialIsAllDay ? toAllDayStart(toDateInputValue(initialStartDate)) : toDateInputValue(initialStartDate));
  const [endsAt, setEndsAt] = useState(() => initialIsAllDay ? toDateInputValue(initialEndDate) : toDateInputValue(initialEndDate));
  const [recurrence, setRecurrence] = useState<Recurrence['frequency']>(event?.recurrence?.frequency ?? '');
  const [allDay, setAllDay] = useState(initialIsAllDay);
  const [isPrivate, setIsPrivate] = useState(event?.private ?? false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const calendarOptions = calendars.map((calendar) => ({ value: String(calendar.id), label: calendar.name }));
  const selectedCalendar = calendars.find((calendar) => String(calendar.id) === calendarId);
  const birthdayContext = isBirthdayContext(title, selectedCalendar);
  const recurrenceOptions = [
    { value: '', label: t('events.none') },
    { value: 'DAILY', label: t('events.daily') },
    { value: 'WEEKLY', label: t('events.weekly') },
    { value: 'MONTHLY', label: t('events.monthly') },
    { value: 'YEARLY', label: t('events.yearly') }
  ] satisfies { value: Recurrence['frequency']; label: string }[];

  useEffect(() => {
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        onClose();
      }
    }
    document.addEventListener('keydown', closeOnEscape);
    return () => document.removeEventListener('keydown', closeOnEscape);
  }, [onClose]);

  useEffect(() => {
    if (birthdayContext && recurrence === '') {
      setRecurrence('YEARLY');
    }
  }, [birthdayContext, recurrence]);

  async function submit(formEvent: FormEvent) {
    formEvent.preventDefault();
    setSaving(true);
    setError('');
    try {
      await onSave({
        calendarId: Number(calendarId),
        title,
        description,
        location,
        startsAt: isoLocal(startsAt),
        endsAt: isoLocal(endsAt),
        timezone: preferences.timezone,
        allDay,
        private: isPrivate,
        completed: event?.completed ?? false,
        birthdayYear: birthdayContext && birthdayYear ? Number(birthdayYear) : undefined,
        recurrence: recurrence ? { frequency: recurrence, interval: 1 } : undefined
      });
    } catch (err) {
      const message = (err as { message?: string }).message;
      setError(message === 'The request is invalid.' ? t('errors.validation') : message ?? t('common.error'));
      setSaving(false);
    }
  }

  function updateStartsAt(next: string) {
    const normalized = allDay ? toAllDayStart(next) : next;
    setStartsAt(normalized);
    if (allDay) {
      setEndsAt(toAllDayEnd(normalized));
    }
  }

  function updateAllDay(checked: boolean) {
    setAllDay(checked);
    if (checked) {
      const normalizedStart = toAllDayStart(startsAt);
      setStartsAt(normalizedStart);
      setEndsAt(toAllDayEnd(normalizedStart));
    }
  }

  return (
    <div className="modal-backdrop" onMouseDown={(event) => {
      if (event.target === event.currentTarget) {
        onClose();
      }
    }}>
      <section className="modal" role="dialog" aria-modal="true" aria-labelledby="event-dialog-title">
        <header className="modal__header">
          <h2 id="event-dialog-title">{event ? t('events.edit') : t('events.new')}</h2>
          <button className="icon-button" type="button" onClick={onClose} aria-label={t('common.cancel')} title={t('common.cancel')}>
            <Icon name="x" />
          </button>
        </header>
        <form className="grid-form modal__form" onSubmit={(event) => void submit(event)}>
          {error && <p className="error field--wide" role="alert">{error}</p>}
          {event?.recurrence && <p className="warning field--wide">{t('events.seriesEditHint')}</p>}
          <FormField label={t('common.title')} name="title" value={title} onChange={(event) => setTitle(event.currentTarget.value)} required />
          <SelectField label={t('calendar.title')} value={calendarId} onChange={setCalendarId} options={calendarOptions} />
          <FormField label={t('common.location')} name="location" value={location} onChange={(event) => setLocation(event.currentTarget.value)} />
          {birthdayContext && (
            <FormField
              label={t('events.birthdayYear')}
              max={new Date(startsAt).getFullYear()}
              min="1850"
              name="birthdayYear"
              onChange={(event) => setBirthdayYear(event.currentTarget.value)}
              type="number"
              value={birthdayYear}
            />
          )}
          <SelectField label={t('events.recurrence')} value={recurrence} onChange={setRecurrence} options={recurrenceOptions} />
          <DateTimePicker label={t('common.start')} minuteStep={Number(preferences.timeGrid)} name="startsAt" timeDisabled={allDay} value={startsAt} onChange={updateStartsAt} />
          <DateTimePicker disabled={allDay} label={t('common.end')} minuteStep={Number(preferences.timeGrid)} name="endsAt" value={endsAt} onChange={setEndsAt} />
          <RichTextField id="event-description" label={t('common.description')} value={description} onChange={setDescription} />
          <label className="check check--switch"><input type="checkbox" checked={allDay} onChange={(event) => updateAllDay(event.currentTarget.checked)} />{t('events.allDay')}</label>
          <label className="check check--switch"><input type="checkbox" checked={isPrivate} onChange={(event) => setIsPrivate(event.currentTarget.checked)} />{t('events.private')}</label>
          <div className="button-row modal__actions">
            <Button disabled={saving || !calendarId} type="submit">{saving || event ? t('common.save') : t('events.add')}</Button>
            <Button disabled={saving} onClick={onClose} type="button" variant="ghost">{t('common.cancel')}</Button>
          </div>
        </form>
      </section>
    </div>
  );
}

function EventDetailDialog({
  calendars,
  event,
  locale,
  onClose,
  onDelete,
  onEdit
}: {
  calendars: Calendar[];
  event: EventItem;
  locale: string;
  onClose: () => void;
  onDelete: () => Promise<void>;
  onEdit: () => void;
}) {
  const { t } = useI18n();
  const [deleting, setDeleting] = useState(false);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const calendar = calendars.find((item) => item.id === event.calendarId);
  const visual = getEventVisualMeta(event, calendar);
  const birthdayAge = getBirthdayAge(event, calendar);
  const isBirthday = birthdayAge !== null || visual?.icon === 'gift';
  const accentColor = visual?.color ?? calendar?.color ?? 'var(--accent)';
  const recurrenceLabel = event.recurrence ? getRecurrenceLabel(event.recurrence.frequency, t) : '';

  useEffect(() => {
    function closeOnEscape(keyEvent: KeyboardEvent) {
      if (keyEvent.key === 'Escape') {
        onClose();
      }
    }
    document.addEventListener('keydown', closeOnEscape);
    return () => document.removeEventListener('keydown', closeOnEscape);
  }, [onClose]);

  async function deleteEvent() {
    setDeleting(true);
    try {
      await onDelete();
    } catch {
      setDeleting(false);
    }
  }

  return (
    <div className="modal-backdrop" onMouseDown={(mouseEvent) => {
      if (mouseEvent.target === mouseEvent.currentTarget) {
        onClose();
      }
    }}>
      <section className="modal event-detail" role="dialog" aria-modal="true" aria-labelledby="event-detail-title">
        <header
          className={isBirthday ? 'event-detail__hero event-detail__hero--birthday' : 'event-detail__hero'}
          style={{ '--event-detail-accent': accentColor } as CSSProperties}
        >
          <div className="event-detail__actions" aria-label={t('events.detailActions')}>
            <button className="event-detail__action" type="button" onClick={onEdit} aria-label={t('common.edit')} title={t('common.edit')}>
              <Icon name="pencil" />
            </button>
            <button className="event-detail__action event-detail__action--danger" disabled={deleting} type="button" onClick={() => setConfirmDeleteOpen(true)} aria-label={t('common.delete')} title={t('common.delete')}>
              <Icon name="trash" />
            </button>
            <button className="event-detail__action" type="button" onClick={onClose} aria-label={t('common.close')} title={t('common.close')}>
              <Icon name="x" />
            </button>
          </div>
          <div className="event-detail__hero-icon">
            <Icon name={visual?.icon ?? (isBirthday ? 'gift' : 'calendar')} />
          </div>
          {birthdayAge !== null && <span className="event-detail__age">{birthdayAge}</span>}
        </header>

        <div className="event-detail__body">
          <div className="event-detail__title-row">
            <span className="event-detail__dot" style={{ background: calendar?.color }} />
            <div>
              <h2 id="event-detail-title">{event.title}</h2>
              <p>{formatDetailTime(event, locale)}</p>
              {birthdayAge !== null && <p>{birthdayAge}. Geburtstag</p>}
            </div>
          </div>

          <div className="event-detail__meta">
            <DetailMeta icon="calendar" label={calendar?.name ?? t('calendar.title')} />
            {event.location && <DetailMeta icon="car" label={event.location} />}
            {recurrenceLabel && <DetailMeta icon="clock" label={recurrenceLabel} />}
            <DetailMeta icon={event.private ? 'moon' : 'users'} label={event.private ? t('events.private') : t('events.public')} />
            {event.completed && <DetailMeta icon="check" label={t('events.completed')} />}
          </div>

          {event.description && (
            <div className="event-detail__description">
              <h3>{t('common.description')}</h3>
              <p>{event.description}</p>
            </div>
          )}
        </div>
      </section>
      {confirmDeleteOpen && (
        <ConfirmDialog
          busy={deleting}
          message={t('events.deleteConfirm')}
          onCancel={() => setConfirmDeleteOpen(false)}
          onConfirm={() => void deleteEvent()}
          title={t('events.deleteTitle')}
        />
      )}
    </div>
  );
}

function DetailMeta({ icon, label }: { icon: IconName; label: string }) {
  return (
    <div className="event-detail__meta-row">
      <Icon name={icon} />
      <span>{label}</span>
    </div>
  );
}

function TimeGridCalendar({
  calendars,
  days,
  events,
  holidays,
  locale,
  onCreateEvent,
  onMoveEvent,
  onOpenEvent,
  preferences,
  tasks,
  today,
  view
}: {
  calendars: Calendar[];
  days: Date[];
  events: EventItem[];
  holidays: Holiday[];
  locale: string;
  onCreateEvent: (draft: EventDraft) => void;
  onMoveEvent: (event: EventItem, targetDate: Date, keepOriginalTime?: boolean) => Promise<void>;
  onOpenEvent: (event: EventItem) => void;
  preferences: GeneralPreferences;
  tasks: TaskItem[];
  today: Date;
  view: 'day' | 'week';
}) {
  const { t } = useI18n();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [scrollbarGutter, setScrollbarGutter] = useState(0);
  const [timeSelection, setTimeSelection] = useState<{ anchor: Date; focus: Date } | null>(null);
  const [allDaySelection, setAllDaySelection] = useState<{ anchor: Date; focus: Date } | null>(null);
  const slotStep = Number(preferences.timeGrid);
  const workingStart = parseTimeToMinutes(preferences.workingHoursStart);
  const workingEnd = parseTimeToMinutes(preferences.workingHoursEnd);
  const slots = useMemo(() => buildTimeSlots(slotStep), [slotStep]);
  const minColumnWidth = view === 'day' ? 'minmax(18rem, 1fr)' : 'minmax(9.5rem, 1fr)';
  const gridTemplateColumns = `4.4rem repeat(${days.length}, ${minColumnWidth})`;
  const weekNumber = getISOWeek(days[0] ?? today);
  const fixedGridStyle = {
    gridTemplateColumns,
    width: scrollbarGutter ? `calc(100% - ${scrollbarGutter}px)` : undefined,
    '--time-scrollbar-gutter': `${scrollbarGutter}px`
  } as CSSProperties;
  const selectedTimeRange = timeSelection ? getSlotSelectionRange(timeSelection.anchor, timeSelection.focus, slotStep) : null;
  const selectedAllDayRange = allDaySelection ? getAllDaySelectionRange(allDaySelection.anchor, allDaySelection.focus) : null;

  useEffect(() => {
    const scrollElement = scrollRef.current;
    const target = scrollElement?.querySelector<HTMLElement>(`[data-time="${roundDownToStep(workingStart, slotStep)}"]`);
    if (!scrollElement || !target) return;
    scrollElement.scrollTop = Math.max(target.offsetTop - 28, 0);
  }, [days.length, slotStep, view, workingStart]);

  useLayoutEffect(() => {
    const scrollElement = scrollRef.current;
    if (!scrollElement) return undefined;
    const element = scrollElement;

    function syncScrollbarGutter() {
      setScrollbarGutter(element.offsetWidth - element.clientWidth);
    }

    syncScrollbarGutter();
    const resizeObserver = new ResizeObserver(syncScrollbarGutter);
    resizeObserver.observe(element);
    return () => resizeObserver.disconnect();
  }, [days.length, view]);

  return (
    <section className={`time-calendar time-calendar--${view} time-calendar--density-${preferences.calendarDensity}`}>
      <div className="time-calendar__header" style={fixedGridStyle}>
        <div className="time-calendar__corner">
          <span>KW</span>
          <strong>{weekNumber}</strong>
        </div>
        {days.map((day) => (
          <header
            className={isSameDay(day, today) ? 'time-calendar__day time-calendar__day--today' : 'time-calendar__day'}
            key={day.toISOString()}
          >
            <span>{formatWeekday(day, locale)}</span>
            <strong>{formatDay(day, locale)}</strong>
          </header>
        ))}
      </div>

      <div className="time-calendar__all-day" style={fixedGridStyle}>
        <div className="time-calendar__all-day-label">{t('events.allDay')}</div>
        {days.map((day) => {
          const dayEvents = events.filter((event) => event.allDay && eventStartDateKey(event) === formatDateKey(day));
          const dayTasks = tasks.filter((task) => task.dueAt && isSameDay(new Date(task.dueAt), day) && isAllDayTask(task));
          const dayHolidays = holidays.filter((holiday) => holiday.date === formatDateKey(day));
          const allDaySelected = selectedAllDayRange ? day >= selectedAllDayRange.start && day < selectedAllDayRange.end : false;
          return (
            <div
              className={allDaySelected ? 'time-calendar__all-day-events time-calendar__all-day-events--selected' : 'time-calendar__all-day-events'}
              key={day.toISOString()}
              onClick={(clickEvent) => {
                if ((clickEvent.target as HTMLElement).closest('.event-chip, .holiday-badge')) return;
                onCreateEvent({ date: day, startsAt: startOfDay(day), endsAt: addDays(startOfDay(day), 1), allDay: true });
              }}
              onMouseDown={(mouseEvent) => {
                if (mouseEvent.button !== 0 || (mouseEvent.target as HTMLElement).closest('.event-chip, .holiday-badge')) return;
                mouseEvent.preventDefault();
                setAllDaySelection({ anchor: startOfDay(day), focus: startOfDay(day) });
              }}
              onMouseEnter={() => {
                setAllDaySelection((current) => current ? { ...current, focus: startOfDay(day) } : current);
              }}
              onMouseUp={() => {
                setAllDaySelection((current) => {
                  if (!current) return null;
                  const next = getAllDaySelectionRange(current.anchor, startOfDay(day));
                  onCreateEvent({ date: next.start, startsAt: next.start, endsAt: next.end, allDay: true });
                  return null;
                });
              }}
            >
              {dayHolidays.map((holiday) => (
                <HolidayBadge holiday={holiday} key={holiday.date} />
              ))}
              {dayEvents.map((event) => (
                <EventChip calendars={calendars} event={event} key={`${event.id}-${event.startsAt}`} onOpenEvent={onOpenEvent} />
              ))}
              {dayTasks.map((task) => <TaskChip key={task.id} task={task} />)}
            </div>
          );
        })}
      </div>

      <div className="time-calendar__scroll" ref={scrollRef}>
        <div
          className="time-calendar__body"
          style={{
            gridTemplateColumns,
            gridTemplateRows: `repeat(${slots.length}, var(--time-slot-height))`
          }}
        >
          {slots.map((slot, slotIndex) => (
            <TimeSlotRow
              days={days}
              events={events}
              isWorkingTime={isWorkingTime(slot.minutes, workingStart, workingEnd)}
              key={slot.minutes}
              onCreateEvent={onCreateEvent}
              onMoveEvent={onMoveEvent}
              selectedRange={selectedTimeRange}
              setTimeSelection={setTimeSelection}
              slot={slot}
              slotStep={slotStep}
              slotIndex={slotIndex}
            />
          ))}
          {days.map((day, dayIndex) => (
            <div key={day.toISOString()} style={{ display: 'contents' }}>
              {events
                .filter((event) => !event.allDay && isSameDay(new Date(event.startsAt), day))
                .map((event) => {
                  const position = getEventGridPosition(event, slotStep);
                  return (
                  <article
                    className={event.completed ? 'time-calendar__event time-calendar__event--completed' : 'time-calendar__event'}
                    draggable
                    key={`${event.id}-${event.startsAt}`}
                    onClick={() => onOpenEvent(event)}
                    onDragStart={(dragEvent) => dragEvent.dataTransfer.setData('text/plain', String(event.id))}
                    onKeyDown={(keyEvent) => {
                      if (keyEvent.key === 'Enter' || keyEvent.key === ' ') {
                        keyEvent.preventDefault();
                        onOpenEvent(event);
                      }
                    }}
                    role="button"
                    style={{
                      borderColor: calendars.find((calendar) => calendar.id === event.calendarId)?.color,
                      gridColumn: dayIndex + 2,
                      gridRow: `${position.start} / span ${position.span}`
                    }}
                    tabIndex={0}
                  >
                    <EventVisuals calendar={calendars.find((calendar) => calendar.id === event.calendarId)} event={event} />
                    <strong>{event.title}</strong>
                    <small>{formatEventTime(event, locale)}</small>
                  </article>
                  );
              })}
              {tasks
                .filter((task) => task.dueAt && !isAllDayTask(task) && isSameDay(new Date(task.dueAt), day))
                .map((task) => {
                  const position = getTaskGridPosition(task, slotStep);
                  return (
                    <article
                      className={task.completed ? 'time-calendar__event time-calendar__event--task time-calendar__event--completed' : 'time-calendar__event time-calendar__event--task'}
                      key={`task-${task.id}`}
                      style={{
                        borderColor: 'var(--accent-2)',
                        gridColumn: dayIndex + 2,
                        gridRow: `${position.start} / span ${position.span}`
                      }}
                    >
                      <Icon name="check-square" />
                      <strong>{task.title}</strong>
                      <small>{new Date(task.dueAt ?? '').toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })}</small>
                    </article>
                  );
                })}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function TimeSlotRow({
  days,
  events,
  isWorkingTime,
  onCreateEvent,
  onMoveEvent,
  selectedRange,
  setTimeSelection,
  slot,
  slotStep,
  slotIndex
}: {
  days: Date[];
  events: EventItem[];
  isWorkingTime: boolean;
  onCreateEvent: (draft: EventDraft) => void;
  onMoveEvent: (event: EventItem, targetDate: Date, keepOriginalTime?: boolean) => Promise<void>;
  selectedRange: { start: Date; end: Date } | null;
  setTimeSelection: Dispatch<SetStateAction<{ anchor: Date; focus: Date } | null>>;
  slot: TimeSlot;
  slotStep: number;
  slotIndex: number;
}) {
  return (
    <>
      <div className="time-calendar__time" data-time={slot.minutes} style={{ gridColumn: 1, gridRow: slotIndex + 1 }}>
        {slot.showLabel ? slot.label : ''}
      </div>
      {days.map((day, dayIndex) => {
        const slotDate = addMinutes(startOfDay(day), slot.minutes);
        const selected = selectedRange ? slotDate >= selectedRange.start && slotDate < selectedRange.end : false;
        return (
          <div
            className={[
              'time-calendar__slot',
              isWorkingTime ? 'time-calendar__slot--work' : '',
              selected ? 'time-calendar__slot--selected' : ''
            ].filter(Boolean).join(' ')}
            key={`${day.toISOString()}-${slot.minutes}`}
            onDragOver={(event) => event.preventDefault()}
            onDrop={(dropEvent) => {
              const id = Number(dropEvent.dataTransfer.getData('text/plain'));
              const item = events.find((event) => event.id === id);
              if (item) void onMoveEvent(item, slotDate, false);
            }}
            onMouseDown={(mouseEvent) => {
              if (mouseEvent.button !== 0) return;
              mouseEvent.preventDefault();
              setTimeSelection({ anchor: slotDate, focus: slotDate });
            }}
            onMouseEnter={() => {
              setTimeSelection((current) => current ? { ...current, focus: slotDate } : current);
            }}
            onMouseUp={() => {
              setTimeSelection((current) => {
                const next = current ? getSlotSelectionRange(current.anchor, slotDate, slotStep) : { start: slotDate, end: addMinutes(slotDate, slotStep) };
                onCreateEvent({ date: next.start, startsAt: next.start, endsAt: next.end });
                return null;
              });
            }}
            style={{ gridColumn: dayIndex + 2, gridRow: slotIndex + 1 }}
          />
        );
      })}
    </>
  );
}

function EventChip({ calendars, event, onOpenEvent }: { calendars: Calendar[]; event: EventItem; onOpenEvent: (event: EventItem) => void }) {
  const calendar = calendars.find((item) => item.id === event.calendarId);
  return (
    <article
      className={event.completed ? 'event-chip event-chip--completed' : 'event-chip'}
      draggable
      onClick={(clickEvent) => {
        clickEvent.stopPropagation();
        onOpenEvent(event);
      }}
      onDragStart={(dragEvent) => dragEvent.dataTransfer.setData('text/plain', String(event.id))}
      onKeyDown={(keyEvent) => {
        if (keyEvent.key === 'Enter' || keyEvent.key === ' ') {
          keyEvent.preventDefault();
          onOpenEvent(event);
        }
      }}
      role="button"
      tabIndex={0}
    >
      <EventVisuals calendar={calendar} event={event} />
      <span className="event-dot" style={{ background: calendar?.color }} />
      <strong>{event.title}</strong>
      {!event.allDay && <small>{new Date(event.startsAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</small>}
    </article>
  );
}

function TaskChip({ task }: { task: TaskItem }) {
  return (
    <article className={task.completed ? 'event-chip event-chip--task event-chip--completed' : 'event-chip event-chip--task'} title={task.title}>
      <Icon name="check-square" />
      <span className="event-dot event-dot--task" />
      <strong>{task.title}</strong>
      {task.dueAt && <small>{new Date(task.dueAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</small>}
    </article>
  );
}

function EventVisuals({ calendar, event }: { calendar?: Calendar; event: EventItem }) {
  const visual = getEventVisualMeta(event, calendar);
  const birthdayAge = getBirthdayAge(event, calendar);
  if (!visual && !event.completed && birthdayAge === null) return null;
  return (
    <span className="event-visuals" aria-hidden="true">
      {visual && (
        <span className="event-visuals__icon" style={{ '--event-icon-color': visual.color } as CSSProperties} title={visual.label}>
          <Icon name={visual.icon} />
        </span>
      )}
      {birthdayAge !== null && <span className="event-visuals__age" title={`${birthdayAge}. Geburtstag`}>{birthdayAge}</span>}
      {event.completed && (
        <span className="event-visuals__done" title="Erledigt">
          <Icon name="check" />
        </span>
      )}
    </span>
  );
}

function HolidayBadge({ holiday }: { holiday: Holiday }) {
  const regionLabel = getHolidayRegionLabel(holiday.region);
  const title = holiday.highlighted ? `${regionLabel}: ${holiday.name}` : holiday.name;
  return (
    <article className={holiday.highlighted ? 'holiday-badge holiday-badge--highlighted' : 'holiday-badge'} title={title}>
      <span />
      <strong>{holiday.name}</strong>
      {holiday.highlighted && <em>{regionLabel}</em>}
    </article>
  );
}

function CalendarTable({
  calendars,
  density,
  events,
  holidays,
  locale,
  month,
  onCreateEvent,
  onMoveEvent,
  onOpenEvent,
  rows,
  tasks,
  today,
  view
}: {
  calendars: Calendar[];
  density: CalendarDensityPreference;
  events: EventItem[];
  holidays: Holiday[];
  locale: string;
  month: number;
  onCreateEvent: (draft: EventDraft) => void;
  onMoveEvent: (event: EventItem, targetDate: Date) => Promise<void>;
  onOpenEvent: (event: EventItem) => void;
  rows: CalendarRow[];
  tasks: TaskItem[];
  today: Date;
  view: 'month';
}) {
  return (
    <div className={`calendar-table calendar-table--${view} calendar-table--density-${density}`}>
      {rows.map((row, rowIndex) => (
        <div
          className="calendar-row"
          key={row.days[0].toISOString()}
          style={{
            gridTemplateColumns: `2rem repeat(${row.days.length}, minmax(0, 1fr))`,
            minWidth: row.days.length > 5 ? '760px' : '580px'
          }}
        >
          <div className="week-number" aria-label={`Kalenderwoche ${row.weekNumber}`}>{row.weekNumber}</div>
          {row.days.map((cell) => {
            const dayEvents = events.filter((event) => eventDateMatchesDay(event, cell));
            const dayTasks = tasks.filter((task) => task.dueAt && isSameDay(new Date(task.dueAt), cell));
            const dayHolidays = holidays.filter((holiday) => holiday.date === formatDateKey(cell));
            const isMuted = cell.getMonth() !== month;
            const isToday = isSameDay(cell, today);
            const showWeekday = rowIndex === 0;
            return (
              <section
                key={cell.toISOString()}
                className={[
                  'calendar-slot',
                  isMuted ? 'calendar-slot--muted' : '',
                  isToday ? 'calendar-slot--today' : ''
                ].filter(Boolean).join(' ')}
                onDragOver={(event) => event.preventDefault()}
                onDrop={(dropEvent) => {
                  const id = Number(dropEvent.dataTransfer.getData('text/plain'));
                  const item = events.find((event) => event.id === id);
                  if (item) void onMoveEvent(item, cell);
                }}
                onClick={(clickEvent) => {
                  if ((clickEvent.target as HTMLElement).closest('.event-chip, .holiday-badge')) return;
                  onCreateEvent({ date: cell, startsAt: startOfDay(cell), endsAt: addDays(startOfDay(cell), 1), allDay: true });
                }}
              >
                <header className="calendar-slot__header">
                  {showWeekday && <span>{formatWeekday(cell, locale)}</span>}
                  <strong>{formatDay(cell, locale)}</strong>
                </header>
                <div className="calendar-slot__events">
                  {dayHolidays.map((holiday) => (
                    <HolidayBadge holiday={holiday} key={holiday.date} />
                  ))}
                  {dayEvents.map((event) => (
                    <EventChip calendars={calendars} event={event} key={`${event.id}-${event.startsAt}`} onOpenEvent={onOpenEvent} />
                  ))}
                  {dayTasks.map((task) => <TaskChip key={task.id} task={task} />)}
                </div>
              </section>
            );
          })}
        </div>
      ))}
    </div>
  );
}

function Agenda({ calendars, events, locale, onOpenEvent, tasks }: { calendars: Calendar[]; events: EventItem[]; locale: string; onOpenEvent: (event: EventItem) => void; tasks: TaskItem[] }) {
  const { t } = useI18n();
  const items = [
    ...events.map((event) => ({ date: event.allDay ? dateFromKey(eventStartDateKey(event)) : new Date(event.startsAt), event, kind: 'event' as const })),
    ...tasks.filter((task) => task.dueAt).map((task) => ({ date: new Date(task.dueAt as string), task, kind: 'task' as const }))
  ].sort((left, right) => left.date.getTime() - right.date.getTime());
  if (!items.length) {
    return <EmptyState message={t('common.empty')} />;
  }
  return (
    <Card>
      <div className="agenda-list">
        {items.map((item) => {
          if (item.kind === 'task') {
            return (
              <article key={`task-${item.task.id}`} className={item.task.completed ? 'agenda-item agenda-item--task agenda-item--completed' : 'agenda-item agenda-item--task'}>
                <Icon name="check-square" />
                <div>
                  <strong>{item.task.title}</strong>
                  <p>{t('nav.tasks')}</p>
                </div>
                <time>{item.task.dueAt ? new Date(item.task.dueAt).toLocaleString() : ''}</time>
              </article>
            );
          }
          const event = item.event;
          return (
            <article
              key={`${event.id}-${event.startsAt}`}
              className={event.completed ? 'agenda-item agenda-item--completed' : 'agenda-item'}
              onClick={() => onOpenEvent(event)}
              onKeyDown={(keyEvent) => {
                if (keyEvent.key === 'Enter' || keyEvent.key === ' ') {
                  keyEvent.preventDefault();
                  onOpenEvent(event);
                }
              }}
              role="button"
              tabIndex={0}
            >
              <EventVisuals calendar={calendars.find((calendar) => calendar.id === event.calendarId)} event={event} />
              <div>
                <strong>{event.title}</strong>
                <p>{event.location}</p>
              </div>
              <time>{event.allDay ? formatDateKeyLabel(eventStartDateKey(event), locale) : new Date(event.startsAt).toLocaleString()}</time>
            </article>
          );
        })}
      </div>
    </Card>
  );
}

function buildRange(date: Date, view: CalendarView, weekStart: WeekStartPreference): { from: Date; to: Date } {
  const from = new Date(date);
  from.setHours(0, 0, 0, 0);
  if (view === 'day') return { from, to: addDays(from, 1) };
  if (view === 'month') {
    from.setDate(1);
    const gridStart = startOfWeek(from, weekStart);
    return { from: gridStart, to: addDays(gridStart, 42) };
  }
  const start = startOfWeek(from, weekStart);
  return { from: start, to: addDays(start, 7) };
}

function addDays(date: Date, days: number): Date {
  const next = new Date(date);
  next.setDate(next.getDate() + days);
  return next;
}

function addMinutes(date: Date, minutes: number): Date {
  const next = new Date(date);
  next.setMinutes(next.getMinutes() + minutes);
  return next;
}

function getSlotSelectionRange(anchor: Date, focus: Date, slotStep: number): { start: Date; end: Date } {
  const first = anchor.getTime() <= focus.getTime() ? anchor : focus;
  const last = anchor.getTime() <= focus.getTime() ? focus : anchor;
  return { start: first, end: addMinutes(last, slotStep) };
}

function getAllDaySelectionRange(anchor: Date, focus: Date): { start: Date; end: Date } {
  const first = anchor.getTime() <= focus.getTime() ? startOfDay(anchor) : startOfDay(focus);
  const last = anchor.getTime() <= focus.getTime() ? startOfDay(focus) : startOfDay(anchor);
  return { start: first, end: addDays(last, 1) };
}

function startOfDay(date: Date): Date {
  const next = new Date(date);
  next.setHours(0, 0, 0, 0);
  return next;
}

interface CalendarRow {
  weekNumber: number;
  days: Date[];
}

function buildCalendarRows(from: Date, view: CalendarView, showWeekends: boolean): CalendarRow[] {
  const rowCount = view === 'month' ? 6 : 0;
  return Array.from({ length: rowCount }, (_, index) => {
    const weekStart = addDays(from, index * 7);
    const days = Array.from({ length: 7 }, (_day, dayIndex) => addDays(weekStart, dayIndex));
    return {
      weekNumber: getISOWeek(weekStart),
      days: showWeekends ? days : days.filter((day) => !isWeekend(day))
    };
  });
}

function buildVisibleDays(from: Date, view: CalendarView, showWeekends: boolean): Date[] {
  if (view === 'day') {
    return [startOfDay(from)];
  }
  if (view !== 'week') {
    return [];
  }
  const days = Array.from({ length: 7 }, (_day, dayIndex) => addDays(from, dayIndex));
  return showWeekends ? days : days.filter((day) => !isWeekend(day));
}

function startOfWeek(date: Date, weekStart: WeekStartPreference): Date {
  const next = new Date(date);
  const day = next.getDay();
  const offset = weekStart === 'sunday' ? day : (day || 7) - 1;
  next.setDate(next.getDate() - offset);
  next.setHours(0, 0, 0, 0);
  return next;
}

function getISOWeek(date: Date): number {
  const target = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
  const dayNumber = target.getUTCDay() || 7;
  target.setUTCDate(target.getUTCDate() + 4 - dayNumber);
  const yearStart = new Date(Date.UTC(target.getUTCFullYear(), 0, 1));
  return Math.ceil((((target.getTime() - yearStart.getTime()) / 86400000) + 1) / 7);
}

function viewIcon(view: CalendarView) {
  return view === 'agenda' ? 'list' : 'calendar';
}

function formatRangeTitle(range: { from: Date; to: Date }, date: Date, view: CalendarView, locale: string): string {
  if (view === 'month') {
    return date.toLocaleDateString(locale, { month: 'long', year: 'numeric' });
  }
  if (view === 'day') {
    return date.toLocaleDateString(locale, { day: 'numeric', month: 'long', year: 'numeric' });
  }

  const start = range.from;
  const end = addDays(range.to, -1);
  if (start.getFullYear() === end.getFullYear() && start.getMonth() === end.getMonth()) {
    const monthYear = end.toLocaleDateString(locale, { month: 'long', year: 'numeric' });
    return `${start.getDate()}. - ${end.getDate()}. ${monthYear}`;
  }
  return `${start.toLocaleDateString(locale, { day: 'numeric', month: 'short' })} - ${end.toLocaleDateString(locale, { day: 'numeric', month: 'short', year: 'numeric' })}`;
}

function formatWeekday(date: Date, locale: string): string {
  return date.toLocaleDateString(locale, { weekday: 'short' }).replace('.', '').toUpperCase();
}

function formatDay(date: Date, locale: string): string {
  if (date.getDate() === 1) {
    return date.toLocaleDateString(locale, { day: 'numeric', month: 'short' });
  }
  return String(date.getDate());
}

interface TimeSlot {
  label: string;
  minutes: number;
  showLabel: boolean;
}

function buildTimeSlots(step: number): TimeSlot[] {
  const safeStep = [5, 10, 15, 30].includes(step) ? step : 15;
  return Array.from({ length: 1440 / safeStep }, (_slot, index) => {
    const minutes = index * safeStep;
    return {
      label: formatSlotLabel(minutes),
      minutes,
      showLabel: minutes % 60 === 0
    };
  });
}

function formatSlotLabel(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const minutePart = minutes % 60;
  return `${String(hours).padStart(2, '0')}:${String(minutePart).padStart(2, '0')}`;
}

function parseTimeToMinutes(value: string): number {
  const match = /^(\d{1,2}):(\d{2})$/.exec(value);
  if (!match) {
    return 0;
  }
  const hours = Math.min(Math.max(Number(match[1]), 0), 23);
  const minutes = Math.min(Math.max(Number(match[2]), 0), 59);
  return hours * 60 + minutes;
}

function roundDownToStep(minutes: number, step: number): number {
  return Math.floor(minutes / step) * step;
}

function isWorkingTime(minutes: number, workingStart: number, workingEnd: number): boolean {
  if (workingEnd <= workingStart) {
    return true;
  }
  return minutes >= workingStart && minutes < workingEnd;
}

function getEventGridPosition(event: EventItem, slotStep: number): { start: number; span: number } {
  const start = new Date(event.startsAt);
  const end = new Date(event.endsAt);
  const startMinutes = start.getHours() * 60 + start.getMinutes();
  const durationMinutes = Math.max((end.getTime() - start.getTime()) / 60000, slotStep);
  return {
    start: Math.max(1, Math.floor(startMinutes / slotStep) + 1),
    span: Math.max(1, Math.ceil(durationMinutes / slotStep))
  };
}

function getTaskGridPosition(task: TaskItem, slotStep: number): { start: number; span: number } {
  const dueAt = new Date(task.dueAt ?? '');
  const startMinutes = dueAt.getHours() * 60 + dueAt.getMinutes();
  return {
    start: Math.max(1, Math.floor(startMinutes / slotStep) + 1),
    span: 1
  };
}

function isAllDayTask(_task: TaskItem): boolean {
  return false;
}

function isWithinRange(date: Date, from: Date, to: Date): boolean {
  return date >= from && date < to;
}

function formatEventTime(event: EventItem, locale: string): string {
  if (event.allDay) {
    return '';
  }
  const start = new Date(event.startsAt);
  const end = new Date(event.endsAt);
  return `${start.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })} - ${end.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })}`;
}

function formatDetailTime(event: EventItem, locale: string): string {
  const start = new Date(event.startsAt);
  const end = new Date(event.endsAt);
  if (event.allDay) {
    const startKey = eventStartDateKey(event);
    const endKey = eventEndDateKey(event);
    const lastVisibleKey = endKey && endKey !== startKey ? addDaysToKey(endKey, -1) : startKey;
    if (!lastVisibleKey || startKey === lastVisibleKey) {
      return formatDateKeyLabel(startKey, locale, { day: 'numeric', month: 'long', year: 'numeric' });
    }
    return `${formatDateKeyLabel(startKey, locale, { day: 'numeric', month: 'long', year: 'numeric' })} - ${formatDateKeyLabel(lastVisibleKey, locale, { day: 'numeric', month: 'long', year: 'numeric' })}`;
  }
  return `${start.toLocaleDateString(locale, { day: 'numeric', month: 'long', year: 'numeric' })}, ${start.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })} - ${end.toLocaleDateString(locale, { day: 'numeric', month: 'long', year: 'numeric' })}, ${end.toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })}`;
}

function getRecurrenceLabel(frequency: Recurrence['frequency'], translate: (key: 'events.daily' | 'events.weekly' | 'events.monthly' | 'events.yearly') => string): string {
  if (frequency === 'DAILY') return translate('events.daily');
  if (frequency === 'WEEKLY') return translate('events.weekly');
  if (frequency === 'MONTHLY') return translate('events.monthly');
  if (frequency === 'YEARLY') return translate('events.yearly');
  return '';
}

interface EventVisualMeta {
  color: string;
  icon: IconName;
  label: string;
}

function getEventVisualMeta(event: EventItem, calendar?: Calendar): EventVisualMeta | null {
  const haystack = [event.title, event.description, event.location, calendar?.name].filter(Boolean).join(' ').toLowerCase();
  if (matchesAny(haystack, ['biotonne', 'bioabfall', 'braune tonne'])) return { color: '#8a5a2b', icon: 'trash', label: 'Biotonne' };
  if (matchesAny(haystack, ['restabfall', 'restmüll', 'restmuell', 'schwarze tonne'])) return { color: '#1f2933', icon: 'trash', label: 'Restabfall' };
  if (matchesAny(haystack, ['gelbe tonne', 'gelber sack', 'wertstoff'])) return { color: '#e0b300', icon: 'trash', label: 'Gelbe Tonne' };
  if (matchesAny(haystack, ['blaue tonne', 'papiertonne', 'altpapier', 'papiermüll', 'papiermuell'])) return { color: '#2f7fd3', icon: 'trash', label: 'Blaue Tonne' };
  if (matchesAny(haystack, ['geburtstag', 'birthday', 'jubilaeum', 'jubiläum'])) return { color: '#d85a8a', icon: 'gift', label: 'Geburtstag' };
  if (matchesAny(haystack, ['friseur', 'frisör', 'haarschnitt', 'barber'])) return { color: '#b76ad9', icon: 'scissors', label: 'Friseur' };
  if (matchesAny(haystack, ['arzt', 'ärzt', 'zahnarzt', 'praxis', 'therapie', 'physio'])) return { color: '#27a28a', icon: 'stethoscope', label: 'Arzt' };
  if (matchesAny(haystack, ['meeting', 'besprechung', 'jour fixe', 'call', 'konferenz'])) return { color: '#6d8cff', icon: 'users', label: 'Meeting' };
  if (matchesAny(haystack, ['mitarbeiter:', 'quelle:'])) return { color: '#7c8a99', icon: 'car', label: 'Geschäftlicher Termin' };
  return null;
}

function isBirthdayContext(title: string, calendar?: Calendar): boolean {
  const haystack = [title, calendar?.name].filter(Boolean).join(' ').toLowerCase();
  return matchesAny(haystack, ['geburtstag', 'birthday', 'jubilaeum', 'jubiläum']);
}

function getBirthdayAge(event: EventItem, calendar?: Calendar): number | null {
  if (!event.birthdayYear || !isBirthdayContext(event.title, calendar)) {
    return null;
  }
  const eventYear = event.allDay ? dateFromKey(eventStartDateKey(event)).getFullYear() : new Date(event.startsAt).getFullYear();
  const age = eventYear - event.birthdayYear;
  return age > 0 && age < 150 ? age : null;
}

function matchesAny(value: string, needles: string[]): boolean {
  return needles.some((needle) => value.includes(needle));
}

function isSameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

function isWeekend(date: Date): boolean {
  const day = date.getDay();
  return day === 0 || day === 6;
}

function eventDateMatchesDay(event: EventItem, day: Date): boolean {
  if (event.allDay) {
    return eventStartDateKey(event) === formatDateKey(day);
  }
  return isSameDay(new Date(event.startsAt), day);
}

function eventStartDateKey(event: EventItem): string {
  return dateKeyInTimeZone(event.startsAt, event.timezone);
}

function eventEndDateKey(event: EventItem): string {
  return dateKeyInTimeZone(event.endsAt, event.timezone);
}

export function defaultEventDates() {
  const start = new Date();
  start.setMinutes(0, 0, 0);
  const end = new Date(start.getTime() + Number(getGeneralPreferences().defaultEventDuration) * 60 * 1000);
  return { startsAt: isoLocal(toDateInputValue(start)), endsAt: isoLocal(toDateInputValue(end)) };
}

function defaultEventStart(date: Date, preferences: GeneralPreferences): Date {
  const next = startOfDay(date);
  const workingStart = parseTimeToMinutes(preferences.workingHoursStart);
  next.setMinutes(workingStart);
  return next;
}

function defaultEventEnd(date: Date, preferences: GeneralPreferences): Date {
	return addMinutes(defaultEventStart(date, preferences), Number(preferences.defaultEventDuration));
}

function toAllDayStart(value: string): string {
	const datePart = value.split("T")[0] || formatDateValue(new Date());
	return `${datePart}T00:00`;
}

function toAllDayEnd(value: string): string {
	const datePart = value.split("T")[0] || formatDateValue(new Date());
	const date = parseDateValue(datePart) ?? new Date();
	return `${formatDateValue(addDays(date, 1))}T00:00`;
}
