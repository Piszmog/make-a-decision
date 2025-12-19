-- Create tags table
CREATE TABLE IF NOT EXISTS tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE COLLATE NOCASE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create junction table for option-tag relationships
CREATE TABLE IF NOT EXISTS option_tags (
  option_id INTEGER NOT NULL,
  tag_id INTEGER NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
  PRIMARY KEY (option_id, tag_id),
  FOREIGN KEY (option_id) REFERENCES options(id) ON DELETE CASCADE,
  FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_option_tags_option_id ON option_tags(option_id);
CREATE INDEX IF NOT EXISTS idx_option_tags_tag_id ON option_tags(tag_id);
