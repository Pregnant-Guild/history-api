CREATE TABLE IF NOT EXISTS system_statistics (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    date DATE UNIQUE NOT NULL,
    
    -- Cumulative stats
    total_users INT NOT NULL DEFAULT 0,
    total_projects INT NOT NULL DEFAULT 0,
    total_commits INT NOT NULL DEFAULT 0,
    total_submissions INT NOT NULL DEFAULT 0,
    total_medias INT NOT NULL DEFAULT 0,
    total_wikis INT NOT NULL DEFAULT 0,
    total_entities INT NOT NULL DEFAULT 0,
    total_geometries INT NOT NULL DEFAULT 0,
    total_storage_bytes BIGINT NOT NULL DEFAULT 0,

    -- Daily stats
    new_users INT NOT NULL DEFAULT 0,
    new_projects INT NOT NULL DEFAULT 0,
    new_commits INT NOT NULL DEFAULT 0,
    new_submissions INT NOT NULL DEFAULT 0,
    new_medias INT NOT NULL DEFAULT 0,
    new_wikis INT NOT NULL DEFAULT 0,
    new_entities INT NOT NULL DEFAULT 0,
    new_geometries INT NOT NULL DEFAULT 0,
    new_storage_bytes BIGINT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT now()
);
