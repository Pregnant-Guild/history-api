CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS rag_chunks (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    source_type VARCHAR(50) NOT NULL,
    source_id UUID NOT NULL,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,        
    content TEXT NOT NULL,            
    embedding vector(1536),           
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_rag_chunks_source ON rag_chunks(source_type, source_id);
CREATE INDEX idx_rag_chunks_project ON rag_chunks(project_id);
CREATE INDEX idx_rag_chunks_embedding_hnsw ON rag_chunks USING hnsw (embedding vector_cosine_ops);

CREATE TRIGGER trigger_rag_chunks_updated_at
BEFORE UPDATE ON rag_chunks
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();
