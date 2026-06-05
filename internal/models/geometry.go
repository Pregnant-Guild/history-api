package models

import (
	"encoding/json"
	"history-api/internal/dtos/response"
	"time"
)

type GeometryEntity struct {
	ID           string          `json:"id"`
	GeoType      int16           `json:"geo_type"`
	DrawGeometry json.RawMessage `json:"draw_geometry"`
	BoundWith    *string         `json:"bound_with"`
	TimeStart    int32           `json:"time_start"`
	TimeEnd      int32           `json:"time_end"`
	Bbox         *response.Bbox  `json:"bbox"`
	ProjectID    string          `json:"project_id"`
	IsDeleted    bool            `json:"is_deleted"`
	CreatedAt    *time.Time      `json:"created_at"`
	UpdatedAt    *time.Time      `json:"updated_at"`
}

type EntityGeometriesSearchEntity struct {
	EntityID          string          `json:"entity_id"`
	EntityName        string          `json:"name"`
	EntityDescription string          `json:"description"`
	GeometryID        string          `json:"id"`
	GeoType           int16           `json:"geo_type"`
	DrawGeometry      json.RawMessage `json:"draw_geometry"`
	BoundWith         *string         `json:"bound_with,omitempty"`
	TimeStart         *int32          `json:"time_start,omitempty"`
	TimeEnd           *int32          `json:"time_end,omitempty"`
}

func (g *GeometryEntity) ToResponse() *response.GeometryResponse {
	if g == nil {
		return nil
	}
	return &response.GeometryResponse{
		ID:           g.ID,
		GeoType:      g.GeoType,
		DrawGeometry: g.DrawGeometry,
		BoundWith:    g.BoundWith,
		TimeStart:    g.TimeStart,
		TimeEnd:      g.TimeEnd,
		Bbox:         g.Bbox,
		ProjectID:    g.ProjectID,
		IsDeleted:    g.IsDeleted,
		CreatedAt:    g.CreatedAt,
		UpdatedAt:    g.UpdatedAt,
	}
}

func GeometriesEntityToResponse(gs []*GeometryEntity) []*response.GeometryResponse {
	if gs == nil {
		return []*response.GeometryResponse{}
	}
	out := make([]*response.GeometryResponse, 0, len(gs))
	for _, g := range gs {
		if g == nil {
			continue
		}
		out = append(out, g.ToResponse())
	}
	return out
}
