package response

import "encoding/json"

type SearchGeometriesByEntityNameResponse struct {
	Items      []*EntityGeometriesSearchItem `json:"items"`
	NextCursor string                        `json:"next_cursor,omitempty"`
}

type EntityGeometriesSearchItem struct {
	EntityID    string                     `json:"entity_id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Geometries  []*EntityGeometrySearchGeo `json:"geometries"`
}

type EntityGeometrySearchGeo struct {
	ID           string          `json:"id"`
	GeoType      int16           `json:"geo_type"`
	DrawGeometry json.RawMessage `json:"draw_geometry"`
	BoundWith    *string         `json:"bound_with,omitempty"`
	TimeStart    *int32          `json:"time_start,omitempty"`
	TimeEnd      *int32          `json:"time_end,omitempty"`
	ReplayIDs    []string        `json:"replay_ids,omitempty"`
}
