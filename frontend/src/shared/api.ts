import type { ApiError, BackupPreview, CalDAVConnection, CalDAVConnectionInput, CalDAVConnectionTestResult, Calendar, ContactItem, DAVCollection, DAVSyncHistoryItem, DAVSyncResult, DueReminder, EventItem, TaskItem, User, AuditEntry } from './types';
import type { GeneralPreferences } from './preferences';

export interface ICSImportPreview {
  allDayCount: number;
  eventCount: number;
  rangeEnd?: string;
  rangeStart?: string;
  recurringCount: number;
  samples: Array<{ allDay: boolean; endsAt: string; location?: string; startsAt: string; title: string }>;
  warnings: string[];
}

export interface ICSImportResult {
  imported: number;
  skipped: number;
  warnings: string[];
}

export interface ExcelImportPreview {
  cancelledRows: number;
  eventCount: number;
  rangeEnd?: string;
  rangeStart?: string;
  rows: number;
  samples: Array<{ completed: boolean; date: string; employee?: string; location?: string; pop?: string; sheet: string; status?: string; title: string; week: number; weekday: string }>;
  warnings: string[];
}

export interface ExcelImportResult {
  imported: number;
  skipped: number;
  skippedCancelled: number;
  updated: number;
  warnings: string[];
}

const jsonHeaders = () => ({
  'Content-Type': 'application/json',
  'X-CSRF-Token': readCookie('ck_csrf')
});

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, {
      credentials: 'same-origin',
      ...init,
      headers: {
        ...(init.body ? jsonHeaders() : { 'X-CSRF-Token': readCookie('ck_csrf') }),
        ...(init.headers ?? {})
      }
    });
  } catch {
    throw backendUnavailableError();
  }
  const contentType = response.headers.get('Content-Type') ?? '';
  const payload = contentType.includes('application/json') ? await response.json() : undefined;
  if (!response.ok) {
    if (!payload && [502, 503, 504].includes(response.status)) {
      throw backendUnavailableError();
    }
    const apiError = (payload ?? { code: 'http_error', message: response.statusText }) as ApiError;
    throw apiError;
  }
  return payload as T;
}

async function upload<T>(path: string, formData: FormData): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, {
      method: 'POST',
      credentials: 'same-origin',
      body: formData,
      headers: { 'X-CSRF-Token': readCookie('ck_csrf') }
    });
  } catch {
    throw backendUnavailableError();
  }
  const contentType = response.headers.get('Content-Type') ?? '';
  const payload = contentType.includes('application/json') ? await response.json() : undefined;
  if (!response.ok) {
    throw (payload ?? { code: 'http_error', message: response.statusText }) as ApiError;
  }
  return payload as T;
}

async function download(path: string): Promise<Blob> {
  let response: Response;
  try {
    response = await fetch(path, {
      credentials: 'same-origin',
      headers: { 'X-CSRF-Token': readCookie('ck_csrf') }
    });
  } catch {
    throw backendUnavailableError();
  }
  if (!response.ok) {
    const contentType = response.headers.get('Content-Type') ?? '';
    const payload = contentType.includes('application/json') ? await response.json() : undefined;
    throw (payload ?? { code: 'http_error', message: response.statusText }) as ApiError;
  }
  return response.blob();
}

function backendUnavailableError(): ApiError {
  return {
    code: 'backend_unavailable',
    message: 'Das Backend ist nicht erreichbar. Starte den API-Server auf Port 8080 und versuche es erneut.'
  };
}

export function isBackendUnavailable(error: unknown): boolean {
  return (error as Partial<ApiError> | undefined)?.code === 'backend_unavailable';
}

function readCookie(name: string): string {
  const value = document.cookie.split('; ').find((row) => row.startsWith(`${name}=`));
  return value ? decodeURIComponent(value.split('=').slice(1).join('=')) : '';
}

export const api = {
  setupStatus: () => request<{ required: boolean }>('/api/v1/setup/status'),
  setupAdmin: (body: { email: string; username: string; displayName: string; password: string }) =>
    request<{ user: User }>('/api/v1/setup/admin', { method: 'POST', body: JSON.stringify(body) }),
  login: (body: { email: string; password: string; totpCode?: string; backupCode?: string }) =>
    request<{ user?: User; csrfToken?: string; twoFactorRequired: boolean; code?: string }>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(body) }),
  logout: () => request<{ ok: boolean }>('/api/v1/auth/logout', { method: 'POST', body: '{}' }),
  me: () => request<{ user: User; csrfToken: string }>('/api/v1/auth/me'),
  preferences: () => request<{ persisted: boolean; preferences: GeneralPreferences }>('/api/v1/preferences'),
  savePreferences: (body: GeneralPreferences) =>
    request<{ persisted: boolean; preferences: GeneralPreferences }>('/api/v1/preferences', { method: 'PUT', body: JSON.stringify(body) }),
  changePassword: (body: { currentPassword: string; newPassword: string }) =>
    request<{ ok: boolean }>('/api/v1/auth/password/change', { method: 'POST', body: JSON.stringify(body) }),
  requestReset: (email: string) => request<{ sent: boolean; localToken?: string }>('/api/v1/auth/password/request-reset', { method: 'POST', body: JSON.stringify({ email }) }),
  resetPassword: (token: string, newPassword: string) => request<{ ok: boolean }>('/api/v1/auth/password/reset', { method: 'POST', body: JSON.stringify({ token, newPassword }) }),
  twoFactorSetup: () => request<{ secret: string; otpauthUri: string }>('/api/v1/security/2fa/setup', { method: 'POST', body: '{}' }),
  twoFactorEnable: (code: string) => request<{ backupCodes: string[] }>('/api/v1/security/2fa/enable', { method: 'POST', body: JSON.stringify({ code }) }),
  twoFactorDisable: (password: string) => request<{ ok: boolean }>('/api/v1/security/2fa/disable', { method: 'POST', body: JSON.stringify({ password }) }),
  users: () => request<{ items: User[] }>('/api/v1/users'),
  createUser: (body: { email: string; username: string; displayName: string; password: string; roles: string[] }) =>
    request<User>('/api/v1/users', { method: 'POST', body: JSON.stringify(body) }),
  calendars: () => request<{ items: Calendar[] }>('/api/v1/calendars'),
  createCalendar: (body: Partial<Calendar>) => request<Calendar>('/api/v1/calendars', { method: 'POST', body: JSON.stringify(body) }),
  updateCalendar: (id: number, body: Partial<Calendar>) => request<Calendar>(`/api/v1/calendars/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteCalendar: (id: number) => request<{ ok: boolean }>(`/api/v1/calendars/${id}`, { method: 'DELETE' }),
  events: (params: URLSearchParams) => request<{ items: EventItem[] }>(`/api/v1/events?${params.toString()}`),
  createEvent: (body: Partial<EventItem>) => request<EventItem>('/api/v1/events', { method: 'POST', body: JSON.stringify(eventPayload(body)) }),
  updateEvent: (id: number, body: Partial<EventItem>) => request<EventItem>(`/api/v1/events/${id}`, { method: 'PUT', body: JSON.stringify(eventPayload(body)) }),
  deleteEvent: (id: number) => request<{ ok: boolean }>(`/api/v1/events/${id}`, { method: 'DELETE' }),
  tasks: (params: URLSearchParams) => request<{ items: TaskItem[] }>(`/api/v1/tasks?${params.toString()}`),
  createTask: (body: Partial<TaskItem>) => request<TaskItem>('/api/v1/tasks', { method: 'POST', body: JSON.stringify(body) }),
  updateTask: (id: number, body: Partial<TaskItem>) => request<TaskItem>(`/api/v1/tasks/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteTask: (id: number) => request<{ ok: boolean }>(`/api/v1/tasks/${id}`, { method: 'DELETE' }),
  markTaskReminderDelivered: (id: number) => request<{ ok: boolean }>(`/api/v1/tasks/${id}/reminder-delivered`, { method: 'POST', body: '{}' }),
  contacts: (params: URLSearchParams) => request<{ items: ContactItem[] }>(`/api/v1/contacts?${params.toString()}`),
  createContact: (body: Partial<ContactItem>) => request<ContactItem>('/api/v1/contacts', { method: 'POST', body: JSON.stringify(body) }),
  updateContact: (id: number, body: Partial<ContactItem>) => request<ContactItem>(`/api/v1/contacts/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteContact: (id: number) => request<{ ok: boolean }>(`/api/v1/contacts/${id}`, { method: 'DELETE' }),
  dueReminders: () => request<{ items: DueReminder[] }>('/api/v1/reminders/due'),
  markReminderDelivered: (id: number) => request<{ ok: boolean }>(`/api/v1/reminders/${id}/delivered`, { method: 'POST', body: '{}' }),
  caldavTokens: () => request<{ items: unknown[] }>('/api/v1/integrations/caldav/tokens'),
  createCalDAVToken: (name: string) => request<{ account: unknown; token: string }>('/api/v1/integrations/caldav/tokens', { method: 'POST', body: JSON.stringify({ name }) }),
  caldavConnection: () => request<CalDAVConnection>('/api/v1/integrations/caldav/connection'),
  saveCalDAVConnection: (body: CalDAVConnectionInput) => request<CalDAVConnection>('/api/v1/integrations/caldav/connection', { method: 'PUT', body: JSON.stringify(body) }),
  testCalDAVConnection: (body: CalDAVConnectionInput) => request<CalDAVConnectionTestResult>('/api/v1/integrations/caldav/test', { method: 'POST', body: JSON.stringify(body) }),
  davCollections: () => request<{ items: DAVCollection[] }>('/api/v1/integrations/caldav/collections'),
  discoverDAVCollections: (body: CalDAVConnectionInput) => request<{ items: DAVCollection[]; message: string }>('/api/v1/integrations/caldav/collections/discover', { method: 'POST', body: JSON.stringify(body) }),
  saveDAVCollections: (items: Array<{ url: string; selected: boolean }>) => request<{ items: DAVCollection[] }>('/api/v1/integrations/caldav/collections', { method: 'PUT', body: JSON.stringify({ items }) }),
  syncDAVNow: (body: { conflictStrategy?: 'local' | 'remote' } = {}) => request<DAVSyncResult>('/api/v1/integrations/caldav/sync', { method: 'POST', body: JSON.stringify(body) }),
  davSyncHistory: () => request<{ items: DAVSyncHistoryItem[] }>('/api/v1/integrations/caldav/sync-history'),
  previewExcelImport: (formData: FormData) => upload<ExcelImportPreview>('/api/v1/imports/excel/preview', formData),
  importExcel: (formData: FormData) => upload<ExcelImportResult>('/api/v1/imports/excel', formData),
  previewICSImport: (formData: FormData) => upload<ICSImportPreview>('/api/v1/imports/ics/preview', formData),
  importICS: (formData: FormData) => upload<ICSImportResult>('/api/v1/imports/ics', formData),
  audit: () => request<{ items: AuditEntry[] }>('/api/v1/audit'),
  updateCheck: () => request<unknown>('/api/v1/system/update-check'),
  downloadBackup: () => download('/api/v1/system/backup'),
  previewBackupRestore: (formData: FormData) => upload<BackupPreview>('/api/v1/system/backup/preview-restore', formData),
  restoreBackup: (formData: FormData) => upload<BackupPreview>('/api/v1/system/backup/restore', formData)
};

function eventPayload(event: Partial<EventItem>) {
  return {
    calendarId: event.calendarId,
    title: event.title,
    description: event.description ?? '',
    location: event.location ?? '',
    startsAt: event.startsAt,
    endsAt: event.endsAt,
    timezone: event.timezone,
    allDay: event.allDay ?? false,
    private: event.private ?? false,
    completed: event.completed ?? false,
    birthdayYear: event.birthdayYear ?? 0,
    recurrence: event.recurrence
      ? {
          frequency: event.recurrence.frequency,
          interval: event.recurrence.interval || 1,
          count: event.recurrence.count ?? 0,
          until: event.recurrence.until,
          byDay: event.recurrence.byDay ?? '',
          rrule: event.recurrence.rrule ?? ''
        }
      : undefined,
    attendees: event.attendees ?? [],
    reminders: event.reminders ?? []
  };
}

export function downloadURL(path: string): void {
  window.location.assign(path);
}
