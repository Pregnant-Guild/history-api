-- name: CreateEntity :one
INSERT INTO entities (
    name, description, thumbnail_url
) VALUES (
    $1, $2, $3
)
RETURNING *;


-- name: GetEntityById :one
SELECT *
FROM entities
WHERE id = $1 AND is_deleted = false;


-- name: UpdateEntity :one
UPDATE entities
SET 
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    thumbnail_url = COALESCE(sqlc.narg('thumbnail_url'), thumbnail_url)
WHERE id = sqlc.arg('id') AND is_deleted = false
RETURNING *;


-- name: DeleteEntity :exec
UPDATE entities
SET 
    is_deleted = true
WHERE id = $1;


-- name: SearchEntities :many
SELECT *
FROM entities
WHERE is_deleted = false
  AND name ILIKE '%' || sqlc.arg('name')::text || '%'
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR id < sqlc.narg('cursor_id')::uuid)
ORDER BY id DESC
LIMIT sqlc.arg('limit_count');

-- name: GetEntitiesByIDs :many
SELECT * FROM entities WHERE id = ANY($1::uuid[]) AND is_deleted = false;
