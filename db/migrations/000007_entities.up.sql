CREATE TABLE IF NOT EXISTS entities (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT,
    description TEXT,
    status SMALLINT,
    time_start INT,
    time_end INT,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);


CREATE INDEX idx_entities_name_search
ON entities USING GIN (name gin_trgm_ops);

CREATE INDEX idx_entities_project_id ON entities(project_id);

CREATE INDEX idx_entities_time_range
ON entities USING GIST (int4range(time_start, time_end, '[]'))
WHERE is_deleted = false;

CREATE INDEX idx_entities_created_active 
ON entities(created_at DESC)
WHERE is_deleted = false;

CREATE TRIGGER trigger_entities_updated_at
BEFORE UPDATE ON entities
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();