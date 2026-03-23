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
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_entities_slug ON entities(slug);

CREATE TRIGGER trigger_entities_updated_at
BEFORE UPDATE ON entities
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();