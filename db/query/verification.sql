-- name: CreateUserVerification :one
INSERT INTO user_verifications (
    user_id, verify_type, content
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetUserVerificationByID :one
SELECT 
    uv.id, 
    uv.user_id, 
    uv.verify_type, 
    uv.content,
    uv.is_deleted, 
    uv.status, 
    uv.reviewed_by, 
    uv.reviewed_at, 
    uv.created_at,
    (
        SELECT COALESCE(
            json_agg(
                json_build_object(
                    'id', m.id,
                    'storage_key', m.storage_key,
                    'original_name', m.original_name,
                    'mime_type', m.mime_type,
                    'size', m.size,
                    'file_metadata', m.file_metadata,
                    'created_at', m.created_at
                )
            ),
            '[]'
        )::json
        FROM verification_medias vm
        JOIN medias m ON vm.media_id = m.id
        WHERE vm.verification_id = uv.id
    ) AS medias
FROM user_verifications uv
WHERE uv.id = $1 AND uv.is_deleted = false;

-- name: GetUserVerifications :many
SELECT 
    uv.id, 
    uv.user_id, 
    uv.verify_type, 
    uv.content,
    uv.is_deleted, 
    uv.status, 
    uv.reviewed_by, 
    uv.reviewed_at, 
    uv.created_at,
    (
        SELECT COALESCE(
            json_agg(
                json_build_object(
                    'id', m.id,
                    'storage_key', m.storage_key,
                    'original_name', m.original_name,
                    'mime_type', m.mime_type,
                    'size', m.size,
                    'file_metadata', m.file_metadata,
                    'created_at', m.created_at
                )
            ),
            '[]'
        )::json
        FROM verification_medias vm
        JOIN medias m ON vm.media_id = m.id
        WHERE vm.verification_id = uv.id
    ) AS medias
FROM user_verifications uv
WHERE uv.user_id = $1 AND uv.is_deleted = false
ORDER BY uv.created_at DESC;

-- name: UpdateUserVerificationStatus :exec
UPDATE user_verifications
SET 
    status = $2,
    reviewed_by = $3,
    reviewed_at = now()
WHERE id = $1 AND is_deleted = false;

-- name: DeleteUserVerification :exec
UPDATE user_verifications
SET is_deleted = true
WHERE id = $1;

-- name: DeleteVerificationMedia :exec
DELETE FROM verification_medias
WHERE verification_id = $1 AND media_id = $2;

-- name: CreateVerificationMedia :exec
INSERT INTO verification_medias (
    verification_id, media_id
)
SELECT $1, unnest($2::uuid[])
ON CONFLICT DO NOTHING;

-- name: BulkDeleteVerificationByMediaId :exec
DELETE FROM verification_medias
WHERE media_id = $1;

-- name: DeleteVerificationMedias :exec
DELETE FROM verification_medias
WHERE verification_id = $1 AND media_id = ANY($2::uuid[]);

-- name: SearchUserVerifications :many
SELECT 
    uv.id, 
    uv.user_id, 
    uv.verify_type, 
    uv.content,
    uv.is_deleted, 
    uv.status, 
    uv.reviewed_by, 
    uv.reviewed_at, 
    uv.created_at,
    (
        SELECT COALESCE(
            json_agg(
                json_build_object(
                    'id', m.id,
                    'storage_key', m.storage_key,
                    'original_name', m.original_name,
                    'mime_type', m.mime_type,
                    'size', m.size,
                    'file_metadata', m.file_metadata,
                    'created_at', m.created_at
                )
            ),
            '[]'
        )::json
        FROM verification_medias vm
        JOIN medias m ON vm.media_id = m.id
        WHERE vm.verification_id = uv.id
    ) AS medias
FROM user_verifications uv
WHERE 
    uv.is_deleted = false
    AND (sqlc.narg('user_ids')::uuid[] IS NULL OR uv.user_id = ANY(sqlc.narg('user_ids')::uuid[]))
    AND (sqlc.narg('verify_types')::text[] IS NULL OR uv.verify_type = ANY(sqlc.narg('verify_types')::text[]))
    AND (sqlc.narg('statuses')::text[] IS NULL OR uv.status = ANY(sqlc.narg('statuses')::text[]))
    AND (sqlc.narg('reviewed_by')::uuid IS NULL OR uv.reviewed_by = sqlc.narg('reviewed_by')::uuid)
    AND (sqlc.narg('created_after')::timestamptz IS NULL OR uv.created_at >= sqlc.narg('created_after')::timestamptz)
    AND (sqlc.narg('created_before')::timestamptz IS NULL OR uv.created_at <= sqlc.narg('created_before')::timestamptz)
    AND (
        sqlc.narg('search_text')::text IS NULL OR 
        uv.id::text ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
        uv.content::text ILIKE '%' || sqlc.narg('search_text')::text || '%'
    )
ORDER BY
    CASE WHEN sqlc.narg('sort') = 'created_at' AND sqlc.narg('order') = 'asc' THEN uv.created_at END ASC,
    CASE WHEN sqlc.narg('sort') = 'created_at' AND sqlc.narg('order') = 'desc' THEN uv.created_at END DESC,
    CASE WHEN sqlc.narg('sort') = 'reviewed_at' AND sqlc.narg('order') = 'asc' THEN uv.reviewed_at END ASC,
    CASE WHEN sqlc.narg('sort') = 'reviewed_at' AND sqlc.narg('order') = 'desc' THEN uv.reviewed_at END DESC,
    CASE WHEN sqlc.narg('sort') = 'status' AND sqlc.narg('order') = 'asc' THEN uv.status END ASC,
    CASE WHEN sqlc.narg('sort') = 'status' AND sqlc.narg('order') = 'desc' THEN uv.status END DESC,
    CASE WHEN sqlc.narg('sort') IS NULL THEN uv.created_at END DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CountUserVerifications :one
SELECT count(*) 
FROM user_verifications uv
WHERE 
    uv.is_deleted = false
    AND (sqlc.narg('user_ids')::uuid[] IS NULL OR uv.user_id = ANY(sqlc.narg('user_ids')::uuid[]))
    AND (sqlc.narg('verify_types')::text[] IS NULL OR uv.verify_type = ANY(sqlc.narg('verify_types')::text[]))
    AND (sqlc.narg('statuses')::text[] IS NULL OR uv.status = ANY(sqlc.narg('statuses')::text[]))
    AND (sqlc.narg('reviewed_by')::uuid IS NULL OR uv.reviewed_by = sqlc.narg('reviewed_by')::uuid)
    AND (sqlc.narg('created_after')::timestamptz IS NULL OR uv.created_at >= sqlc.narg('created_after')::timestamptz)
    AND (sqlc.narg('created_before')::timestamptz IS NULL OR uv.created_at <= sqlc.narg('created_before')::timestamptz)
    AND (
        sqlc.narg('search_text')::text IS NULL OR 
        uv.id::text ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
        uv.content::text ILIKE '%' || sqlc.narg('search_text')::text || '%'
    );