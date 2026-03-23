-- name: CreateUser :one
INSERT INTO users (
    name,
    email,
    password_hash,
    avatar_url
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET 
    name = $1, 
    avatar_url = $2, 
    is_active = $3,
    is_verified = $4,
    updated_at = now()
WHERE users.id = $5 AND users.is_deleted = false 
RETURNING 
    users.id, 
    users.name, 
    users.email, 
    users.password_hash, 
    users.avatar_url, 
    users.is_active, 
    users.is_verified, 
    users.token_version, 
    users.refresh_token,
    users.is_deleted, 
    users.created_at, 
    users.updated_at,
    (
        SELECT COALESCE(json_agg(json_build_object('id', roles.id, 'name', roles.name)), '[]')::json
        FROM user_roles
        JOIN roles ON user_roles.role_id = roles.id
        WHERE user_roles.user_id = users.id
    ) AS roles;

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


-- name: VerifyUser :exec
UPDATE users
SET
    is_verified = true
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

-- name: ExistsUserByEmail :one
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE email = $1
      AND is_deleted = false
);

-- name: GetUsers :many
SELECT
    u.id, 
    u.name, 
    u.email, 
    u.password_hash, 
    u.avatar_url,
    u.is_active, 
    u.is_verified, 
    u.token_version, 
    u.refresh_token, 
    u.is_deleted,
    u.created_at, 
    u.updated_at,
    COALESCE(
        json_agg(
            json_build_object('id', r.id, 'name', r.name)
        ) FILTER (WHERE r.id IS NOT NULL),
        '[]'
    )::json AS roles
FROM users u
LEFT JOIN user_roles ur ON u.id = ur.user_id
LEFT JOIN roles r ON ur.role_id = r.id
WHERE u.is_deleted = false
GROUP BY u.id;

-- name: GetUserByID :one
SELECT
    u.id, 
    u.name, 
    u.email, 
    u.password_hash, 
    u.avatar_url,
    u.is_active, 
    u.is_verified, 
    u.token_version, 
    u.refresh_token,
    u.is_deleted,
    u.created_at, 
    u.updated_at,
    COALESCE(
        json_agg(
            json_build_object('id', r.id, 'name', r.name)
        ) FILTER (WHERE r.id IS NOT NULL),
        '[]'
    )::json AS roles
FROM users u
LEFT JOIN user_roles ur ON u.id = ur.user_id
LEFT JOIN roles r ON ur.role_id = r.id
WHERE u.id = $1 AND u.is_deleted = false
GROUP BY u.id;

-- name: GetUserByEmail :one
SELECT
    u.id, 
    u.name, 
    u.email, 
    u.password_hash, 
    u.avatar_url,
    u.is_active, 
    u.is_verified, 
    u.token_version, 
    u.is_deleted,
    u.created_at, 
    u.updated_at,
    COALESCE(
        json_agg(
            json_build_object('id', r.id, 'name', r.name)
        ) FILTER (WHERE r.id IS NOT NULL),
        '[]'
    )::json AS roles
FROM users u
LEFT JOIN user_roles ur ON u.id = ur.user_id
LEFT JOIN roles r ON ur.role_id = r.id
WHERE u.email = $1 AND u.is_deleted = false
GROUP BY u.id;
