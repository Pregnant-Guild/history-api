CREATE TABLE IF NOT EXISTS user_verifications (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    verify_type SMALLINT NOT NULL, -- 1 = ID_CARD, 2 = EDUCATION, 3 = EXPERT
    document_url TEXT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1, -- 1 pending, 2 approved, 3 rejected
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);


CREATE INDEX idx_user_verifications_user_id ON user_verifications(user_id);
CREATE INDEX idx_user_verifications_user_type ON user_verifications(user_id, verify_type);
CREATE INDEX idx_user_verifications_status ON user_verifications(status);

CREATE INDEX idx_user_verifications_status_created 
ON user_verifications(status, created_at DESC);