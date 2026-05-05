package response

import "encoding/json"

// SearchGeometriesByEntityNameResponse groups geometries by matched entities.
// Cursor pagination is based on entity_id (descending).
type SearchGeometriesByEntityNameResponse struct {
	Items      []*EntityGeometriesSearchItem `json:"items"`
	NextCursor string                       `json:"next_cursor,omitempty"`
}

type EntityGeometriesSearchItem struct {
	EntityID    string                    `json:"entity_id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Geometries  []*EntityGeometrySearchGeo `json:"geometries"`
}

type EntityGeometrySearchGeo struct {
	ID           string `json:"id"`
	GeoType      int16  `json:"geo_type"`
	DrawGeometry json.RawMessage `json:"draw_geometry"`
	Binding      json.RawMessage `json:"binding,omitempty"`
	TimeStart    *int32 `json:"time_start,omitempty"`
	TimeEnd      *int32 `json:"time_end,omitempty"`
}
