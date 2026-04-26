package request

import "time"

type CreateSubmissionDto struct {
	ProjectID string `json:"project_id" validate:"required"`
	CommitID  string `json:"commit_id" validate:"required"`
	Content   string `json:"content"`
}

type UpdateSubmissionStatusDto struct {
	Status     string `json:"status" validate:"required"`
	ReviewNote string `json:"review_note" validate:"required,min=10"`
}

type SearchSubmissionDto struct {
	PaginationDto
	ProjectID   string     `json:"project_id" query:"project_id" validate:"omitempty,uuid"`
	Sort        string     `json:"sort" query:"sort" validate:"omitempty,oneof=id created_at reviewed_at status"`
	Search      string     `json:"search" query:"search" validate:"omitempty,min=2,max=200"`
	UserIDs     []string   `json:"user_ids" query:"user_ids" validate:"omitempty,dive,uuid"`
	Statuses    []string   `json:"statuses" query:"statuses" validate:"omitempty,dive,oneof=PENDING APPROVED REJECTED"`
	ReviewedBy  *string    `json:"reviewed_by" query:"reviewed_by" validate:"omitempty,uuid"`
	CreatedFrom *time.Time `json:"created_from" query:"created_from"`
	CreatedTo   *time.Time `json:"created_to" query:"created_to"`
}
