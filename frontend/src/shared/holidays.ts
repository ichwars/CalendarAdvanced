import type { HolidayHighlightRegionPreference, HolidayRegionPreference } from './preferences';

export interface Holiday {
  date: string;
  highlighted?: boolean;
  name: string;
  region: HolidayRegionPreference;
}

const holidayRegions = [
  ['DE', 'Deutschland, bundesweit'],
  ['DE-BB', 'Brandenburg'],
  ['DE-BE', 'Berlin'],
  ['DE-BW', 'Baden-Württemberg'],
  ['DE-BY', 'Bayern'],
  ['DE-HB', 'Bremen'],
  ['DE-HE', 'Hessen'],
  ['DE-HH', 'Hamburg'],
  ['DE-MV', 'Mecklenburg-Vorpommern'],
  ['DE-NI', 'Niedersachsen'],
  ['DE-NW', 'Nordrhein-Westfalen'],
  ['DE-RP', 'Rheinland-Pfalz'],
  ['DE-SH', 'Schleswig-Holstein'],
  ['DE-SL', 'Saarland'],
  ['DE-SN', 'Sachsen'],
  ['DE-ST', 'Sachsen-Anhalt'],
  ['DE-TH', 'Thüringen']
] as const satisfies readonly [HolidayRegionPreference, string][];

export function getHolidayRegionOptions(): { value: HolidayRegionPreference; label: string }[] {
  return holidayRegions.map(([value, label]) => ({ value, label }));
}

export function getHolidayRegionLabel(region: HolidayRegionPreference): string {
  return holidayRegions.find(([value]) => value === region)?.[1] ?? region;
}

export function getHolidayHighlightRegionOptions(noneLabel: string): { value: HolidayHighlightRegionPreference; label: string }[] {
  return [
    { value: '', label: noneLabel },
    ...holidayRegions.filter(([value]) => value !== 'DE').map(([value, label]) => ({ value: value as HolidayHighlightRegionPreference, label }))
  ];
}

export function getCalendarHolidaysInRange(
  from: Date,
  to: Date,
  region: HolidayRegionPreference,
  highlightedRegion: HolidayHighlightRegionPreference
): Holiday[] {
  const baseHolidays = getHolidaysInRange(from, to, region).map((holiday) => ({ ...holiday, highlighted: false }));
  if (!highlightedRegion) {
    return baseHolidays;
  }

  const existing = new Set(baseHolidays.map((holiday) => holidayKey(holiday)));
  const highlightedHolidays = getHolidaysInRange(from, to, highlightedRegion)
    .filter((holiday) => !existing.has(holidayKey(holiday)))
    .map((holiday) => ({ ...holiday, highlighted: true }));

  return [...baseHolidays, ...highlightedHolidays].sort((a, b) => a.date.localeCompare(b.date) || a.name.localeCompare(b.name));
}

export function getHolidaysInRange(from: Date, to: Date, region: HolidayRegionPreference): Holiday[] {
  const years = Array.from(new Set([from.getFullYear(), to.getFullYear()]));
  if (from.getMonth() === 11) {
    years.push(from.getFullYear() + 1);
  }
  if (to.getMonth() === 0) {
    years.push(to.getFullYear() - 1);
  }

  return Array.from(new Set(years))
    .flatMap((year) => getGermanHolidays(year, region))
    .filter((holiday) => {
      const date = parseHolidayDate(holiday.date);
      return date >= startOfDay(from) && date < startOfDay(to);
    })
    .sort((a, b) => a.date.localeCompare(b.date));
}

function holidayKey(holiday: Holiday): string {
  return `${holiday.date}:${holiday.name}`;
}

function getGermanHolidays(year: number, region: HolidayRegionPreference): Holiday[] {
  const easter = calculateEasterSunday(year);
  const holidays: Holiday[] = [
    fixedHoliday(year, 1, 1, 'Neujahr', region),
    relativeHoliday(easter, -2, 'Karfreitag', region),
    relativeHoliday(easter, 1, 'Ostermontag', region),
    fixedHoliday(year, 5, 1, 'Tag der Arbeit', region),
    relativeHoliday(easter, 39, 'Christi Himmelfahrt', region),
    relativeHoliday(easter, 50, 'Pfingstmontag', region),
    fixedHoliday(year, 10, 3, 'Tag der Deutschen Einheit', region),
    fixedHoliday(year, 12, 25, '1. Weihnachtstag', region),
    fixedHoliday(year, 12, 26, '2. Weihnachtstag', region)
  ];

  if (isRegion(region, 'DE-BW', 'DE-BY', 'DE-ST')) {
    holidays.push(fixedHoliday(year, 1, 6, 'Heilige Drei Könige', region));
  }
  if (isRegion(region, 'DE-BE', 'DE-MV')) {
    holidays.push(fixedHoliday(year, 3, 8, 'Internationaler Frauentag', region));
  }
  if (isRegion(region, 'DE-BW', 'DE-BY', 'DE-HE', 'DE-NW', 'DE-RP', 'DE-SL')) {
    holidays.push(relativeHoliday(easter, 60, 'Fronleichnam', region));
  }
  if (isRegion(region, 'DE-SL')) {
    holidays.push(fixedHoliday(year, 8, 15, 'Mariä Himmelfahrt', region));
  }
  if (isRegion(region, 'DE-BB', 'DE-HB', 'DE-HH', 'DE-MV', 'DE-NI', 'DE-SH', 'DE-SN', 'DE-ST', 'DE-TH')) {
    holidays.push(fixedHoliday(year, 10, 31, 'Reformationstag', region));
  }
  if (isRegion(region, 'DE-SN')) {
    holidays.push(repentanceAndPrayerDay(year, region));
  }
  if (isRegion(region, 'DE-TH')) {
    holidays.push(fixedHoliday(year, 9, 20, 'Weltkindertag', region));
  }
  if (isRegion(region, 'DE-BW', 'DE-BY', 'DE-NW', 'DE-RP', 'DE-SL')) {
    holidays.push(fixedHoliday(year, 11, 1, 'Allerheiligen', region));
  }

  return holidays;
}

function isRegion(region: HolidayRegionPreference, ...regions: HolidayRegionPreference[]): boolean {
  return regions.includes(region);
}

function fixedHoliday(year: number, month: number, day: number, name: string, region: HolidayRegionPreference): Holiday {
  return { date: formatDateKey(new Date(year, month - 1, day)), name, region };
}

function relativeHoliday(easter: Date, offsetDays: number, name: string, region: HolidayRegionPreference): Holiday {
  const date = new Date(easter);
  date.setDate(date.getDate() + offsetDays);
  return { date: formatDateKey(date), name, region };
}

function repentanceAndPrayerDay(year: number, region: HolidayRegionPreference): Holiday {
  const november23 = new Date(year, 10, 23);
  const daysSinceWednesday = (november23.getDay() - 3 + 7) % 7 || 7;
  november23.setDate(november23.getDate() - daysSinceWednesday);
  return { date: formatDateKey(november23), name: 'Buß- und Bettag', region };
}

function calculateEasterSunday(year: number): Date {
  const goldenNumber = year % 19;
  const century = Math.floor(year / 100);
  const skippedLeapYears = Math.floor(century / 4);
  const correction = Math.floor((century + 8) / 25);
  const moonCorrection = Math.floor((century - correction + 1) / 3);
  const epact = (19 * goldenNumber + century - skippedLeapYears - moonCorrection + 15) % 30;
  const weekDayCorrection = (32 + 2 * (century % 4) + 2 * Math.floor((year % 100) / 4) - epact - (year % 100) % 4) % 7;
  const paschalFullMoon = epact + weekDayCorrection - 7 * Math.floor((goldenNumber + 11 * epact + 22 * weekDayCorrection) / 451) + 114;
  const month = Math.floor(paschalFullMoon / 31);
  const day = (paschalFullMoon % 31) + 1;
  return new Date(year, month - 1, day);
}

export function formatDateKey(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

function parseHolidayDate(value: string): Date {
  const [year, month, day] = value.split('-').map(Number);
  return new Date(year, month - 1, day);
}

function startOfDay(date: Date): Date {
  const next = new Date(date);
  next.setHours(0, 0, 0, 0);
  return next;
}
