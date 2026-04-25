-- name: CreateProject :one
INSERT INTO projects (
    title, description, project_status, user_id
) VALUES (
    $1, $2, $3, $4
)
RETURNING 
    *,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    '{}'::uuid[] AS commit_ids,
    '{}'::uuid[] AS submission_ids;

-- name: GetProjectById :one
SELECT 
    p.*,
    COALESCE(
        (SELECT array_agg(id) FROM revisions WHERE project_id = p.id), 
        '{}'
    )::uuid[] AS commit_ids,
    COALESCE(
        (SELECT array_agg(id) FROM submissions WHERE project_id = p.id), 
        '{}'
    )::uuid[] AS submission_ids,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user
FROM projects p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE p.id = $1 AND p.is_deleted = false;

-- name: UpdateProject :one
UPDATE projects
SET 
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    latest_revision_id = COALESCE(sqlc.narg('latest_revision_id'), latest_revision_id),
    version_count = COALESCE(sqlc.narg('version_count'), version_count),
    project_status = COALESCE(sqlc.narg('status'), status),
    locked_by = COALESCE(sqlc.narg('locked_by'), locked_by),
    updated_at = NOW()
FROM projects p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE p.id = sqlc.arg('id') AND p.is_deleted = false
RETURNING 
    p.*,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    COALESCE((SELECT array_agg(id) FROM revisions WHERE project_id = projects.id), '{}')::uuid[] AS commit_ids,
    COALESCE((SELECT array_agg(id) FROM submissions WHERE project_id = projects.id), '{}')::uuid[] AS submission_ids;

-- name: DeleteProject :exec
UPDATE projects
SET 
    is_deleted = true
WHERE id = $1;

-- name: SearchProjects :many
SELECT 
    p.*,
    COALESCE(
        (SELECT array_agg(id) FROM revisions WHERE project_id = p.id), 
        '{}'
    )::uuid[] AS commit_ids,
    COALESCE(
        (SELECT array_agg(id) FROM submissions WHERE project_id = p.id), 
        '{}'
    )::uuid[] AS submission_ids,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user
FROM projects p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE p.is_deleted = false
  AND (
      sqlc.narg('statuses')::text[] IS NULL 
      OR p.project_status = ANY(sqlc.narg('statuses')::text[])
  )
  AND (sqlc.narg('user_ids')::uuid[] IS NULL OR p.user_id = ANY(sqlc.narg('user_ids')::uuid[]))
  AND (
      sqlc.narg('search_text')::text IS NULL OR 
      p.title ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
      p.description ILIKE '%' || sqlc.narg('search_text')::text || '%'
  )
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR p.created_at >= sqlc.narg('created_from')::timestamptz)
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR p.created_at <= sqlc.narg('created_to')::timestamptz)
ORDER BY
    CASE WHEN sqlc.narg('sort') = 'created_at' AND sqlc.narg('order') = 'asc' THEN p.created_at END ASC,
    CASE WHEN sqlc.narg('sort') = 'created_at' AND sqlc.narg('order') = 'desc' THEN p.created_at END DESC,
    CASE WHEN sqlc.narg('sort') = 'updated_at' AND sqlc.narg('order') = 'asc' THEN p.updated_at END ASC,
    CASE WHEN sqlc.narg('sort') = 'updated_at' AND sqlc.narg('order') = 'desc' THEN p.updated_at END DESC,
    CASE WHEN sqlc.narg('sort') = 'title' AND sqlc.narg('order') = 'asc' THEN p.title END ASC,
    CASE WHEN sqlc.narg('sort') = 'title' AND sqlc.narg('order') = 'desc' THEN p.title END DESC,
    CASE WHEN sqlc.narg('sort') IS NULL THEN p.updated_at END DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CountProjects :one
SELECT count(*)
FROM projects p
WHERE p.is_deleted = false
  AND (
      sqlc.narg('statuses')::text[] IS NULL 
      OR p.project_status = ANY(sqlc.narg('statuses')::text[])
  )
  AND (sqlc.narg('user_ids')::uuid[] IS NULL OR p.user_id = ANY(sqlc.narg('user_ids')::uuid[]))
  AND (
      sqlc.narg('search_text')::text IS NULL OR 
      p.title ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
      p.description ILIKE '%' || sqlc.narg('search_text')::text || '%'
  )
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR p.created_at >= sqlc.narg('created_from')::timestamptz)
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR p.created_at <= sqlc.narg('created_to')::timestamptz);

-- name: GetProjectsByUserId :many
SELECT 
    p.*,
    COALESCE(
        (SELECT array_agg(id) FROM revisions WHERE project_id = p.id), 
        '{}'
    )::uuid[] AS commit_ids,
    COALESCE(
        (SELECT array_agg(id) FROM submissions WHERE project_id = p.id), 
        '{}'
    )::uuid[] AS submission_ids,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user
FROM projects p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE p.user_id = $1 
  AND p.is_deleted = false 
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR p.id < sqlc.narg('cursor_id')::uuid)
ORDER BY p.updated_at DESC
LIMIT sqlc.arg('limit');


-- name: GetProjectsByIDs :many
SELECT 
    p.*,
    COALESCE((SELECT array_agg(id) FROM revisions WHERE project_id = p.id), '{}')::uuid[] AS commit_ids,
    COALESCE((SELECT array_agg(id) FROM submissions WHERE project_id = p.id), '{}')::uuid[] AS submission_ids,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user
FROM projects p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE p.id = ANY($1::uuid[]) AND p.is_deleted = false;
