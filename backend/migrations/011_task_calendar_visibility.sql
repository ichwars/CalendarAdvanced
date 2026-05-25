ALTER TABLE tasks ADD COLUMN show_in_calendar INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_tasks_user_calendar_due ON tasks(user_id, show_in_calendar, due_at);
