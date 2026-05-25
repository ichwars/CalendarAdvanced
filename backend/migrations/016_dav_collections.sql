CREATE TABLE IF NOT EXISTS dav_collections (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  display_name TEXT NOT NULL,
  url TEXT NOT NULL,
  selected INTEGER NOT NULL DEFAULT 1,
  supports_events INTEGER NOT NULL DEFAULT 0,
  supports_tasks INTEGER NOT NULL DEFAULT 0,
  ctag TEXT,
  sync_token TEXT,
  last_seen_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(user_id, url)
);
