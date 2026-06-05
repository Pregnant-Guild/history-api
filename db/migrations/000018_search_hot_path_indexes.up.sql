CREATE INDEX IF NOT EXISTS idx_wikis_active_id_desc
ON wikis (id DESC)
WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS idx_wikis_active_project_id_id_desc
ON wikis (project_id, id DESC)
WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS idx_entities_active_id_desc
ON entities (id DESC)
WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS idx_entities_active_project_id_id_desc
ON entities (project_id, id DESC)
WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS idx_geometries_active_id_desc
ON geometries (id DESC)
WHERE is_deleted = false;

CREATE INDEX IF NOT EXISTS idx_geometries_active_project_id_id_desc
ON geometries (project_id, id DESC)
WHERE is_deleted = false;
