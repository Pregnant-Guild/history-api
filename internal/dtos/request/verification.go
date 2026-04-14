package request

import (
	"time"
)

type SearchUserVerificationDto struct {
	PaginationDto
	Sort   string `json:"sort" query:"sort" validate:"omitempty,oneof=id created_at reviewed_at status"`
	Search string `json:"search" query:"search" validate:"omitempty,min=2,max=200"`
	UserIDs     []string `json:"user_ids" query:"user_ids" validate:"omitempty,dive,uuid"`
	VerifyTypes []string `json:"verify_types" query:"verify_types" validate:"omitempty,dive,oneof=ID_CARD EDUCATION EXPERT OTHER"`
	Statuses    []string `json:"statuses" query:"statuses" validate:"omitempty,dive,oneof=PENDING APPROVED REJECTED"`
	ReviewedBy *string `json:"reviewed_by" query:"reviewed_by" validate:"omitempty,uuid"`
	CreatedFrom *time.Time `json:"created_from" query:"created_from"`
	CreatedTo   *time.Time `json:"created_to" query:"created_to"`
}

type CreateUserVerificationDto struct {
	VerifyType string   `json:"verify_type" validate:"required,oneof=ID_CARD EDUCATION EXPERT OTHER"`
	Content    string   `json:"content" validate:"required,min=10"`
	MediaIDs   []string `json:"media_ids" validate:"omitempty,dive,uuid"`
}

type UpdateVerificationStatusDto struct {
	Status     string `json:"status" validate:"required,oneof=PENDING APPROVED REJECTED"`
	ReviewNote string `json:"review_note" validate:"required,min=5,max=3000"`
}