-- Seed data for E2E testing

-- Create options with various tag combinations for testing
INSERT INTO options (id, name, bio, duration_minutes, weight, created_at) VALUES 
(1, 'Video Games', NULL, 60, 5, datetime('now')),
(2, 'Reading a Book', NULL, 30, 3, datetime('now')),
(3, 'Going for a Run', NULL, 45, 2, datetime('now')),
(4, 'Meditation', NULL, 15, 1, datetime('now')),
(5, 'Watch Movie Marathon', NULL, 180, 1, datetime('now')),
(6, 'Board Games', NULL, 90, 4, datetime('now'));

-- Create tags
INSERT INTO tags (id, name, created_at) VALUES
(1, 'indoor', datetime('now')),
(2, 'outdoor', datetime('now')),
(3, 'gaming', datetime('now')),
(4, 'relaxing', datetime('now')),
(5, 'active', datetime('now'));

-- Associate options with tags
-- Option 1: Video Games - gaming, indoor
INSERT INTO option_tags (option_id, tag_id, created_at) VALUES
(1, 3, datetime('now')),
(1, 1, datetime('now'));

-- Option 2: Reading - indoor, relaxing
INSERT INTO option_tags (option_id, tag_id, created_at) VALUES
(2, 1, datetime('now')),
(2, 4, datetime('now'));

-- Option 3: Running - outdoor, active
INSERT INTO option_tags (option_id, tag_id, created_at) VALUES
(3, 2, datetime('now')),
(3, 5, datetime('now'));

-- Option 4: Meditation - NO TAGS (critical for testing!)

-- Option 5: Movie Marathon - indoor, relaxing
INSERT INTO option_tags (option_id, tag_id, created_at) VALUES
(5, 1, datetime('now')),
(5, 4, datetime('now'));

-- Option 6: Board Games - indoor, gaming
INSERT INTO option_tags (option_id, tag_id, created_at) VALUES
(6, 1, datetime('now')),
(6, 3, datetime('now'));
