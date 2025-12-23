-- Seed data for E2E testing

-- Clear existing data
DELETE FROM option_tags;
DELETE FROM tags;
DELETE FROM options;
DELETE FROM sessions;
DELETE FROM users;

-- Create test user
-- Password is 'password123' hashed with bcrypt cost 10
INSERT INTO users (id, email, password_hash, created_at) VALUES 
(1, 'test@example.com', '$2a$10$08Tf43MlgLm0FkwgpH3I.uo8wp92YOfhnNhZq2oaRVmrHT2T96alG', datetime('now'));

-- Create options with various tag combinations for testing
INSERT INTO options (name, bio, duration_minutes, weight, user_id, created_at) VALUES 
('Video Games', NULL, 60, 5, 1, datetime('now')),
('Reading a Book', NULL, 30, 3, 1, datetime('now')),
('Going for a Run', NULL, 45, 2, 1, datetime('now')),
('Meditation', NULL, 15, 1, 1, datetime('now')),
('Watch Movie Marathon', NULL, 180, 1, 1, datetime('now')),
('Board Games', NULL, 90, 4, 1, datetime('now'));

-- Create tags
INSERT INTO tags (name, user_id, created_at) VALUES
('indoor', 1, datetime('now')),
('outdoor', 1, datetime('now')),
('gaming', 1, datetime('now')),
('relaxing', 1, datetime('now')),
('active', 1, datetime('now'));

-- Associate options with tags
-- Option 1: Video Games - gaming, indoor
INSERT INTO option_tags (option_id, tag_id, created_at) 
SELECT o.id, t.id, datetime('now') 
FROM options o, tags t 
WHERE o.name = 'Video Games' AND t.name IN ('gaming', 'indoor');

-- Option 2: Reading - indoor, relaxing
INSERT INTO option_tags (option_id, tag_id, created_at) 
SELECT o.id, t.id, datetime('now') 
FROM options o, tags t 
WHERE o.name = 'Reading a Book' AND t.name IN ('indoor', 'relaxing');

-- Option 3: Running - outdoor, active
INSERT INTO option_tags (option_id, tag_id, created_at) 
SELECT o.id, t.id, datetime('now') 
FROM options o, tags t 
WHERE o.name = 'Going for a Run' AND t.name IN ('outdoor', 'active');

-- Option 4: Meditation - NO TAGS (critical for testing!)

-- Option 5: Movie Marathon - indoor, relaxing
INSERT INTO option_tags (option_id, tag_id, created_at) 
SELECT o.id, t.id, datetime('now') 
FROM options o, tags t 
WHERE o.name = 'Watch Movie Marathon' AND t.name IN ('indoor', 'relaxing');

-- Option 6: Board Games - indoor, gaming
INSERT INTO option_tags (option_id, tag_id, created_at) 
SELECT o.id, t.id, datetime('now') 
FROM options o, tags t 
WHERE o.name = 'Board Games' AND t.name IN ('indoor', 'gaming');
