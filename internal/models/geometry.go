package models

import (
	"encoding/json"
	"history-api/internal/dtos/response"
	"time"
)

type GeometryEntity struct {
	ID           string          `json:"id"`
	GeoType      string          `json:"geo_type"`
	DrawGeometry json.RawMessage `json:"draw_geometry"`
	Binding      json.RawMessage `json:"binding"`
	TimeStart    int32           `json:"time_start"`
	TimeEnd      int32           `json:"time_end"`
	Bbox         *response.Bbox  `json:"bbox"`
	IsDeleted    bool            `json:"is_deleted"`
	CreatedAt    *time.Time      `json:"created_at"`
	UpdatedAt    *time.Time      `json:"updated_at"`
}

func (g *GeometryEntity) ToResponse() *response.GeometryResponse {
	if g == nil {
		return nil
	}
	return &response.GeometryResponse{
		ID:           g.ID,
		GeoType:      g.GeoType,
		DrawGeometry: g.DrawGeometry,
		Binding:      g.Binding,
		TimeStart:    g.TimeStart,
		TimeEnd:      g.TimeEnd,
		Bbox:         g.Bbox,
		IsDeleted:    g.IsDeleted,
		CreatedAt:    g.CreatedAt,
		UpdatedAt:    g.UpdatedAt,
	}
}

func GeometriesEntityToResponse(gs []*GeometryEntity) []*response.GeometryResponse {
	out := make([]*response.GeometryResponse, 0)
	if gs == nil {
		return out
	}
	for _, g := range gs {
		if g == nil {
			continue
		}
		out = append(out, g.ToResponse())
	}
	return out
}
