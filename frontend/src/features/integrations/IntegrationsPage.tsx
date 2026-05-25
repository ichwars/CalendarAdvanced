import { useEffect, useMemo, useRef, useState } from 'react';
import { api } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { Card } from '../../shared/components/Card';
import { FormField } from '../../shared/components/FormField';
import { SelectField } from '../../shared/components/SelectField';
import { useI18n } from '../../shared/i18n';
import type { TranslationKey } from '../../shared/i18nTranslations';
import type { ApiError, CalDAVConnection, CalDAVConnectionInput, CalDAVConnectionTestResult, DAVCollection, DAVSyncResult } from '../../shared/types';

type SaveState = 'idle' | 'saved' | 'error';
type ConnectionStatus = 'ok' | 'error' | '';
type AutoSaveState = 'idle' | 'saving' | 'saved' | 'error';
type SyncToastState = 'idle' | 'syncing' | 'success' | 'error';

const defaultConnection: CalDAVConnectionInput = {
  displayName: 'Radicale',
  baseUrl: '',
  username: '',
  password: '',
  syncEnabled: false,
  syncDirection: 'pull',
  syncEvents: true,
  syncTasks: false,
  syncContacts: true,
  syncIntervalMinutes: 60,
  syncWindowPastDays: 30,
  syncWindowFutureDays: 365
};

export function IntegrationsPage() {
  const { t } = useI18n();
  const [connection, setConnection] = useState<CalDAVConnectionInput>(defaultConnection);
  const [passwordConfigured, setPasswordConfigured] = useState(false);
  const [saveState, setSaveState] = useState<SaveState>('idle');
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('');
  const [testResult, setTestResult] = useState<CalDAVConnectionTestResult | null>(null);
  const [syncResult, setSyncResult] = useState<DAVSyncResult | null>(null);
  const [collections, setCollections] = useState<DAVCollection[]>([]);
  const [discoveryMessage, setDiscoveryMessage] = useState('');
  const [autoSaveState, setAutoSaveState] = useState<AutoSaveState>('idle');
  const [syncToastState, setSyncToastState] = useState<SyncToastState>('idle');
  const [autoSaveToken, setAutoSaveToken] = useState(0);
  const [working, setWorking] = useState<'save' | 'test' | 'discover' | 'sync' | null>(null);
  const loadedRef = useRef(false);

  const directionOptions = useMemo(() => [
    { value: 'pull', label: t('integrations.caldavDirectionPull') },
    { value: 'push', label: t('integrations.caldavDirectionPush') },
    { value: 'two_way', label: t('integrations.caldavDirectionTwoWay') }
  ] satisfies Array<{ value: CalDAVConnectionInput['syncDirection']; label: string }>, [t]);

  const canSubmit = Boolean(connection.baseUrl.trim() && connection.username.trim() && (connection.password?.trim() || passwordConfigured));
  const selectedCollections = collections.filter((collection) => collection.selected);
  const collectionGroups = getCollectionGroups(collections, t);
  const activeDataTypes = [
    connection.syncEvents ? t('integrations.caldavSyncEvents') : '',
    connection.syncTasks ? t('integrations.caldavSyncTasks') : '',
    connection.syncContacts ? t('integrations.caldavSyncContacts') : ''
  ].filter(Boolean);

  useEffect(() => {
    void loadConnection();
  }, []);

  useEffect(() => {
    if (!autoSaveToken || !loadedRef.current || !canSubmit) {
      return;
    }
    const timeout = window.setTimeout(() => {
      void autoSave();
    }, 700);
    return () => window.clearTimeout(timeout);
  }, [autoSaveToken, canSubmit]);

  useEffect(() => {
    if (autoSaveState !== 'saved' && autoSaveState !== 'error') {
      return;
    }
    const timeout = window.setTimeout(() => setAutoSaveState('idle'), 2600);
    return () => window.clearTimeout(timeout);
  }, [autoSaveState]);

  useEffect(() => {
    if (syncToastState !== 'success' && syncToastState !== 'error') {
      return;
    }
    const timeout = window.setTimeout(() => setSyncToastState('idle'), 3200);
    return () => window.clearTimeout(timeout);
  }, [syncToastState]);

  useEffect(() => {
    if (!syncResult?.ok || syncResult.warnings?.length) {
      return;
    }
    const timeout = window.setTimeout(() => setSyncResult(null), 8000);
    return () => window.clearTimeout(timeout);
  }, [syncResult]);

  async function loadConnection() {
    const result = await api.caldavConnection();
    setConnection(connectionToInput(result));
    setPasswordConfigured(result.passwordConfigured);
    const collectionResult = await api.davCollections();
    setCollections(collectionResult.items);
    if (result.lastTestStatus) {
      setConnectionStatus(result.lastTestStatus === 'ok' ? 'ok' : 'error');
    }
    loadedRef.current = true;
  }

  async function saveConnection() {
    if (!canSubmit) {
      return;
    }
    setWorking('save');
    setSaveState('idle');
    try {
      const saved = await api.saveCalDAVConnection(connection);
      if (collections.length > 0) {
        const collectionResult = await api.saveDAVCollections(collections.map((collection) => ({ url: collection.url, selected: collection.selected })));
        setCollections(collectionResult.items);
      }
      setConnection(connectionToInput(saved));
      setPasswordConfigured(saved.passwordConfigured);
      setSaveState('saved');
    } catch {
      setSaveState('error');
    } finally {
      setWorking(null);
    }
  }

  async function autoSave(showToast = true): Promise<boolean> {
    if (!canSubmit) {
      return false;
    }
    if (showToast) {
      setAutoSaveState('saving');
    }
    try {
      await api.saveCalDAVConnection(connection);
      if (collections.length > 0) {
        const collectionResult = await api.saveDAVCollections(collections.map((collection) => ({ url: collection.url, selected: collection.selected })));
        setCollections(collectionResult.items);
      }
      if (showToast) {
        setAutoSaveState('saved');
      }
      return true;
    } catch {
      if (showToast) {
        setAutoSaveState('error');
      }
      return false;
    }
  }

  async function testConnection() {
    if (!canSubmit) {
      return;
    }
    setWorking('test');
    setTestResult(null);
    try {
      const result = await api.testCalDAVConnection(connection);
      setConnectionStatus(result.ok ? 'ok' : 'error');
      setTestResult(result);
    } catch (error) {
      setConnectionStatus('error');
      setTestResult({ ok: false, status: 'error', message: (error as ApiError).message || t('integrations.caldavTestFailed') });
    } finally {
      setWorking(null);
    }
  }

  async function discoverCollections() {
    if (!canSubmit) {
      return;
    }
    setWorking('discover');
    setDiscoveryMessage('');
    try {
      const result = await api.discoverDAVCollections(connection);
      setCollections(result.items);
      setDiscoveryMessage(result.message);
    } catch (error) {
      setDiscoveryMessage((error as ApiError).message || t('integrations.davDiscoveryFailed'));
    } finally {
      setWorking(null);
    }
  }

  async function syncNow(conflictStrategy?: 'local' | 'remote') {
    if (!canSubmit) {
      return;
    }
    setWorking('sync');
    setSyncToastState('syncing');
    setAutoSaveState('idle');
    setSyncResult(null);
    try {
      const saved = await autoSave(false);
      if (!saved) {
        throw new Error(t('integrations.autoSaveFailed'));
      }
      const result = await api.syncDAVNow(conflictStrategy ? { conflictStrategy } : {});
      setSyncResult(result);
      setConnectionStatus(result.ok ? 'ok' : 'error');
      setSyncToastState(result.ok ? 'success' : 'error');
    } catch (error) {
      setSyncResult({
        ok: false,
        status: 'error',
        message: (error as ApiError).message || t('integrations.davSyncFailed'),
        eventsImported: 0,
        eventsUpdated: 0,
        eventsExported: 0,
        eventsDeleted: 0,
        tasksImported: 0,
        tasksUpdated: 0,
        tasksExported: 0,
        tasksDeleted: 0,
        contactsImported: 0,
        contactsUpdated: 0,
        contactsExported: 0,
        contactsDeleted: 0,
        skipped: 0
      });
      setConnectionStatus('error');
      setSyncToastState('error');
    } finally {
      setWorking(null);
    }
  }

  function updateConnection(patch: Partial<CalDAVConnectionInput>, autoSaveChange = false) {
    setConnection((current) => ({ ...current, ...patch }));
    setSaveState('idle');
    if (autoSaveChange && loadedRef.current) {
      setAutoSaveToken((current) => current + 1);
    }
  }

  function updateCollectionSelection(url: string, selected: boolean) {
    setCollections((current) => current.map((collection) => collection.url === url ? { ...collection, selected } : collection));
    setSaveState('idle');
    if (loadedRef.current) {
      setAutoSaveToken((current) => current + 1);
    }
  }

  return (
    <div className="page">
      <header className="page-header"><h1>{t('integrations.title')}</h1></header>
      <div className="integrations-summary">
        <span>
          <strong>{connection.syncEnabled ? t('overview.davStatusActive') : t('overview.davStatusPaused')}</strong>
          {t('integrations.summarySync')}
        </span>
        <span>
          <strong>{selectedCollections.length}/{collections.length}</strong>
          {t('integrations.summaryCollections')}
        </span>
        <span>
          <strong>{activeDataTypes.join(', ') || '-'}</strong>
          {t('integrations.summaryDataTypes')}
        </span>
      </div>
      <div className="settings-grid settings-grid--tiles integrations-grid">
        <Card className="integrations-card">
          <div className="stack">
            <h2 className="integrations-card__title">
              <span>{t('integrations.caldavConnection')}</span>
              <StatusDot ok={connectionStatus === 'ok'} />
            </h2>
            <p>{t('integrations.caldavConnectionHelp')}</p>
            <div className="grid-form">
              <FormField label={t('common.name')} name="caldavDisplayName" value={connection.displayName} onChange={(event) => updateConnection({ displayName: event.currentTarget.value })} />
              <FormField label={t('integrations.caldavServerUrl')} name="caldavBaseUrl" value={connection.baseUrl} onChange={(event) => updateConnection({ baseUrl: event.currentTarget.value })} placeholder="https://cloud.example.com/remote.php/dav/" />
              <FormField label={t('integrations.caldavUsername')} name="caldavUsername" value={connection.username} onChange={(event) => updateConnection({ username: event.currentTarget.value })} />
              <FormField
                autoComplete="new-password"
                label={passwordConfigured ? t('integrations.caldavPasswordConfigured') : t('integrations.caldavPassword')}
                name="caldavPassword"
                onChange={(event) => updateConnection({ password: event.currentTarget.value })}
                placeholder={passwordConfigured ? t('integrations.caldavPasswordKeep') : ''}
                type="password"
                value={connection.password ?? ''}
              />
            </div>
            <div className="button-row">
              <Button disabled={!canSubmit || working === 'save'} onClick={() => void saveConnection()} type="button">{t('common.save')}</Button>
              <Button disabled={!canSubmit || working === 'test'} onClick={() => void testConnection()} type="button" variant="ghost">{t('integrations.caldavTest')}</Button>
              <Button disabled={!canSubmit || working === 'discover'} onClick={() => void discoverCollections()} type="button" variant="ghost">{t('integrations.davDiscover')}</Button>
            </div>
            {saveState === 'saved' && <p className="success">{t('integrations.caldavSaved')}</p>}
            {saveState === 'error' && <p className="error">{t('integrations.caldavSaveFailed')}</p>}
            {testResult && (
              <p className={testResult.ok ? 'success' : 'error'} role="status">
                {testResult.message}
              </p>
            )}
            {discoveryMessage && <p className={collections.length > 0 ? 'success' : 'error'} role="status">{discoveryMessage}</p>}
          </div>
        </Card>

        <Card className="integrations-card">
          <div className="stack">
            <h2 className="integrations-card__title">
              <span>{t('integrations.caldavSync')}</span>
              <StatusDot ok={connection.syncEnabled} />
            </h2>
            <p>{t('integrations.caldavSyncHelp')}</p>
            <div className="integrations-sync-layout">
              <div className="integrations-switches">
                <SelectField label={t('integrations.caldavDirection')} value={connection.syncDirection} onChange={(value) => updateConnection({ syncDirection: value }, true)} options={directionOptions} />
                <label className="check check--switch">
                  <input checked={connection.syncEnabled} onChange={(event) => updateConnection({ syncEnabled: event.currentTarget.checked }, true)} type="checkbox" />
                  {t('integrations.caldavSyncEnabled')}
                </label>
                <label className="check check--switch">
                  <input checked={connection.syncEvents} onChange={(event) => updateConnection({ syncEvents: event.currentTarget.checked }, true)} type="checkbox" />
                  {t('integrations.caldavSyncEvents')}
                </label>
                <label className="check check--switch">
                  <input checked={connection.syncTasks} onChange={(event) => updateConnection({ syncTasks: event.currentTarget.checked }, true)} type="checkbox" />
                  {t('integrations.caldavSyncTasks')}
                </label>
                <label className="check check--switch">
                  <input checked={connection.syncContacts} onChange={(event) => updateConnection({ syncContacts: event.currentTarget.checked }, true)} type="checkbox" />
                  {t('integrations.caldavSyncContacts')}
                </label>
              </div>
              <div className="integrations-sync-fields">
                <FormField label={t('integrations.caldavInterval')} min="15" max="1440" name="caldavInterval" type="number" value={String(connection.syncIntervalMinutes)} onChange={(event) => updateConnection({ syncIntervalMinutes: Number(event.currentTarget.value) }, true)} />
                <FormField label={t('integrations.caldavPastDays')} min="0" max="3650" name="caldavPastDays" type="number" value={String(connection.syncWindowPastDays)} onChange={(event) => updateConnection({ syncWindowPastDays: Number(event.currentTarget.value) }, true)} />
                <FormField label={t('integrations.caldavFutureDays')} min="1" max="3650" name="caldavFutureDays" type="number" value={String(connection.syncWindowFutureDays)} onChange={(event) => updateConnection({ syncWindowFutureDays: Number(event.currentTarget.value) }, true)} />
              </div>
            </div>
            <div className="button-row">
              <Button disabled={!canSubmit || working === 'sync'} onClick={() => void syncNow()} type="button" variant="ghost">{t('integrations.davSyncNow')}</Button>
            </div>
            {syncResult && <SyncResultPanel result={syncResult} onResolve={(strategy) => void syncNow(strategy)} busy={working === 'sync'} />}
          </div>
        </Card>

        <Card className="integrations-card integrations-card--wide">
          <div className="stack">
            <h2>{t('integrations.davCollections')}</h2>
            <p>{t('integrations.davCollectionsHelp')}</p>
            <div className="dav-collections">
              {collections.length === 0 ? (
                <p className="empty">{t('integrations.davCollectionsEmpty')}</p>
              ) : collectionGroups.map((group) => (
                <section className="dav-collection-group" key={group.key}>
                  <h3>
                    {group.label}
                    <span>{group.items.filter((item) => item.selected).length}/{group.items.length}</span>
                  </h3>
                  {group.items.map((collection) => (
                    <label className="dav-collection" key={collection.url}>
                      <input checked={collection.selected} onChange={(event) => updateCollectionSelection(collection.url, event.currentTarget.checked)} type="checkbox" />
                      <span className="dav-collection__main">
                        <strong>{collection.displayName}</strong>
                        <span>{collection.kind === 'addressbook' ? t('integrations.davAddressbook') : collectionLabel(collection, t)}</span>
                        <code>{collection.url}</code>
                      </span>
                    </label>
                  ))}
                </section>
              ))}
            </div>
          </div>
        </Card>
      </div>
      {syncToastState !== 'idle' ? (
        <div className={syncToastState === 'error' ? 'autosave-toast autosave-toast--error' : 'autosave-toast'} role="status">
          {syncToastState === 'syncing' ? t('integrations.syncRunning') : syncToastState === 'success' ? t('integrations.syncSuccess') : t('integrations.davSyncFailed')}
        </div>
      ) : autoSaveState !== 'idle' && (
        <div className={autoSaveState === 'error' ? 'autosave-toast autosave-toast--error' : 'autosave-toast'} role="status">
          {autoSaveState === 'saving' ? t('integrations.autoSaving') : autoSaveState === 'saved' ? t('integrations.autoSaved') : t('integrations.autoSaveFailed')}
        </div>
      )}
    </div>
  );
}

function StatusDot({ ok }: { ok: boolean }) {
  return <span aria-hidden="true" className={ok ? 'status-dot status-dot--ok' : 'status-dot status-dot--error'} />;
}

function SyncResultPanel({ result, onResolve, busy }: { result: DAVSyncResult; onResolve: (strategy: 'local' | 'remote') => void; busy: boolean }) {
  const { t } = useI18n();
  const warnings = result.warnings ?? [];
  const conflicts = warnings.filter(isConflictWarning);
  const affectedItems = conflicts.map((warning) => warning.split(':')[0]).filter(Boolean);
  const stats = [
    { key: 'events-imported', label: t('integrations.syncEventsImported'), value: result.eventsImported + result.eventsUpdated },
    { key: 'events-exported', label: t('integrations.syncEventsExported'), value: result.eventsExported },
    { key: 'events-deleted', label: t('integrations.syncEventsDeleted'), value: result.eventsDeleted },
    { key: 'tasks-imported', label: t('integrations.syncTasksImported'), value: result.tasksImported + result.tasksUpdated },
    { key: 'tasks-exported', label: t('integrations.syncTasksExported'), value: result.tasksExported },
    { key: 'tasks-deleted', label: t('integrations.syncTasksDeleted'), value: result.tasksDeleted },
    { key: 'contacts-imported', label: t('integrations.syncContactsImported'), value: result.contactsImported + result.contactsUpdated },
    { key: 'contacts-exported', label: t('integrations.syncContactsExported'), value: result.contactsExported },
    { key: 'contacts-deleted', label: t('integrations.syncContactsDeleted'), value: result.contactsDeleted },
    { key: 'skipped', label: t('integrations.syncSkipped'), value: result.skipped }
  ];

  return (
    <div className={result.ok ? 'sync-result sync-result--ok' : 'sync-result sync-result--error'} role="status">
      <p>{result.message}</p>
      <div className="sync-result__stats">
        {stats.map((item) => (
          <span key={item.key}>
            <strong>{item.value}</strong>
            {item.label}
          </span>
        ))}
      </div>
      {conflicts.length > 0 && (
        <div className="sync-conflicts">
          <h3>{t('integrations.syncConflictsTitle')}</h3>
          <p>{t('integrations.syncConflictsText')}</p>
          {affectedItems.length > 0 && (
            <ul className="sync-conflicts__affected">
              {affectedItems.map((item, index) => <li key={`${item}-${index}`}>{item}</li>)}
            </ul>
          )}
          <div className="button-row">
            <Button disabled={busy} onClick={() => onResolve('remote')} type="button" variant="ghost">{t('integrations.syncUseRemote')}</Button>
            <Button disabled={busy} onClick={() => onResolve('local')} type="button" variant="danger">{t('integrations.syncUseLocal')}</Button>
          </div>
        </div>
      )}
      {warnings.length > 0 && (
        <ul className="sync-result__warnings">
          {warnings.map((warning, index) => <li key={`${warning}-${index}`}>{formatSyncWarning(warning)}</li>)}
        </ul>
      )}
    </div>
  );
}

function isConflictWarning(warning: string): boolean {
  const normalized = warning.toLowerCase();
  return normalized.includes('konflikt') || normalized.includes('conflict') || normalized.includes('verändert') || normalized.includes('precondition');
}

function formatSyncWarning(warning: string): string {
  const normalized = warning.toLowerCase();
  if (isConflictWarning(warning)) {
    const [label] = warning.split(':');
    const prefix = label && label !== warning ? `${label}: ` : '';
    return `${prefix}Remote wurde geaendert. Bitte Konfliktloesung waehlen oder zuerst importieren.`;
  }
  if (normalized.includes('nicht erreichbar')) {
    return warning.replace('DAV-Server ist nicht erreichbar oder antwortet nicht.', 'DAV-Server nicht erreichbar.');
  }
  return warning;
}

function collectionLabel(collection: DAVCollection, t: (key: TranslationKey) => string): string {
  if (collection.supportsEvents && collection.supportsTasks) {
    return t('integrations.davCalendarAndTasks');
  }
  if (collection.supportsTasks) {
    return t('integrations.davTaskCalendar');
  }
  return t('integrations.davCalendar');
}

function getCollectionGroups(collections: DAVCollection[], t: (key: TranslationKey) => string) {
  return [
    { key: 'calendar', label: t('integrations.davCalendar'), items: collections.filter((collection) => collection.kind === 'calendar' && collection.supportsEvents && !collection.supportsTasks) },
    { key: 'tasks', label: t('integrations.davTaskCalendar'), items: collections.filter((collection) => collection.kind === 'calendar' && collection.supportsTasks) },
    { key: 'addressbook', label: t('integrations.davAddressbook'), items: collections.filter((collection) => collection.kind === 'addressbook') }
  ].filter((group) => group.items.length > 0);
}

function connectionToInput(connection: CalDAVConnection): CalDAVConnectionInput {
  return {
    displayName: connection.displayName || defaultConnection.displayName,
    baseUrl: connection.baseUrl || '',
    username: connection.username || '',
    password: '',
    syncEnabled: connection.syncEnabled,
    syncDirection: connection.syncDirection || 'pull',
    syncEvents: connection.syncEvents,
    syncTasks: connection.syncTasks,
    syncContacts: connection.syncContacts ?? true,
    syncIntervalMinutes: connection.syncIntervalMinutes || 60,
    syncWindowPastDays: connection.syncWindowPastDays || 30,
    syncWindowFutureDays: connection.syncWindowFutureDays || 365
  };
}
