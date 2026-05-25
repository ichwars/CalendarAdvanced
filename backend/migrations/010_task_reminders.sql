ALTER TABLE tasks ADD COLUMN reminder_at TEXT;
ALTER TABLE tasks ADD COLUMN reminder_delivered_at TEXT;

CREATE INDEX IF NOT EXISTS idx_tasks_user_reminder ON tasks(user_id, reminder_at, reminder_delivered_at);
