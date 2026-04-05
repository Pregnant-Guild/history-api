package request

type PreSignedDto struct {
	FileName    string `json:"fileName" validate:"required"`
	ContentType string `json:"contentType" validate:"required"`
	Size        int64  `json:"size" validate:"required"`
}

type PreSignedCompleteDto struct {
	TokenID string `json:"token_id" validate:"required"`
}

type SearchMediaDto struct {
	CursorPaginationDto
	Search string `json:"search" query:"search" validate:"omitempty,min=2,max=200"`
}
