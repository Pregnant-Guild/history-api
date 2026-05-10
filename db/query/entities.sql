-- name: CreateEntity :one
INSERT INTO entities (
    id, name, slug, description, project_id, status, time_start, time_end
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, uuidv7()), $1, $2, $3, $4, $5, $6, $7
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
    slug = COALESCE(sqlc.narg('slug'), slug),
    description = COALESCE(sqlc.narg('description'), description),
    project_id = COALESCE(sqlc.narg('project_id'), project_id),
    status = COALESCE(sqlc.narg('status'), status),
    time_start = COALESCE(sqlc.narg('time_start'), time_start),
    time_end = COALESCE(sqlc.narg('time_end'), time_end)
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
  AND (sqlc.narg('project_id')::uuid IS NULL OR project_id = sqlc.narg('project_id')::uuid)
  AND (sqlc.narg('name')::text IS NULL OR name ILIKE '%' || sqlc.narg('name')::text || '%')
  AND (
    sqlc.narg('time_point')::int IS NULL OR
    int4range(time_start, time_end, '[]') @> sqlc.narg('time_point')::int
  )
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR id < sqlc.narg('cursor_id')::uuid)
ORDER BY id DESC
LIMIT sqlc.arg('limit_count');

-- name: GetEntitiesByIDs :many
SELECT * FROM entities WHERE id = ANY($1::uuid[]) AND is_deleted = false;

-- name: GetEntitiesByProjectId :many
SELECT *
FROM entities
WHERE project_id = $1 AND is_deleted = false;

-- name: DeleteEntitiesByIDs :exec
UPDATE entities
SET is_deleted = true
WHERE id = ANY($1::uuid[]);

-- name: GetEntityBySlug :one
SELECT *
FROM entities
WHERE slug = $1 AND is_deleted = false;

-- name: GetEntitiesBySlugs :many
SELECT * FROM entities WHERE slug = ANY($1::text[]) AND is_deleted = false;