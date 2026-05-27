package request

type GetWikisByEntityIDsDto struct {
	EntityIDs []string `json:"entity_ids" query:"entity_ids" validate:"required,min=1,dive,uuid"`
}

type GetEntitiesByGeometryIDsDto struct {
	GeometryIDs []string `json:"geometry_ids" query:"geometry_ids" validate:"required,min=1,dive,uuid"`
}

type GetWikiContentsPreviewDto struct {
	IDs []string `json:"ids" query:"ids" validate:"required,min=1,dive,uuid"`
}
