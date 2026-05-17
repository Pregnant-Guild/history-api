-- name: CreateBattleReplay :one
INSERT INTO battle_replays (
    id, geometry_id, project_id, target_geometry_ids, detail
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, uuidv7()), $1, $2, $3, $4
)
RETURNING *;

-- name: GetBattleReplayById :one
SELECT *
FROM battle_replays
WHERE id = $1 AND is_deleted = false;

-- name: GetBattleReplaysByIDs :many
SELECT * FROM battle_replays WHERE id = ANY($1::uuid[]) AND is_deleted = false;

-- name: GetBattleReplaysByGeometryId :many
SELECT *
FROM battle_replays
WHERE geometry_id = $1 AND is_deleted = false;

-- name: GetBattleReplaysByGeometryIDs :many
SELECT *
FROM battle_replays
WHERE geometry_id = ANY($1::uuid[]) AND is_deleted = false;

-- name: GetBattleReplaysByProjectId :many
SELECT *
FROM battle_replays
WHERE project_id = $1 AND is_deleted = false;

-- name: UpdateBattleReplay :one
UPDATE battle_replays
SET
    geometry_id = COALESCE(sqlc.narg('geometry_id'), geometry_id),
    target_geometry_ids = COALESCE(sqlc.narg('target_geometry_ids'), target_geometry_ids),
    detail = COALESCE(sqlc.narg('detail'), detail)
WHERE id = sqlc.arg('id') AND is_deleted = false
RETURNING *;

-- name: DeleteBattleReplay :exec
UPDATE battle_replays
SET is_deleted = true
WHERE id = $1;

-- name: DeleteBattleReplaysByIDs :exec
UPDATE battle_replays
SET is_deleted = true
WHERE id = ANY($1::uuid[]);
