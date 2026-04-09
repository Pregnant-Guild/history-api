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

-- name: DeleteMedias :exec
DELETE FROM medias
WHERE id = ANY($1::uuid[]);

-- name: SearchMedias :many
SELECT 
    id, user_id, storage_key, original_name, mime_type, size, file_metadata, created_at, updated_at
FROM medias
WHERE 
    (sqlc.narg('user_ids')::uuid[] IS NULL OR user_id = ANY(sqlc.narg('user_ids')::uuid[]))
    AND (sqlc.narg('mime_type')::text IS NULL OR mime_type ILIKE sqlc.narg('mime_type')::text || '%')
    AND (sqlc.narg('min_size')::bigint IS NULL OR size >= sqlc.narg('min_size')::bigint)
    AND (sqlc.narg('max_size')::bigint IS NULL OR size <= sqlc.narg('max_size')::bigint)
    AND (
        sqlc.narg('search_text')::text IS NULL OR 
        id::text ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
        original_name ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
        storage_key ILIKE '%' || sqlc.narg('search_text')::text || '%'
    )
ORDER BY
    CASE WHEN sqlc.narg('sort') = 'id' AND sqlc.narg('order') = 'asc' THEN id END ASC,
    CASE WHEN sqlc.narg('sort') = 'id' AND sqlc.narg('order') = 'desc' THEN id END DESC,

    CASE WHEN sqlc.narg('sort') = 'created_at' AND sqlc.narg('order') = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.narg('sort') = 'created_at' AND sqlc.narg('order') = 'desc' THEN created_at END DESC,

    CASE WHEN sqlc.narg('sort') = 'updated_at' AND sqlc.narg('order') = 'asc' THEN updated_at END ASC,
    CASE WHEN sqlc.narg('sort') = 'updated_at' AND sqlc.narg('order') = 'desc' THEN updated_at END DESC,

    CASE WHEN sqlc.narg('sort') = 'size' AND sqlc.narg('order') = 'asc' THEN size END ASC,
    CASE WHEN sqlc.narg('sort') = 'size' AND sqlc.narg('order') = 'desc' THEN size END DESC,

    CASE WHEN sqlc.narg('sort') = 'original_name' AND sqlc.narg('order') = 'asc' THEN original_name END ASC,
    CASE WHEN sqlc.narg('sort') = 'original_name' AND sqlc.narg('order') = 'desc' THEN original_name END DESC,
 
    CASE WHEN sqlc.narg('sort') = 'storage_key' AND sqlc.narg('order') = 'asc' THEN storage_key END ASC,
    CASE WHEN sqlc.narg('sort') = 'storage_key' AND sqlc.narg('order') = 'desc' THEN storage_key END DESC,

    CASE WHEN sqlc.narg('sort') = 'mime_type' AND sqlc.narg('order') = 'asc' THEN mime_type END ASC,
    CASE WHEN sqlc.narg('sort') = 'mime_type' AND sqlc.narg('order') = 'desc' THEN mime_type END DESC,
    
    id ASC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');


-- name: CountMedias :one
SELECT count(*) 
FROM medias
WHERE 
    (sqlc.narg('user_ids')::uuid[] IS NULL OR user_id = ANY(sqlc.narg('user_ids')::uuid[]))
    AND (sqlc.narg('mime_type')::text IS NULL OR mime_type ILIKE sqlc.narg('mime_type')::text || '%')
    AND (sqlc.narg('min_size')::bigint IS NULL OR size >= sqlc.narg('min_size')::bigint)
    AND (sqlc.narg('max_size')::bigint IS NULL OR size <= sqlc.narg('max_size')::bigint)
    AND (
        sqlc.narg('search_text')::text IS NULL OR 
        id::text ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
        original_name ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
        storage_key ILIKE '%' || sqlc.narg('search_text')::text || '%'
    );

-- name: GetMediasByUserID :many
SELECT * FROM medias
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetMediaByID :one
SELECT * FROM medias
WHERE id = $1;