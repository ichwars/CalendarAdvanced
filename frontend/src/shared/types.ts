export type RoleName = 'admin' | 'editor' | 'viewer';

export interface User {
  id: number;
  email: string;
  username: string;
  displayName: string;
  active: boolean;
  roles: RoleName[];
  twoFactorEnabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Calendar {
  id: number;
  ownerUserId: number;
  name: string;
  description?: string;
  color: string;
  timezone: string;
  visible: boolean;
  reminderEnabled: boolean;
  reminderDaysBefore: number;
  reminderTime: string;
  sameDayReminderTime: string;
}

export interface Recurrence {
  frequency: '' | 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'YEARLY';
  interval: number;
  count?: number;
  until?: string;
  byDay?: string;
  rrule?: string;
}

export interface Attendee {
  email: string;
  displayName?: string;
  status?: 'needs_action' | 'accepted' | 'declined' | 'tentative';
}

export interface Reminder {
  minutesBefore: number;
}

export interface DueReminder {
  id: number;
  eventId: number;
  taskId?: number;
  kind: 'event' | 'task';
  title: string;
  calendarName: string;
  startsAt: string;
  minutesBefore: number;
  dueAt: string;
}

export interface EventItem {
  id: number;
  calendarId: number;
  uid: string;
  title: string;
  description?: string;
  location?: string;
  startsAt: string;
  endsAt: string;
  timezone: string;
  allDay: boolean;
  private: boolean;
  completed: boolean;
  birthdayYear?: number;
  status: string;
  recurrence?: Recurrence;
  attendees?: Attendee[];
  reminders?: Reminder[];
  conflicts?: Array<{ eventId: number; title: string }>;
  davSynced?: boolean;
}

export interface TaskItem {
  id: number;
  userId: number;
  title: string;
  description?: string;
  dueAt?: string;
  reminderAt?: string;
  priority: 'low' | 'normal' | 'high';
  completed: boolean;
  showInCalendar: boolean;
  completedAt?: string;
  reminderDeliveredAt?: string;
  createdAt: string;
  updatedAt: string;
  davSynced?: boolean;
}

export interface ContactItem {
  id: number;
  userId: number;
  firstName: string;
  lastName: string;
  company?: string;
  companyEmail?: string;
  companyPhone?: string;
  companyMobile?: string;
  email?: string;
  phone?: string;
  mobile?: string;
  address?: string;
  birthday?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CalDAVConnection {
  id?: number;
  userId?: number;
  displayName: string;
  baseUrl: string;
  username: string;
  passwordConfigured: boolean;
  syncEnabled: boolean;
  syncDirection: 'pull' | 'push' | 'two_way';
  syncEvents: boolean;
  syncTasks: boolean;
  syncContacts: boolean;
  syncIntervalMinutes: number;
  syncWindowPastDays: number;
  syncWindowFutureDays: number;
  lastTestAt?: string;
  lastTestStatus?: string;
  lastTestMessage?: string;
  lastSyncAt?: string;
  lastSyncStatus?: string;
  lastSyncMessage?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface CalDAVConnectionInput extends Omit<CalDAVConnection, 'id' | 'userId' | 'passwordConfigured' | 'lastTestAt' | 'lastTestStatus' | 'lastTestMessage' | 'lastSyncAt' | 'lastSyncStatus' | 'lastSyncMessage' | 'createdAt' | 'updatedAt'> {
  password?: string;
}

export interface CalDAVConnectionTestResult {
  ok: boolean;
  status: string;
  message: string;
  statusCode?: number;
  calendarUrl?: string;
}

export interface DAVCollection {
  id: number;
  userId: number;
  kind: 'calendar' | 'addressbook';
  displayName: string;
  url: string;
  selected: boolean;
  supportsEvents: boolean;
  supportsTasks: boolean;
  ctag?: string;
  syncToken?: string;
  lastSeenAt: string;
  updatedAt: string;
}

export interface DAVSyncResult {
  ok: boolean;
  status: string;
  message: string;
  eventsImported: number;
  eventsUpdated: number;
  eventsExported: number;
  eventsDeleted: number;
  tasksImported: number;
  tasksUpdated: number;
  tasksExported: number;
  tasksDeleted: number;
  contactsImported: number;
  contactsUpdated: number;
  contactsExported: number;
  contactsDeleted: number;
  skipped: number;
  warnings?: string[];
}

export interface DAVSyncHistoryItem {
  id: number;
  mode: 'manual' | 'auto' | string;
  status: string;
  message: string;
  events: number;
  tasks: number;
  contacts: number;
  skipped: number;
  warnings?: string[];
  createdAt: string;
}

export interface BackupPreview {
  app: string;
  version: number;
  createdAt: string;
  counts: Record<string, number>;
}

export interface AuditEntry {
  id: number;
  actorId?: number;
  action: string;
  entityType?: string;
  entityId?: string;
  ip?: string;
  userAgent?: string;
  metadata?: string;
  createdAt: string;
}

export interface ApiError {
  code: string;
  message: string;
  details?: unknown;
}
