package request

type SearchGeometryDto struct {
	MinLng    *float64 `query:"min_lng" validate:"required,gte=-180,lte=180"`
	MinLat    *float64 `query:"min_lat" validate:"required,gte=-90,lte=90"`
	MaxLng    *float64 `query:"max_lng" validate:"required,gte=-180,lte=180"`
	MaxLat    *float64 `query:"max_lat" validate:"required,gte=-90,lte=90"`
	TimePoint *int32   `json:"time" query:"time" validate:"omitempty,number"`
	EntityID  *string  `json:"entity_id" query:"entity_id" validate:"omitempty,uuid"`
}
