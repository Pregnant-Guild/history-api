CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE medias (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    storage_key VARCHAR(255) UNIQUE NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    file_metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_medias_user_created ON medias (user_id, created_at DESC);
CREATE INDEX idx_medias_original_name_trgm ON medias USING GIN (original_name gin_trgm_ops);