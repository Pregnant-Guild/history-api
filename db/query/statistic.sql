-- name: UpsertSystemStatistics :one
INSERT INTO system_statistics (
    date,
    total_users, total_projects, total_commits, total_submissions, total_medias, total_wikis, total_entities, total_geometries, total_storage_bytes,
    new_users, new_projects, new_commits, new_submissions, new_medias, new_wikis, new_entities, new_geometries, new_storage_bytes
) VALUES (
    $1,
    (SELECT COUNT(*)::INT FROM users),
    (SELECT COUNT(*)::INT FROM projects),
    (SELECT COUNT(*)::INT FROM commits),
    (SELECT COUNT(*)::INT FROM submissions),
    (SELECT COUNT(*)::INT FROM medias),
    (SELECT COUNT(*)::INT FROM wikis),
    (SELECT COUNT(*)::INT FROM entities),
    (SELECT COUNT(*)::INT FROM geometries),
    COALESCE((SELECT SUM(size)::BIGINT FROM medias), 0),

    (SELECT COUNT(*)::INT FROM users WHERE created_at >= $1::DATE AND created_at < $1::DATE + INTERVAL '1 day'),
    (SELECT COUNT(*)::INT FROM projects WHERE created_at >= $1::DATE AND created_at < $1::DATE + INTERVAL '1 day'),
    (SELECT COUNT(*)::INT FROM commits WHERE created_at >= $1::DATE AND created_at < $1::DATE + INTERVAL '1 day'),
    (SELECT COUNT(*)::INT FROM submissions WHERE created_at >= $1::DATE AND created_at < $1::DATE + INTERVAL '1 day'),
    (SELECT COUNT(*)::INT FROM medias WHERE created_at >= $1::DATE AND created_at < $1::DATE + INTERVAL '1 day'),
    (SELECT COUNT(*)::INT FROM wikis WHERE created_at >= $1::DATE AND created_at < $1::DATE + INTERVAL '1 day'),
    (SELECT COUNT(*)::INT FROM entities WHERE created_at >= $1::DATE AND created_at < $1::DATE + INTERVAL '1 day'),
    (SELECT COUNT(*)::INT FROM geometries WHERE created_at >= $1::DATE AND created_at < $1::DATE + INTERVAL '1 day'),
    COALESCE((SELECT SUM(size)::BIGINT FROM medias WHERE created_at >= $1::DATE AND created_at < $1::DATE + INTERVAL '1 day'), 0)
)
ON CONFLICT (date) DO UPDATE SET
    total_users = EXCLUDED.total_users,
    total_projects = EXCLUDED.total_projects,
    total_commits = EXCLUDED.total_commits,
    total_submissions = EXCLUDED.total_submissions,
    total_medias = EXCLUDED.total_medias,
    total_wikis = EXCLUDED.total_wikis,
    total_entities = EXCLUDED.total_entities,
    total_geometries = EXCLUDED.total_geometries,
    total_storage_bytes = EXCLUDED.total_storage_bytes,
    new_users = EXCLUDED.new_users,
    new_projects = EXCLUDED.new_projects,
    new_commits = EXCLUDED.new_commits,
    new_submissions = EXCLUDED.new_submissions,
    new_medias = EXCLUDED.new_medias,
    new_wikis = EXCLUDED.new_wikis,
    new_entities = EXCLUDED.new_entities,
    new_geometries = EXCLUDED.new_geometries,
    new_storage_bytes = EXCLUDED.new_storage_bytes
RETURNING *;

-- name: SearchSystemStatistics :many
SELECT * FROM system_statistics 
WHERE 
    (sqlc.narg('start_date')::DATE IS NULL OR date >= sqlc.narg('start_date')::DATE) AND
    (sqlc.narg('end_date')::DATE IS NULL OR date <= sqlc.narg('end_date')::DATE)
ORDER BY date DESC;

-- name: GetSystemStatisticsByDate :one
SELECT * FROM system_statistics WHERE date = $1 LIMIT 1;

-- name: GetSystemStatisticsByIDs :many
SELECT * FROM system_statistics WHERE id = ANY($1::UUID[]);


