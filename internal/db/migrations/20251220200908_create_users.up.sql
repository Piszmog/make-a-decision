CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
  CHECK (length(email) > 0 AND email LIKE '%_@__%.__%')
);

CREATE INDEX idx_users_email ON users(email);
