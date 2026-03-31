-- name: CreateMedia :one
INSERT INTO medias (
    user_id, storage_key, original_name, mime_type, size, file_metadata
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: DeleteMedia :exec
DELETE FROM medias
WHERE id = $1;

-- name: SearchMedias :many
SELECT *
FROM medias
WHERE 
    (sqlc.narg('cursor')::uuid IS NULL OR id > sqlc.narg('cursor')::uuid)
    AND (
        sqlc.narg('search_text')::text IS NULL OR 
        original_name ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
        storage_key ILIKE '%' || sqlc.narg('search_text')::text || '%'
    )
ORDER BY id ASC
LIMIT sqlc.arg('limit');

-- name: GetMediasByUserID :many
SELECT * FROM medias
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetMediaByID :one
SELECT * FROM medias
WHERE id = $1;