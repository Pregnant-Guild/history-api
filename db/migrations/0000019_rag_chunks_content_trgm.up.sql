CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_rag_chunks_content_trgm
ON rag_chunks USING GIN (content gin_trgm_ops);
