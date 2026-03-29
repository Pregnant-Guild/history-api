-- name: UpsertUser :one
INSERT INTO users (
    email,
    password_hash,
    google_id,
    auth_provider
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (email) 
DO UPDATE SET
    google_id = EXCLUDED.google_id,
    auth_provider = EXCLUDED.auth_provider
RETURNING *;

-- name: CreateUserProfile :one
INSERT INTO user_profiles (
    user_id,
    display_name,
    avatar_url
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: UpdateUserProfile :one
UPDATE user_profiles
SET
    display_name = $1,
    full_name = $2,
    avatar_url = $3,
    bio = $4,
    location = $5,
    website = $6,
    country_code = $7,
    phone = $8,
    updated_at = now()
WHERE user_id = $9
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users
SET
    password_hash = $2
WHERE id = $1
  AND is_deleted = false;

-- name: UpdateUserRefreshToken :exec
UPDATE users
SET
    refresh_token = $2
WHERE id = $1
  AND is_deleted = false;


-- name: DeleteUser :exec
UPDATE users
SET
    is_deleted = true
WHERE id = $1;

-- name: RestoreUser :exec
UPDATE users
SET
    is_deleted = false
WHERE id = $1;

-- name: GetUserByID :one
SELECT
    u.id,
    u.email,
    u.password_hash,
    u.token_version,
    u.refresh_token,
    u.is_deleted,
    u.created_at,
    u.updated_at,

    -- profile JSON
    (
        SELECT json_build_object(
            'display_name', p.display_name,
            'full_name', p.full_name,
            'avatar_url', p.avatar_url,
            'bio', p.bio,
            'location', p.location,
            'website', p.website,
            'country_code', p.country_code,
            'phone', p.phone
        )
        FROM user_profiles p
        WHERE p.user_id = u.id
    ) AS profile,

    -- roles JSON
    (
        SELECT COALESCE(
            json_agg(json_build_object('id', r.id, 'name', r.name)),
            '[]'
        )::json
        FROM user_roles ur
        JOIN roles r ON ur.role_id = r.id
        WHERE ur.user_id = u.id
    ) AS roles

FROM users u
WHERE u.id = $1 AND u.is_deleted = false;

-- name: GetUserByIDWithoutDeleted :one
SELECT
    u.id,
    u.email,
    u.password_hash,
    u.token_version,
    u.refresh_token,
    u.is_deleted,
    u.created_at,
    u.updated_at,

    -- profile JSON
    (
        SELECT json_build_object(
            'display_name', p.display_name,
            'full_name', p.full_name,
            'avatar_url', p.avatar_url,
            'bio', p.bio,
            'location', p.location,
            'website', p.website,
            'country_code', p.country_code,
            'phone', p.phone
        )
        FROM user_profiles p
        WHERE p.user_id = u.id
    ) AS profile,

    -- roles JSON
    (
        SELECT COALESCE(
            json_agg(json_build_object('id', r.id, 'name', r.name)),
            '[]'
        )::json
        FROM user_roles ur
        JOIN roles r ON ur.role_id = r.id
        WHERE ur.user_id = u.id
    ) AS roles

FROM users u
WHERE u.id = $1;

-- name: GetTokenVersion :one
SELECT token_version
FROM users
WHERE id = $1 AND is_deleted = false;

-- name: UpdateTokenVersion :exec
UPDATE users
SET token_version = $2
WHERE id = $1 AND is_deleted = false;

-- name: GetUserByEmail :one
SELECT
    u.id,
    u.email,
    u.password_hash,
    u.token_version,
    u.is_deleted,
    u.created_at,
    u.updated_at,

    (
        SELECT json_build_object(
            'display_name', p.display_name,
            'full_name', p.full_name,
            'avatar_url', p.avatar_url,
            'bio', p.bio,
            'location', p.location,
            'website', p.website,
            'country_code', p.country_code,
            'phone', p.phone
        )
        FROM user_profiles p
        WHERE p.user_id = u.id
    ) AS profile,

    (
        SELECT COALESCE(
            json_agg(json_build_object('id', r.id, 'name', r.name)),
            '[]'
        )::json
        FROM user_roles ur
        JOIN roles r ON ur.role_id = r.id
        WHERE ur.user_id = u.id
    ) AS roles

FROM users u
WHERE u.email = $1 AND u.is_deleted = false;

-- name: GetUsers :many
SELECT
    u.id,
    u.email,
    u.password_hash,
    u.token_version,
    u.refresh_token,
    u.is_deleted,
    u.created_at,
    u.updated_at,

    -- profile JSON
    (
        SELECT json_build_object(
            'display_name', p.display_name,
            'full_name', p.full_name,
            'avatar_url', p.avatar_url,
            'bio', p.bio,
            'location', p.location,
            'website', p.website,
            'country_code', p.country_code,
            'phone', p.phone
        )
        FROM user_profiles p
        WHERE p.user_id = u.id
    ) AS profile,

    -- roles JSON
    (
        SELECT COALESCE(
            json_agg(json_build_object('id', r.id, 'name', r.name)),
            '[]'
        )::json
        FROM user_roles ur
        JOIN roles r ON ur.role_id = r.id
        WHERE ur.user_id = u.id
    ) AS roles

FROM users u
WHERE 
    (sqlc.narg('cursor')::uuid IS NULL OR u.id > sqlc.narg('cursor')::uuid)
    AND (sqlc.narg('is_deleted')::boolean IS NULL OR u.is_deleted = sqlc.narg('is_deleted')::boolean)
    AND (
        sqlc.narg('role_ids')::uuid[] IS NULL OR 
        EXISTS (
            SELECT 1 FROM user_roles ur2 
            WHERE ur2.user_id = u.id AND ur2.role_id = ANY(sqlc.narg('role_ids')::uuid[])
        )
    )
ORDER BY u.id ASC
LIMIT sqlc.arg('limit');


-- name: SearchUsers :many
SELECT
    u.id,
    u.email,
    u.password_hash,
    u.token_version,
    u.refresh_token,
    u.is_deleted,
    u.created_at,
    u.updated_at,

    (
        SELECT json_build_object(
            'display_name', p.display_name,
            'full_name', p.full_name,
            'avatar_url', p.avatar_url,
            'bio', p.bio,
            'location', p.location,
            'website', p.website,
            'country_code', p.country_code,
            'phone', p.phone
        )
        FROM user_profiles p
        WHERE p.user_id = u.id
    ) AS profile,

    (
        SELECT COALESCE(
            json_agg(json_build_object('id', r.id, 'name', r.name)),
            '[]'
        )::json
        FROM user_roles ur
        JOIN roles r ON ur.role_id = r.id
        WHERE ur.user_id = u.id
    ) AS roles

FROM users u
WHERE 
    (sqlc.narg('cursor')::uuid IS NULL OR u.id > sqlc.narg('cursor')::uuid)
    
    AND (sqlc.narg('is_deleted')::boolean IS NULL OR u.is_deleted = sqlc.narg('is_deleted')::boolean)
    AND (
        sqlc.narg('role_ids')::uuid[] IS NULL OR 
        EXISTS (
            SELECT 1 FROM user_roles ur2 
            WHERE ur2.user_id = u.id AND ur2.role_id = ANY(sqlc.narg('role_ids')::uuid[])
        )
    )
    
    AND (sqlc.narg('search_id')::uuid IS NULL OR u.id = sqlc.narg('search_id')::uuid)
    AND (
        sqlc.narg('search_text')::text IS NULL OR 
        u.email ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
        EXISTS (
            SELECT 1 FROM user_profiles p 
            WHERE p.user_id = u.id AND p.display_name ILIKE '%' || sqlc.narg('search_text')::text || '%'
        )
    )
ORDER BY u.id ASC
LIMIT sqlc.arg('limit');