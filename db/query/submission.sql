-- name: CreateSubmission :one
WITH inserted_submission AS (
    INSERT INTO submissions (
        project_id, revision_id, submitted_by, status
    ) VALUES (
        $1, $2, $3, $4
    )
    RETURNING *
)
SELECT 
    s.id, s.project_id, s.revision_id, s.submitted_by, s.submitted_at, s.status, s.reviewed_by, s.reviewed_at, s.review_note, s.is_deleted,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS submitter,
    CASE WHEN s.reviewed_by IS NOT NULL THEN
        json_build_object(
            'id', ru.id,
            'email', ru.email,
            'display_name', rup.display_name,
            'full_name', rup.full_name,
            'avatar_url', rup.avatar_url
        )::json
    ELSE NULL::json END AS reviewer
FROM inserted_submission s
JOIN users u ON s.submitted_by = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON s.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id;

-- name: GetSubmissionById :one
SELECT 
    s.id, s.project_id, s.revision_id, s.submitted_by, s.submitted_at, s.status, s.reviewed_by, s.reviewed_at, s.review_note, s.is_deleted,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS submitter,
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
JOIN users u ON s.submitted_by = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON s.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id
WHERE s.id = $1 AND s.is_deleted = false;

-- name: UpdateSubmission :one
UPDATE submissions
SET 
    status = COALESCE(sqlc.narg('status'), status),
    reviewed_by = COALESCE(sqlc.narg('reviewed_by'), reviewed_by),
    reviewed_at = COALESCE(sqlc.narg('reviewed_at'), reviewed_at),
    review_note = COALESCE(sqlc.narg('review_note'), review_note)
FROM submissions s
JOIN users u ON s.submitted_by = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON s.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id
WHERE s.id = sqlc.arg('id')
RETURNING 
    s.id, s.project_id, s.revision_id, s.submitted_by, s.submitted_at, s.status, s.reviewed_by, s.reviewed_at, s.review_note, s.is_deleted,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS submitter,
    CASE WHEN s.reviewed_by IS NOT NULL THEN
        json_build_object(
            'id', ru.id,
            'email', ru.email,
            'display_name', rup.display_name,
            'full_name', rup.full_name,
            'avatar_url', rup.avatar_url
        )::json
    ELSE NULL::json END AS reviewer;

-- name: DeleteSubmission :exec
UPDATE submissions
SET is_deleted = true
WHERE id = $1;

-- name: SearchSubmissions :many
SELECT 
    s.id, s.project_id, s.revision_id, s.submitted_by, s.submitted_at, s.status, s.reviewed_by, s.reviewed_at, s.review_note, s.is_deleted,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS submitter,
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
JOIN users u ON s.submitted_by = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON s.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id
WHERE s.is_deleted = false
  AND (sqlc.narg('project_id')::uuid IS NULL OR s.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('submitted_by')::uuid IS NULL OR s.submitted_by = sqlc.narg('submitted_by'))
  AND (sqlc.narg('reviewed_by')::uuid IS NULL OR s.reviewed_by = sqlc.narg('reviewed_by'))
  AND (
      sqlc.narg('statuses')::text[] IS NULL 
      OR s.status = ANY(sqlc.narg('statuses')::text[])
  )
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR s.submitted_at >= sqlc.narg('created_from')::timestamptz)
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR s.submitted_at <= sqlc.narg('created_to')::timestamptz)
  AND (
      sqlc.narg('search_text')::text IS NULL OR 
      s.id::text ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
      s.review_note ILIKE '%' || sqlc.narg('search_text')::text || '%'
  )
ORDER BY
    CASE WHEN sqlc.narg('sort') = 'submitted_at' AND sqlc.narg('order') = 'asc' THEN s.submitted_at END ASC,
    CASE WHEN sqlc.narg('sort') = 'submitted_at' AND sqlc.narg('order') = 'desc' THEN s.submitted_at END DESC,
    CASE WHEN sqlc.narg('sort') = 'reviewed_at' AND sqlc.narg('order') = 'asc' THEN s.reviewed_at END ASC,
    CASE WHEN sqlc.narg('sort') = 'reviewed_at' AND sqlc.narg('order') = 'desc' THEN s.reviewed_at END DESC,
    CASE WHEN sqlc.narg('sort') = 'status' AND sqlc.narg('order') = 'asc' THEN s.status END ASC,
    CASE WHEN sqlc.narg('sort') = 'status' AND sqlc.narg('order') = 'desc' THEN s.status END DESC,
    CASE WHEN sqlc.narg('sort') IS NULL THEN s.submitted_at END DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CountSubmissions :one
SELECT count(*)
FROM submissions s
WHERE s.is_deleted = false
  AND (sqlc.narg('project_id')::uuid IS NULL OR s.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('submitted_by')::uuid IS NULL OR s.submitted_by = sqlc.narg('submitted_by'))
  AND (sqlc.narg('reviewed_by')::uuid IS NULL OR s.reviewed_by = sqlc.narg('reviewed_by'))
  AND (
      sqlc.narg('statuses')::text[] IS NULL 
      OR s.status = ANY(sqlc.narg('statuses')::text[])
  )
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR s.submitted_at >= sqlc.narg('created_from')::timestamptz)
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR s.submitted_at <= sqlc.narg('created_to')::timestamptz)
  AND (
      sqlc.narg('search_text')::text IS NULL OR 
      s.id::text ILIKE '%' || sqlc.narg('search_text')::text || '%' OR
      s.review_note ILIKE '%' || sqlc.narg('search_text')::text || '%'
  );


-- name: GetSubmissionsByIDs :many
SELECT 
    s.id, s.project_id, s.revision_id, s.submitted_by, s.submitted_at, s.status, s.reviewed_by, s.reviewed_at, s.review_note, s.is_deleted,
    json_build_object(
        'id', u.id,
        'email', u.email,
        'display_name', up.display_name,
        'full_name', up.full_name,
        'avatar_url', up.avatar_url
    )::json AS submitter,
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
JOIN users u ON s.submitted_by = u.id
LEFT JOIN user_profiles up ON u.id = up.user_id
LEFT JOIN users ru ON s.reviewed_by = ru.id
LEFT JOIN user_profiles rup ON ru.id = rup.user_id
WHERE s.id = ANY($1::uuid[]) AND s.is_deleted = false;
