package response

import "time"

type UserVerificationResponse struct {
	ID         string                 `json:"id"`
	User       *UserSimpleResponse    `json:"user"`
	VerifyType string                 `json:"verify_type"`
	Content    string                 `json:"content"`
	Status     string                 `json:"status"`
	Reviewer   *UserSimpleResponse    `json:"reviewer"`
	ReviewNote string                 `json:"review_note"`
	ReviewedAt *time.Time             `json:"reviewed_at"`
	CreatedAt  *time.Time             `json:"created_at"`
	Medias     []*MediaSimpleResponse `json:"media"`
}
