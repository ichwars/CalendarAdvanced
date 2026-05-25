import { useEffect, useId, useMemo, useRef, useState } from 'react';
import { Icon } from './Icon';

interface TimePickerProps {
  ariaLabel: string;
  disabled?: boolean;
  minuteStep?: number;
  name?: string;
  onChange: (value: string) => void;
  value: string;
}

export function TimePicker({ ariaLabel, disabled = false, minuteStep = 5, name, onChange, value }: TimePickerProps) {
  const id = useId();
  const rootRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const normalized = normalizeTime(value);
  const [hourValue, minuteValue] = normalized.split(':');
  const minuteOptions = useMemo(() => buildMinuteOptions(minuteStep, Number(minuteValue)), [minuteStep, minuteValue]);

  useEffect(() => {
    function closeOnOutsideInteraction(event: PointerEvent) {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    }

    document.addEventListener('pointerdown', closeOnOutsideInteraction);
    document.addEventListener('keydown', closeOnEscape);
    return () => {
      document.removeEventListener('pointerdown', closeOnOutsideInteraction);
      document.removeEventListener('keydown', closeOnEscape);
    };
  }, []);

  function updateHour(nextHour: number) {
    onChange(`${formatPart(nextHour)}:${minuteValue}`);
  }

  function updateMinute(nextMinute: number) {
    onChange(`${hourValue}:${formatPart(nextMinute)}`);
    setOpen(false);
  }

  return (
    <div className="time-picker" ref={rootRef}>
      <input name={name} readOnly type="hidden" value={normalized} />
      <button
        aria-expanded={open}
        aria-haspopup="dialog"
        aria-label={ariaLabel}
        className="time-picker__trigger"
        disabled={disabled}
        onClick={() => setOpen((current) => !current)}
        type="button"
      >
        <span>{normalized}</span>
        <Icon name="clock" />
      </button>
      {open && !disabled && (
        <div className="time-picker__panel" id={`${id}-panel`} role="dialog" aria-label={ariaLabel}>
          <div className="time-picker__column" aria-label="HH">
            {Array.from({ length: 24 }, (_item, hour) => (
              <button
                className={Number(hourValue) === hour ? 'time-picker__option active' : 'time-picker__option'}
                key={hour}
                onClick={() => updateHour(hour)}
                type="button"
              >
                {formatPart(hour)}
              </button>
            ))}
          </div>
          <div className="time-picker__separator">:</div>
          <div className="time-picker__column" aria-label="MM">
            {minuteOptions.map((minute) => (
              <button
                className={Number(minuteValue) === minute ? 'time-picker__option active' : 'time-picker__option'}
                key={minute}
                onClick={() => updateMinute(minute)}
                type="button"
              >
                {formatPart(minute)}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export function normalizeTime(value: string): string {
  const [rawHours = '00', rawMinutes = '00'] = value.split(':');
  const hours = Math.min(Math.max(Number(rawHours) || 0, 0), 23);
  const minutes = Math.min(Math.max(Number(rawMinutes) || 0, 0), 59);
  return `${formatPart(hours)}:${formatPart(minutes)}`;
}

function buildMinuteOptions(step: number, currentMinute: number): number[] {
  const safeStep = Math.min(Math.max(Math.round(step) || 5, 1), 30);
  const options = new Set<number>();
  for (let minute = 0; minute < 60; minute += safeStep) {
    options.add(minute);
  }
  options.add(currentMinute);
  return [...options].sort((a, b) => a - b);
}

function formatPart(value: number): string {
  return String(value).padStart(2, '0');
}
