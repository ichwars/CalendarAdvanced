import { useEffect, useState } from 'react';
import { Button } from '../../shared/components/Button';
import { Card } from '../../shared/components/Card';
import { FormField } from '../../shared/components/FormField';
import { SelectField } from '../../shared/components/SelectField';
import { api, downloadURL, type ExcelImportPreview, type ExcelImportResult, type ICSImportPreview, type ICSImportResult } from '../../shared/api';
import { useI18n } from '../../shared/i18n';
import { getGeneralPreferences } from '../../shared/preferences';
import type { BackupPreview, Calendar } from '../../shared/types';

export function ExportsPage() {
  const { locale, t } = useI18n();
  const [calendars, setCalendars] = useState<Calendar[]>([]);
  const [calendarSelection, setCalendarSelection] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [excelFile, setExcelFile] = useState<File | null>(null);
  const [backupFile, setBackupFile] = useState<File | null>(null);
  const [excelAllDay, setExcelAllDay] = useState(false);
  const [excelEmployeeQuery, setExcelEmployeeQuery] = useState('');
  const [preview, setPreview] = useState<ICSImportPreview | null>(null);
  const [excelPreview, setExcelPreview] = useState<ExcelImportPreview | null>(null);
  const [result, setResult] = useState<ICSImportResult | null>(null);
  const [excelResult, setExcelResult] = useState<ExcelImportResult | null>(null);
  const [backupPreview, setBackupPreview] = useState<BackupPreview | null>(null);
  const [backupDownload, setBackupDownload] = useState<{ name: string; url: string } | null>(null);
  const [status, setStatus] = useState<'idle' | 'loading' | 'importing' | 'error'>('idle');
  const [message, setMessage] = useState('');
  const calendarOptions = calendars.map((calendar) => ({ value: String(calendar.id), label: calendar.name }));
  const targetCalendarOptions = [
    { value: '', label: t('importExport.chooseTargetCalendar') },
    ...calendarOptions
  ];

  useEffect(() => {
    api.calendars().then((response) => {
      setCalendars(response.items);
    }).catch(() => undefined);
  }, []);

  useEffect(() => () => {
    if (backupDownload) {
      URL.revokeObjectURL(backupDownload.url);
    }
  }, [backupDownload]);

  function buildImportFormData(targetCalendarId?: string): FormData | null {
    if (!file) {
      setStatus('error');
      setMessage(t('importExport.importFileRequired'));
      return null;
    }
    const formData = new FormData();
    formData.append('file', file);
    formData.append('timezone', getGeneralPreferences().timezone);
    if (targetCalendarId) {
      formData.append('calendarId', targetCalendarId);
    }
    return formData;
  }

  function buildExcelImportFormData(targetCalendarId?: string): FormData | null {
    if (!excelFile) {
      setStatus('error');
      setMessage(t('importExport.excelFileRequired'));
      return null;
    }
    const preferences = getGeneralPreferences();
    const formData = new FormData();
    formData.append('file', excelFile);
    formData.append('timezone', preferences.timezone);
    formData.append('allDay', String(excelAllDay));
    formData.append('employeeQuery', excelEmployeeQuery);
    formData.append('workStart', preferences.workingHoursStart);
    formData.append('workEnd', preferences.workingHoursEnd);
    if (targetCalendarId) {
      formData.append('calendarId', targetCalendarId);
    }
    return formData;
  }

  async function previewImport() {
    const formData = buildImportFormData();
    if (!formData) return;
    setStatus('loading');
    setMessage('');
    setResult(null);
    try {
      const response = await api.previewICSImport(formData);
      setPreview(normalizePreview(response));
      setStatus('idle');
    } catch (error) {
      setStatus('error');
      setMessage((error as { message?: string }).message ?? t('importExport.importFailed'));
    }
  }

  async function runImport() {
    if (!preview) {
      setStatus('error');
      setMessage(t('importExport.previewRequired'));
      return;
    }
    if (!calendarSelection) {
      setStatus('error');
      setMessage(t('importExport.targetCalendarRequired'));
      return;
    }
    setStatus('importing');
    setMessage('');
    try {
      const formData = buildImportFormData(calendarSelection);
      if (!formData) return;
      const response = await api.importICS(formData);
      setResult(normalizeImportResult(response));
      setCalendarSelection('');
      setStatus('idle');
    } catch (error) {
      setStatus('error');
      setMessage((error as { message?: string }).message ?? t('importExport.importFailed'));
    }
  }

  async function previewExcelImport() {
    const formData = buildExcelImportFormData();
    if (!formData) return;
    setStatus('loading');
    setMessage('');
    setExcelResult(null);
    try {
      const response = await api.previewExcelImport(formData);
      setExcelPreview(normalizeExcelPreview(response));
      setStatus('idle');
    } catch (error) {
      setStatus('error');
      setMessage((error as { message?: string }).message ?? t('importExport.excelImportFailed'));
    }
  }

  async function runExcelImport() {
    if (!excelPreview) {
      setStatus('error');
      setMessage(t('importExport.previewRequired'));
      return;
    }
    if (!calendarSelection) {
      setStatus('error');
      setMessage(t('importExport.targetCalendarRequired'));
      return;
    }
    setStatus('importing');
    setMessage('');
    try {
      const formData = buildExcelImportFormData(calendarSelection);
      if (!formData) return;
      const response = await api.importExcel(formData);
      setExcelResult(normalizeExcelResult(response));
      setCalendarSelection('');
      setStatus('idle');
    } catch (error) {
      setStatus('error');
      setMessage((error as { message?: string }).message ?? t('importExport.excelImportFailed'));
    }
  }

  function buildBackupFormData(): FormData | null {
    if (!backupFile) {
      setStatus('error');
      setMessage(t('importExport.backupFileRequired'));
      return null;
    }
    const formData = new FormData();
    formData.append('file', backupFile);
    return formData;
  }

  async function previewBackupRestore() {
    const formData = buildBackupFormData();
    if (!formData) return;
    setStatus('loading');
    setMessage('');
    try {
      const response = await api.previewBackupRestore(formData);
      setBackupPreview(response);
      setStatus('idle');
    } catch (error) {
      setStatus('error');
      setMessage((error as { message?: string }).message ?? t('importExport.backupRestoreFailed'));
    }
  }

  async function downloadBackup() {
    setStatus('loading');
    setMessage('');
    try {
      const blob = await api.downloadBackup();
      const payload = await blob.text();
      const fileBlob = new Blob([payload], { type: 'application/json' });
      const parsed = JSON.parse(payload) as BackupEnvelopePayload;
      const createdAt = parsed.createdAt ? new Date(parsed.createdAt) : new Date();
      const filename = `calendaradvanced-backup-${formatBackupFilenameDate(createdAt)}.json`;
      const url = URL.createObjectURL(fileBlob);
      setBackupDownload((current) => {
        if (current) {
          URL.revokeObjectURL(current.url);
        }
        return { name: filename, url };
      });
      setBackupPreview(backupPreviewFromEnvelope(parsed));
      triggerDownload(url, filename);
      setStatus('idle');
      setMessage(t('importExport.backupDownloadReady').replace('{name}', filename));
    } catch (error) {
      setStatus('error');
      setMessage((error as { message?: string }).message ?? t('importExport.backupDownloadFailed'));
    }
  }

  async function restoreBackup() {
    if (!backupPreview) {
      setStatus('error');
      setMessage(t('importExport.backupPreviewRequired'));
      return;
    }
    if (!window.confirm(t('importExport.backupRestoreConfirm'))) {
      return;
    }
    const formData = buildBackupFormData();
    if (!formData) return;
    setStatus('importing');
    setMessage('');
    try {
      const response = await api.restoreBackup(formData);
      setBackupPreview(response);
      setStatus('idle');
      setMessage(t('importExport.backupRestoreSuccess'));
      void api.calendars().then((calendarResponse) => setCalendars(calendarResponse.items));
    } catch (error) {
      setStatus('error');
      setMessage((error as { message?: string }).message ?? t('importExport.backupRestoreFailed'));
    }
  }

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>{t('importExport.title')}</h1>
          <p>{t('importExport.subtitle')}</p>
        </div>
      </header>
      <div className="settings-grid settings-grid--tiles import-export-grid">
        <div className="import-export-section-title">
          <h2>{t('importExport.importSection')}</h2>
          <p>{t('importExport.importSectionHelp')}</p>
        </div>
        <Card className="import-export-card">
          <div className="stack">
            <h2>{t('importExport.targetTitle')}</h2>
            <p>{t('importExport.targetInfo')}</p>
            <div className="grid-form">
              <SelectField label={t('importExport.targetCalendar')} value={calendarSelection} onChange={setCalendarSelection} options={targetCalendarOptions} />
            </div>
          </div>
        </Card>
        <Card className="import-export-card">
          <div className="stack">
            <h2>{t('importExport.importICS')}</h2>
            <p>{t('importExport.importInfo')}</p>
            <div className="grid-form">
              <label className="field file-field">
                <span>{t('importExport.icsFile')}</span>
                <span className="file-picker">
                  <input
                    accept=".ics,text/calendar"
                    onChange={(event) => {
                      setFile(event.currentTarget.files?.[0] ?? null);
                      setPreview(null);
                      setResult(null);
                      setStatus('idle');
                      setMessage('');
                    }}
                    type="file"
                  />
                  <span className="file-picker__action">{t('importExport.chooseFile')}</span>
                  <span className="file-picker__name">{file?.name ?? t('importExport.noFileSelected')}</span>
                </span>
              </label>
            </div>
            <div className="button-row">
              <Button disabled={status === 'loading'} onClick={() => void previewImport()} type="button">{t('importExport.preview')}</Button>
              <Button disabled={status === 'importing' || !calendarSelection || !preview} onClick={() => void runImport()} type="button">{t('importExport.import')}</Button>
            </div>
            {status === 'error' && <p className="error">{message}</p>}
            {preview && (
              <section className="import-preview">
                <div className="import-preview__stats">
                  <span>{t('importExport.events')}: <strong>{preview.eventCount}</strong></span>
                  <span>{t('events.allDay')}: <strong>{preview.allDayCount}</strong></span>
                  <span>{t('events.recurrence')}: <strong>{preview.recurringCount}</strong></span>
                </div>
                <p>{formatPreviewRange(preview, locale)}</p>
                <div className="cards-list">
                  {(preview.samples ?? []).map((sample) => (
                    <article className="event-row" key={`${sample.startsAt}-${sample.title}`}>
                      <div>
                        <strong>{sample.title}</strong>
                        <p>{sample.location}</p>
                      </div>
                      <time>{new Date(sample.startsAt).toLocaleString(locale)}</time>
                    </article>
                  ))}
                </div>
                <WarningList warnings={preview.warnings} />
              </section>
            )}
            {result && (
              <>
                <p className="success">{t('importExport.importResult').replace('{imported}', String(result.imported)).replace('{skipped}', String(result.skipped))}</p>
                <WarningList warnings={result.warnings} />
              </>
            )}
          </div>
        </Card>
        <div className="import-export-section-title">
          <h2>{t('importExport.exportSection')}</h2>
          <p>{t('importExport.exportSectionHelp')}</p>
        </div>
        <Card className="import-export-card">
          <div className="stack">
            <h2>{t('importExport.importExcel')}</h2>
            <p>{t('importExport.excelInfo')}</p>
            <div className="import-target-grid">
              <label className="field file-field">
                <span>{t('importExport.excelFile')}</span>
                <span className="file-picker">
                  <input
                    accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                    onChange={(event) => {
                      setExcelFile(event.currentTarget.files?.[0] ?? null);
                      setExcelPreview(null);
                      setExcelResult(null);
                      setStatus('idle');
                      setMessage('');
                    }}
                    type="file"
                  />
                  <span className="file-picker__action">{t('importExport.chooseFile')}</span>
                  <span className="file-picker__name">{excelFile?.name ?? t('importExport.noFileSelected')}</span>
                </span>
              </label>
              <label className="check check--switch import-reminder-grid__toggle">
                <input checked={excelAllDay} onChange={(event) => setExcelAllDay(event.currentTarget.checked)} type="checkbox" />
                <span>{t('importExport.excelAllDay')}</span>
              </label>
              <FormField
                label={t('importExport.excelEmployeeFilter')}
                onChange={(event) => {
                  setExcelEmployeeQuery(event.currentTarget.value);
                  setExcelPreview(null);
                  setExcelResult(null);
                }}
                placeholder={t('importExport.excelEmployeeFilterPlaceholder')}
                value={excelEmployeeQuery}
              />
            </div>
            <p className="source">{t('importExport.excelMapping')}</p>
            <div className="button-row">
              <Button disabled={status === 'loading'} onClick={() => void previewExcelImport()} type="button">{t('importExport.preview')}</Button>
              <Button disabled={status === 'importing' || !calendarSelection || !excelPreview} onClick={() => void runExcelImport()} type="button">{t('importExport.import')}</Button>
            </div>
            {status === 'error' && <p className="error">{message}</p>}
            {excelPreview && (
              <section className="import-preview">
                <div className="import-preview__stats">
                  <span>{t('importExport.events')}: <strong>{excelPreview.eventCount}</strong></span>
                  <span>{t('importExport.excelRows')}: <strong>{excelPreview.rows}</strong></span>
                  <span>{t('importExport.excelCancelledRows')}: <strong>{excelPreview.cancelledRows}</strong></span>
                </div>
                <p>{formatExcelPreviewRange(excelPreview, locale)}</p>
                <div className="cards-list">
                  {(excelPreview.samples ?? []).map((sample) => (
                    <article className="event-row" key={`${sample.sheet}-${sample.week}-${sample.weekday}-${sample.title}`}>
                      <div>
                        <strong>{sample.title}</strong>
                        <p>{sample.weekday}, KW {sample.week} · {sample.employee || '-'} · {sample.completed ? t('importExport.excelCompleted') : t('importExport.excelOpen')}</p>
                      </div>
                      <time>{new Date(sample.date).toLocaleDateString(locale)}</time>
                    </article>
                  ))}
                </div>
                <WarningList warnings={excelPreview.warnings} />
              </section>
            )}
            {excelResult && (
              <>
                <p className="success">{t('importExport.excelImportResult').replace('{imported}', String(excelResult.imported)).replace('{updated}', String(excelResult.updated)).replace('{skipped}', String(excelResult.skipped))}</p>
                {excelResult.skippedCancelled > 0 && <p className="warning">{t('importExport.excelSkippedCancelled').replace('{count}', String(excelResult.skippedCancelled))}</p>}
                <WarningList warnings={excelResult.warnings} />
              </>
            )}
          </div>
        </Card>
        <Card className="import-export-card">
          <div className="stack">
            <h2>{t('importExport.exportTitle')}</h2>
            <p>{t('importExport.exportInfo')}</p>
            <div className="button-row">
              <Button onClick={() => downloadURL('/api/v1/exports/csv')}>{t('exports.csv')}</Button>
              <Button onClick={() => downloadURL('/api/v1/exports/xlsx')}>{t('exports.xlsx')}</Button>
            </div>
          </div>
        </Card>
        <Card className="import-export-card import-export-card--wide">
          <div className="stack">
            <h2>{t('importExport.backupTitle')}</h2>
            <p>{t('importExport.backupInfo')}</p>
            <div className="button-row">
              <Button disabled={status === 'loading'} onClick={() => void downloadBackup()} type="button">{t('importExport.backupDownload')}</Button>
              {backupDownload && (
                <a className="button button--ghost" download={backupDownload.name} href={backupDownload.url}>{t('importExport.backupDownloadAgain')}</a>
              )}
            </div>
            <div className="grid-form">
              <label className="field file-field">
                <span>{t('importExport.backupFile')}</span>
                <span className="file-picker">
                  <input
                    accept=".json,application/json"
                    onChange={(event) => {
                      setBackupFile(event.currentTarget.files?.[0] ?? null);
                      setBackupPreview(null);
                      setStatus('idle');
                      setMessage('');
                    }}
                    type="file"
                  />
                  <span className="file-picker__action">{t('importExport.chooseFile')}</span>
                  <span className="file-picker__name">{backupFile?.name ?? t('importExport.noFileSelected')}</span>
                </span>
              </label>
            </div>
            <div className="button-row">
              <Button disabled={status === 'loading'} onClick={() => void previewBackupRestore()} type="button" variant="ghost">{t('importExport.backupPreview')}</Button>
              <Button disabled={status === 'importing' || !backupPreview} onClick={() => void restoreBackup()} type="button" variant="danger">{t('importExport.backupRestore')}</Button>
            </div>
            {message && <p className={status === 'error' ? 'error' : 'success'}>{message}</p>}
            {backupPreview && (
              <section className="import-preview">
                <div className="import-preview__stats">
                  <span>{t('calendar.title')}: <strong>{backupPreview.counts.calendars ?? 0}</strong></span>
                  <span>{t('events.title')}: <strong>{backupPreview.counts.events ?? 0}</strong></span>
                  <span>{t('tasks.title')}: <strong>{backupPreview.counts.tasks ?? 0}</strong></span>
                  <span>{t('contacts.title')}: <strong>{backupPreview.counts.contacts ?? 0}</strong></span>
                </div>
                <p>{t('importExport.backupCreatedAt')}: {new Date(backupPreview.createdAt).toLocaleString(locale)}</p>
              </section>
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}

function normalizePreview(response: ICSImportPreview): ICSImportPreview {
  return {
    ...response,
    samples: response.samples ?? [],
    warnings: response.warnings ?? []
  };
}

function normalizeImportResult(response: ICSImportResult): ICSImportResult {
  return {
    ...response,
    warnings: response.warnings ?? []
  };
}

function normalizeExcelPreview(response: ExcelImportPreview): ExcelImportPreview {
  return {
    ...response,
    cancelledRows: response.cancelledRows ?? 0,
    samples: response.samples ?? [],
    warnings: response.warnings ?? []
  };
}

function normalizeExcelResult(response: ExcelImportResult): ExcelImportResult {
  return {
    ...response,
    updated: response.updated ?? 0,
    warnings: response.warnings ?? []
  };
}

interface BackupEnvelopePayload {
  app?: string;
  version?: number;
  createdAt?: string;
  data?: Record<string, unknown>;
}

function backupPreviewFromEnvelope(envelope: BackupEnvelopePayload): BackupPreview {
  const counts: Record<string, number> = {};
  Object.entries(envelope.data ?? {}).forEach(([key, value]) => {
    if (Array.isArray(value)) {
      counts[key] = value.length;
    }
  });
  return {
    app: envelope.app ?? 'CalendarAdvanced',
    version: envelope.version ?? 1,
    createdAt: envelope.createdAt ?? new Date().toISOString(),
    counts
  };
}

function formatBackupFilenameDate(value: Date): string {
  return value.toISOString().replace(/[:.]/g, '-');
}

function triggerDownload(url: string, filename: string): void {
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
}

function formatPreviewRange(preview: ICSImportPreview, locale: string): string {
  if (!preview.rangeStart || !preview.rangeEnd) {
    return '';
  }
  return `${new Date(preview.rangeStart).toLocaleDateString(locale)} - ${new Date(preview.rangeEnd).toLocaleDateString(locale)}`;
}

function formatExcelPreviewRange(preview: ExcelImportPreview, locale: string): string {
  if (!preview.rangeStart || !preview.rangeEnd) {
    return '';
  }
  return `${new Date(preview.rangeStart).toLocaleDateString(locale)} - ${new Date(preview.rangeEnd).toLocaleDateString(locale)}`;
}

function WarningList({ warnings }: { warnings: string[] }) {
  if (!warnings?.length) {
    return null;
  }
  return (
    <ul className="warning-list">
      {warnings.map((warning) => <li key={warning}>{warning}</li>)}
    </ul>
  );
}
