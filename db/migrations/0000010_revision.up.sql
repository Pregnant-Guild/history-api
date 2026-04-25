CREATE TABLE IF NOT EXISTS revisions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version_no INT NOT NULL,
    snapshot_json JSONB NOT NULL,
    snapshot_hash TEXT,
    parent_id UUID REFERENCES revisions(id), 
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    edit_summary TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_revisions_project_history 
ON revisions (project_id, version_no DESC);

CREATE INDEX idx_revisions_user_history 
ON revisions (user_id, created_at DESC);    

CREATE INDEX idx_revisions_parent_id 
ON revisions (parent_id);