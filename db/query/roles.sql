-- name: CreateRole :one
INSERT INTO roles (name)
VALUES ($1)
RETURNING *;

-- name: GetRoleByName :one
SELECT id, name, is_deleted, created_at, updated_at FROM roles
WHERE name = $1 AND is_deleted = false;

-- name: GetRoleByID :one
SELECT id, name, is_deleted, created_at, updated_at  FROM roles
WHERE id = $1 AND is_deleted = false;

-- name: AddUserRole :exec
INSERT INTO user_roles (user_id, role_id)
SELECT $1, r.id
FROM roles r
WHERE r.name = $2
ON CONFLICT DO NOTHING;

-- name: RemoveUserRole :exec
DELETE FROM user_roles ur
USING roles r
WHERE ur.role_id = r.id
  AND ur.user_id = $1
  AND r.name = $2;

-- name: RemoveAllRolesFromUser :exec
DELETE FROM user_roles 
WHERE user_id = $1;

-- name: RemoveAllUsersFromRole :exec
DELETE FROM user_roles 
WHERE role_id = $1;

-- name: GetRoles :many
SELECT *
FROM roles
WHERE is_deleted = false;

-- name: UpdateRole :one
UPDATE roles 
SET 
  name = $1,
  updated_at = now()
WHERE id = $2 AND is_deleted = false
RETURNING *;

-- name: DeleteRole :exec
UPDATE roles
SET
    is_deleted = true,
    updated_at = now()
WHERE id = $1;

-- name: RestoreRole :exec
UPDATE roles
SET
    is_deleted = false,
    updated_at = now()
WHERE id = $1;