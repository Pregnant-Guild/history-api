package request

type SearchGeometryDto struct {
	MinLng    *float64 `json:"min_lng" query:"min_lng" validate:"required,gte=-180,lte=180"`
	MinLat    *float64 `json:"min_lat" query:"min_lat" validate:"required,gte=-90,lte=90"`
	MaxLng    *float64 `json:"max_lng" query:"max_lng" validate:"required,gte=-180,lte=180"`
	MaxLat    *float64 `json:"max_lat" query:"max_lat" validate:"required,gte=-90,lte=90"`
	TimePoint *int32   `json:"time" query:"time" validate:"omitempty,number"`
	EntityID  *string  `json:"entity_id" query:"entity_id" validate:"omitempty,uuid"`
}
