CREATE TABLE IF NOT EXISTS wiki_content (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    wiki_id UUID NOT NULL REFERENCES wikis(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    content TEXT,
    preview TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_wiki_content_wiki_id ON wiki_content(wiki_id);
CREATE INDEX idx_wiki_content_created_at ON wiki_content(created_at DESC);
