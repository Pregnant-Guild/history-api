CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT,
    google_id VARCHAR(255) UNIQUE,
    auth_provider VARCHAR(50) NOT NULL DEFAULT 'local',
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    token_version INT NOT NULL DEFAULT 1,
    refresh_token TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

ALTER TABLE users DROP CONSTRAINT IF EXISTS check_auth_provider;
ALTER TABLE users ADD CONSTRAINT check_auth_provider 
CHECK (auth_provider IN ('local', 'google', 'facebook', 'github'));

CREATE INDEX idx_users_provider_created_at ON users (auth_provider, created_at DESC);

CREATE INDEX idx_users_email_active
ON users (email)
WHERE is_deleted = false;

CREATE INDEX idx_users_email_trgm ON users USING gin (email gin_trgm_ops);
CREATE INDEX idx_users_id_trgm ON users USING gin ((id::text) gin_trgm_ops);

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
