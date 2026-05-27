package request

type GetBattleReplaysByGeometryIDsDto struct {
	GeometryIDs []string `json:"geometry_ids" query:"geometry_ids" validate:"required,min=1,dive,uuid"`
}
