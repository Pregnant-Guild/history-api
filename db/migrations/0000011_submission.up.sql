CREATE TABLE IF NOT EXISTS submissions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    revision_id UUID NOT NULL REFERENCES revisions(id) ON DELETE CASCADE,
    submitted_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status SMALLINT NOT NULL DEFAULT 1,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_submissions_revision_id 
ON submissions (revision_id);

CREATE INDEX idx_submissions_moderator_queue 
ON submissions (status, submitted_at ASC);

CREATE INDEX idx_submissions_user_history 
ON submissions (submitted_by, status, submitted_at DESC);

CREATE INDEX idx_submissions_project_queue 
ON submissions (project_id, submitted_at DESC);