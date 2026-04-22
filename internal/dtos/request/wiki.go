package request

type SearchWikiDto struct {
	Cursor   string `json:"cursor" query:"cursor" validate:"omitempty,uuid"`
	Limit    int    `json:"limit" query:"limit" validate:"omitempty,min=1,max=100"`
	Title    string `json:"title" query:"title" validate:"omitempty,max=1000"`
	EntityID string `json:"entity_id" query:"entity_id" validate:"omitempty,uuid"`
}
