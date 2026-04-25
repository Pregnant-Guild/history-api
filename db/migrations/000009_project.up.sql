CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    title TEXT NOT NULL,
    description TEXT,
    latest_revision_id UUID, 
    version_count INT NOT NULL DEFAULT 0,
    project_status SMALLINT NOT NULL DEFAULT 1,
    locked_by UUID, 
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE INDEX idx_projects_latest_revision_id 
ON projects (latest_revision_id);

CREATE INDEX idx_projects_user_status_updated 
ON projects (user_id, project_status, updated_at DESC);

CREATE INDEX idx_projects_status_updated 
ON projects (project_status, updated_at DESC);

CREATE INDEX idx_projects_title_trgm 
ON projects USING GIN (title gin_trgm_ops);