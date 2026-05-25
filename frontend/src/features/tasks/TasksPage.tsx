import { type FormEvent, useEffect, useMemo, useState } from 'react';
import { api } from '../../shared/api';
import { Button } from '../../shared/components/Button';
import { ConfirmDialog } from '../../shared/components/ConfirmDialog';
import { DateTimePicker } from '../../shared/components/DateTimePicker';
import { EmptyState } from '../../shared/components/EmptyState';
import { Icon } from '../../shared/components/Icon';
import { RichTextField } from '../../shared/components/RichTextField';
import { SelectField } from '../../shared/components/SelectField';
import { useI18n } from '../../shared/i18n';
import { getGeneralPreferences } from '../../shared/preferences';
import type { TaskItem } from '../../shared/types';

type TaskFilter = 'all' | 'open' | 'completed';

export function TasksPage() {
  const { locale, t } = useI18n();
  const [tasks, setTasks] = useState<TaskItem[]>([]);
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState<TaskFilter>('open');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<TaskItem | null>(null);
  const [deleteTask, setDeleteTask] = useState<TaskItem | null>(null);
  const [deleting, setDeleting] = useState(false);

  async function load() {
    const params = new URLSearchParams({ q: query, limit: '500' });
    if (filter === 'open') {
      params.set('completed', 'false');
    }
    if (filter === 'completed') {
      params.set('completed', 'true');
    }
    const response = await api.tasks(params);
    setTasks(response.items);
  }

  useEffect(() => {
    void load();
  }, [filter, query]);

  async function saveTask(body: Partial<TaskItem>) {
    if (editingTask) {
      await api.updateTask(editingTask.id, toTaskPayload({ ...editingTask, ...body }));
      setEditingTask(null);
    } else {
      await api.createTask(toTaskPayload(body));
      setDialogOpen(false);
    }
    await load();
  }

  async function toggleTask(task: TaskItem) {
    await api.updateTask(task.id, toTaskPayload({ ...task, completed: !task.completed }));
    await load();
  }

  async function confirmDeleteTask() {
    if (!deleteTask) {
      return;
    }
    setDeleting(true);
    try {
      await api.deleteTask(deleteTask.id);
      setDeleteTask(null);
      await load();
    } finally {
      setDeleting(false);
    }
  }

  const stats = useMemo(() => ({
    open: tasks.filter((task) => !task.completed).length,
    completed: tasks.filter((task) => task.completed).length
  }), [tasks]);

  return (
    <div className="page">
      <header className="page-header tasks-page-header">
        <div>
          <h1>{t('tasks.title')}</h1>
        </div>
        <div className="events-page-header__actions">
          <div className={query ? 'events-search events-search--active' : 'events-search'}>
            <Icon name="search" />
            <input aria-label={t('tasks.search')} value={query} onChange={(event) => setQuery(event.currentTarget.value)} placeholder={t('tasks.search')} />
            {query && (
              <button type="button" onClick={() => setQuery('')} aria-label={t('common.clear')} title={t('common.clear')}>
                <Icon name="x" />
              </button>
            )}
          </div>
          <Button onClick={() => setDialogOpen(true)} type="button">+ {t('tasks.add')}</Button>
        </div>
      </header>

      <div className="tasks-toolbar" role="tablist" aria-label={t('tasks.filter')}>
        <button className={filter === 'open' ? 'tasks-tab active' : 'tasks-tab'} onClick={() => setFilter('open')} type="button">{t('tasks.open')} <span>{stats.open}</span></button>
        <button className={filter === 'completed' ? 'tasks-tab active' : 'tasks-tab'} onClick={() => setFilter('completed')} type="button">{t('tasks.completed')} <span>{stats.completed}</span></button>
        <button className={filter === 'all' ? 'tasks-tab active' : 'tasks-tab'} onClick={() => setFilter('all')} type="button">{t('tasks.all')}</button>
      </div>

      {!tasks.length ? <EmptyState message={t('tasks.empty')} /> : (
        <div className="table-wrap events-table-wrap">
          <table className="events-table tasks-table">
            <thead>
              <tr>
                <th>{t('common.title')}</th>
                <th>{t('tasks.dueAt')}</th>
                <th>{t('tasks.reminderAt')}</th>
                <th>{t('tasks.priority')}</th>
                <th>{t('tasks.status')}</th>
                <th className="events-table__actions-heading">{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {tasks.map((task) => (
                <tr key={task.id}>
                  <td>
                    <div className="tasks-title-cell">
                      <button className={task.completed ? 'task-check active' : 'task-check'} type="button" onClick={() => void toggleTask(task)} aria-label={task.completed ? t('tasks.markOpen') : t('tasks.markCompleted')} title={task.completed ? t('tasks.markOpen') : t('tasks.markCompleted')}>
                        {task.completed && <Icon name="check" />}
                      </button>
                      <div>
                        <strong>{task.title}</strong>
                        {task.davSynced && <span className="dav-sync-badge" title={t('common.davSynced')}>DAV</span>}
                        {task.description && <p>{task.description}</p>}
                      </div>
                    </div>
                  </td>
                  <td>{task.dueAt ? new Date(task.dueAt).toLocaleString(locale, { dateStyle: 'short', timeStyle: 'short' }) : '-'}</td>
                  <td><span className={task.reminderAt ? 'event-status event-status--completed' : 'event-status'}>{task.reminderAt ? t('tasks.notificationOn') : t('tasks.notificationOff')}</span></td>
                  <td><span className={`task-priority task-priority--${task.priority}`}>{taskPriorityLabel(task.priority, t)}</span></td>
                  <td><span className={task.completed ? 'event-status event-status--completed' : 'event-status event-status--open'}>{task.completed ? t('tasks.completed') : t('tasks.open')}</span></td>
                  <td>
                    <div className="events-table__actions">
                      <button className="icon-button" type="button" onClick={() => setEditingTask(task)} aria-label={t('common.edit')} title={t('common.edit')}>
                        <Icon name="pencil" />
                      </button>
                      <button className="icon-button icon-button--danger" type="button" onClick={() => setDeleteTask(task)} aria-label={t('common.delete')} title={t('common.delete')}>
                        <Icon name="trash" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {(dialogOpen || editingTask) && (
        <TaskDialog
          task={editingTask}
          onClose={() => {
            setDialogOpen(false);
            setEditingTask(null);
          }}
          onSave={saveTask}
        />
      )}
      {deleteTask && (
        <ConfirmDialog
          busy={deleting}
          message={`${t('tasks.deleteConfirm')}\n\n${t('tasks.deleteDavWarning')}`}
          onCancel={() => setDeleteTask(null)}
          onConfirm={() => void confirmDeleteTask()}
          title={t('tasks.deleteTitle')}
        />
      )}
    </div>
  );
}

function TaskDialog({ task, onClose, onSave }: { task?: TaskItem | null; onClose: () => void; onSave: (body: Partial<TaskItem>) => Promise<void> }) {
  const { t } = useI18n();
  const preferences = useMemo(() => getGeneralPreferences(), []);
  const [title, setTitle] = useState(task?.title ?? '');
  const [description, setDescription] = useState(task?.description ?? '');
  const [dueAt, setDueAt] = useState(task?.dueAt ? toLocalDateTimeValue(new Date(task.dueAt)) : toLocalDateTimeValue(roundToNextStep(new Date(), Number(getGeneralPreferences().timeGrid))));
  const [reminderEnabled, setReminderEnabled] = useState(Boolean(task?.reminderAt));
  const [showInCalendar, setShowInCalendar] = useState(task?.showInCalendar ?? false);
  const [priority, setPriority] = useState<TaskItem['priority']>(task?.priority ?? 'normal');
  const [saving, setSaving] = useState(false);
  const priorityOptions = [
    { value: 'low', label: t('tasks.priority.low') },
    { value: 'normal', label: t('tasks.priority.normal') },
    { value: 'high', label: t('tasks.priority.high') }
  ] satisfies { value: TaskItem['priority']; label: string }[];

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    try {
      await onSave({
        title,
        description,
        dueAt: dueAt ? new Date(dueAt).toISOString() : undefined,
        reminderAt: reminderEnabled && dueAt ? new Date(dueAt).toISOString() : undefined,
        priority,
        completed: task?.completed ?? false,
        showInCalendar
      });
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="modal-backdrop" onMouseDown={(event) => {
      if (event.target === event.currentTarget) {
        onClose();
      }
    }}>
      <section className="modal" role="dialog" aria-modal="true" aria-labelledby="task-dialog-title">
        <header className="modal__header">
          <h2 id="task-dialog-title">{task ? t('tasks.edit') : t('tasks.new')}</h2>
          <button className="icon-button" type="button" onClick={onClose} aria-label={t('common.close')} title={t('common.close')}>
            <Icon name="x" />
          </button>
        </header>
        <form className="grid-form modal__form" onSubmit={(event) => void submit(event)}>
          <label className="field field--wide" htmlFor="taskTitle">
            <span>{t('common.title')}</span>
            <input id="taskTitle" name="taskTitle" value={title} onChange={(event) => setTitle(event.currentTarget.value)} required />
          </label>
          <DateTimePicker label={t('tasks.dueAt')} minuteStep={Number(preferences.timeGrid)} name="taskDueAt" value={dueAt} onChange={setDueAt} />
          <SelectField label={t('tasks.priority')} value={priority} onChange={setPriority} options={priorityOptions} />
          <label className="check check--switch task-reminder-toggle">
            <input checked={showInCalendar} onChange={(event) => setShowInCalendar(event.currentTarget.checked)} type="checkbox" />
            <span>{t('tasks.showInCalendar')}</span>
          </label>
          <label className="check check--switch task-reminder-toggle">
            <input checked={reminderEnabled} onChange={(event) => setReminderEnabled(event.currentTarget.checked)} type="checkbox" />
            <span>{t('tasks.reminderAt')}</span>
          </label>
          <RichTextField id="task-description" label={t('common.description')} value={description} onChange={setDescription} />
          <div className="button-row modal__actions">
            <Button disabled={saving} type="submit">{task ? t('common.save') : t('tasks.add')}</Button>
            <Button disabled={saving} onClick={onClose} type="button" variant="ghost">{t('common.cancel')}</Button>
          </div>
        </form>
      </section>
    </div>
  );
}

function toLocalDateTimeValue(date: Date): string {
  const offset = date.getTimezoneOffset();
  const local = new Date(date.getTime() - offset * 60_000);
  return local.toISOString().slice(0, 16);
}

function roundToNextStep(date: Date, minuteStep: number): Date {
  const next = new Date(date);
  next.setSeconds(0, 0);
  const step = minuteStep > 0 ? minuteStep : 15;
  const minutes = next.getMinutes();
  const rounded = Math.ceil(minutes / step) * step;
  next.setMinutes(rounded);
  return next;
}

function taskPriorityLabel(priority: TaskItem['priority'], t: ReturnType<typeof useI18n>['t']): string {
  if (priority === 'low') return t('tasks.priority.low');
  if (priority === 'high') return t('tasks.priority.high');
  return t('tasks.priority.normal');
}

function toTaskPayload(task: Partial<TaskItem>): Partial<TaskItem> {
  return {
    title: task.title,
    description: task.description,
    dueAt: task.dueAt || undefined,
    reminderAt: task.reminderAt || undefined,
    priority: task.priority ?? 'normal',
    completed: task.completed ?? false,
    showInCalendar: task.showInCalendar ?? false
  };
}
