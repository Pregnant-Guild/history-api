-- name: CreateUserVerification :one
WITH inserted_uv AS (
    INSERT INTO user_verifications (
        user_id, verify_type, content
    ) VALUES (
        $1, $2, $3
    )
    RETURNING *
)
SELECT 
    i.id, 
    i.verify_type, 
    i.content,
    i.is_deleted, 
    i.status, 
    i.review_note,
    i.reviewed_at, 
    i.created_at,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    NULL::json AS reviewer, -- Khi mới tạo thì reviewer luôn null
    '[]'::json AS medias
FROM inserted_uv i
JOIN users u ON i.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id;


-- name: GetUserVerificationByID :one
SELECT 
    uv.id, 
    uv.verify_type, 
    uv.content,
    uv.is_deleted, 
    uv.status, 
    uv.review_note,
    uv.reviewed_at, 
    uv.created_at,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    CASE WHEN uv.reviewed_by IS NOT NULL THEN
        json_build_object(
            'id', ru.id,
            'email', ru.email,
            'display_name', rup.display_name,
            'full_name', rup.full_name,
            'avatar_url', rup.avatar_url
        )::json
    ELSE NULL::json END AS reviewer,
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
JOIN users u ON uv.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON uv.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id
WHERE uv.id = $1 AND uv.is_deleted = false;


-- name: GetUserVerifications :many
SELECT 
    uv.id, 
    uv.verify_type, 
    uv.content,
    uv.is_deleted, 
    uv.status, 
    uv.review_note,
    uv.reviewed_at, 
    uv.created_at,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    CASE WHEN uv.reviewed_by IS NOT NULL THEN
        json_build_object(
            'id', ru.id,
            'email', ru.email,
            'display_name', rup.display_name,
            'full_name', rup.full_name,
            'avatar_url', rup.avatar_url
        )::json
    ELSE NULL::json END AS reviewer,
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
JOIN users u ON uv.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON uv.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id
WHERE uv.user_id = $1 AND uv.is_deleted = false
ORDER BY uv.created_at DESC;


-- name: UpdateUserVerificationStatus :exec
UPDATE user_verifications
SET 
    status = $2,
    review_note = $3,
    reviewed_by = $4,
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


-- name: BulkDeleteVerificationMediaByMediaId :many
DELETE FROM verification_medias
WHERE media_id = $1
RETURNING verification_id;


-- name: SearchUserVerifications :many
SELECT 
    uv.id, 
    uv.verify_type, 
    uv.content,
    uv.is_deleted, 
    uv.status, 
    uv.review_note, 
    uv.reviewed_at, 
    uv.created_at,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    CASE WHEN uv.reviewed_by IS NOT NULL THEN
        json_build_object(
            'id', ru.id,
            'email', ru.email,
            'display_name', rup.display_name,
            'full_name', rup.full_name,
            'avatar_url', rup.avatar_url
        )::json
    ELSE NULL::json END AS reviewer,
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
JOIN users u ON uv.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON uv.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id
WHERE 
    uv.is_deleted = false
    AND (sqlc.narg('user_ids')::uuid[] IS NULL OR uv.user_id = ANY(sqlc.narg('user_ids')::uuid[]))
    AND (
        sqlc.narg('verify_types')::smallint[] IS NULL 
        OR uv.verify_type = ANY(sqlc.narg('verify_types')::smallint[])
    )
    AND (
        sqlc.narg('statuses')::smallint[] IS NULL 
        OR uv.status = ANY(sqlc.narg('statuses')::smallint[])
    )
    AND (sqlc.narg('reviewed_by')::uuid IS NULL OR uv.reviewed_by = sqlc.narg('reviewed_by')::uuid)
    AND (sqlc.narg('created_from')::timestamptz IS NULL OR uv.created_at >= sqlc.narg('created_from')::timestamptz)
    AND (sqlc.narg('created_to')::timestamptz IS NULL OR uv.created_at <= sqlc.narg('created_to')::timestamptz)
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
    AND (
        sqlc.narg('verify_types')::smallint[] IS NULL 
        OR uv.verify_type = ANY(sqlc.narg('verify_types')::smallint[])
    )
    AND (
        sqlc.narg('statuses')::smallint[] IS NULL 
        OR uv.status = ANY(sqlc.narg('statuses')::smallint[])
    )
    AND (sqlc.narg('reviewed_by')::uuid IS NULL OR uv.reviewed_by = sqlc.narg('reviewed_by')::uuid)
    AND (sqlc.narg('created_from')::timestamptz IS NULL OR uv.created_at >= sqlc.narg('created_from')::timestamptz)
    AND (sqlc.narg('created_to')::timestamptz IS NULL OR uv.created_at <= sqlc.narg('created_to')::timestamptz)
    AND (
        sqlc.narg('search_text')::text IS NULL OR 
        uv.id::text ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
        uv.content::text ILIKE '%' || sqlc.narg('search_text')::text || '%'
    );