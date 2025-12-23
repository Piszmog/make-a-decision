-- name: GetOptions :many
SELECT
  *
FROM
  options
WHERE
  user_id = ?
ORDER BY
  created_at;

-- name: GetOption :one
SELECT
  *
FROM
  options
WHERE
  id = ? AND user_id = ?
LIMIT
  1;

-- name: CreateOption :one
INSERT INTO
  options (name, bio, duration_minutes, weight, user_id)
VALUES
  (?, ?, ?, ?, ?) RETURNING id,
  name,
  bio,
  duration_minutes,
  weight,
  user_id,
  created_at;

-- name: UpdateOption :exec
UPDATE options
SET
  name = ?,
  bio = ?,
  duration_minutes = ?,
  weight = ?
WHERE
  id = ? AND user_id = ?;

-- name: UpdateDuration :exec
UPDATE options
SET
  duration_minutes = ?
WHERE
  id = ? AND user_id = ?;

-- name: UpdateWeight :exec
UPDATE options
SET
  weight = ?
WHERE
  id = ? AND user_id = ?;

-- name: DeleteOption :exec
DELETE FROM options
WHERE
  id = ? AND user_id = ?;

-- name: GetOrCreateTag :one
INSERT INTO
  tags (name, user_id)
VALUES
  (LOWER(?), ?) ON CONFLICT (name, user_id) DO
UPDATE
SET
  name = LOWER(excluded.name) RETURNING *;

-- name: GetTagsForOption :many
SELECT
  t.id,
  t.name,
  t.user_id,
  t.created_at
FROM
  tags t
  INNER JOIN option_tags ot ON t.id = ot.tag_id
WHERE
  ot.option_id = ? AND t.user_id = ?
ORDER BY
  ot.created_at;

-- name: AddTagToOption :exec
INSERT
OR IGNORE INTO option_tags (option_id, tag_id)
VALUES
  (?, ?);

-- name: ClearTagsForOption :exec
DELETE FROM option_tags
WHERE
  option_id = ?;

-- name: DeleteUnusedTags :exec
DELETE FROM tags
WHERE
  user_id = ? AND id NOT IN (
    SELECT DISTINCT
      tag_id
    FROM
      option_tags
  );

-- name: GetAllTags :many
SELECT DISTINCT
  t.id,
  t.name,
  t.user_id,
  t.created_at
FROM
  tags t
  INNER JOIN option_tags ot ON t.id = ot.tag_id
WHERE
  t.user_id = ?
ORDER BY
  t.name;

-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES (?, ?)
RETURNING id, email, created_at;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = ?
LIMIT 1;

-- name: UserExists :one
SELECT COUNT(*) > 0 as user_exists FROM users
WHERE email = ?;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = ?
LIMIT 1;

-- Session queries

-- name: InsertSession :exec
INSERT INTO sessions (user_id, token, expires_at, user_agent, ip_address)
VALUES (?, ?, ?, ?, ?);

-- name: GetSessionByToken :one
SELECT * FROM sessions
WHERE token = ?
LIMIT 1;

-- name: UpdateSessionExpiresAt :exec
UPDATE sessions
SET expires_at = ?
WHERE token = ?;

-- name: DeleteSessionByToken :exec
DELETE FROM sessions
WHERE token = ?;

-- name: DeleteOldUserSessions :exec
DELETE FROM sessions
WHERE user_id = ?
AND created_at < datetime('now', '-7 days');
