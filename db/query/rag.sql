-- name: CreateRagChunk :one
INSERT INTO rag_chunks (
    id, source_type, source_id, project_id, chunk_index, content, embedding
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, uuidv7()), 
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: SearchRagChunks :many
SELECT 
    id, source_type, source_id, project_id, chunk_index, content, created_at, updated_at,
    (1 - (embedding <=> sqlc.arg('embedding')))::float8 AS similarity
FROM rag_chunks
WHERE 1=1
  AND (sqlc.narg('project_id')::uuid IS NULL OR project_id = sqlc.narg('project_id')::uuid)
  AND (sqlc.narg('source_type')::varchar IS NULL OR source_type = sqlc.narg('source_type')::varchar)
  AND (1 - (embedding <=> sqlc.arg('embedding')))::float8 >= sqlc.arg('match_threshold')::float8
ORDER BY embedding <=> sqlc.arg('embedding')
LIMIT sqlc.arg('match_count');

-- name: SearchRagChunksLexical :many
WITH terms AS (
    SELECT DISTINCT trim(term) AS term
    FROM unnest(sqlc.arg('terms')::text[]) AS term
    WHERE trim(term) <> ''
),
matched AS (
    SELECT
        r.id,
        r.source_type,
        r.source_id,
        r.project_id,
        r.chunk_index,
        r.content,
        r.created_at,
        r.updated_at,
        COUNT(*)::float8 AS similarity
    FROM rag_chunks r
    JOIN terms t ON r.content ILIKE '%' || t.term || '%'
    WHERE (sqlc.narg('project_id')::uuid IS NULL OR r.project_id = sqlc.narg('project_id')::uuid)
    GROUP BY
        r.id,
        r.source_type,
        r.source_id,
        r.project_id,
        r.chunk_index,
        r.content,
        r.created_at,
        r.updated_at
)
SELECT
    id, source_type, source_id, project_id, chunk_index, content, created_at, updated_at, similarity
FROM matched
ORDER BY similarity DESC, chunk_index ASC
LIMIT sqlc.arg('match_count');

-- name: DeleteRagChunksBySourceIDs :exec
DELETE FROM rag_chunks
WHERE source_type = $1 AND source_id = ANY($2::uuid[]);
