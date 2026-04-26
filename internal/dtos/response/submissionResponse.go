package response

import "time"

type SubmissionResponse struct {
	ID         string               `json:"id"`
	ProjectID  string               `json:"project_id"`
	CommitID   string               `json:"commit_id"`
	UserID     string               `json:"user_id"`
	CreatedAt  *time.Time           `json:"created_at"`
	Status     string               `json:"status"`
	ReviewedBy *string              `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time           `json:"reviewed_at,omitempty"`
	ReviewNote *string              `json:"review_note,omitempty"`
	Content    *string              `json:"content,omitempty"`
	User       *UserSimpleResponse  `json:"user"`
	Reviewer   *UserSimpleResponse  `json:"reviewer,omitempty"`
}
