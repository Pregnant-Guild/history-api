package request

type PreSignedDto struct {
	FileName    string `json:"fileName" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
	Size        int64  `json:"size" validate:"required"`
}

type PreSignedCompleteDto struct {
	TokenID string `json:"token_id" validate:"required"`
}

type SearchMediaDto struct {
	PaginationDto
	Sort     string   `json:"sort" query:"sort" validate:"omitempty,oneof=id created_at updated_at size original_name storage_key mime_type"`
	Search   string   `json:"search" query:"search" validate:"omitempty,min=2,max=200"`
	UserIDs  []string `json:"user_ids" query:"user_ids" validate:"omitempty,dive,uuid"`
	MimeType string   `json:"mime_type" query:"mime_type" validate:"omitempty,max=100"`
	MinSize  *int64   `json:"min_size" query:"min_size" validate:"omitempty,min=0"`
	MaxSize  *int64   `json:"max_size" query:"max_size" validate:"omitempty,min=0,gtefield=MinSize"`
}

type MediaBulkDeleteDto struct {
	MediaIDs []string `json:"media_ids" validate:"required,dive,uuid"`
}
