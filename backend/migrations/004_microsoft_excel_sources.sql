CREATE TABLE IF NOT EXISTS microsoft_excel_sources (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  file_url TEXT NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  last_checked_at TEXT,
  last_imported_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
