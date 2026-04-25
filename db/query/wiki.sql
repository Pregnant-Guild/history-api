-- name: CreateWiki :one
INSERT INTO wikis (
    title, content
) VALUES (
    $1, $2
)
RETURNING *;

-- name: GetWikiById :one
SELECT *
FROM wikis
WHERE id = $1 AND is_deleted = false;

-- name: UpdateWiki :one
UPDATE wikis
SET 
    title = COALESCE(sqlc.narg('title'), title),
    content = COALESCE(sqlc.narg('content'), content)
WHERE id = sqlc.arg('id') AND is_deleted = false
RETURNING *;

-- name: DeleteWiki :exec
UPDATE wikis
SET 
    is_deleted = true
WHERE id = $1;

-- name: SearchWikis :many
SELECT w.*
FROM wikis w
WHERE w.is_deleted = false
  AND w.title ILIKE '%' || sqlc.arg('title')::text || '%'
  AND (
    sqlc.narg('entity_id')::uuid IS NULL OR
    EXISTS (
        SELECT 1 
        FROM entity_wikis ew 
        WHERE ew.wiki_id = w.id 
          AND ew.entity_id = sqlc.narg('entity_id')::uuid
    )
  )
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR w.id < sqlc.narg('cursor_id')::uuid)

ORDER BY w.id DESC
LIMIT sqlc.arg('limit_count');

-- name: BulkDeleteEntityWikisByEntityId :many
DELETE FROM entity_wikis
WHERE entity_id = $1
RETURNING wiki_id;

-- name: CreateEntityWikis :exec
INSERT INTO entity_wikis (
    entity_id, wiki_id
)
SELECT $1, unnest(@wiki_ids::uuid[]);


-- name: GetWikisByIDs :many
SELECT * FROM wikis WHERE id = ANY($1::uuid[]) AND is_deleted = false;
