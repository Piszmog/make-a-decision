-- ============================================================
-- Add user scoping to options and tags tables
-- ============================================================

-- Disable foreign keys temporarily (required for table recreation)
PRAGMA foreign_keys=OFF;

-- ============================================================
-- Recreate options table with user_id and proper constraints
-- ============================================================

CREATE TABLE options_new (
  id INTEGER PRIMARY KEY,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
  name TEXT NOT NULL,
  bio TEXT,
  duration_minutes INTEGER NULL,
  weight INTEGER DEFAULT 1 CHECK (weight >= 1 AND weight <= 10),
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  CHECK (
    duration_minutes IS NULL
    OR (
      duration_minutes >= 0
      AND duration_minutes <= 1440
    )
  )
);

-- Drop old table and rename new one
DROP TABLE options;
ALTER TABLE options_new RENAME TO options;

-- Create indexes for performance
CREATE INDEX idx_options_user_id ON options(user_id);
CREATE INDEX idx_options_created_at ON options(created_at);

-- ============================================================
-- Recreate tags table with user_id and proper constraints
-- ============================================================

CREATE TABLE tags_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
  UNIQUE(name, user_id)
);

-- Drop old table and rename new one
DROP TABLE tags;
ALTER TABLE tags_new RENAME TO tags;

-- Create indexes
CREATE INDEX idx_tags_user_id ON tags(user_id);
CREATE INDEX idx_tags_name ON tags(name);

-- ============================================================
-- Recreate option_tags junction table (ensure FK constraints)
-- ============================================================

CREATE TABLE option_tags_new (
  option_id INTEGER NOT NULL REFERENCES options(id) ON DELETE CASCADE,
  tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
  PRIMARY KEY (option_id, tag_id)
);

-- Drop old table and rename new one
DROP TABLE option_tags;
ALTER TABLE option_tags_new RENAME TO option_tags;

-- Create indexes
CREATE INDEX idx_option_tags_option_id ON option_tags(option_id);
CREATE INDEX idx_option_tags_tag_id ON option_tags(tag_id);

-- Re-enable foreign keys
PRAGMA foreign_keys=ON;

-- Note: All default options and existing data have been removed
-- Users will start with empty options and create their own
