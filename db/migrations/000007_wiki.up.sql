CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS wikis (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    title TEXT,
    content TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);


CREATE TABLE IF NOT EXISTS entity_wikis (
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,
    wiki_id UUID REFERENCES wikis(id) ON DELETE CASCADE,
    PRIMARY KEY (entity_id, wiki_id)
);

CREATE INDEX idx_entity_wikis_wiki_id
ON entity_wikis(wiki_id);

CREATE INDEX idx_wikis_created_active 
ON wikis(created_at DESC)
WHERE is_deleted = false;

CREATE INDEX idx_wikis_title_search
ON wikis USING GIN (title gin_trgm_ops)
WHERE is_deleted = false;

CREATE TRIGGER trigger_wikis_updated_at
BEFORE UPDATE ON wikis
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();