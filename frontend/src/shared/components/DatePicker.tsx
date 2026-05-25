import { useEffect, useMemo, useRef, useState } from 'react';
import { Icon } from './Icon';
import { useI18n } from '../i18n';
import { getGeneralPreferences, type WeekStartPreference } from '../preferences';

interface DatePickerProps {
  ariaLabel?: string;
  disabled?: boolean;
  id?: string;
  onChange: (value: string) => void;
  value: string;
}

export function DatePicker({ ariaLabel, disabled = false, id, onChange, value }: DatePickerProps) {
  const { locale } = useI18n();
  const rootRef = useRef<HTMLDivElement>(null);
  const selectedDate = useMemo(() => parseDateValue(value) ?? startOfDay(new Date()), [value]);
  const weekStart = getGeneralPreferences().weekStart;
  const [open, setOpen] = useState(false);
  const [viewDate, setViewDate] = useState(() => startOfMonth(selectedDate));
  const today = useMemo(() => startOfDay(new Date()), []);
  const cells = useMemo(() => buildPickerCells(viewDate, weekStart), [viewDate, weekStart]);

  useEffect(() => {
    setViewDate(startOfMonth(selectedDate));
  }, [selectedDate.getFullYear(), selectedDate.getMonth()]);

  useEffect(() => {
    if (disabled) {
      setOpen(false);
    }
  }, [disabled]);

  useEffect(() => {
    function handlePointerDown(event: PointerEvent) {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    }

    document.addEventListener('pointerdown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, []);

  function selectDate(next: Date) {
    onChange(formatDateValue(next));
    setOpen(false);
  }

  return (
    <div className="date-picker" ref={rootRef}>
      <button
        aria-expanded={open}
        aria-haspopup="dialog"
        aria-label={ariaLabel}
        className="date-picker__trigger"
        disabled={disabled}
        id={id}
        onClick={() => setOpen((current) => !current)}
        type="button"
      >
        <span>{selectedDate.toLocaleDateString(locale)}</span>
        <Icon name="calendar" />
      </button>
      {open && !disabled && (
        <div className="date-picker__panel" role="dialog" aria-label={ariaLabel}>
          <div className="date-picker__header">
            <strong>{viewDate.toLocaleDateString(locale, { month: 'long', year: 'numeric' })}</strong>
            <div className="date-picker__nav">
              <button type="button" onClick={() => setViewDate(addMonths(viewDate, -1))} aria-label="Vorheriger Monat"><Icon name="chevron-left" /></button>
              <button type="button" onClick={() => setViewDate(addMonths(viewDate, 1))} aria-label="Nächster Monat"><Icon name="chevron-right" /></button>
            </div>
          </div>
          <div className="date-picker__weekdays">
            {buildWeekdays(locale, weekStart).map((day) => <span key={day}>{day}</span>)}
          </div>
          <div className="date-picker__grid">
            {cells.map((cell) => {
              const outsideMonth = cell.getMonth() !== viewDate.getMonth();
              const selected = isSameDay(cell, selectedDate);
              const current = isSameDay(cell, today);
              return (
                <button
                  className={[
                    'date-picker__day',
                    outsideMonth ? 'date-picker__day--muted' : '',
                    selected ? 'date-picker__day--selected' : '',
                    current ? 'date-picker__day--today' : ''
                  ].filter(Boolean).join(' ')}
                  key={cell.toISOString()}
                  onClick={() => selectDate(cell)}
                  type="button"
                >
                  {cell.getDate()}
                </button>
              );
            })}
          </div>
          <div className="date-picker__footer">
            <button type="button" onClick={() => selectDate(today)}>{locale === 'de' ? 'Heute' : 'Today'}</button>
          </div>
        </div>
      )}
    </div>
  );
}

export function parseDateValue(value: string): Date | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return null;
  }
  const [year, month, day] = value.split('-').map(Number);
  return new Date(year, month - 1, day, 12, 0, 0, 0);
}

export function formatDateValue(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function buildPickerCells(viewDate: Date, weekStart: WeekStartPreference): Date[] {
  const start = startOfWeek(startOfMonth(viewDate), weekStart);
  return Array.from({ length: 42 }, (_item, index) => addDays(start, index));
}

function buildWeekdays(locale: string, weekStart: WeekStartPreference): string[] {
  const start = weekStart === 'sunday' ? new Date(2026, 4, 17) : new Date(2026, 4, 18);
  return Array.from({ length: 7 }, (_item, index) => addDays(start, index).toLocaleDateString(locale, { weekday: 'short' }).replace('.', ''));
}

function startOfDay(date: Date): Date {
  const next = new Date(date);
  next.setHours(0, 0, 0, 0);
  return next;
}

function startOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), 1, 12, 0, 0, 0);
}

function startOfWeek(date: Date, weekStart: WeekStartPreference): Date {
  const next = new Date(date);
  const day = next.getDay();
  const offset = weekStart === 'sunday' ? day : (day || 7) - 1;
  next.setDate(next.getDate() - offset);
  return startOfDay(next);
}

function addDays(date: Date, days: number): Date {
  const next = new Date(date);
  next.setDate(next.getDate() + days);
  return next;
}

function addMonths(date: Date, months: number): Date {
  return new Date(date.getFullYear(), date.getMonth() + months, 1, 12, 0, 0, 0);
}

function isSameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}
