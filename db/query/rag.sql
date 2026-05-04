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
    id, source_type, source_id, project_id, chunk_index, content,
    (1 - (embedding <=> sqlc.arg('embedding')))::float8 AS similarity
FROM rag_chunks
WHERE 1=1
  AND (sqlc.narg('project_id')::uuid IS NULL OR project_id = sqlc.narg('project_id')::uuid)
  AND (sqlc.narg('source_type')::varchar IS NULL OR source_type = sqlc.narg('source_type')::varchar)
  AND (1 - (embedding <=> sqlc.arg('embedding')))::float8 >= sqlc.arg('match_threshold')::float8
ORDER BY embedding <=> sqlc.arg('embedding')
LIMIT sqlc.arg('match_count');

-- name: DeleteRagChunksBySourceIDs :exec
DELETE FROM rag_chunks
WHERE source_type = $1 AND source_id = ANY($2::uuid[]);
