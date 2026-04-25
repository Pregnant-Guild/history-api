-- name: CreateRevision :one
INSERT INTO revisions (
    project_id, version_no, snapshot_json, snapshot_hash, parent_id, user_id, edit_summary
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetRevisionById :one
SELECT *
FROM revisions
WHERE id = $1 AND is_deleted = false;

-- name: DeleteRevision :exec
UPDATE revisions
SET is_deleted = true
WHERE id = $1;

-- name: SearchRevisions :many
SELECT *
FROM revisions
WHERE is_deleted = false
  AND (sqlc.narg('project_id')::uuid IS NULL OR project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('user_id')::uuid IS NULL OR user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR id < sqlc.narg('cursor_id')::uuid)
ORDER BY version_no DESC
LIMIT sqlc.arg('limit');

-- name: GetRevisionsByIDs :many
SELECT * FROM revisions WHERE id = ANY($1::uuid[]) AND is_deleted = false;
