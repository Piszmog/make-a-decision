-- ============================================================
-- Revert user scoping changes
-- ============================================================

PRAGMA foreign_keys=OFF;

-- ============================================================
-- Recreate original options table (no user_id)
-- ============================================================

CREATE TABLE options_old (
  id INTEGER PRIMARY KEY,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
  name TEXT NOT NULL,
  bio TEXT,
  duration_minutes INTEGER NULL,
  weight INTEGER DEFAULT 1 CHECK (weight >= 1 AND weight <= 10),
  CHECK (
    duration_minutes IS NULL
    OR (
      duration_minutes >= 0
      AND duration_minutes <= 1440
    )
  )
);

DROP TABLE options;
ALTER TABLE options_old RENAME TO options;

-- ============================================================
-- Recreate original tags table (no user_id)
-- ============================================================

CREATE TABLE tags_old (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

DROP TABLE tags;
ALTER TABLE tags_old RENAME TO tags;

-- ============================================================
-- Recreate original option_tags junction table
-- ============================================================

CREATE TABLE option_tags_old (
  option_id INTEGER NOT NULL REFERENCES options(id) ON DELETE CASCADE,
  tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
  PRIMARY KEY (option_id, tag_id)
);

DROP TABLE option_tags;
ALTER TABLE option_tags_old RENAME TO option_tags;

CREATE INDEX idx_option_tags_option_id ON option_tags(option_id);
CREATE INDEX idx_option_tags_tag_id ON option_tags(tag_id);

PRAGMA foreign_keys=ON;

-- Restore default options
INSERT INTO options (name, weight) VALUES
  ('Go for a walk', 1),
  ('Read a book', 1),
  ('Watch a movie', 1),
  ('Call a friend', 1),
  ('Try a new recipe', 1),
  ('Listen to music', 1);
