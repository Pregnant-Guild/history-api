-- name: CreateCommit :one
INSERT INTO commits (
    project_id, snapshot_json, snapshot_hash, user_id, edit_summary
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetCommitById :one
SELECT *
FROM commits
WHERE id = $1 AND is_deleted = false;

-- name: GetCommitsByProjectID :many
SELECT *
FROM commits
WHERE project_id = $1 AND is_deleted = false
ORDER BY created_at DESC;

-- name: DeleteCommit :exec
UPDATE commits
SET is_deleted = true
WHERE id = $1;

-- name: SearchCommits :many
SELECT *
FROM commits
WHERE is_deleted = false
  AND (sqlc.narg('project_id')::uuid IS NULL OR project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('user_id')::uuid IS NULL OR user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR id < sqlc.narg('cursor_id')::uuid)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit');

-- name: GetCommitsByIDs :many
SELECT * FROM commits WHERE id = ANY($1::uuid[]) AND is_deleted = false;
