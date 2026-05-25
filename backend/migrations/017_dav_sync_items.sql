CREATE TABLE IF NOT EXISTS dav_sync_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  collection_url TEXT NOT NULL,
  resource_url TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('event','task','contact')),
  local_id INTEGER NOT NULL,
  uid TEXT,
  etag TEXT,
  updated_at TEXT NOT NULL,
  UNIQUE(user_id, resource_url, kind)
);

CREATE INDEX IF NOT EXISTS idx_dav_sync_items_user_kind ON dav_sync_items(user_id, kind);
