export function dateKeyInTimeZone(value: string | Date, timeZone?: string): string {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  if (timeZone) {
    try {
      const parts = new Intl.DateTimeFormat('en-CA', {
        day: '2-digit',
        month: '2-digit',
        timeZone,
        year: 'numeric'
      }).formatToParts(date);
      const year = parts.find((part) => part.type === 'year')?.value;
      const month = parts.find((part) => part.type === 'month')?.value;
      const day = parts.find((part) => part.type === 'day')?.value;
      if (year && month && day) {
        return `${year}-${month}-${day}`;
      }
    } catch {
      // Fall back to the browser timezone when a stored TZID is unknown.
    }
  }
  return formatLocalDateKey(date);
}

export function dateFromKey(key: string): Date {
  const [year, month, day] = key.split('-').map(Number);
  if (!year || !month || !day) {
    return new Date();
  }
  return new Date(year, month - 1, day, 12, 0, 0, 0);
}

export function addDaysToKey(key: string, days: number): string {
  const date = dateFromKey(key);
  date.setDate(date.getDate() + days);
  return formatLocalDateKey(date);
}

export function formatDateKeyLabel(key: string, locale: string, options?: Intl.DateTimeFormatOptions): string {
  return dateFromKey(key).toLocaleDateString(locale, options);
}

function formatLocalDateKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}
