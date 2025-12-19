-- Create the options table with all required columns
CREATE TABLE IF NOT EXISTS options (
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

-- Insert default options
INSERT INTO options (name, weight) VALUES
  ('Go for a walk', 1),
  ('Read a book', 1),
  ('Watch a movie', 1),
  ('Call a friend', 1),
  ('Try a new recipe', 1),
  ('Listen to music', 1);
