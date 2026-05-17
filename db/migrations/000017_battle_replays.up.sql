CREATE TABLE IF NOT EXISTS battle_replays (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    geometry_id UUID NOT NULL REFERENCES geometries(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    target_geometry_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    detail JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_battle_replays_geometry_id ON battle_replays(geometry_id)
    WHERE is_deleted = false;

CREATE INDEX idx_battle_replays_project_id ON battle_replays(project_id)
    WHERE is_deleted = false;

CREATE INDEX idx_battle_replays_target_geometry_ids ON battle_replays USING GIN (target_geometry_ids)
    WHERE is_deleted = false;

CREATE INDEX idx_battle_replays_updated_at ON battle_replays(updated_at DESC)
    WHERE is_deleted = false;

DROP TRIGGER IF EXISTS trigger_battle_replays_updated_at ON battle_replays;
CREATE TRIGGER trigger_battle_replays_updated_at
BEFORE UPDATE ON battle_replays
FOR EACH ROW
EXECUTE FUNCTION update_updated_at();
