package models

import (
	"encoding/json"
	"history-api/internal/dtos/response"
	"history-api/pkg/constants"
	"time"
)

type SubmissionEntity struct {
	ID          string               `json:"id"`
	ProjectID   string               `json:"project_id"`
	CommitID    string               `json:"commit_id"`
	UserID      string               `json:"user_id"`
	CreatedAt   *time.Time           `json:"created_at"`
	Status      constants.StatusType `json:"status"`
	ReviewedBy  *string              `json:"reviewed_by"`
	ReviewedAt  *time.Time           `json:"reviewed_at"`
	ReviewNote  *string              `json:"review_note"`
	Content     *string              `json:"content"`
	IsDeleted   bool                 `json:"is_deleted"`
	User        *UserSimpleEntity    `json:"user"`
	Reviewer    *UserSimpleEntity    `json:"reviewer"`
}

func (s *SubmissionEntity) ParseUser(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		s.User = nil
		return nil
	}
	return json.Unmarshal(data, &s.User)
}

func (s *SubmissionEntity) ParseReviewer(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		s.Reviewer = nil
		return nil
	}
	return json.Unmarshal(data, &s.Reviewer)
}

func (s *SubmissionEntity) ToResponse() *response.SubmissionResponse {
	if s == nil {
		return nil
	}

	return &response.SubmissionResponse{
		ID:         s.ID,
		ProjectID:  s.ProjectID,
		CommitID:   s.CommitID,
		UserID:     s.UserID,
		CreatedAt:  s.CreatedAt,
		Status:     s.Status.String(),
		ReviewedBy: s.ReviewedBy,
		ReviewedAt: s.ReviewedAt,
		ReviewNote: s.ReviewNote,
		Content:    s.Content,
		User:       s.User.ToResponse(),
		Reviewer:   s.Reviewer.ToResponse(),
	}
}

func SubmissionsEntityToResponse(submissions []*SubmissionEntity) []*response.SubmissionResponse {
	out := make([]*response.SubmissionResponse, 0)
	if submissions == nil {
		return out
	}
	for _, submission := range submissions {
		if submission == nil {
			continue
		}
		out = append(out, submission.ToResponse())
	}
	return out
}
