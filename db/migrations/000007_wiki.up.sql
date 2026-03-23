CREATE TABLE IF NOT EXISTS wiki_pages (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,
    title TEXT,
    content TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wiki_versions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    wiki_id UUID REFERENCES wiki_pages(id) ON DELETE CASCADE,
    created_user UUID REFERENCES users(id),
    note TEXT,
    content TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    approved_at TIMESTAMPTZ
);

CREATE INDEX idx_wiki_entity ON wiki_pages(entity_id);

CREATE TRIGGER trigger_wiki_pages_updated_at
BEFORE UPDATE ON wiki_pages
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();