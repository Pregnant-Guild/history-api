package request

type PreSignedDto struct {
	FileName    string `json:"fileName" validate:"required"`
	ContentType string `json:"contentType" validate:"required"`
}

type PreSignedCompleteDto struct {
	FileName  string `json:"fileName" validate:"required"`
	MediaId   string `json:"mediaId" validate:"required"`
	PublicUrl string `json:"publicUrl" validate:"required"`
}

type SearchMediaDto struct {
	MediaId  string `query:"media_id" validate:"omitempty"`
	FileName string `query:"file_name" validate:"omitempty"`
	SortBy   string `query:"sort_by" default:"created_at" validate:"oneof=created_at updated_at"`
	Order    string `query:"order" default:"desc" validate:"oneof=asc desc"`
	Page     int    `query:"page" default:"1" validate:"min=1"`
	Limit    int    `query:"limit" default:"10" validate:"min=1,max=100"`
}
