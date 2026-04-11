CREATE TABLE IF NOT EXISTS user_verifications (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    verify_type SMALLINT NOT NULL,
    content TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    status SMALLINT NOT NULL DEFAULT 1,
    reviewed_by UUID REFERENCES users(id),
    review_note TEXT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS verification_medias (
    verification_id UUID REFERENCES user_verifications(id) ON DELETE CASCADE,
    media_id UUID REFERENCES medias(id) ON DELETE CASCADE,
    PRIMARY KEY (verification_id, media_id)
);

CREATE INDEX idx_user_verifications_user_type 
ON user_verifications(user_id, verify_type)
WHERE is_deleted = false;

CREATE INDEX idx_user_verifications_status_created 
ON user_verifications(status, created_at DESC)
WHERE is_deleted = false;

CREATE INDEX idx_verification_medias_media_id 
ON verification_medias(media_id);