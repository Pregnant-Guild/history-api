package response

import "time"

type UserVerificationResponse struct {
	ID         string           `json:"id"`
	UserID     string           `json:"user_id"`
	VerifyType string           `json:"verify_type"`
	Content    string           `json:"content"`
	Status     string           `json:"status"`
	ReviewedBy *string          `json:"reviewed_by"`
	ReviewedAt *time.Time       `json:"reviewed_at"`
	CreatedAt  time.Time        `json:"created_at"`
	Medias     []*MediaSimpleResponse `json:"media"`
}
