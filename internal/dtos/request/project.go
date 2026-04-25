package request

import "time"

type SearchProjectDto struct {
	PaginationDto
	Sort        string     `json:"sort" query:"sort" validate:"omitempty,oneof=created_at updated_at title"`
	Search      string     `json:"search" query:"search" validate:"omitempty,max=200"`
	UserIDs     []string   `json:"user_ids" query:"user_ids" validate:"omitempty,dive,uuid"`
	Statuses    []string   `json:"statuses" query:"statuses" validate:"omitempty,dive,oneof=PRIVATE PUBLIC ARCHIVE"`
	CreatedFrom *time.Time `json:"created_from" query:"created_from"`
	CreatedTo   *time.Time `json:"created_to" query:"created_to"`
}

type GetProjectsByUserDto struct {
	CursorID string `json:"cursor_id" query:"cursor_id" validate:"omitempty,uuid"`
	Limit    int32  `json:"limit" query:"limit" validate:"omitempty,min=1,max=100"`
}

type CreateProjectDto struct {
	Title       string  `json:"title" validate:"required,max=255"`
	Description *string `json:"description" validate:"omitempty"`
	Status      *string `json:"status" validate:"omitempty,oneof=PRIVATE PUBLIC ARCHIVE"`
}

type UpdateProjectDto struct {
	Title       *string `json:"title" validate:"omitempty,max=255"`
	Description *string `json:"description" validate:"omitempty"`
	Status      *string `json:"status" validate:"omitempty,oneof=PRIVATE PUBLIC ARCHIVE"`
}
