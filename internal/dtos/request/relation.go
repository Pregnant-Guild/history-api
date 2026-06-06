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

type GetEntitiesByWikiIDsDto struct {
	WikiIDs []string `json:"wiki_ids" query:"wiki_ids" validate:"required,min=1,dive,uuid"`
}

type GetGeometriesByEntityIDsDto struct {
	EntityIDs []string `json:"entity_ids" query:"entity_ids" validate:"required,min=1,dive,uuid"`
}

type GetRelationsDto struct {
	Type string   `json:"type" query:"type" validate:"required,oneof=wiki-entity entity-wiki geometry-entity entity-geometry"`
	IDs  []string `json:"ids" query:"ids" validate:"required,min=1,dive,uuid"`
}


