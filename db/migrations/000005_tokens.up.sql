CREATE TABLE IF NOT EXISTS user_tokens (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    token_type SMALLINT NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_user_tokens_token 
ON user_tokens(token)
WHERE is_deleted = false;

CREATE INDEX idx_user_tokens_user_id 
ON user_tokens(user_id)
WHERE is_deleted = false;

CREATE INDEX idx_user_tokens_type 
ON user_tokens(token_type)
WHERE is_deleted = false;

CREATE INDEX idx_user_tokens_expires_at 
ON user_tokens(expires_at)
WHERE is_deleted = false;