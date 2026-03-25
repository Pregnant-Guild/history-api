CREATE TABLE IF NOT EXISTS entity_types (
    id SMALLSERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS entities (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    type_id SMALLINT REFERENCES entity_types(id),
    name TEXT NOT NULL,
    slug TEXT UNIQUE,
    description TEXT,
    thumbnail_url TEXT,
    status SMALLINT DEFAULT 1, -- 1 draft, 2 published
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX uniq_entities_slug_active
ON entities(slug)
WHERE is_deleted = false;

CREATE INDEX idx_entities_type 
ON entities(type_id)
WHERE is_deleted = false;


CREATE INDEX idx_entities_status_created 
ON entities(status, created_at DESC)
WHERE is_deleted = false;


CREATE INDEX idx_entities_type_status 
ON entities(type_id, status)
WHERE is_deleted = false;


CREATE INDEX idx_entities_reviewed_by
ON entities(reviewed_by)
WHERE is_deleted = false;

CREATE INDEX idx_entities_name_search
ON entities USING gin (name gin_trgm_ops);

CREATE TRIGGER trigger_entities_updated_at
BEFORE UPDATE ON entities
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();