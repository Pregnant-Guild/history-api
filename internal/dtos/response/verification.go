package response

import "time"

type UserVerificationResponse struct {
	ID         string                 `json:"id"`
	User       *UserSimpleResponse    `json:"user,omitempty"`
	VerifyType string                 `json:"verify_type"`
	Content    string                 `json:"content"`
	Status     string                 `json:"status"`
	Reviewer   *UserSimpleResponse    `json:"reviewer,omitempty"`
	ReviewNote string                 `json:"review_note,omitempty"`
	ReviewedAt *time.Time             `json:"reviewed_at,omitempty"`
	CreatedAt  *time.Time             `json:"created_at,omitempty"`
	Medias     []*MediaSimpleResponse `json:"media,omitempty"`
}
