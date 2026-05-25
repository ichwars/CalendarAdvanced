ALTER TABLE users ADD COLUMN username TEXT NOT NULL DEFAULT '';

UPDATE users
SET username = 'user' || id
WHERE username = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
