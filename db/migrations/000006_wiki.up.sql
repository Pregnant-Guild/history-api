CREATE TABLE IF NOT EXISTS wikis (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    title TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    note TEXT,
    content TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_wiki_entity 
ON wikis(entity_id)
WHERE is_deleted = false;

CREATE TRIGGER trigger_wikis_updated_at
BEFORE UPDATE ON wikis
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();