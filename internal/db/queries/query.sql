-- name: GetAuthor :one
SELECT * FROM authors
WHERE id = ? LIMIT 1;

-- name: ListAuthors :many
SELECT * FROM authors
ORDER BY name;

-- name: CreateAuthor :one
INSERT INTO authors (
  name, bio
) VALUES (
  ?, ?
)
RETURNING *;

-- name: UpdateAuthor :exec
UPDATE authors
SET name = ?,
bio = ?
WHERE id = ?;

-- name: DeleteAuthor :exec
DELETE FROM authors
WHERE id = ?;

-- name: GetOptions :many
SELECT * FROM options
ORDER BY created_at;

-- name: GetOption :one
SELECT * FROM options
WHERE id = ? LIMIT 1;

-- name: CreateOption :one
INSERT INTO options (
  name, bio, duration_minutes, weight
) VALUES (
  ?, ?, ?, ?
)
RETURNING id, name, bio, duration_minutes, weight, created_at;

-- name: UpdateOption :exec
UPDATE options
SET name = ?,
    bio = ?,
    duration_minutes = ?,
    weight = ?
WHERE id = ?;

-- name: UpdateDuration :exec
UPDATE options
SET duration_minutes = ?
WHERE id = ?;

-- name: UpdateWeight :exec
UPDATE options
SET weight = ?
WHERE id = ?;

-- name: DeleteOption :exec
DELETE FROM options
WHERE id = ?;

-- name: GetOrCreateTag :one
INSERT INTO tags (name) 
VALUES (LOWER(?))
ON CONFLICT(name) DO UPDATE SET name = LOWER(excluded.name)
RETURNING *;

-- name: GetTagsForOption :many
SELECT t.id, t.name, t.created_at FROM tags t
INNER JOIN option_tags ot ON t.id = ot.tag_id
WHERE ot.option_id = ?
ORDER BY ot.created_at;

-- name: AddTagToOption :exec
INSERT OR IGNORE INTO option_tags (option_id, tag_id) 
VALUES (?, ?);

-- name: ClearTagsForOption :exec
DELETE FROM option_tags WHERE option_id = ?;

-- name: DeleteUnusedTags :exec
DELETE FROM tags 
WHERE id NOT IN (SELECT DISTINCT tag_id FROM option_tags);

-- name: GetAllTags :many
SELECT DISTINCT t.id, t.name, t.created_at 
FROM tags t
INNER JOIN option_tags ot ON t.id = ot.tag_id
ORDER BY t.name;
