CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS geometries (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    geom GEOMETRY, -- point / polygon / line
    time_start INT,
    time_end INT,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    bbox GEOMETRY, -- optional
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS geo_versions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    geo_id UUID REFERENCES geometries(id) ON DELETE CASCADE,
    created_user UUID REFERENCES users(id),
    geom GEOMETRY,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    note TEXT,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS entity_geometries (
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,
    geometry_id UUID REFERENCES geometries(id) ON DELETE CASCADE,
    PRIMARY KEY (entity_id, geometry_id)
);

CREATE INDEX idx_geom_spatial_active 
ON geometries USING GIST (geom)
WHERE is_deleted = false;

CREATE INDEX idx_geom_bbox 
ON geometries USING GIST (bbox)
WHERE is_deleted = false;

CREATE INDEX idx_geom_time_range
ON geometries
USING GIST (int4range(time_start, time_end))
WHERE is_deleted = false;

CREATE INDEX idx_geo_versions_geo_id
ON geo_versions(geo_id)
WHERE is_deleted = false;

CREATE INDEX idx_geo_versions_reviewed_by
ON geo_versions(reviewed_by)
WHERE is_deleted = false;

CREATE INDEX idx_geo_versions_created_at
ON geo_versions(created_at DESC)
WHERE is_deleted = false;

CREATE INDEX idx_entity_geometries_geometry
ON entity_geometries(geometry_id);

CREATE TRIGGER trigger_geometries_updated_at
BEFORE UPDATE ON geometries
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();