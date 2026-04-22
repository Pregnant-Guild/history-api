CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS geometries (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    geo_type VARCHAR(50) NOT NULL DEFAULT 'id',
    draw_geometry JSONB NOT NULL,
    binding JSONB,
    time_start INT,
    time_end INT,
    bbox GEOMETRY,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

ALTER TABLE geometries DROP CONSTRAINT IF EXISTS check_geo_type;
ALTER TABLE geometries ADD CONSTRAINT check_geo_type 
CHECK (geo_type IN ('id', 'name', 'icon', 'variant', 'description'));

CREATE TABLE IF NOT EXISTS entity_geometries (
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,
    geometry_id UUID REFERENCES geometries(id) ON DELETE CASCADE,
    PRIMARY KEY (entity_id, geometry_id)
);

CREATE INDEX idx_geom_draw_geometry 
ON geometries USING GIN (draw_geometry);

CREATE INDEX idx_geom_bbox 
ON geometries USING GIST (bbox)
WHERE is_deleted = false;

CREATE INDEX idx_geom_time_range
ON geometries USING GIST (int4range(time_start, time_end))
WHERE is_deleted = false;

CREATE INDEX idx_entity_geometries_geometry
ON entity_geometries(geometry_id);

CREATE INDEX idx_geom_binding 
ON geometries USING GIN (binding);

CREATE INDEX idx_geom_updated_at 
ON geometries (updated_at DESC) 
WHERE is_deleted = false;

CREATE TRIGGER trigger_geometries_updated_at
BEFORE UPDATE ON geometries
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();