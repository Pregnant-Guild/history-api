CREATE TABLE IF NOT EXISTS commits (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    snapshot_json JSONB NOT NULL,
    snapshot_hash TEXT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    edit_summary TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_commits_project_history 
ON commits (project_id, created_at DESC);

CREATE INDEX idx_commits_user_history 
ON commits (user_id, created_at DESC);    