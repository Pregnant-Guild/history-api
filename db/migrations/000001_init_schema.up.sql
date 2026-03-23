CREATE EXTENSION IF NOT EXISTS postgis;

-- CREATE EXTENSION IF NOT EXISTS pgcrypto;

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

CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT,
    full_name TEXT,
    avatar_url TEXT,
    bio TEXT,
    location TEXT,
    website TEXT,
    country_code CHAR(2),
    phone TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE user_verifications (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    verify_type SMALLINT NOT NULL, -- 1 = ID_CARD, 2 = EDUCATION, 3 = EXPERT
    document_url TEXT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1, -- 1 pending, 2 approved, 3 rejected
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT UNIQUE NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_users_active_created_at 
ON users (created_at DESC) 
WHERE is_deleted = false;

CREATE INDEX idx_users_email_active
ON users (email)
WHERE is_deleted = false;

CREATE INDEX idx_users_active_verified
ON users (is_active, is_verified)
WHERE is_deleted = false;

CREATE INDEX idx_user_roles_user_id 
ON user_roles (user_id);

CREATE INDEX idx_user_roles_role_id 
ON user_roles (role_id);

CREATE INDEX idx_roles_active
ON roles (name)
WHERE is_deleted = false;

INSERT INTO roles (name) VALUES 
    ('USER'),
    ('ADMIN'),
    ('MOD'),
    ('HISTORIAN'),
    ('BANNED')
ON CONFLICT (name) DO NOTHING;

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


CREATE TABLE IF NOT EXISTS user_tokens (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    token_type SMALLINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_user_tokens_token 
ON user_tokens(token);

CREATE INDEX idx_user_tokens_user_id 
ON user_tokens(user_id);

CREATE INDEX idx_user_tokens_type 
ON user_tokens(token_type);

CREATE INDEX idx_user_tokens_expires_at 
ON user_tokens(expires_at);