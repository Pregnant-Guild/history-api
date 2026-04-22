package request

type SearchEntityDto struct {
	Cursor string `json:"cursor" query:"cursor" validate:"omitempty,uuid"`
	Limit  int    `json:"limit" query:"limit" validate:"omitempty,min=1,max=100"`
	Name   string `json:"name" query:"name" validate:"omitempty,max=255"`
}
