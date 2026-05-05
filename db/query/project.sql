-- name: CreateProject :one
WITH inserted_project AS (
    INSERT INTO projects (
        title, description, project_status, user_id
    ) VALUES (
        $1, $2, $3, $4
    )
    RETURNING *
)
SELECT 
    p.*,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    '[]'::json AS commits,
    '[]'::json AS submissions,
    '[]'::json AS members
FROM inserted_project p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id;

-- name: GetProjectById :one
SELECT 
    p.*,
    COALESCE(
        (SELECT json_agg(json_build_object('id', c.id, 'edit_summary', c.edit_summary) ORDER BY c.created_at DESC) 
         FROM commits c WHERE c.project_id = p.id AND c.is_deleted = false), 
        '[]'
	    )::json AS commits,
	    COALESCE(
	        (SELECT json_agg(json_build_object('id', s.id, 'status', s.status)) FROM submissions s WHERE s.project_id = p.id AND s.is_deleted = false),
	        '[]'
	    )::json AS submissions,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'user_id', pm.user_id, 'role', pm.role,
            'display_name', mup.display_name, 'avatar_url', mup.avatar_url
         ) ORDER BY pm.role ASC, pm.created_at ASC)
         FROM project_members pm
         JOIN users mu ON pm.user_id = mu.id
         LEFT JOIN user_profiles mup ON mu.id = mup.user_id
         WHERE pm.project_id = p.id),
        '[]'
    )::json AS members
FROM projects p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE p.id = $1 AND p.is_deleted = false;

-- name: UpdateProject :one
UPDATE projects
SET 
    title = COALESCE(sqlc.narg('title'), projects.title),
    description = COALESCE(sqlc.narg('description'), projects.description),
    latest_commit_id = COALESCE(sqlc.narg('latest_commit_id'), projects.latest_commit_id),
    project_status = COALESCE(sqlc.narg('status'), projects.project_status),
    locked_by = COALESCE(sqlc.narg('locked_by'), projects.locked_by),
    updated_at = NOW()
FROM users u
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE projects.id = sqlc.arg('id') AND projects.is_deleted = false
  AND projects.user_id = u.id
RETURNING 
    projects.*,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    COALESCE(
        (SELECT json_agg(json_build_object('id', c.id, 'edit_summary', c.edit_summary) ORDER BY c.created_at DESC)
         FROM commits c WHERE c.project_id = projects.id AND c.is_deleted = false),
        '[]'
	    )::json AS commits,
	    COALESCE(
	        (SELECT json_agg(json_build_object('id', s.id, 'status', s.status)) FROM submissions s WHERE s.project_id = projects.id AND s.is_deleted = false),
	        '[]'
	    )::json AS submissions,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'user_id', pm.user_id, 'role', pm.role,
            'display_name', mup.display_name, 'avatar_url', mup.avatar_url
         ) ORDER BY pm.role ASC, pm.created_at ASC)
         FROM project_members pm
         JOIN users mu ON pm.user_id = mu.id
         LEFT JOIN user_profiles mup ON mu.id = mup.user_id
         WHERE pm.project_id = projects.id),
        '[]'
    )::json AS members;

-- name: DeleteProject :exec
UPDATE projects
SET 
    is_deleted = true
WHERE id = $1;

-- name: SearchProjects :many
SELECT 
    p.*,
    COALESCE(
        (SELECT json_agg(json_build_object('id', c.id, 'edit_summary', c.edit_summary) ORDER BY c.created_at DESC)
         FROM commits c WHERE c.project_id = p.id AND c.is_deleted = false),
        '[]'
	    )::json AS commits,
	    COALESCE(
	        (SELECT json_agg(json_build_object('id', s.id, 'status', s.status)) FROM submissions s WHERE s.project_id = p.id AND s.is_deleted = false),
	        '[]'
	    )::json AS submissions,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'user_id', pm.user_id, 'role', pm.role,
            'display_name', mup.display_name, 'avatar_url', mup.avatar_url
         ) ORDER BY pm.role ASC, pm.created_at ASC)
         FROM project_members pm
         JOIN users mu ON pm.user_id = mu.id
         LEFT JOIN user_profiles mup ON mu.id = mup.user_id
         WHERE pm.project_id = p.id),
        '[]'
    )::json AS members
FROM projects p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE p.is_deleted = false
  AND (
      sqlc.narg('statuses')::smallint[] IS NULL 
      OR p.project_status = ANY(sqlc.narg('statuses')::smallint[])
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
      sqlc.narg('statuses')::smallint[] IS NULL 
      OR p.project_status = ANY(sqlc.narg('statuses')::smallint[])
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
        (SELECT json_agg(json_build_object('id', c.id, 'edit_summary', c.edit_summary) ORDER BY c.created_at DESC)
         FROM commits c WHERE c.project_id = p.id AND c.is_deleted = false),
        '[]'
    )::json AS commits,
    COALESCE(
        (SELECT json_agg(json_build_object('id', s.id, 'status', s.status)) FROM submissions s WHERE s.project_id = p.id AND s.is_deleted = false),
        '[]'
    )::json AS submissions,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'user_id', pm.user_id, 'role', pm.role,
            'display_name', mup.display_name, 'avatar_url', mup.avatar_url
         ) ORDER BY pm.role ASC, pm.created_at ASC)
         FROM project_members pm
         JOIN users mu ON pm.user_id = mu.id
         LEFT JOIN user_profiles mup ON mu.id = mup.user_id
         WHERE pm.project_id = p.id),
        '[]'
    )::json AS members
FROM projects p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE (p.user_id = $1 OR EXISTS (SELECT 1 FROM project_members pm2 WHERE pm2.project_id = p.id AND pm2.user_id = $1))
  AND p.is_deleted = false 
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR p.id < sqlc.narg('cursor_id')::uuid)
ORDER BY p.updated_at DESC
LIMIT sqlc.arg('limit');

-- name: GetProjectsByIDs :many
SELECT 
    p.*,
    COALESCE(
        (SELECT json_agg(json_build_object('id', c.id, 'edit_summary', c.edit_summary) ORDER BY c.created_at DESC)
         FROM commits c WHERE c.project_id = p.id AND c.is_deleted = false),
        '[]'
    )::json AS commits,
    COALESCE(
        (SELECT json_agg(json_build_object('id', s.id, 'status', s.status)) FROM submissions s WHERE s.project_id = p.id AND s.is_deleted = false),
        '[]'
    )::json AS submissions,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    COALESCE(
        (SELECT json_agg(json_build_object(
            'user_id', pm.user_id, 'role', pm.role,
            'display_name', mup.display_name, 'avatar_url', mup.avatar_url
         ) ORDER BY pm.role ASC, pm.created_at ASC)
         FROM project_members pm
         JOIN users mu ON pm.user_id = mu.id
         LEFT JOIN user_profiles mup ON mu.id = mup.user_id
         WHERE pm.project_id = p.id),
        '[]'
    )::json AS members
FROM projects p
JOIN users u ON p.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
WHERE p.id = ANY($1::uuid[]) AND p.is_deleted = false;

-- name: AddProjectMember :one
INSERT INTO project_members (
    project_id, user_id, role, invited_by
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateProjectMemberRole :one
UPDATE project_members
SET role = $3
WHERE project_id = $1 AND user_id = $2
RETURNING *;

-- name: RemoveProjectMember :exec
DELETE FROM project_members
WHERE project_id = $1 AND user_id = $2;

-- name: CheckProjectPermission :one
SELECT role FROM project_members
WHERE project_id = $1 AND user_id = $2;

-- name: ChangeProjectOwner :exec
UPDATE projects
SET user_id = $2, updated_at = NOW()
WHERE id = $1 AND is_deleted = false;

-- name: UpdateLatestCommit :exec
UPDATE projects
SET latest_commit_id = $2, updated_at = NOW()
WHERE id = $1 AND is_deleted = false;
