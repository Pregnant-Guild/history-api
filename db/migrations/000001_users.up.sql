CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    google_id VARCHAR(255) UNIQUE,
    auth_provider VARCHAR(50) NOT NULL DEFAULT 'local',
    is_verified BOOLEAN NOT NULL DEFAULT false,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    token_version INT NOT NULL DEFAULT 1,
    refresh_token TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_users_active_created_at 
ON users (created_at DESC) 
WHERE is_deleted = false;

CREATE INDEX idx_users_email_active
ON users (email)
WHERE is_deleted = false;

CREATE INDEX idx_users_verified
ON users (is_verified)
WHERE is_deleted = false;

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
