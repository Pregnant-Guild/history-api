package request

type SearchGeometryDto struct {
	MinLng    *float64 `json:"min_lng" query:"min_lng" validate:"required,gte=-180,lte=180"`
	MinLat    *float64 `json:"min_lat" query:"min_lat" validate:"required,gte=-90,lte=90"`
	MaxLng    *float64 `json:"max_lng" query:"max_lng" validate:"required,gte=-180,lte=180"`
	MaxLat    *float64 `json:"max_lat" query:"max_lat" validate:"required,gte=-90,lte=90"`
	TimePoint *int32   `json:"time" query:"time" validate:"omitempty,number"`
	EntityID  *string  `json:"entity_id" query:"entity_id" validate:"omitempty,uuid"`
	ProjectID *string  `json:"project_id" query:"project_id" validate:"omitempty,uuid"`
}

type SearchGeometriesByEntityNameDto struct {
	// Entity name keyword for searching. FE will render the result list by entity name.
	Name string `json:"name" query:"name" validate:"required,max=255"`
	// Cursor is entity UUID (id < cursor).
	Cursor string `json:"cursor" query:"cursor" validate:"omitempty,uuid"`
	Limit  int    `json:"limit" query:"limit" validate:"omitempty,min=1,max=100"`
}
