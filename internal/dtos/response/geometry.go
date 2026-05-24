package response

import (
	"encoding/json"
	"time"
)

type Bbox struct {
	MinLng float64 `json:"min_lng"`
	MinLat float64 `json:"min_lat"`
	MaxLng float64 `json:"max_lng"`
	MaxLat float64 `json:"max_lat"`
}

type GeometryResponse struct {
	ID           string          `json:"id"`
	GeoType      int16           `json:"geo_type"`
	DrawGeometry json.RawMessage `json:"draw_geometry"`
	BoundWith    *string         `json:"bound_with,omitempty"`
	TimeStart    int32           `json:"time_start,omitempty"`
	TimeEnd      int32           `json:"time_end,omitempty"`
	Bbox         *Bbox           `json:"bbox,omitempty"`
	ProjectID    string          `json:"project_id"`
	IsDeleted    bool            `json:"is_deleted,omitempty"`
	CreatedAt    *time.Time      `json:"created_at,omitempty"`
	UpdatedAt    *time.Time      `json:"updated_at,omitempty"`
}
