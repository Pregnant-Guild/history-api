package models

import (
	"encoding/json"
	"history-api/internal/dtos/response"
	"time"
)

type BattleReplayEntity struct {
	ID                string          `json:"id"`
	GeometryID        string          `json:"geometry_id"`
	ProjectID         string          `json:"project_id"`
	TargetGeometryIDs json.RawMessage `json:"target_geometry_ids"`
	Detail            json.RawMessage `json:"detail"`
	IsDeleted         bool            `json:"is_deleted"`
	CreatedAt         *time.Time      `json:"created_at"`
	UpdatedAt         *time.Time      `json:"updated_at"`
}

func (b *BattleReplayEntity) ToResponse() *response.BattleReplayResponse {
	if b == nil {
		return nil
	}
	return &response.BattleReplayResponse{
		ID:                b.ID,
		GeometryID:        b.GeometryID,
		ProjectID:         b.ProjectID,
		TargetGeometryIDs: b.TargetGeometryIDs,
		Detail:            b.Detail,
		IsDeleted:         b.IsDeleted,
		CreatedAt:         b.CreatedAt,
		UpdatedAt:         b.UpdatedAt,
	}
}

func BattleReplaysEntityToResponse(bs []*BattleReplayEntity) []*response.BattleReplayResponse {
	out := make([]*response.BattleReplayResponse, 0)
	if bs == nil {
		return out
	}
	for _, b := range bs {
		if b == nil {
			continue
		}
		out = append(out, b.ToResponse())
	}
	return out
}
