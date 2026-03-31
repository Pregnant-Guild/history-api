-- name: CreateUserVerification :one
INSERT INTO user_verifications (
    user_id, verify_type
) VALUES (
    $1, $2
)
RETURNING *;

-- name: CreateVerificationMedia :exec
INSERT INTO verification_medias (
    verification_id, media_id
) VALUES (
    $1, $2
);

-- name: GetUserVerificationByID :one
SELECT 
    uv.id, 
    uv.user_id, 
    uv.verify_type, 
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