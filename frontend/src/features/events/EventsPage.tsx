import { useEffect, useMemo, useState } from 'react';
import { api } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { ConfirmDialog } from '../../shared/components/ConfirmDialog';
import { EmptyState } from '../../shared/components/EmptyState';
import { Icon } from '../../shared/components/Icon';
import { SelectField } from '../../shared/components/SelectField';
import { addDaysToKey, dateKeyInTimeZone, formatDateKeyLabel } from '../../shared/dates';
import { useI18n } from '../../shared/i18n';
import { getGeneralPreferences } from '../../shared/preferences';
import type { Calendar, EventItem } from '../../shared/types';
import { EventDialog } from '../calendar/CalendarPage';

type SortKey = 'title' | 'calendar' | 'startsAt' | 'endsAt' | 'location' | 'status';
type SortDirection = 'asc' | 'desc';
type StatusFilter = 'all' | 'open' | 'completed' | 'cancelled' | 'recurring';
type RangeFilter = 'all' | 'today' | 'upcoming' | 'past';

export function EventsPage() {
  const { locale, t } = useI18n();
  const preferences = useMemo(() => getGeneralPreferences(), []);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [query, setQuery] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingEvent, setEditingEvent] = useState<EventItem | null>(null);
  const [deleteEvent, setDeleteEvent] = useState<EventItem | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [sortKey, setSortKey] = useState<SortKey>('startsAt');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');
  const [calendarFilter, setCalendarFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [rangeFilter, setRangeFilter] = useState<RangeFilter>('all');

  async function load() {
    const params = new URLSearchParams({ q: query, limit: '500' });
    if (calendarFilter) {
      params.set('calendarId', calendarFilter);
    }
    applyRangeFilter(params, rangeFilter);
    const [calendarResponse, eventResponse] = await Promise.all([
      api.calendars(),
      api.events(params)
    ]);
    setCalendars(calendarResponse.items);
    setEvents(eventResponse.items);
  }

  useEffect(() => {
    void load();
  }, [calendarFilter, query, rangeFilter]);

  async function createEvent(body: Partial<EventItem>) {
    await api.createEvent(body);
    setDialogOpen(false);
    await load();
  }

  async function updateEvent(event: EventItem, body: Partial<EventItem>) {
    await api.updateEvent(event.id, { ...event, ...body });
    setEditingEvent(null);
    await load();
  }

  async function confirmDeleteEvent() {
    if (!deleteEvent) {
      return;
    }
    setDeleting(true);
    try {
      await api.deleteEvent(deleteEvent.id);
      setDeleteEvent(null);
      await load();
    } finally {
      setDeleting(false);
    }
  }

  function changeSort(nextKey: SortKey) {
    if (sortKey === nextKey) {
      setSortDirection((current) => current === 'asc' ? 'desc' : 'asc');
      return;
    }
    setSortKey(nextKey);
    setSortDirection('asc');
  }

  const filteredEvents = useMemo(() => {
    return events.filter((event) => {
      if (statusFilter === 'completed') return event.completed;
      if (statusFilter === 'cancelled') return event.status === 'cancelled';
      if (statusFilter === 'recurring') return Boolean(event.recurrence);
      if (statusFilter === 'open') return !event.completed && event.status !== 'cancelled';
      return true;
    });
  }, [events, statusFilter]);

  const sortedEvents = useMemo(() => {
    const collator = new Intl.Collator(locale, { numeric: true, sensitivity: 'base' });
    return [...filteredEvents].sort((left, right) => {
      const leftValue = getSortValue(left, sortKey, calendars);
      const rightValue = getSortValue(right, sortKey, calendars);
      const result = typeof leftValue === 'number' && typeof rightValue === 'number'
        ? leftValue - rightValue
        : collator.compare(String(leftValue), String(rightValue));
      return sortDirection === 'asc' ? result : -result;
    });
  }, [calendars, filteredEvents, locale, sortDirection, sortKey]);

  const calendarFilterOptions = [
    { value: '', label: t('events.filterAllCalendars') },
    ...calendars.map((calendar) => ({ value: String(calendar.id), label: calendar.name }))
  ];
  const statusFilterOptions = [
    { value: 'all', label: t('events.filterAllStatus') },
    { value: 'open', label: t('events.open') },
    { value: 'completed', label: t('events.completed') },
    { value: 'cancelled', label: t('events.cancelled') },
    { value: 'recurring', label: t('events.filterRecurring') }
  ] satisfies { value: StatusFilter; label: string }[];
  const rangeFilterOptions = [
    { value: 'all', label: t('events.filterAllRanges') },
    { value: 'today', label: t('events.filterToday') },
    { value: 'upcoming', label: t('events.filterUpcoming') },
    { value: 'past', label: t('events.filterPast') }
  ] satisfies { value: RangeFilter; label: string }[];

  return (
    <div className="page">
      <header className="page-header events-page-header">
        <div>
          <h1>{t('events.title')}</h1>
        </div>
        <div className="events-page-header__actions">
          <div className={query ? 'events-search events-search--active' : 'events-search'}>
            <Icon name="search" />
            <input aria-label={t('events.search')} value={query} onChange={(event) => setQuery(event.currentTarget.value)} placeholder={t('events.search')} />
            {query && (
              <button type="button" onClick={() => setQuery('')} aria-label={t('common.clear')} title={t('common.clear')}>
                <Icon name="x" />
              </button>
            )}
          </div>
          <Button disabled={!calendars.length} onClick={() => setDialogOpen(true)} type="button">
            + {t('events.add')}
          </Button>
        </div>
      </header>

      <div className="events-filterbar">
        <SelectField label={t('calendar.title')} value={calendarFilter} onChange={setCalendarFilter} options={calendarFilterOptions} />
        <SelectField label={t('events.status')} value={statusFilter} onChange={setStatusFilter} options={statusFilterOptions} />
        <SelectField label={t('events.filterRange')} value={rangeFilter} onChange={setRangeFilter} options={rangeFilterOptions} />
      </div>

      {!sortedEvents.length ? <EmptyState message={t('common.empty')} /> : (
        <div className="table-wrap events-table-wrap">
          <table className="events-table">
            <thead>
              <tr>
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('common.title')} onSort={changeSort} sortKey="title" />
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('calendar.title')} onSort={changeSort} sortKey="calendar" />
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('common.start')} onSort={changeSort} sortKey="startsAt" />
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('common.end')} onSort={changeSort} sortKey="endsAt" />
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('common.location')} onSort={changeSort} sortKey="location" />
                <SortableHeader activeKey={sortKey} direction={sortDirection} label={t('events.status')} onSort={changeSort} sortKey="status" />
                <th className="events-table__actions-heading">{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {sortedEvents.map((event) => {
                const calendar = calendars.find((item) => item.id === event.calendarId);
                return (
                  <tr key={`${event.id}-${event.startsAt}`}>
                    <td>
                      <div className="events-table__title-cell">
                        <span className="event-dot" style={{ background: calendar?.color }} />
                        <div>
                          <strong>{event.title}</strong>
                          {event.davSynced && <span className="dav-sync-badge" title={t('common.davSynced')}>DAV</span>}
                          {event.recurrence && <span className="events-table__meta">{t('events.series')}</span>}
                          {event.conflicts?.length ? <span className="warning">{t('events.conflict')}</span> : null}
                        </div>
                      </div>
                    </td>
                    <td>{calendar?.name ?? '-'}</td>
                    <td>{formatEventDate(event.startsAt, event.allDay, locale, event.timezone)}</td>
                    <td>{formatEventEndDate(event, locale)}</td>
                    <td>{event.location || '-'}</td>
                    <td><EventStatus event={event} /></td>
                    <td>
                      <div className="events-table__actions">
                        <button className="icon-button" type="button" onClick={() => setEditingEvent(event)} aria-label={t('common.edit')} title={t('common.edit')}>
                          <Icon name="pencil" />
                        </button>
                        <button className="icon-button icon-button--danger" type="button" onClick={() => setDeleteEvent(event)} aria-label={t('common.delete')} title={t('common.delete')}>
                          <Icon name="trash" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {dialogOpen && (
        <EventDialog
          calendars={calendars}
          date={new Date()}
          onClose={() => setDialogOpen(false)}
          onSave={createEvent}
          preferences={preferences}
        />
      )}
      {editingEvent && (
        <EventDialog
          calendars={calendars}
          date={new Date(editingEvent.startsAt)}
          event={editingEvent}
          onClose={() => setEditingEvent(null)}
          onSave={(body) => updateEvent(editingEvent, body)}
          preferences={preferences}
        />
      )}
      {deleteEvent && (
        <ConfirmDialog
          busy={deleting}
          message={`${deleteEvent.recurrence ? t('events.deleteSeriesConfirm') : t('events.deleteConfirm')}\n\n${t('events.deleteDavWarning')}`}
          onCancel={() => setDeleteEvent(null)}
          onConfirm={() => void confirmDeleteEvent()}
          title={t('events.deleteTitle')}
        />
      )}
    </div>
  );
}

function SortableHeader({
  activeKey,
  direction,
  label,
  onSort,
  sortKey
}: {
  activeKey: SortKey;
  direction: SortDirection;
  label: string;
  onSort: (key: SortKey) => void;
  sortKey: SortKey;
}) {
  const active = activeKey === sortKey;
  return (
    <th aria-sort={active ? (direction === 'asc' ? 'ascending' : 'descending') : 'none'}>
      <button className={active ? 'events-table__sort active' : 'events-table__sort'} type="button" onClick={() => onSort(sortKey)}>
        <span>{label}</span>
        <Icon className={active ? `sort-icon sort-icon--${direction}` : 'sort-icon'} name="sort" />
      </button>
    </th>
  );
}

function EventStatus({ event }: { event: EventItem }) {
  const { t } = useI18n();
  if (event.completed) {
    return <span className="event-status event-status--completed">{t('events.completed')}</span>;
  }
  if (event.status === 'cancelled') {
    return <span className="event-status event-status--cancelled">{t('events.cancelled')}</span>;
  }
  return <span className="event-status event-status--open">{t('events.open')}</span>;
}

function getSortValue(event: EventItem, sortKey: SortKey, calendars: Calendar[]): string | number {
  if (sortKey === 'calendar') {
    return calendars.find((calendar) => calendar.id === event.calendarId)?.name ?? '';
  }
  if (sortKey === 'startsAt') return new Date(event.startsAt).getTime();
  if (sortKey === 'endsAt') return new Date(event.endsAt).getTime();
  if (sortKey === 'location') return event.location ?? '';
  if (sortKey === 'status') return event.completed ? 'completed' : event.status;
  return event.title;
}

function applyRangeFilter(params: URLSearchParams, rangeFilter: RangeFilter): void {
  const now = new Date();
  if (rangeFilter === 'upcoming') {
    params.set('from', now.toISOString());
    return;
  }
  if (rangeFilter === 'past') {
    params.set('to', now.toISOString());
    return;
  }
  if (rangeFilter === 'today') {
    const start = new Date(now);
    start.setHours(0, 0, 0, 0);
    const end = new Date(start);
    end.setDate(end.getDate() + 1);
    params.set('from', start.toISOString());
    params.set('to', end.toISOString());
  }
}

function formatEventDate(value: string, allDay: boolean, locale: string, timezone?: string): string {
  const date = new Date(value);
  if (allDay) {
    return formatDateKeyLabel(dateKeyInTimeZone(value, timezone), locale);
  }
  return date.toLocaleString(locale, { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function formatEventEndDate(event: EventItem, locale: string): string {
  if (event.allDay) {
    const startKey = dateKeyInTimeZone(event.startsAt, event.timezone);
    const endKey = dateKeyInTimeZone(event.endsAt, event.timezone);
    const lastVisibleKey = endKey && endKey !== startKey ? addDaysToKey(endKey, -1) : startKey;
    return lastVisibleKey && lastVisibleKey !== startKey ? formatDateKeyLabel(lastVisibleKey, locale) : '-';
  }
  return formatEventDate(event.endsAt, false, locale, event.timezone);
}
