package models

import (
	"encoding/json"
	"history-api/internal/dtos/response"
	"history-api/pkg/constants"
	"time"
)

type ProjectEntity struct {
	ID               string                      `json:"id"`
	Title            string                      `json:"title"`
	Description      string                      `json:"description"`
	LatestRevisionID *string                     `json:"latest_revision_id"`
	VersionCount     int32                       `json:"version_count"`
	ProjectStatus    constants.ProjectStatusType `json:"project_status"`
	LockedBy         *string                     `json:"locked_by"`
	IsDeleted        bool                        `json:"is_deleted"`
	UserID           string                      `json:"user_id"`
	CreatedAt        *time.Time                  `json:"created_at"`
	UpdatedAt        *time.Time                  `json:"updated_at"`
	User             *UserSimpleEntity           `json:"user"`
	CommitIds        []string                    `json:"commit_ids"`
	SubmissionIds    []string                    `json:"submission_ids"`
}

func (p *ProjectEntity) ParseUser(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		p.User = nil
		return nil
	}
	return json.Unmarshal(data, &p.User)
}

func (p *ProjectEntity) ToResponse() *response.ProjectResponse {
	if p == nil {
		return nil
	}
	var userResponse *response.UserSimpleResponse
	if p.User != nil {
		userResponse = p.User.ToResponse()
	}

	return &response.ProjectResponse{
		ID:               p.ID,
		Title:            p.Title,
		Description:      p.Description,
		LatestRevisionID: p.LatestRevisionID,
		VersionCount:     p.VersionCount,
		ProjectStatus:    p.ProjectStatus.String(),
		LockedBy:         p.LockedBy,
		IsDeleted:        p.IsDeleted,
		UserID:           p.UserID,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
		User:             userResponse,
		CommitIds:        p.CommitIds,
		SubmissionIds:    p.SubmissionIds,
	}
}

func ProjectsEntityToResponse(projects []*ProjectEntity) []*response.ProjectResponse {
	out := make([]*response.ProjectResponse, 0)
	if projects == nil {
		return out
	}
	for _, project := range projects {
		if project == nil {
			continue
		}
		out = append(out, project.ToResponse())
	}
	return out
}
