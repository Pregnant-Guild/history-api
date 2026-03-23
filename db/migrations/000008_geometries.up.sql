CREATE TABLE IF NOT EXISTS geometries (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    geom GEOMETRY, -- point / polygon / line
    time_start INT,
    time_end INT,
    bbox GEOMETRY, -- optional
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS geo_versions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    geo_id UUID REFERENCES geometries(id) ON DELETE CASCADE,
    created_user UUID REFERENCES users(id),
    geom GEOMETRY,
    note TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    approved_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS entity_geometries (
    entity_id UUID REFERENCES entities(id) ON DELETE CASCADE,
    geometry_id UUID REFERENCES geometries(id) ON DELETE CASCADE,
    PRIMARY KEY (entity_id, geometry_id)
);

CREATE INDEX idx_geo_time ON geometries(time_start, time_end);
CREATE INDEX idx_geom_spatial ON geometries USING GIST (geom);

CREATE TRIGGER trigger_geometries_updated_at
BEFORE UPDATE ON geometries
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();