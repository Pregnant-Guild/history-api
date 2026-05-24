-- name: CreateSubmission :one
INSERT INTO submissions (
    project_id, commit_id, user_id, status, content
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING 
    id, project_id, commit_id, user_id, created_at, status, reviewed_by, reviewed_at, review_note, content, is_deleted,
    (SELECT title FROM projects WHERE id = submissions.project_id) AS project_title,
    (SELECT description FROM projects WHERE id = submissions.project_id) AS project_description,
    (
        SELECT json_build_object(
            'id', u.id,
            'email', u.email,
            'display_name', up.display_name,
            'full_name', up.full_name,
            'avatar_url', up.avatar_url
        )
        FROM users u 
        LEFT JOIN user_profiles up ON u.id = up.user_id
        WHERE u.id = submissions.user_id
    )::json AS user,
    (
        SELECT json_build_object(
            'id', ru.id,
            'email', ru.email,
            'display_name', rup.display_name,
            'full_name', rup.full_name,
            'avatar_url', rup.avatar_url
        )
        FROM users ru 
        LEFT JOIN user_profiles rup ON ru.id = rup.user_id
        WHERE ru.id = submissions.reviewed_by
    )::json AS reviewer;

-- name: GetSubmissionById :one
SELECT 
    s.id, s.project_id, s.commit_id, s.user_id, s.created_at, s.status, s.reviewed_by, s.reviewed_at, s.review_note, s.content, s.is_deleted,
    p.title AS project_title, p.description AS project_description,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    CASE WHEN s.reviewed_by IS NOT NULL THEN
        json_build_object(
            'id', ru.id,
            'email', ru.email,
            'display_name', rup.display_name,
            'full_name', rup.full_name,
            'avatar_url', rup.avatar_url
        )::json
    ELSE NULL::json END AS reviewer
FROM submissions s
JOIN projects p ON s.project_id = p.id
JOIN users u ON s.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON s.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id
WHERE s.id = $1 AND s.is_deleted = false;

-- name: UpdateSubmission :one
UPDATE submissions
SET 
    status = COALESCE(sqlc.narg('status'), status),
    reviewed_by = COALESCE(sqlc.narg('reviewed_by'), reviewed_by),
    reviewed_at = CASE WHEN sqlc.narg('status') IS NOT NULL THEN NOW() ELSE reviewed_at END,
    review_note = COALESCE(sqlc.narg('review_note'), review_note),
    content = COALESCE(sqlc.narg('content'), content)
WHERE submissions.id = sqlc.arg('id')
RETURNING 
    submissions.id, submissions.project_id, submissions.commit_id, submissions.user_id, submissions.created_at, submissions.status, submissions.reviewed_by, submissions.reviewed_at, submissions.review_note, submissions.content, submissions.is_deleted,
    (SELECT title FROM projects WHERE id = submissions.project_id) AS project_title,
    (SELECT description FROM projects WHERE id = submissions.project_id) AS project_description,
    (
        SELECT json_build_object(
            'id', u.id,
            'email', u.email,
            'display_name', up.display_name,
            'full_name', up.full_name,
            'avatar_url', up.avatar_url
        )
        FROM users u 
        LEFT JOIN user_profiles up ON u.id = up.user_id
        WHERE u.id = submissions.user_id
    )::json AS user,
    (
        SELECT json_build_object(
            'id', ru.id,
            'email', ru.email,
            'display_name', rup.display_name,
            'full_name', rup.full_name,
            'avatar_url', rup.avatar_url
        )
        FROM users ru 
        LEFT JOIN user_profiles rup ON ru.id = rup.user_id
        WHERE ru.id = submissions.reviewed_by
    )::json AS reviewer;

-- name: DeleteSubmission :exec
UPDATE submissions
SET is_deleted = true
WHERE id = $1;

-- name: SearchSubmissions :many
SELECT 
    s.id, s.project_id, s.commit_id, s.user_id, s.created_at, s.status, s.reviewed_by, s.reviewed_at, s.review_note, s.content, s.is_deleted,
    p.title AS project_title, p.description AS project_description,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    CASE WHEN s.reviewed_by IS NOT NULL THEN
        json_build_object(
            'id', ru.id,
            'email', ru.email,
            'display_name', rup.display_name,
            'full_name', rup.full_name,
            'avatar_url', rup.avatar_url
        )::json
    ELSE NULL::json END AS reviewer
FROM submissions s
JOIN projects p ON s.project_id = p.id
JOIN users u ON s.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON s.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id
WHERE s.is_deleted = false
  AND (sqlc.narg('project_id')::uuid IS NULL OR s.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('user_ids')::uuid[] IS NULL OR s.user_id = ANY(sqlc.narg('user_ids')::uuid[]))
  AND (sqlc.narg('reviewed_by')::uuid IS NULL OR s.reviewed_by = sqlc.narg('reviewed_by'))
  AND (
      sqlc.narg('statuses')::smallint[] IS NULL 
      OR s.status = ANY(sqlc.narg('statuses')::smallint[])
  )
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR s.created_at >= sqlc.narg('created_from')::timestamptz)
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR s.created_at <= sqlc.narg('created_to')::timestamptz)
  AND (
      sqlc.narg('search_text')::text IS NULL OR 
      s.id::text ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
      s.review_note ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
      s.content ILIKE '%' || sqlc.narg('search_text')::text || '%'
  )
ORDER BY
    CASE WHEN sqlc.narg('sort') = 'created_at' AND sqlc.narg('order') = 'asc' THEN s.created_at END ASC,
    CASE WHEN sqlc.narg('sort') = 'created_at' AND sqlc.narg('order') = 'desc' THEN s.created_at END DESC,
    CASE WHEN sqlc.narg('sort') = 'reviewed_at' AND sqlc.narg('order') = 'asc' THEN s.reviewed_at END ASC,
    CASE WHEN sqlc.narg('sort') = 'reviewed_at' AND sqlc.narg('order') = 'desc' THEN s.reviewed_at END DESC,
    CASE WHEN sqlc.narg('sort') = 'status' AND sqlc.narg('order') = 'asc' THEN s.status END ASC,
    CASE WHEN sqlc.narg('sort') = 'status' AND sqlc.narg('order') = 'desc' THEN s.status END DESC,
    CASE WHEN sqlc.narg('sort') IS NULL THEN s.created_at END DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CountSubmissions :one
SELECT count(*)
FROM submissions s
WHERE s.is_deleted = false
  AND (sqlc.narg('project_id')::uuid IS NULL OR s.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('user_ids')::uuid[] IS NULL OR s.user_id = ANY(sqlc.narg('user_ids')::uuid[]))
  AND (sqlc.narg('reviewed_by')::uuid IS NULL OR s.reviewed_by = sqlc.narg('reviewed_by'))
  AND (
      sqlc.narg('statuses')::smallint[] IS NULL 
      OR s.status = ANY(sqlc.narg('statuses')::smallint[])
  )
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR s.created_at >= sqlc.narg('created_from')::timestamptz)
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR s.created_at <= sqlc.narg('created_to')::timestamptz)
  AND (
      sqlc.narg('search_text')::text IS NULL OR 
      s.id::text ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
      s.review_note ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
      s.content ILIKE '%' || sqlc.narg('search_text')::text || '%'
  );

-- name: GetSubmissionsByIDs :many
SELECT 
    s.id, s.project_id, s.commit_id, s.user_id, s.created_at, s.status, s.reviewed_by, s.reviewed_at, s.review_note, s.content, s.is_deleted,
    p.title AS project_title, p.description AS project_description,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS user,
    CASE WHEN s.reviewed_by IS NOT NULL THEN
        json_build_object(
            'id', ru.id,
            'email', ru.email,
            'display_name', rup.display_name,
            'full_name', rup.full_name,
            'avatar_url', rup.avatar_url
        )::json
    ELSE NULL::json END AS reviewer
FROM submissions s
JOIN projects p ON s.project_id = p.id
JOIN users u ON s.user_id = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON s.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id
WHERE s.id = ANY($1::uuid[]) AND s.is_deleted = false;

-- name: GetLatestApprovedSubmissionExcluding :one
SELECT 
    id, project_id, commit_id, user_id, created_at, status, reviewed_by, reviewed_at, review_note, content, is_deleted
FROM submissions
WHERE project_id = $1 
  AND status = 2
  AND id != $2
  AND is_deleted = false
ORDER BY reviewed_at DESC, created_at DESC
LIMIT 1;

-- name: GetLatestApprovedSubmission :one
SELECT 
    id, project_id, commit_id, user_id, created_at, status, reviewed_by, reviewed_at, review_note, content, is_deleted
FROM submissions
WHERE project_id = $1 
  AND status = 2
  AND is_deleted = false
ORDER BY reviewed_at DESC, created_at DESC
LIMIT 1;

