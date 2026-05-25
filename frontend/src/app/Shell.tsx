import { useEffect, useMemo, useState } from 'react';
import { routes } from './routes';
import { CalendarPage } from '../features/calendar/CalendarPage';
import { EventsPage } from '../features/events/EventsPage';
import { TasksPage } from '../features/tasks/TasksPage';
import { ContactsPage } from '../features/contacts/ContactsPage';
import { IntegrationsPage } from '../features/integrations/IntegrationsPage';
import { ExportsPage } from '../features/exports/ExportsPage';
import { SettingsPage } from '../features/settings/SettingsPage';
import { UsersPage } from '../features/users/UsersPage';
import { SecurityPage } from '../features/security/SecurityPage';
import { AuditPage } from '../features/audit/AuditPage';
import { api } from '../shared/api';
import { Button } from '../shared/components/Button';
import { Card } from '../shared/components/Card';
import { Icon, type IconName } from '../shared/components/Icon';
import { useI18n } from '../shared/i18n';
import { getGeneralPreferences, getLastRoute, setLastRoute } from '../shared/preferences';
import { getTheme, setTheme, type ThemeName } from '../shared/theme';
import type { CalDAVConnection, DAVCollection, DAVSyncHistoryItem, DueReminder, EventItem, TaskItem, User } from '../shared/types';
import type { AppRoute } from './routes';
import packageJson from '../../package.json';

const appVersion = `v${packageJson.version}`;
const sidebarCollapsedKey = 'calendaradvanced.sidebarCollapsed';
const githubUrl = import.meta.env.VITE_GITHUB_URL || 'https://github.com/droth/CalendarAdvanced';
const routeIcons: Record<string, IconName> = {
  overview: 'layout',
  calendar: 'calendar',
  events: 'list',
  tasks: 'check-square',
  contacts: 'users',
  settings: 'settings'
};

export function Shell({ user, onUserChange }: { user: User; onUserChange: (user: User | null) => void }) {
  const { locale, t } = useI18n();
  const [route, setRoute] = useState(() => resolveInitialRoute());
  const [themeChoice, setThemeChoice] = useState<ThemeName>(getTheme());
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => localStorage.getItem(sidebarCollapsedKey) === 'true');
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [dueReminders, setDueReminders] = useState<DueReminder[]>([]);

  useEffect(() => {
    const onHashChange = () => {
      const nextRoute = window.location.hash.replace('#/', '') || getGeneralPreferences().startPage;
      setRoute(nextRoute);
      if (getGeneralPreferences().rememberLastRoute) {
        setLastRoute(nextRoute);
      }
      setMobileMenuOpen(false);
    };
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  useEffect(() => {
    let stopped = false;

    async function pollDueReminders() {
      try {
        const response = await api.dueReminders();
        if (stopped || !response.items.length) {
          return;
        }
        const freshItems = response.items.filter((item) => !dueReminders.some((current) => current.id === item.id));
        if (!freshItems.length) {
          return;
        }
        setDueReminders((current) => [...freshItems, ...current].slice(0, 4));
        freshItems.forEach((item) => {
          showBrowserNotification(item, locale, t('reminders.title'));
          if (item.kind === 'task') {
            void api.markTaskReminderDelivered(item.id);
          } else {
            void api.markReminderDelivered(item.id);
          }
        });
      } catch {
        // Erinnerungen sollen die App nicht stören, wenn der Poll einmal fehlschlägt.
      }
    }

    void pollDueReminders();
    const interval = window.setInterval(() => void pollDueReminders(), 60_000);
    return () => {
      stopped = true;
      window.clearInterval(interval);
    };
  }, [dueReminders, locale, t]);

  const visibleRoutes = useMemo(
    () => routes
      .filter((item) => item.nav === 'main')
      .filter((item) => !item.adminOnly || user.roles.includes('admin'))
      .filter((item) => !item.minRole || user.roles.includes(item.minRole) || user.roles.includes('admin')),
    [user.roles]
  );
  const settingsRoutes = useMemo<AppRoute[]>(
    () => [
      { path: 'settings', label: 'settings.general' },
      ...routes
        .filter((item) => item.nav === 'settings')
        .filter((item) => !item.adminOnly || user.roles.includes('admin'))
        .filter((item) => !item.minRole || user.roles.includes(item.minRole) || user.roles.includes('admin'))
    ],
    [user.roles]
  );
  const activeRoute = routes.find((item) => item.path === route);
  const settingsOpen = route === 'settings' || activeRoute?.nav === 'settings';

  const page = route === 'calendar' ? <CalendarPage />
    : route === 'events' ? <EventsPage />
    : route === 'tasks' ? <TasksPage />
    : route === 'contacts' ? <ContactsPage />
    : route === 'integrations' ? <IntegrationsPage />
    : route === 'exports' ? <ExportsPage />
    : route === 'settings' ? <SettingsPage />
    : route === 'users' ? <UsersPage />
    : route === 'security' ? <SecurityPage user={user} onUserChange={onUserChange} />
    : route === 'audit' ? <AuditPage />
    : <OverviewPage />;

  async function logout() {
    await api.logout();
    onUserChange(null);
  }

  function toggleTheme() {
    const next = themeChoice === 'dark' ? 'light' : 'dark';
    setThemeChoice(next);
    setTheme(next);
  }

  function toggleSidebar() {
    setSidebarCollapsed((current) => {
      const next = !current;
      localStorage.setItem(sidebarCollapsedKey, String(next));
      return next;
    });
  }

  const shellClassName = [
    'shell',
    sidebarCollapsed ? 'shell--collapsed' : '',
    mobileMenuOpen ? 'shell--mobile-open' : ''
  ].filter(Boolean).join(' ');

  return (
    <div className={shellClassName}>
      <a className="skip-link" href="#content">{t('app.skip')}</a>
      <header className="mobile-topbar">
        <div className="brand">CalendarAdvanced</div>
        <button
          className="icon-button"
          type="button"
          onClick={() => setMobileMenuOpen((current) => !current)}
          title={mobileMenuOpen ? t('nav.closeMenu') : t('nav.openMenu')}
          aria-label={mobileMenuOpen ? t('nav.closeMenu') : t('nav.openMenu')}
          aria-expanded={mobileMenuOpen}
          aria-controls="sidebar-navigation"
        >
          <Icon name={mobileMenuOpen ? 'x' : 'menu'} />
        </button>
      </header>
      <aside className="sidebar" aria-label="CalendarAdvanced">
        <div className="sidebar-header">
          <div className="brand" title="CalendarAdvanced">
            <span className="brand__full">CalendarAdvanced</span>
            <span className="brand__compact" aria-hidden="true">CA</span>
          </div>
        </div>
        <nav id="sidebar-navigation" className="nav">
          {visibleRoutes.map((item) => item.path === 'settings' ? (
            <div key={item.path} className="nav-group">
              <a className={settingsOpen ? 'active nav-parent' : 'nav-parent'} href="#/settings" title={t(item.label)}>
                <Icon name={routeIcons[item.path]} />
                <span className="nav-label">{t(item.label)}</span>
                {!sidebarCollapsed && <Icon className="chevron" name="chevron-down" />}
              </a>
              {settingsOpen && !sidebarCollapsed && (
                <div className="subnav">
                  {settingsRoutes.map((subItem) => (
                    <a key={subItem.path} className={route === subItem.path ? 'active' : ''} href={`#/${subItem.path}`}>{t(subItem.label)}</a>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <a key={item.path} className={route === item.path ? 'active' : ''} href={`#/${item.path}`} title={t(item.label)}>
              <Icon name={routeIcons[item.path]} />
              <span className="nav-label">{t(item.label)}</span>
            </a>
          ))}
        </nav>
        <div className="sidebar-collapse-bar">
          <button
            className="icon-button sidebar-toggle"
            type="button"
            onClick={toggleSidebar}
            title={sidebarCollapsed ? t('nav.expandSidebar') : t('nav.collapseSidebar')}
            aria-label={sidebarCollapsed ? t('nav.expandSidebar') : t('nav.collapseSidebar')}
            aria-expanded={!sidebarCollapsed}
          >
            <Icon name={sidebarCollapsed ? 'chevron-right' : 'chevron-left'} />
          </button>
        </div>
        <div className="sidebar__footer">
          <div className="sidebar-actions" aria-label={t('nav.sidebarActions')}>
            <a className={settingsOpen ? 'icon-button active' : 'icon-button'} href="#/settings" title={t('settings.title')} aria-label={t('settings.title')}><Icon name="settings" /></a>
            <a className="icon-button" href={githubUrl} target="_blank" rel="noreferrer" title={t('nav.github')} aria-label={t('nav.github')}><Icon name="github" /></a>
            <button className="icon-button" type="button" onClick={toggleTheme} title={themeChoice === 'dark' ? t('settings.light') : t('settings.dark')} aria-label={themeChoice === 'dark' ? t('settings.light') : t('settings.dark')}><Icon name={themeChoice === 'dark' ? 'sun' : 'moon'} /></button>
            <button className="icon-button" type="button" onClick={() => void logout()} title={t('nav.logout')} aria-label={t('nav.logout')}><Icon name="log-out" /></button>
          </div>
          <div className="app-version">{appVersion}</div>
        </div>
      </aside>
      <main id="content" className="content" tabIndex={-1}>{page}</main>
      {dueReminders.length > 0 && (
        <div className="reminder-stack" aria-live="polite">
          {dueReminders.map((reminder) => (
            <article className="reminder-toast" key={reminder.id}>
              <Icon name="clock" />
              <div>
                <strong>{reminder.title}</strong>
                <p>{reminder.calendarName} · {new Date(reminder.startsAt).toLocaleString(locale, { dateStyle: 'short', timeStyle: 'short' })}</p>
              </div>
              <button
                className="icon-button"
                type="button"
                onClick={() => setDueReminders((current) => current.filter((item) => item.id !== reminder.id))}
                aria-label={t('common.close')}
                title={t('common.close')}
              >
                <Icon name="x" />
              </button>
            </article>
          ))}
        </div>
      )}
    </div>
  );
}

function showBrowserNotification(reminder: DueReminder, locale: string, title: string) {
  if (!('Notification' in window) || Notification.permission !== 'granted') {
    return;
  }
  new Notification(title, {
    body: `${reminder.title} · ${new Date(reminder.startsAt).toLocaleString(locale, { dateStyle: 'short', timeStyle: 'short' })}`
  });
}

function resolveInitialRoute(): string {
  const hashRoute = window.location.hash.replace('#/', '');
  if (hashRoute) {
    return hashRoute;
  }
  const preferences = getGeneralPreferences();
  return preferences.rememberLastRoute ? (getLastRoute() ?? preferences.startPage) : preferences.startPage;
}

function OverviewPage() {
  const { locale, t } = useI18n();
  const [davConnection, setDavConnection] = useState<CalDAVConnection | null>(null);
  const [davCollections, setDavCollections] = useState<DAVCollection[]>([]);
  const [davHistory, setDavHistory] = useState<DAVSyncHistoryItem[]>([]);
  const [davLoading, setDavLoading] = useState(true);
  const [davSyncing, setDavSyncing] = useState(false);
  const [davSyncMessage, setDavSyncMessage] = useState('');
  const [davSyncStatus, setDavSyncStatus] = useState<'ok' | 'error' | ''>('');
  const [davHistoryFilter, setDavHistoryFilter] = useState<'all' | 'warnings' | 'errors'>('all');
  const [todayEvents, setTodayEvents] = useState<EventItem[]>([]);
  const [openTasks, setOpenTasks] = useState<TaskItem[]>([]);
  const [overviewReminders, setOverviewReminders] = useState<DueReminder[]>([]);

  async function loadDAVStatus() {
    setDavLoading(true);
    try {
      const [connection, collections] = await Promise.all([
        api.caldavConnection(),
        api.davCollections()
      ]);
      setDavConnection(connection);
      setDavCollections(collections.items);
      setDavSyncStatus(connection.lastSyncStatus === 'ok' ? 'ok' : connection.lastSyncStatus ? 'error' : '');
      void loadDAVHistory();
    } catch {
      setDavConnection(null);
      setDavCollections([]);
      setDavHistory([]);
      setDavSyncStatus('error');
    } finally {
      setDavLoading(false);
    }
  }

  async function loadDAVHistory() {
    try {
      const response = await api.davSyncHistory();
      setDavHistory(response.items);
    } catch {
      setDavHistory([]);
    }
  }

  useEffect(() => {
    void loadDAVStatus();
    void loadOverviewData();
  }, []);

  async function loadOverviewData() {
    const today = new Date();
    const start = new Date(today);
    start.setHours(0, 0, 0, 0);
    const end = new Date(today);
    end.setHours(23, 59, 59, 999);
    const eventParams = new URLSearchParams({ from: start.toISOString(), to: end.toISOString(), limit: '6', expand: 'true' });
    const taskParams = new URLSearchParams({ completed: 'false', limit: '6' });
    try {
      const [eventsResponse, tasksResponse, remindersResponse] = await Promise.all([
        api.events(eventParams),
        api.tasks(taskParams),
        api.dueReminders()
      ]);
      setTodayEvents(eventsResponse.items);
      setOpenTasks(tasksResponse.items);
      setOverviewReminders(remindersResponse.items);
    } catch {
      setTodayEvents([]);
      setOpenTasks([]);
      setOverviewReminders([]);
    }
  }

  async function syncDAVFromOverview() {
    if (!davConnection || davSyncing) {
      return;
    }
    setDavSyncing(true);
    setDavSyncMessage(t('integrations.syncRunning'));
    setDavSyncStatus('');
    try {
      const result = await api.syncDAVNow();
      setDavSyncStatus(result.ok ? 'ok' : 'error');
      setDavSyncMessage(result.ok ? t('integrations.syncSuccess') : result.message);
      await loadDAVStatus();
      await loadDAVHistory();
    } catch {
      setDavSyncStatus('error');
      setDavSyncMessage(t('integrations.davSyncFailed'));
    } finally {
      setDavSyncing(false);
    }
  }

  const activeDAVConnection = davConnection && davConnection.baseUrl && davConnection.passwordConfigured ? davConnection : null;
  const davConfigured = Boolean(activeDAVConnection);
  const selectedCollections = davCollections.filter((collection) => collection.selected);
  const syncTypes = activeDAVConnection ? [
    activeDAVConnection.syncEvents ? t('integrations.caldavSyncEvents') : '',
    activeDAVConnection.syncTasks ? t('integrations.caldavSyncTasks') : '',
    activeDAVConnection.syncContacts ? t('integrations.caldavSyncContacts') : ''
  ].filter(Boolean).join(', ') : '-';
  const lastSyncLabel = activeDAVConnection?.lastSyncAt ? formatOverviewDateTime(activeDAVConnection.lastSyncAt, locale) : t('overview.davNeverSynced');
  const nextSyncLabel = getNextDAVSyncLabel(activeDAVConnection, locale, t);
  const statusLabel = getDAVStatusLabel(davConnection, davConfigured, davSyncStatus, t);
  const statusClass = davSyncStatus === 'error' || davConnection?.lastSyncStatus === 'error' ? 'overview-status overview-status--error'
    : davConnection?.syncEnabled ? 'overview-status overview-status--ok'
    : 'overview-status';
  const displayedDAVHistory = davHistory.length > 0 ? davHistory
    : activeDAVConnection?.lastSyncAt ? [{
        id: -1,
        mode: 'auto',
        status: activeDAVConnection.lastSyncStatus || 'ok',
        message: activeDAVConnection.lastSyncMessage || '',
        events: 0,
        tasks: 0,
        contacts: 0,
        skipped: 0,
        createdAt: activeDAVConnection.lastSyncAt
      }] satisfies DAVSyncHistoryItem[]
    : [];
  const filteredDAVHistory = displayedDAVHistory.filter((item) => {
    if (davHistoryFilter === 'errors') {
      return item.status === 'error';
    }
    if (davHistoryFilter === 'warnings') {
      return Boolean(item.warnings?.length);
    }
    return true;
  });
  const lastAutoSync = displayedDAVHistory.find((item) => item.mode === 'auto');
  const lastManualSync = displayedDAVHistory.find((item) => item.mode !== 'auto');
  const quickStats = [
    { key: 'events', label: t('overview.summaryEventsToday'), value: todayEvents.length, href: '#/events' },
    { key: 'tasks', label: t('overview.summaryOpenTasks'), value: openTasks.length, href: '#/tasks' },
    { key: 'reminders', label: t('overview.summaryDueReminders'), value: overviewReminders.length, href: '#/calendar' },
    { key: 'dav', label: t('overview.summaryDav'), value: davConfigured && davConnection?.syncEnabled ? t('overview.davStatusActive') : t('overview.davStatusPaused'), href: '#/integrations' }
  ];

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>{t('overview.title')}</h1>
          <p>{t('overview.subtitle')}</p>
        </div>
      </header>
      <div className="overview-summary">
        {quickStats.map((item) => (
          <a href={item.href} key={item.key}>
            <strong>{item.value}</strong>
            <span>{item.label}</span>
          </a>
        ))}
      </div>
      <div className="overview-grid">
        <Card className="overview-card overview-card--dav">
          <div className="overview-card__header">
            <div>
              <h2>{t('overview.davTitle')}</h2>
              <p>{t('overview.davSubtitle')}</p>
            </div>
            <span className={statusClass}>{statusLabel}</span>
          </div>
          {davLoading ? (
            <p className="overview-muted">{t('common.loading')}</p>
          ) : !davConfigured ? (
            <div className="overview-empty-block">
              <p>{t('overview.davNotConfigured')}</p>
              <a className="button button--ghost overview-link-button" href="#/integrations">{t('overview.davOpenSettings')}</a>
            </div>
          ) : (
            <>
              <div className="overview-metrics">
                <span>
                  <strong>{lastSyncLabel}</strong>
                  {t('overview.davLastSync')}
                </span>
                <span>
                  <strong>{nextSyncLabel}</strong>
                  {t('overview.davNextSync')}
                </span>
                <span>
                  <strong>{selectedCollections.length}</strong>
                  {t('overview.davCollections')}
                </span>
              </div>
              <div className="overview-dav-details">
                <span>{t('overview.davData')}: <strong>{syncTypes || '-'}</strong></span>
                <span>{t('integrations.caldavDirection')}: <strong>{formatDAVDirection(activeDAVConnection!.syncDirection, t)}</strong></span>
                <span>{t('overview.davAutoSync')}: <strong>{activeDAVConnection!.syncEnabled ? t('overview.davStatusActive') : t('overview.davStatusPaused')}</strong></span>
                <span>{t('overview.davLastAutoSync')}: <strong>{lastAutoSync ? formatOverviewDateTime(lastAutoSync.createdAt, locale) : '-'}</strong></span>
                <span>{t('overview.davLastManualSync')}: <strong>{lastManualSync ? formatOverviewDateTime(lastManualSync.createdAt, locale) : '-'}</strong></span>
              </div>
              {activeDAVConnection!.lastSyncMessage && <p className={activeDAVConnection!.lastSyncStatus === 'error' ? 'overview-message overview-message--error' : 'overview-message'}>{activeDAVConnection!.lastSyncMessage}</p>}
              {davSyncMessage && <p className={davSyncStatus === 'error' ? 'overview-message overview-message--error' : 'overview-message'}>{davSyncMessage}</p>}
              <div className="button-row">
                <Button disabled={!davConfigured || davSyncing} onClick={() => void syncDAVFromOverview()} type="button" variant="ghost">
                  {davSyncing ? t('integrations.syncRunning') : t('integrations.davSyncNow')}
                </Button>
                <a className="button button--ghost overview-link-button" href="#/integrations">{t('overview.davOpenSettings')}</a>
              </div>
              <div className="overview-history">
                <div className="overview-history__top">
                  <h3>{t('overview.davHistory')}</h3>
                  <div className="overview-history__filters">
                    <button className={davHistoryFilter === 'all' ? 'active' : ''} onClick={() => setDavHistoryFilter('all')} type="button">{t('overview.filterAll')}</button>
                    <button className={davHistoryFilter === 'warnings' ? 'active' : ''} onClick={() => setDavHistoryFilter('warnings')} type="button">{t('overview.filterWarnings')}</button>
                    <button className={davHistoryFilter === 'errors' ? 'active' : ''} onClick={() => setDavHistoryFilter('errors')} type="button">{t('overview.filterErrors')}</button>
                  </div>
                </div>
                {!filteredDAVHistory.length ? (
                  <p className="overview-muted">{t('overview.davHistoryEmpty')}</p>
                ) : (
                  <ol>
                    {filteredDAVHistory.slice(0, 5).map((item) => (
                      <li key={item.id}>
                        <details className="overview-history__item">
                          <summary>
                            <span className={item.status === 'error' ? 'overview-history__status overview-history__status--error' : 'overview-history__status'}>{item.status === 'error' ? t('overview.davStatusError') : t('common.ok')}</span>
                            <span>
                              <strong>{formatOverviewDateTime(item.createdAt, locale)}</strong>
                              {formatDAVHistoryLine(item, t)}
                            </span>
                            <Icon name="chevron-down" />
                          </summary>
                          <div className="overview-history__details">
                            <div className="overview-history__counts">
                              <span><strong>{item.events}</strong>{t('integrations.caldavSyncEvents')}</span>
                              <span><strong>{item.tasks}</strong>{t('integrations.caldavSyncTasks')}</span>
                              <span><strong>{item.contacts}</strong>{t('integrations.caldavSyncContacts')}</span>
                              <span><strong>{item.skipped}</strong>{t('integrations.syncSkipped')}</span>
                            </div>
                            {item.message && <p>{item.message}</p>}
                            {item.warnings?.length ? (
                              <ul>
                                {item.warnings.map((warning, index) => <li key={`${warning}-${index}`}>{warning}</li>)}
                              </ul>
                            ) : (
                              <p>{t('overview.davHistoryNoWarnings')}</p>
                            )}
                          </div>
                        </details>
                      </li>
                    ))}
                  </ol>
                )}
                <Button disabled={!davConfigured || davSyncing} onClick={() => void syncDAVFromOverview()} type="button" variant="ghost">{t('overview.davRetry')}</Button>
              </div>
            </>
          )}
        </Card>
        <Card className="overview-card">
          <div className="overview-card__header">
            <div>
              <h2>{t('overview.todayTitle')}</h2>
              <p>{t('overview.todaySubtitle')}</p>
            </div>
          </div>
          <div className="overview-today">
            <section>
              <h3>{t('events.title')}</h3>
              {!todayEvents.length ? <p>{t('overview.noEvents')}</p> : todayEvents.map((event) => (
                <a className="overview-row" href="#/events" key={`${event.id}-${event.startsAt}`}>
                  <strong>{event.title}</strong>
                  <span>{formatOverviewDateTime(event.startsAt, locale)}</span>
                </a>
              ))}
            </section>
            <section>
              <h3>{t('tasks.title')}</h3>
              {!openTasks.length ? <p>{t('overview.noTasks')}</p> : openTasks.map((task) => (
                <a className="overview-row" href="#/tasks" key={task.id}>
                  <strong>{task.title}</strong>
                  <span>{task.dueAt ? formatOverviewDateTime(task.dueAt, locale) : t('overview.noDueDate')}</span>
                </a>
              ))}
            </section>
            <section>
              <h3>{t('reminders.title')}</h3>
              {!overviewReminders.length ? <p>{t('overview.noReminders')}</p> : overviewReminders.map((reminder) => (
                <a className="overview-row" href={reminder.kind === 'task' ? '#/tasks' : '#/events'} key={reminder.id}>
                  <strong>{reminder.title}</strong>
                  <span>{formatOverviewDateTime(reminder.dueAt, locale)}</span>
                </a>
              ))}
            </section>
          </div>
        </Card>
      </div>
    </div>
  );
}

function formatOverviewDateTime(value: string, locale: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '-';
  }
  return date.toLocaleString(locale, { dateStyle: 'short', timeStyle: 'short' });
}

function getNextDAVSyncLabel(connection: CalDAVConnection | null, locale: string, t: ReturnType<typeof useI18n>['t']): string {
  if (!connection?.syncEnabled) {
    return t('overview.davPaused');
  }
  if (!connection.lastSyncAt) {
    return t('overview.davDueNow');
  }
  const lastSync = new Date(connection.lastSyncAt);
  if (Number.isNaN(lastSync.getTime())) {
    return t('overview.davDueNow');
  }
  const intervalMinutes = Math.max(connection.syncIntervalMinutes || 60, 15);
  const nextSync = new Date(lastSync.getTime() + intervalMinutes * 60_000);
  if (nextSync.getTime() <= Date.now()) {
    return t('overview.davDueNow');
  }
  return nextSync.toLocaleString(locale, { dateStyle: 'short', timeStyle: 'short' });
}

function getDAVStatusLabel(connection: CalDAVConnection | null, configured: boolean, status: 'ok' | 'error' | '', t: ReturnType<typeof useI18n>['t']): string {
  if (!configured) {
    return t('overview.davNotConfiguredBadge');
  }
  if (status === 'error' || connection?.lastSyncStatus === 'error') {
    return t('overview.davStatusError');
  }
  if (connection?.syncEnabled) {
    return t('overview.davStatusActive');
  }
  return t('overview.davStatusPaused');
}

function formatDAVDirection(direction: CalDAVConnection['syncDirection'], t: ReturnType<typeof useI18n>['t']): string {
  if (direction === 'push') {
    return t('integrations.caldavDirectionPush');
  }
  if (direction === 'two_way') {
    return t('integrations.caldavDirectionTwoWay');
  }
  return t('integrations.caldavDirectionPull');
}

function formatDAVHistoryLine(item: DAVSyncHistoryItem, t: ReturnType<typeof useI18n>['t']): string {
  const mode = item.mode === 'auto' ? t('overview.davHistoryAuto') : t('overview.davHistoryManual');
  const counts = [
    `${item.events} ${t('integrations.caldavSyncEvents')}`,
    `${item.tasks} ${t('integrations.caldavSyncTasks')}`,
    `${item.contacts} ${t('integrations.caldavSyncContacts')}`
  ].join(', ');
  const warnings = item.warnings?.length ? `, ${item.warnings.length} ${t('overview.davHistoryWarnings')}` : '';
  return `${mode}: ${counts}${warnings}`;
}
