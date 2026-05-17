package response

import (
	"encoding/json"
	"time"
)

type BattleReplayResponse struct {
	ID                string          `json:"id"`
	GeometryID        string          `json:"geometry_id"`
	ProjectID         string          `json:"project_id"`
	TargetGeometryIDs json.RawMessage `json:"target_geometry_ids,omitempty"`
	Detail            json.RawMessage `json:"detail,omitempty"`
	IsDeleted         bool            `json:"is_deleted,omitempty"`
	CreatedAt         *time.Time      `json:"created_at,omitempty"`
	UpdatedAt         *time.Time      `json:"updated_at,omitempty"`
}
