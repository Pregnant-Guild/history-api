package request

import "time"

type SearchUserVerificationDto struct {
	PaginationDto
	Sort string `json:"sort" query:"sort" validate:"omitempty,oneof=id created_at reviewed_at status"`
	Search string `json:"search" query:"search" validate:"omitempty,min=2,max=200"`
	UserIDs     []string `json:"user_ids" query:"user_ids" validate:"omitempty,dive,uuid"`
	VerifyTypes []string `json:"verify_types" query:"verify_types" validate:"omitempty,dive,ascii"`
	Statuses    []string `json:"statuses" query:"statuses" validate:"omitempty,dive,ascii"`
	ReviewedBy *string `json:"reviewed_by" query:"reviewed_by" validate:"omitempty,uuid"`
	CreatedAfter  *time.Time `json:"created_after" query:"created_after" validate:"omitempty"`
	CreatedBefore *time.Time `json:"created_before" query:"created_before" validate:"omitempty,gtfield=CreatedAfter"`
}

type CreateUserVerificationDto struct {
	VerifyType string   `json:"verify_type" validate:"required,oneof=ID_CARD EDUCATION EXPERT OTHER"`
	Content    string   `json:"content" validate:"required"`
	MediaIDs   []string `json:"media_ids" validate:"omitempty,dive,uuid"`
}

type UpdateVerificationStatusDto struct {
	Status     string `json:"status" validate:"required,oneof=PENDING APPROVED REJECTED"`
}